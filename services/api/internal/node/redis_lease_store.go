package node

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisLeaseStore struct {
	client *redis.Client
}

func NewRedisLeaseStore(client *redis.Client) *RedisLeaseStore {
	return &RedisLeaseStore{client: client}
}

func (s *RedisLeaseStore) PutLease(ctx context.Context, lease LeaseSnapshot, ttl time.Duration) error {
	payload, err := json.Marshal(lease)
	if err != nil {
		return err
	}
	if err := s.client.Set(ctx, leaseKey(lease.NodeID), payload, ttl).Err(); err != nil {
		return err
	}
	route := map[string]string{
		"node_id":         lease.NodeID,
		"session_id":      lease.SessionID,
		"api_instance_id": lease.APIInstanceID,
	}
	routePayload, err := json.Marshal(route)
	if err != nil {
		return err
	}
	return s.client.Set(ctx, sessionKey(lease.NodeID), routePayload, ttl).Err()
}

func (s *RedisLeaseStore) GetLease(ctx context.Context, nodeID string) (LeaseSnapshot, bool, error) {
	raw, err := s.client.Get(ctx, leaseKey(nodeID)).Bytes()
	if errors.Is(err, redis.Nil) {
		return LeaseSnapshot{}, false, nil
	}
	if err != nil {
		return LeaseSnapshot{}, false, err
	}

	var lease LeaseSnapshot
	if err := json.Unmarshal(raw, &lease); err != nil {
		return LeaseSnapshot{}, false, err
	}
	return lease, true, nil
}

func leaseKey(nodeID string) string {
	return "rayip:lease:node:" + nodeID
}

func sessionKey(nodeID string) string {
	return "rayip:session:node:" + nodeID
}
