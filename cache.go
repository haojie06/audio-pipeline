package main

import (
	"context"
	"time"

	"github.com/allegro/bigcache/v3"
)

var (
	cache *bigcache.BigCache
)

const (
	streamCachePrefix = "stream_"
)

func init() {
	var err error
	cache, err = bigcache.New(context.Background(), bigcache.DefaultConfig(60*time.Minute))
	if err != nil {
		panic(err)
	}
}

func setStreamCache(stream *Stream) error {
	return cache.Set(streamCachePrefix+stream.StreamId, stream.Marshal())
}

func getStreamCache(streamId string) (*Stream, error) {
	data, err := cache.Get(streamCachePrefix + streamId)
	if err != nil {
		return nil, err
	}
	var stream Stream
	if err := stream.Unmarshal(data); err != nil {
		return nil, err
	}
	return &stream, nil
}
