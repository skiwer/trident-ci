package processor

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	v1 "github.com/skiwer/trident-ci/api/pb/v1"
	"github.com/skiwer/trident-ci/log"
	"github.com/skiwer/trident-ci/processor/define"
	"github.com/skiwer/trident-ci/processor/logger"
	"github.com/skiwer/trident-ci/processor/utils"
	"github.com/skiwer/trident-ci/queue"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	"sync"
	"time"
)

type PipeLineProcessor struct {
	runnerMp   map[v1.FlowType]define.FlowRunner
	rootPath   string
	ctx        context.Context
	progressMp *sync.Map
}

type PipelineRunEntity struct {
	Progress   *v1.PipelineProgress `json:"progress"`
	JobDir     string               `json:"jobDir"`
	CancelFunc context.CancelFunc
}

func NewPipelineProcessor(ctx context.Context, rootPath string, runnerMp map[v1.FlowType]define.FlowRunner) *PipeLineProcessor {
	return &PipeLineProcessor{
		ctx:        ctx,
		runnerMp:   runnerMp,
		rootPath:   rootPath,
		progressMp: &sync.Map{},
	}
}

func (p *PipeLineProcessor) getJobLogFile(dir string) string {
	return fmt.Sprintf("%s/data/job.log", dir)
}

func (p *PipeLineProcessor) InitPipeline(pl *v1.Pipeline) {
	runEntity := PipelineRunEntity{
		Progress: &v1.PipelineProgress{
			Pipeline:       pl,
			Status:         v1.Status_Created,
			CreateTime:     time.Now().UnixNano(),
			FlowProgresses: []*v1.FlowProgress{},
			Env:            nil,
		},
	}

	p.updatePipelineRunEntity(pl.Uid, runEntity)
}

func (p *PipeLineProcessor) GetPipelineLog(pipelineId string) ([]byte, error) {
	pl, exists := p.progressMp.Load(pipelineId)

	if !exists {
		return nil, errors.New("流水线任务不存在")
	}

	entity, ok := pl.(PipelineRunEntity)

	if !ok {
		return nil, errors.New("流水线任务数据转换失败")
	}

	jobLogFile := p.getJobLogFile(entity.JobDir)

	if !utils.FileExists(jobLogFile) {
		return nil, nil
	}

	return os.ReadFile(jobLogFile)
}

func (p *PipeLineProcessor) GetPipelineProgress(pipelineId string) (progress *v1.PipelineProgress, err error) {
	pl, exists := p.progressMp.Load(pipelineId)

	if !exists {
		return nil, errors.New("流水线任务不存在")
	}

	entity, ok := pl.(PipelineRunEntity)

	if !ok {
		return nil, errors.New("流水线任务数据转换失败")
	}

	return entity.Progress, nil
}

func (p *PipeLineProcessor) StopPipeline(pipelineId string) error {
	pl, exists := p.progressMp.Load(pipelineId)

	if !exists {
		return errors.New("流水线任务不存在")
	}

	entity, ok := pl.(PipelineRunEntity)

	if !ok {
		return errors.New("流水线任务数据转换失败")
	}

	entity.CancelFunc()

	return nil
}

func (p *PipeLineProcessor) DeletePipeline(pipelineId string) error {
	pl, exists := p.progressMp.Load(pipelineId)

	if !exists {
		return errors.New("流水线任务不存在")
	}

	defer p.progressMp.Delete(pipelineId)

	entity, ok := pl.(PipelineRunEntity)

	if !ok {
		return errors.New("流水线任务数据转换失败")
	}

	defer os.RemoveAll(entity.JobDir)

	entity.CancelFunc()

	return nil
}

func (p *PipeLineProcessor) updatePipelineRunEntity(pipelineId string, entity PipelineRunEntity) {
	p.progressMp.Store(pipelineId, entity)
}

func (p *PipeLineProcessor) Run(ctx context.Context, msg *queue.Message) bool {
	job, ok := msg.Data.(*v1.Pipeline)

	if !ok {
		log.GetLogger().Warn("msg type error, can not convert to *v1.Pipeline",
			zap.Any("msg", msg.Data),
			zap.String("msgId", msg.ID))
		return true
	}

	if job.Uid == "" {
		log.GetLogger().Warn("pipeline uid 不能为空",
			zap.Any("pipeline", job),
			zap.String("msgId", msg.ID))
		return false
	}

	jobRootDir := fmt.Sprintf("%s/job-%s", p.rootPath, job.Uid)

	jobWorkDir := fmt.Sprintf("%s/workspace", jobRootDir)
	jobDataDir := fmt.Sprintf("%s/data", jobRootDir)
	logFilePath := p.getJobLogFile(jobRootDir)

	jobCtx, jobCancel := context.WithTimeout(ctx, define.PipelineTimeout)
	defer jobCancel()

	processCtx := &define.ProcessCtx{Env: make(map[string]string)}
	processCtx.AppendEnv(map[string]string{
		define.GlobalParamsBuildId: job.Uid,
	})
	processCtx.AppendEnv(job.Params)

	runEntity := PipelineRunEntity{
		Progress: &v1.PipelineProgress{
			Pipeline:       job,
			Status:         v1.Status_Started,
			StartTime:      time.Now().UnixNano(),
			FlowProgresses: []*v1.FlowProgress{},
			Env:            processCtx.Env,
		},
		JobDir:     jobRootDir,
		CancelFunc: jobCancel,
	}

	p.updatePipelineRunEntity(job.Uid, runEntity)

	if err := os.MkdirAll(jobWorkDir, 0755); err != nil {
		log.GetLogger().Error("创建流水线临时工作路径失败", zap.Error(err), zap.String("path", jobWorkDir))
		return false
	}

	defer os.RemoveAll(jobWorkDir)

	if err := os.MkdirAll(jobDataDir, 0755); err != nil {
		log.GetLogger().Error("创建流水线数据存储路径失败", zap.Error(err), zap.String("path", jobDataDir))
		return false
	}

	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	encoder := zapcore.NewConsoleEncoder(encoderCfg)

	logFile, err := os.Create(logFilePath)

	if err != nil {
		log.GetLogger().Error("创建流水线日志文件失败", zap.Error(err), zap.String("path", logFilePath))
		return false
	}

	allCores := []zapcore.Core{
		zapcore.NewCore(encoder, zapcore.AddSync(logFile), zapcore.DebugLevel),
		zapcore.NewCore(encoder, zapcore.Lock(os.Stdout), zapcore.DebugLevel),
	}

	jobLogger := logger.NewLogger(job.Uid, zap.New(zapcore.NewTee(allCores...)))

	jobLogger.Info("开始执行pipeline", zap.String("title", job.Title))

	defer jobLogger.Sync()

	runEntity.Progress.Status = v1.Status_Running
	p.updatePipelineRunEntity(job.Uid, runEntity)

	if len(job.Flows) <= 0 {
		return false
	}

	for idx, flow := range job.Flows {
		flowCtx, _ := context.WithTimeout(jobCtx, define.FlowTimeout)

		runEntity.Progress.FlowProgresses = append(runEntity.Progress.FlowProgresses, &v1.FlowProgress{
			Flow:      flow,
			Status:    v1.Status_Running,
			StartTime: time.Now().UnixNano(),
		})
		runEntity.Progress.Status = v1.Status_Running
		runEntity.Progress.CurRunningFlowId = flow.Uid

		p.updatePipelineRunEntity(job.Uid, runEntity)

		err := p.runFlow(flowCtx, idx, flow, jobWorkDir, processCtx, jobLogger)

		breakNow := false

		var flowError error

		if err != nil {
			flowError = err
		} else {
			if processCtx.PipelineFailed() {
				flowError = errors.New(processCtx.GetFailReason())
			} else if processCtx.PipelineSucceed() {
				breakNow = true
			}
		}

		if flowError != nil {
			if errors.Is(flowError, context.Canceled) {
				runEntity.Progress.FlowProgresses[idx].Status = v1.Status_Canceled
				runEntity.Progress.FlowProgresses[idx].FailReason = fmt.Sprintf("流程执行被取消: %s", flowError.Error())
			} else {
				runEntity.Progress.FlowProgresses[idx].Status = v1.Status_Failed
				runEntity.Progress.FlowProgresses[idx].FailReason = flowError.Error()
			}
			log.GetLogger().Error("流程执行失败", zap.Error(flowError), zap.Any("flow", flow), zap.Int("flowIndex", idx))
			breakNow = true
		} else {
			runEntity.Progress.FlowProgresses[idx].Status = v1.Status_Succeed
		}

		runEntity.Progress.Env = processCtx.Env
		runEntity.Progress.FlowProgresses[idx].FinishTime = time.Now().UnixNano()
		p.updatePipelineRunEntity(job.Uid, runEntity)

		if breakNow {
			break
		}
	}

	for idx, flowProgress := range runEntity.Progress.FlowProgresses {
		if flowProgress.Status == v1.Status_Failed {
			runEntity.Progress.Status = v1.Status_Failed
			runEntity.Progress.FailReason = fmt.Sprintf("流程[索引=%d]执行出错: %s", idx, flowProgress.FailReason)
			break
		} else if flowProgress.Status == v1.Status_Canceled {
			runEntity.Progress.Status = v1.Status_Canceled
			runEntity.Progress.FailReason = fmt.Sprintf("流程[索引=%d]执行被取消", idx)
			break
		}
	}

	if runEntity.Progress.Status != v1.Status_Canceled && runEntity.Progress.Status != v1.Status_Failed {
		runEntity.Progress.Status = v1.Status_Succeed
	}

	runEntity.Progress.FinishTime = time.Now().UnixNano()
	p.updatePipelineRunEntity(job.Uid, runEntity)

	jobLogger.Info("pipeline执行结束", zap.String("title", job.Title))

	return false
}

func (p *PipeLineProcessor) runFlow(ctx context.Context, flowIndex int, flow *v1.Flow, jobWorkDir string, processCtx *define.ProcessCtx, jobLogger *logger.Logger) (err error) {
	defer func() {
		jobLogger.Info(fmt.Sprintf("=========== pipeline流程[%d]执行结束 ===========", flowIndex))
		if e := recover(); e != nil {
			err = fmt.Errorf("runFlow internal error: %v", e)
		}
	}()

	runner, ok := p.runnerMp[flow.Type]
	if !ok {
		jobLogger.Warn("未知的pipeline流程",
			zap.String("flowType", flow.Type.String()),
		)
		return fmt.Errorf("未知的pipeline流程类型[%s]", flow.Type.String())
	}

	jobLogger.Info(fmt.Sprintf("=========== 开始执行pipeline流程[%d] ===========", flowIndex))

	jobLogger.Info("流程信息",
		zap.Int("index", flowIndex),
		zap.String("flowType", flow.Type.String()),
	)

	flowDir := jobWorkDir

	if err := runner.Run(ctx, flowDir, flow, processCtx, jobLogger); err != nil {
		jobLogger.Error("pipeline流程执行失败",
			zap.Int("index", flowIndex),
			zap.String("flowType", flow.Type.String()),
			zap.String("error", err.Error()),
		)
		return errors.Wrapf(err, "流程执行失败, 流程index=[%d], 流程type=[%s]", flowIndex, flow.Type.String())
	} else {
		jobLogger.Info("pipeline流程执行成功",
			zap.Int("index", flowIndex),
			zap.String("flowType", flow.Type.String()),
		)
	}

	return nil
}
