package bm1365Model

import (
	"../../../log"
	"fmt"
	"os"
	"path/filepath"
)

// 日志记录器
var logger = log.DLogger()

// 检查目录路径
func checkDirPath(dirPath string) (absDirPath string, err error) {
	if dirPath == "" {
		err = fmt.Errorf("无效的目录路径: %s", dirPath)
		return
	}
	if filepath.IsAbs(dirPath) {
		absDirPath = dirPath
	} else {
		absDirPath, err = filepath.Abs(dirPath)
		if err != nil {
			return
		}
	}
	var dir *os.File
	dir, err = os.Open(absDirPath)
	if err != nil && !os.IsNotExist(err) {
		return
	}
	if dir == nil {
		err = os.MkdirAll(absDirPath, 0700)
		if err != nil && !os.IsExist(err) {
			return
		}
	} else {
		var fileInfo os.FileInfo
		fileInfo, err = dir.Stat()
		if err != nil {
			return
		}
		if !fileInfo.IsDir() {
			err = fmt.Errorf("不是目录: %s", absDirPath)
			return
		}
	}
	return
}

// 用于记录日志
func Record(level byte, content string) {
	if content == "" {
		return
	}
	switch level {
	case 0:
		logger.Info(content)
	case 1:
		logger.Warnln(content)
	case 2:
		logger.Error(content)
	}
}
