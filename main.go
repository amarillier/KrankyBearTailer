package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"image/color"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/cmd/fyne_settings/settings"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const (
	appName    = "Kranky Bear Tailer"
	appVersion = "0.1.1"
	appAuthor  = "Allan Marillier"
)

var appCopyright = "Copyright (c) Allan Marillier, 2024-" + strconv.Itoa(time.Now().Year())

// ScrollableLines is a custom container that displays log lines without separators
type ScrollableLines struct {
	scroll *container.Scroll
	vbox   *fyne.Container
	items  []fyne.CanvasObject // Store references to line widgets
}

func (sl *ScrollableLines) Refresh() {
	// Refresh all items in the VBox
	if sl.vbox != nil {
		sl.vbox.Refresh()
	}
	if sl.scroll != nil {
		sl.scroll.Refresh()
	}
}

// Refresh rebuilds the lines display - must be called from UI thread
func (ft *FileTailer) refreshLines() {
	ft.rebuildLines()
	ft.lines.Refresh()
}

func (sl *ScrollableLines) ScrollToBottom() {
	if sl.scroll != nil {
		sl.scroll.ScrollToBottom()
	}
}

// rebuildLines rebuilds the VBox content from the FileTailer's data
func (ft *FileTailer) rebuildLines() {
	ft.mutex.RLock()
	data := make([]string, len(ft.data))
	copy(data, ft.data)
	highlightColors := make([]color.Color, len(ft.highlightColors))
	copy(highlightColors, ft.highlightColors)
	ft.mutex.RUnlock()

	// Create widgets for each line
	newItems := make([]fyne.CanvasObject, 0, len(data))
	for i, line := range data {
		label := widget.NewLabel(line)
		label.Wrapping = fyne.TextWrapWord

		var bgRect *canvas.Rectangle
		if i < len(highlightColors) && highlightColors[i] != nil {
			bgRect = canvas.NewRectangle(highlightColors[i])
		} else {
			bgRect = canvas.NewRectangle(color.Transparent)
		}

		lineContainer := container.NewStack(bgRect, label)
		newItems = append(newItems, lineContainer)
	}

	// Replace the VBox content by creating a new VBox
	ft.lines.items = newItems
	ft.lines.vbox.Objects = newItems
	ft.lines.vbox.Refresh()
}

type FileTailer struct {
	filePath        string
	file            *os.File
	fileInfo        os.FileInfo
	scanner         *bufio.Scanner
	lines           *ScrollableLines
	data            []string
	highlightColors []color.Color          // Color for each line (nil = not highlighted)
	keywordColors   map[string]color.Color // Map keyword -> color
	mutex           sync.RWMutex
	stopChan        chan struct{}
}

type App struct {
	w           fyne.Window
	app         fyne.App
	tabs        *container.DocTabs
	fileTailers map[string]*FileTailer
	keywords    []string
	pathForTab  map[*container.TabItem]string
}

func NewFileTailer(filePath string) (*FileTailer, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}

	// Get file info
	fileInfo, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, err
	}

	ft := &FileTailer{
		filePath:        filePath,
		file:            file,
		fileInfo:        fileInfo,
		scanner:         bufio.NewScanner(file),
		data:            []string{},
		highlightColors: []color.Color{},
		keywordColors:   make(map[string]color.Color),
		stopChan:        make(chan struct{}),
	}

	// Create scrollable lines container (no separators between items)
	vbox := container.NewVBox()
	scroll := container.NewScroll(vbox)
	ft.lines = &ScrollableLines{
		scroll: scroll,
		vbox:   vbox,
		items:  []fyne.CanvasObject{},
	}

	return ft, nil
}

func (ft *FileTailer) LoadInitialContent() error {
	// Read the last 20 lines of existing content
	err := ft.loadLastLines(20)
	if err != nil {
		// If we can't read last lines, that's ok - just continue
	}

	// Now seek to end for tailing
	ft.file.Seek(ft.fileInfo.Size(), 0)
	ft.scanner = bufio.NewScanner(ft.file)

	return nil
}

func (ft *FileTailer) loadLastLines(numLines int) error {
	file, err := os.Open(ft.filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Get file size
	stat, err := file.Stat()
	if err != nil {
		return err
	}

	// Read file from end to get last N lines
	fileSize := stat.Size()
	var bufferSize int64 = 4096
	if fileSize < bufferSize {
		bufferSize = fileSize
	}

	// Start reading from near the end
	var offset int64 = fileSize - bufferSize
	if offset < 0 {
		offset = 0
	}

	file.Seek(offset, 0)
	scanner := bufio.NewScanner(file)

	// Read all lines from this position
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	// Get the last numLines
	if len(lines) > numLines {
		lines = lines[len(lines)-numLines:]
	}

	// Add to data - highlighting will be applied via refreshHighlighting if keywords exist
	ft.mutex.Lock()
	for _, line := range lines {
		ft.data = append(ft.data, line)
		ft.highlightColors = append(ft.highlightColors, ft.getHighlightColorLocked(line))
	}
	ft.mutex.Unlock()

	return nil
}

func (ft *FileTailer) AddKeyword(keyword string, keywordColor color.Color) {
	ft.mutex.Lock()
	defer ft.mutex.Unlock()

	// Check if keyword already exists
	if _, exists := ft.keywordColors[keyword]; exists {
		// Update color if it exists
		ft.keywordColors[keyword] = keywordColor
	} else {
		// Add new keyword with color
		ft.keywordColors[keyword] = keywordColor
	}

	// Recheck all existing lines for highlighting (lock already held)
	ft.refreshHighlightingLocked()
}

func (ft *FileTailer) RemoveKeyword(keyword string) {
	ft.mutex.Lock()
	defer ft.mutex.Unlock()

	delete(ft.keywordColors, keyword)

	// Recheck all existing lines for highlighting (lock already held)
	ft.refreshHighlightingLocked()
}

func (ft *FileTailer) refreshHighlightingLocked() {
	// Recheck all lines for highlighting, requires caller holds ft.mutex
	// Ensure highlightColors slice is the right size
	for len(ft.highlightColors) < len(ft.data) {
		ft.highlightColors = append(ft.highlightColors, nil)
	}
	for len(ft.highlightColors) > len(ft.data) {
		ft.highlightColors = ft.highlightColors[:len(ft.data)]
	}

	for i, line := range ft.data {
		ft.highlightColors[i] = ft.getHighlightColorLocked(line)
	}
}

func (ft *FileTailer) refreshHighlighting() {
	ft.mutex.Lock()
	ft.refreshHighlightingLocked()
	ft.mutex.Unlock()
}

func (ft *FileTailer) getHighlightColorLocked(line string) color.Color {
	// Return the color of the first matching keyword
	lineLower := strings.ToLower(line)
	for keyword, keywordColor := range ft.keywordColors {
		if strings.Contains(lineLower, strings.ToLower(keyword)) {
			return keywordColor
		}
	}
	return nil
}

func (ft *FileTailer) shouldHighlightLocked(line string) bool {
	return ft.getHighlightColorLocked(line) != nil
}

func (ft *FileTailer) GetKeywords() []string {
	ft.mutex.RLock()
	defer ft.mutex.RUnlock()
	keywords := make([]string, 0, len(ft.keywordColors))
	for k := range ft.keywordColors {
		keywords = append(keywords, k)
	}
	// Sort keywords for stable ordering
	sort.Strings(keywords)
	return keywords
}

func (ft *FileTailer) GetKeywordColor(keyword string) (color.Color, bool) {
	ft.mutex.RLock()
	defer ft.mutex.RUnlock()
	c, exists := ft.keywordColors[keyword]
	return c, exists
}

func (ft *FileTailer) GetAllKeywordColors() map[string]color.Color {
	ft.mutex.RLock()
	defer ft.mutex.RUnlock()
	// Return a copy to avoid race conditions
	result := make(map[string]color.Color)
	for k, v := range ft.keywordColors {
		result[k] = v
	}
	return result
}

// Helper functions to encode/decode colors for preferences storage
func colorToHex(c color.Color) string {
	r, g, b, a := c.RGBA()
	// Convert from 16-bit to 8-bit
	return fmt.Sprintf("#%02x%02x%02x%02x", r>>8, g>>8, b>>8, a>>8)
}

func hexToColor(hex string) (color.Color, error) {
	if len(hex) != 9 || hex[0] != '#' {
		return nil, fmt.Errorf("invalid color format")
	}
	r, err1 := strconv.ParseUint(hex[1:3], 16, 8)
	g, err2 := strconv.ParseUint(hex[3:5], 16, 8)
	b, err3 := strconv.ParseUint(hex[5:7], 16, 8)
	a, err4 := strconv.ParseUint(hex[7:9], 16, 8)
	if err1 != nil || err2 != nil || err3 != nil || err4 != nil {
		return nil, fmt.Errorf("invalid color format")
	}
	return color.RGBA{uint8(r), uint8(g), uint8(b), uint8(a)}, nil
}

func saveKeywordColors(prefs fyne.Preferences, key string, keywordColors map[string]color.Color) {
	// Save as JSON: {"keyword1": "#rrggbbaa", "keyword2": "#rrggbbaa", ...}
	colorMap := make(map[string]string)
	for kw, c := range keywordColors {
		colorMap[kw] = colorToHex(c)
	}
	jsonData, err := json.Marshal(colorMap)
	if err == nil {
		prefs.SetString(key+"_colors", string(jsonData))
	}
}

func loadKeywordColors(prefs fyne.Preferences, key string) map[string]color.Color {
	keywordColors := make(map[string]color.Color)
	jsonData := prefs.StringWithFallback(key+"_colors", "")
	if jsonData == "" {
		return keywordColors
	}
	var colorMap map[string]string
	if err := json.Unmarshal([]byte(jsonData), &colorMap); err == nil {
		for kw, hexColor := range colorMap {
			if c, err := hexToColor(hexColor); err == nil {
				keywordColors[kw] = c
			}
		}
	}
	return keywordColors
}

func (ft *FileTailer) SetKeywords(keywords map[string]color.Color) {
	ft.mutex.Lock()
	ft.keywordColors = keywords
	ft.mutex.Unlock()

	// Refresh highlighting for all existing lines
	ft.refreshHighlighting()

	// Refresh the widget to show the updated highlighting - must be on main thread
	fyne.Do(func() {
		ft.refreshLines()
	})
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func (ft *FileTailer) shouldHighlight(line string) bool {
	ft.mutex.RLock()
	defer ft.mutex.RUnlock()
	return ft.shouldHighlightLocked(line)
}

func (ft *FileTailer) StartTail() {
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ft.stopChan:
				return
			case <-ticker.C:
				// Check if file has new content
				stat, err := os.Stat(ft.filePath)
				if err != nil {
					continue
				}

				ft.mutex.RLock()
				currentSize := ft.fileInfo.Size()
				ft.mutex.RUnlock()

				// Check if file was rotated (truncated) or grew
				if stat.Size() < currentSize {
					// File was rotated, reopen
					ft.file.Close()
					ft.file, err = os.Open(ft.filePath)
					if err == nil {
						ft.mutex.Lock()
						ft.fileInfo = stat
						ft.mutex.Unlock()
						ft.file.Seek(stat.Size(), 0)
						ft.scanner = bufio.NewScanner(ft.file)
					}
				} else if stat.Size() > currentSize {
					// File grew - read new content
					newFile, err := os.Open(ft.filePath)
					if err != nil {
						continue
					}

					// Seek to where we left off
					newFile.Seek(currentSize, 0)

					// Read all new lines
					var newLines []string
					scanner := bufio.NewScanner(newFile)
					for scanner.Scan() {
						newLines = append(newLines, scanner.Text())
					}
					newFile.Close()

					// Add all new lines to data and check for keyword matches
					ft.mutex.Lock()
					playSound := false
					for _, line := range newLines {
						highlightColor := ft.getHighlightColorLocked(line)
						if highlightColor != nil {
							playSound = true
						}
						ft.data = append(ft.data, line)
						ft.highlightColors = append(ft.highlightColors, highlightColor)
					}
					ft.fileInfo = stat
					ft.mutex.Unlock()

					// Play sound if keyword match found (only once per batch, in background)
					// Only play if sound is enabled
					if playSound && len(ft.keywordColors) > 0 && soundEnabled {
						go func() {
							// Try multiple possible sound paths
							var soundPath string
							var err error

							// First try relative to current working directory
							soundPath = "Resources/Sounds/boing.mp3"
							if _, err = os.Stat(soundPath); err == nil {
								// fmt.Printf("DEBUG: Playing sound from: %s\n", soundPath)
								playMp3(soundPath)
								return
							}

							// Then try relative to executable
							exePath, err := os.Executable()
							if err == nil {
								appDir := filepath.Dir(exePath)
								soundPath = filepath.Join(appDir, "Resources", "Sounds", "boing.mp3")
								if _, err := os.Stat(soundPath); err == nil {
									// fmt.Printf("DEBUG: Playing sound from: %s\n", soundPath)
									playMp3(soundPath)
									return
								}
							}

							// Fallback: try from source directory structure
							cwd, err := os.Getwd()
							if err == nil {
								soundPath = filepath.Join(cwd, "Resources", "Sounds", "boing.mp3")
								if _, err := os.Stat(soundPath); err == nil {
									// fmt.Printf("DEBUG: Playing sound from: %s\n", soundPath)
									playMp3(soundPath)
								}
							}

							// Sound file not found
						}()
					}

					// Refresh UI - use fyne.Do for thread safety
					if len(newLines) > 0 {
						fyne.Do(func() {
							ft.refreshLines()
							ft.lines.ScrollToBottom()
						})
					}

					// Reopen file for next check
					ft.file.Close()
					ft.file, err = os.Open(ft.filePath)
					if err == nil {
						ft.file.Seek(stat.Size(), 0)
					}
				}
			}
		}
	}()
}

func (ft *FileTailer) Stop() {
	close(ft.stopChan)
	if ft.file != nil {
		ft.file.Close()
	}
}

func (app *App) openFile() {
	// Custom dialog that supports typing a path or browsing via file chooser
	pathEntry := widget.NewEntry()
	pathEntry.SetPlaceHolder("Type full path to file (e.g., /var/log/syslog)")

	browseBtn := widget.NewButtonWithIcon("Browse...", theme.FolderOpenIcon(), func() {
		fd := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil || reader == nil {
				return
			}
			defer reader.Close()
			pathEntry.SetText(reader.URI().Path())
		}, app.w)
		// Optional: show common log extensions first
		// fd.SetFilter(storage.NewExtensionFileFilter([]string{".log", ".txt"}))
		fd.SetDismissText("Cancel")
		fd.Resize(fyne.NewSize(800, 600))
		fd.Show()
	})

	content := container.NewVBox(
		widget.NewLabel("Open a log file"),
		pathEntry,
		container.NewHBox(layout.NewSpacer(), browseBtn),
	)

	dialog.NewCustomConfirm("Open File", "Open", "Cancel", content, func(ok bool) {
		if !ok {
			return
		}
		path := strings.TrimSpace(pathEntry.Text)
		if path == "" {
			dialog.ShowError(fmt.Errorf("Please enter or select a file"), app.w)
			return
		}
		if stat, err := os.Stat(path); err != nil || stat.IsDir() {
			dialog.ShowError(fmt.Errorf("Invalid file: %s", path), app.w)
			return
		}
		app.addFileTab(path)
	}, app.w).Show()
}

func (app *App) addFileTab(filePath string) {
	fileName := filepath.Base(filePath)

	// Check if tab already exists
	for tabName := range app.fileTailers {
		if tabName == filePath {
			// Find and select the existing tab
			for i, tab := range app.tabs.Items {
				if tab.Text == fileName {
					app.tabs.SelectIndex(i)
					return
				}
			}
		}
	}

	ft, err := NewFileTailer(filePath)
	if err != nil {
		dialog.ShowError(err, app.w)
		return
	}

	app.fileTailers[filePath] = ft

	// Load initial content (last 20 lines) BEFORE starting tail
	ft.LoadInitialContent()

	ft.StartTail()

	// Create tab content
	content := container.NewBorder(
		app.createKeywordPanel(ft),
		nil,
		nil,
		nil,
		ft.lines.scroll,
	)

	tabItem := container.NewTabItem(fileName, content)
	app.tabs.Append(tabItem)
	app.pathForTab[tabItem] = filePath

	// Rebuild lines to display initial content
	fyne.Do(func() {
		ft.refreshLines()
		ft.lines.ScrollToBottom()
	})

	// Set the OnClosed callback once if not already set
	if app.tabs.OnClosed == nil {
		app.tabs.OnClosed = func(item *container.TabItem) {
			// Resolve full file path for this tab
			path, ok := app.pathForTab[item]
			if ok {
				delete(app.pathForTab, item)
			} else {
				// Fallback by matching on base name
				for p := range app.fileTailers {
					if filepath.Base(p) == item.Text {
						path = p
						break
					}
				}
			}

			if path != "" {
				if ft, exists := app.fileTailers[path]; exists {
					ft.Stop()
					delete(app.fileTailers, path)
				}
				// Immediately update preferences to remove this file and its keywords
				prefs := app.app.Preferences()

				// Remove from open_files list
				current := prefs.StringListWithFallback("open_files", nil)
				updated := make([]string, 0, len(current))
				for _, p := range current {
					if p != path {
						updated = append(updated, p)
					}
				}
				prefs.SetStringList("open_files", updated)

				// Clear stored keywords for this file
				prefs.SetStringList("keywords_"+path, []string{})
			}
		}
	}

	app.tabs.SelectIndex(len(app.tabs.Items) - 1)
}

func (app *App) createKeywordPanel(ft *FileTailer) fyne.CanvasObject {
	entry := widget.NewEntry()
	entry.SetPlaceHolder("Enter keyword to highlight...")

	addBtn := widget.NewButtonWithIcon("Add", theme.ContentAddIcon(), func() {
		keyword := entry.Text
		if keyword != "" {
			// Show color picker dialog
			colorPicker := dialog.NewColorPicker("Select Color", "Choose a color for this keyword:", func(c color.Color) {
				ft.AddKeyword(keyword, c)
				entry.SetText("")
				// We are on the UI thread here; refresh directly
				ft.refreshLines()
			}, app.w)
			// Note: SetColor requires advanced mode which isn't available in this API
			// User will need to pick color manually each time
			colorPicker.Show()
		}
	})

	removeBtn := widget.NewButtonWithIcon("Remove", theme.ContentRemoveIcon(), func() {
		keyword := entry.Text
		if keyword != "" {
			ft.RemoveKeyword(keyword)
			entry.SetText("")
			// We are on the UI thread here; refresh directly
			ft.refreshLines()
		}
	})

	viewBtn := widget.NewButtonWithIcon("View", theme.InfoIcon(), func() {
		showKeywords(app.w, ft)
	})

	clearBtn := widget.NewButtonWithIcon("Clear All", theme.DeleteIcon(), func() {
		keywords := ft.GetKeywords()
		for _, k := range keywords {
			ft.RemoveKeyword(k)
		}
		// We are on the UI thread here; refresh directly
		ft.refreshLines()
	})

	buttonBox := container.NewHBox(addBtn, removeBtn, viewBtn, clearBtn)
	keywordPanel := container.NewBorder(nil, nil, nil, buttonBox, entry)

	return keywordPanel
}

func showKeywords(parent fyne.Window, ft *FileTailer) {
	// Get just the filename from the path
	filename := filepath.Base(ft.filePath)

	// Initialize the map if needed
	if keywordsWinFor == nil {
		keywordsWinFor = make(map[string]fyne.Window)
	}

	// Ensure there is at most one window; if an entry exists, close and recreate to avoid stale pointers
	if existingWin, exists := keywordsWinFor[ft.filePath]; exists {
		if existingWin != nil {
			existingWin.Close()
		}
		delete(keywordsWinFor, ft.filePath)
	}

	// Close keywords windows for other files
	for filePath, win := range keywordsWinFor {
		if filePath != ft.filePath && win != nil {
			win.Close()
			delete(keywordsWinFor, filePath)
		}
	}

	// Create window for keywords
	keywordsWindow := fyne.CurrentApp().NewWindow("Watched Keywords - " + filename)
	// Set seasonal window icon
	{
		_, month, _ := time.Now().Date()
		if month == time.December {
			keywordsWindow.SetIcon(resourceKrankyBearChristmasGrinchPng)
		} else {
			keywordsWindow.SetIcon(resourceKrankyBearHogwartsSortingPng)
		}
	}
	keywordsWin = keywordsWindow
	keywordsWinFor[ft.filePath] = keywordsWindow
	keywordsWindow.Resize(fyne.NewSize(350, 400))

	var list *widget.List

	// Create list widget for keywords
	list = widget.NewList(
		func() int {
			keywords := ft.GetKeywords()
			return len(keywords)
		},
		func() fyne.CanvasObject {
			removeBtn := widget.NewButton("Remove", nil)
			// Create colored rectangle (will be updated with actual color)
			colorSwatch := canvas.NewRectangle(color.Gray{200})
			colorSwatch.SetMinSize(fyne.NewSize(30, 25))
			// Border around the swatch
			borderRect := canvas.NewRectangle(color.RGBA{150, 150, 150, 255})
			borderRect.SetMinSize(fyne.NewSize(32, 27))
			// Transparent button on top for click handling
			clickBtn := widget.NewButton("", nil)
			clickBtn.Importance = widget.LowImportance
			clickBtn.Resize(fyne.NewSize(32, 27))
			// Stack: border, color, button (button on top for clicks)
			colorContainer := container.NewStack(borderRect, colorSwatch, clickBtn)
			label := widget.NewLabel("keyword")
			return container.NewBorder(nil, nil, nil, container.NewHBox(colorContainer, removeBtn), label)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			keywords := ft.GetKeywords()
			if id >= len(keywords) {
				return
			}

			box := obj.(*fyne.Container)
			label := box.Objects[0].(*widget.Label)
			buttonBox := box.Objects[1].(*fyne.Container)
			colorContainer := buttonBox.Objects[0].(*fyne.Container)
			removeBtn := buttonBox.Objects[1].(*widget.Button)

			keyword := keywords[id]
			label.SetText(keyword)

			// Extract components from the stacked container
			// Stack structure: [borderRect, colorSwatch, clickBtn]
			colorSwatch := colorContainer.Objects[1].(*canvas.Rectangle)
			clickBtn := colorContainer.Objects[2].(*widget.Button)

			// Get the keyword color and update the swatch
			keywordColor, exists := ft.GetKeywordColor(keyword)
			if exists && keywordColor != nil {
				colorSwatch.FillColor = keywordColor
			} else {
				colorSwatch.FillColor = color.Gray{200} // Default gray if no color set
			}
			colorSwatch.Refresh()

			// Set up click handler on the button (which is on top of the stack)
			clickBtn.OnTapped = func() {
				// Show color picker to change keyword color
				colorPicker := dialog.NewColorPicker("Select Color", fmt.Sprintf("Choose a color for '%s':", keyword), func(c color.Color) {
					ft.AddKeyword(keyword, c)
					ft.refreshLines()
					list.Refresh()
				}, keywordsWindow)
				colorPicker.Show()
			}

			removeBtn.OnTapped = func() {
				// On UI thread: update model then refresh widgets directly
				ft.RemoveKeyword(keyword)
				ft.refreshLines()
				list.Refresh()
			}
		},
	)

	// Create window content
	content := container.NewBorder(
		widget.NewLabel("Click 'Remove' on a keyword to stop highlighting it:"),
		widget.NewButton("Close", func() {
			keywordsWindow.Close()
		}),
		nil,
		nil,
		list,
	)

	keywordsWindow.SetContent(content)
	keywordsWindow.CenterOnScreen()

	keywordsWindow.SetOnClosed(func() {
		delete(keywordsWinFor, ft.filePath)
		keywordsWin = nil
	})

	keywordsWindow.Show()
}

func showAbout(parent fyne.Window) {
	// Load the embedded image - use Christmas image in December
	aboutImg := resourceKrankyBearHogwartsSortingPng
	_, month, _ := time.Now().Date()
	if month == time.December {
		aboutImg = resourceKrankyBearChristmasGrinchPng
	}
	img := canvas.NewImageFromResource(aboutImg)
	img.FillMode = canvas.ImageFillContain
	img.SetMinSize(fyne.NewSize(150, 150))

	aboutText := fmt.Sprintf(`# %s

**Version:** %s

A cross-platform GUI log tail application for monitoring log files in real-time.

**Features:**
- Follow multiple log files in tabs
- Real-time file tailing with live updates
- Keyword highlighting for easy log scanning
- Optional audio notifications for matching lines, mp3 files in app dir/Resources/Sounds or browse to any others
- Cross-platform (Windows, Linux, macOS)

**Author:** %s

**Copyright:** %s

**Repository:** [https://github.com/amarillier/KrankyBearTailer](https://github.com/amarillier/KrankyBearTailer)

Built with [Fyne](https://fyne.io) and Go.

`, appName, appVersion, appAuthor, appCopyright)

	aboutContent := widget.NewRichTextFromMarkdown(aboutText)
	aboutContent.Wrapping = fyne.TextWrapWord

	// Create horizontal layout with image and text
	aboutScroll := container.NewScroll(aboutContent)
	aboutScroll.SetMinSize(fyne.NewSize(400, 400)) // Give it a minimum width and height

	// Use HBox with proper alignment to keep image top-left
	imgContainer := container.NewVBox(img) // VBox prevents centering
	combinedContent := container.NewHBox(imgContainer, aboutScroll)

	// Check if window already exists
	if aboutWin == nil {
		aboutWin = fyne.CurrentApp().NewWindow("About")
		// Set seasonal window icon
		{
			_, month, _ := time.Now().Date()
			if month == time.December {
				aboutWin.SetIcon(resourceKrankyBearChristmasGrinchPng)
			} else {
				aboutWin.SetIcon(resourceKrankyBearHogwartsSortingPng)
			}
		}
		aboutWin.Resize(fyne.NewSize(600, 350))
		aboutWin.SetContent(combinedContent)
		aboutWin.CenterOnScreen()
		aboutWin.SetCloseIntercept(func() {
			aboutWin.Close()
			aboutWin = nil
		})
		aboutWin.Show()
	} else {
		// Window already open - play easter egg
		aboutWin.Show()
		easterEgg(fyne.CurrentApp(), aboutWin)
	}
}

func main() {
	// On Linux, if running without a GUI (no DISPLAY or WAYLAND_DISPLAY), exit cleanly
	if runtime.GOOS == "linux" {
		if os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "" {
			fmt.Println("No GUI available (DISPLAY/WAYLAND_DISPLAY not set). Exiting.")
			return
		}
	}

	myApp := app.NewWithID("com.krankybeartailer.app")

	// Initialize speaker at startup for faster audio
	initSpeaker()

	w := myApp.NewWindow(appName)
	// Set seasonal window icon for main window
	{
		_, month, _ := time.Now().Date()
		if month == time.December {
			w.SetIcon(resourceKrankyBearChristmasGrinchPng)
		} else {
			w.SetIcon(resourceKrankyBearHogwartsSortingPng)
		}
	}
	w.Resize(fyne.NewSize(800, 600))
	w.CenterOnScreen()

	app := &App{
		w:           w,
		app:         myApp,
		fileTailers: make(map[string]*FileTailer),
		keywords:    []string{},
		pathForTab:  make(map[*container.TabItem]string),
	}

	// Create tabs container with close buttons
	app.tabs = container.NewDocTabs()

	// Load the embedded image - use Christmas image in December
	welcomeImg := resourceKrankyBearHogwartsSortingPng
	_, month, _ := time.Now().Date()
	if month == time.December {
		welcomeImg = resourceKrankyBearChristmasGrinchPng
	}
	img := canvas.NewImageFromResource(welcomeImg)
	img.FillMode = canvas.ImageFillContain
	img.SetMinSize(fyne.NewSize(150, 150))

	// Create welcome tab
	welcome := widget.NewRichTextFromMarkdown(fmt.Sprintf(`# %s

Welcome to %s!

**Features:**
- Follow multiple log files in tabs
- Real-time file tailing with live updates
- Keyword highlighting for easy log scanning
- Cross-platform (Windows, Linux, macOS)

**Getting Started:**
1. Click "File" → "Open File" or use Cmd+O (Mac) / Ctrl+O (Linux/Windows)
2. Select a text file to follow
3. Enter keywords in the field at the top and click "Add" or "Remove" to highlight or stop highlighting matching lines. Use the View button to see all watched keywords.
4. Open multiple files using tabs

**Tips:**
- Keywords are case-insensitive
- Click "Add" to highlight lines containing the keyword
- Click "Remove" to stop highlighting a keyword
- Click "Clear All" to remove all keywords
`, appName, appName))
	welcome.Wrapping = fyne.TextWrapWord

	// Create horizontal layout with image and text
	welcomeScroll := container.NewScroll(welcome)
	welcomeScroll.SetMinSize(fyne.NewSize(400, 0)) // Give it a minimum width

	// Use HBox with proper alignment to keep image top-left
	imgContainer := container.NewVBox(img) // VBox prevents centering
	welcomeContent := container.NewHBox(imgContainer, welcomeScroll)

	welcomeTab := container.NewTabItem("Welcome", welcomeContent)
	// Welcome tab should not be closable
	app.tabs.Append(welcomeTab)

	// Create toolbar with Open File button
	toolbar := container.NewBorder(
		nil,
		nil,
		widget.NewButtonWithIcon("Open File", theme.FolderOpenIcon(), func() {
			app.openFile()
		}),
		widget.NewLabel(""),
		nil,
	)

	// Main content with toolbar and tabs
	mainContent := container.NewBorder(toolbar, nil, nil, nil, app.tabs)
	w.SetContent(mainContent)

	// Menu bar
	fileMenu := fyne.NewMenu("File",
		fyne.NewMenuItem("Open File...", func() {
			app.openFile()
		}),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Quit", func() {
			myApp.Quit()
		}),
	)

	settingsMenu := fyne.NewMenu("Settings",
		fyne.NewMenuItem("Application Settings...", func() {
			showSettings(app.w, myApp)
		}),
		fyne.NewMenuItem("Theme Settings...", func() {
			showThemeSettings(app.w, myApp)
		}),
	)

	helpMenu := fyne.NewMenu("Help",
		fyne.NewMenuItem("Check for Updates...", func() {
			go func() {
				updtmsg, _ := updateChecker("amarillier", "KrankyBearTailer", "Kranky Bear Tailer", "https://github.com/amarillier/KrankyBearTailer/releases/latest")
				fyne.Do(func() {
					updateAlert(myApp, updtmsg)
				})
			}()
		}),
		fyne.NewMenuItem("About", func() {
			showAbout(app.w)
		}),
	)

	mainMenu := fyne.NewMainMenu(fileMenu, settingsMenu, helpMenu)
	w.SetMainMenu(mainMenu)

	// check update first (in background to not block startup)
	go func() {
		updtmsg, updateAvail := updateChecker("amarillier", "KrankyBearTailer", "Kranky Bear Tailer", "https://github.com/amarillier/KrankyBearTailer/releases/latest")
		if updateAvail {
			fyne.Do(func() {
				updateAlert(myApp, updtmsg)
			})
		}
	}()

	// Handle window close
	w.SetCloseIntercept(func() {
		// Save preferences before closing
		prefs := myApp.Preferences()

		// Save open files
		var filePaths []string
		for path := range app.fileTailers {
			filePaths = append(filePaths, path)
		}

		// Save file paths
		if len(filePaths) > 0 {
			prefs.SetStringList("open_files", filePaths)
		}

		// Save keywords and their colors for each file
		for path, ft := range app.fileTailers {
			keywords := ft.GetKeywords()
			if len(keywords) > 0 {
				key := "keywords_" + path
				prefs.SetStringList(key, keywords)
				// Save keyword colors
				keywordColors := ft.GetAllKeywordColors()
				if len(keywordColors) > 0 {
					saveKeywordColors(prefs, key, keywordColors)
				}
			}
		}

		// Stop all tailers
		for _, ft := range app.fileTailers {
			ft.Stop()
		}
		myApp.Quit()
	})

	// Restore preferences on startup (after window is set)
	go func() {
		// Small delay to ensure window is shown
		time.Sleep(100 * time.Millisecond)
		restorePreferences(app, myApp)
	}()

	// Setup system tray using Fyne's native desktop.App
	if desk, ok := myApp.(desktop.App); ok {
		show := fyne.NewMenuItem("Show", func() {
			w.Show()
			w.RequestFocus()
		})
		hide := fyne.NewMenuItem("Hide", w.Hide)

		openFile := fyne.NewMenuItem("Open File...", func() {
			app.openFile()
			w.Show()
			w.RequestFocus()
		})

		themeSettings := fyne.NewMenuItem("Theme Settings...", func() {
			showThemeSettings(w, myApp)
		})

		appSettings := fyne.NewMenuItem("Settings...", func() {
			showSettings(w, myApp)
		})

		checkUpdates := fyne.NewMenuItem("Check for Updates...", func() {
			go func() {
				updtmsg, _ := updateChecker("amarillier", "KrankyBearTailer", "Kranky Bear Tailer", "https://github.com/amarillier/KrankyBearTailer/releases/latest")
				fyne.Do(func() {
					updateAlert(myApp, updtmsg)
				})
			}()
		})

		about := fyne.NewMenuItem("About", func() {
			showAbout(w)
		})

		quit := fyne.NewMenuItem("Quit", myApp.Quit)

		menu := fyne.NewMenu(appName,
			show,
			hide,
			fyne.NewMenuItemSeparator(),
			openFile,
			fyne.NewMenuItemSeparator(),
			themeSettings,
			appSettings,
			fyne.NewMenuItemSeparator(),
			checkUpdates,
			about,
			fyne.NewMenuItemSeparator(),
			quit)
		desk.SetSystemTrayMenu(menu)

		// Set tray icon based on season
		_, month, _ := time.Now().Date()
		if month == time.December {
			desk.SetSystemTrayIcon(resourceKrankyBearChristmasGrinchPng)
		} else {
			desk.SetSystemTrayIcon(resourceKrankyBearHogwartsSortingPng)
		}
	}

	w.ShowAndRun()
}

func restorePreferences(app *App, myApp fyne.App) {
	prefs := myApp.Preferences()

	// Get saved file paths
	filePaths := prefs.StringListWithFallback("open_files", nil)

	if len(filePaths) == 0 {
		return
	}

	// Restore each file tab
	for _, filePath := range filePaths {
		// Check if file still exists
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			continue
		}

		// Get saved keywords BEFORE creating the tab
		key := "keywords_" + filePath
		keywords := prefs.StringListWithFallback(key, nil)

		// Restore keyword colors from preferences
		keywordColors := loadKeywordColors(prefs, key)
		// Ensure all keywords have colors (fill in any missing with default)
		for _, kw := range keywords {
			if _, exists := keywordColors[kw]; !exists {
				keywordColors[kw] = theme.PrimaryColor()
			}
		}

		// Add the file tab - this creates the FileTailer and loads last 20 lines
		app.addFileTab(filePath)

		// Restore keywords for this file
		// Note: Keywords are set BEFORE the tab content is displayed
		// so that loadLastLines applies highlighting correctly
		if len(keywordColors) > 0 && app.fileTailers[filePath] != nil {
			ft := app.fileTailers[filePath]
			// Set keywords with colors BEFORE loading content so highlighting works
			ft.mutex.Lock()
			ft.keywordColors = keywordColors
			ft.mutex.Unlock()

			// Now load the initial content with highlighting
			ft.LoadInitialContent()

			// Refresh highlighting to ensure all lines use the correct colors
			ft.refreshHighlighting()

			// Refresh to show the highlighting - use fyne.Do for thread safety
			fyne.Do(func() {
				ft.refreshLines()
			})
		}
	}
}

func showThemeSettings(parent fyne.Window, myApp fyne.App) {
	// Check if window already exists
	if themeWin != nil {
		themeWin.RequestFocus()
		return
	}

	// Use Fyne's built-in settings to allow theme customization
	// This gives users the ability to switch between Light/Dark themes
	// and customize appearance that affects ALL Fyne apps

	s := settings.NewSettings()
	themeWindow := myApp.NewWindow("Theme Settings - All Fyne Apps")
	// Set seasonal window icon
	{
		_, month, _ := time.Now().Date()
		if month == time.December {
			themeWindow.SetIcon(resourceKrankyBearChristmasGrinchPng)
		} else {
			themeWindow.SetIcon(resourceKrankyBearHogwartsSortingPng)
		}
	}
	themeWin = themeWindow
	themeWindow.Resize(fyne.NewSize(520, 520))
	themeWindow.CenterOnScreen()

	appearance := s.LoadAppearanceScreen(parent)

	// Add a helpful label
	infoLabel := widget.NewLabel("Changing theme affects ALL Fyne-based applications")
	infoLabel.Alignment = fyne.TextAlignCenter

	// Add Close button at the bottom
	closeButton := widget.NewButton("Close", func() {
		themeWindow.Close()
	})
	closeButton.Importance = widget.MediumImportance

	// Wrap appearance content with Close button at the bottom
	appearanceWithButton := container.NewBorder(
		nil,
		container.NewHBox(layout.NewSpacer(), closeButton),
		nil,
		nil,
		appearance,
	)

	tabs := container.NewAppTabs(
		&container.TabItem{
			Text: "Theme",
			Icon: s.AppearanceIcon(),
			Content: container.NewVBox(
				infoLabel,
				appearanceWithButton,
			),
		},
	)
	tabs.SetTabLocation(container.TabLocationLeading)
	themeWindow.SetContent(tabs)

	themeWindow.SetCloseIntercept(func() {
		themeWindow.Close()
		themeWin = nil
	})

	themeWindow.Show()
}

func showSettings(parent fyne.Window, myApp fyne.App) {
	if settingsWin != nil {
		settingsWin.RequestFocus()
		return
	}

	settingsWin = myApp.NewWindow("Settings")
	// Set seasonal window icon
	{
		_, month, _ := time.Now().Date()
		if month == time.December {
			settingsWin.SetIcon(resourceKrankyBearChristmasGrinchPng)
		} else {
			settingsWin.SetIcon(resourceKrankyBearHogwartsSortingPng)
		}
	}
	settingsWin.Resize(fyne.NewSize(400, 300))
	settingsWin.CenterOnScreen()

	// Sound enable checkbox
	soundCheckbox := widget.NewCheck("Enable sound notifications", func(enabled bool) {
		soundEnabled = enabled
		prefs := myApp.Preferences()
		prefs.SetBool("sound_enabled", soundEnabled)
	})
	soundCheckbox.SetChecked(soundEnabled)

	// Load saved sound enabled state
	prefs := myApp.Preferences()
	soundEnabled = prefs.BoolWithFallback("sound_enabled", true)
	soundCheckbox.SetChecked(soundEnabled)

	// Sound file selector
	soundFileLabel := widget.NewLabel("Sound file: " + soundFile)
	soundFileButton := widget.NewButton("Select Sound File...", func() {
		fd := dialog.NewFileOpen(func(read fyne.URIReadCloser, err error) {
			if err != nil || read == nil {
				return
			}
			defer read.Close()

			selectedPath := read.URI().Path()
			selectedFile := filepath.Base(selectedPath)

			// Only allow .mp3 files from Resources/Sounds
			if filepath.Ext(selectedFile) == ".mp3" {
				soundFile = selectedFile
				soundFileLabel.SetText("Sound file: " + soundFile)

				prefs := myApp.Preferences()
				prefs.SetString("sound_file", soundFile)
			} else {
				dialog.ShowError(fmt.Errorf("Please select a .mp3 file"), settingsWin)
			}
		}, settingsWin)

		// Set filter for .mp3 files
		fd.SetFilter(storage.NewExtensionFileFilter([]string{".mp3"}))

		// Try to set location to Resources/Sounds
		exePath, err := os.Executable()
		if err == nil {
			appDir := filepath.Dir(exePath)
			soundsDir := filepath.Join(appDir, "Resources", "Sounds")

			// Try to find the sounds directory from various locations
			if _, err := os.Stat(soundsDir); err == nil {
				uri, err := storage.Child(storage.NewFileURI(soundsDir), "")
				if err == nil && uri != nil {
					if luri, ok := uri.(fyne.ListableURI); ok {
						fd.SetLocation(luri)
					}
				}
			}
		}

		// Also try current working directory
		cwd, err := os.Getwd()
		if err == nil {
			soundsDir := filepath.Join(cwd, "Resources", "Sounds")
			if _, err := os.Stat(soundsDir); err == nil {
				uri, err := storage.Child(storage.NewFileURI(soundsDir), "")
				if err == nil && uri != nil {
					if luri, ok := uri.(fyne.ListableURI); ok {
						fd.SetLocation(luri)
					}
				}
			}
		}

		fd.Show()
	})

	// Load saved sound file
	soundFile = prefs.StringWithFallback("sound_file", "boing.mp3")
	soundFileLabel.SetText("Sound file: " + soundFile)

	// Close button
	closeButton := widget.NewButton("Close", func() {
		settingsWin.Close()
		settingsWin = nil
	})
	closeButton.Importance = widget.MediumImportance

	// Layout
	content := container.NewVBox(
		widget.NewLabel("Audio Settings"),
		widget.NewSeparator(),
		soundCheckbox,
		widget.NewSeparator(),
		soundFileLabel,
		soundFileButton,
		widget.NewSeparator(),
		container.NewHBox(layout.NewSpacer(), closeButton),
	)

	settingsWin.SetContent(content)
	settingsWin.SetCloseIntercept(func() {
		settingsWin.Close()
		settingsWin = nil
	})
	settingsWin.Show()
}

var (
	updt           fyne.Window
	kbimg          *canvas.Image
	settingsWin    fyne.Window
	themeWin       fyne.Window
	soundEnabled   = true
	soundFile      = "boing.mp3"
	aboutWin       fyne.Window
	keywordsWin    fyne.Window
	keywordsWinFor map[string]fyne.Window // Track which file each keywords window is for
)

func updateAlert(a fyne.App, updtmsg string) {
	// Check if window already exists
	if updt != nil {
		updt.RequestFocus()
		return
	}

	// open a window to show the update message
	releaselink, rerr := url.Parse("https://github.com/amarillier/KrankyBearTailer/releases/latest")
	if rerr != nil {
		fyne.LogError("Could not parse URL", rerr)
	}
	myreleaselink := widget.NewHyperlink("https://github.com/amarillier/KrankyBearTailer/releases/latest", releaselink)
	myreleaselink.Alignment = fyne.TextAlignLeading

	releasenoteslink, rnerr := url.Parse("https://github.com/amarillier/KrankyBearTailer/blob/main/ReleaseNotes.txt")
	if rnerr != nil {
		fyne.LogError("Could not parse URL", rnerr)
	}
	myreleasenoteslink := widget.NewHyperlink("https://github.com/amarillier/KrankyBearTailer/blob/main/ReleaseNotes.txt", releasenoteslink)
	myreleasenoteslink.Alignment = fyne.TextAlignLeading

	if strings.Contains(updtmsg, "newer version") {
		kbimg = canvas.NewImageFromResource(resourceKrankyBearHardHatPng)
		kbimg.FillMode = canvas.ImageFillOriginal
	} else if strings.Contains(updtmsg, "running the latest") {
		kbimg = canvas.NewImageFromResource(resourceKrankyBearHogwartsSortingPng)
		kbimg.FillMode = canvas.ImageFillOriginal
	} else {
		// For errors, just play a beep
		playBeep("up")
		kbimg = canvas.NewImageFromResource(resourceKrankyBearVikingHelmetPng)
		kbimg.FillMode = canvas.ImageFillOriginal
	}

	text := widget.NewLabel(updtmsg)
	content := container.NewVBox(kbimg, text, myreleaselink, myreleasenoteslink)
	updt = a.NewWindow(appName + ": Update Check")
	_, month, _ := time.Now().Date()
	if month == time.December {
		updt.SetIcon(resourceKrankyBearChristmasGrinchPng)
	} else {
		updt.SetIcon(resourceKrankyBearHogwartsSortingPng)
	}
	updt.Resize(fyne.NewSize(50, 100))
	updt.SetContent(content)
	updt.SetCloseIntercept(func() {
		updt.Close()
		updt = nil
	})
	// updt.CenterOnScreen() // run centered on primary (laptop) display
	updt.Show()
}

// "Now this is not even the end. It is not even the beginning of the end. But it is, perhaps, the end of the beginning." Winston Churchill, November 10, 1942
