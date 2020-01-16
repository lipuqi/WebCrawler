package analyzer

import "../../../errors"

// genError 用于生成爬虫错误值
func genError(errMsg string) error {
	return errors.NewCrawlerError(errors.ERROR_TYPE_ANALYER, errMsg)
}

// genParameterError 用于生成爬虫参数错误值
func genParameterError(errMsg string) error {
	return errors.NewCrawlerErrorBy(errors.ERROR_TYPE_ANALYER, errors.NewIllegalParameterError(errMsg))
}
