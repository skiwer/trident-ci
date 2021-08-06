package consumer

import (
	"context"
	"github.com/skiwer/trident-ci/log"
	"github.com/skiwer/trident-ci/queue"
	"go.uber.org/zap"
	"sync"
)

type Processor interface {
	Run(ctx context.Context, msg *queue.Message) bool
}

type Consumer interface {
	Consume(ctx context.Context, q queue.Queue, p Processor)
}

type MultiWorkerConsumer struct {
	maxConcurrency int
}

func NewMultiWorkerConsumer(maxConcurrency int) *MultiWorkerConsumer {
	return &MultiWorkerConsumer{maxConcurrency: maxConcurrency}
}

func (c *MultiWorkerConsumer) Consume(ctx context.Context, q queue.Queue, p Processor) {
	wg := &sync.WaitGroup{}
	for i := 0; i < c.maxConcurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					msg, err := q.Pop()
					if err != nil {
						return
					} else {
						if retry := p.Run(ctx, msg); retry {
							if err := q.Push(msg); err != nil {
								log.GetLogger().Error("消息重入失败", zap.Error(err))
							}
						}
					}
				}
			}
		}()
	}
	wg.Wait()
}
