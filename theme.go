//go:generate fyne bundle -o bundled.go -a assets/images/KrankyBearHogwartsSorting.png
//go:generate fyne bundle -o bundled.go -a assets/images/KrankyBearHogwartsSorting.png
//go:generate fyne bundle -o bundled.go -a assets/images/KrankyBearHogwartsSorting.png
//go:generate fyne bundle -o bundled.go -a assets/images/KrankyBearHogwartsSorting.png
//go:generate fyne bundle -o bundled.go -a assets/images/KrankyBearHogwartsSorting.png
//go:generate fyne bundle -o bundled.go -a assets/images/KrankyBearHogwartsSorting.png
//go:generate fyne bundle -o bundled.go -a assets/images/KrankyBearHogwartsSorting.png
//go:generate fyne bundle -o bundled.go -a assets/images/KrankyBearHardHat.png
//go:generate fyne bundle -o bundled.go -a assets/images/KrankyBearHogwartsSorting.png
//go:generate fyne bundle -o bundled.go -a assets/images/KrankyBearHogwartsSorting.png
//go:generate fyne bundle -o bundled.go -a assets/images/KrankyBearHogwartsSorting.png
//go:generate fyne bundle -o bundled.go -a assets/images/http418.png

//go:generate fyne bundle -o bundled.go -a assets/sounds/boing.mp3

package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// Theme switching functions (from KrankyBearGitExplorer)
func setLightTheme(a fyne.App) {
	a.Settings().SetTheme(theme.LightTheme())
	a.Preferences().SetString("theme", "light")
}

func setDarkTheme(a fyne.App) {
	a.Settings().SetTheme(theme.DarkTheme())
	a.Preferences().SetString("theme", "dark")
}

func setSystemTheme(a fyne.App) {
	a.Settings().SetTheme(theme.DefaultTheme())
	a.Preferences().SetString("theme", "system")
}

func loadTheme(a fyne.App) {
	themePref := a.Preferences().StringWithFallback("theme", "system")
	switch themePref {
	case "light":
		a.Settings().SetTheme(theme.LightTheme())
	case "dark":
		a.Settings().SetTheme(theme.DarkTheme())
	default:
		a.Settings().SetTheme(theme.DefaultTheme())
	}
}

// "Now this is not the end. It is not even the beginning of the end. But it is, perhaps, the end of the beginning." Winston Churchill, November 10, 1942
