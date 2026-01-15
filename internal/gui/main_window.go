package gui

import (
	"fmt"

	"ahMakerdir/internal/config"
	"ahMakerdir/internal/logic"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// RunApp initializes and runs the Fyne application
func RunApp() {
	myApp := app.New()
	myWindow := myApp.NewWindow("ahMakeDir - Image Processor")
	myWindow.Resize(fyne.NewSize(800, 600))

	// Load Config
	cfgPath := config.GetConfigPath()
	cfg, err := config.Load(cfgPath)
	if err != nil {
		dialog.ShowError(fmt.Errorf("failed to load config: %v", err), myWindow)
		cfg = config.DefaultConfig()
	}

	// UI Components

	// Config Inputs
	workPathEntry := widget.NewEntry()
	workPathEntry.SetText(cfg.WorkPath)

	picDirEntry := widget.NewEntry()
	picDirEntry.SetText(cfg.PictureDirName)

	sizeTablePathEntry := widget.NewEntry()
	sizeTablePathEntry.SetText(cfg.SizeTablePath)

	widthEntry := widget.NewEntry()
	widthEntry.SetText(cfg.Width)

	heightEntry := widget.NewEntry()
	heightEntry.SetText(cfg.Height)

	qualityEntry := widget.NewEntry()
	qualityEntry.SetText(fmt.Sprintf("%d", cfg.Quality))

	// Log Area - using RichText for better text visibility
	logText := widget.NewRichText()
	logText.Wrapping = fyne.TextWrapWord

	// Create a scroll container for the log
	logScroll := container.NewScroll(logText)
	logScroll.SetMinSize(fyne.NewSize(0, 400)) // Set minimum height

	logFunc := func(msg string) {
		// Use fyne.Do to ensure this runs on the main thread
		fyne.Do(func() {
			// Create text segment with foreground color (will be white in dark theme)
			segment := &widget.TextSegment{
				Text: msg + "\n",
				Style: widget.RichTextStyle{
					ColorName: theme.ColorNameForeground,
					TextStyle: fyne.TextStyle{},
				},
			}
			logText.Segments = append(logText.Segments, segment)
			logText.Refresh()

			// Scroll to bottom
			logScroll.ScrollToBottom()
		})
	}

	// Buttons
	saveBtn := widget.NewButton("Save Config", func() {
		cfg.WorkPath = workPathEntry.Text
		cfg.PictureDirName = picDirEntry.Text
		cfg.SizeTablePath = sizeTablePathEntry.Text
		cfg.Width = widthEntry.Text
		cfg.Height = heightEntry.Text
		fmt.Sscanf(qualityEntry.Text, "%d", &cfg.Quality)

		if err := config.Save(cfgPath, cfg); err != nil {
			dialog.ShowError(err, myWindow)
		} else {
			logFunc("Configuration saved.")
		}
	})

	var smallDirs []string // Store result from split to pass to compress

	runSplitBtn := widget.NewButton("1. Run Split", func() {
		logFunc("--- Starting Split ---")
		// Update config from UI before running
		cfg.WorkPath = workPathEntry.Text
		cfg.PictureDirName = picDirEntry.Text
		cfg.SizeTablePath = sizeTablePathEntry.Text

		go func() {
			var err error
			smallDirs, err = logic.RunSplit(cfg, func(msg string) {
				logFunc(msg)
			})

			if err != nil {
				dialog.ShowError(err, myWindow)
				logFunc(fmt.Sprintf("Error: %v", err))
			} else {
				// Success Alert
				fyne.Do(func() {
					dialog.ShowInformation("Done", "Split Process Completed!", myWindow)
				})
				logFunc("--- Split Completed ---")
			}
		}()
	})

	runCompressBtn := widget.NewButton("2. Run Compress", func() {
		logFunc("--- Starting Compress ---")
		// Update config from UI
		cfg.Width = widthEntry.Text
		cfg.Height = heightEntry.Text
		fmt.Sscanf(qualityEntry.Text, "%d", &cfg.Quality)

		go func() {
			// If smallDirs is empty (user restarted app), logic.RunCompress will scan
			err := logic.RunCompress(smallDirs, cfg, func(msg string) {
				logFunc(msg)
			})
			if err != nil {
				dialog.ShowError(err, myWindow)
				logFunc(fmt.Sprintf("Error: %v", err))
			} else {
				// Success Alert
				fyne.Do(func() {
					dialog.ShowInformation("Done", "Compression Process Completed!", myWindow)
				})
				logFunc("--- Compress Completed ---")
			}
		}()
	})

	runAllBtn := widget.NewButton("Run ALL", func() {
		logFunc("--- Running ALL ---")
		runSplitBtn.OnTapped()
		// runCompressBtn.OnTapped() // Avoid parallel run for now
	})

	// Layout
	form := container.New(layout.NewFormLayout(),
		widget.NewLabel("Work Path:"), workPathEntry,
		widget.NewLabel("Picture Dir Name:"), picDirEntry,
		widget.NewLabel("Size Table Path:"), sizeTablePathEntry,
		widget.NewLabel("Resize Width:"), widthEntry,
		widget.NewLabel("Resize Height:"), heightEntry,
		widget.NewLabel("Quality (0-100):"), qualityEntry,
	)

	actions := container.NewHBox(saveBtn, layout.NewSpacer(), runSplitBtn, runCompressBtn, runAllBtn)

	topContainer := container.NewVBox(widget.NewLabel("Configuration"), form, actions)
	bottomContainer := container.NewVBox(widget.NewLabel("Logs"), logScroll)

	// Use VSplit for resizable log area
	split := container.NewVSplit(topContainer, bottomContainer)
	split.SetOffset(0.4) // Give 40% space to config, rest to logs

	myWindow.SetContent(split)
	myWindow.ShowAndRun()
}
