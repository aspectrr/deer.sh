package kafkastub

import (
	"context"
	"fmt"
)

type Consumer interface {
	ReadMessage(context.Context) (Record, error)
	Close() error
}

type Producer interface {
	WriteMessage(context.Context, Record) error
	Close() error
}

type Transport interface {
	NewConsumer(CaptureConfig) (Consumer, error)
	NewProducer(string) (Producer, error)
}

type noopTransport struct{}

func (noopTransport) NewConsumer(CaptureConfig) (Consumer, error) {
	return nil, fmt.Errorf("kafka transport not configured")
}

func (noopTransport) NewProducer(string) (Producer, error) {
	return nil, fmt.Errorf("kafka transport not configured")
}
