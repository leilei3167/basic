# shutdown 包设计

## 关键

将优雅关闭抽象为`ShutdownManager`接口:
```go
type ShutdownManager interface {
	GetName() string
	// Start 会监听退出的请求,如果收到退出请求,将会调用gs的退出方法
	Start(gs GSInterface) error
	// ShutdownStart 当接收到退出请求时,会先执行
	ShutdownStart() error
	// ShutdonwFinish 当所有的回调函数被执行完毕时,会执行
	ShutdownFinish() error
}
```
以上接口可以实现不同信号接收形式,默认实现的posix是接收默认的信号


其参数`GSInterface`为在执行优雅关停中的行为的抽象
```go
type GSInterface interface {
	StartShutdown(sm ShutdownManager)
	ReportError(err error)
	AddShutdownCallback(shutdowncallback ShutdownCallback)
}
```

`GracefulShutdown`是一个实现该接口的实例,其中维护了回调函数和manager的列表
```go
type GSInterface interface {
type GracefulShutdown struct {
callbacks    []ShutdownCallback
managers     []ShutdownManager
errorHandler ErrorHandler
}
}
```

其`start`方法会启动所有的manager的`Start`方法,并将自己作为参数传入:

```go
func (gs *GracefulShutdown) Start() error {
//启动所有的manager
for _, manager := range gs.managers {
if err := manager.Start(gs); err != nil {//因此要求manager的Start方法不应该是阻塞的
return err
}
}
return nil
}
```

其中任意一个manager在收到信号时,都会使得gs执行所有的回调函数:
```go
	var wg sync.WaitGroup
	for _, shutdownCallback := range gs.callbacks {
		wg.Add(1)
		go func(shutdownCallback ShutdownCallback) {
			defer wg.Done()
			gs.ReportError(shutdownCallback.OnShutdown(sm.GetName()))
		}(shutdownCallback)
	}
	wg.Wait()
```


关键在于`ShutdownManager`如何实现其`Start`方法,其中定义如何接收,接收何种信号,满足什么条件后执行gs的`StarShutdown`方法


对要添加的函数也抽象为了接口的设计,并参考http.Handler的设计添加了帮助函数,便于转换

```go
// ShutdownCallback 是退出回调函数要实现的接口
type ShutdownCallback interface {
	// OnShutdown 定义了在关闭时要执行的动作
	OnShutdown(name string) error
}

// ShutdownFunc 帮助函数,可以将相同签名的函数快速转换为ShutdownCallback的实现,参照了标准库Handler的设计
type ShutdownFunc func(string) error

func (f ShutdownFunc) OnShutdown(shutdownManager string) error {
	return f(shutdownManager)
}

type ErrorHandler interface {
// OnError 定义当错误发生时需要执行的操作
OnError(err error)
}

type ErrorFunc func(err error)

func (f ErrorFunc) OnError(err error) {
f(err)
}

```

实际使用时在添加回调函数时可以充分利用函数闭包来传递参数,如:

```go
server2 := http.Server{Addr: ":8081", Handler: mux2}

gs.AddShutdownCallback(shutdown.ShutdownFunc(func(s string) error {
		fmt.Println("server2 开始退出!")
		return server2.Shutdown(context.Background())
	}))

```