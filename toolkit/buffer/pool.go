package buffer

import (
	"fmt"
	"sync"
	"sync/atomic"

	"../../errors"
)

// 数据缓冲池的接口类型
type Pool interface {
	// 用于获取池中缓冲器的统一容量
	BufferCap() uint32
	// 用于获取池中缓冲器的最大数量
	MaxBufferNumber() uint32
	// 用于获取池中缓冲器的数量
	BufferNumber() uint32
	// 用于获取缓冲池中数据的总数
	Total() uint64
	// Put 用于向缓冲池放入数据
	// 注意！ 本方法是阻塞的
	// 若缓冲池已关闭，则会直接返回非nil的错误值
	Put(datum interface{}) error
	// Get用于从缓冲池获取数据
	// 注意！本方法是阻塞的
	// 若缓冲池已关闭，则会直接返回非nil的错误值
	Get() (datum interface{}, err error)
	// Close用于关闭缓冲池
	// 若缓冲池之前已关闭则返回false，否则返回true
	Close() bool
	// 用于判断缓冲池是否已关闭
	Closed() bool
}

// 数据缓冲池接口的实现类型
type myPool struct {
	// 缓冲器的统一容量
	bufferCap uint32
	// 缓冲器的最大数量
	maxBufferNumber uint32
	// 缓冲器的实际数量
	bufferNumber uint32
	// 池中数据的总数
	total uint64
	// 存放缓冲器的通道
	bufCh chan Buffer
	// 缓冲池的关闭状态：0-未关闭； 1-已关闭
	closed uint32
	// 代表保护内部共享资源的读写锁
	rwLock sync.RWMutex
}

// 用于创建一个数据缓冲池
// 参数bufferCap代表池内缓冲器的统一容量
// 参数maxBufferNumber代表池内最多包含的缓冲器的数量
func NewPool(bufferCap uint32, maxBufferNumber uint32) (Pool, error) {
	if bufferCap == 0 {
		errMsg := fmt.Sprintf("illegal buffer cap for buffer pool: %d", bufferCap)
		return nil, errors.NewIllegalParameterError(errMsg)
	}
	if maxBufferNumber == 0 {
		errMsg := fmt.Sprintf("illegal max buffer number for buffer pool: %d", maxBufferNumber)
		return nil, errors.NewIllegalParameterError(errMsg)
	}
	bufCh := make(chan Buffer, maxBufferNumber)
	buf, _ := NewBuffer(bufferCap)
	bufCh <- buf
	return &myPool{
		bufferCap:       bufferCap,
		maxBufferNumber: maxBufferNumber,
		bufferNumber:    1,
		bufCh:           bufCh,
	}, nil
}

func (pool *myPool) BufferCap() uint32 {
	return pool.bufferCap
}

func (pool *myPool) MaxBufferNumber() uint32 {
	return pool.maxBufferNumber
}

func (pool *myPool) BufferNumber() uint32 {
	return atomic.LoadUint32(&pool.bufferNumber)
}

func (pool *myPool) Total() uint64 {
	return atomic.LoadUint64(&pool.total)
}

func (pool *myPool) Put(datum interface{}) (err error) {
	if pool.Closed() {
		return ErrClosedBufferPool
	}
	var count uint32
	maxCount := pool.BufferNumber() * 5
	var ok bool
	for buf := range pool.bufCh {
		ok, err = pool.putData(buf, datum, &count, maxCount)
		if ok || err != nil {
			break
		}
	}
	return
}

func (pool *myPool) putData(buf Buffer, datum interface{}, count *uint32, maxCount uint32) (ok bool, err error) {
	if pool.Closed() {
		return false, ErrClosedBufferPool
	}
	defer func() {
		pool.rwLock.RLock()
		if pool.Closed() {
			atomic.AddUint32(&pool.bufferNumber, ^uint32(0))
			err = ErrClosedBufferPool
		} else {
			pool.bufCh <- buf
		}
		pool.rwLock.RUnlock()
	}()
	ok, err = buf.Put(datum)
	if ok {
		atomic.AddUint64(&pool.total, 1)
	}
	if err != nil {
		return
	}
	// 若因缓冲器已满而未放入数据就递增计数
	*count++
	// 如果尝试向缓冲器放入数据的失败次数达到阈值
	// 并且池中缓冲器的数量未达到最大值
	// 那么就尝试创建一个新的缓冲器， 先放入数据再把它放入池
	if *count >= maxCount && pool.BufferNumber() < pool.MaxBufferNumber() {
		pool.rwLock.Lock()
		if pool.BufferNumber() < pool.MaxBufferNumber() {
			if pool.Closed() {
				pool.rwLock.Unlock()
				return
			}
			newBuf, _ := NewBuffer(pool.bufferCap)
			newBuf.Put(datum)
			pool.bufCh <- newBuf
			atomic.AddUint32(&pool.bufferNumber, 1)
			atomic.AddUint64(&pool.total, 1)
			ok = true
		}
		pool.rwLock.Unlock()
		*count = 0
	}
	return
}

func (pool *myPool) Get() (datum interface{}, err error) {
	if pool.Closed() {
		return nil, ErrClosedBufferPool
	}
	var count uint32
	maxCount := pool.BufferNumber() * 10
	for buf := range pool.bufCh {
		datum, err = pool.getData(buf, &count, maxCount)
		if datum != nil || err != nil {
			break
		}
	}
	return
}

func (pool *myPool) getData(buf Buffer, count *uint32, maxCount uint32) (datum interface{}, err error) {
	if pool.Closed() {
		return nil, ErrClosedBufferPool
	}
	defer func() {
		// 如果尝试从缓冲器获取数据失败次数达到阈值
		// 同时当前缓冲器已空且池中缓冲器的数量大于1
		// 那么就直接关闭当前缓冲器，并不归还给池
		if *count >= maxCount && buf.Len() == 0 && pool.BufferNumber() > 1 {
			buf.Close()
			atomic.AddUint32(&pool.bufferNumber, ^uint32(0))
			*count = 0
			return
		}
		pool.rwLock.RLock()
		if pool.Closed() {
			atomic.AddUint32(&pool.bufferNumber, ^uint32(0))
			err = ErrClosedBufferPool
		} else {
			pool.bufCh <- buf
		}
		pool.rwLock.RUnlock()
	}()
	datum, err = buf.Get()
	if datum != nil {
		atomic.AddUint64(&pool.total, ^uint64(0))
		return
	}
	if err != nil {
		return
	}
	// 若因缓冲器已空未取出数据就递增计数
	*count++
	return
}

func (pool *myPool) Close() bool {
	if !atomic.CompareAndSwapUint32(&pool.closed, 0, 1) {
		return false
	}
	pool.rwLock.Lock()
	defer pool.rwLock.Unlock()
	close(pool.bufCh)
	for buf := range pool.bufCh {
		buf.Close()
	}
	return true
}

func (pool *myPool) Closed() bool {
	if atomic.LoadUint32(&pool.closed) == 1 {
		return true
	}
	return false
}
