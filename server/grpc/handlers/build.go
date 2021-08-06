package handlers

import (
	"context"
	"github.com/google/uuid"
	v1 "github.com/skiwer/trident-ci/api/pb/v1"
	"github.com/skiwer/trident-ci/processor"
	"github.com/skiwer/trident-ci/queue"
)

type BuildServer struct {
	processor *processor.PipeLineProcessor
	queue     queue.Queue
}

func NewBuildServer(p *processor.PipeLineProcessor, queue queue.Queue) *BuildServer {
	return &BuildServer{processor: p, queue: queue}
}

func (b *BuildServer) Build(ctx context.Context, in *v1.BuildRequest) (*v1.BuildResponse, error) {
	in.Pipeline.Uid = uuid.NewString()
	msg := &queue.Message{
		ID:   in.Pipeline.Uid,
		Data: in.Pipeline,
	}
	if err := b.queue.Push(msg); err != nil {
		return nil, err
	}

	return &v1.BuildResponse{BuildId: in.Pipeline.Uid}, nil
}

func (b *BuildServer) GetBuildResult(ctx context.Context, in *v1.GetBuildRequest) (*v1.BuildDetail, error) {
	progress, err := b.processor.GetPipelineProgress(in.BuildId)

	if err != nil {
		return nil, err
	}

	return &v1.BuildDetail{Progress: progress}, nil
}

func (b *BuildServer) DeleteBuild(ctx context.Context, in *v1.DeleteBuildRequest) (*v1.EmptyResponse, error) {
	err := b.processor.DeletePipeline(in.BuildId)
	if err != nil {
		return nil, err
	}

	return &v1.EmptyResponse{}, nil
}

func (b *BuildServer) StopBuild(ctx context.Context, in *v1.StopBuildRequest) (*v1.EmptyResponse, error) {
	err := b.processor.StopPipeline(in.BuildId)
	if err != nil {
		return nil, err
	}

	return &v1.EmptyResponse{}, nil
}
