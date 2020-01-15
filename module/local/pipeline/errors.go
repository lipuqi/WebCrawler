package pipeline

import "../../../errors"

// genError 用于生成爬虫错误值
func getError(errMsg string) error {
	return errors.NewCrawlerError(errors.ERROR_TYPE_PIPELINE, errMsg)
}

// genParameterError 用于生成爬虫参数错误值
func genParameterError(errMsg string) error {
	return errors.NewCrawlerErrorBy(errors.ERROR_TYPE_PIPELINE, errors.NewIllegalParameterError(errMsg))
}
