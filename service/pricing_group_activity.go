package service

import (
	"context"
	"encoding/base64"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"

	"github.com/go-redis/redis/v8"
)

const (
	pricingGroupActivityLease        = 30 * time.Second
	pricingGroupActivityHeartbeat    = 10 * time.Second
	pricingGroupActivityRedisTimeout = 500 * time.Millisecond
	pricingGroupActivityKeyPrefix    = "new-api:pricing-group:activity:"
)

type PricingGroupActivity struct {
	Users       int `json:"users"`
	Connections int `json:"connections"`
}

type pricingGroupActivityEntry struct {
	userID    int
	expiresAt int64
}

var localPricingGroupActivity = struct {
	sync.Mutex
	groups map[string]map[string]pricingGroupActivityEntry
}{groups: make(map[string]map[string]pricingGroupActivityEntry)}

func pricingGroupActivityRedisKey(group string) string {
	encoded := base64.RawURLEncoding.EncodeToString([]byte(group))
	return pricingGroupActivityKeyPrefix + encoded
}

func refreshLocalPricingGroupActivity(group, member string, userID int, expiresAt int64) {
	localPricingGroupActivity.Lock()
	defer localPricingGroupActivity.Unlock()
	entries := localPricingGroupActivity.groups[group]
	if entries == nil {
		entries = make(map[string]pricingGroupActivityEntry)
		localPricingGroupActivity.groups[group] = entries
	}
	entries[member] = pricingGroupActivityEntry{userID: userID, expiresAt: expiresAt}
}

func removeLocalPricingGroupActivity(group, member string) {
	localPricingGroupActivity.Lock()
	defer localPricingGroupActivity.Unlock()
	entries := localPricingGroupActivity.groups[group]
	delete(entries, member)
	if len(entries) == 0 {
		delete(localPricingGroupActivity.groups, group)
	}
}

func refreshRedisPricingGroupActivity(group, member string, expiresAt int64) {
	if !common.RedisEnabled || common.RDB == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), pricingGroupActivityRedisTimeout)
	defer cancel()
	pipeline := common.RDB.TxPipeline()
	pipeline.ZAdd(ctx, pricingGroupActivityRedisKey(group), &redis.Z{
		Score:  float64(expiresAt),
		Member: member,
	})
	pipeline.Expire(ctx, pricingGroupActivityRedisKey(group), 2*pricingGroupActivityLease)
	_, _ = pipeline.Exec(ctx)
}

func removeRedisPricingGroupActivity(group, member string) {
	if !common.RedisEnabled || common.RDB == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), pricingGroupActivityRedisTimeout)
	defer cancel()
	_ = common.RDB.ZRem(ctx, pricingGroupActivityRedisKey(group), member).Err()
}

// BeginPricingGroupActivity registers one in-flight relay request. The returned
// function must be deferred so normal completions disappear immediately; the
// short renewable lease also removes requests left behind by a crashed node.
func BeginPricingGroupActivity(group string, userID int, requestID string) func() {
	group = strings.TrimSpace(group)
	if group == "" {
		return func() {}
	}
	requestID = strings.TrimSpace(requestID)
	if requestID == "" {
		requestID = common.NewRequestId()
	}
	member := strconv.Itoa(userID) + "|" + common.NodeName + "|" + requestID

	refresh := func() {
		expiresAt := time.Now().Add(pricingGroupActivityLease).UnixMilli()
		refreshLocalPricingGroupActivity(group, member, userID, expiresAt)
		refreshRedisPricingGroupActivity(group, member, expiresAt)
	}
	refresh()

	done := make(chan struct{})
	var heartbeat sync.WaitGroup
	heartbeat.Add(1)
	go func() {
		defer heartbeat.Done()
		ticker := time.NewTicker(pricingGroupActivityHeartbeat)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				refresh()
			case <-done:
				return
			}
		}
	}()

	var finishOnce sync.Once
	return func() {
		finishOnce.Do(func() {
			close(done)
			heartbeat.Wait()
			removeLocalPricingGroupActivity(group, member)
			removeRedisPricingGroupActivity(group, member)
		})
	}
}

func GetPricingGroupActivity(groups []string) map[string]PricingGroupActivity {
	now := time.Now().UnixMilli()
	membersByGroup := make(map[string]map[string]int, len(groups))
	uniqueGroups := make([]string, 0, len(groups))
	for _, group := range groups {
		group = strings.TrimSpace(group)
		if group == "" {
			continue
		}
		if _, exists := membersByGroup[group]; exists {
			continue
		}
		membersByGroup[group] = make(map[string]int)
		uniqueGroups = append(uniqueGroups, group)
	}

	localPricingGroupActivity.Lock()
	for _, group := range uniqueGroups {
		entries := localPricingGroupActivity.groups[group]
		for member, entry := range entries {
			if entry.expiresAt <= now {
				delete(entries, member)
				continue
			}
			membersByGroup[group][member] = entry.userID
		}
		if len(entries) == 0 {
			delete(localPricingGroupActivity.groups, group)
		}
	}
	localPricingGroupActivity.Unlock()

	if common.RedisEnabled && common.RDB != nil && len(uniqueGroups) > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), pricingGroupActivityRedisTimeout)
		pipeline := common.RDB.Pipeline()
		commands := make(map[string]*redis.StringSliceCmd, len(uniqueGroups))
		for _, group := range uniqueGroups {
			key := pricingGroupActivityRedisKey(group)
			pipeline.ZRemRangeByScore(ctx, key, "-inf", strconv.FormatInt(now, 10))
			commands[group] = pipeline.ZRangeByScore(ctx, key, &redis.ZRangeBy{
				Min: strconv.FormatInt(now+1, 10),
				Max: "+inf",
			})
		}
		_, err := pipeline.Exec(ctx)
		cancel()
		if err == nil {
			for group, command := range commands {
				members, commandErr := command.Result()
				if commandErr != nil {
					continue
				}
				for _, member := range members {
					userText, _, _ := strings.Cut(member, "|")
					userID, _ := strconv.Atoi(userText)
					membersByGroup[group][member] = userID
				}
			}
		}
	}

	activityByGroup := make(map[string]PricingGroupActivity, len(uniqueGroups))
	for _, group := range uniqueGroups {
		users := make(map[int]struct{})
		for _, userID := range membersByGroup[group] {
			if userID > 0 {
				users[userID] = struct{}{}
			}
		}
		activityByGroup[group] = PricingGroupActivity{
			Users:       len(users),
			Connections: len(membersByGroup[group]),
		}
	}
	return activityByGroup
}
