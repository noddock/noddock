package graph

import (
	"context"
	"sync"
)

// Stream 节点间的增量数据通道，供 UI 展示流式输出（LLM chunk 等），channel 满时静默丢弃、不阻塞生产者。
type Stream[T any] struct {
	mu   sync.RWMutex
	subs map[chan T]struct{}
	open bool
	buf  int

	expectedReady int
	readyCount    int
	readyCh       chan struct{}
	readyOnce     sync.Once
}

func NewStream[T any]() *Stream[T] {
	return &Stream[T]{subs: make(map[chan T]struct{}), buf: 64}
}

func NewStreamLarge[T any](size int) *Stream[T] {
	return &Stream[T]{subs: make(map[chan T]struct{}), buf: size}
}

// 标记流打开，生产者在触发下游前调用。
func (s *Stream[T]) Open() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.open = true
}

// 报告流是否处于打开状态（已 Open 且未 Close）。
func (s *Stream[T]) IsOpen() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.open
}

// 声明期望的 ready 消费者数（n<=0 表示不启用等待）。
func (s *Stream[T]) SetExpectedReady(n int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if n <= 0 {
		s.expectedReady = 0
		s.readyCh = nil
		return
	}
	s.expectedReady = n
	s.readyCh = make(chan struct{})
}

// 消费者就绪时调用，达到期望数量后放行所有 WaitReady。
func (s *Stream[T]) Ready() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.readyCh == nil {
		return
	}
	s.readyCount++
	if s.readyCount >= s.expectedReady {
		s.readyOnce.Do(func() { close(s.readyCh) })
	}
}

// 阻塞直到所有期望的消费者 ready，或 ctx 取消时返回 ctx 错误。
func (s *Stream[T]) WaitReady(ctx context.Context) error {
	s.mu.Lock()
	ch := s.readyCh
	s.mu.Unlock()
	if ch == nil {
		return nil
	}
	select {
	case <-ch:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// 返回一个增量 channel；流未打开时返回已关闭的 channel。
func (s *Stream[T]) Subscribe() <-chan T {
	ch := make(chan T, s.buf)
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.open {
		close(ch)
		return ch
	}
	s.subs[ch] = struct{}{}
	return ch
}

// 向所有订阅者广播一个增量，慢消费者静默丢弃。
func (s *Stream[T]) Send(v T) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.open {
		return
	}
	for ch := range s.subs {
		select {
		case ch <- v:
		default:
		}
	}
}

// 广播增量，ctx 取消或流未打开时返回 false。
func (s *Stream[T]) SendSafe(ctx context.Context, v T) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.open {
		return false
	}
	for ch := range s.subs {
		select {
		case ch <- v:
		case <-ctx.Done():
			return false
		default:
		}
	}
	return true
}

// 关闭流并关闭所有订阅 channel，幂等。
func (s *Stream[T]) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.open {
		return
	}
	s.open = false
	for ch := range s.subs {
		close(ch)
	}
	s.subs = nil
}
