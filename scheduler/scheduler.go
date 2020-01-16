package scheduler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
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
/*func NewScheduler() Scheduler {
	return &myScheduler{}
}*/

func (sched *myScheduler) Init(requestArgs RequestArgs, dataArgs DataArgs, moduleArgs ModuleArgs) (err error) {
	// 检查状态
	logger.Info("Check status for initialization...")
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
	logger.Info("Check request arguments...")
	if err = requestArgs.Check(); err != nil {
		return err
	}
	logger.Info("Check data arguments...")
	if err = dataArgs.Check(); err != nil {
		return err
	}
	logger.Info("Data arguments are valid.")
	logger.Info("Check module arguments...")
	if err = moduleArgs.Check(); err != nil {
		return err
	}
	logger.Info("Module arguments are valid.")

	// 初始化内部字段
	logger.Info("Initialize scheduler's fields...")
	if sched.registrar == nil {
		sched.registrar = module.NewRegistrar()
	} else {
		sched.registrar.Clear()
	}
	sched.maxDepth = requestArgs.MaxDepth
	logger.Infof("-- Max depth: %d", sched.maxDepth)
	sched.acceptedDomainMap, _ = cmap.NewConcurrentMap(1, nil)
	for _, domain := range requestArgs.AcceptedDomains {
		sched.acceptedDomainMap.Put(domain, struct{}{})
	}
	logger.Infof("-- Accepted primary domains: %v", requestArgs.AcceptedDomains)
	sched.urlMap, _ = cmap.NewConcurrentMap(16, nil)
	logger.Infof("-- URL map: length: %d, concurrency: %d", sched.urlMap.Len(), sched.urlMap.Concurrency())
	sched.initBufferPool(dataArgs)
	sched.resetContext()
	sched.summary = newSchedSummary(requestArgs, dataArgs, moduleArgs, sched)

	// 注册组件
	logger.Info("Register modules...")
	if err = sched.registerModules(moduleArgs); err != nil {
		return err
	}
	logger.Info("Scheduler has been initialized.")
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
	logger.Info("Start scheduler...")

	//检查状态
	logger.Info("Check status for start...")
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
	logger.Info("Check first HTTP request...")
	if firstHTTPReq == nil {
		err = genParameterError("nil first HTTP request")
		return
	}
	logger.Info("The first HTTP request is valid.")

	// 获得首次请求的主域名， 并将其添加到可接受的主域名的字典
	logger.Info("Get the primary domain...")
	logger.Info("-- Host: %s", firstHTTPReq.Host)
	var primaryDomain string
	primaryDomain, err = getPrimaryDomain(firstHTTPReq.Host)
	if err != nil {
		return
	}
	logger.Infof("-- Primary domain: %s", primaryDomain)
	sched.acceptedDomainMap.Put(primaryDomain, struct{}{})

	// 开始调度数据和组件
	if err = sched.checkBufferPoolForStart(); err != nil {
		return
	}

}

func (sched *myScheduler) Status() Status {
	var status Status
	sched.statusLock.RLock()
	status = sched.status
	sched.statusLock.RUnlock()
	return status
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
			errMsg := fmt.Sprintf("Couldn't register downloader instance with MID %q!", d.ID())
			return genError(errMsg)
		}
	}
	logger.Infof("All downloads have been registered. (number: %d)", len(moduleArgs.Downloaders))

	for _, a := range moduleArgs.Analyzers {
		if a == nil {
			continue
		}
		ok, err := sched.registrar.Register(a)
		if err != nil {
			return genErrorByError(err)
		}
		if !ok {
			errMsg := fmt.Sprintf("Couldn't register analyzer instance with MID %q!", a.ID())
			return genError(errMsg)
		}
	}
	logger.Infof("All analyzers have been registered. (number: %d)", len(moduleArgs.Analyzers))

	for _, p := range moduleArgs.Pipelines {
		if p == nil {
			continue
		}
		ok, err := sched.registrar.Register(p)
		if err != nil {
			return genErrorByError(err)
		}
		if !ok {
			errMsg := fmt.Sprintf("Couldn't register pipeline instance with MID %q!", p.ID())
			return genError(errMsg)
		}
	}
	logger.Infof("All pipelines have been registered. (number: %d)", len(moduleArgs.Pipelines))

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
			datum, err := sched.respBufferPool.Get()
			if err != nil {
				logger.Warnln("The request buffer pool was closed. Break request reception.")
				break
			}
			req, ok := datum.(*module.Request)
			if !ok {
				errMsg := fmt.Sprintf("incorrect request type: %T", datum)
				sendError(errors.New(errMsg), "", sched.errorBufferPool)
			}

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
		errMsg := fmt.Sprintf("couldn't get a downloader: %s", err)
		sendError(errors.New(errMsg), "", sched.errorBufferPool)
		sched
	}
}

// 向请求缓冲池发送请求
// 不符合要求的请求会被过滤掉
func (sched *myScheduler) sendReq(req *module.Request) bool {
	if req == nil {
		return false
	}
	if sched.canceled() {
		return false
	}
	httpReq := req.HTTPReq()
	if httpReq == nil {

	}
}

// 用于按照给定的参数初始化缓冲池
// 如果某个缓冲池可用且未关闭，就先关闭该缓冲池
func (sched *myScheduler) initBufferPool(dataArgs DataArgs) {
	// 初始化请求缓冲池
	if sched.reqBufferPool != nil && !sched.reqBufferPool.Closed() {
		sched.reqBufferPool.Close()
	}
	sched.reqBufferPool, _ = buffer.NewPool(dataArgs.ReqBufferCap, dataArgs.ReqMaxBufferNumber)
	logger.Infof("-- Request buffer pool: bufferCap: %d, maxBufferNumber: %d",
		sched.reqBufferPool.BufferCap(), sched.reqBufferPool.MaxBufferNumber())

	// 初始化响应缓冲池
	if sched.respBufferPool != nil && !sched.respBufferPool.Closed() {
		sched.respBufferPool.Close()
	}
	sched.respBufferPool, _ = buffer.NewPool(dataArgs.RespBufferCap, dataArgs.RespMaxBufferNumber)
	logger.Infof("-- Response buffer pool: bufferCap: %d, maxBufferNumber: %d",
		sched.respBufferPool.BufferCap(), sched.respBufferPool.MaxBufferNumber())

	// 初始化条目缓冲池
	if sched.itemBufferPool != nil && !sched.itemBufferPool.Closed() {
		sched.itemBufferPool.Close()
	}
	sched.itemBufferPool, _ = buffer.NewPool(dataArgs.ItemBufferCap, dataArgs.ItemMaxBufferNumber)
	logger.Infof("-- Item buffer pool: bufferCap: %d, maxBufferNumber: %d",
		sched.itemBufferPool.BufferCap(), sched.itemBufferPool.MaxBufferNumber())

	// 初始化错误缓冲池
	if sched.errorBufferPool != nil && !sched.errorBufferPool.Closed() {
		sched.errorBufferPool.Close()
	}
	sched.errorBufferPool, _ = buffer.NewPool(dataArgs.ErrorBufferCap, dataArgs.ErrorMaxBufferNumber)
	logger.Infof("-- Error buffer pool: bufferCap: %d, maxBufferNumber: %d",
		sched.errorBufferPool.BufferCap(), sched.errorBufferPool.MaxBufferNumber())

}

// 检查缓冲池是否已为调度器的启动准备就绪
// 如果某个缓冲池不可用，就直接返回错误值报告此情况
// 如果某个缓冲池已关闭，就按照原先的参数重新初始化它
func (sched *myScheduler) checkBufferPoolForStart() error {
	// 检查请求缓冲池
	if sched.reqBufferPool == nil {
		return genError("nil request buffer pool")
	}
	if sched.reqBufferPool != nil && sched.reqBufferPool.Closed() {
		sched.reqBufferPool, _ = buffer.NewPool(sched.reqBufferPool.BufferCap(), sched.reqBufferPool.MaxBufferNumber())
	}

	// 检查响应缓冲池
	if sched.respBufferPool == nil {
		return genError("nil response buffer pool")
	}
	if sched.respBufferPool != nil && sched.respBufferPool.Closed() {
		sched.respBufferPool, _ = buffer.NewPool(sched.respBufferPool.BufferCap(), sched.respBufferPool.MaxBufferNumber())
	}

	// 检查条目缓冲池
	if sched.itemBufferPool == nil {
		return genError("nil item buffer pool")
	}
	if sched.itemBufferPool != nil && sched.itemBufferPool.Closed() {
		sched.itemBufferPool, _ = buffer.NewPool(sched.itemBufferPool.BufferCap(), sched.itemBufferPool.MaxBufferNumber())
	}

	// 检查错误缓冲池
	if sched.errorBufferPool == nil {
		return genError("nil error buffer pool")
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
