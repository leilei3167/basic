// Copyright 2020 Lingfei Kong <colin404@foxmail.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"github.com/spf13/pflag"
	"net-mapping/pkg/log"
)

func main() {

	// logger配置,设置默认值
	opts := &log.Options{
		Level:            "debug",
		Format:           "console",
		EnableColor:      false, // if you need output to local path, with EnableColor must be false.
		DisableCaller:    false,
		OutputPaths:      []string{"test.log", "stdout"},
		ErrorOutputPaths: []string{"error.log"},
	}
	opts.AddFlags(pflag.CommandLine) //添加到flagSet中,可修改默认的opts
	pflag.Parse()

	// 初始化全局logger
	log.Init(opts)
	defer log.Flush()

	// Debug、Info(with field)、Warnf、Errorw使用
	log.Debug("This is a debug message")
	log.Info("This is a info message", log.Int32("int_key", 10))
	log.Warnf("This is a formatted %s message", "warn")
	log.Errorw("Message printed with Errorw", "X-Request-ID", "fbf54504-64da-4088-9b86-67824a7fb508")

	// WithValues使用,使用指定的k-v创建logger,之后通过它打印的日志都会带有该kv
	lv := log.WithValues("X-Request-ID", "7a7b9f24-4cae-4b2a-9464-69088b45b904")
	lv.Infow("Info message printed with [WithValues] logger")
	lv.Infow("Debug message printed with [WithValues] logger")

	// Context使用
	ctx := lv.WithContext(context.Background()) //将lv存入ctx中传递
	lc := log.FromContext(ctx)                  //从ctx中取出携带有k-v的logger
	lc.Info("Message printed with [WithContext] logger")

	//再lv之中新建一个带有名称的子logger
	ln := lv.WithName("test")
	ln.Info("Message printed with [WithName] logger")

	// V level使用
	log.V(log.InfoLevel).Info("This is a V level message")
	log.V(log.ErrorLevel).
		Infow("This is a V level message with fields", "X-Request-ID", "7a7b9f24-4cae-4b2a-9464-69088b45b904")
}
