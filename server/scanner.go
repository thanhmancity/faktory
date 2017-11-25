package server

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/contribsys/faktory/storage"
	"github.com/contribsys/faktory/util"
)

type scannerTask func() (int64, error)

type scanner struct {
	name     string
	task     scannerTask
	set      storage.SortedSet
	jobs     int64
	cycles   int64
	walltime int64
}

func (s *scanner) Name() string {
	return s.name
}

func (s *scanner) Execute() error {
	start := time.Now()

	count, err := s.task()
	if err != nil {
		return err
	}

	if count > 0 {
		util.Infof("%s processed %d jobs", s.name, count)
	}

	end := time.Now()
	atomic.AddInt64(&s.cycles, 1)
	atomic.AddInt64(&s.jobs, count)
	atomic.AddInt64(&s.walltime, end.Sub(start).Nanoseconds())
	return nil
}

func (s *scanner) Stats() map[string]interface{} {
	return map[string]interface{}{
		"enqueued":      atomic.LoadInt64(&s.jobs),
		"cycles":        atomic.LoadInt64(&s.cycles),
		"size":          s.set.Size(),
		"wall_time_sec": (float64(atomic.LoadInt64(&s.walltime)) / 1000000000),
	}
}

func (s *Server) startScanners(waiter *sync.WaitGroup) {
	s.taskRunner.AddTask(5, &scanner{name: "Scheduled", set: s.store.Scheduled(), task: s.manager.EnqueueScheduledJobs})
	s.taskRunner.AddTask(5, &scanner{name: "Retries", set: s.store.Retries(), task: s.manager.RetryJobs})
	s.taskRunner.AddTask(60, &scanner{name: "Dead", set: s.store.Dead(), task: s.manager.Purge})
}
