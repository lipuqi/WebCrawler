package scheduler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"../cmap"
	"../log"
	"../module"
	"../toolkit/buffer"
)

// logger 代表日志记录器。
var logger = log.DLogger()

// 调度器的接口类型
type Scheduler interface {
	// Init用于初始化调度器
	// 参数requestArgs代表请求相关的参数
	// 参数dataArgs代表数据相关的参数
	// 参数moduleArgs代表组件相关的参数
	Init(requestArgs RequestArgs, dataArgs DataArgs, moduleArgs ModuleArgs) (err error)
	// Start用于启动调度器并执行爬取流程
	// 参数firstHTTPReq代表首次请求，调度器会以此为起始点开始执行爬取流程
	Start(firstHTTPReq *http.Request) (err error)
	// Stop用于停止调度器的运行
	// 所有处理模块执行的流程都会被中止
	Stop() (err error)
	// 用于获取调度器的状态
	Status() Status
	// ErrorChan用于获得错误通道
	// 调度器以及各个处理模块运行过程中出现的所有错误都会被发送到该通道
	// 若结果为nil，则说明错误通道不可用或调度器已停止
	ErrorChan() <-chan error
	// 用于判断所有处理模块是否都处于空闲状况
	Idle() bool
	// 用于获取摘要实例
	Summary() SchedSummary

	SendReq(req *module.Request) bool
}

// 调度器的实现类型
type myScheduler struct {
	// 爬取的最大深度，首次请求的深度为0
	maxDepth uint32
	// 可以接受的URL的主域名的字典
	acceptedDomainMap cmap.ConcurrentMap
	// 组件注册器
	registrar module.Registrar
	// 请求的缓冲池
	reqBufferPool buffer.Pool
	// 响应的缓冲池
	respBufferPool buffer.Pool
	// 条目的缓冲池
	itemBufferPool buffer.Pool
	// 错误的缓冲池
	errorBufferPool buffer.Pool
	// 已处理的URL的字典
	urlMap cmap.ConcurrentMap
	// 上下文， 用于感知调度器的停止
	ctx context.Context
	// 取消函数， 用于停止调度器
	cancelFunc context.CancelFunc
	// 状态
	status Status
	// 专用于状态的读写锁
	statusLock sync.RWMutex
	// 摘要信息
	summary SchedSummary
}

// 创建调度器实例
func NewScheduler() Scheduler {
	return &myScheduler{}
}

func (sched *myScheduler) Init(requestArgs RequestArgs, dataArgs DataArgs, moduleArgs ModuleArgs) (err error) {
	// 检查状态
	logger.Info("检查状态以进行初始化...")
	var oldStatus Status
	oldStatus, err = sched.checkAndSetStatus(SCHED_STATUS_INITIALIZING)
	if err != nil {
		return
	}
	defer func() {
		sched.statusLock.Lock()
		if err != nil {
			sched.status = oldStatus
		} else {
			sched.status = SCHED_STATUS_INITIALIZED
		}
		sched.statusLock.Unlock()
	}()

	// 检查参数
	logger.Info("检查请求参数...")
	if err = requestArgs.Check(); err != nil {
		return err
	}
	logger.Info("请求参数检查完毕")
	logger.Info("检查基础数据参数...")
	if err = dataArgs.Check(); err != nil {
		return err
	}
	logger.Info("基础数据参数检查完毕")
	logger.Info("检查组件参数...")
	if err = moduleArgs.Check(); err != nil {
		return err
	}
	logger.Info("组件参数检查完毕")

	// 初始化内部字段
	logger.Info("初始化调度器的字段...")
	if sched.registrar == nil {
		sched.registrar = module.NewRegistrar()
	} else {
		sched.registrar.Clear()
	}
	sched.maxDepth = requestArgs.MaxDepth
	logger.Infof("-- 最大爬取深度: %d", sched.maxDepth)
	sched.acceptedDomainMap, _ = cmap.NewConcurrentMap(1, nil)
	for _, domain := range requestArgs.AcceptedDomains {
		pd, _ := getPrimaryDomain(domain)
		sched.acceptedDomainMap.Put(pd, struct{}{})
	}
	logger.Infof("-- 主要请求地址: %v", requestArgs.AcceptedDomains)
	sched.urlMap, _ = cmap.NewConcurrentMap(16, nil)
	logger.Infof("-- URL字典: 长度: %d, 并发量: %d", sched.urlMap.Len(), sched.urlMap.Concurrency())
	sched.initBufferPool(dataArgs)
	sched.resetContext()
	sched.summary = newSchedSummary(requestArgs, dataArgs, moduleArgs, sched)

	// 注册组件
	logger.Info("注册模块...")
	if err = sched.registerModules(moduleArgs); err != nil {
		return err
	}
	logger.Info("调度器已经注册完成")
	return nil
}

func (sched *myScheduler) Start(firstHTTPReq *http.Request) (err error) {
	defer func() {
		if p := recover(); p != nil {
			errMsg := fmt.Sprintf("Fatal scheduler error: %s", p)
			logger.Fatal(errMsg)
			err = genError(errMsg)
		}
	}()
	logger.Info("启动调度器...")

	//检查状态
	logger.Info("检查调度器启动状态...")
	var oldStatus Status
	oldStatus, err = sched.checkAndSetStatus(SCHED_STATUS_STARTING)
	defer func() {
		sched.statusLock.Lock()
		if err != nil {
			sched.status = oldStatus
		} else {
			sched.status = SCHED_STATUS_STARTED
		}
		sched.statusLock.Unlock()
	}()
	if err != nil {
		return
	}

	// 检查参数
	logger.Info("检查首次请求HTTP参数...")
	if firstHTTPReq != nil {
		logger.Info("HTTP主域名请求检查完毕")

		// 获得首次请求的主域名， 并将其添加到可接受的主域名的字典
		logger.Info("获得首次请求的主域名...")
		logger.Infof("-- host: %s", firstHTTPReq.Host)
		var primaryDomain string
		primaryDomain, err = getPrimaryDomain(firstHTTPReq.Host)
		if err != nil {
			return
		}
		logger.Infof("-- 主域名: %s", primaryDomain)
		sched.acceptedDomainMap.Put(primaryDomain, struct{}{})

		// 放入第一请求
		firstReq := module.NewRequest(firstHTTPReq, 0)
		sched.SendReq(firstReq)
	}
	// 开始调度数据和组件
	if err = sched.checkBufferPoolForStart(); err != nil {
		return
	}
	sched.download()
	sched.analyze()
	sched.pick()
	logger.Info("调度器已经启动.")
	return nil
}

func (sched *myScheduler) Stop() (err error) {
	logger.Info("停止调度器...")
	// 检查状态
	logger.Info("检查停止调度器参数...")
	var oldStatus Status
	oldStatus, err = sched.checkAndSetStatus(SCHED_STATUS_STOPPING)
	defer func() {
		sched.statusLock.Lock()
		if err != nil {
			sched.status = oldStatus
		} else {
			sched.status = SCHED_STATUS_STOPPED
		}
		sched.statusLock.Unlock()
	}()
	if err != nil {
		return
	}
	sched.cancelFunc()
	sched.reqBufferPool.Close()
	sched.respBufferPool.Close()
	sched.itemBufferPool.Close()
	sched.errorBufferPool.Close()
	logger.Info("调度器已停止")
	return nil
}

func (sched *myScheduler) Status() Status {
	var status Status
	sched.statusLock.RLock()
	status = sched.status
	sched.statusLock.RUnlock()
	return status
}

func (sched *myScheduler) ErrorChan() <-chan error {
	errBuffer := sched.errorBufferPool
	errCh := make(chan error, errBuffer.BufferCap())
	go func(errBuffer buffer.Pool, errCh chan error) {
		for {
			if sched.canceled() {
				close(errCh)
				break
			}
			datum, err := errBuffer.Get()
			if err != nil {
				logger.Warnln("错误缓冲池已关闭。 中断错误接收")
				close(errCh)
				break
			}
			err, ok := datum.(error)
			if !ok {
				errMsg := fmt.Sprintf("错误类型不正确: %T", datum)
				sendError(errors.New(errMsg), "", sched.errorBufferPool)
				continue
			}
			if sched.canceled() {
				close(errCh)
				break
			}
			errCh <- err
		}
	}(errBuffer, errCh)
	return errCh
}

func (sched *myScheduler) Idle() bool {
	moduleMap := sched.registrar.GetAll()
	for _, module := range moduleMap {
		if module.HandlingNumber() > 0 {
			return false
		}
	}
	if sched.reqBufferPool.Total() > 0 ||
		sched.respBufferPool.Total() > 0 ||
		sched.itemBufferPool.Total() > 0 {
		return false
	}
	return true
}

func (sched *myScheduler) Summary() SchedSummary {
	return sched.summary
}

// 用于状态的检查，并在条件满足时设置状态
func (sched *myScheduler) checkAndSetStatus(wantedStatus Status) (oldStatus Status, err error) {
	sched.statusLock.Lock()
	defer sched.statusLock.Unlock()
	oldStatus = sched.status
	err = checkStatus(oldStatus, wantedStatus, nil)
	if err == nil {
		sched.status = wantedStatus
	}
	return
}

// 用于注册所有给定的组件
func (sched *myScheduler) registerModules(moduleArgs ModuleArgs) error {
	for _, d := range moduleArgs.Downloaders {
		if d == nil {
			continue
		}
		ok, err := sched.registrar.Register(d)
		if err != nil {
			return genErrorByError(err)
		}
		if !ok {
			errMsg := fmt.Sprintf("无法使用MID注册下载器实例 %q!", d.ID())
			return genError(errMsg)
		}
	}
	logger.Infof("所有下载器均已注册 (数量: %d)", len(moduleArgs.Downloaders))

	for _, a := range moduleArgs.Analyzers {
		if a == nil {
			continue
		}
		ok, err := sched.registrar.Register(a)
		if err != nil {
			return genErrorByError(err)
		}
		if !ok {
			errMsg := fmt.Sprintf("无法使用MID注册分析器实例 %q!", a.ID())
			return genError(errMsg)
		}
	}
	logger.Infof("所有分析器均已注册 (数量: %d)", len(moduleArgs.Analyzers))

	for _, p := range moduleArgs.Pipelines {
		if p == nil {
			continue
		}
		ok, err := sched.registrar.Register(p)
		if err != nil {
			return genErrorByError(err)
		}
		if !ok {
			errMsg := fmt.Sprintf("无法使用MID注册条目处理管道实例 %q!", p.ID())
			return genError(errMsg)
		}
	}
	logger.Infof("所有条目处理管道均已注册 (数量: %d)", len(moduleArgs.Pipelines))
	return nil
}

// 从请求缓冲池取出请求并下载
// 然后把得到的响应放入响应缓冲池
func (sched *myScheduler) download() {
	go func() {
		for {
			if sched.canceled() {
				break
			}
			datum, err := sched.reqBufferPool.Get()
			if err != nil {
				logger.Warnln("请求缓冲池已关闭。 中断请求接收")
				break
			}
			req, ok := datum.(*module.Request)
			if !ok {
				errMsg := fmt.Sprintf("错误的请求类型: %T", datum)
				sendError(errors.New(errMsg), "", sched.errorBufferPool)
			}
			sched.downloadOne(req)
		}
	}()
}

// 根据给定的请求执行下载并把响应放入响应缓冲池
func (sched *myScheduler) downloadOne(req *module.Request) {
	if req == nil {
		return
	}
	if sched.canceled() {
		return
	}
	m, err := sched.registrar.Get(module.TYPE_DOWNLOADER)
	if err != nil || m == nil {
		errMsg := fmt.Sprintf("不能获取下载器: %s", err)
		sendError(errors.New(errMsg), "", sched.errorBufferPool)
		sched.SendReq(req)
		return
	}
	downloader, ok := m.(module.Downloader)
	if !ok {
		errMsg := fmt.Sprintf("错误的下载器类型: %T (MID: %s)",
			m, m.ID())
		sendError(errors.New(errMsg), m.ID(), sched.errorBufferPool)
		sched.SendReq(req)
		return
	}
	resp, err := downloader.Download(req)
	if resp != nil {
		sendResp(resp, sched.respBufferPool)
	}
	if err != nil {
		sendError(err, m.ID(), sched.errorBufferPool)
	}
}

// 从响应缓冲池取出响应并解析
// 然后把得到的条目或请求放入相应的缓冲池
func (sched *myScheduler) analyze() {
	go func() {
		for {
			if sched.canceled() {
				break
			}
			datum, err := sched.respBufferPool.Get()
			if err != nil {
				logger.Warnln("响应缓冲池已关闭。 中断响应接收")
				break
			}
			resp, ok := datum.(*module.Response)
			if !ok {
				errMsg := fmt.Sprintf("错误的响应类型: %T", datum)
				sendError(errors.New(errMsg), "", sched.errorBufferPool)
			}
			sched.analyzeOne(resp)
		}
	}()
}

// 会根据给定的响应执行解析并把结果放入相应的缓冲池
func (sched *myScheduler) analyzeOne(resp *module.Response) {
	if resp == nil {
		return
	}
	if sched.canceled() {
		return
	}
	m, err := sched.registrar.Get(module.TYPE_ANALYZER)
	if err != nil || m == nil {
		errMsg := fmt.Sprintf("不能获取分析器: %s", err)
		sendError(errors.New(errMsg), "", sched.errorBufferPool)
		sendResp(resp, sched.respBufferPool)
		return
	}
	analyzer, ok := m.(module.Analyzer)
	if !ok {
		errMsg := fmt.Sprintf("分析器类型不正确: %T (MID: %s)",
			m, m.ID())
		sendError(errors.New(errMsg), m.ID(), sched.errorBufferPool)
		sendResp(resp, sched.respBufferPool)
		return
	}
	dataList, errs := analyzer.Analyze(resp)
	if dataList != nil {
		for _, data := range dataList {
			if data == nil {
				continue
			}
			switch d := data.(type) {
			case *module.Request:
				sched.SendReq(d)
			case module.Item:
				sendItem(d, sched.itemBufferPool)
			default:
				errMsg := fmt.Sprintf("不支持的数据类型 %T! (data: %#v)", d, d)
				sendError(errors.New(errMsg), m.ID(), sched.errorBufferPool)
			}
		}
	}
	if errs != nil {
		for _, err := range errs {
			sendError(err, m.ID(), sched.errorBufferPool)
		}
	}
}

// 从条目缓冲池取出条目并处理
func (sched *myScheduler) pick() {
	go func() {
		for {
			if sched.canceled() {
				break
			}
			datum, err := sched.itemBufferPool.Get()
			if err != nil {
				logger.Warnln("条目缓冲池已关闭。 终止条目接收")
				break
			}
			item, ok := datum.(module.Item)
			if !ok {
				errMsg := fmt.Sprintf("条目类型非法: %T", datum)
				sendError(errors.New(errMsg), "", sched.errorBufferPool)
			}
			sched.pickOne(item)
		}
	}()
}

// 处理给定的条目
func (sched *myScheduler) pickOne(item module.Item) {
	if sched.canceled() {
		return
	}
	m, err := sched.registrar.Get(module.TYPE_PIPELINE)
	if err != nil || m == nil {
		errMsg := fmt.Sprintf("不能获取条目处理管道: %s", err)
		sendError(errors.New(errMsg), "", sched.errorBufferPool)
		sendItem(item, sched.itemBufferPool)
		return
	}
	pipeline, ok := m.(module.Pipeline)
	if !ok {
		errMsg := fmt.Sprintf("条目处理管道类型非法: %T (MID: %s)",
			m, m.ID())
		sendError(errors.New(errMsg), m.ID(), sched.errorBufferPool)
		sendItem(item, sched.itemBufferPool)
		return
	}
	errs := pipeline.Send(item)
	if errs != nil {
		for _, err := range errs {
			sendError(err, m.ID(), sched.errorBufferPool)
		}
	}
}

// 向请求缓冲池发送请求
// 不符合要求的请求会被过滤掉
func (sched *myScheduler) SendReq(req *module.Request) bool {
	if req == nil {
		return false
	}
	if sched.canceled() {
		return false
	}
	httpReq := req.HTTPReq()
	if httpReq == nil {
		logger.Warnln("忽略请求！ HTTP请求无效！")
		return false
	}
	reqURL := httpReq.URL
	if reqURL == nil {
		logger.Warnln("忽略请求！ URL无效！")
		return false
	}
	scheme := strings.ToLower(reqURL.Scheme)
	if scheme != "http" && scheme != "https" {
		logger.Warnf("忽略请求！ 这个URL请求协议为 %q, 不是 %q or %q. (URL: %s)\n",
			scheme, "http", "https", reqURL)
		return false
	}
	if v := sched.urlMap.Get(reqURL.String()); v != nil {
		logger.Warnf("忽略请求！ URL是重复的 (URL: %s)\n", reqURL)
		return false
	}
	pd, _ := getPrimaryDomain(httpReq.Host)
	if sched.acceptedDomainMap.Get(pd) == nil {
		if pd == "bing.net" {
			panic(httpReq.URL)
		}
		logger.Warnf("忽略请求！ 这个 host %q 不在接收的字典里 (URL: %s)\n",
			httpReq.Host, reqURL)
		return false
	}
	if req.Depth() > sched.maxDepth {
		logger.Warnf("忽略请求！ 这个爬取深度 %d 已经大于预设值 %d. (URL: %s)\n",
			req.Depth(), sched.maxDepth, reqURL)
		return false
	}
	go func(req *module.Request) {
		if err := sched.reqBufferPool.Put(req); err != nil {
			logger.Warnln("请求缓冲池已关闭。 忽略请求发送")
		}
	}(req)
	sched.urlMap.Put(reqURL.String(), struct{}{})
	return true
}

// 向响应缓冲池发送响应
func sendResp(resp *module.Response, respBufferPool buffer.Pool) bool {
	if resp == nil || respBufferPool == nil || respBufferPool.Closed() {
		return false
	}
	go func(resp *module.Response) {
		if err := respBufferPool.Put(resp); err != nil {
			logger.Warnln("响应缓冲池已关闭。 忽略响应发送")
		}
	}(resp)
	return true
}

// 向条目缓冲池发送条目
func sendItem(item module.Item, itemBuferPool buffer.Pool) bool {
	if item == nil || itemBuferPool == nil || itemBuferPool.Closed() {
		return false
	}
	go func(item module.Item) {
		if err := itemBuferPool.Put(item); err != nil {
			logger.Warnln("条目缓冲池已关闭。 忽略条目发送")
		}
	}(item)
	return true
}

// 用于按照给定的参数初始化缓冲池
// 如果某个缓冲池可用且未关闭，就先关闭该缓冲池
func (sched *myScheduler) initBufferPool(dataArgs DataArgs) {
	// 初始化请求缓冲池
	if sched.reqBufferPool != nil && !sched.reqBufferPool.Closed() {
		sched.reqBufferPool.Close()
	}
	sched.reqBufferPool, _ = buffer.NewPool(dataArgs.ReqBufferCap, dataArgs.ReqMaxBufferNumber)
	logger.Infof("-- 请求缓冲池: 容量: %d, 最大容量: %d",
		sched.reqBufferPool.BufferCap(), sched.reqBufferPool.MaxBufferNumber())

	// 初始化响应缓冲池
	if sched.respBufferPool != nil && !sched.respBufferPool.Closed() {
		sched.respBufferPool.Close()
	}
	sched.respBufferPool, _ = buffer.NewPool(dataArgs.RespBufferCap, dataArgs.RespMaxBufferNumber)
	logger.Infof("-- 响应缓冲池: 容量: %d, 最大容量: %d",
		sched.respBufferPool.BufferCap(), sched.respBufferPool.MaxBufferNumber())

	// 初始化条目缓冲池
	if sched.itemBufferPool != nil && !sched.itemBufferPool.Closed() {
		sched.itemBufferPool.Close()
	}
	sched.itemBufferPool, _ = buffer.NewPool(dataArgs.ItemBufferCap, dataArgs.ItemMaxBufferNumber)
	logger.Infof("-- 条目缓冲池: 容量: %d, 最大容量: %d",
		sched.itemBufferPool.BufferCap(), sched.itemBufferPool.MaxBufferNumber())

	// 初始化错误缓冲池
	if sched.errorBufferPool != nil && !sched.errorBufferPool.Closed() {
		sched.errorBufferPool.Close()
	}
	sched.errorBufferPool, _ = buffer.NewPool(dataArgs.ErrorBufferCap, dataArgs.ErrorMaxBufferNumber)
	logger.Infof("-- 错误缓冲池: 容量: %d, 最大容量: %d",
		sched.errorBufferPool.BufferCap(), sched.errorBufferPool.MaxBufferNumber())

}

// 检查缓冲池是否已为调度器的启动准备就绪
// 如果某个缓冲池不可用，就直接返回错误值报告此情况
// 如果某个缓冲池已关闭，就按照原先的参数重新初始化它
func (sched *myScheduler) checkBufferPoolForStart() error {
	// 检查请求缓冲池
	if sched.reqBufferPool == nil {
		return genError("空的请求缓冲池")
	}
	if sched.reqBufferPool != nil && sched.reqBufferPool.Closed() {
		sched.reqBufferPool, _ = buffer.NewPool(sched.reqBufferPool.BufferCap(), sched.reqBufferPool.MaxBufferNumber())
	}

	// 检查响应缓冲池
	if sched.respBufferPool == nil {
		return genError("空的响应缓冲池")
	}
	if sched.respBufferPool != nil && sched.respBufferPool.Closed() {
		sched.respBufferPool, _ = buffer.NewPool(sched.respBufferPool.BufferCap(), sched.respBufferPool.MaxBufferNumber())
	}

	// 检查条目缓冲池
	if sched.itemBufferPool == nil {
		return genError("空的条目缓冲池")
	}
	if sched.itemBufferPool != nil && sched.itemBufferPool.Closed() {
		sched.itemBufferPool, _ = buffer.NewPool(sched.itemBufferPool.BufferCap(), sched.itemBufferPool.MaxBufferNumber())
	}

	// 检查错误缓冲池
	if sched.errorBufferPool == nil {
		return genError("空的错误缓冲池")
	}
	if sched.errorBufferPool != nil && sched.errorBufferPool.Closed() {
		sched.errorBufferPool, _ = buffer.NewPool(sched.errorBufferPool.BufferCap(), sched.errorBufferPool.MaxBufferNumber())
	}
	return nil
}

// 用于重置调度器的上下文
func (sched *myScheduler) resetContext() {
	sched.ctx, sched.cancelFunc = context.WithCancel(context.Background())
}

// 用于判断调度器的上下文是否被取消
func (sched *myScheduler) canceled() bool {
	select {
	case <-sched.ctx.Done():
		return true
	default:
		return false
	}
}
