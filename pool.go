package workerpool

import (
	"errors"
	"fmt"
	"sync"
)

type Task func()

var defaultCap = 8
var maxCap = 12

type Pool struct {
	capacity int //最大同时工作的 routine 数

	active chan struct{} //通过channel限制worker数量
	tasks  chan Task     //通过tasks channel 和worker通信

	wg   sync.WaitGroup
	quit chan struct{}
}

func New(cap int) *Pool {
	if cap <= 0 {
		cap = defaultCap
	}
	if cap > maxCap {
		cap = maxCap
	}

	p := &Pool{
		capacity: cap,
		active:   make(chan struct{}, cap),
		tasks:    make(chan Task),
		quit:     make(chan struct{}),
	}

	fmt.Println("worker pool start")
	go p.run()

	return p
}

// todo 这里不就是预先分配了吗？
func (p *Pool) run() {
	idx := 0 //worker的唯一ID

	for {
		select {
		case <-p.quit:
			return
		case p.active <- struct{}{}:
			idx++
			p.newWorker(idx)
		}
	}
}

func (p *Pool) newWorker(id int) {
	p.wg.Add(1)
	go func() {
		//防止运行时异常影响整个线程池
		defer func() {
			if err := recover(); err != nil {
				fmt.Printf("worker[%03d]: recover panic[%s] and exit\n", id, err)
			}
			<-p.active
			p.wg.Done()
		}()

		fmt.Printf("worker[%03d]: running\n", id)

		for {
			select {
			case <-p.quit:
				fmt.Printf("worker[%03d]: stop\n", id)
				return
			case t := <-p.tasks:
				fmt.Printf("worker[%03d]: receive a task\n", id)
				t() //执行任务
			}
		}

	}()
}

var ErrWorkerPollFreed = errors.New("worker pool freed")

func (p *Pool) Schedule(t Task) error {
	select {
	case <-p.quit:
		return ErrWorkerPollFreed
	case p.tasks <- t:
		return nil
	}
}

func (p *Pool) Free() {
	close(p.quit)
	p.wg.Wait()
	fmt.Printf("workpool freed\n")
}
