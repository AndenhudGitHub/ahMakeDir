package logic

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"ahMakerdir/internal/config"

	"github.com/disintegration/imaging"
)

// RunCompress executes the image compression logic
func RunCompress(targetDirs []string, cfg config.Config, progress func(string)) error {
	progress("Starting Compression Process...")

	width, _ := strconv.Atoi(cfg.Width)
	height, _ := strconv.Atoi(cfg.Height)
	quality := cfg.Quality
	if quality == 0 {
		quality = 85
	}

	// If no target dirs provided, scan for them
	if len(targetDirs) == 0 {
		progress("No target directories provided, scanning for 'SMALL' folders...")
		targetDirs = findSmallDirs(cfg.WorkPath)
	}

	progress(fmt.Sprintf("Found %d directories to process", len(targetDirs)))

	for _, dir := range targetDirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			progress(fmt.Sprintf("Error reading dir %s: %v", dir, err))
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			lowerName := strings.ToLower(entry.Name())
			if !strings.HasSuffix(lowerName, ".jpg") && !strings.HasSuffix(lowerName, ".png") {
				continue
			}

			filePath := filepath.Join(dir, entry.Name())

			// Resize
			err := resizeImage(filePath, width, height, quality)
			if err != nil {
				progress(fmt.Sprintf("Failed to resize %s: %v", entry.Name(), err))
			} else {
				// progress(fmt.Sprintf("Resized %s", entry.Name())) // Too verbose?
			}
		}
		progress(fmt.Sprintf("Completed directory: %s", filepath.Base(dir)))
	}

	progress("Compression Process Complete.")
	return nil
}

func resizeImage(path string, width, height, quality int) error {
	// Open file for reading ICC profile
	file, err := os.Open(path)
	if err != nil {
		return err
	}

	// Extract ICC profile
	profile, err := extractICCProfile(file)
	file.Close() // Close immediately to release lock

	if err != nil {
		// It's okay if there is no profile or if extraction fails, we just proceed without it
		// But maybe we should log it? For now, ignore error as per original PHP behavior (mostly)
		// PHP: if ($MyJpeg->LoadFromJPEG($filePath)) ...
		profile = nil
	}
	// Reset file pointer for imaging.Open (which takes a filename, so it opens it again)
	// actually imaging.Open takes a filename.

	// Open image for resizing
	src, err := imaging.Open(path)
	if err != nil {
		return err
	}

	// Resize
	dst := imaging.Resize(src, width, height, imaging.Lanczos)

	// Save to temp buffer first to embed ICC
	buf := new(bytes.Buffer)
	err = imaging.Encode(buf, dst, imaging.JPEG, imaging.JPEGQuality(quality))
	if err != nil {
		return err
	}

	// If we have a profile, embed it
	var finalOutput io.Reader = buf
	if len(profile) > 0 {
		outBuf := new(bytes.Buffer)
		if err := embedICCProfile(outBuf, buf, profile); err == nil {
			finalOutput = outBuf
		} else {
			// If embedding fails, fallback to image without profile
			// Maybe log error?
		}
	}

	// Save to temp file
	tempPath := path + ".tmp"
	out, err := os.Create(tempPath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, finalOutput)
	if err != nil {
		return err
	}

	// Close file before rename
	out.Close()

	// Overwrite original
	return os.Rename(tempPath, path)
}

func findSmallDirs(root string) []string {
	var dirs []string
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() && info.Name() == "SMALL" {
			dirs = append(dirs, path)
		}
		return nil
	})
	return dirs
}
