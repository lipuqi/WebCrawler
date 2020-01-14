package module

// 组件注册器的接口
type Registrar interface {
	// 用于注册组件的实例
	Register(module Module) (bool, error)
	// 用于注销组件的实例
	Unregister(mid MID) (bool, error)
	// 用于获取一个指定类型的组件实例
	// 该函数基于负载均衡策略返回实例
	Get(moduleType Type) (Module, error)
	// 用于获取指定类型的所有组件实例
	GetAllByType(moduleType Type) (map[MID]Module, error)
	// 用于获取所有组件实例
	GetAll() map[MID]Module
	// 清除所有的组件注册记录
	Clear()
}
