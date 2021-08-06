package define

import (
	"context"
	v1 "github.com/skiwer/trident-ci/api/pb/v1"
	"github.com/skiwer/trident-ci/processor/logger"
	"regexp"
	"time"
)

const (
	FlowTimeout     = 10 * time.Minute
	PipelineTimeout = 1 * time.Hour
)

const (
	// 构建id
	GlobalParamsBuildId = "CI_BUILD_ID"
	// 构建状态
	GlobalParamsPipelineStatus = "CI_BUILD_STATUS"
	// 构建失败原因
	GlobalParamsPipelineFailReason = "CI_BUILD_FAIL_REASON"

	// 构建成功
	BuildSuccess = "success"
	// 构建失败
	BuildFailed = "failed"
)

type ProcessCtx struct {
	Env map[string]string
}

var reg *regexp.Regexp

func init() {
	reg, _ = regexp.Compile(`\$\{([a-zA-Z][_a-zA-Z0-9]{0,50})\}`)
}

func getArgNameByPattern(pattern string) string {
	if len(pattern) <= 3 {
		return ""
	}
	return pattern[2 : len(pattern)-1]
}

func (p *ProcessCtx) AppendEnv(env map[string]string) {
	if env == nil {
		env = map[string]string{}
	}

	if p.Env == nil {
		p.Env = env
	}

	for k, v := range env {
		p.Env[k] = v
	}
}

func (p *ProcessCtx) PipelineFailed() bool {
	if s, ok := p.Env[GlobalParamsPipelineStatus]; ok && s == BuildFailed {
		return true
	}

	return false
}

func (p *ProcessCtx) PipelineSucceed() bool {
	if s, ok := p.Env[GlobalParamsPipelineStatus]; ok && s == BuildSuccess {
		return true
	}

	return false
}

func (p *ProcessCtx) GetFailReason() string {
	return p.Env[GlobalParamsPipelineFailReason]
}

func (p *ProcessCtx) Fail(failReason string) {
	p.Env[GlobalParamsPipelineStatus] = BuildFailed
	p.Env[GlobalParamsPipelineFailReason] = failReason
}

func (p *ProcessCtx) RenderByEnv(input string) (output string) {
	if input == "" {
		return
	}

	output = reg.ReplaceAllStringFunc(input, func(s string) string {
		k := getArgNameByPattern(s)

		if k == "" {
			return s
		}

		v, exist := p.Env[k]

		if !exist {
			return s
		}

		return v
	})

	return
}

type FlowRunner interface {
	Run(ctx context.Context, workDir string, flowCfg *v1.Flow, processCtx *ProcessCtx, logger *logger.Logger) error
}
