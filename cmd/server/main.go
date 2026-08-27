package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "启动失败:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	cfg, err := parseConfig(args)
	if err != nil {
		return err
	}
	cleanup := func() {}
	if cfg.selfcheck {
		cfg.dataDir, cleanup, err = prepareSelfcheckData()
		if err != nil {
			return err
		}
		defer cleanup()
	}
	app, err := newApplication(cfg)
	if err != nil {
		return err
	}
	listener, err := app.listen()
	if err != nil {
		_ = app.store.Close()
		return err
	}
	serveDone := make(chan error, 1)
	go func() { serveDone <- app.serve(listener) }()
	if cfg.selfcheck {
		checkErr := runSelfcheck("http://" + listener.Addr().String())
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		shutdownErr := app.shutdown(ctx)
		serveErr := <-serveDone
		if checkErr != nil {
			return checkErr
		}
		if shutdownErr != nil {
			return shutdownErr
		}
		if serveErr != nil {
			return serveErr
		}
		fmt.Println("自检通过：真实 HTTP 全流程已完成，退修改稿、重新排演、摘要和双格式导出一致。")
		return nil
	}
	fmt.Printf("口述影像脚本排演台已监听 http://%s\n", listener.Addr().String())
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	select {
	case err := <-serveDone:
		return err
	case <-signals:
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return app.shutdown(ctx)
	}
}
