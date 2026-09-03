package scheduler

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

type Runner func(context.Context) error

// Service 负责把单个任务按 cron 周期跑起来。
// 它会在启动后立即执行一次，并在任务未结束时跳过重入执行。
type Service struct {
	cron     *cron.Cron
	interval string
	timeout  time.Duration
	job      Runner
	logf     func(string, ...any)
	mu       sync.Mutex
	running  bool
	ctx      context.Context
	cancel   context.CancelFunc
}

func New(interval string, timeout time.Duration, job Runner, logf func(string, ...any)) (*Service, error) {
	// 调度表达式必须存在；单次任务超时则使用固定默认值。
	if interval == "" {
		return nil, errors.New("interval is required")
	}
	if timeout <= 0 {
		timeout = 10 * time.Minute
	}
	if logf == nil {
		logf = func(string, ...any) {}
	}
	c := cron.New(cron.WithSeconds())
	return &Service{cron: c, interval: interval, timeout: timeout, job: job, logf: logf}, nil
}

func (s *Service) Start(parent context.Context, runOnStart bool) error {
	// 绑定父 context，收到退出信号时一起结束。
	s.ctx, s.cancel = context.WithCancel(parent)

	// 用 6 段 cron 表达式注册周期任务。
	if _, err := s.cron.AddFunc(s.interval, func() {
		s.runOnce()
	}); err != nil {
		return err
	}
	// 服务启动后先跑一次，避免等到下一个调度点。
	if runOnStart {
		go s.runOnce()
	}
	s.cron.Start()
	return nil
}

func (s *Service) Stop() <-chan struct{} {
	if s.cancel != nil {
		s.cancel()
	}
	return s.cron.Stop().Done()
}

func (s *Service) runOnce() {
	// 防止上一轮还没结束，下一轮又进来。
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		s.logf("skip scheduled run: previous job still running")
		return
	}
	s.running = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
	}()

	// 单轮任务强制受 timeout 约束。
	ctx, cancel := context.WithTimeout(s.ctx, s.timeout)
	defer cancel()

	if err := s.job(ctx); err != nil {
		s.logf("job failed: %v", err)
	}
}
