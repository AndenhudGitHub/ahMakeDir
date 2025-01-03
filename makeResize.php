<?php
require 'JPEG_ICC.php';


$config    = file_get_contents(__DIR__ . DIRECTORY_SEPARATOR  ."config.json");
$configArr = json_decode($config, true);

$smallPath    = file_get_contents(__DIR__ .DIRECTORY_SEPARATOR. "smallPath.txt");
$smallPath    = str_replace("\n", "", $smallPath);
$smallPathArr = array_unique(explode(';', $smallPath));
if (count($smallPathArr) > 0) {
    foreach ($smallPathArr as $filepath) {
        if ($filepath == "") {
            continue;
        }
        $pathFileArr = scandir($filepath);
        foreach ($pathFileArr as $fileJpg) {
            if (strpos($fileJpg, '.jpg') !== false) {
                $fileDir = $filepath . $fileJpg;
                $outDir  = $filepath . "SMALL_" . $fileJpg;
                if (file_exists($fileDir)) {
                    resize($fileDir, $outDir, $configArr['width'], $configArr['height'],$configArr['quality']);
                }
                rename($outDir, $fileDir);
            }
        }
    }
}

/**
 * resize
 */
function resize($filePath, $outPutPath, $outWidth, $outHeight, $quality = 85)
{
    $MyJpeg               = new JPEG_ICC();
    $src                  = imagecreatefromjpeg($filePath);
    list($width, $height) = getimagesize($filePath);
    $tmp                  = imagecreatetruecolor($outWidth, $outHeight);
    $savefilename         = $outPutPath;
    imagecopyresampled($tmp, $src, 0, 0, 0, 0, $outWidth, $outHeight, $width, $height);
    imagejpeg($tmp, $savefilename, $quality);
    if ($MyJpeg->LoadFromJPEG($filePath)) {
        $MyJpeg->SaveToJPEG($savefilename);
    }
    echo "Resize檔案，" . $filePath . "，成功\n";
}

