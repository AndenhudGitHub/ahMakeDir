package logic

import (
	"ahMakerdir/internal/config"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jlaffaye/ftp"
)

// RunUpload handles FTP upload of the SMALL directory and calls Laravel API
func RunUpload(cfg config.Config, log func(string)) error {
	log("Connecting to FTP...")
	c, err := ftp.Dial(cfg.FtpHost+":"+cfg.FtpPort, ftp.DialWithTimeout(10*time.Second))
	if err != nil {
		return fmt.Errorf("FTP dial error: %v", err)
	}
	defer c.Quit()

	if err := c.Login(cfg.FtpUser, cfg.FtpPassword); err != nil {
		return fmt.Errorf("FTP login error: %v", err)
	}
	log("FTP Connected successfully.")

	// Find all SMALL directories
	var sourceDirs []string
	log(fmt.Sprintf("Scanning for SMALL directories in %s...", cfg.WorkPath))
	filepath.WalkDir(cfg.WorkPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() && strings.EqualFold(d.Name(), "SMALL") {
			sourceDirs = append(sourceDirs, path)
			return filepath.SkipDir // Don't search inside SMALL
		}
		return nil
	})

	if len(sourceDirs) == 0 {
		return fmt.Errorf("no SMALL directories found in %s", cfg.WorkPath)
	}

	log(fmt.Sprintf("Found %d SMALL directories.", len(sourceDirs)))

	var uploadedFiles []string
	
	// Target format: GoodsColor/YYYYMMDD/filename
	uploadDate := time.Now().Format("20060102")
	targetRoot := "GoodsColor"
	remoteDir := fmt.Sprintf("%s/%s", targetRoot, uploadDate)

	// Ensure remote directory exists once
	if err := ensureFtpDir(c, remoteDir); err != nil {
		log(fmt.Sprintf("Warning: Could not create remote dir %s: %v", remoteDir, err))
	}

	// Load Manifest
	manifestPath := filepath.Join(cfg.WorkPath, "manifest.json")
	manifest := make(map[string]ImageMetadata)
	if manifestData, err := os.ReadFile(manifestPath); err == nil {
		json.Unmarshal(manifestData, &manifest)
	} else {
		log(fmt.Sprintf("Warning: Could not load manifest.json: %v. API data might be incomplete.", err))
	}

	type ApiPayloadItem struct {
		ExcelColD    string `json:"excel_col_d"`
		FtpPath      string `json:"ftp_path"`
		Sort         int    `json:"sort"`
		IsDef        int    `json:"is_def"`
		ColorPic     string `json:"color_pic,omitempty"`
	}
	apiPayload := make(map[string]ApiPayloadItem) // Changed to map as requested

	for _, sourceDir := range sourceDirs {
		// Walk through each source directory
		err = filepath.WalkDir(sourceDir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}

			// Get filename
			filename := filepath.Base(path)
			
			// Remote path
			remotePath := fmt.Sprintf("%s/%s", remoteDir, filename)
			// Path to store in DB (with /image/ prefix)
			storedPath := fmt.Sprintf("/image/%s", remotePath)
			
			// Open local file
			f, err := os.Open(path)
			if err != nil {
				log(fmt.Sprintf("Failed to open local file %s: %v", d.Name(), err))
				return nil
			}
			defer f.Close()

			// Upload file
			//log(fmt.Sprintf("Uploading %s -> %s", filename, remotePath))
			err = c.Stor(remotePath, f)
			if err != nil {
				log(fmt.Sprintf("Failed to upload %s: %v", filename, err))
				return nil
			}

			// Add to API payload
			meta, ok := manifest[filename]
			
			// Don't error if not found, it might be the color pic itself being uploaded (which is not a key in manifest usually, or handle differently?)
			// Or maybe the color pic IS in the manifest with its own key? 
			// Wait, the manifest keyed by "newFilename" (Item_01.jpg). The Color Pic (Item_Color.jpg) is ALSO in the folder.
			// When walk hits "Item_Color.jpg", it won't be in manifest keys (because split logic key is `newFilename`).
			// So `ok` will be false. We should skip adding payload for the color pic file itself, 
			// BUT we need to add the color pic PATH to the payloads of Item01, Item02...
			
			if !ok {
				// Likely a file we copied there (like the Color Pic itself) but not one of the main items.
				// Or an untracked file. Just skip creating a payload item for it.
			} else {
				// Update manifest with FTP path
				meta.FtpPath = storedPath
				manifest[filename] = meta
				
				// Calculate Color Pic Remote Path if exists
				colorPicPath := ""
				if meta.ColorPicFilename != "" {
					// The color pic was copied to the same remote dir
					colorPicRemotePath := fmt.Sprintf("%s/%s", remoteDir, meta.ColorPicFilename)
					colorPicPath = fmt.Sprintf("/image/%s", colorPicRemotePath)
				}

				apiPayload[filename] = ApiPayloadItem{
					ExcelColD:    meta.ExcelColD,
					FtpPath:      storedPath,
					Sort:         meta.Sort,
					IsDef:        meta.IsDef,
					ColorPic:     colorPicPath,
				}
			}
			
			uploadedFiles = append(uploadedFiles, filename)
			return nil
		})
		if err != nil {
			log(fmt.Sprintf("Error walking source dir %s: %v", sourceDir, err))
		}
	}

	log(fmt.Sprintf("Uploaded %d files.", len(uploadedFiles)))

	// Update manifest.json with FTP paths
	if updatedManifest, err := json.MarshalIndent(manifest, "", "  "); err == nil {
		if err := os.WriteFile(manifestPath, updatedManifest, 0644); err != nil {
			log(fmt.Sprintf("Warning: Failed to save updated manifest.json: %v", err))
		} else {
			log("Updated manifest.json with FTP paths.")
		}
	}

	// Debug: Log payload regardless of API URL
	//debugPayload, _ := json.MarshalIndent(apiPayload, "", "  ")
	//log(fmt.Sprintf("Payload: %s", string(debugPayload)))

	// Call Laravel API
	if cfg.ApiUrl != "" {
		log("Calling Laravel API...")
		
		type ApiResponse struct {
			Status                  string   `json:"status"`
			Message                 string   `json:"massage"` // Match user's PHP typo
			NotFoundSNs             []string `json:"not_found_sns"`
			SuccessGoodsColorPicIDs []int    `json:"success_goods_color_pic_ids"`
			SuccessGoodsColorIDs    []int    `json:"success_goods_color_ids"`
		}

		respBody, err := callLaravelAPI(cfg.ApiUrl, apiPayload)
		if err != nil {
			// Try to parse error message from JSON body
			var apiErrResp ApiResponse
			if jsonErr := json.Unmarshal([]byte(respBody), &apiErrResp); jsonErr == nil && apiErrResp.Message != "" {
				log("---------------------------------------------------")
				log(fmt.Sprintf("API SERVER ERROR: %s", apiErrResp.Message))
				log("---------------------------------------------------")
			} else {
				log(fmt.Sprintf("API Error: %v", err))
			}
		} else {
			log("API notification sent successfully.")
			
			var apiResp ApiResponse
			if jsonErr := json.Unmarshal([]byte(respBody), &apiResp); jsonErr == nil {
				// Handle Not Found Warnings & Cleanup FTP
				if len(apiResp.NotFoundSNs) > 0 {
					log("---------------------------------------------------")
					log(fmt.Sprintf("WARNING: %d Items Not Found in Database. Cleaning up FTP...", len(apiResp.NotFoundSNs)))
					
					// Create map for SN lookup
					missingSNs := make(map[string]bool)
					for _, sn := range apiResp.NotFoundSNs {
						missingSNs[sn] = true
						log(fmt.Sprintf(" - %s (Not found, deleting from FTP)", sn))
					}

					deletedCount := 0
					deletedPaths := make(map[string]bool)

					for _, item := range apiPayload {
						if missingSNs[item.ExcelColD] {
							// 1. Delete main image
							if item.FtpPath != "" && !deletedPaths[item.FtpPath] {
								ftpPath := strings.TrimPrefix(item.FtpPath, "/image/")
								if err := c.Delete(ftpPath); err == nil {
									deletedCount++
								}
								deletedPaths[item.FtpPath] = true
							}

							// 2. Delete color pic
							if item.ColorPic != "" && !deletedPaths[item.ColorPic] {
								ftpColorPath := strings.TrimPrefix(item.ColorPic, "/image/")
								c.Delete(ftpColorPath) // Delete silently
								deletedPaths[item.ColorPic] = true
							}
						}
					}
					if deletedCount > 0 {
						log(fmt.Sprintf("Successfully removed %d invalid images from FTP.", deletedCount))
					}
					log("---------------------------------------------------")
				}

				// Prepare Results Directory
				resultsDir := filepath.Join(cfg.WorkPath, "ApiResults")
				if _, err := os.Stat(resultsDir); os.IsNotExist(err) {
					os.MkdirAll(resultsDir, 0755)
				}
				timestamp := time.Now().Format("20060102_150405")

				// Helper to save ID list
				saveIDs := func(name string, ids []int) {
					if len(ids) > 0 {
						fileName := fmt.Sprintf("%s_%s.json", name, timestamp)
						filePath := filepath.Join(resultsDir, fileName)
						if data, err := json.MarshalIndent(ids, "", "  "); err == nil {
							if err := os.WriteFile(filePath, data, 0644); err == nil {
								log(fmt.Sprintf("Saved %d %s to: %s", len(ids), name, fileName))
							} else {
								log(fmt.Sprintf("Error saving %s: %v", name, err))
							}
						}
					}
				}

				// Save GoodsColorPic IDs
				saveIDs("success_goods_color_pic_ids", apiResp.SuccessGoodsColorPicIDs)
				
				// Save GoodsColor IDs
				saveIDs("success_goods_color_ids", apiResp.SuccessGoodsColorIDs)
			}
		}
	} else {
		log("Skipping API call (URL not set).")
	}

	return nil
}



func callLaravelAPI(url string, payload interface{}) (string, error) {
	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("key", "salesAh29078955") // Auth key requested by user

	client := &http.Client{Timeout: 30 * time.Second} // Increased timeout for dd() which might be slow or large
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %v", err)
	}
	bodyString := string(bodyBytes)

	if resp.StatusCode >= 400 {
		return bodyString, fmt.Errorf("API returned status: %s. Body: %s", resp.Status, bodyString)
	}

	return bodyString, nil
}

// ensureFtpDir checks if simple directory structure exists, creating it if not.
func ensureFtpDir(c *ftp.ServerConn, path string) error {
	currentDir, _ := c.CurrentDir()
	
	if err := c.ChangeDir(path); err == nil {
		c.ChangeDir(currentDir)
		return nil
	}

	// Simple approach: Split by slash and traverse
	parts := strings.Split(path, "/")
	buildPath := ""
	
	for _, part := range parts {
		if part == "" { continue }
		buildPath = buildPath + part + "/"
		c.MakeDir(buildPath) 
	}
	
	return nil
}

