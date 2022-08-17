package main

import (
	"context"
	"fmt"
	"golang.org/x/sync/errgroup"
	"log"
	"net-mapping/pkg/errors"
	"net-mapping/pkg/shutdown"
	"net-mapping/pkg/shutdown/managers"
	"net/http"
)

func main() {
	//模拟一个进程中有多个不同的服务需要优雅退出
	//http服务
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		fmt.Fprintln(writer, "hello server1")
	})
	httpserver1 := http.Server{Addr: ":8080", Handler: mux}

	//另一个http服务
	mux2 := http.NewServeMux()
	mux2.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		fmt.Fprintln(writer, "this is server2")
	})
	server2 := http.Server{Addr: ":8081", Handler: mux2}

	gs := shutdown.New()
	//添加默认的信号监听manager
	gs.AddShutdownManager(managers.NewPosixSignalManager())
	gs.SetErrorHandler(shutdown.ErrorFunc(func(err error) {
		log.Printf("got an error when shuting down:%v", err)
	}))
	gs.AddShutdownCallback(shutdown.ShutdownFunc(func(s string) error {
		fmt.Println("httpserver1 开始退出!")
		return httpserver1.Shutdown(context.Background())
	}))
	gs.AddShutdownCallback(shutdown.ShutdownFunc(func(s string) error {
		fmt.Println("server2 开始退出!")
		return server2.Shutdown(context.Background())
	}))
	gs.Start()

	var g errgroup.Group
	g.Go(func() error {
		return httpserver1.ListenAndServe()
	})
	g.Go(func() error {
		return server2.ListenAndServe()
	})

	if err := g.Wait(); err != nil {
		if errors.Is(err, http.ErrServerClosed) {
			fmt.Println("all server closed")
			return
		} else {
			fmt.Println("err:", err)
			return
		}
	}
}
