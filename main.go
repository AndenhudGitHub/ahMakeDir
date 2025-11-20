package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

type ByNumericalFilename []os.DirEntry

func main() {

	// 獲取執行文件的目錄
	execPath, err := os.Executable()
	if err != nil {
		fmt.Printf("Error getting executable path: %v\n", err)
		os.Exit(3)
	}
	execDir := filepath.Dir(execPath)

	configPath := filepath.Join(execDir, "config.json")

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// 如果執行文件目錄找不到 config.json，嘗試從當前工作目錄加載
		configPath = "config.json"
	}

	//尺寸表路徑
	data, err := os.ReadFile(configPath)
	if err != nil {
		fmt.Print(err)
		os.Exit(3)
	}
	type config struct {
		WorkPath       string `json:"WorkPath"`
		PictureDirName string `json:"PictureDirName"`
		SizeTablePath  string `json:"SizeTablePath"`
	}
	var obj config
	err = json.Unmarshal(data, &obj)
	if err != nil {
		fmt.Println("error:", err)
		os.Exit(3)
	}

	// 非 Windows: 直接使用 TrimSpace
	DirPath := strings.TrimSpace(obj.WorkPath)
	SpecPath := strings.TrimSpace(obj.SizeTablePath)

	if runtime.GOOS == "windows" {
		DirPath = strings.Replace(obj.WorkPath, "\\", "\\\\", -1)
		SpecPath = strings.Replace(obj.SizeTablePath, "\\", "\\\\", -1)
	}

	//掃描DIR
	dirArr := scandir(DirPath)
	//excel資料夾檔名
	var excelArr []string
	//圖片資料夾檔名
	var imageDirArr []string
	//掃描圖片儲存陣列
	var imagePicArr []string
	//掃描第一層有的 EXCEL 及 圖片資料夾
	for _, file := range dirArr {
		if strings.Index(file, ".xlsx") > -1 {
			excelArr = append(excelArr, file)
		} else {
			imageDirArr = append(imageDirArr, file)
		}
	}

	imgDirName := obj.PictureDirName

	//讀圖片 塞入陣列
	imagePath := DirPath + string(os.PathSeparator) + imgDirName
	imageFileList := scandir_sort(imagePath)

	for _, file := range imageFileList {
		if strings.Index(file, ".jpg") > -1 || strings.Index(file, ".png") > -1 {
			imagePicArr = append(imagePicArr, file)
		}
	}

	
	//讀excel 跑陣列
	excelPath := DirPath + string(os.PathSeparator) + excelArr[0]
	xlsx, err := excelize.OpenFile(excelPath)
	if err != nil {
		fmt.Println(err)
		return
	}
	sheetDefaultIndex := xlsx.GetActiveSheetIndex()
	defaultSheetName := xlsx.GetSheetName(sheetDefaultIndex)

	// 獲取 Sheet1 上所有儲存格
	rows, err := xlsx.GetRows(defaultSheetName)
	begin := 0
	end := 3
	var smaillSizeArr []string
	var smaillPathArr []string
	var failSizeTable []string

	var totalExcelCount int = 0

	for _, row := range rows {
		i, err := strconv.Atoi(row[8])
		if err != nil {
			fmt.Println(err)
			os.Exit(2)
		}
		totalExcelCount += i
	}


	// imageFileListCount := len(imagePicArr)
	// if totalExcelCount != imageFileListCount {
	// 	fmt.Println("EXCEL 檔案I欄位總數:" + strconv.Itoa(totalExcelCount) + "，資料夾內圖片總數:" + strconv.Itoa(imageFileListCount) + "，親愛的請檢查您的圖片總數。")
	// 	fmt.Scanln()
	// 	os.Exit(2)
	// }

	for index, row := range rows {
		step, errturn := strconv.Atoi(row[8])
		if errturn != nil {
			fmt.Println("轉換失敗!!")
		}
		if index == 0 {
			end = step - 1
		} else {
			end = end + step
		}

		leve1MakeDir := DirPath + string(os.PathSeparator) + row[0] + "_" + row[1]
		row[6] = strings.Replace(row[6], "/", "", -1)
		leve2MakeDir := leve1MakeDir + string(os.PathSeparator) + row[3] + "_" + row[6]
		leve3MakeDir := leve2MakeDir + string(os.PathSeparator) + "BIG"
		leve4MakeDir := leve2MakeDir + string(os.PathSeparator) + "SMALL"
		leve15MakeDir := leve1MakeDir + string(os.PathSeparator) + "OUT"
		styleNo := strings.Split(row[2], "-")
		styleNoPath := SpecPath + string(os.PathSeparator) + styleNo[0] + ".jpg"
		if _, err1 := os.Stat(leve1MakeDir); os.IsNotExist(err1) {
			os.Mkdir(leve1MakeDir, 0755)
			fmt.Println("建立資料夾:" + leve1MakeDir)
		}
		if _, err2 := os.Stat(leve2MakeDir); os.IsNotExist(err2) {
			os.Mkdir(leve2MakeDir, 0755)
			fmt.Println("建立資料夾:" + leve2MakeDir)
		}
		if _, err3 := os.Stat(leve3MakeDir); os.IsNotExist(err3) {
			os.Mkdir(leve3MakeDir, 0755)
			fmt.Println("建立資料夾:" + leve3MakeDir)
		}
		if _, err4 := os.Stat(leve4MakeDir); os.IsNotExist(err4) {
			os.Mkdir(leve4MakeDir, 0755)
			fmt.Println("建立資料夾:" + leve4MakeDir)
		}
		if _, err6 := os.Stat(leve15MakeDir); os.IsNotExist(err6) {
			os.Mkdir(leve15MakeDir, 0755)
			fmt.Println("建立資料夾:" + leve15MakeDir)
		}
		err3 := CopyFile(styleNoPath, leve15MakeDir+string(os.PathSeparator)+row[3]+"_"+styleNo[0]+".jpg")
		if err3 != nil {
			fmt.Printf("複製尺寸表 %s 失敗 %q\n", styleNoPath, err3)
			failSizeTable = append(failSizeTable, "複製尺寸表:"+styleNoPath+"失敗\n")
		} else {
			fmt.Printf("複製尺寸表 %s\n", styleNoPath+".jpg")
		}
		count := 1
		for i := begin; i <= end; i++ {
			pathOneImg := imagePath + string(os.PathSeparator) + imagePicArr[i]
			imageNewPath := leve3MakeDir + string(os.PathSeparator) + row[3] + "_0" + strconv.Itoa(count) + ".jpg"
			err := CopyFile(pathOneImg, imageNewPath)
			if err != nil {
				fmt.Printf("複製圖片失敗 %q\n", err)
			} else {
				fmt.Println("複製圖片從:" + pathOneImg + "到 " + imageNewPath)
			}
			imageNewPath2 := leve4MakeDir + string(os.PathSeparator) + row[3] + "_0" + strconv.Itoa(count) + ".jpg"
			err2 := CopyFile(pathOneImg, imageNewPath2)
			if err2 != nil {
				fmt.Printf("複製圖片失敗 %q\n", err2)
			} else {
				fmt.Println("複製圖片從:" + pathOneImg + "到 " + imageNewPath2)
				smaillSizeArr = append(smaillSizeArr, imageNewPath2)
				smaillPathArr = append(smaillPathArr, leve4MakeDir+string(os.PathSeparator)+";")
			}
			imageNewPath4 := leve15MakeDir + string(os.PathSeparator) + row[3] + "_0" + strconv.Itoa(count) + ".jpg"
			err4 := CopyFile(pathOneImg, imageNewPath4)
			if err4 != nil {
				fmt.Printf("複製圖片失敗 %q\n", err4)
			} else {
				fmt.Println("複製圖片從:" + pathOneImg + "到 " + imageNewPath4)
			}
			count++
		}
		begin = begin + step
	}
	var txtString string = ""
	//resize 圖片
	if len(failSizeTable) > 0 {
		for _, errorMsg := range failSizeTable {
			txtString = txtString + errorMsg
		}
		content := []byte(txtString)
		logPath := filepath.Join(execDir, "log.txt")
		if _, err := os.Stat(logPath); os.IsNotExist(err) {
			logPath = "log.txt"
		}
		err := os.WriteFile(logPath, content, 0666)
		if err != nil {
			fmt.Println("ioutil WriteFile error: ", err)
		}
	}

	// fmt.Println(smaillPathArr)

	if len(smaillPathArr) > 0 {
		txtString = ""
		for _, smallPath := range smaillPathArr {
			txtString = txtString + smallPath + "\n"
		}
		content := []byte(txtString)
		txtPath := filepath.Join(execDir, "smallPath.txt")
		if runtime.GOOS == "windows" {
			txtPath = "smallPath.txt"
		}
		err := os.WriteFile(txtPath, content, 0666)
		if err != nil {
			fmt.Println("ioutil WriteFile error: ", err)
		}
	}
	fmt.Println("執行完成")
	fmt.Scanln()
}

func (nf ByNumericalFilename) Len() int      { return len(nf) }
func (nf ByNumericalFilename) Swap(i, j int) { nf[i], nf[j] = nf[j], nf[i] }
func (nf ByNumericalFilename) Less(i, j int) bool {

	pathA := nf[i].Name()
	pathB := nf[j].Name()

	isImgA := strings.HasSuffix(pathA, ".jpg") || strings.HasSuffix(pathA, ".png")
	isImgB := strings.HasSuffix(pathB, ".jpg") || strings.HasSuffix(pathB, ".png")

	if isImgA && isImgB {

		aStart := strings.LastIndex(pathA, "(")
		aEnd := strings.LastIndex(pathA, ")")
		bStart := strings.LastIndex(pathB, "(")
		bEnd := strings.LastIndex(pathB, ")")

		if aStart == -1 || aEnd == -1 || bStart == -1 || bEnd == -1 {
			return pathA < pathB
		}

		a, err1 := strconv.ParseInt(pathA[aStart+1:aEnd], 10, 64)
		b, err2 := strconv.ParseInt(pathB[bStart+1:bEnd], 10, 64)

		if err1 != nil || err2 != nil {
			return pathA < pathB
		}

		return a < b
	}

	return pathA < pathB
}

// 扫描文件夹并排序
func scandir_sort(dir string) []string {
	var files []string

	// 读取目录内容
	filelist, err := os.ReadDir(dir)
	if err != nil {
		log.Fatal(err)
	}

	// 排序文件列表
	sort.Sort(ByNumericalFilename(filelist))

	// 提取文件名
	for _, f := range filelist {
		files = append(files, f.Name())
	}
	return files
}

// 掃描資料夾底下檔案
func scandir(dir string) []string {
	var files []string
	filelist, err := os.ReadDir(dir)
	if err != nil {
		log.Fatal(err)
	}
	for _, f := range filelist {
		files = append(files, f.Name())
	}
	return files
}

// byte 轉 string
func BytesToString(data []byte) string {
	return string(data[:])
}

func moveFile(orgPath string, movePath string) {

	fmt.Println(movePath)
	path := filepath.Dir(movePath)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.Mkdir(path, 0777)
	}
	err := os.Rename(orgPath, movePath)
	if err != nil {
		fmt.Println("移動檔案失敗!!")
	}
}

// CopyFile copies a file from src to dst. If src and dst files exist, and are
// the same, then return success. Otherise, attempt to create a hard link
// between the two files. If that fail, copy the file contents from src to dst.
func CopyFile(src, dst string) error {
	sfi, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !sfi.Mode().IsRegular() {
		return fmt.Errorf("CopyFile: non-regular source file %s (%q)", sfi.Name(), sfi.Mode().String())
	}

	dfi, err := os.Stat(dst)
	if err == nil {
		if !dfi.Mode().IsRegular() {
			return fmt.Errorf("CopyFile: non-regular destination file %s (%q)", dfi.Name(), dfi.Mode().String())
		}
		if os.SameFile(sfi, dfi) {
			return nil
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	// 直接複製內容，避免使用 os.Link
	return copyFileContents(src, dst)
}


// copyFileContents copies the contents of the file named src to the file named
// by dst. The file will be created if it does not already exist. If the
// destination file exists, all it's contents will be replaced by the contents
// of the source file.
func copyFileContents(src, dst string) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return
	}
	defer func() {
		cerr := out.Close()
		if err == nil {
			err = cerr
		}
	}()
	if _, err = io.Copy(out, in); err != nil {
		return
	}
	err = out.Sync()
	return
}
