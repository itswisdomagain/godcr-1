package governance

import (
	"fmt"
	"image/color"
	"strconv"

	"gioui.org/text"
	"gioui.org/widget"
	"github.com/planetdecred/godcr/ui/decredmaterial"
	"github.com/planetdecred/godcr/ui/load"
	"github.com/planetdecred/godcr/ui/values"
)

type inputVoteOptionsWidgets struct {
	label     string
	activeBg  color.NRGBA
	dotColor  color.NRGBA
	input     decredmaterial.Editor
	increment decredmaterial.IconButton
	decrement decredmaterial.IconButton
	max       decredmaterial.Button
}

func newInputVoteOptions(l *load.Load, label string) *inputVoteOptionsWidgets {
	iconBtnStyle := &values.IconButtonStyle{
		Color:      l.Theme.Color.Text,
		Background: color.NRGBA{},
	}
	i := &inputVoteOptionsWidgets{
		label:     label,
		activeBg:  l.Theme.Color.Green50,
		dotColor:  l.Theme.Color.Green500,
		input:     l.Theme.Editor(new(widget.Editor), ""),
		increment: l.Theme.StyledIconButton(l.Icons.ContentAdd, iconBtnStyle),
		decrement: l.Theme.StyledIconButton(l.Icons.ContentRemove, iconBtnStyle),
		max:       l.Theme.Button("MAX"),
	}
	i.max.Background = l.Theme.Color.Surface
	i.max.Color = l.Theme.Color.GrayText1
	i.max.Font.Weight = text.SemiBold

	i.increment.SetSize(values.TextSize18)
	i.decrement.SetSize(values.TextSize18)

	i.input.Bordered = false
	i.input.Editor.SetText("0")
	i.input.Editor.Alignment = text.Middle
	return i
}

func (i *inputVoteOptionsWidgets) voteCount() int {
	value, err := strconv.Atoi(i.input.Editor.Text())
	if err != nil {
		fmt.Println(err)
		return 0
	}
	return value
}

func (i *inputVoteOptionsWidgets) reset() {
	i.input.Editor.SetText("0")
}

func (vm *voteModal) handleVoteCountButtons(i *inputVoteOptionsWidgets) {
	if i.increment.Clicked() {
		value, err := strconv.Atoi(i.input.Editor.Text())
		if err != nil {
			return
		}
		if vm.remainingVotes() <= 0 {
			return
		}
		value++
		i.input.Editor.SetText(fmt.Sprintf("%d", value))
	}

	if i.decrement.Clicked() {
		value, err := strconv.Atoi(i.input.Editor.Text())
		if err != nil {
			return
		}
		value--
		if value < 0 {
			return
		}
		i.input.Editor.SetText(fmt.Sprintf("%d", value))
	}

	if i.max.Clicked() {
		max := vm.remainingVotes() + i.voteCount()
		i.input.Editor.SetText(fmt.Sprint(max))
	}

	for _, e := range i.input.Editor.Events() {
		switch e.(type) {
		case widget.ChangeEvent:
			count := i.voteCount()
			if count < 0 {
				i.input.Editor.SetText("0")
			}
		}
	}
}
