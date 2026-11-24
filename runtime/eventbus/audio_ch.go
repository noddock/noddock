package eventbus

import "sync"

// AudioStream 一次音频流式播放的全局通道，生产者写 Ch、消费者按 key 读取。
type AudioStream struct {
	Ch   chan []byte
	Mime string
}

type AudioBroker struct {
	streams sync.Map
}

var AudioBus = AudioBroker{}

func (b *AudioBroker) Register(key string, s *AudioStream) {
	b.streams.Store(key, s)
}

func (b *AudioBroker) Get(key string) (*AudioStream, bool) {
	v, ok := b.streams.Load(key)
	if !ok {
		return nil, false
	}
	return v.(*AudioStream), true
}

func (b *AudioBroker) Delete(key string) {
	b.streams.Delete(key)
}
