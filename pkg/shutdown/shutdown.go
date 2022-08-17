package shutdown

import "sync"

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

type ShutdownManager interface {
	GetName() string
	// Start 会监听退出的请求,如果收到退出请求,将会调用gs的相关方法
	Start(gs GSInterface) error
	// ShutdownStart 当接收到退出请求时,会先执行
	ShutdownStart() error
	// ShutdonwFinish 当所有的回调函数被执行完毕时,会执行
	ShutdownFinish() error
}

// ErrorHandler 可以传递给SetErrorHandler 来处理异步错误
type ErrorHandler interface {
	// OnError 定义当错误发生时需要执行的操作
	OnError(err error)
}

type ErrorFunc func(err error)

func (f ErrorFunc) OnError(err error) {
	f(err)
}

// GSInterface 作为参数传递给ShutdownManager
type GSInterface interface {
	StartShutdown(sm ShutdownManager)
	ReportError(err error)
	AddShutdownCallback(shutdowncallback ShutdownCallback)
}

// GracefulShutdown 是最主要的结构,维护回调函数等
type GracefulShutdown struct {
	callbacks    []ShutdownCallback
	managers     []ShutdownManager
	errorHandler ErrorHandler
}

func New() *GracefulShutdown {
	return &GracefulShutdown{
		callbacks: make([]ShutdownCallback, 0, 10),
		managers:  make([]ShutdownManager, 0, 3),
	}
}

// AddShutdownManager 添加一个监听退出请求的manager
func (gs *GracefulShutdown) AddShutdownManager(manager ShutdownManager) {
	gs.managers = append(gs.managers, manager)
}

// AddShutdownCallback 添加在接收到关闭请求时执行的回调函数
func (gs *GracefulShutdown) AddShutdownCallback(shutdowncallback ShutdownCallback) {
	gs.callbacks = append(gs.callbacks, shutdowncallback)
}
func (gs *GracefulShutdown) SetErrorHandler(errorHandler ErrorHandler) {
	gs.errorHandler = errorHandler
}

// StartShutdown 是被ShutdownManager调用的,调用时将会先调用ShutdownManager的ShutdownStart(执行所有的已注册回调
//等到所有的回调被执行完毕后,会再调用ShutdownFinish
func (gs *GracefulShutdown) StartShutdown(sm ShutdownManager) {
	gs.ReportError(sm.ShutdownStart())

	var wg sync.WaitGroup
	for _, shutdownCallback := range gs.callbacks {
		wg.Add(1)
		go func(shutdownCallback ShutdownCallback) {
			defer wg.Done()
			gs.ReportError(shutdownCallback.OnShutdown(sm.GetName()))
		}(shutdownCallback)
	}
	wg.Wait()
	gs.ReportError(sm.ShutdownFinish())
}

func (gs *GracefulShutdown) ReportError(err error) {
	if err != nil && gs.errorHandler != nil {
		gs.errorHandler.OnError(err)
	}
}

func (gs *GracefulShutdown) Start() error {
	//启动所有的manager
	for _, manager := range gs.managers {
		if err := manager.Start(gs); err != nil {
			return err
		}
	}
	return nil
}
