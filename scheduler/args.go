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

// 组件相关的参数容器的类型
type ModuleArgs struct {
	// 下载器列表
	Downloaders []module.Downloader
	// 分析器列表
	Analyzers []module.Analyzer
	// 条目处理管道管道列表
	Pipelines []module.Pipeline
}
