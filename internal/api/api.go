package api

import (
	"fmt"
	"log"
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

	g := route.Group("/api")

	{
		g.POST("/upload", UploadFileRouteEndpoint)
	}
	return route
}

// UploadFile godoc
// @Summary Upload file and saves it in files/ dir
// @Description Uploads a file via multipart/form-data
// @Tags files
// @Accept multipart/form-data
// @Produce text/plain
// @Param file formData file true "file"
// @Success 200 {string} string "uploaded"
// @Failure 400 {string} string "bad request"
// @Router /api/upload [post]
func UploadFileRouteEndpoint(c *gin.Context) {
	file, _ := c.FormFile("file")
	log.Println(file.Filename)

	c.SaveUploadedFile(file, "./files/"+file.Filename)

	c.String(http.StatusOK, fmt.Sprintf("'%s' uploaded!", file.Filename))
}
