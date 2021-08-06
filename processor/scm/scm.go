package scm

import (
	"context"
	"github.com/pkg/errors"
	v1 "github.com/skiwer/trident-ci/api/pb/v1"
	"github.com/skiwer/trident-ci/processor/define"
	"github.com/skiwer/trident-ci/processor/logger"
	"io"
)

type Runner struct {
	rootPath string
}

type Cloner interface {
	Clone(ctx context.Context, workDir string, cfg *v1.ScmCfg, logWriter io.Writer) error
}

func NewScmRunner() *Runner {
	return &Runner{}
}

func (r *Runner) renderCfg(flowCfg *v1.Flow, processCtx *define.ProcessCtx) {
	if flowCfg.NoEnvRender {
		return
	}

	flowCfg.ScmCfg.Address = processCtx.RenderByEnv(flowCfg.ScmCfg.Address)
	flowCfg.ScmCfg.Branch = processCtx.RenderByEnv(flowCfg.ScmCfg.Branch)

	if flowCfg.ScmCfg.Credit != nil {
		if flowCfg.ScmCfg.Credit.Type != v1.CreditType_NoCredit {
			flowCfg.ScmCfg.Credit.Username = processCtx.RenderByEnv(flowCfg.ScmCfg.Credit.Username)
			flowCfg.ScmCfg.Credit.Password = processCtx.RenderByEnv(flowCfg.ScmCfg.Credit.Password)
			flowCfg.ScmCfg.Credit.PrivateKey = processCtx.RenderByEnv(flowCfg.ScmCfg.Credit.PrivateKey)
		}
	}
}

func (r *Runner) Run(ctx context.Context, workDir string, flowCfg *v1.Flow, processCtx *define.ProcessCtx, logger *logger.Logger) (err error) {
	var c Cloner

	switch flowCfg.ScmCfg.VcsType {
	case v1.VCSType_Git:
		c = &GitScm{}
	default:
		return errors.Wrapf(err, "找不到该平台[%s]的代码拉取工具", flowCfg.ScmCfg.VcsType.String())
	}

	r.renderCfg(flowCfg, processCtx)

	err = c.Clone(ctx, workDir, flowCfg.ScmCfg, logger)

	if err != nil {
		return errors.Wrap(err, "代码拉取失败")
	}

	return nil
}
