package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/skiwer/trident-ci/processor"
	"github.com/skiwer/trident-ci/queue"
	"github.com/skiwer/trident-ci/server/web/models"
	"github.com/skiwer/trident-ci/server/web/utils"
	"net/http"
)

type BuildHandler struct {
	processor *processor.PipeLineProcessor
	queue     queue.Queue
}

func NewBuildHandler(p *processor.PipeLineProcessor, queue queue.Queue) *BuildHandler {
	return &BuildHandler{
		processor: p,
		queue:     queue,
	}
}

func (h *BuildHandler) Build(c *gin.Context) {
	p := new(models.PipelineBuildParams)

	if !p.Validate(c) {
		return
	}

	id := uuid.NewString()
	p.Pipeline.Uid = id

	msg := &queue.Message{
		ID:   id,
		Data: p.Pipeline,
	}

	if err := h.queue.Push(msg); err != nil {
		c.JSON(http.StatusInternalServerError, utils.BuildResp(err.Error(), utils.PipelineBuildJobCreateFailed, nil))
		return
	}

	h.processor.InitPipeline(p.Pipeline)

	c.JSON(http.StatusOK, utils.BuildResp("流水线任务创建成功", utils.Success, id))
}

func (h *BuildHandler) GetBuildLog(c *gin.Context) {
	p := new(models.PipelineBuildIdBind)

	if !p.Validate(c) {
		return
	}

	logBytes, err := h.processor.GetPipelineLog(p.Id)

	if err != nil {
		c.JSON(http.StatusInternalServerError, utils.BuildResp(err.Error(), utils.PipelineBuildJobLogGetFailed, nil))
		return
	}

	c.JSON(http.StatusOK, utils.BuildResp("流水线任务执行日志查询成功", utils.Success, string(logBytes)))
}

func (h *BuildHandler) GetBuildProgress(c *gin.Context) {
	p := new(models.PipelineBuildIdBind)

	if !p.Validate(c) {
		return
	}

	progress, err := h.processor.GetPipelineProgress(p.Id)

	if err != nil {
		c.JSON(http.StatusInternalServerError, utils.BuildResp(err.Error(), utils.PipelineBuildJobProgressGetFailed, nil))
		return
	}

	c.JSON(http.StatusOK, utils.BuildResp("流水线任务执行进度查询成功", utils.Success, progress))
}

func (h *BuildHandler) StopBuild(c *gin.Context) {
	p := new(models.PipelineBuildIdBind)

	if !p.Validate(c) {
		return
	}

	err := h.processor.StopPipeline(p.Id)

	if err != nil {
		c.JSON(http.StatusInternalServerError, utils.BuildResp(err.Error(), utils.PipelineBuildJobStopFailed, nil))
		return
	}

	c.JSON(http.StatusOK, utils.BuildResp("流水线任务停止成功", utils.Success, nil))
}

func (h *BuildHandler) DeleteBuild(c *gin.Context) {
	p := new(models.PipelineBuildIdBind)

	if !p.Validate(c) {
		return
	}

	err := h.processor.DeletePipeline(p.Id)

	if err != nil {
		c.JSON(http.StatusInternalServerError, utils.BuildResp(err.Error(), utils.PipelineBuildJobDeleteFailed, nil))
		return
	}

	c.JSON(http.StatusOK, utils.BuildResp("流水线任务删除成功", utils.Success, nil))
}
