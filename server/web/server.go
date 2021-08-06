package web

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/skiwer/trident-ci/processor"
	"github.com/skiwer/trident-ci/queue"
	"github.com/skiwer/trident-ci/server/web/routers"
	"net/http"
	"time"
)

type Server struct {
	processor *processor.PipeLineProcessor
	queue     queue.Queue
}

func NewServer(p *processor.PipeLineProcessor, queue queue.Queue) *Server {
	return &Server{processor: p, queue: queue}
}

func (s *Server) getRouter(p *processor.PipeLineProcessor, queue queue.Queue) http.Handler {
	r := gin.Default()

	routers.InitRouters(r, p, queue)

	return r
}

func (s *Server) Start(ctx context.Context, port int) (err error) {
	httpSvr := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: s.getRouter(s.processor, s.queue),
	}
	go func() {
		select {
		case <-ctx.Done():
			shutDownCtx, _ := context.WithTimeout(context.Background(), 2*time.Second)
			httpSvr.Shutdown(shutDownCtx)
		}
	}()

	if err := httpSvr.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}

	return nil
}
