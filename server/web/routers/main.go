package routers

import (
	"github.com/gin-gonic/gin"
	"github.com/skiwer/trident-ci/processor"
	"github.com/skiwer/trident-ci/queue"
	"net/http"
)

//路由组接口
type RouterInterface interface {
	GetGroupName() string
	GetRouterGroup(r *gin.RouterGroup, op func(engine *gin.RouterGroup, item RouterItem))
}

//路由项
type RouterItem struct {
	method   string
	path     string
	function func(c *gin.Context)
}

//初始化路由
func InitRouters(r *gin.Engine, p *processor.PipeLineProcessor, queue queue.Queue) {
	routerSlice := getRoutersSlice(p, queue)

	for _, item := range routerSlice {
		GetRouterGroup(item, r)
	}
}

//获取路由组对象list
func getRoutersSlice(p *processor.PipeLineProcessor, queue queue.Queue) []RouterInterface {
	return []RouterInterface{
		NewBuildRecordRouter(p, queue),
	}
}

//根据路由组对象注册路由组
func GetRouterGroup(routers RouterInterface, r *gin.Engine) {
	var routerGroup *gin.RouterGroup
	routerGroup = r.Group(routers.GetGroupName())
	routers.GetRouterGroup(routerGroup, buildRouter)
}

//根据路由类型注册路由
func buildRouter(engine *gin.RouterGroup, item RouterItem) {
	switch item.method {
	case http.MethodGet:
		engine.GET(item.path, item.function)
	case http.MethodPost:
		engine.POST(item.path, item.function)
	case http.MethodPut:
		engine.PUT(item.path, item.function)
	case http.MethodDelete:
		engine.DELETE(item.path, item.function)
	case http.MethodPatch:
		engine.PATCH(item.path, item.function)
	case http.MethodOptions:
		engine.OPTIONS(item.path, item.function)
	default:
		panic("router method error, method value:" + item.method + " route path:" + item.path)
	}
}
