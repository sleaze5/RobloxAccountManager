package appservice

import "sync"

// forEachIndexConcurrently calls work once for each index from 0 to count-1,
// with at most workerLimit calls running at once, and returns when every call
// has returned.
func forEachIndexConcurrently(count, workerLimit int, work func(index int)) {
	jobs := make(chan int, count)
	for index := range count {
		jobs <- index
	}
	close(jobs)
	var workers sync.WaitGroup
	for range min(workerLimit, count) {
		workers.Go(func() {
			for index := range jobs {
				work(index)
			}
		})
	}
	workers.Wait()
}
