package core

import (
	"context"
	"sync"
	"time"
)

// Job описывает периодическую задачу.
type Job func(ctx context.Context) error

// Scheduler запускает задачи с фиксированным интервалом.
type Scheduler struct {
	interval time.Duration
	jobs     []Job
	wg       sync.WaitGroup
}

// NewScheduler создает scheduler с заданным интервалом.
func NewScheduler(interval time.Duration) *Scheduler {
	return &Scheduler{interval: interval}
}

// Add добавляет задачу в расписание.
func (s *Scheduler) Add(job Job) {
	s.jobs = append(s.jobs, job)
}

// Start запускает scheduler до отмены контекста.
func (s *Scheduler) Start(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	for {
		select {
		case <-ctx.Done():
			ticker.Stop()
			s.wg.Wait()
			return
		case <-ticker.C:
			for _, job := range s.jobs {
				job := job
				s.wg.Add(1)
				go func() {
					defer s.wg.Done()
					_ = job(ctx)
				}()
			}
		}
	}
}
