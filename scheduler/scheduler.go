package scheduler

import (
	"context"
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
func NewScheduler() Scheduler {
	return &myScheduler{}
}

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
