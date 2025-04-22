package dispatcher

import (
	"sync"

	"github.com/iamlongalong/listenmail/pkg/types"
)

// Dispatcher implements the types.Dispatcher interface
type Dispatcher struct {
	handlers    []types.Handler
	handlersMap map[types.Handler]bool
	mu          sync.RWMutex
	workers     chan struct{}
	mailCh      chan *dispatchJob
	done        chan struct{}
}

type dispatchJob struct {
	mail *types.Mail
	err  chan error
}

// New creates a new Dispatcher
func New() *Dispatcher {
	d := &Dispatcher{
		handlers:    make([]types.Handler, 0),
		handlersMap: make(map[types.Handler]bool),
		workers:     make(chan struct{}, 10),      // 最多10个并发worker
		mailCh:      make(chan *dispatchJob, 100), // 邮件处理队列，缓冲100个
		done:        make(chan struct{}),
	}

	// 启动worker pool
	go d.run()

	return d
}

// run 管理worker pool
func (d *Dispatcher) run() {
	for {
		select {
		case <-d.done:
			return
		case job := <-d.mailCh:
			// 获取worker槽位
			d.workers <- struct{}{}

			// 启动goroutine处理邮件
			go func(job *dispatchJob) {
				defer func() {
					<-d.workers // 释放worker槽位
				}()

				// 处理邮件
				err := d.dispatchToHandlers(job.mail)
				job.err <- err
			}(job)
		}
	}
}

// AddHandler implements types.Dispatcher
func (d *Dispatcher) AddHandlers(handlers ...types.Handler) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, h := range handlers {
		if !d.handlersMap[h] {
			d.handlers = append(d.handlers, h)
			d.handlersMap[h] = true
		}
	}
	return nil
}

// RemoveHandler implements types.Dispatcher
func (d *Dispatcher) RemoveHandlers(handlers ...types.Handler) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, h := range handlers {
		for i, handler := range d.handlers {
			if handler == h {
				d.handlers = append(d.handlers[:i], d.handlers[i+1:]...)
				delete(d.handlersMap, h)
				break
			}
		}
	}
	return nil
}

// Dispatch implements types.Dispatcher
func (d *Dispatcher) Dispatch(mail *types.Mail) error {
	// 创建新的任务
	job := &dispatchJob{
		mail: mail,
		err:  make(chan error, 1),
	}

	// 发送到处理队列
	select {
	case d.mailCh <- job:
		// 等待处理完成
		return <-job.err
	case <-d.done:
		return nil
	}
}

// dispatchToHandlers 将邮件分发给匹配的处理器
func (d *Dispatcher) dispatchToHandlers(mail *types.Mail) error {
	d.mu.RLock()
	handlers := make([]types.Handler, len(d.handlers))
	copy(handlers, d.handlers)
	d.mu.RUnlock()

	for _, handler := range handlers {
		if handler.Match(mail) {
			if err := handler.Handle(mail); err != nil {
				return err
			}
		}
	}
	return nil
}

// Close 关闭dispatcher
func (d *Dispatcher) Close() error {
	close(d.done)
	return nil
}
