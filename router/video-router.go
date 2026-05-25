package router

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/controller"
	"github.com/QuantumNous/new-api/middleware"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"

	"github.com/gin-gonic/gin"
)

func SetVideoRouter(router *gin.Engine) {
	// Video proxy: accepts either session auth (dashboard) or token auth (API clients)
	videoProxyRouter := router.Group("/v1")
	videoProxyRouter.Use(middleware.RouteTag("relay"))
	videoProxyRouter.Use(middleware.TokenOrUserAuth())
	{
		videoProxyRouter.GET("/videos/:task_id/content", controller.VideoProxy)
	}

	videoV1Router := router.Group("/v1")
	videoV1Router.Use(middleware.RouteTag("relay"))
	videoV1Router.Use(middleware.TokenAuth(), middleware.Distribute())
	{
		videoV1Router.POST("/video/generations", controller.RelayTask)
		videoV1Router.GET("/video/generations/:task_id", controller.RelayTaskFetch)
		videoV1Router.POST("/videos/:video_id/remix", controller.RelayTask)
	}
	// openai compatible API video routes
	// docs: https://platform.openai.com/docs/api-reference/videos/create
	{
		videoV1Router.POST("/videos", controller.RelayTask)
		videoV1Router.GET("/videos/:task_id", controller.RelayTaskFetch)
	}

	volcesSubmitRouter := router.Group("/volces")
	volcesSubmitRouter.Use(middleware.RouteTag("relay"))
	volcesSubmitRouter.Use(middleware.TokenAuth(), middleware.Distribute())
	{
		volcesSubmitRouter.POST("/api/v3/contents/generations/tasks", middleware.VolcesSeedanceRequestConvert(), controller.RelayTask)
	}

	volcesFetchRouter := router.Group("/volces")
	volcesFetchRouter.Use(middleware.RouteTag("relay"))
	volcesFetchRouter.Use(middleware.TokenAuth())
	{
		volcesFetchRouter.GET("/api/v3/contents/generations/tasks/:task_id", func(c *gin.Context) {
			common.SetContextKey(c, constant.ContextKeyVolcesCompat, true)
			c.Set("relay_mode", relayconstant.RelayModeVideoFetchByID)
			c.Set("task_id", c.Param("task_id"))
			controller.RelayTaskFetch(c)
		})
	}

	mViduSubmitRouter := router.Group("/m-vidu/ent/v2")
	mViduSubmitRouter.Use(middleware.RouteTag("relay"))
	mViduSubmitRouter.Use(middleware.MViduRequestConvert(), middleware.TokenAuth(), middleware.Distribute())
	{
		mViduSubmitRouter.POST("/text2video", controller.RelayTask)
		mViduSubmitRouter.POST("/img2video", controller.RelayTask)
		mViduSubmitRouter.POST("/start-end2video", controller.RelayTask)
		mViduSubmitRouter.POST("/reference2video", controller.RelayTask)
	}

	mViduFetchRouter := router.Group("/m-vidu/ent/v2")
	mViduFetchRouter.Use(middleware.RouteTag("relay"))
	mViduFetchRouter.Use(middleware.TokenAuth())
	{
		mViduFetchRouter.GET("/tasks/:task_id/creations", func(c *gin.Context) {
			common.SetContextKey(c, constant.ContextKeyMViduCompat, true)
			c.Set("relay_mode", relayconstant.RelayModeVideoFetchByID)
			c.Set("task_id", c.Param("task_id"))
			controller.RelayTaskFetch(c)
		})
	}

	klingV1Router := router.Group("/kling/v1")
	klingV1Router.Use(middleware.RouteTag("relay"))
	klingV1Router.Use(middleware.KlingRequestConvert(), middleware.TokenAuth(), middleware.Distribute())
	{
		klingV1Router.POST("/videos/text2video", controller.RelayTask)
		klingV1Router.POST("/videos/image2video", controller.RelayTask)
		klingV1Router.GET("/videos/text2video/:task_id", controller.RelayTaskFetch)
		klingV1Router.GET("/videos/image2video/:task_id", controller.RelayTaskFetch)
	}

	// Jimeng official API routes - direct mapping to official API format
	jimengOfficialGroup := router.Group("jimeng")
	jimengOfficialGroup.Use(middleware.RouteTag("relay"))
	jimengOfficialGroup.Use(middleware.JimengRequestConvert(), middleware.TokenAuth(), middleware.Distribute())
	{
		// Maps to: /?Action=CVSync2AsyncSubmitTask&Version=2022-08-31 and /?Action=CVSync2AsyncGetResult&Version=2022-08-31
		jimengOfficialGroup.POST("/", controller.RelayTask)
	}
}
