package docker_build

import (
	"context"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"
	v1 "github.com/skiwer/trident-ci/api/pb/v1"
	"github.com/skiwer/trident-ci/processor/build_context"
	"github.com/skiwer/trident-ci/processor/define"
	"github.com/skiwer/trident-ci/processor/logger"
	"io"
)

type Runner struct {
	dockerClient *client.Client
}

func NewDockerBuildRunner(dockerClient *client.Client) *Runner {
	return &Runner{
		dockerClient: dockerClient,
	}
}

func (r *Runner) renderCfg(flowCfg *v1.Flow, processCtx *define.ProcessCtx) {
	if flowCfg.NoEnvRender {
		return
	}

	flowCfg.DockerBuildCfg.BaseImage = processCtx.RenderByEnv(flowCfg.DockerBuildCfg.BaseImage)
	flowCfg.DockerBuildCfg.TargetImage = processCtx.RenderByEnv(flowCfg.DockerBuildCfg.TargetImage)
	flowCfg.DockerBuildCfg.Dockerfile = processCtx.RenderByEnv(flowCfg.DockerBuildCfg.Dockerfile)
}

func (r *Runner) Run(ctx context.Context, workDir string, flowCfg *v1.Flow, processCtx *define.ProcessCtx, logger *logger.Logger) (err error) {
	r.renderCfg(flowCfg, processCtx)

	dockerfileContent := flowCfg.DockerBuildCfg.Dockerfile

	source, err := build_context.New(workDir, build_context.WithDockerfile(dockerfileContent))

	if err != nil {
		return errors.Wrapf(err, "docker build context创建失败")
	}

	defer source.Close()

	buildCtxReader, err := source.AsTarReader()

	if err != nil {
		return errors.Wrapf(err, "docker build tar reader创建失败")
	}

	resp, err := r.dockerClient.ImageBuild(ctx, buildCtxReader, types.ImageBuildOptions{
		Tags:           []string{flowCfg.DockerBuildCfg.TargetImage},
		SuppressOutput: false,
		RemoteContext:  "",
		NoCache:        false,
		Remove:         true,
		ForceRemove:    true,
		PullParent:     false,
		Isolation:      "",
		CPUSetCPUs:     "",
		CPUSetMems:     "",
		CPUShares:      0,
		CPUQuota:       0,
		CPUPeriod:      0,
		Memory:         0,
		MemorySwap:     0,
		CgroupParent:   "",
		NetworkMode:    "",
		ShmSize:        0,
		Dockerfile:     "",
		Ulimits:        nil,
		BuildArgs:      nil,
		AuthConfigs:    nil,
		Context:        nil,
		Labels:         nil,
		Squash:         false,
		CacheFrom:      nil,
		SecurityOpt:    nil,
		ExtraHosts:     nil,
		Target:         "",
		SessionID:      "",
		Platform:       "",
		Version:        "",
		BuildID:        "",
		Outputs:        nil,
	})

	if err != nil {
		return errors.Wrapf(err, "docker构建请求出错")
	}

	_, err = io.Copy(logger, resp.Body)

	resp.Body.Close()

	if err != nil {
		return errors.Wrapf(err, "docker build日志io copy失败")
	}

	if !flowCfg.DockerBuildCfg.PushAfterBuild {
		return nil
	}

	_, err = r.dockerClient.ImagePush(ctx, flowCfg.DockerBuildCfg.TargetImage, types.ImagePushOptions{
		All:           false,
		RegistryAuth:  "",
		PrivilegeFunc: nil,
		Platform:      "",
	})

	if err != nil {
		return errors.Wrapf(err, "docker image push 出错")
	}

	return nil
}
