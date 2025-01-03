#!/bin/bash

# 獲取當前腳本所在目錄
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# 切換到腳本所在目錄
cd "$SCRIPT_DIR"

# 執行 PHP 腳本
php ./makeResize.php
