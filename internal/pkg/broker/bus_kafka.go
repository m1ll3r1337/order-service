package broker

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/IBM/sarama"
	"github.com/gofrs/uuid"
	"github.com/rs/zerolog/log"

	"github.com/m1ll3r1337/order-service/internal/pkg/broker/codec"
)

var _ Bus[any] = (*kafkaBus[any])(nil)

const (
	consumeBackoffInitial = 500 * time.Millisecond
	consumeBackoffMax     = 30 * time.Second
	consumeMaxRetries     = 5
)

type EventIdGetter interface {
	EventId() string
}

type kafkaBus[T any] struct {
	client        *KafkaClient
	codec         codec.Codec[T]
	topic         string
	consumerGroup string
	consumer      sarama.ConsumerGroup
}

func NewBus[T any](client *KafkaClient, c codec.Codec[T], topic, consumerGroup string) (Bus[T], error) {
	if client == nil {
		return nil, fmt.Errorf("broker: nil client")
	}
	if topic == "" {
		return nil, fmt.Errorf("broker: empty topic")
	}

	group := Coalesce(consumerGroup, client.DefaultConsumerGroup())
	if group == "" {
		return nil, fmt.Errorf("broker: empty group")
	}

	return &kafkaBus[T]{
		client:        client,
		codec:         c,
		topic:         topic,
		consumerGroup: group,
	}, nil
}

func MustKafkaBus[T any](client *KafkaClient, c codec.Codec[T], topic, consumerGroup string) Bus[T] {
	bus, err := NewBus(client, c, topic, consumerGroup)
	if err != nil {
		log.Fatal().Err(err).Str("topic", topic).Msg("broker: failed to create kafka bus")
	}

	return bus
}

func getEventId[T any](v *T) string {
	var iface any = v
	getter, ok := iface.(EventIdGetter)
	if ok {
		id := getter.EventId()
		if id != "" {
			return id
		}
	}

	return uuid.Must(uuid.NewV4()).String()
}

func (b *kafkaBus[T]) Send(_ context.Context, msg *T, headers ...Header) error {
	msgBytes, err := b.codec.Encode(msg)
	if err != nil {
		return err
	}

	id := getEventId(msg)

	prodMsg := &sarama.ProducerMessage{
		Topic: b.topic,
		Key:   sarama.StringEncoder(id),
		Value: sarama.ByteEncoder(msgBytes),
	}

	if len(headers) > 0 {
		hdrs := make([]sarama.RecordHeader, len(headers))
		for i, h := range headers {
			hdrs[i] = sarama.RecordHeader{
				Key:   []byte(h.Key),
				Value: []byte(h.Value),
			}
		}

		prodMsg.Headers = hdrs
	}

	_, _, err = b.client.Producer().SendMessage(prodMsg)
	if err != nil {
		return fmt.Errorf("broker: failed to send producer message: %w", err)
	}

	return nil
}

func (b *kafkaBus[T]) QueueName() string {
	return b.topic
}

func (b *kafkaBus[T]) Subscribe(ctx context.Context, wg *sync.WaitGroup, handler MessageHandler[T]) error {
	consumer, err := b.client.NewConsumerGroup(b.consumerGroup)
	if err != nil {
		return fmt.Errorf("broker: create consumer group %q: %w", b.consumerGroup, err)
	}
	b.consumer = consumer

	h := &consumerGroupHandler[T]{codec: b.codec, handler: handler, topic: b.topic}

	if wg != nil {
		wg.Add(1)
	}

	go func() {
		for err := range consumer.Errors() {
			log.Warn().Err(err).Str("topic", b.topic).Msg("consumer group error")
		}
	}()

	go func() {
		if wg != nil {
			defer wg.Done()
		}
		defer func() {
			if err := consumer.Close(); err != nil && !errors.Is(err, sarama.ErrClosedClient) {
				log.Error().Err(err).Str("topic", b.topic).Msg("failed to close consumer group")
			}
		}()

		backoff := consumeBackoffInitial
		fails := 0
		for {
			err := consumer.Consume(ctx, []string{b.topic}, h)
			if ctx.Err() != nil {
				return
			}
			if err == nil {
				backoff = consumeBackoffInitial
				fails = 0
				continue
			}

			fails++
			if fails >= consumeMaxRetries {
				log.Fatal().Err(err).Msg("max retries reached")
			}

			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
			}

			backoff = min(backoff*2, consumeBackoffMax)
		}
	}()

	return nil
}

type consumerGroupHandler[T any] struct {
	codec   codec.Codec[T]
	handler MessageHandler[T]
	topic   string
}

func (h *consumerGroupHandler[T]) Setup(sarama.ConsumerGroupSession) error   { return nil }
func (h *consumerGroupHandler[T]) Cleanup(sarama.ConsumerGroupSession) error { return nil }

func (h *consumerGroupHandler[T]) ConsumeClaim(
	session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim,
) error {
	ctx := session.Context()
	defer session.Commit()
	for {
		select {
		case <-ctx.Done():
			return nil
		case msg, ok := <-claim.Messages():
			if !ok {
				return nil
			}
			if err := h.process(ctx, session, msg); err != nil {
				return err
			}
		}
	}
}

func (h *consumerGroupHandler[T]) process(
	ctx context.Context, session sarama.ConsumerGroupSession, msg *sarama.ConsumerMessage,
) error {
	decoded, err := h.codec.Decode(msg.Value)
	if err != nil {
		log.Error().Err(err).Msg("invalid msg")
		session.MarkMessage(msg, "")
		return nil
	}

	hdrs := make([]Header, len(msg.Headers))
	for i, h := range msg.Headers {
		hdrs[i] = Header{
			Key:   string(h.Key),
			Value: string(h.Value),
		}
	}

	err = h.handler(ctx, decoded, hdrs)
	if err != nil && IsNotCriticalError(err) {
		log.Warn().Err(err).Msg("not critical error")
		session.MarkMessage(msg, "")
		return nil
	}
	if err != nil {
		log.Error().Err(err).Msg("critical error")
		return err
	}

	session.MarkMessage(msg, "")
	return nil
}

func (b *kafkaBus[T]) Close() error {
	if b.consumer != nil {
		err := b.consumer.Close()
		return err
	}

	return nil
}
