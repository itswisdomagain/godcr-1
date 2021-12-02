package values

import (
	"image/color"
)

// SwitchStyle defines display properties that may be used to style a
// Switch widget.
type SwitchStyle struct {
	ActiveColor   color.NRGBA
	InactiveColor color.NRGBA
	ThumbColor    color.NRGBA
}

// IconButtonStyle defines display properties that may be used to style
// an IconButton.
type IconButtonStyle struct {
	Background color.NRGBA
	// Color is the icon color.
	Color color.NRGBA
}

// WidgetStyles is a collection of various widget styles.
type WidgetStyles struct {
	SwitchStyle     *SwitchStyle
	IconButtonStyle *IconButtonStyle
}

// DefaultWidgetStyles returns a new collection of widget styles with default
// values.
func DefaultWidgetStyles() *WidgetStyles {
	return &WidgetStyles{
		SwitchStyle:     &SwitchStyle{},
		IconButtonStyle: &IconButtonStyle{},
	}
}
