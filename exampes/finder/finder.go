package main

import (
	"../../log"
	sched "../../scheduler"
	"./bm1365Model"
	lib "./bm1365Model"
	"./monitor"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
)

// 命令参数
var (
	firstURL string
	domains  string
	depth    uint
	dirPath  string
)

// 日志记录器
var logger = log.DLogger()

func init() {
	flag.StringVar(&firstURL, "first", "http://www.bml365.com/qy/prod/v/1249-1598",
		"您要访问的第一个URL")
	flag.StringVar(&domains, "domains", "www.bml365.com",
		"域名白名单"+" "+
			"请使用逗号分隔的多个域")
	flag.UintVar(&depth, "depth", 1, "爬行的深度")
	flag.StringVar(&dirPath, "dir", "G:/bm1365/pictures",
		"您要保存图像文件的路径")
}

func Usage() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "\tfinder [flages] \n")
	fmt.Fprintf(os.Stderr, "Flags:\n")
	flag.PrintDefaults()
}

func main() {
	flag.Usage = Usage
	flag.Parse()
	// 创建调度器
	scheduler := sched.NewScheduler()
	// 准备调度器的初始化参数
	domainParts := strings.Split(domains, ",")
	acceptedDomains := []string{}
	for _, domain := range domainParts {
		domain = strings.TrimSpace(domain)
		if domain != "" {
			acceptedDomains = append(acceptedDomains, domain)
		}
	}
	requestArgs := sched.RequestArgs{
		AcceptedDomains: acceptedDomains,
		MaxDepth:        uint32(depth),
	}
	dataArgs := sched.DataArgs{
		ReqBufferCap:         50,
		ReqMaxBufferNumber:   1000,
		RespBufferCap:        50,
		RespMaxBufferNumber:  10,
		ItemBufferCap:        50,
		ItemMaxBufferNumber:  100,
		ErrorBufferCap:       50,
		ErrorMaxBufferNumber: 1,
	}
	downloaders, err := lib.GetDownloaders(1)
	if err != nil {
		logger.Fatalf("创建下载器发生异常: %s", err)
	}
	analyzers, err := lib.GetAnalyzers(1)
	if err != nil {
		logger.Fatalf("创建分析器发生异常: %s", err)
	}
	pipelines, err := lib.GetPipelines(1, dirPath)
	if err != nil {
		logger.Fatalf("创建条目处理管道发生异常: %s", err)
	}
	moduleArgs := sched.ModuleArgs{
		Downloaders: downloaders,
		Analyzers:   analyzers,
		Pipelines:   pipelines,
	}
	// 初始化调度器
	err = scheduler.Init(requestArgs, dataArgs, moduleArgs)
	if err != nil {
		logger.Fatalf("初始化调度器发生异常: %s", err)
	}
	// 准备监控参数
	checkInterval := 2 * time.Second
	summarizeInterval := time.Second
	maxIdleCount := uint(15)
	// 开始监控
	checkCountChan := monitor.Monitor(
		scheduler, checkInterval, summarizeInterval, maxIdleCount, true, lib.Record)
	// 准备调度器的启动参数
	/*	firstHTTPReq, err := http.NewRequest("GET", firstURL, nil)
		if err != nil {
			logger.Fatal(err)
			return
		}*/
	// 开启调度器
	err = scheduler.Start(nil)
	if err != nil {
		logger.Fatalf("开启调度器发送异常: %s", err)
	}
	//自定义首次发送
	bm1365Model.InitReqList(1, 3, scheduler)
	// 等待监控结束
	<-checkCountChan
	bm1365Model.SaveExcel("G:/bm1365/bm1365.xlsx")
}
