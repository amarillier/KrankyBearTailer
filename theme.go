//go:generate fyne bundle -o bundled.go -a Resources/Images/KrankyBear.png
//go:generate fyne bundle -o bundled.go -a Resources/Images/KrankyBearBeret.png
//go:generate fyne bundle -o bundled.go -a Resources/Images/KrankyBearBeanieMultiColor.png
//go:generate fyne bundle -o bundled.go -a Resources/Images/KrankyBearChristmas.png
//go:generate fyne bundle -o bundled.go -a Resources/Images/KrankyBearChristmasGrinch.png
//go:generate fyne bundle -o bundled.go -a Resources/Images/KrankyBearCowboyBrown.png
//go:generate fyne bundle -o bundled.go -a Resources/Images/KrankyBearFedoraRed.png
//go:generate fyne bundle -o bundled.go -a Resources/Images/KrankyBearHardHat.png
//go:generate fyne bundle -o bundled.go -a Resources/Images/KrankyBearHogwartsSorting.png
//go:generate fyne bundle -o bundled.go -a Resources/Images/KrankyBearTrapperRedPlaid.png
//go:generate fyne bundle -o bundled.go -a Resources/Images/KrankyBearVikingHelmet.png
//go:generate fyne bundle -o bundled.go -a Resources/Images/http418.png

//go:generate fyne bundle -o bundled.go -a Resources/Sounds/boing.mp3

package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type appTheme struct {
	fyne.Theme
}

func (a *appTheme) Size(n fyne.ThemeSizeName) float32 {
	if n == theme.SizeNameHeadingText {
		return a.Theme.Size(n) * 1.5
	}

	return a.Theme.Size(n)
}

func themeTimer(text *widget.RichText, time int) {
	seg := text.Segments[0].(*widget.TextSegment)
	if time <= 30 {
		seg.Style.ColorName = theme.ColorNameError
	} else if time < 150 {
		seg.Style.ColorName = theme.ColorNameWarning
	} else {
		seg.Style.ColorName = theme.ColorNameSuccess
	}
	text.Refresh()
}

// "Now this is not the end. It is not even the beginning of the end. But it is, perhaps, the end of the beginning." Winston Churchill, November 10, 1942
