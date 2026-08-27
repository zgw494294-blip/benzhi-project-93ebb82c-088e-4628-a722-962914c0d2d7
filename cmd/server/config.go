package main

import (
	"errors"
	"flag"
	"fmt"
	"net"
	"strconv"
)

type config struct {
	addr      string
	dataDir   string
	selfcheck bool
}

func parseConfig(args []string) (config, error) {
	set := flag.NewFlagSet("口述影像脚本排演台", flag.ContinueOnError)
	var cfg config
	set.StringVar(&cfg.addr, "addr", "127.0.0.1:19081", "HTTP 监听地址，必须为回环地址")
	set.StringVar(&cfg.dataDir, "data", "./data", "本地持久化目录")
	set.BoolVar(&cfg.selfcheck, "selfcheck", false, "启动真实 HTTP 服务并执行有界全流程自检")
	if err := set.Parse(args); err != nil {
		return config{}, err
	}
	if set.NArg() != 0 {
		return config{}, fmt.Errorf("未知参数: %v", set.Args())
	}
	if err := validateAddress(cfg.addr); err != nil {
		return config{}, err
	}
	if cfg.dataDir == "" {
		return config{}, errors.New("数据目录不能为空")
	}
	return cfg, nil
}

func validateAddress(address string) error {
	host, portText, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("监听地址格式无效: %w", err)
	}
	if host == "" {
		return errors.New("监听地址必须明确指定回环主机")
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return errors.New("监听地址仅允许 127.0.0.0/8 或 ::1 回环地址")
	}
	port, err := strconv.Atoi(portText)
	if err != nil || port < 1 || port > 65535 {
		return errors.New("监听端口必须在 1 到 65535 之间")
	}
	return nil
}
