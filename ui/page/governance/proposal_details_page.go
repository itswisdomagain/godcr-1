package governance

import (
	"context"
	"fmt"
	"time"

	"gioui.org/font/gofont"
	"gioui.org/layout"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/planetdecred/dcrlibwallet"
	"github.com/planetdecred/godcr/ui/decredmaterial"
	"github.com/planetdecred/godcr/ui/load"
	"github.com/planetdecred/godcr/ui/page/components"
	"github.com/planetdecred/godcr/ui/renderers"
	"github.com/planetdecred/godcr/ui/values"
	"github.com/planetdecred/godcr/wallet"
)

const ProposalDetailsPageID = "proposal_details"

type proposalItemWidgets struct {
	widgets    []layout.Widget
	clickables map[string]*widget.Clickable
}

type ProposalDetails struct {
	*load.Load
	ctx                context.Context // page context
	ctxCancel          context.CancelFunc
	voteBar            *components.VoteBar
	loadingDescription bool
	proposal           *dcrlibwallet.Proposal
	descriptionCard    decredmaterial.Card
	proposalItems      map[string]proposalItemWidgets
	descriptionList    *layout.List
	scrollbarList      *widget.List
	redirectIcon       *decredmaterial.Image
	rejectedIcon       *widget.Icon
	downloadIcon       *decredmaterial.Image
	successIcon        *widget.Icon
	vote               decredmaterial.Button
	backButton         decredmaterial.IconButton
	viewInPoliteiaBtn  *decredmaterial.Clickable
}

func NewProposalDetailsPage(l *load.Load, proposal *dcrlibwallet.Proposal) *ProposalDetails {
	pg := &ProposalDetails{
		Load: l,

		loadingDescription: false,
		proposal:           proposal,
		descriptionCard:    l.Theme.Card(),
		descriptionList:    &layout.List{Axis: layout.Vertical},
		scrollbarList: &widget.List{
			List: layout.List{Axis: layout.Vertical},
		},
		redirectIcon:      l.Icons.RedirectIcon,
		downloadIcon:      l.Icons.DownloadIcon,
		proposalItems:     make(map[string]proposalItemWidgets),
		rejectedIcon:      l.Icons.NavigationCancel,
		successIcon:       l.Icons.ActionCheckCircle,
		viewInPoliteiaBtn: l.Theme.NewClickable(true),
		voteBar:           components.NewVoteBar(l),
	}

	pg.backButton, _ = components.SubpageHeaderButtons(l)

	pg.vote = l.Theme.Button("Vote")
	pg.vote.TextSize = values.TextSize14
	pg.vote.Background = l.Theme.Color.Primary
	pg.vote.Color = l.Theme.Color.Surface
	pg.vote.Inset = layout.Inset{
		Top:    values.MarginPadding8,
		Bottom: values.MarginPadding8,
		Left:   values.MarginPadding12,
		Right:  values.MarginPadding12,
	}

	return pg
}

func (pg *ProposalDetails) ID() string {
	return ProposalDetailsPageID
}

func (pg *ProposalDetails) OnResume() {
	pg.ctx, pg.ctxCancel = context.WithCancel(context.TODO())
	pg.listenForSyncNotifications()
}

func (pg *ProposalDetails) Handle() {
	for token := range pg.proposalItems {
		for location, clickable := range pg.proposalItems[token].clickables {
			if clickable.Clicked() {
				components.GoToURL(location)
			}
		}
	}

	if pg.vote.Clicked() {
		newVoteModal(pg.Load, pg.proposal).Show()
	}

	for pg.viewInPoliteiaBtn.Clicked() {
		host := "https://proposals.decred.org/record/"
		if pg.WL.MultiWallet.NetType() == dcrlibwallet.Testnet3 {
			host = "https://test-proposals.decred.org/record/"
		}

		components.GoToURL(host + pg.proposal.Token)
	}
}

func (pg *ProposalDetails) listenForSyncNotifications() {

	go func() {
		for {
			var notification interface{}

			select {
			case notification = <-pg.Receiver.NotificationsUpdate:
			case <-pg.ctx.Done():
				return
			}

			switch n := notification.(type) {
			case wallet.Proposal:
				if n.ProposalStatus == wallet.Synced {
					proposal, err := pg.WL.MultiWallet.Politeia.GetProposalRaw(pg.proposal.Token)
					if err == nil {
						pg.proposal = proposal
						pg.RefreshWindow()
					}
				}
			}
		}
	}()
}

func (pg *ProposalDetails) OnClose() {
	pg.ctxCancel()
}

// - Layout

func (pg *ProposalDetails) layoutProposalVoteBar(gtx C) D {
	proposal := pg.proposal

	yes := float32(proposal.YesVotes)
	no := float32(proposal.NoVotes)
	quorumPercent := float32(proposal.QuorumPercentage)
	passPercentage := float32(proposal.PassPercentage)
	eligibleTickets := float32(proposal.EligibleTickets)

	return pg.voteBar.
		SetYesNoVoteParams(yes, no).
		SetVoteValidityParams(eligibleTickets, quorumPercent, passPercentage).
		SetProposalDetails(proposal.NumComments, proposal.PublishedAt, proposal.Token).
		Layout(gtx)
}

func (pg *ProposalDetails) layoutProposalVoteAction(gtx C) D {
	gtx.Constraints.Min.X = gtx.Constraints.Max.X
	return pg.vote.Layout(gtx)
}

func (pg *ProposalDetails) layoutInDiscussionState(gtx C) D {
	stateText1 := "Waiting for author to authorize voting"
	stateText2 := "Waiting for admin to trigger the start of voting"

	proposal := pg.proposal

	c := func(gtx layout.Context, val int32, info string) layout.Dimensions {
		return layout.Flex{}.Layout(gtx,
			layout.Rigid(func(gtx C) D {
				if proposal.VoteStatus == val || proposal.VoteStatus < val {
					c := pg.Theme.Card()
					c.Color = pg.Theme.Color.Primary
					c.Radius = decredmaterial.Radius(9.5)
					lbl := pg.Theme.Body1(fmt.Sprint(val))
					lbl.Color = pg.Theme.Color.Surface
					if proposal.VoteStatus < val {
						c.Color = pg.Theme.Color.Gray4
						lbl.Color = pg.Theme.Color.GrayText3
					}
					return c.Layout(gtx, func(gtx C) D {
						m := values.MarginPadding6
						return layout.Inset{Left: m, Right: m}.Layout(gtx, lbl.Layout)
					})
				}
				icon := decredmaterial.NewIcon(pg.successIcon)
				icon.Color = pg.Theme.Color.Primary
				return layout.Inset{
					Left:   values.MarginPaddingMinus2,
					Right:  values.MarginPaddingMinus2,
					Bottom: values.MarginPaddingMinus2,
				}.Layout(gtx, func(gtx C) D {
					return icon.Layout(gtx, values.MarginPadding24)
				})
			}),
			layout.Rigid(func(gtx C) D {
				col := pg.Theme.Color.Primary
				txt := info + "..."
				if proposal.VoteStatus != val {
					txt = info
					col = pg.Theme.Color.GrayText3
					if proposal.VoteStatus > 1 {
						col = pg.Theme.Color.Text
					}
				}
				lbl := pg.Theme.Body1(txt)
				lbl.Color = col
				return layout.Inset{Left: values.MarginPadding16}.Layout(gtx, lbl.Layout)
			}),
		)
	}

	gtx.Constraints.Min.X = gtx.Constraints.Max.X
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			return c(gtx, 1, stateText1)
		}),
		layout.Rigid(func(gtx C) D {
			height, width := gtx.Px(values.MarginPadding26), gtx.Px(values.MarginPadding4)
			line := pg.Theme.Line(height, width)
			if proposal.VoteStatus > 1 {
				line.Color = pg.Theme.Color.Primary
			} else {
				line.Color = pg.Theme.Color.Gray2
			}
			return layout.Inset{Left: values.MarginPadding8}.Layout(gtx, line.Layout)
		}),
		layout.Rigid(func(gtx C) D {
			return c(gtx, 2, stateText2)
		}),
	)
}

func (pg *ProposalDetails) layoutNormalTitle(gtx C) D {
	var label decredmaterial.Label
	var icon *decredmaterial.Icon
	proposal := pg.proposal
	switch proposal.Category {
	case dcrlibwallet.ProposalCategoryApproved:
		label = pg.Theme.Body2("Approved")
		icon = decredmaterial.NewIcon(pg.successIcon)
		icon.Color = pg.Theme.Color.Success
	case dcrlibwallet.ProposalCategoryRejected:
		label = pg.Theme.Body2("Rejected")
		icon = decredmaterial.NewIcon(pg.rejectedIcon)
		icon.Color = pg.Theme.Color.Danger
	case dcrlibwallet.ProposalCategoryAbandoned:
		label = pg.Theme.Body2("Abandoned")
	case dcrlibwallet.ProposalCategoryActive:
		label = pg.Theme.Body2("Voting in progress...")
	}
	timeagoLabel := pg.Theme.Body2(components.TimeAgo(proposal.Timestamp))

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			return layout.Flex{}.Layout(gtx,
				layout.Rigid(func(gtx C) D {
					if icon == nil {
						return D{}
					}
					return icon.Layout(gtx, values.MarginPadding20)
				}),
				layout.Rigid(func(gtx C) D {
					return layout.Inset{Left: values.MarginPadding5}.Layout(gtx, label.Layout)
				}),
				layout.Flexed(1, func(gtx C) D {
					return layout.E.Layout(gtx, func(gtx C) D {
						return layout.Flex{}.Layout(gtx,
							layout.Rigid(func(gtx C) D {
								if proposal.Category == dcrlibwallet.ProposalCategoryActive {
									ic := pg.Icons.TimerIcon
									if pg.WL.MultiWallet.ReadBoolConfigValueForKey(load.DarkModeConfigKey, false) {
										ic = pg.Icons.TimerDarkMode
									}
									return layout.Inset{
										Right: values.MarginPadding4,
										Top:   values.MarginPadding3,
									}.Layout(gtx, ic.Layout12dp)
								}
								return D{}
							}),
							layout.Rigid(timeagoLabel.Layout),
						)
					})
				}),
			)
		}),
		layout.Rigid(pg.lineSeparator(layout.Inset{Top: values.MarginPadding10, Bottom: values.MarginPadding10})),
		layout.Rigid(pg.layoutProposalVoteBar),
		layout.Rigid(func(gtx C) D {
			if proposal.Category != dcrlibwallet.ProposalCategoryActive {
				return D{}
			}
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(pg.lineSeparator(layout.Inset{Top: values.MarginPadding10, Bottom: values.MarginPadding10})),
				layout.Rigid(pg.layoutProposalVoteAction),
			)
		}),
	)
}

func (pg *ProposalDetails) layoutTitle(gtx C) D {
	proposal := pg.proposal

	return pg.descriptionCard.Layout(gtx, func(gtx C) D {
		return layout.UniformInset(values.MarginPadding15).Layout(gtx, func(gtx C) D {
			if proposal.Category == dcrlibwallet.ProposalCategoryPre {
				return pg.layoutInDiscussionState(gtx)
			}
			return pg.layoutNormalTitle(gtx)
		})
	})
}

func (pg *ProposalDetails) layoutDescription(gtx C) D {
	grayCol := pg.Theme.Color.GrayText2
	proposal := pg.proposal

	dotLabel := pg.Theme.H4(" . ")
	dotLabel.Color = grayCol

	userLabel := pg.Theme.Body2(proposal.Username)
	userLabel.Color = grayCol

	versionLabel := pg.Theme.Body2("Version " + proposal.Version)
	versionLabel.Color = grayCol

	publishedLabel := pg.Theme.Body2("Published " + components.TimeAgo(proposal.PublishedAt))
	publishedLabel.Color = grayCol

	updatedLabel := pg.Theme.Body2("Updated " + components.TimeAgo(proposal.Timestamp))
	updatedLabel.Color = grayCol

	w := []layout.Widget{
		func(gtx C) D {
			lbl := pg.Theme.H5(proposal.Name)
			lbl.Font.Weight = text.SemiBold
			return lbl.Layout(gtx)
		},
		pg.lineSeparator(layout.Inset{Top: values.MarginPadding16, Bottom: values.MarginPadding16}),
		func(gtx C) D {
			return layout.Flex{}.Layout(gtx,
				layout.Rigid(userLabel.Layout),
				layout.Rigid(func(gtx C) D {
					return layout.Inset{Top: values.MarginPaddingMinus22}.Layout(gtx, dotLabel.Layout)
				}),
				layout.Rigid(publishedLabel.Layout),
				layout.Rigid(func(gtx C) D {
					return layout.Inset{Top: values.MarginPaddingMinus22}.Layout(gtx, dotLabel.Layout)
				}),
				layout.Rigid(versionLabel.Layout),
				layout.Rigid(func(gtx C) D {
					return layout.Inset{Top: values.MarginPaddingMinus22}.Layout(gtx, dotLabel.Layout)
				}),
				layout.Rigid(updatedLabel.Layout),
			)
		},
		pg.lineSeparator(layout.Inset{Top: values.MarginPadding16, Bottom: values.MarginPadding16}),
	}

	_, ok := pg.proposalItems[proposal.Token]
	if ok {
		w = append(w, pg.proposalItems[proposal.Token].widgets...)
	} else {
		th := material.NewTheme(gofont.Collection())
		loading := func(gtx C) D {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx, layout.Flexed(1, func(gtx C) D {
				return layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(8)}.Layout(gtx, func(gtx C) D {
					return layout.Center.Layout(gtx, material.Loader(th).Layout)
				})
			}))
		}

		w = append(w, loading)
	}

	w = append(w, pg.layoutRedirect("View on Politeia", pg.redirectIcon, pg.viewInPoliteiaBtn))

	return pg.descriptionCard.Layout(gtx, func(gtx C) D {
		return pg.Theme.List(pg.scrollbarList).Layout(gtx, 1, func(gtx C, i int) D {
			return layout.UniformInset(values.MarginPadding16).Layout(gtx, func(gtx C) D {
				return pg.descriptionList.Layout(gtx, len(w), func(gtx C, i int) D {
					return layout.UniformInset(unit.Dp(0)).Layout(gtx, w[i])
				})
			})
		})
	})
}

func (pg *ProposalDetails) layoutRedirect(text string, icon *decredmaterial.Image, btn *decredmaterial.Clickable) layout.Widget {
	return func(gtx C) D {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(pg.lineSeparator(layout.Inset{Top: values.MarginPadding12, Bottom: values.MarginPadding12})),
			layout.Rigid(func(gtx C) D {
				return btn.Layout(gtx, func(gtx C) D {
					gtx.Constraints.Min.X = gtx.Constraints.Max.X
					return layout.Flex{Spacing: layout.SpaceBetween}.Layout(gtx,
						layout.Rigid(func(gtx C) D {
							return pg.Theme.Body1(text).Layout(gtx)
						}),
						layout.Rigid(func(gtx C) D {
							return layout.Inset{}.Layout(gtx, func(gtx C) D {
								return layout.E.Layout(gtx, icon.Layout24dp)
							})
						}),
					)
				})
			}),
		)
	}
}

func (pg *ProposalDetails) lineSeparator(inset layout.Inset) layout.Widget {
	return func(gtx C) D {
		return inset.Layout(gtx, pg.Theme.Separator().Layout)
	}
}

func (pg *ProposalDetails) Layout(gtx C) D {
	proposal := pg.proposal
	_, ok := pg.proposalItems[proposal.Token]
	if !ok && !pg.loadingDescription {
		pg.loadingDescription = true
		go func() {
			var proposalDescription string
			if proposal.IndexFile != "" && proposal.IndexFileVersion == proposal.Version {
				proposalDescription = proposal.IndexFile
			} else {
				var err error
				proposalDescription, err = pg.WL.MultiWallet.Politeia.FetchProposalDescription(proposal.Token)
				if err != nil {
					fmt.Printf("Error loading proposal description: %v", err)
					time.Sleep(7 * time.Second)
					pg.loadingDescription = false
					return
				}
			}

			r := renderers.RenderMarkdown(gtx, pg.Theme, proposalDescription)
			proposalWidgets, proposalClickables := r.Layout()
			pg.proposalItems[proposal.Token] = proposalItemWidgets{
				widgets:    proposalWidgets,
				clickables: proposalClickables,
			}
			pg.loadingDescription = false
		}()
	}

	body := func(gtx C) D {
		page := components.SubPage{
			Load:       pg.Load,
			Title:      components.TruncateString(proposal.Name, 40),
			BackButton: pg.backButton,
			Back: func() {
				pg.PopFragment()
			},
			Body: func(gtx C) D {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx C) D {
						return layout.Inset{Bottom: values.MarginPadding10}.Layout(gtx, pg.layoutTitle)
					}),
					layout.Rigid(pg.layoutDescription),
				)
			},
			ExtraItem: pg.viewInPoliteiaBtn,
			ExtraText: "View on Politeia",
			Extra: func(gtx C) D {
				return layout.Inset{}.Layout(gtx, func(gtx C) D {
					return layout.E.Layout(gtx, pg.redirectIcon.Layout24dp)
				})
			},
		}
		return page.Layout(gtx)
	}
	return components.UniformPadding(gtx, body)
}
