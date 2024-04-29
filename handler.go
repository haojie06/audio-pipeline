package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/allegro/bigcache/v3"
	"github.com/gin-gonic/gin"
)

func ConvertAudiosToStream(c *gin.Context) {
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
}

func CreateStream(c *gin.Context) {
	var req CreateStreamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	stream := NewStreamModel()
	stream.Audios = req.Audios
	for _, audio := range req.Audios {
		length, err := getAudioLength(audio)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		stream.AudioLengths = append(stream.AudioLengths, length)
	}
	setStreamCache(&stream)
	c.JSON(200, CreateStreamResponse{StreamId: stream.StreamId})
}

func HeadStreamHandler(c *gin.Context) {
	streamId := c.Param("stream_id")
	stream, err := getStreamCache(streamId)
	// bigcache.ErrEntryNotFound
	if err != nil {
		if errors.Is(err, bigcache.ErrEntryNotFound) {
			c.JSON(404, gin.H{"error": "stream not found"})
		} else {
			c.JSON(400, gin.H{"error": err.Error()})
		}
		return
	}
	c.Header("Content-Type", "audio/mpeg")
	c.Header("Accept-Ranges", "bytes")
	c.Header("Transfer-Encoding", "chunked")
	if stream.Completed {
		length := 0
		for _, l := range stream.AudioLengths {
			length += l
		}
		c.Header("Content-Length", fmt.Sprintf("%d", length))
	}
	c.Status(200)
}

func GetStreamByRange(c *gin.Context) {
	streamId := c.Param("stream_id")
	stream, err := getStreamCache(streamId)
	if err != nil {
		if errors.Is(err, bigcache.ErrEntryNotFound) {
			c.JSON(404, gin.H{"error": "stream not found"})
		} else {
			c.JSON(400, gin.H{"error": err.Error()})
		}
		return
	}
	rangeHeader := c.GetHeader("Range")
	// map range to audio
	// Range: bytes=0-100,200-300
	var startPoint, endPoint int
	if rangeHeader == "" || rangeHeader == "bytes=0-" {
		startPoint = 0
		endPoint = 8192
	} else {
		var err error
		startPoint, endPoint, err = parseRangeHeader(rangeHeader)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
	}
	// find start and end audio index
	tempLength := 0
	startAudioIndex := -1
	endAudioIndex := -1
	for i, audioLength := range stream.AudioLengths {
		tempLength += audioLength
		if startPoint < tempLength && startAudioIndex == -1 {
			startAudioIndex = i
		}
		if endPoint < tempLength && endAudioIndex == -1 {
			endAudioIndex = i
			break
		}
		if startAudioIndex != -1 && endAudioIndex != -1 {
			break
		}
	}
	if startAudioIndex == -1 || endAudioIndex == -1 {
		c.JSON(http.StatusRequestedRangeNotSatisfiable, gin.H{"error": fmt.Sprintf("range %s not satisfiable", rangeHeader)})
		return
	}
	var audioBuffer bytes.Buffer
	for i := startAudioIndex; i <= endAudioIndex; i++ {
		req, err := http.NewRequest("GET", stream.Audios[i], nil)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		if startAudioIndex == endAudioIndex {
			// all data in the same audio
			req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", startPoint, endPoint))
		} else {
			if i == startAudioIndex {
				// in the first audio
				req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", startPoint, stream.AudioLengths[i]-1))
			} else if i == endAudioIndex {
				// in the last audio
				req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", 0, endPoint))
			} else {
				// load all data
				req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", 0, stream.AudioLengths[i]-1))
			}
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		if _, err := audioBuffer.ReadFrom(resp.Body); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		if stream.Completed {
			length := 0
			for _, l := range stream.AudioLengths {
				length += l
			}
			c.Header("Content-Range", fmt.Sprintf("bytes %d-%d/%d", startPoint, endPoint, length))
		} else {
			c.Header("Content-Range", fmt.Sprintf("bytes %d-%d/*", startPoint, endPoint))
		}
		c.Data(http.StatusPartialContent, "audio/mpeg", audioBuffer.Bytes())
	}

	// for i, audioLength := range stream.AudioLengths {
	// 	tempLength += audioLength
	// 	if startPoint < tempLength {
	// 		req, err := http.NewRequest("GET", stream.Audios[i], nil)
	// 		if err != nil {
	// 			c.JSON(500, gin.H{"error": err.Error()})
	// 			return
	// 		}
	// 		if endPoint < tempLength {
	// 			// all data in the same audio
	// 			req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", startPoint, endPoint))
	// 		} else {
	// 			// in the next audio
	// 			req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", startPoint, audioLength-1))
	// 			endPoint = endPoint - audioLength
	// 		}
	// 		resp, err := http.DefaultClient.Do(req)
	// 		if err != nil {
	// 			c.JSON(500, gin.H{"error": err.Error()})
	// 			return
	// 		}
	// 		if _, err := audioBuffer.ReadFrom(resp.Body); err != nil {
	// 			c.JSON(500, gin.H{"error": err.Error()})
	// 			return
	// 		}
	// 	}
	// }
}

func getAudioLength(audio string) (int, error) {
	resp, err := http.Head(audio)
	if err != nil {
		return 0, err
	}
	if resp.Header.Get("Accept-Ranges") != "bytes" {
		return 0, errors.New("server does not support Range requests")
	}
	return int(resp.ContentLength), nil
}

func parseRangeHeader(rangeHeader string) (int, int, error) {
	rangeStr := strings.TrimPrefix(rangeHeader, "bytes=")
	rangeParts := strings.Split(rangeStr, "-")
	if len(rangeParts) != 2 {
		return 0, 0, errors.New("invalid range header")
	}
	startPoint, err := strconv.Atoi(rangeParts[0])
	if err != nil {
		return 0, 0, err
	}
	endPoint, err := strconv.Atoi(rangeParts[1])
	if err != nil {
		return 0, 0, err
	}
	return startPoint, endPoint, nil
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
