package service

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type PresenceService struct {
	rdb *redis.Client
}

func NewPresenceService(rdb *redis.Client) *PresenceService {
	return &PresenceService{rdb: rdb}
}

func (p *PresenceService) SetOnline(userID uint64) error {
	if p.rdb == nil {
		return nil
	}
	return p.rdb.Set(context.Background(), fmt.Sprintf("online:%d", userID), "1", 60*time.Second).Err()
}

func (p *PresenceService) SetOffline(userID uint64) error {
	if p.rdb == nil {
		return nil
	}
	return p.rdb.Del(context.Background(), fmt.Sprintf("online:%d", userID)).Err()
}

func (p *PresenceService) RefreshOnline(userID uint64) error {
	if p.rdb == nil {
		return nil
	}
	return p.rdb.Expire(context.Background(), fmt.Sprintf("online:%d", userID), 60*time.Second).Err()
}

func (p *PresenceService) GetOnlineStatus(userIDs []uint64) map[uint64]bool {
	result := make(map[uint64]bool)
	if p.rdb == nil {
		for _, id := range userIDs {
			result[id] = false
		}
		return result
	}

	ctx := context.Background()
	pipe := p.rdb.Pipeline()
	cmds := make([]*redis.StringCmd, len(userIDs))
	for i, id := range userIDs {
		cmds[i] = pipe.Get(ctx, fmt.Sprintf("online:%d", id))
	}
	pipe.Exec(ctx)

	for i, cmd := range cmds {
		result[userIDs[i]] = cmd.Err() == nil
	}
	return result
}
