package http

import (
	"bufio"
	"errors"
	"runtime"
	"sync/atomic"
)

type WorkerPool struct {
	Pool    [WorkerPoolSize]RequestCtx
	Ready   RingBuffer[*RequestCtx]
	Handler Handler
}

func NewWorkerPool(handler Handler) WorkerPool {
	wp := WorkerPool{}
	wp.Handler = handler
	wp.Ready = NewRingBuffer[*RequestCtx]()
	for i := range wp.Pool {
		wp.Pool[i].ConnReader = bufio.NewReaderSize(nil, DefaultReadBufferSize)
		wp.Pool[i].ConnWriter = bufio.NewWriterSize(nil, DefaultWriteBufferSize)
		wp.Ready.Enqueue(&wp.Pool[i])
	}
	return wp
}

var (
	ErrFull  = errors.New("ring buffer is full")
	ErrEmpty = errors.New("ring buffer is empty")
)

type RingBuffer[T any] struct {
	buffer [WorkerPoolSize]slot[T]
	mask   uint64
	enqPos uint64
	deqPos uint64
}

type slot[T any] struct {
	sequence uint64
	value    T
}

// NewRingBuffer creates a new ring buffer of size N (must be power of 2)
func NewRingBuffer[T any]() RingBuffer[T] {
	var buf [WorkerPoolSize]slot[T]
	for i := range buf {
		buf[i].sequence = uint64(i)
	}
	return RingBuffer[T]{
		buffer: buf,
		mask:   WorkerPoolSize - 1,
	}
}

// Enqueue adds an item to the ring buffer
func (q *RingBuffer[T]) Enqueue(val T) error {
	for {
		pos := atomic.LoadUint64(&q.enqPos)
		slot := &q.buffer[pos&q.mask]

		seq := atomic.LoadUint64(&slot.sequence)
		delta := int64(seq) - int64(pos)

		if delta == 0 {
			if atomic.CompareAndSwapUint64(&q.enqPos, pos, pos+1) {
				slot.value = val
				atomic.StoreUint64(&slot.sequence, pos+1)
				return nil
			}
		} else if delta < 0 {
			return ErrFull
		} else {
			runtime.Gosched()
		}
	}
}

// Dequeue removes and returns the oldest item
func (q *RingBuffer[T]) Dequeue() (T, error) {
	var zero T
	for {
		pos := atomic.LoadUint64(&q.deqPos)
		slot := &q.buffer[pos&q.mask]

		seq := atomic.LoadUint64(&slot.sequence)
		delta := int64(seq) - int64(pos+1)

		if delta == 0 {
			if atomic.CompareAndSwapUint64(&q.deqPos, pos, pos+1) {
				val := slot.value
				atomic.StoreUint64(&slot.sequence, pos+q.mask+1)
				return val, nil
			}
		} else if delta < 0 {
			return zero, ErrEmpty
		} else {
			runtime.Gosched()
		}
	}
}
