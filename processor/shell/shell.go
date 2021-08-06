package shell

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	v1 "github.com/skiwer/trident-ci/api/pb/v1"
	"github.com/skiwer/trident-ci/log"
	"github.com/skiwer/trident-ci/processor/define"
	"github.com/skiwer/trident-ci/processor/logger"
	"github.com/skiwer/trident-ci/processor/utils"
	"go.uber.org/zap"
	"io"
	"io/ioutil"
	"os"
	"strings"
)

const WorkDirInContainer = "/code"

type Runner struct {
	dockerClient *client.Client
}

func NewShellRunner(dockerClient *client.Client) *Runner {
	return &Runner{
		dockerClient: dockerClient,
	}
}

func (r *Runner) wrapShellScript(script string) string {
	template := `#!/bin/bash
%s
`
	ret := fmt.Sprintf(template, script)

	return ret
}

//func (r *Runner) readReturnEnvFromFile(file string) map[string]string {
//	bytes, err := ioutil.ReadFile(file)
//
//	env := map[string]string{}
//
//	if err != nil {
//		return env
//	}
//
//	lines := strings.Split(string(bytes), "\n")
//
//	for _, line := range lines {
//		line = strings.Trim(line, " ")
//		if line == "" {
//			continue
//		}
//		envItem := strings.SplitN(line, "=", 2)
//		if len(envItem) <= 1 {
//			continue
//		}
//		env[envItem[0]] = envItem[1]
//	}
//
//	return env
//}

func (r *Runner) imageExistsLocal(ctx context.Context, dockerImage string) bool {
	strList := strings.Split(dockerImage, "/")

	endStr := strList[len(strList)-1]

	// no tag
	if !strings.Contains(endStr, ":") {
		dockerImage += ":latest"
	}

	args := filters.NewArgs()
	args.Add("reference", dockerImage)
	options := types.ImageListOptions{
		All:     false,
		Filters: args,
	}
	images, err := r.dockerClient.ImageList(ctx, options)

	if err != nil {
		log.GetLogger().Error("docker image list error", zap.Error(err))
		return false
	}

	if len(images) > 0 {
		return true
	}

	return false
}

func (r *Runner) pullImage(ctx context.Context, imagePullPolicy v1.ImagePullPolicy, dockerImage string, logger *logger.Logger) error {
	pull := false

	switch imagePullPolicy {
	case v1.ImagePullPolicy_Never:
		pull = false
	case v1.ImagePullPolicy_Always:
		pull = true
	case v1.ImagePullPolicy_IfNotPresent:
		if !r.imageExistsLocal(ctx, dockerImage) {
			pull = true
		}
	}

	if !pull {
		return nil
	}

	logger.Info("开始拉取镜像...", zap.String("image", dockerImage))

	pullLog, err := r.dockerClient.ImagePull(ctx, dockerImage, types.ImagePullOptions{
		All:           false,
		RegistryAuth:  "",
		PrivilegeFunc: nil,
		Platform:      "",
	})

	if err != nil {
		return errors.Wrapf(err, "docker 镜像[%s]拉取失败", dockerImage)
	}

	_, err = io.Copy(logger, pullLog)

	pullLog.Close()

	if err != nil {
		return errors.Wrapf(err, "docker 拉取镜像日志io copy失败")
	}

	return nil
}

func (r *Runner) renderCfg(flowCfg *v1.Flow, processCtx *define.ProcessCtx) {
	if flowCfg.NoEnvRender {
		return
	}

	flowCfg.ShellCfg.Cmd = processCtx.RenderByEnv(flowCfg.ShellCfg.Cmd)
	flowCfg.ShellCfg.DockerImage = processCtx.RenderByEnv(flowCfg.ShellCfg.DockerImage)
}

func (r *Runner) Run(ctx context.Context, workDir string, flowCfg *v1.Flow, processCtx *define.ProcessCtx, logger *logger.Logger) error {
	uid := uuid.NewString()
	shellFileName := fmt.Sprintf("shell-%s.sh", uid)
	shellFileAbsolutePath := fmt.Sprintf("%s/%s", workDir, shellFileName)
	shellFilePathInContainer := fmt.Sprintf("%s/%s", WorkDirInContainer, shellFileName)

	r.renderCfg(flowCfg, processCtx)

	script := r.wrapShellScript(flowCfg.ShellCfg.Cmd)

	log.GetLogger().Info("shell script", zap.String("script", script))

	err := ioutil.WriteFile(shellFileAbsolutePath, []byte(script), os.ModePerm)

	if err != nil {
		return errors.Wrap(err, "shell脚本写入文件失败")
	}

	defer os.Remove(shellFileAbsolutePath)

	if !flowCfg.ShellCfg.WithDocker {

	}

	dockerImage := flowCfg.ShellCfg.DockerImage

	err = r.pullImage(ctx, flowCfg.ShellCfg.ImagePullPolicy, dockerImage, logger)

	if err != nil {
		return err
	}

	env := utils.ConvertEnvMp2StrSlice(processCtx.Env)

	containerCfg := &container.Config{
		Hostname:        "",
		Domainname:      "",
		User:            "",
		AttachStdin:     false,
		AttachStdout:    false,
		AttachStderr:    false,
		ExposedPorts:    nil,
		OpenStdin:       false,
		StdinOnce:       false,
		Tty:             true,
		Env:             env,
		Cmd:             nil,
		Healthcheck:     nil,
		ArgsEscaped:     false,
		Image:           dockerImage,
		WorkingDir:      WorkDirInContainer,
		Entrypoint:      []string{"sh", shellFilePathInContainer},
		NetworkDisabled: false,
		MacAddress:      "",
		OnBuild:         nil,
		Labels:          nil,
		StopSignal:      "",
		StopTimeout:     nil,
		Shell:           nil,
	}

	hostConfig := &container.HostConfig{
		Binds:           nil,
		ContainerIDFile: "",
		LogConfig:       container.LogConfig{},
		NetworkMode:     "",
		PortBindings:    nil,
		RestartPolicy:   container.RestartPolicy{},
		AutoRemove:      false,
		VolumeDriver:    "",
		VolumesFrom:     nil,
		CapAdd:          nil,
		CapDrop:         nil,
		CgroupnsMode:    "",
		DNS:             nil,
		DNSOptions:      nil,
		DNSSearch:       nil,
		ExtraHosts:      nil,
		GroupAdd:        nil,
		IpcMode:         "",
		Cgroup:          "",
		Links:           nil,
		OomScoreAdj:     0,
		PidMode:         "",
		Privileged:      false,
		PublishAllPorts: false,
		ReadonlyRootfs:  false,
		SecurityOpt:     nil,
		StorageOpt:      nil,
		Tmpfs:           nil,
		UTSMode:         "",
		UsernsMode:      "",
		ShmSize:         0,
		Sysctls:         nil,
		Runtime:         "",
		ConsoleSize:     [2]uint{},
		Isolation:       "",
		Resources:       container.Resources{},
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeBind,
				Source: workDir,
				Target: WorkDirInContainer,
			},
		},
		MaskedPaths:   nil,
		ReadonlyPaths: nil,
		Init:          nil,
	}

	logger.Info("开始创建容器...")

	resp, err := r.dockerClient.ContainerCreate(ctx, containerCfg, hostConfig, nil, nil, "")

	if err != nil {
		return errors.Wrap(err, "容器创建失败")
	}

	logger.Info("开始启动容器...")

	err = r.dockerClient.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{
		CheckpointID:  "",
		CheckpointDir: "",
	})

	if err != nil {
		return errors.Wrap(err, "容器启动失败")
	}

	defer r.dockerClient.ContainerRemove(ctx, resp.ID, types.ContainerRemoveOptions{Force: true})

	logOut, err := r.dockerClient.ContainerLogs(ctx, resp.ID, types.ContainerLogsOptions{
		ShowStdout: true,
		Follow:     true,
		Timestamps: true,
	})

	if err != nil {
		return errors.Wrap(err, "获取docker日志出错")
	}

	_, err = io.Copy(logger, logOut)

	logOut.Close()

	if err != nil {
		return errors.Wrapf(err, "docker 容器运行日志io copy失败")
	}

	logger.Info("等待容器运行完毕...")

	//defer func() {
	//	shellReturnEnv := r.readReturnEnvFromFile(shellEnvFileAbsolutePath)
	//	processCtx.AppendEnv(shellReturnEnv)
	//}()

	statusCh, errCh := r.dockerClient.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case <-ctx.Done():
		return fmt.Errorf("shell脚本执行超时")
	case err := <-errCh:
		if err != nil {
			return errors.Wrap(err, "shell脚本执行出错")
		}
	case status := <-statusCh:
		log.GetLogger().Info("ContainerWait status", zap.Any("status", status))

		if status.StatusCode != 0 {
			return fmt.Errorf("shell脚本执行exit code = %d", status.StatusCode)
		}
		if status.Error != nil {
			return fmt.Errorf("等待docker 容器运行完成出错: %s", status.Error.Message)
		}
	}

	return nil
}
