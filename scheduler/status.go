package scheduler

import (
	"fmt"
	"sync"
)

// 调度器状态的类型
type Status uint8

const (
	// 未初始化的状态
	SCHED_STATUS_UNINITIALIZED Status = 0
	// 正在初始化的状态
	SCHED_STATUS_INITIALIZING Status = 1
	// 已初始化的状态
	SCHED_STATUS_INITIALIZED Status = 2
	// 正在启动的状态
	SCHED_STATUS_STARTING Status = 3
	// 正在启动的状态
	SCHED_STATUS_STARTED Status = 4
	// 正在停止的状态
	SCHED_STATUS_STOPPING Status = 5
	// 已停止的状态
	SCHED_STATUS_STOPPED Status = 6
)

// 用于状态的检查
// 参数currentStatus代表当前的状态
// 参数wantedStatus代表想要的状态
// 检查规则
//		1. 处于正在初始化、正在启动或正在停止状态时，不能从外部改变状态
//		2. 想要的状况只能是正在初始化、正在启动或正在停止状态中的一个
//		3. 处于未初始化状态时，不能变为正在启动或正在停止状态
//		4. 处于已启动状态时，不能变为正在初始化或正在启动状态
//		5. 只要未处于已启动状态就不能变为正在停止状态
func checkStatus(currentStatus Status, wantedStatus Status, lock sync.Locker) (err error) {
	if lock != nil {
		lock.Lock()
		defer lock.Unlock()
	}
	switch currentStatus {
	case SCHED_STATUS_INITIALIZING:
		err = getError("the scheduler is being initialized!")
	case SCHED_STATUS_STARTING:
		err = getError("the scheduler is being started!")
	case SCHED_STATUS_STOPPING:
		err = getError("the scheduler is being stopped!")
	}
	if err != nil {
		return
	}
	if currentStatus == SCHED_STATUS_UNINITIALIZED &&
		(wantedStatus == SCHED_STATUS_STARTING || wantedStatus == SCHED_STATUS_STOPPING) {
		err = getError("the scheduler has not yet been initialized!")
		return
	}
	switch wantedStatus {
	case SCHED_STATUS_INITIALIZING:
		switch currentStatus {
		case SCHED_STATUS_STARTED:
			err = getError("the scheduler has been started!")
		}
	case SCHED_STATUS_STARTING:
		switch currentStatus {
		case SCHED_STATUS_UNINITIALIZED:
			err = getError("the scheduler has not been initialized!")
		case SCHED_STATUS_STARTED:
			err = getError("the scheduler has been started!")
		}
	case SCHED_STATUS_STOPPING:
		if currentStatus != SCHED_STATUS_STARTED {
			err = getError("the scheduler has not been started!")
		}
	default:
		errMsg := fmt.Sprintf("unsupported wanted status for check! (wantedStatus: %d)", wantedStatus)
		err = getError(errMsg)
	}
	return
}

// 用于获取状态的文字描述
func GetStatusDescription(status Status) string {
	switch status {
	case SCHED_STATUS_UNINITIALIZED:
		return "uninitialized"
	case SCHED_STATUS_INITIALIZING:
		return "initializing"
	case SCHED_STATUS_INITIALIZED:
		return "initialized"
	case SCHED_STATUS_STARTING:
		return "starting"
	case SCHED_STATUS_STARTED:
		return "started"
	case SCHED_STATUS_STOPPING:
		return "stopping"
	case SCHED_STATUS_STOPPED:
		return "stopped"
	default:
		return "unknown"
	}
}
