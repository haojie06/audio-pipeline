package main

import (
	"encoding/json"

	"github.com/google/uuid"
)

type CreateStreamRequest struct {
	Audios []string `json:"audios"`
}

type CreateStreamResponse struct {
	StreamId string `json:"stream_id"`
}

type Stream struct {
	StreamId     string   `json:"stream_id"`
	Audios       []string `json:"audios"`
	AudioLengths []int    `json:"audio_lengths"` // for range request mapping
	Completed    bool     `json:"completed"`
}

func NewStreamModel() Stream {
	return Stream{
		StreamId: uuid.New().String(),
	}
}

func (s *Stream) Marshal() []byte {
	b, _ := json.Marshal(s)
	return b
}

func (s *Stream) Unmarshal(data []byte) error {
	return json.Unmarshal(data, s)
}
