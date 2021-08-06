package models

import (
	"github.com/gin-gonic/gin"
	v1 "github.com/skiwer/trident-ci/api/pb/v1"
	"github.com/skiwer/trident-ci/server/web/utils"
	"net/http"
)

type PipelineBuildParams struct {
	Pipeline *v1.Pipeline `json:"pipeline" bind:"required"`
}

func (p *PipelineBuildParams) Validate(c *gin.Context) bool {
	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(http.StatusBadRequest, utils.BuildResp(err.Error(), utils.ParamBindError, nil))
		return false
	}

	if p.Pipeline == nil {
		c.JSON(http.StatusBadRequest, utils.BuildResp("流水线不能为空", utils.ParamBindError, nil))
		return false
	}

	return true
}

type PipelineBuildIdBind struct {
	Id string `uri:"id" binding:"required,uuid4"`
}

// 校验job是否存在
func (p *PipelineBuildIdBind) Validate(c *gin.Context) (success bool) {
	if err := c.ShouldBindUri(&p); err != nil {
		c.JSON(http.StatusBadRequest, utils.BuildResp(err.Error(), utils.PathBindError, nil))
		return false
	}

	return true
}
