package internal

import (
	"../../../module"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// 用于生成条目处理器
func genItemProcessors(dirPath string) []module.ProcessItem {
	savePicture := func(itme module.Item) (result module.Item, err error) {
		if itme == nil {
			return nil, errors.New("无效的条目")
		}
		// 检查和准备数据
		var absDirPath string
		if absDirPath, err = checkDirPath(dirPath); err != nil {
			return
		}
		v := itme["reader"]
		reader, ok := v.(io.Reader)
		if !ok {
			return nil, fmt.Errorf("条目处理管道 reader 类型错误: %T", v)
		}
		readCloser, ok := reader.(io.ReadCloser)
		if ok {
			defer readCloser.Close()
		}
		v = itme["name"]
		name, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("条目处理管道 name 类型错误: %T", v)
		}
		// 创建图片文件
		fileName := name
		filePath := filepath.Join(absDirPath, fileName)
		file, err := os.Create(filePath)
		if err != nil {
			return nil, fmt.Errorf("创建文件失败: %s (path: %s)",
				err, filePath)
		}
		defer file.Close()
		// 写图片文件
		_, err = io.Copy(file, reader)
		if err != nil {
			return nil, err
		}
		// 生成新条目
		result = make(map[string]interface{})
		for k, v := range itme {
			result[k] = v
		}
		result["file_path"] = filePath
		fileInfo, err := file.Stat()
		if err != nil {
			return nil, err
		}
		result["file_size"] = fileInfo.Size()
		return result, nil
	}
	recordPicture := func(item module.Item) (result module.Item, err error) {
		v := item["file_path"]
		path, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("条目处理管道 file_path 类型错误: %T", v)
		}
		v = item["file_size"]
		size, ok := v.(int64)
		if !ok {
			return nil, fmt.Errorf("条目处理管道 file_size 类型错误: %T", v)
		}
		logger.Infof("保存文件: %s, 文件大小: %d byte(s).", path, size)
		return nil, nil
	}
	return []module.ProcessItem{savePicture, recordPicture}
}
