package nats_cli

import (
	"time"

	"github.com/nats-io/nats.go"
)

func InitNatsForTest() {
	natsClient = &mockNats{}
}

type mockNats struct {
}

func (m *mockNats) Drain() error {
	return nil
}

func (m *mockNats) Flush() error {
	return nil
}
func (m *mockNats) Close() {
	return
}
func (m *mockNats) Request(subj, msgName string, data []byte, timeout time.Duration) (*nats.Msg, error) {
	return nil, nil
}
func (m *mockNats) Publish(subj, msgName string, data []byte) error {
	return nil
}
func (m *mockNats) Subscribe(subj string, cb nats.MsgHandler) (*nats.Subscription, error) {
	return nil, nil
}
func (m *mockNats) QueueSubscribe(subj, queue string, cb nats.MsgHandler) (*nats.Subscription, error) {
	return nil, nil
}
