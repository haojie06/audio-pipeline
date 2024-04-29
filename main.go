package main

import (
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	r.GET("/convert_stream", func(c *gin.Context) {
		audios := c.QueryArray("audios")
		fmt.Printf("convert audios: %+v\n", audios)
		// todo check audios by head request
		stream, err := generateStream(audios)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.Header("Content-Type", "audio/mpeg")
		c.Stream(func(w io.Writer) bool {
			data, ok := <-stream
			if !ok {
				fmt.Printf("stream closed\n")
				return false
			}
			if _, err := w.Write(data); err != nil {
				fmt.Printf("stream error: %s\n", err)
				return false
			}
			fmt.Printf("streaming %d bytes\n", len(data))
			return true
		})
	})
	r.Run("0.0.0.0:8080")
}

type ChanWriter struct {
	Ch chan<- []byte
}

func (w ChanWriter) Write(p []byte) (n int, err error) {
	w.Ch <- p
	return len(p), nil
}

func generateStream(audios []string) (chan []byte, error) {
	dataChan := make(chan []byte)
	go func() {
		for _, audio := range audios {
			fmt.Printf("loading %s\n", audio)
			resp, err := http.Get(audio)
			if err != nil {
				fmt.Printf("http get error: %s\n", err)
				close(dataChan)
				return
			}
			fmt.Printf("streaming %s\n", audio)
			_, err = io.Copy(ChanWriter{Ch: dataChan}, resp.Body)
			if err != nil {
				fmt.Printf("io copy error: %s\n", err)
				close(dataChan)
				return
			}
			fmt.Printf("streamed %s\n", audio)
		}
		close(dataChan)
	}()
	return dataChan, nil
}
