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
		r, err := generateStream(audios)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.Header("Content-Type", "audio/mpeg")
		c.Stream(func(w io.Writer) bool {
			if _, err := io.Copy(w, r); err != nil {
				fmt.Printf("io copy error: %s\n", err)
			}
			return false
		})
	})
	r.Run("0.0.0.0:8080")
}

func generateStream(audios []string) (*io.PipeReader, error) {
	r, w := io.Pipe()
	go func() {
		defer w.Close()
		for _, audio := range audios {
			fmt.Printf("loading %s\n", audio)
			resp, err := http.Get(audio)
			if err != nil {
				fmt.Printf("http get error: %s\n", err)
				return
			}
			fmt.Printf("start streaming %s\n", audio)
			_, err = io.Copy(w, resp.Body)
			if err != nil && err != io.EOF {
				fmt.Printf("io copy error: %s\n", err)
				return
			}
			fmt.Printf("streamed %s\n", audio)
		}
	}()
	return r, nil
}
