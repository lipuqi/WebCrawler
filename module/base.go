package module

import "net/http"

// 用于汇集组件内部计数的类型
type Counts struct {
	// 调用计数
	CalledCount uint64
	// 接收计数
	AcceptedCount uint64
	// 成功完成的计数
	CompletedCount uint64
	// 实时处理数
	HandlingNumber uint64
}

// 组件摘要结构的类型
type SummaryStruct struct {
	ID        MID         `json:"id"`
	Called    uint64      `json:"called"`
	Accepted  uint64      `json:"accepted"`
	Completed uint64      `json:"completed"`
	Handling  uint64      `json:"handling"`
	Extra     interface{} `json:"extra, omitempty"`
}

// Module代表组件的基础接口类型
// 该接口实现类型必须是并发安全的
type Module interface {
	// 用于获取当前组件ID
	ID() MID
	// 用于获取当前组件的网络地址的字符串形式
	Addr() string
	// 用于获取当前组件的评分
	Score() uint64
	// 用于设置当前组件的评分
	SetScore(score uint64)
	// 用于获取评分计算器
	ScoreCalculator() CalculateScore
	// 用于获取当前组件被调用的计数
	CalledCount() uint64
	// 用于获取当前组件接受的调用的计数
	// 组件一般会由于超负荷或参数有误而拒绝调用
	AcceptedCount() uint64
	// 用于获取当前组件已成功完成的调用的计数
	CompletedCount() uint64
	// 用于获取当前组件正在处理的调用的数量
	HandlingNumber() uint64
	// 用于一次性获取所有计数
	Counts() Counts
	// 用于获取组件摘要
	Summary() SummaryStruct
}

// Downloader代表下载器的接口类型
// 该接口的实现类型必须是并发安全的
type Downloader interface {
	Module
	// 根据请求获取内容并返回响应
	Download(req *Request) (*Response, error)
}

// Analyzer 代表分析器的接口类型
// 该接口的实现类型必须是并发安全的
type Analyzer interface {
	Module
	// 用于返回当前分析器使用的响应解析函数的列表
	RespParsers() []ParseResponse
	// 根据规则分析响应并返回请求的条目
	// 响应需要分别经过若干响应解析函数的处理，然后合并结果
	Analyze(resp *Response) ([]Data, []error)
}

// 用于解析http响应的函数的类型
type ParseResponse func(httpResp *http.Response, respDepth uint32) ([]Data, []error)

// Pipeline 代表条目处理管道的接口类型
// 该接口的实现类型必须是并发安全的
type Pipeline interface {
	Module
	// 用于返回当前条目处理管道使用的条目处理函数的列表
	ItemProcessors() []ProcessItem
	// send 会向条目处理管道发送条目
	// 条目需要依次经过若干条目处理函数的处理
	Send(item Item) []error
	// FailFast 方法会返回一个布尔值，该值表示当前条目处理管道是否是快速失败的
	// 这里的快速失败是指：只要在处理某个条目时在某个步骤上出错
	// 那么条目处理管道就会忽略掉后续的所有处理步骤并报告错误
	FailFast() bool
	// 设置是否快速失败
	SetFailFast(failFast bool)
}

// 用于处理条目的函数类型
type ProcessItem func(item Item) (result Item, err error)
