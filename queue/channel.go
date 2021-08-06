package queue

import (
	"errors"
	"flag"
	"github.com/skiwer/trident-ci/log"
	"go.uber.org/zap"
)

type ChannelQueue struct {
	queueCh chan *Message
}

type ChannelQueueConfig struct {
	Cap int64
}

func (c *ChannelQueueConfig) Parse() error {
	flag.Int64Var(&c.Cap, "queue-cap", 1000, "队列容量")

	if c.Cap <= 0 {
		return errors.New("queue cap must > 0")
	}

	return nil
}

func (c *ChannelQueueConfig) NewQueue() (q Queue, err error) {
	q = NewChannelQueue(c.Cap)
	return
}

func NewChannelQueue(cap int64) *ChannelQueue {
	return &ChannelQueue{queueCh: make(chan *Message, cap)}
}

func (q *ChannelQueue) Push(msg *Message) (err error) {
	log.GetLogger().Info("[channel-queue]msg push", zap.Any("msg", msg))

	defer func() {
		if err != nil {
			log.GetLogger().Error("[channel-queue]push msg failed", zap.Error(err))
		}
	}()

	select {
	case q.queueCh <- msg:
	default:
		err = errors.New("队列已满")
	}

	return
}

func (q *ChannelQueue) Pop() (msg *Message, err error) {
	log.GetLogger().Info("[channel-queue]msg pop request")

	msg, open := <-q.queueCh

	defer func() {
		if err != nil {
			log.GetLogger().Warn("[channel-queue]pop msg failed", zap.Error(err))
		}
	}()

	if !open {
		err = errors.New("消息队列channel已关闭")
	}
	return
}

func (q *ChannelQueue) Close() {
	log.GetLogger().Info("[channel-queue]close")

	close(q.queueCh)
}
