package shutdown

import (
	"testing"
)

//实现ShutdownManager接口,用于测试

type fakeManager func() error

func (f fakeManager) GetName() string {
	return "test-fake"
}

func (f fakeManager) ShutdownStart() error {
	return f()
}

func (f fakeManager) ShutdownFinish() error {
	return f()
}

func (f fakeManager) Start(gs GSInterface) error {
	return f()
}

func TestCallbackGetCalled(t *testing.T) {
	gs := New()
	c := make(chan int, 100)
	for i := 0; i < 100; i++ {
		//添加一百个回调函数
		gs.AddShutdownCallback(ShutdownFunc(func(s string) error {
			c <- 1
			return nil
		}))
	}
	gs.StartShutdown(fakeManager(func() error {
		return nil
	}))
	if len(c) != 100 {
		t.Errorf("Expected 100 but got: %d", len(c))
	}
}

func TestStartGetsCalled(t *testing.T) {
	gs := New()
	c := make(chan int, 100)
	for i := 0; i < 100; i++ {
		gs.AddShutdownManager(fakeManager(func() error {
			c <- 1
			return nil
		}))
	}
	gs.Start() //start会启动所有的manager
	if len(c) != 100 {
		t.Error("Expected 100 Start to be called, got ", len(c))
	}
}
