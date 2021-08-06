package queue

import (
	"fmt"
	"github.com/skiwer/trident-ci/config"
)

type Queue interface {
	Push(msg *Message) error
	Pop() (msg *Message, err error)
	Close()
}

type Type string

const (
	TypeChannel Type = "channel"
)

type Maker interface {
	config.FlagParser
	NewQueue() (Queue, error)
}

func NewQueueByType(tp Type) (q Queue, err error) {
	var maker Maker
	switch tp {
	case TypeChannel:
		maker = &ChannelQueueConfig{}
	default:
		return nil, fmt.Errorf("未知的队列类型: %s", tp)
	}

	if err = maker.Parse(); err != nil {
		return
	}

	return maker.NewQueue()
}
