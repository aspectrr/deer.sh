package kafkastub

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"strings"
	"time"

	kafka "github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl"
	"github.com/segmentio/kafka-go/sasl/plain"
	"github.com/segmentio/kafka-go/sasl/scram"
)

func NewKafkaGoTransport() Transport {
	return kafkaGoTransport{}
}

type kafkaGoTransport struct{}

type kafkaGoConsumer struct {
	reader *kafka.Reader
}

type kafkaGoProducer struct {
	writer *kafka.Writer
}

func (kafkaGoTransport) NewConsumer(cfg CaptureConfig) (Consumer, error) {
	dialer, err := kafkaDialer(cfg)
	if err != nil {
		return nil, err
	}
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:         append([]string(nil), cfg.BootstrapServers...),
		GroupID:         consumerGroupID(cfg.ID),
		GroupTopics:     append([]string(nil), cfg.Topics...),
		Dialer:          dialer,
		StartOffset:     kafka.LastOffset,
		CommitInterval:  time.Second,
		MinBytes:        1,
		MaxBytes:        10e6,
		MaxWait:         2 * time.Second,
		ReadLagInterval: -1,
	})
	return &kafkaGoConsumer{reader: reader}, nil
}

func (kafkaGoTransport) NewProducer(endpoint string) (Producer, error) {
	writer := &kafka.Writer{
		Addr:                   kafka.TCP(endpoint),
		Balancer:               &kafka.LeastBytes{},
		RequiredAcks:           kafka.RequireOne,
		AllowAutoTopicCreation: true,
		BatchTimeout:           100 * time.Millisecond,
		WriteTimeout:           10 * time.Second,
		ReadTimeout:            10 * time.Second,
	}
	return &kafkaGoProducer{writer: writer}, nil
}

func (c *kafkaGoConsumer) ReadMessage(ctx context.Context) (Record, error) {
	msg, err := c.reader.ReadMessage(ctx)
	if err != nil {
		return Record{}, err
	}
	headers := make([]Header, 0, len(msg.Headers))
	for _, header := range msg.Headers {
		headers = append(headers, Header{
			Key:   header.Key,
			Value: append([]byte(nil), header.Value...),
		})
	}
	return Record{
		Topic:     msg.Topic,
		Partition: int32(msg.Partition),
		Offset:    msg.Offset,
		Key:       append([]byte(nil), msg.Key...),
		Headers:   headers,
		Timestamp: msg.Time.UTC(),
		Value:     append([]byte(nil), msg.Value...),
	}, nil
}

func (c *kafkaGoConsumer) Close() error {
	return c.reader.Close()
}

func (p *kafkaGoProducer) WriteMessage(ctx context.Context, record Record) error {
	headers := make([]kafka.Header, 0, len(record.Headers))
	for _, header := range record.Headers {
		headers = append(headers, kafka.Header{
			Key:   header.Key,
			Value: append([]byte(nil), header.Value...),
		})
	}
	return p.writer.WriteMessages(ctx, kafka.Message{
		Topic:   record.Topic,
		Key:     append([]byte(nil), record.Key...),
		Value:   append([]byte(nil), record.Value...),
		Headers: headers,
		Time:    record.Timestamp.UTC(),
	})
}

func (p *kafkaGoProducer) Close() error {
	return p.writer.Close()
}

func consumerGroupID(configID string) string {
	return "deer-kafkastub-" + sanitizeID(configID)
}

func kafkaDialer(cfg CaptureConfig) (*kafka.Dialer, error) {
	dialer := &kafka.Dialer{
		Timeout:   10 * time.Second,
		DualStack: true,
	}
	if cfg.TLSEnabled {
		tlsConfig := &tls.Config{
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: cfg.InsecureSkipVerify,
		}
		if cfg.TLSCAPEM != "" {
			pool := x509.NewCertPool()
			if !pool.AppendCertsFromPEM([]byte(cfg.TLSCAPEM)) {
				return nil, fmt.Errorf("invalid kafka TLS CA PEM")
			}
			tlsConfig.RootCAs = pool
		}
		dialer.TLS = tlsConfig
	}
	if cfg.Username != "" || cfg.Password != "" {
		mech, err := saslMechanism(cfg)
		if err != nil {
			return nil, err
		}
		dialer.SASLMechanism = mech
	}
	return dialer, nil
}

func saslMechanism(cfg CaptureConfig) (sasl.Mechanism, error) {
	switch strings.ToLower(strings.TrimSpace(cfg.SASLMechanism)) {
	case "", "plain":
		return plain.Mechanism{Username: cfg.Username, Password: cfg.Password}, nil
	case "scram-sha-256":
		return scram.Mechanism(scram.SHA256, cfg.Username, cfg.Password)
	case "scram-sha-512":
		return scram.Mechanism(scram.SHA512, cfg.Username, cfg.Password)
	default:
		return nil, fmt.Errorf("unsupported SASL mechanism %q", cfg.SASLMechanism)
	}
}
