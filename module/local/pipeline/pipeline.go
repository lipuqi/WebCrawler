package pipeline

import (
	"../../../log"
	"../../../module"
	"../../stub"
	"fmt"
)

// logger 代表日志记录器。
var logger = log.DLogger()

// 代表条目处理管道的实现类型
type myPipeline struct {
	// 代表组件基础实例
	stub.ModuleInternal
	// 代表条目处理器的列表
	itemProcessors []module.ProcessItem
	// 代表处理是否需要快速失败
	failFast bool
}

func New(mid module.MID, itemProcessors []module.ProcessItem,
	scoreCalculator module.CalculateScore) (module.Pipeline, error) {
	moduleBase, err := stub.NewModuleInternal(mid, scoreCalculator)
	if err != nil {
		return nil, err
	}
	if itemProcessors == nil {
		return nil, genParameterError("空的条目处理列表")
	}
	if len(itemProcessors) == 0 {
		return nil, genParameterError("条目处理列表长度为空")
	}
	var innerProcessors []module.ProcessItem
	for i, pipeline := range itemProcessors {
		if pipeline == nil {
			err := genParameterError(fmt.Sprintf("空的条目处理管道[%d]", i))
			return nil, err
		}
		innerProcessors = append(innerProcessors, pipeline)
	}
	return &myPipeline{
		ModuleInternal: moduleBase,
		itemProcessors: innerProcessors,
	}, nil
}

func (pipeline *myPipeline) ItemProcessors() []module.ProcessItem {
	processors := make([]module.ProcessItem, len(pipeline.itemProcessors))
	copy(processors, pipeline.itemProcessors)
	return processors
}

func (pipeline *myPipeline) Send(item module.Item) []error {
	pipeline.ModuleInternal.IncrHandlingNumber()
	defer pipeline.ModuleInternal.DecrHandlingNumber()
	pipeline.ModuleInternal.IncrCalledCount()
	var errs []error
	if item == nil {
		err := genParameterError("空条目")
		errs = append(errs, err)
		return errs
	}
	pipeline.ModuleInternal.IncrAcceptedCount()
	//logger.Infof("处理项目 %+v... \n", item)
	var currentItem = item
	for _, processor := range pipeline.itemProcessors {
		processedItem, err := processor(currentItem)
		if err != nil {
			errs = append(errs, err)
			if pipeline.failFast {
				break
			}
		}
		if processedItem != nil {
			currentItem = processedItem
		}
	}
	if len(errs) == 0 {
		pipeline.ModuleInternal.IncrCompletedCount()
	}
	return errs
}

func (pipeline *myPipeline) FailFast() bool {
	return pipeline.failFast
}

func (pipeline *myPipeline) SetFailFast(failFast bool) {
	pipeline.failFast = failFast
}

// 代表条目处理管道额外信息的摘要类型
type extraSummaryStruct struct {
	FailFast        bool `json:"fail_fast"`
	ProcessorNumber int  `json:"processor_number"`
}

func (pipeline *myPipeline) Summary() module.SummaryStruct {
	summary := pipeline.ModuleInternal.Summary()
	summary.Extra = extraSummaryStruct{
		FailFast:        pipeline.failFast,
		ProcessorNumber: len(pipeline.itemProcessors),
	}
	return summary
}
