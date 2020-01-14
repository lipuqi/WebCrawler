package module

// 组件的类型
type Type string

// 当前认可的组件类型的常量
const (
	// 下载器
	TYPE_DOWNLOADER Type = "downloader"
	// 分析器
	TYPE_ANALYZER Type = "analyzer"
	// 条目处理管道
	TYPE_PIPELINE Type = "pipeline"
)

// 合法的组件类型-字母的映射
var legalTypeLetterMap = map[Type]string{
	TYPE_DOWNLOADER: "D",
	TYPE_ANALYZER:   "A",
	TYPE_PIPELINE:   "P",
}
