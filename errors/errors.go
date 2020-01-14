package errors

import (
	"bytes"
	"fmt"
	"strings"
)

// ErrorType 代表错误的类型
type ErrorType string

// 错误类型的常量
const (
	// 下载器错误
	ERROR_TYPE_DOWNLOADER ErrorType = "downloader error"
	// 分析器错误
	ERROR_TYPE_ANALYER ErrorType = "analyzer error"
	// 条目处理管道错误
	ERROR_TYPE_PIPELINE ErrorType = "pipeline error"
	// 调度器错误
	ERROR_TYPE_SCHEDULER ErrorType = "scheduler error"
)

// 爬虫错误的接口类型
type CrawlerError interface {
	// 用于获取错误的类型
	Type() ErrorType
	// 用于获取错误提示信息
	Error() string
}

// 爬虫错误的实现类型
type myCrawlerError struct {
	// 错误的类型
	errType ErrorType
	// 错误的提示信息
	errMsg string
	// 完整的错误提示信息
	fullErrMsg string
}

// 用于创建一个新的爬虫错误值
func NewCrawlerError(errType ErrorType, errMsg string) CrawlerError {
	return &myCrawlerError{
		errType: errType,
		errMsg:  strings.TrimSpace(errMsg),
	}
}

// 用于根据给定的错误值创建一个新的爬虫错误值
func NewCrawlerErrorBy(errType ErrorType, err error) CrawlerError {
	return NewCrawlerError(errType, err.Error())
}

func (ce *myCrawlerError) Type() ErrorType {
	return ce.errType
}

func (ce *myCrawlerError) Error() string {
	if ce.fullErrMsg == "" {
		ce.genFullErrMsg()
	}
	return ce.fullErrMsg
}

// 用于生成错误提示信息，并给相应的字段赋值
func (ce *myCrawlerError) genFullErrMsg() {
	var buffer bytes.Buffer
	buffer.WriteString("crawler error: ")
	if ce.errType != "" {
		buffer.WriteString(string(ce.errType))
		buffer.WriteString(" : ")
	}
	buffer.WriteString(ce.errMsg)
	ce.fullErrMsg = fmt.Sprintf("%s", buffer.String())
}

// 代表非法的参数的错误类型
type IllegalParameterError struct {
	msg string
}

// 会创建一个IllegalParameterError类型的实例
func NewIllegalParameterError(errMsg string) IllegalParameterError {
	return IllegalParameterError{msg: fmt.Sprintf("illegal parameter: %s", strings.TrimSpace(errMsg))}
}

func (ipe IllegalParameterError) Error() string {
	return ipe.msg
}
