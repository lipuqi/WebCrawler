package stub

import (
	".."
	"../../errors"
	"fmt"
	"sync/atomic"
)

// 组件内部基础接口的实现类型
type myModule struct {
	// 组件ID
	mid module.MID
	// 组件的网络地址
	addr string
	// 组件评分
	score uint64
	// 评分计算器
	scoreCalculator module.CalculateScore
	// 调用计数
	calledCount uint64
	// 接受计数
	acceptedCount uint64
	// 成功完成计数
	completedCount uint64
	// 实时处理数
	handlingNumber uint64
}

// 用于创建一个组件内部基础类型的实例
func NewModuleInternal(mid module.MID, scoreCalculator module.CalculateScore) (ModuleInternal, error) {
	parts, err := module.SplitMID(mid)
	if err != nil {
		return nil, errors.NewIllegalParameterError(fmt.Sprintf("illegal ID %q: %s", mid, err))
	}
	return &myModule{
		mid:             mid,
		addr:            parts[2],
		scoreCalculator: scoreCalculator,
	}, nil
}

func (m *myModule) ID() module.MID {
	return m.mid
}

func (m *myModule) Addr() string {
	return m.addr
}

func (m *myModule) Score() uint64 {
	return atomic.LoadUint64(&m.score)
}

func (m *myModule) SetScore(score uint64) {
	atomic.StoreUint64(&m.score, score)
}

func (m *myModule) ScoreCalculator() module.CalculateScore {
	return m.scoreCalculator
}

func (m *myModule) CalledCount() uint64 {
	return atomic.LoadUint64(&m.calledCount)
}

func (m *myModule) AcceptedCount() uint64 {
	return atomic.LoadUint64(&m.acceptedCount)
}

func (m *myModule) CompletedCount() uint64 {
	count := atomic.LoadUint64(&m.completedCount)
	return count
}

func (m *myModule) HandlingNumber() uint64 {
	return atomic.LoadUint64(&m.handlingNumber)
}

func (m *myModule) Counts() module.Counts {
	return module.Counts{
		CalledCount:    atomic.LoadUint64(&m.calledCount),
		AcceptedCount:  atomic.LoadUint64(&m.acceptedCount),
		CompletedCount: atomic.LoadUint64(&m.completedCount),
		HandlingNumber: atomic.LoadUint64(&m.handlingNumber),
	}
}

func (m *myModule) Summary() module.SummaryStruct {
	counts := m.Counts()
	return module.SummaryStruct{
		ID:        m.ID(),
		Called:    counts.CalledCount,
		Accepted:  counts.AcceptedCount,
		Completed: counts.CompletedCount,
		Handling:  counts.HandlingNumber,
		Extra:     nil,
	}
}

func (m *myModule) IncrCalledCount() {
	atomic.AddUint64(&m.calledCount, 1)
}

func (m *myModule) IncrAcceptedCount() {
	atomic.AddUint64(&m.acceptedCount, 1)
}

func (m *myModule) IncrCompletedCount() {
	atomic.AddUint64(&m.completedCount, 1)
}

func (m *myModule) IncrHandlingNumber() {
	atomic.AddUint64(&m.handlingNumber, 1)
}

func (m *myModule) DecrHandlingNumber() {
	atomic.AddUint64(&m.handlingNumber, ^uint64(0))
}

func (m *myModule) Clear() {
	atomic.StoreUint64(&m.calledCount, 0)
	atomic.StoreUint64(&m.acceptedCount, 0)
	atomic.StoreUint64(&m.completedCount, 0)
	atomic.StoreUint64(&m.handlingNumber, 0)
}
