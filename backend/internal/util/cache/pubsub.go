package cache_utils

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"dockvol-backend/internal/util/logger"
)

// subscription pairs a handler with the context cancel that tears it down.
type subscription struct {
	handler func(message string)
	cancel  context.CancelFunc
}

// PubSubManager is an in-process fan-out bus. DockVol runs single-process, so a
// map of channel -> subscribers reaches every listener: there are no other
// instances. Because Subscribe registers the handler before returning, delivery
// is synchronous and no publish/ready handshake is needed.
type PubSubManager struct {
	mu            sync.RWMutex
	subscriptions map[string]*subscription
	logger        *slog.Logger
}

func NewPubSubManager() *PubSubManager {
	return &PubSubManager{
		subscriptions: make(map[string]*subscription),
		logger:        logger.GetLogger(),
	}
}

func (m *PubSubManager) Subscribe(
	ctx context.Context,
	channel string,
	handler func(message string),
) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.subscriptions[channel]; exists {
		return fmt.Errorf("already subscribed to channel: %s", channel)
	}

	subCtx, cancel := context.WithCancel(ctx)
	registered := &subscription{handler: handler, cancel: cancel}
	m.subscriptions[channel] = registered

	// Identity check, not existence: a later Subscribe on the same channel must
	// not be torn down by this subscription's context cancellation.
	context.AfterFunc(subCtx, func() {
		m.mu.Lock()
		defer m.mu.Unlock()

		if m.subscriptions[channel] == registered {
			delete(m.subscriptions, channel)
		}
	})

	m.logger.Info("started subscription", "channel", channel)

	return nil
}

func (m *PubSubManager) Publish(_ context.Context, channel, message string) error {
	m.mu.RLock()
	sub, exists := m.subscriptions[channel]
	m.mu.RUnlock()

	if !exists {
		return nil
	}

	m.deliver(channel, sub.handler, message)

	return nil
}

func (m *PubSubManager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for channel, sub := range m.subscriptions {
		sub.cancel()
		delete(m.subscriptions, channel)
	}

	return nil
}

func (m *PubSubManager) deliver(channel string, handler func(message string), message string) {
	defer func() {
		if r := recover(); r != nil {
			m.logger.Error("panic in message handler", "channel", channel, "panic", r)
		}
	}()

	handler(message)
}
