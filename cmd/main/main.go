package main

import (
	"context"
	"flag"
	"github.com/docker/docker/client"
	v1 "github.com/skiwer/trident-ci/api/pb/v1"
	"github.com/skiwer/trident-ci/config"
	"github.com/skiwer/trident-ci/consumer"
	"github.com/skiwer/trident-ci/log"
	"github.com/skiwer/trident-ci/processor"
	"github.com/skiwer/trident-ci/processor/define"
	"github.com/skiwer/trident-ci/processor/docker_build"
	"github.com/skiwer/trident-ci/processor/lua"
	"github.com/skiwer/trident-ci/processor/scm"
	"github.com/skiwer/trident-ci/processor/shell"
	"github.com/skiwer/trident-ci/queue"
	rpc "github.com/skiwer/trident-ci/server/grpc"
	"github.com/skiwer/trident-ci/server/web"
	"go.uber.org/zap"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

func main() {
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wg := &sync.WaitGroup{}

	cfg := &config.Config{}
	err := cfg.Parse()

	if err != nil {
		panic(err)
	}

	err = log.InitLogger(cfg.Env)

	if err != nil {
		panic(err)
	}

	q, err := queue.NewQueueByType(queue.Type(cfg.QueueType))

	if err != nil {
		panic(err)
	}

	csm := consumer.NewMultiWorkerConsumer(cfg.MaxConcurrencyOfConsumer)

	dockerCli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}

	flowRunnerMp := map[v1.FlowType]define.FlowRunner{
		v1.FlowType_SCM:         scm.NewScmRunner(),
		v1.FlowType_Shell:       shell.NewShellRunner(dockerCli),
		v1.FlowType_DockerBuild: docker_build.NewDockerBuildRunner(dockerCli),
		v1.FlowType_Lua:         lua.NewLuaRunner(lua.NewLuaPool(cfg.MaxConcurrencyOfConsumer)),
	}

	pipelineProcessor := processor.NewPipelineProcessor(ctx, cfg.WorkDir, flowRunnerMp)

	wg.Add(1)
	go func() {
		defer wg.Done()
		csm.Consume(ctx, q, pipelineProcessor)
	}()

	grpcServer := rpc.NewServer(pipelineProcessor, q)

	wg.Add(1)
	go func() {
		defer wg.Done()

		if err := grpcServer.Start(ctx, cfg.RpcPort); err != nil {
			panic(err)
		}
	}()

	webServer := web.NewServer(pipelineProcessor, q)

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := webServer.Start(ctx, cfg.HttpPort); err != nil {
			panic(err)
		}
	}()

	c := make(chan os.Signal)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	stopSig := <-c

	log.GetLogger().Info("收到信号，开始退出程序", zap.String("signal", stopSig.String()))

	cancel()

	q.Close()

	wg.Wait()
}
