package api

import (
	"github.com/gin-gonic/gin"
)

func SetupRouter(e *gin.Engine) {
	e.GET("/ping", Ping)
}
