package buffer

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
