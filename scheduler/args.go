package scheduler

import "../module"

// 参数容器的接口类型
type Args interface {
	// Check 用于自检参数的有效性
	// 若结果值为nil，则说明未发现问题，否则就意味着自检未通过
	Check() error
}

// 请求相关的参数容器的类型
type RequestArgs struct {
	// AcceptedDomains 代表可以接受的URL的主域名的列表
	// URL主域名不在列表中的请求都会被忽略
	AcceptedDomains []string `json:"accepted_primary_domains"`
	// MaxDepth 代表需要爬取的最大深度
	// 实际深度大于此值的请求都会被忽略
	MaxDepth uint32 `json:"max_depth"`
}

func (args *RequestArgs) Check() error {
	if args.AcceptedDomains == nil {
		return genError("nil accepted primary domain list")
	}
	return nil
}

func (args *RequestArgs) Same(another *RequestArgs) bool {
	if another == nil {
		return false
	}
	if another.MaxDepth != args.MaxDepth {
		return false
	}
	anotherDomains := another.AcceptedDomains
	anotherDomainsLen := len(anotherDomains)
	if anotherDomainsLen != len(args.AcceptedDomains) {
		return false
	}
	if anotherDomainsLen > 0 {
		for i, domain := range anotherDomains {
			if domain != args.AcceptedDomains[i] {
				return false
			}
		}
	}
	return true
}

// 数据相关的参数容器的类型
type DataArgs struct {
	// 请求缓冲器的容量
	ReqBufferCap uint32 `json:"req_buffer_cap"`
	// 请求缓冲器的最大数量
	ReqMaxBufferNumber uint32 `json:"req_max_buffer_number"`
	// 响应缓冲器的容量
	RespBufferCap uint32 `json:"resp_buffer_cap"`
	// 响应缓冲器的最大数量
	RespMaxBufferNumber uint32 `json:"resp_max_buffer_number"`
	// 条目缓冲器的容量
	ItemBufferCap uint32 `json:"item_buffer_cap"`
	// 条目缓冲器的最大数量
	ItemMaxBufferNumber uint32 `json:"item_max_buffer_number"`
	// 错误缓冲器的容量
	ErrorBufferCap uint32 `json:"error_buffer_cap"`
	// 错误缓冲器的最大数量
	ErrorMaxBufferNumber uint32 `json:"error_max_buffer_number"`
}

func (args *DataArgs) Check() error {
	if args.ReqBufferCap == 0 {
		return genError("zero request buffer capacity")
	}
	if args.ReqMaxBufferNumber == 0 {
		return genError("zero max request buffer number")
	}
	if args.RespBufferCap == 0 {
		return genError("zero response buffer capacity")
	}
	if args.RespMaxBufferNumber == 0 {
		return genError("zero max response buffer number")
	}
	if args.ItemBufferCap == 0 {
		return genError("zero item buffer capacity")
	}
	if args.ItemMaxBufferNumber == 0 {
		return genError("zero max item buffer number")
	}
	if args.ErrorBufferCap == 0 {
		return genError("zero error buffer capacity")
	}
	if args.ErrorMaxBufferNumber == 0 {
		return genError("zero max error buffer number")
	}
	return nil
}

// 代表组件相关的参数容器的摘要类型
type ModuleArgsSummary struct {
	DownloaderListSize int `json:"downloader_list_size"`
	AnalyzerListSize   int `json:"analyzer_list_size"`
	PipelineListSize   int `json:"pipeline_list_size"`
}

// 组件相关的参数容器的类型
type ModuleArgs struct {
	// 下载器列表
	Downloaders []module.Downloader
	// 分析器列表
	Analyzers []module.Analyzer
	// 条目处理管道管道列表
	Pipelines []module.Pipeline
}

// 用于当前参数容器的有效性
func (args *ModuleArgs) Check() error {
	if len(args.Downloaders) == 0 {
		return genError("empty downloader list")
	}
	if len(args.Analyzers) == 0 {
		return genError("empty analyzer list")
	}
	if len(args.Pipelines) == 0 {
		return genError("empty pipeline list")
	}
	return nil
}

func (args *ModuleArgs) Summary() ModuleArgsSummary {
	return ModuleArgsSummary{
		DownloaderListSize: len(args.Downloaders),
		AnalyzerListSize:   len(args.Analyzers),
		PipelineListSize:   len(args.Pipelines),
	}
}
