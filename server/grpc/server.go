package grpc

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	v1 "github.com/skiwer/trident-ci/api/pb/v1"
	"github.com/skiwer/trident-ci/log"
	"github.com/skiwer/trident-ci/processor"
	"github.com/skiwer/trident-ci/queue"
	"github.com/skiwer/trident-ci/server/grpc/handlers"
	"google.golang.org/grpc"
	"net"
)

type Server struct {
	processor *processor.PipeLineProcessor
	queue     queue.Queue
	rpcSvr    *grpc.Server
}

func NewServer(p *processor.PipeLineProcessor, queue queue.Queue) *Server {
	return &Server{processor: p, rpcSvr: grpc.NewServer(), queue: queue}
}

func (s *Server) Start(ctx context.Context, port int) (err error) {

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))

	if err != nil {
		return errors.Wrapf(err, "failed to listen given port: %d", port)
	}

	v1.RegisterBuildServer(s.rpcSvr, handlers.NewBuildServer(s.processor, s.queue))

	go func() {
		select {
		case <-ctx.Done():
			log.GetLogger().Info("ctx.Done触发，开始停止grpc server")
			s.rpcSvr.GracefulStop()
			log.GetLogger().Info("grpc server已停止")
		}
	}()

	if err := s.rpcSvr.Serve(listener); err != nil && err != grpc.ErrServerStopped {
		return err
	}

	return nil
}
