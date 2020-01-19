package module

import (
	"fmt"
	"sync"

	"../errors"
)

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

// 代表组件注册器的实现类型
type myRegistrar struct {
	// moduleTypeMap 代表组件类型与对应组件实例的映射
	moduleTypeMap map[Type]map[MID]Module
	// rwlock 代表组件注册专用读写锁
	rwLock sync.RWMutex
}

// 用于创建一个组件注册器的实例
func NewRegistrar() Registrar {
	return &myRegistrar{
		moduleTypeMap: map[Type]map[MID]Module{},
	}
}

func (registrar *myRegistrar) Register(module Module) (bool, error) {
	if module == nil {
		return false, errors.NewIllegalParameterError("空组件实例")
	}
	mid := module.ID()
	parts, err := SplitMID(mid)
	if err != nil {
		return false, err
	}
	moduleType := legalLetterTypeMap[parts[0]]
	if !CheckType(moduleType, module) {
		errMsg := fmt.Sprintf("组件实例类型异常: %s", moduleType)
		return false, errors.NewIllegalParameterError(errMsg)
	}
	registrar.rwLock.Lock()
	defer registrar.rwLock.Unlock()
	modules := registrar.moduleTypeMap[moduleType]
	if modules == nil {
		modules = map[MID]Module{}
	}
	if _, ok := modules[mid]; ok {
		return false, nil
	}
	modules[mid] = module
	registrar.moduleTypeMap[moduleType] = modules
	return true, nil
}

func (registrar *myRegistrar) Unregister(mid MID) (bool, error) {
	parts, err := SplitMID(mid)
	if err != nil {
		return false, err
	}
	moduleType := legalLetterTypeMap[parts[0]]
	var deleted bool
	registrar.rwLock.Lock()
	defer registrar.rwLock.Unlock()
	if modules, ok := registrar.moduleTypeMap[moduleType]; ok {
		if _, ok := modules[mid]; ok {
			delete(modules, mid)
			deleted = true
		}
	}
	return deleted, nil
}

// 用于获取一个指定类型的组件的实例
// 本函数会基于负载均衡策略返回实例
func (registrar *myRegistrar) Get(moduleType Type) (Module, error) {
	modules, err := registrar.GetAllByType(moduleType)
	if err != nil {
		return nil, err
	}
	minScore := uint64(0)
	var selectedModule Module
	for _, module := range modules {
		SetScore(module)
		score := module.Score()
		if minScore == 0 || score < minScore {
			selectedModule = module
			minScore = score
		}
	}
	return selectedModule, nil
}

// 用于获取指定类型的所有组件实例
func (registrar *myRegistrar) GetAllByType(moduleType Type) (map[MID]Module, error) {
	if !LegalType(moduleType) {
		errMsg := fmt.Sprintf("组件实例类型异常: %s", moduleType)
		return nil, errors.NewIllegalParameterError(errMsg)
	}
	registrar.rwLock.RLock()
	defer registrar.rwLock.RUnlock()
	modules := registrar.moduleTypeMap[moduleType]
	if len(modules) == 0 {
		return nil, ErrNotFoundModuleInstance
	}
	result := map[MID]Module{}
	for mid, module := range modules {
		result[mid] = module
	}
	return result, nil
}

// 用于获取所有组件实例
func (registrar *myRegistrar) GetAll() map[MID]Module {
	result := map[MID]Module{}
	registrar.rwLock.RLock()
	defer registrar.rwLock.RUnlock()
	for _, modules := range registrar.moduleTypeMap {
		for mid, module := range modules {
			result[mid] = module
		}
	}
	return result
}

// 清除所有组件注册记录
func (registrar *myRegistrar) Clear() {
	registrar.rwLock.Lock()
	defer registrar.rwLock.Unlock()
	registrar.moduleTypeMap = map[Type]map[MID]Module{}
}
