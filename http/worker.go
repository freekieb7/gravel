package http

import (
	"bufio"
	"errors"
	"runtime"
	"sync/atomic"
)

type WorkerPool struct {
	Ready   RingBuffer[*RequestCtx]
	Size    uint64
	Start   uint64
	End     uint64
	Handler Handler
}

func NewWorkerPool(handler Handler, size uint64) WorkerPool {
	wp := WorkerPool{}
	wp.Handler = handler
	wp.Size = size
	wp.Ready = NewRingBuffer[*RequestCtx](size)

	for range size {
		reqCtx := RequestCtx{
			ConnReader: bufio.NewReaderSize(nil, MaxRequestSize),
			ConnWriter: bufio.NewWriterSize(nil, MaxResponseSize),
		}
		wp.Ready.Enqueue(&reqCtx)
	}

	return wp
}

var (
	ErrFull  = errors.New("ring buffer is full")
	ErrEmpty = errors.New("ring buffer is empty")
)

type slot[T any] struct {
	sequence uint64
	value    T
}

type RingBuffer[T any] struct {
	buffer []slot[T]
	mask   uint64
	_      [8]uint64 // padding to avoid false sharing
	enqPos uint64
	_      [8]uint64
	deqPos uint64
}

// NewRingBuffer creates a new ring buffer of size N (must be power of 2)
func NewRingBuffer[T any](size uint64) RingBuffer[T] {
	if size == 0 || (size&(size-1)) != 0 {
		panic("size must be a power of 2")
	}

	buf := make([]slot[T], size)
	for i := range buf {
		buf[i].sequence = uint64(i)
	}

	return RingBuffer[T]{
		buffer: buf,
		mask:   size - 1,
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
