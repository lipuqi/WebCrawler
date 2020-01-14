package reader

import "io"

// 多重读取器的接口
type MultipleReader interface {
	// Reader用于获取一个可关闭读取器的实例
	// 后者会持有该多重读取器的数据
	Reader() io.ReadCloser
}
