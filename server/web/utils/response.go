package utils

import "github.com/gin-gonic/gin"

func BuildResp(msg string, code int, data interface{}) gin.H {
	return gin.H{"msg": msg, "code": code, "data": data}
}

const (
	Success                           = 20000
	PathBindError                     = 30000
	NetWorkError                      = 30002
	ParamBindError                    = 30003
	PipelineBuildJobCreateFailed      = 30004
	PipelineBuildJobLogGetFailed      = 30005
	PipelineBuildJobProgressGetFailed = 30006
	PipelineBuildJobStopFailed        = 30007
	PipelineBuildJobDeleteFailed      = 30008
)
