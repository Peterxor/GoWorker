package router

import (
	"dishrank-go-worker/controllers/check"
	"dishrank-go-worker/controllers/readProbe"
	"github.com/gin-gonic/gin"
)

func Router() *gin.Engine {
	route := gin.Default()

	route.GET("/read-probe", readProbe.Probe)
	route.GET("/check-live", check.CheckAlive)

	return route
}