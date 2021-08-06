package config

import "flag"

type Config struct {
	Env                      string
	HttpPort                 int
	RpcPort                  int
	WorkDir                  string
	QueueType                string
	MaxConcurrencyOfConsumer int
}

type QueueConfig struct {
	Type string
	Cap  int64
}

type FlagParser interface {
	Parse() error
}

func (c *Config) Parse() error {
	flag.StringVar(&c.Env, "env", "dev", "工作模式：dev为开发，prod为生产")
	flag.IntVar(&c.HttpPort, "http-port", 80, "http服务监听端口")
	flag.IntVar(&c.RpcPort, "rpc-port", 81, "rpc服务监听端口")
	flag.StringVar(&c.WorkDir, "work-dir", "/tmp", "工作目录")
	flag.StringVar(&c.QueueType, "queue-type", "channel", "消息队列类型")
	flag.IntVar(&c.MaxConcurrencyOfConsumer, "max-concurrency-of-consumer", 5, "消费者最大并发处理任务数")

	return nil
}
