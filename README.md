# ahMakeDir 專案交接文件

這份文件旨在幫助新加入的工程師（特別是 Go 語言新手）快速理解並上手 `ahMakeDir` 專案。

## 1. 專案結構 (Project Structure)

專案採用標準的 Go 專案結構（參考 [Go Project Layout](https://github.com/golang-standards/project-layout)）：

```text
ahMakeDir/
├── ahMakerdir.exe          # 編譯後的可執行檔 (Windows)
├── config.json             # 設定檔 (存儲上次使用的路徑與參數)
├── go.mod / go.sum         # Go Module 依賴管理文件
├── internal/               # 內部私有程式碼 (不對外開放)
│   ├── config/             # 設定檔讀寫邏輯 (Configuration)
│   ├── gui/                # UI 介面邏輯 (Fyne 框架)
│   └── logic/              # 核心業務邏輯 (Split, Compress, Upload)
├── ApiResults/             # 存放 API 回傳的成功 ID 列表 (自動生成)
└── test/                   # 測試相關文件 (如有)
```

*   **Entry Point (進入點)**: 雖然根目錄沒看到 `main.go`，但根據 `go.mod` `module ahMakerdir` 以及一般慣例，程式碼通常從 `main` package 開始。在這個專案結構中，可能在根目錄或透過 `internal/gui/main_window.go` 被呼叫。
*   **internal**: 核心邏輯都在這裡，`gui` 負責畫面，`logic` 負責圖片處理。

## 2. 拿到專案要做些什麼 (Getting Started)

這個專案是一個 **圖片自動化處理工具**，主要用於電商（推測是 AndenHud）的商品圖上架流程。

### 操作流程：
1.  **準備工作目錄 (WorkPath)**：
    *   你需要一個工作資料夾。
    *   裡面必須放一個 **Excel 檔 (.xlsx)**，定義商品與圖片的對應關係。
    *   裡面必須有一個 **圖片資料夾 (PictureDirName)**，預設可能叫 `2.修圖` 或其他，裡面放著原始商品圖。
2.  **執行程式**：
    *   打開 `ahMakerdir.exe` 或執行 `go run .`。
3.  **設定參數**：
    *   在 UI 上確認路徑設定是否正確。
    *   `SizeTablePath`: 尺寸表圖片的來源路徑。
    *   `ColorPicPath`: 色塊圖來源路徑 (L 欄需使用的圖片)。
    *   填寫 **FTP 資訊** 與 **API URL**。
4.  **執行分解 (Split)**：
    *   點擊 **"1. Run Split"**。
    *   程式會讀取 Excel，將圖片與色塊圖依照邏輯分配到 `BIG` (大圖), `SMALL` (小圖), `OUT` (輸出) 等資料夾，並產生 `manifest.json` 與尺寸表圖片。
5.  **執行壓縮 (Compress)**：
    *   點擊 **"2. Run Compress"**。
    *   程式會針對 `SMALL` 資料夾內的圖片進行縮圖與壓縮（保留 ICC Profile）。
6.  **執行上傳 (Upload)**：
    *   點擊 **"3. Run Upload"**。
    *   程式自動將圖片上傳至 FTP，並呼叫 API 進行資料庫對應 (處理回傳與自動清理)。

## 3. 環境 (Environment)

*   **作業系統**: Windows (專案目前主要在 Windows 上運行，路徑處理要注意 `\` vs `/`，雖然 `path/filepath` 會自動處理)。
*   **語言**: Go (Golang) 1.23+。
*   **GUI 框架要求**: 這個專案使用了 **Fyne**，它依賴 CGO。
    *   **必須安裝 GCC**: 在 Windows 上你需要安裝 [TDM-GCC](https://jmeubank.github.io/tdm-gcc/) 或 MinGW，並確保 `gcc` 指令在 PATH 環境變數中。

## 4. 解釋功能 (Feature Explanation)

### A. Split (圖片分派) - `internal/logic/split.go`
這是最核心的邏輯。
*   **讀取 Excel**: 程式會自動在 WorkPath 內找第一個 `.xlsx` 檔案。
*   **讀取圖片**: 掃描圖片資料夾，並依照 **數字順序** 排序。
*   **配對邏輯**:
    *   讀取 Excel 的每一列 (Row)。
    *   第 `I` 欄 (Col 8) 指定了該商品有幾張圖 (Step)。
    *   第 `J` 欄 (Col 9) 與 `K` 欄 (Col 10) 決定哪張圖是 `is_def` (預設圖)。
    *   第 `L` 欄 (Col 11) 指定 **色塊圖** 名稱(無須副檔名如.jpg)，程式會自動搜尋並複製。
    *   **Manifest 生成**: 產出 `manifest.json` 記錄圖片與料號對應關係。
    *   將圖片複製到依照 Excel 欄位 (Folder Name, Item ID, Color) 產生出的資料夾結構中。
    *   同時去 `SizeTablePath` 抓取對應的尺寸表圖片。

### B. Compress (圖片壓縮) - `internal/logic/compress.go`
針對分派後的圖片做優化。
*   **目標**: 掃描 Split 步驟產生的 `SMALL` 資料夾。
*   **縮放**: 依照介面設定的 `Width` 和 `Height` 進行縮圖。
*   **壓縮**: 轉存為 JPEG，並依照設定的 `Quality` (品質) 進行壓縮。
*   **色彩管理**: 程式有特殊邏輯 (利用 `go-iccjpeg`) 來提取並保留圖片的 **ICC Profile**，確保壓縮後顏色不失真（這在電商圖片很重要）。

### C. Upload (上傳與串接) - `internal/logic/upload.go`
負責與外部系統整合。
*   **FTP 上傳**: 自動將處理後的圖片上傳至 Company FTP。
*   **API 串接**: 呼叫 Laravel API 將圖片資訊寫入資料庫。
*   **自動清理**: API 若回傳無效料號 (`not_found_sns`)，程式會自動刪除 FTP 上的無用圖片。
*   **結果保存**: 成功寫入資料庫的 ID 會被記錄在 `ApiResults` 資料夾中。

## 5. 需安裝套件 (Required Packages)

主要依賴項 (在 `go.mod` 中)：

*   **UI 介面**: `fyne.io/fyne/v2` (跨平台 GUI 庫)
*   **圖片處理**: `github.com/disintegration/imaging` (強大的圖片處理庫，用來 Resize)
*   **Excel 處理**: `github.com/xuri/excelize/v2` (讀寫 Excel)
*   **JPEG ICC 支援**: `github.com/vimeo/go-iccjpeg` (處理 JPEG 內嵌的色彩設定檔)

安裝指令：
```bash
go mod download
go mod tidy
```

## 6. 你的補充 (Bonus Tips)

1.  **Excel 格式很關鍵**：
    *   程式邏輯高度依賴 Excel 的特定欄位順序（例如第 9 欄是張數，第 1、2 欄是資料夾名）。如果 Excel 格式變了，程式會壞掉或分錯圖。
    *   **Debug 技巧**: 如果分圖結果不對，先檢查 Excel 是否有多餘的空行，或者欄位順序是否跑掉。
2.  **CGO 編譯問題**：
    *   因為用到 Fyne，編譯時會比較慢，且必須有 GCC。
    *   建議編譯指令：`go build -ldflags -H=windowsgui` (隱藏 CMD 視窗)。
    *   OR : `go build -ldflags "-H=windowsgui -s -w -extldflags '-static'" -o ahMakeDir_static.exe .` (靜態編譯)。
3.  **UI 凍結問題**：
    *   目前的程式碼已使用 `go func() { ... }` 將耗時操作放到背景執行，並透過 `fyne.Do` 更新 UI，這是正確的做法，避免畫面卡死。
4.  **設定檔 (config.json)**：
    *   如果路徑設定一直跑掉，檢查 `config.json` 是否有寫入權限，或直接手動修改它。

---
*文件生成時間: 2025-01-15*

## 2026-02 功能更新補充 (Feature Updates)

本次更新重點在於自動化上傳流程與 API 整合：

1.  **Split 階段新增邏輯**:
    *   **色塊圖 (Color Pic)**: 自動讀取 Excel **L 欄** (Col 11) 的圖片名稱，並從設定的 `ColorPicPath` 複製該圖至 `SMALL` 資料夾，命名為 `[ItemCode]_Color.ext`。
    *   **預設圖判斷 (IsDef)**: 依據 Excel **J 欄** (=1) 與 **K 欄** (=2) 設定每張圖片的 `is_def` 屬性。
    *   **Manifest**: 生成 `manifest.json` 記錄圖片與料號的對應關係。

2.  **Upload 階段 (全新功能)**:
    *   **FTP 上傳**: 自動將 `SMALL` 資料夾內的圖片上傳至 FTP 伺服器 (`GoodsColor/YYYYMMDD/`)。
    *   **API 串接**:
        *   上傳完成後，自動呼叫 Laravel API (`/savePicDataFromGo`)。
        *   Payload 包含: `excel_col_d` (料號), `ftp_path` (/image/...), `sort`, `is_def`, `color_pic` (色塊圖路徑)。
    *   **自動清理**: 若 API 回傳 `not_found_sns` (無效料號)，程式會 **自動從 FTP 刪除** 該料號的所有圖片，確保 FTP 整潔。
    *   **結果保存**: 成功處理的 ID 會存入 `ApiResults/` 資料夾中。
    *   **錯誤提示**: 支援顯示後端回傳的 `massage` 錯誤訊息。

3.  **UI 調整**:
    *   新增 `Color Pic Path` 設定欄位。
    *   Log 視窗高度與字體優化，增強可讀性。
