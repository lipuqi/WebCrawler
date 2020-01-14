package scheduler

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
