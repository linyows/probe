package probe

import (
	"sync"
	"time"
)

type Scheduler struct {
	jobs []Job
}

func (s *Scheduler) Do() {
	var wg sync.WaitGroup

	for _, job := range s.jobs {
		// No repeat
		if job.Repeat == nil {
			go func() {
				defer wg.Done()
				wg.Add(1)
			}()
			continue
		}

		// Repeat
		for i := 0; i < job.Repeat.Count; i++ {
			go func() {
				defer wg.Done()
				wg.Add(1)
			}()
			time.Sleep(time.Duration(job.Repeat.Interval) * time.Second)
		}
	}

	wg.Wait()
}
