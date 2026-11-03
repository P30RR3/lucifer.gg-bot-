package cerebras

import (
	"context"
	"sync"
	"time"
)

type BatchItem struct {
	Input    ClassifyInput
	ResultCh chan *Classification
}

type Batcher struct {
	client     *Client
	batchSize  int
	flushEvery time.Duration

	mu    sync.Mutex
	queue []*BatchItem
	timer *time.Timer

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	processed atomic64
}

type atomic64 struct {
	mu sync.Mutex
	v  int64
}

func (a *atomic64) Add(d int64) int64 {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.v += d
	return a.v
}

func (a *atomic64) Load() int64 {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.v
}

func NewBatcher(c *Client, batchSize int, flushEvery time.Duration) *Batcher {
	if batchSize <= 0 {
		batchSize = 8
	}
	if batchSize > 64 {
		batchSize = 64
	}
	if flushEvery <= 0 {
		flushEvery = 2 * time.Second
	}
	ctx, cancel := context.WithCancel(context.Background())
	b := &Batcher{
		client:     c,
		batchSize:  batchSize,
		flushEvery: flushEvery,
		ctx:        ctx,
		cancel:     cancel,
	}
	b.wg.Add(1)
	go b.loop()
	return b
}

func (b *Batcher) Submit(in ClassifyInput) <-chan *Classification {
	item := &BatchItem{Input: in, ResultCh: make(chan *Classification, 1)}

	b.mu.Lock()
	b.queue = append(b.queue, item)
	shouldFlush := len(b.queue) >= b.batchSize
	if b.timer == nil {
		b.timer = time.AfterFunc(b.flushEvery, b.flushSafe)
	}
	b.mu.Unlock()

	if shouldFlush {
		go b.flushSafe()
	}
	return item.ResultCh
}

func (b *Batcher) SubmitSync(ctx context.Context, in ClassifyInput) *Classification {
	ch := b.Submit(in)
	select {
	case <-ctx.Done():
		return FallbackError(ctx.Err())
	case res := <-ch:
		if res == nil {
			return FallbackSafe()
		}
		return res
	}
}

func (b *Batcher) loop() {
	defer b.wg.Done()
	<-b.ctx.Done()
	b.flushSafe()
}

func (b *Batcher) flushSafe() {
	b.mu.Lock()
	items := b.queue
	b.queue = nil
	if b.timer != nil {
		b.timer.Stop()
		b.timer = nil
	}
	b.mu.Unlock()

	if len(items) == 0 {
		return
	}

	var wg sync.WaitGroup
	sem := make(chan struct{}, 4)
	for _, it := range items {
		it := it
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			res := b.client.ClassifyMessage(b.ctx, it.Input)
			it.ResultCh <- res
			close(it.ResultCh)
		}()
	}
	wg.Wait()
	b.processed.Add(int64(len(items)))
}

func (b *Batcher) Stats() (processed int64, pending int) {
	b.mu.Lock()
	pending = len(b.queue)
	b.mu.Unlock()
	return b.processed.Load(), pending
}

func (b *Batcher) Flush() {
	b.flushSafe()
}

func (b *Batcher) Close() {
	b.cancel()
	b.wg.Wait()
	b.flushSafe()
}
