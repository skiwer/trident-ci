package routers

import (
	"github.com/gin-gonic/gin"
	"github.com/skiwer/trident-ci/processor"
	"github.com/skiwer/trident-ci/queue"
	"github.com/skiwer/trident-ci/server/web/handlers"
	"net/http"
)

type pipelineBuildRouter struct {
	name       string
	routerList []RouterItem
}

func NewBuildRecordRouter(p *processor.PipeLineProcessor, queue queue.Queue) RouterInterface {
	serverHandler := handlers.NewBuildHandler(p, queue)
	routerList := []RouterItem{
		{http.MethodPost, "", serverHandler.Build},
		{http.MethodGet, "/:id/progress", serverHandler.GetBuildProgress},
		{http.MethodGet, "/:id/log", serverHandler.GetBuildLog},
		{http.MethodPost, "/:id/stop", serverHandler.StopBuild},
		{http.MethodDelete, "/:id", serverHandler.DeleteBuild},
	}
	return &pipelineBuildRouter{"build", routerList}
}

//构建组路由
func (j pipelineBuildRouter) GetGroupName() string {
	return "/api/v1/build"
}

func (j pipelineBuildRouter) GetRouterGroup(r *gin.RouterGroup, op func(engine *gin.RouterGroup, item RouterItem)) {
	for _, route := range j.routerList {
		op(r, route)
	}
}
