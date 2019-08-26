package main

import (
	"fmt"
	"github.com/sniperHW/flyfish/conf"
	flyfish "github.com/sniperHW/flyfish/server"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime"
	"syscall"
)

func main() {
	flyfish.Must(nil, conf.LoadConfig(os.Args[1]))

	flyfish.InitLogger()

	if !flyfish.LoadTableConfig() {
		fmt.Println("InitTableConfig failed")
		return
	}

	go func() {
		http.ListenAndServe("0.0.0.0:8899", nil)
	}()

	defer func() {
		if r := recover(); r != nil {
			buf := make([]byte, 1024*1024*4)
			l := runtime.Stack(buf, false)
			flyfish.Errorln(fmt.Sprintf("%v: %s", r, buf[:l]))
		}
	}()

	err := flyfish.Start()
	if nil == err {
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGINT) //监听指定信号
		_ = <-c                          //阻塞直至有信号传入
		flyfish.Stop()
		fmt.Println("server stop")
	} else {
		fmt.Println(err)
	}
}