package managers

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/leilei3167/basic/pkg/shutdown"
)

const Name = "PosixSignalManager"

// PosixSignalManager 提供一个ShutdownManager的实现
type PosixSignalManager struct {
	signals []os.Signal
}

func NewPosixSignalManager(sig ...os.Signal) *PosixSignalManager {
	if len(sig) == 0 {
		//默认接收SIGINT和SIGTERM信号
		sig := make([]os.Signal, 2)
		sig[0] = syscall.SIGINT
		sig[1] = syscall.SIGTERM
	}
	return &PosixSignalManager{signals: sig}
}
func (p *PosixSignalManager) GetName() string {
	return Name
}

func (p *PosixSignalManager) Start(gs shutdown.GSInterface) error {
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, p.signals...)

		//收到信号时执行退出
		<-c
		gs.StartShutdown(p)
	}()
	return nil
}

func (p *PosixSignalManager) ShutdownStart() error {
	//可以指定在关机前要执行的操作
	return nil
}

func (p *PosixSignalManager) ShutdownFinish() error {
	os.Exit(0) //成功退出
	return nil
}
