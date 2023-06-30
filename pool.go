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

	preAlloc bool //表示是否需要提前创建好Worker 默认为false
	block    bool //在Worker满了的时候，是否阻塞等待(挂起) 默认为true

}

func WithBlock(block bool) Option {
	return func(pool *Pool) {
		pool.block = block
	}
}

func WithPreAlloc(preAlloc bool) Option {
	return func(pool *Pool) {
		pool.preAlloc = preAlloc
	}
}

func New(cap int, opt ...Option) *Pool {
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
		preAlloc: false,
		block:    true,
	}

	for _, opt := range opt {
		opt(p)
	}
	fmt.Printf("worker pool start(preAlloc=%t)\n", p.preAlloc)

	if p.preAlloc {
		for i := 0; i < p.capacity; i++ {
			p.newWorker(i + 1)
			p.active <- struct{}{}
		}
	}

	go p.run()

	return p
}

func (p *Pool) run() {
	idx := len(p.active) //worker的唯一ID

	if !p.preAlloc {
		// 监听是否有任务提交，如果存在，就创建一个Worker 处理任务
	loop:
		for t := range p.tasks {
			p.returnTask(t)
			select {
			case <-p.quit:
				return
			case p.active <- struct{}{}:
				idx++
				p.newWorker(idx)
			default:
				break loop
			}
		}
	}

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

func (p *Pool) returnTask(t Task) {
	go func() {
		p.tasks <- t
	}()
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
var ErrNoIdleWorkerInPool = errors.New("no idle worker in pool")

func (p *Pool) Schedule(t Task) error {
	select {
	case <-p.quit:
		return ErrWorkerPollFreed
	case p.tasks <- t:
		return nil
	default:
		if p.block {
			p.tasks <- t
			return nil
		}
		return ErrNoIdleWorkerInPool
	}
}

func (p *Pool) Free() {
	close(p.quit)
	p.wg.Wait()
	fmt.Printf("workpool freed\n")
}

type Option func(pool *Pool)
