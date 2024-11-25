package fc_fusion

import "sync"

// FIFOQueue defines a circular queue
type FIFOQueue struct {
	sync.Mutex
	data     []*fusionRequest
	capacity int
	head     int
	tail     int
	size     int
}

type queue interface {
	Enqueue(r *fusionRequest) bool
	Dequeue() *fusionRequest
	Front() *fusionRequest
	Len() int
	Lock()
	Unlock()
}

// NewFIFOQueue creates a queue
func NewFIFOQueue(n int) *FIFOQueue {
	if n < 1 {
		return nil
	}
	return &FIFOQueue{
		data:     make([]*fusionRequest, n),
		capacity: n,
		head:     0,
		tail:     0,
		size:     0,
	}
}

// IsEmpty returns true if queue is empty
func (q *FIFOQueue) IsEmpty() bool {
	return q != nil && q.size == 0
}

// IsFull returns true if queue is full
func (q *FIFOQueue) IsFull() bool {
	return q.size == q.capacity
}

// Enqueue pushes an element to the back
func (q *FIFOQueue) Enqueue(v *fusionRequest) bool {
	if q.IsFull() {
		return false
	}

	q.data[q.tail] = v
	q.tail = (q.tail + 1) % q.capacity
	q.size = q.size + 1
	return true
}

// Dequeue fetches a element from queue
func (q *FIFOQueue) Dequeue() *fusionRequest {
	if q.IsEmpty() {
		return nil
	}
	v := q.data[q.head]
	q.head = (q.head + 1) % q.capacity
	q.size = q.size - 1
	return v
}

func (q *FIFOQueue) Front() *fusionRequest {
	if q.IsEmpty() {
		return nil
	}
	v := q.data[q.head]
	return v
}

// Len returns the current length of the queue
func (q *FIFOQueue) Len() int {
	return q.size
}
