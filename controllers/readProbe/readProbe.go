package readProbe

import (
	"dishrank-go-worker/controllers/check"
	"github.com/gin-gonic/gin"
	"net/http"
)

func Probe (c *gin.Context) {
	c.JSON(http.StatusOK, check.AliveResponse{Success: true, Messsage: "probe success"})
}
