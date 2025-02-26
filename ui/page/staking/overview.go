package staking

import (
	"fmt"
	"image/color"

	"gioui.org/layout"
	"gioui.org/widget"

	"github.com/decred/dcrd/dcrutil"
	"github.com/planetdecred/dcrlibwallet"
	"github.com/planetdecred/godcr/ui/decredmaterial"
	"github.com/planetdecred/godcr/ui/load"
	"github.com/planetdecred/godcr/ui/modal"
	"github.com/planetdecred/godcr/ui/page/components"
	tpage "github.com/planetdecred/godcr/ui/page/transaction"
	"github.com/planetdecred/godcr/ui/values"
)

type (
	C = layout.Context
	D = layout.Dimensions
)

const OverviewPageID = "staking"

type Page struct {
	*load.Load

	ticketPageContainer *layout.List
	ticketsLive         *decredmaterial.ClickableList

	stakeBtn decredmaterial.Button

	ticketPrice  string
	totalRewards string

	autoPurchaseEnabled *decredmaterial.Switch
	toTickets           decredmaterial.TextAndIconButton

	ticketOverview *dcrlibwallet.StakingOverview
	liveTickets    []*transactionItem
	list           *widget.List
}

func NewStakingPage(l *load.Load) *Page {
	pg := &Page{
		Load: l,

		ticketsLive:         l.Theme.NewClickableList(layout.Vertical),
		ticketPageContainer: &layout.List{Axis: layout.Vertical},
		stakeBtn:            l.Theme.Button("Stake"),

		autoPurchaseEnabled: l.Theme.Switch(),
		toTickets:           l.Theme.TextAndIconButton("See All", l.Icons.NavigationArrowForward),
	}

	pg.list = &widget.List{
		List: layout.List{
			Axis: layout.Vertical,
		},
	}
	pg.toTickets.Color = l.Theme.Color.Primary
	pg.toTickets.BackgroundColor = color.NRGBA{}

	pg.ticketOverview = new(dcrlibwallet.StakingOverview)
	return pg
}

func (pg *Page) ID() string {
	return OverviewPageID
}

func (pg *Page) OnResume() {

	pg.loadPageData()

	go pg.WL.GetVSPList()
	// TODO: automatic ticket purchase functionality
	pg.autoPurchaseEnabled.SetEnabled(false)
}

func (pg *Page) loadPageData() {
	go func() {
		ticketPrice, err := pg.WL.MultiWallet.TicketPrice()
		if err != nil {
			pg.Toast.NotifyError(err.Error())
		} else {
			pg.ticketPrice = dcrutil.Amount(ticketPrice.TicketPrice).String()
			pg.RefreshWindow()
		}
	}()

	go func() {
		totalRewards, err := pg.WL.MultiWallet.TotalStakingRewards()
		if err != nil {
			pg.Toast.NotifyError(err.Error())
		} else {
			pg.totalRewards = dcrutil.Amount(totalRewards).String()
			pg.RefreshWindow()
		}
	}()

	go func() {
		overview, err := pg.WL.MultiWallet.StakingOverview()
		if err != nil {
			pg.Toast.NotifyError(err.Error())
		} else {
			pg.ticketOverview = overview
			pg.RefreshWindow()
		}
	}()

	go func() {
		mw := pg.WL.MultiWallet
		tickets, err := allLiveTickets(mw)
		if err != nil {
			pg.Toast.NotifyError(err.Error())
			return
		}

		txItems, err := stakeToTransactionItems(pg.Load, tickets, true, func(filter int32) bool {
			switch filter {
			case dcrlibwallet.TxFilterUnmined:
				fallthrough
			case dcrlibwallet.TxFilterImmature:
				fallthrough
			case dcrlibwallet.TxFilterLive:
				return true
			}

			return false
		})
		if err != nil {
			pg.Toast.NotifyError(err.Error())
			return
		}

		pg.liveTickets = txItems
		pg.RefreshWindow()
	}()
}

func (pg *Page) Layout(gtx layout.Context) layout.Dimensions {
	widgets := []layout.Widget{
		func(ctx layout.Context) layout.Dimensions {
			return components.UniformHorizontalPadding(gtx, func(gtx layout.Context) layout.Dimensions {
				return pg.stakePriceSection(gtx)
			})
		},
		func(ctx layout.Context) layout.Dimensions {
			return components.UniformHorizontalPadding(gtx, func(gtx layout.Context) layout.Dimensions {
				return pg.stakeLiveSection(gtx)
			})
		},
		func(ctx layout.Context) layout.Dimensions {
			return components.UniformHorizontalPadding(gtx, func(gtx layout.Context) layout.Dimensions {
				return pg.stakingRecordSection(gtx)
			})
		},
	}

	return layout.Inset{Top: values.MarginPadding24}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return pg.Theme.List(pg.list).Layout(gtx, len(widgets), func(gtx C, i int) D {
			return widgets[i](gtx)
		})
	})
}

func (pg *Page) pageSections(gtx layout.Context, body layout.Widget) layout.Dimensions {
	return layout.Inset{
		Bottom: values.MarginPadding8,
	}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return pg.Theme.Card().Layout(gtx, func(gtx C) D {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			return layout.UniformInset(values.MarginPadding16).Layout(gtx, body)
		})
	})
}

func (pg *Page) titleRow(gtx layout.Context, leftWidget, rightWidget func(C) D) layout.Dimensions {
	return layout.Flex{Axis: layout.Horizontal, Spacing: layout.SpaceBetween}.Layout(gtx,
		layout.Rigid(leftWidget),
		layout.Rigid(rightWidget),
	)
}

func (pg *Page) stakePriceSection(gtx layout.Context) layout.Dimensions {
	return pg.pageSections(gtx, func(gtx C) D {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx C) D {
				return layout.Inset{
					Bottom: values.MarginPadding11,
				}.Layout(gtx, func(gtx C) D {
					// leftWg := func(gtx C) D {
					return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx C) D {
							title := pg.Theme.Label(values.TextSize14, "Ticket Price")
							title.Color = pg.Theme.Color.GrayText2
							return title.Layout(gtx)
						}),
						layout.Rigid(func(gtx C) D {
							return layout.Inset{
								Left:  values.MarginPadding8,
								Right: values.MarginPadding4,
							}.Layout(gtx, func(gtx C) D {
								ic := pg.Icons.TimerIcon
								if pg.WL.MultiWallet.ReadBoolConfigValueForKey(load.DarkModeConfigKey, false) {
									ic = pg.Icons.TimerDarkMode
								}
								return ic.Layout12dp(gtx)
							})
						}),
						layout.Rigid(func(gtx C) D {
							secs, _ := pg.WL.MultiWallet.NextTicketPriceRemaining()
							txt := pg.Theme.Label(values.TextSize14, nextTicketRemaining(int(secs)))
							txt.Color = pg.Theme.Color.GrayText2
							return txt.Layout(gtx)
						}),
					)
					// }
					//TODO: auto ticket purchase.
					// return pg.titleRow(gtx, leftWg, pg.autoPurchaseEnabled.Layout)-
				})
			}),
			layout.Rigid(func(gtx C) D {
				return layout.Inset{
					Bottom: values.MarginPadding8,
				}.Layout(gtx, func(gtx C) D {
					ic := pg.Icons.NewStakeIcon
					return layout.Center.Layout(gtx, ic.Layout48dp)
				})
			}),
			layout.Rigid(func(gtx C) D {
				return layout.Inset{
					Bottom: values.MarginPadding16,
				}.Layout(gtx, func(gtx C) D {
					return layout.Center.Layout(gtx, func(gtx C) D {
						return components.LayoutBalanceSize(gtx, pg.Load, pg.ticketPrice, values.TextSize28)
					})
				})
			}),
			layout.Rigid(func(gtx C) D {
				return layout.Center.Layout(gtx, func(gtx C) D {
					gtx.Constraints.Min.X = gtx.Px(values.MarginPadding150)
					return pg.stakeBtn.Layout(gtx)
				})
			}),
		)
	})
}

func (pg *Page) stakeLiveSection(gtx layout.Context) layout.Dimensions {
	return pg.pageSections(gtx, func(gtx C) D {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx C) D {
				return layout.Inset{Bottom: values.MarginPadding14}.Layout(gtx, func(gtx C) D {
					title := pg.Theme.Label(values.TextSize14, "Live Tickets")
					title.Color = pg.Theme.Color.GrayText2
					return pg.titleRow(gtx, title.Layout, func(gtx C) D {
						return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
							pg.stakingCountIcon(pg.Icons.TicketUnminedIcon, pg.ticketOverview.Unmined),
							pg.stakingCountIcon(pg.Icons.TicketImmatureIcon, pg.ticketOverview.Immature),
							pg.stakingCountIcon(pg.Icons.TicketLiveIcon, pg.ticketOverview.Live),
							layout.Rigid(func(gtx C) D {
								if len(pg.liveTickets) > 0 {
									return pg.toTickets.Layout(gtx)
								}
								return D{}
							}),
						)
					})
				})
			}),
			layout.Rigid(func(gtx C) D {
				if len(pg.liveTickets) == 0 {
					noLiveStake := pg.Theme.Label(values.TextSize16, "No live tickets yet.")
					noLiveStake.Color = pg.Theme.Color.GrayText3
					return noLiveStake.Layout(gtx)
				}
				return pg.ticketsLive.Layout(gtx, len(pg.liveTickets), func(gtx C, index int) D {
					return ticketListLayout(gtx, pg.Load, pg.liveTickets[index], index, true)
				})
			}),
		)
	})
}

func (pg *Page) stakingCountIcon(icon *decredmaterial.Image, count int) layout.FlexChild {
	return layout.Rigid(func(gtx C) D {
		if count == 0 {
			return D{}
		}
		return layout.Inset{Right: values.MarginPadding14}.Layout(gtx, func(gtx C) D {
			return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx C) D {
					return icon.Layout16dp(gtx)
				}),
				layout.Rigid(func(gtx C) D {
					return layout.Inset{Left: values.MarginPadding4}.Layout(gtx, func(gtx C) D {
						label := pg.Theme.Label(values.TextSize14, fmt.Sprintf("%d", count))
						return label.Layout(gtx)
					})
				}),
			)
		})
	})
}

func (pg *Page) stakingRecordSection(gtx C) D {
	return pg.pageSections(gtx, func(gtx C) D {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx C) D {
				return layout.Inset{
					Bottom: values.MarginPadding14,
				}.Layout(gtx, func(gtx C) D {
					title := pg.Theme.Label(values.TextSize14, "Ticket Record")
					title.Color = pg.Theme.Color.GrayText2

					if pg.ticketOverview.All == 0 {
						return pg.titleRow(gtx, title.Layout, func(gtx C) D { return D{} })
					}
					return pg.titleRow(gtx, title.Layout, pg.toTickets.Layout)
				})
			}),
			layout.Rigid(func(gtx C) D {
				wdgs := []layout.Widget{
					pg.ticketRecordIconCount(pg.Icons.TicketUnminedIcon, pg.ticketOverview.Unmined, "Unmined"),
					pg.ticketRecordIconCount(pg.Icons.TicketImmatureIcon, pg.ticketOverview.Immature, "Immature"),
					pg.ticketRecordIconCount(pg.Icons.TicketLiveIcon, pg.ticketOverview.Live, "Live"),
					pg.ticketRecordIconCount(pg.Icons.TicketVotedIcon, pg.ticketOverview.Voted, "Voted"),
					pg.ticketRecordIconCount(pg.Icons.TicketExpiredIcon, pg.ticketOverview.Expired, "Expired"),
					pg.ticketRecordIconCount(pg.Icons.TicketRevokedIcon, pg.ticketOverview.Revoked, "Revoked"),
				}

				return decredmaterial.GridWrap{
					Axis:      layout.Horizontal,
					Alignment: layout.End,
				}.Layout(gtx, len(wdgs), func(gtx C, i int) D {
					return wdgs[i](gtx)
				})
			}),
			layout.Rigid(func(gtx C) D {
				return decredmaterial.LinearLayout{
					Width:       decredmaterial.MatchParent,
					Height:      decredmaterial.WrapContent,
					Background:  pg.Theme.Color.Success2,
					Padding:     layout.Inset{Top: values.MarginPadding16, Bottom: values.MarginPadding16},
					Border:      decredmaterial.Border{Radius: decredmaterial.Radius(8)},
					Direction:   layout.Center,
					Alignment:   layout.Middle,
					Orientation: layout.Vertical,
				}.Layout(gtx,
					layout.Rigid(func(gtx C) D {
						return layout.Inset{Bottom: values.MarginPadding4}.Layout(gtx, func(gtx C) D {
							txt := pg.Theme.Label(values.TextSize14, "Rewards Earned")
							txt.Color = pg.Theme.Color.Turquoise700
							return txt.Layout(gtx)
						})
					}),
					layout.Rigid(func(gtx C) D {
						return layout.Flex{}.Layout(gtx,
							layout.Rigid(func(gtx C) D {
								ic := pg.Icons.StakeyIcon
								return layout.Inset{Right: values.MarginPadding6}.Layout(gtx, ic.Layout24dp)
							}),
							layout.Rigid(func(gtx C) D {
								award := pg.Theme.Color.Text
								noAward := pg.Theme.Color.GrayText3
								if pg.WL.MultiWallet.ReadBoolConfigValueForKey(load.DarkModeConfigKey, false) {
									award = pg.Theme.Color.Gray3
									noAward = pg.Theme.Color.Gray3
								}

								if pg.totalRewards == "0 DCR" {
									txt := pg.Theme.Label(values.TextSize16, "Stakey sees no rewards")
									txt.Color = noAward
									return txt.Layout(gtx)
								}

								return components.LayoutBalanceColor(gtx, pg.Load, pg.totalRewards, award)
							}),
						)
					}),
				)
			}),
		)
	})
}

func (pg *Page) ticketRecordIconCount(icon *decredmaterial.Image, count int, status string) layout.Widget {
	return func(gtx C) D {
		return layout.Inset{Bottom: values.MarginPadding16, Right: values.MarginPadding40}.Layout(gtx, func(gtx C) D {
			return layout.Flex{}.Layout(gtx,
				layout.Rigid(func(gtx C) D {
					return icon.Layout24dp(gtx)
				}),
				layout.Rigid(func(gtx C) D {
					return layout.Inset{Left: values.MarginPadding4}.Layout(gtx, func(gtx C) D {
						return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
							layout.Rigid(func(gtx C) D {
								label := pg.Theme.Label(values.TextSize16, fmt.Sprintf("%d", count))
								return label.Layout(gtx)
							}),
							layout.Rigid(func(gtx C) D {
								txt := pg.Theme.Label(values.TextSize12, status)
								txt.Color = pg.Theme.Color.GrayText2
								return txt.Layout(gtx)
							}),
						)
					})
				}),
			)
		})
	}
}

func (pg *Page) Handle() {
	if pg.stakeBtn.Clicked() {
		newStakingModal(pg.Load).
			TicketPurchased(func() {
				align := layout.Center
				successIcon := decredmaterial.NewIcon(pg.Icons.ActionCheckCircle)
				successIcon.Color = pg.Theme.Color.Success
				info := modal.NewInfoModal(pg.Load).
					Icon(successIcon).
					Title("Ticket(s) Confirmed").
					SetContentAlignment(align, align).
					PositiveButton("Back to staking", func() {})
				pg.ShowModal(info)
				pg.loadPageData()
			}).Show()
	}

	if pg.toTickets.Button.Clicked() {
		pg.ChangeFragment(newListPage(pg.Load))
	}

	if clicked, selectedItem := pg.ticketsLive.ItemClicked(); clicked {
		pg.ChangeFragment(tpage.NewTransactionDetailsPage(pg.Load, pg.liveTickets[selectedItem].transaction))
	}
}

func (pg *Page) OnClose() {}
