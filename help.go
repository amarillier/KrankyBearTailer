package main

import (
	"net/url"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

var helpWin fyne.Window

// showHelp displays an in-app help window for Kranky Bear Tailer.
func showHelp(a fyne.App) {
	if helpWin != nil {
		helpWin.RequestFocus()
		return
	}

	helpText := `# ` + appName + ` — Help

## OVERVIEW
A cross-platform GUI log tail application for monitoring log files in
real-time. Each opened file gets its own tab; every tab keeps tailing
independently while you work in another.

## OPENING FILES
- File → Open File..., the toolbar "Open File" button, or the Welcome
  tab's "Open File" button.
- Recently-opened files are restored automatically the next time you
  launch the app, along with their keywords and colors.

## KEYWORD HIGHLIGHTING
- Each tab has its own keyword panel — add a keyword and a highlight
  color, and any matching line is colored as it streams in.
- Keywords and their colors persist per file across restarts.
- Optionally play a sound whenever a highlighted keyword appears (see
  Settings below).

## SOUND ALERTS
Settings → Application Settings... lets you:
- Enable/disable the audio alert on keyword matches.
- Choose the sound file to play (defaults to assets/sounds/boing.mp3;
  browse to any other .mp3 on disk).

## THEME
Settings → Dark Theme / Light Theme / System Theme switches the
application's appearance immediately and remembers your choice for
the next launch. The same three options are available from the
system tray menu.

## SYSTEM TRAY
Where supported, the tray icon gives quick access to Show/Hide the
window, Open File..., Settings, theme switching, Check for Updates,
About, Help, and Quit — without needing the main window focused.

## UPDATES
Help → Check for Updates... compares your installed version against
the latest GitHub release. The app also checks once at startup and
lets you know if a newer release is available.

## ABOUT / EASTER EGGS
Help → About shows version, author, and license info — click About
again while it's already open for a surprise (a random Kranky Bear
hat and a dad joke).

## MORE INFORMATION
- GitHub: https://github.com/amarillier/KrankyBearTailer
- License: https://github.com/amarillier/KrankyBearTailer/blob/main/LICENSE
`

	helpContent := widget.NewRichTextFromMarkdown(helpText)
	helpContent.Wrapping = fyne.TextWrapWord

	helpScroll := container.NewScroll(helpContent)
	helpScroll.SetMinSize(fyne.NewSize(560, 480))

	img := canvas.NewImageFromResource(resourceKrankyBearHogwartsSortingPng)
	img.FillMode = canvas.ImageFillContain
	img.SetMinSize(fyne.NewSize(150, 150))
	imgContainer := container.NewVBox(img)

	githubURL, _ := url.Parse("https://github.com/amarillier/KrankyBearTailer")
	githubLink := widget.NewHyperlink("Visit GitHub Repository", githubURL)
	githubLink.Alignment = fyne.TextAlignCenter

	footer := container.NewVBox(
		widget.NewSeparator(),
		container.NewCenter(githubLink),
	)

	mainArea := container.NewHBox(imgContainer, helpScroll)
	content := container.NewBorder(nil, footer, nil, nil, mainArea)

	helpWin = a.NewWindow(appName + " - Help")
	helpWin.SetIcon(resourceKrankyBearHogwartsSortingPng)
	helpWin.SetContent(content)
	helpWin.Resize(fyne.NewSize(700, 550))
	helpWin.CenterOnScreen()
	helpWin.SetCloseIntercept(func() {
		helpWin.Close()
		helpWin = nil
	})
	helpWin.Show()
}

// "Now this is not the end. It is not even the beginning of the end. But it is, perhaps, the end of the beginning." Winston Churchill, November 10, 1942
