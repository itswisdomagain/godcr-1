package dexclient

import (
	"fmt"
	"sync/atomic"

	"decred.org/dcrdex/client/core"
	"decred.org/dcrdex/dex/msgjson"
)

func (pg *Page) connectDex(h string, password []byte) {
	pg.Dexc.ConnectDexes(h, password)
	go pg.listenerMessages()
	go pg.readNotifications()
	pg.updateOrderBook()
}

func (pg *Page) updateOrderBook() {
	orderBook, err := pg.Dexc.Book(pg.selectedMaket.host, pg.selectedMaket.marketBaseID, pg.selectedMaket.marketQuoteID)
	if err != nil {
		return
	}
	pg.orderBook = orderBook
	pg.miniTradeFormWdg.orderBook = pg.orderBook
}

func (pg *Page) listenerMessages() {
	msgs := pg.Dexc.MessageSource(pg.selectedMaket.host)
	for {
		select {
		case msg, ok := <-msgs:
			if !ok {
				fmt.Println("[ERROR] Listen(wc): Connection terminated for.", "test")
				return
			}
			switch msg.Type {
			case msgjson.Request:
			case msgjson.Notification:
				pg.noteHandlers(msg)
			case msgjson.Response:
				// client/comms.wsConn handles responses to requests we sent.
				continue
			default:
				continue
			}
		case <-pg.ctx.Done():
			return
		}
	}
}

var listening uint32

// readNotifications reads from the Core notification channel.
// TODO: Why is this called more than once causing duplicate logs?
func (pg *Page) readNotifications() {
	if !atomic.CompareAndSwapUint32(&listening, 0, 1) {
		return
	}
	ch := pg.Dexc.NotificationFeed()
	for {
		select {
		case n := <-ch:
			switch n.Type() {
			case core.NoteTypeBalance:
				break

			case core.NoteTypeWalletState:
				wallet := n.(*core.WalletStateNote).Wallet
				fmt.Printf("== %s wallet synced = %t, sync progress = %f ==\n", wallet.Symbol, wallet.Synced, wallet.SyncProgress)

			case core.NoteTypeFeePayment:
				pg.RefreshWindow()
				fallthrough

			default:
				fmt.Println("Recv notification", n)
				// fmt.Println("<INFO>", n.ID())
				// fmt.Println("<INFO>", n.Severity())
				// fmt.Println("<INFO>", n.Type())
				// fmt.Println("<INFO>", n.DBNote())
				// fmt.Println("<INFO>", n.String())
				// fmt.Println("<INFO>", n.Subject())
			}
			pg.refreshUser()
		case <-pg.ctx.Done():
			return
		}
	}
}

func (pg *Page) noteHandlers(msg *msgjson.Message) {
	fmt.Println(">>> Receive message source: noteHandlers", msg.Route)
	switch msg.Route {
	case msgjson.BookOrderRoute:
		pg.updateOrderBook()
	case msgjson.EpochOrderRoute:
		pg.updateOrderBook()
	case msgjson.UnbookOrderRoute:
		pg.updateOrderBook()
	case msgjson.MatchProofRoute:
	case msgjson.UpdateRemainingRoute:
	case msgjson.EpochReportRoute:
	case msgjson.SuspensionRoute:
	case msgjson.ResumptionRoute:
	case msgjson.NotifyRoute:
	case msgjson.PenaltyRoute:
	case msgjson.NoMatchRoute:
	case msgjson.RevokeOrderRoute:
	case msgjson.RevokeMatchRoute:
	}
}
