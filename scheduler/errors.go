package scheduler

import (
	"../errors"
	"../module"
	"../toolkit/buffer"
)

// 用于生成爬虫错误值
func genError(errMsg string) error {
	return errors.NewCrawlerError(errors.ERROR_TYPE_SCHEDULER, errMsg)
}

// 用于基于给定的错误值生成爬虫错误值
func genErrorByError(err error) error {
	return errors.NewCrawlerError(errors.ERROR_TYPE_PIPELINE, err.Error())
}

// 用于生成爬虫错误值
func genParameterError(errMsg string) error {
	return errors.NewCrawlerErrorBy(errors.ERROR_TYPE_SCHEDULER, errors.NewIllegalParameterError(errMsg))
}

// 用于向错误缓冲池发送错误值
func sendError(err error, mid module.MID, errorBufferPool buffer.Pool) bool {
	if err == nil || errorBufferPool == nil || errorBufferPool.Closed() {
		return false
	}
	var crawlerError errors.CrawlerError
	var ok bool
	crawlerError, ok = err.(errors.CrawlerError)
	if !ok {
		var moduleType module.Type
		var errorType errors.ErrorType
		ok, moduleType = module.GetType(mid)
		if !ok {
			errorType = errors.ERROR_TYPE_SCHEDULER
		} else {
			switch moduleType {
			case module.TYPE_DOWNLOADER:
				errorType = errors.ERROR_TYPE_DOWNLOADER
			case module.TYPE_ANALYZER:
				errorType = errors.ERROR_TYPE_ANALYER
			case module.TYPE_PIPELINE:
				errorType = errors.ERROR_TYPE_PIPELINE
			}
		}
		crawlerError = errors.NewCrawlerError(errorType, err.Error())
	}
	if errorBufferPool.Closed() {
		return false
	}
	go func(crawlerError errors.CrawlerError) {
		if err := errorBufferPool.Put(crawlerError); err != nil {
			logger.Warnln("错误缓冲池已关闭！忽略错误发送")
		}
	}(crawlerError)
	return true
}
