package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	statsd_relay "statsd-relay-go/statsd_relay"
	"syscall"
	"time"
)

var (
	configFile = flag.String("config", "config.yml", "config file")
	queueSize  = flag.Int("queueSize", 8192, "queue buffer size")
	print      = flag.Bool("print", false, "print config and exit")
	workerNum  = flag.Int("workerNum", 16, "filter worker thread number")
	distriNum  = flag.Int(
		"distributionNum", 4, "distribution worker thread number")
)

type WorkingQueue struct {
	InputQueue  chan []byte // 输入队列
	WlQueue     chan []byte // 发给白名单的队列
	BlQueue     chan []byte // 发给黑名单的队列
	OutputQueue chan []byte // 发给输出模块的队列
}

func output(q chan []byte) {
	out := statsd_relay.StdoutOutput{}
	out.Init()
	out.Write(q)
}

func printQueueStats(wq WorkingQueue) {
	log.Println("Queue Stats:")
	log.Printf("  InputQueue: %d\n", len(wq.InputQueue))
	log.Printf("  WlQueue: %d\n", len(wq.WlQueue))
	log.Printf("  BlQueue: %d\n", len(wq.BlQueue))
	log.Printf("  OutputQueue: %d\n", len(wq.OutputQueue))
}

// 打印读到的配置情况
func printConfig(config statsd_relay.Config) {
	fmt.Printf("black report: %v\n", config.BlacklistReport)
	fmt.Println("Here is the rewrites:")
	for _, rule := range config.Filters.Rewrite {
		fmt.Println("  rewrite: " + rule.Regexp)
	}
	fmt.Println("Here is the whitelists:")
	for _, wl := range config.Filters.Whitelist {
		fmt.Println("  whitelist: " + wl)
	}
	fmt.Println("Here is the blacklists:")
	for _, wl := range config.Filters.Whitelist {
		fmt.Println("  blacklist: " + wl)
	}
	for _, in := range config.Input {
		fmt.Printf("input: %s://%s:%d\n", in.Type, in.Host, in.Port)
	}
	for _, o := range config.Output {
		fmt.Printf("output: %s://%s:%d\n", o.Type, o.Host, o.Port)
	}
}

// 初始化logger
func setupLogger() {
	log.SetPrefix("[main] ")
	log.SetFlags(
		log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)
}

// 初始化工作队列
func setupWorkingQueue(size int) WorkingQueue {
	rv := WorkingQueue{
		InputQueue:  make(chan []byte, size),
		WlQueue:     make(chan []byte, size),
		BlQueue:     make(chan []byte, size),
		OutputQueue: make(chan []byte, size),
	}
	return rv
}

func setupInput(inputs []statsd_relay.InputConfig, q chan []byte) {
	for _, in := range inputs {
		if in.Type != "udp" {
			log.Fatalf("unsupported input type: %s\n", in.Type)
		}

		uin := statsd_relay.UDPInput{
			Host: in.Host,
			Port: in.Port,
		}
		uin.Init()
		// 目前每个输入都只开一个routine
		go uin.Drain(q)
	}
}

func setupOutput(
	outputs []statsd_relay.OutputConfig, q chan []byte,
	queueSize int, workerNum int) {

	var qs [](chan []byte)

	for _, out := range outputs {
		if out.Type == "udp" {
			o := statsd_relay.UdpOutput{
				Host: out.Host,
				Port: out.Port,
			}
			o.Init()
			_q := make(chan []byte, queueSize)
			log.Printf(
				"setup udp output to %s:%d\n",
				out.Host, out.Port)
			go o.Write(_q)
			qs = append(qs, _q)
		} else if out.Type == "stdout" {
			o := statsd_relay.StdoutOutput{}
			o.Init()
			_q := make(chan []byte, queueSize)
			go o.Write(_q)
			qs = append(qs, _q)
			log.Println("setup stdout output")
		} else {
			log.Fatalf("unsupported output type: %s\n", out.Type)
		}
	}

	setupDistribution(q, qs, workerNum)

}

// 配置输出分发器
func setupDistribution(
	in chan []byte, qs [](chan []byte), workerNum int) {
	for i := 0; i < workerNum; i++ {
		go distribute(in, qs)
	}
}

// 分发器实际工作函数
func distribute(in chan []byte, qs [](chan []byte)) {
	for m := range in {
		for _, _q := range qs {
			_q <- m
		}
	}
}

// 配置过滤器
func setupFilter(
	config statsd_relay.FilterConfig, wq WorkingQueue,
	blacklistReport bool, workerNum int) {

	// 管道定义
	var rewriteIn chan []byte = wq.InputQueue
	var rewriteOut chan []byte = wq.WlQueue
	var whitelistIn chan []byte = wq.WlQueue
	var whitelistMatchOut chan []byte = wq.OutputQueue
	var whitelistNoMatchOut chan []byte = wq.BlQueue
	var blacklistIn chan []byte = wq.BlQueue
	var blacklistOut chan []byte = wq.OutputQueue

	// 配置重写
	setupRewrite(config.Rewrite, rewriteIn, rewriteOut, workerNum)

	// 配置白名单
	setupWhitelist(
		config.Whitelist, whitelistIn,
		whitelistMatchOut, whitelistNoMatchOut, workerNum)
	// 配置黑名单

	// 检查是否有黑名单过滤器
	setupBlacklist(
		config.Blacklist, blacklistIn, blacklistOut,
		workerNum, blacklistReport)
}

func setupRewrite(
	rewrites []statsd_relay.RewriteFilterConfig,
	in chan []byte, out chan []byte, workerNum int) {

	if len(rewrites) <= 0 {
		log.Println("no rewrites defined")
		// 没有定义重写，直接将输入输出短接
		go joinPipe(in, out)
		return
	}

	// 根据配置初始化重写过滤器规则
	var filters []statsd_relay.RewriteFilter
	for _, rewriteConfig := range rewrites {
		log.Printf("setup rewrite: %s\n", rewriteConfig.Regexp)
		filter := statsd_relay.RewriteFilter{
			Pattern: rewriteConfig.Regexp,
			Replace: rewriteConfig.Replace,
		}
		filter.Init()
		filters = append(filters, filter)
	}

	// 启动线程
	for i := 0; i < workerNum; i++ {
		go rewrite(filters, in, out)
	}
}

// 重写过滤器真正执行的函数
func rewrite(
	filters []statsd_relay.RewriteFilter,
	in chan []byte, out chan []byte) {

	var replaced []byte
	var matched bool

	for b := range in {
		replaced = b
		for _, filter := range filters {
			matched, replaced = filter.Filter(b)
			if matched {
				break
			}
		}
		out <- replaced
	}
}

// 配置白名单过滤器
func setupWhitelist(
	whitelists []string,
	in chan []byte, out chan []byte, noMatchOut chan []byte,
	workerNum int) {

	if len(whitelists) <= 0 {
		log.Println("no whitelists defined")
		// 没有定义白名单，直接将输入输出短接
		go joinPipe(in, noMatchOut)
		return
	}

	// 根据配置初始化白名单过滤器规则
	var filters []statsd_relay.WhitelistFilter
	for _, wlConfig := range whitelists {
		log.Printf("setup whitelist: %s\n", wlConfig)
		filter := statsd_relay.WhitelistFilter{
			Pattern: wlConfig,
		}
		filter.Init()
		filters = append(filters, filter)
	}

	for i := 0; i < workerNum; i++ {
		go whitelist(filters, in, out, noMatchOut)
	}

}

// 白名单真正执行的函数
func whitelist(
	filters []statsd_relay.WhitelistFilter,
	in chan []byte, out chan []byte, noMatchOut chan []byte) {

	var matched bool

	for b := range in {
		for _, filter := range filters {
			matched = filter.Filter(b)
			if matched {
				// 匹配上的情况下直接发送给输出管道
				out <- b
				break
			}
		}

		// 未匹配上的情况下发送给黑名单处理
		if !matched {
			// log.Printf("sendto BlQueue %s\n", string(b))
			noMatchOut <- b
		}
	}
}

// 配置黑名单过滤器
func setupBlacklist(
	blacklists []string,
	in chan []byte, out chan []byte,
	workerNum int, report bool) {

	if len(blacklists) <= 0 {
		log.Println("no blacklists defined")
		// 没有定义黑名单，直接将输入输出短接
		go joinPipe(in, out)
		return
	}

	// 根据配置初始化黑名单过滤器规则
	var filters []statsd_relay.BlacklistFilter
	for _, blConfig := range blacklists {
		log.Printf("setup blacklist: %s\n", blConfig)
		filter := statsd_relay.BlacklistFilter{
			Pattern: blConfig,
		}
		filter.Init()
		filters = append(filters, filter)
	}

	for i := 0; i < workerNum; i++ {
		go blacklist(filters, in, out, report)
	}
}

// 黑名单真正执行的函数
func blacklist(
	filters []statsd_relay.BlacklistFilter,
	in chan []byte, out chan []byte, report bool) {

	var matched bool

	for b := range in {
		for _, filter := range filters {
			matched = filter.Filter(b)
			if matched {
				if report {
					log.Printf(
						"blacklisted: %s\n", string(b))
				}
				continue
			}
		}
		if !matched {
			out <- b
		}
	}
}

// 空的方法，只负责把一个管道内容接到另外一个管道里
func joinPipe(in chan []byte, out chan []byte) {
	for m := range in {
		out <- m
	}
}

func main() {

	// 本软件采用流水线设计思路，多个环节之间使用管道连接
	// 管道与工作池的关系：
	//
	// [input] => inputQueue => [worker section] => outputQueue
	//
	// 输出部分较为特殊，可能会有多个输出，即需要复制多份
	//
	//                               +=> outQueue1 => [output1]
	//                               |
	// outputQueue => [distribution] +=> outQueue2 => [output2]
	//                               |
	//                               +=> outQueue3 => [output3]
	//
	// 此外，工作部分内部3个环节之间也使用管道连接：
	//
	//       [rewrite]          [whitelist]          [blacklist]
	// in => [rewrite] => wq => [whitelist] => bq => [blacklist] => out
	//       [rewrite]          [whitelist]          [blacklist]
	//
	flag.Parse()

	// FIXME: 信号处理
	// 现在好像没什么特殊事情需要做

	config := statsd_relay.LoadConfig(*configFile)
	if *print {
		printConfig(config)
		os.Exit(1)
	}

	// 0. 初始化logger
	setupLogger()

	// 1. 初始化队列管道
	workingQueue := setupWorkingQueue(*queueSize)

	// 2. 初始化输入
	setupInput(config.Input, workingQueue.InputQueue)

	// 3. 初始化过滤器
	setupFilter(
		config.Filters, workingQueue,
		config.BlacklistReport, *workerNum)

	// 4. 初始化输出，含分发器模块
	setupOutput(
		config.Output, workingQueue.OutputQueue, *queueSize, *distriNum)

	// 5. 信号处理
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGHUP)
	go func() {
		for sig := range sigs {

			if sig == syscall.SIGHUP {
				// HUP信号打印队列情况
				printQueueStats(workingQueue)
			}
		}
	}()

	for {
		time.Sleep(time.Second * 1)
	}

}
