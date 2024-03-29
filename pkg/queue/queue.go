package queue

import (
	"github.com/enriquebris/goconcurrentqueue"
	"golang.org/x/sync/errgroup"
)

// QueueWorker allow us to run N concurrent jobs in order
type QueueWorker struct {
	queue   goconcurrentqueue.Queue
	workers int
}

// Closure is the function used for each item in the Queue to handle the logic
type Closure func(v interface{}) error

// New initiates a new FIFO queue
func New(capacity, workers int) *QueueWorker {
	q := goconcurrentqueue.NewFixedFIFO(capacity)
	// dont allow more workers than there are items in the queue
	if workers > q.GetCap() {
		workers = q.GetCap()
	}
	return &QueueWorker{
		queue:   q,
		workers: workers,
	}
}

// Push item to the queue
func (qw *QueueWorker) Push(v interface{}) error {
	return qw.queue.Enqueue(v)
}

// Work starts N coroutines based on QueueWorker.workers
// The queue will stop at first failure if any.
// The workers expect the queue to be populated before any work is started.
func (qw *QueueWorker) Work(closure Closure) error {
	g := new(errgroup.Group)
	for i := 0; i < qw.workers; i++ {
		g.Go(func() error {
			for qw.queue.GetLen() > 0 {
				// we will explicitly suppress any Dequeue errors
				// since our queue implementation will not support
				// concurrently push to the queue while we are working it.
				value, _ := qw.queue.Dequeue()
				if err := closure(value); err != nil {
					return err
				}
			}
			return nil
		})
	}
	return g.Wait()
}
