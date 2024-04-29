package main

import (
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	r.GET("/convert_stream", ConvertAudiosToStream)
	r.POST("/stream", CreateStream)
	r.HEAD("/stream/:stream_id", HeadStreamHandler)
	r.GET("/stream/:stream_id", GetStreamByRange)
	r.Run("0.0.0.0:8080")
}
