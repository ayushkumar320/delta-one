package events

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/ayush/delta-one/shared/middleware"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// maxLen caps the stream length. Events are for notification and audit, not
// durable storage, so old entries are trimmed rather than kept forever.
const maxLen = 10_000

// Publisher writes events to the shared stream.
type Publisher struct {
	rdb *redis.Client
}

// NewPublisher returns a Publisher backed by the given Redis client.
func NewPublisher(rdb *redis.Client) *Publisher { return &Publisher{rdb: rdb} }

// Publish writes one event. The caller's request ID travels with the payload so
// an asynchronous side effect can be traced back to the request that caused it.
func (p *Publisher) Publish(ctx context.Context, eventType string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal %s payload: %w", eventType, err)
	}
	env := Envelope{
		ID:         uuid.NewString(),
		Type:       eventType,
		OccurredAt: time.Now().UTC(),
		RequestID:  middleware.RequestIDFrom(ctx),
		Data:       data,
	}
	body, err := json.Marshal(env)
	if err != nil {
		return fmt.Errorf("marshal %s envelope: %w", eventType, err)
	}
	err = p.rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: Stream,
		MaxLen: maxLen,
		Approx: true,
		Values: map[string]any{"envelope": string(body)},
	}).Err()
	if err != nil {
		return fmt.Errorf("publish %s: %w", eventType, err)
	}
	return nil
}

// Handler processes a single event. Returning an error leaves the event
// unacknowledged so it can be redelivered.
type Handler func(ctx context.Context, env Envelope) error

// Consumer reads the shared stream as part of a named consumer group.
type Consumer struct {
	rdb      *redis.Client
	group    string
	consumer string
}

// NewConsumer returns a Consumer for the given group, which should be the
// service name so each service receives every event once.
func NewConsumer(rdb *redis.Client, group, consumer string) *Consumer {
	return &Consumer{rdb: rdb, group: group, consumer: consumer}
}

// Run consumes events until ctx is cancelled. Events whose handler fails are
// left unacknowledged and are redelivered on the next pass over the pending
// list, so a transient failure does not drop a notification.
//
// ponytail: redelivery has no attempt limit or dead-letter stream. Add one if a
// permanently failing event ever starts spinning.
func (c *Consumer) Run(ctx context.Context, handle Handler) error {
	// MKSTREAM creates the stream if no event has been published yet.
	if err := c.rdb.XGroupCreateMkStream(ctx, Stream, c.group, "0").Err(); err != nil &&
		!strings.Contains(err.Error(), "BUSYGROUP") {
		return fmt.Errorf("create consumer group %s: %w", c.group, err)
	}

	// Start with previously delivered but unacknowledged events, then switch to
	// new ones. ">" means "never delivered to this group".
	cursor := "0"
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		streams, err := c.rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    c.group,
			Consumer: c.consumer,
			Streams:  []string{Stream, cursor},
			Count:    16,
			Block:    5 * time.Second,
		}).Result()
		if errors.Is(err, redis.Nil) {
			continue
		}
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			log.Printf("events: read %s: %v", c.group, err)
			time.Sleep(time.Second)
			continue
		}

		delivered := 0
		for _, stream := range streams {
			delivered += len(stream.Messages)
			for _, msg := range stream.Messages {
				c.handleMessage(ctx, msg, handle)
			}
		}
		// The pending list is drained; move on to undelivered events.
		if cursor == "0" && delivered == 0 {
			cursor = ">"
		}
	}
}

func (c *Consumer) handleMessage(ctx context.Context, msg redis.XMessage, handle Handler) {
	raw, _ := msg.Values["envelope"].(string)
	var env Envelope
	if err := json.Unmarshal([]byte(raw), &env); err != nil {
		// A malformed entry can never succeed; acknowledge it so it stops
		// being redelivered, and log it loudly.
		log.Printf("events: discarding malformed entry %s: %v", msg.ID, err)
		c.rdb.XAck(ctx, Stream, c.group, msg.ID)
		return
	}
	if err := handle(ctx, env); err != nil {
		log.Printf("events: handler failed for %s (%s): %v", env.Type, msg.ID, err)
		return
	}
	if err := c.rdb.XAck(ctx, Stream, c.group, msg.ID).Err(); err != nil {
		log.Printf("events: ack %s: %v", msg.ID, err)
	}
}
