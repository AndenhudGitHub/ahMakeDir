package logic

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"ahMakerdir/internal/config"

	"github.com/xuri/excelize/v2"
)

// RunSplit executes the image splitting logic
func RunSplit(cfg config.Config, progress func(string)) ([]string, error) {
	progress("Starting Split Process...")

	dirPath := strings.TrimSpace(cfg.WorkPath)
	specPath := strings.TrimSpace(cfg.SizeTablePath)

	// Scan directory for Excel and Image folders
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read work path: %w", err)
	}

	var excelFile string
	for _, entry := range entries {
		if !entry.IsDir() && strings.Contains(entry.Name(), ".xlsx") && !strings.HasPrefix(entry.Name(), "~$") {
			excelFile = entry.Name()
			break
		}
	}

	if excelFile == "" {
		return nil, fmt.Errorf("no Excel file found in %s", dirPath)
	}
	progress(fmt.Sprintf("Found Excel file: %s", excelFile))

	// Read Images
	imagePath := filepath.Join(dirPath, cfg.PictureDirName)
	imageFiles, err := scanDirSort(imagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read image directory: %w", err)
	}

	// Filter for images
	var imagePicArr []string
	for _, file := range imageFiles {
		lowerFile := strings.ToLower(file)
		if strings.HasSuffix(lowerFile, ".jpg") || strings.HasSuffix(lowerFile, ".png") {
			imagePicArr = append(imagePicArr, file)
		}
	}
	progress(fmt.Sprintf("Found %d images", len(imagePicArr)))

	// Read Excel
	xlsx, err := excelize.OpenFile(filepath.Join(dirPath, excelFile))
	if err != nil {
		return nil, fmt.Errorf("failed to open Excel file: %w", err)
	}
	defer xlsx.Close()

	sheetName := xlsx.GetSheetName(xlsx.GetActiveSheetIndex())
	rows, err := xlsx.GetRows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("failed to get rows: %w", err)
	}

	// Skip header if necessary? Original code didn't seem to skip explicitly,
	// but usually row 0 is header. Original code: `for index, row := range rows`
	// and used `index` logic. It seems row 0 IS data in original code or they just processed it.
	// Wait, original code: `for _, row := range rows` (line 114) to count total.
	// Then `for index, row := range rows` (line 131).
	// Let's assume all rows are data for now to match original behavior,
	// OR check if the first row looks like a header.
	// Original code logic: `i, err := strconv.Atoi(row[8])`. If row[8] is "Count" (header), this would fail.
	// So likely the Excel file has NO header or the user knows to not include it.
	// However, `strconv.Atoi` error in original code just printed "轉換失敗!!" and continued.

	begin := 0
	end := 0
	var smallDirs []string
	var failSizeTable []string
	manifest := make(map[string]ImageMetadata)

	for index, row := range rows {
		if len(row) < 9 {
			continue // Skip invalid rows
		}

		step, err := strconv.Atoi(row[8])
		if err != nil {
			progress(fmt.Sprintf("Row %d: Invalid step count (col I), skipping.", index+1))
			continue
		}

		if index == 0 {
			end = step - 1
		} else {
			end = end + step
		}

		// Directory paths
		// row[0]: Folder Name 1
		// row[1]: Folder Name 2
		// row[3]: Item ID?
		// row[6]: Color?

		// Clean row[6]
		row[6] = strings.ReplaceAll(row[6], "/", "")

		level1 := filepath.Join(dirPath, row[0]+"_"+row[1])
		level2 := filepath.Join(level1, row[3]+"_"+row[6])
		level3 := filepath.Join(level2, "BIG")
		level4 := filepath.Join(level2, "SMALL")
		level15 := filepath.Join(level1, "OUT")

		ensureDir(level1)
		ensureDir(level2)
		ensureDir(level3)
		ensureDir(level4)
		ensureDir(level15)

		// Copy Size Table
		styleNo := strings.Split(row[2], "-")[0]
		styleNoPath := filepath.Join(specPath, styleNo+".jpg")
		destSizeTable := filepath.Join(level15, row[3]+"_"+styleNo+".jpg")

		if err := copyFile(styleNoPath, destSizeTable); err != nil {
			failSizeTable = append(failSizeTable, fmt.Sprintf("Failed to copy size table: %s", styleNoPath))
		}


		// Copy Images
		count := 1
		for i := begin; i <= end && i < len(imagePicArr); i++ {
			originalName := imagePicArr[i]
			srcImg := filepath.Join(imagePath, originalName)
			
			// New filename base
			newFilename := fmt.Sprintf("%s_0%d.jpg", row[3], count)

			// BIG
			destBig := filepath.Join(level3, newFilename)
			copyFile(srcImg, destBig)

			// SMALL
			destSmall := filepath.Join(level4, newFilename)
			copyFile(srcImg, destSmall)

			// OUT
			destOut := filepath.Join(level15, newFilename)
			copyFile(srcImg, destOut)

			// Calculate IsDef
			// Col J (index 9) -> IsDef = 1
			// Col K (index 10) -> IsDef = 2
			isDef := 0
			
			// Helper to get value from column
			getColVal := func(colIndex int) int {
				if len(row) > colIndex {
					if val, err := strconv.Atoi(row[colIndex]); err == nil {
						return val
					}
				}
				return -1 // Not found or invalid
			}

			valJ := getColVal(9)
			valK := getColVal(10)

			if count == valJ {
				isDef = 1
			} else if count == valK {
				isDef = 2
			}

			// Handle Color Pic (Col L / Index 11)
			colorPicName := ""
			if len(row) > 11 {
				colorPicInput := strings.TrimSpace(row[11])
				if colorPicInput != "" {
						
						// Determine extension and source path
						ext := filepath.Ext(colorPicInput)
						srcColorPic := ""
						
						if ext != "" {
							// User provided extension
							srcColorPic = filepath.Join(cfg.ColorPicPath, colorPicInput)
							if _, err := os.Stat(srcColorPic); os.IsNotExist(err) {
								progress(fmt.Sprintf("Warning: Color pic not found: %s", srcColorPic))
								srcColorPic = ""
							}
						} else {
							// No extension provided, try .jpg then .png
							tryJpg := filepath.Join(cfg.ColorPicPath, colorPicInput+".jpg")
							tryPng := filepath.Join(cfg.ColorPicPath, colorPicInput+".png")
							
							if _, err := os.Stat(tryJpg); err == nil {
								srcColorPic = tryJpg
								ext = ".jpg"
							} else if _, err := os.Stat(tryPng); err == nil {
								srcColorPic = tryPng
								ext = ".png"
							} else {
								progress(fmt.Sprintf("Warning: Color pic not found (tried .jpg/.png): %s", colorPicInput))
							}
						}

						if srcColorPic != "" {
							// Copy to SMALL
							// Target name: row[3] (ItemCode) + "_Color" + ext
							destColorPicName := fmt.Sprintf("%s_Color%s", row[3], ext)
							destColorPicPath := filepath.Join(level4, destColorPicName)
							
							if err := copyFile(srcColorPic, destColorPicPath); err != nil {
								progress(fmt.Sprintf("Warning: Failed to copy color pic: %v", err))
							} else {
								colorPicName = destColorPicName
							}
						}
				}
			}

			// Record to manifest
			manifest[newFilename] = ImageMetadata{
				ExcelColD:        row[3],
				Sort:             count,
				IsDef:            isDef,
				ColorPicFilename: colorPicName,
			}

			count++
		}

		smallDirs = append(smallDirs, level4)
		begin = begin + step
		progress(fmt.Sprintf("Processed %s", row[3]))
	}

	if len(failSizeTable) > 0 {
		return smallDirs, fmt.Errorf("completed with errors: %v", failSizeTable)
	}

	// Save Manifest (Standard)
	manifestPath := filepath.Join(dirPath, "manifest.json")
	manifestData, err := json.MarshalIndent(manifest, "", "  ")
	if err == nil {
		if err := os.WriteFile(manifestPath, manifestData, 0644); err != nil {
			progress(fmt.Sprintf("Warning: Failed to save manifest.json: %v", err))
		}
	} else {
		progress(fmt.Sprintf("Warning: Failed to marshal manifest: %v", err))
	}

	progress("Split Process Complete.")
	return smallDirs, nil
}

// ImageMetadata holds info for API upload
type ImageMetadata struct {
	ExcelColD string `json:"excel_col_d"`
	FtpPath   string `json:"ftp_path,omitempty"`
	Sort      int    `json:"sort"`
	IsDef     int    `json:"is_def"`
	ColorPicFilename string `json:"color_pic_filename,omitempty"`
}

// Helper functions

func ensureDir(path string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.MkdirAll(path, 0755)
	}
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

// Sorting logic from original main.go

type byNumericalFilename []os.DirEntry

func (nf byNumericalFilename) Len() int      { return len(nf) }
func (nf byNumericalFilename) Swap(i, j int) { nf[i], nf[j] = nf[j], nf[i] }
func (nf byNumericalFilename) Less(i, j int) bool {
	pathA := nf[i].Name()
	pathB := nf[j].Name()

	isImgA := strings.HasSuffix(strings.ToLower(pathA), ".jpg") || strings.HasSuffix(strings.ToLower(pathA), ".png")
	isImgB := strings.HasSuffix(strings.ToLower(pathB), ".jpg") || strings.HasSuffix(strings.ToLower(pathB), ".png")

	if isImgA && isImgB {
		aStart := strings.LastIndex(pathA, "(")
		aEnd := strings.LastIndex(pathA, ")")
		bStart := strings.LastIndex(pathB, "(")
		bEnd := strings.LastIndex(pathB, ")")

		if aStart != -1 && aEnd != -1 && bStart != -1 && bEnd != -1 {
			a, err1 := strconv.ParseInt(pathA[aStart+1:aEnd], 10, 64)
			b, err2 := strconv.ParseInt(pathB[bStart+1:bEnd], 10, 64)
			if err1 == nil && err2 == nil {
				return a < b
			}
		}
	}
	return pathA < pathB
}

func scanDirSort(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	sort.Sort(byNumericalFilename(entries))

	var files []string
	for _, entry := range entries {
		files = append(files, entry.Name())
	}
	return files, nil
}
