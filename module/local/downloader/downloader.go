package downloader

import (
	"net/http"

	"../../../log"
	"../../../module"
	"../../stub"
)

// logger 代表日志记录器。
var logger = log.DLogger()

// 代表下载器的实现类型
type myDownloader struct {
	// 代表组件基础实例
	stub.ModuleInternal
	// 代表下载用的HTTP客户端
	httpClient http.Client
}

// 用于创建一个下载器实例
func New(mid module.MID, client *http.Client, scoreCalculator module.CalculateScore) (module.Downloader, error) {
	moduleBase, err := stub.NewModuleInternal(mid, scoreCalculator)
	if err != nil {
		return nil, err
	}
	if client == nil {
		return nil, genParameterError("空HTTP客户端")
	}
	return &myDownloader{
		ModuleInternal: moduleBase,
		httpClient:     *client,
	}, nil
}

func (downloader *myDownloader) Download(req *module.Request) (*module.Response, error) {
	downloader.ModuleInternal.IncrHandlingNumber()
	defer downloader.ModuleInternal.DecrHandlingNumber()
	downloader.ModuleInternal.IncrCalledCount()
	if req == nil {
		return nil, genParameterError("空的请求")
	}
	httpReq := req.HTTPReq()
	if httpReq == nil {
		return nil, genParameterError("空的 HTTP 响应")
	}
	downloader.ModuleInternal.IncrAcceptedCount()
	logger.Infof("下载器正在进行请求 (URL: %s, depth: %d)... \n", httpReq.URL, req.Depth())
	httpResp, err := downloader.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	downloader.ModuleInternal.IncrCompletedCount()
	return module.NewResponse(httpResp, req.Depth()), nil
}
