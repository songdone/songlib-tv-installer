package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/bitxeno/atvloadly/internal/offlineregistry"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	root := flag.String("root", "", "bundled image directory")
	addr := flag.String("listen", "127.0.0.1:15933", "loopback address")
	lifetime := flag.Duration("ttl", 20*time.Minute, "maximum lifetime")
	flag.Parse()
	host, _, err := net.SplitHostPort(*addr)
	if err != nil || host != "127.0.0.1" || *lifetime <= 0 {
		fmt.Fprintln(os.Stderr, "离线镜像源只能监听本机回环地址")
		os.Exit(1)
	}
	registry, err := offlineregistry.Open(*root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "离线镜像文件校验失败:", err)
		os.Exit(1)
	}
	defer func() { _ = registry.Close() }()
	server := &http.Server{Addr: *addr, Handler: registry, ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 30 * time.Second}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go func() {
		timer := time.NewTimer(*lifetime)
		defer timer.Stop()
		select {
		case <-ctx.Done():
		case <-timer.C:
		}
		_ = server.Close()
	}()
	if err = server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Fprintln(os.Stderr, "无法启动本机离线镜像源:", err)
		os.Exit(1)
	}
}
