package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func GetDefaultRoute() *gin.Engine {
	route := gin.Default()
	route.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})
	return route
}
