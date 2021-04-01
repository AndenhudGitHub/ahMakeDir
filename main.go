package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"io"
	"io/ioutil"
	"log"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Luxurioust/excelize"
)

func main() {
	//尺寸表路徑
	data, err := ioutil.ReadFile("config.json")
	if err != nil {
		fmt.Print(err)
		os.Exit(3)
	}
	type config struct {
		WorkPath      string `json:"WorkPath"`
		SizeTablePath string `json:"SizeTablePath"`
	}
	var obj config
	err = json.Unmarshal(data, &obj)
	if err != nil {
		fmt.Println("error:", err)
		os.Exit(3)
	}
	DirPath := strings.Replace(obj.WorkPath, "\\", "\\\\", -1)
	SpecPath := strings.Replace(obj.SizeTablePath, "\\", "\\\\", -1)
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
	//讀圖片 塞入陣列
	imagePath := DirPath + string(os.PathSeparator) + imageDirArr[0]
	imageFileList := scandir(imagePath)
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
		err := ioutil.WriteFile("log.txt", content, 0666)
		if err != nil {
			fmt.Println("ioutil WriteFile error: ", err)
		}
	}
	if len(smaillPathArr) > 0 {
		txtString = ""
		for _, smallPath := range smaillPathArr {
			txtString = txtString + smallPath + "\n"
		}
		content := []byte(txtString)
		err := ioutil.WriteFile("smallPath.txt", content, 0666)
		if err != nil {
			fmt.Println("ioutil WriteFile error: ", err)
		}
	}
	fmt.Println("執行完成")
	fmt.Scanln()
}

//掃描資料夾底下檔案
func scandir(dir string) []string {
	var files []string
	filelist, err := ioutil.ReadDir(dir)
	if err != nil {
		log.Fatal(err)
	}
	for _, f := range filelist {
		files = append(files, f.Name())
	}
	return files
}

//byte 轉 string
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
func CopyFile(src, dst string) (err error) {
	sfi, err := os.Stat(src)
	if err != nil {
		return
	}
	if !sfi.Mode().IsRegular() {
		// cannot copy non-regular files (e.g., directories,
		// symlinks, devices, etc.)
		return fmt.Errorf("CopyFile: non-regular source file %s (%q)", sfi.Name(), sfi.Mode().String())
	}
	dfi, err := os.Stat(dst)
	if err != nil {
		if !os.IsNotExist(err) {
			return
		}
	} else {
		if !(dfi.Mode().IsRegular()) {
			return fmt.Errorf("CopyFile: non-regular destination file %s (%q)", dfi.Name(), dfi.Mode().String())
		}
		if os.SameFile(sfi, dfi) {
			return
		}
	}
	if err = os.Link(src, dst); err == nil {
		return
	}
	err = copyFileContents(src, dst)
	return
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

func resizeSelf(img image.Image, length int, width int) image.Image {
	//truncate pixel size
	minX := img.Bounds().Min.X
	minY := img.Bounds().Min.Y
	maxX := img.Bounds().Max.X
	maxY := img.Bounds().Max.Y
	for (maxX-minX)%length != 0 {
		maxX--
	}
	for (maxY-minY)%width != 0 {
		maxY--
	}
	scaleX := (maxX - minX) / length
	scaleY := (maxY - minY) / width

	imgRect := image.Rect(0, 0, length, width)
	resImg := image.NewRGBA(imgRect)
	draw.Draw(resImg, resImg.Bounds(), &image.Uniform{C: color.White}, image.ZP, draw.Src)
	for y := 0; y < width; y += 1 {
		for x := 0; x < length; x += 1 {
			averageColor := getAverageColor(img, minX+x*scaleX, minX+(x+1)*scaleX, minY+y*scaleY, minY+(y+1)*scaleY)
			resImg.Set(x, y, averageColor)
		}
	}
	return resImg
}

func getAverageColor(img image.Image, minX int, maxX int, minY int, maxY int) color.Color {
	var averageRed float64
	var averageGreen float64
	var averageBlue float64
	var averageAlpha float64
	scale := 1.0 / float64((maxX-minX)*(maxY-minY))

	for i := minX; i < maxX; i++ {
		for k := minY; k < maxY; k++ {
			r, g, b, a := img.At(i, k).RGBA()
			averageRed += float64(r) * scale
			averageGreen += float64(g) * scale
			averageBlue += float64(b) * scale
			averageAlpha += float64(a) * scale
		}
	}

	averageRed = math.Sqrt(averageRed)
	averageGreen = math.Sqrt(averageGreen)
	averageBlue = math.Sqrt(averageBlue)
	averageAlpha = math.Sqrt(averageAlpha)

	averageColor := color.RGBA{
		R: uint8(averageRed),
		G: uint8(averageGreen),
		B: uint8(averageBlue),
		A: uint8(averageAlpha)}

	return averageColor
}

func imgToBytes(img image.Image) []byte {
	var opt jpeg.Options
	opt.Quality = 80

	buff := bytes.NewBuffer(nil)
	err := jpeg.Encode(buff, img, &opt)
	if err != nil {
		log.Fatal(err)
	}

	return buff.Bytes()
}
func loadImage(filename string) image.Image {
	f, err := os.Open(filename)
	if err != nil {
		log.Fatalf("os.Open failed: %v", err)
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		log.Fatalf("image.Decode failed: %v", err)
	}
	return img
}

func saveImage(filename string, img image.Image) {
	f, err := os.Create(filename)
	if err != nil {
		log.Fatalf("os.Create failed: %v", err)
	}
	defer f.Close()
	err = jpeg.Encode(f, img, nil)
	if err != nil {
		log.Fatalf("png.Encode failed: %v", err)
	}
}
