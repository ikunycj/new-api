package service

import (
	"context"
	"encoding/base64"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

const (
	pricingGroupActivityLease        = 30 * time.Second
	pricingGroupActivityHeartbeat    = 10 * time.Second
	pricingGroupActivityRedisTimeout = 500 * time.Millisecond
	pricingGroupActivityKeyPrefix    = "routing:pricing-group:activity:"
	pricingGroupActivityContextKey   = "pricing_group_activity_session"
)

type PricingGroupActivity struct {
	Users       int `json:"users"`
	Connections int `json:"connections"`
}

type pricingGroupActivityEntry struct {
	userID    int
	expiresAt int64
}

type pricingGroupActivityRedisOperation struct {
	group     string
	member    string
	expiresAt int64
	remove    bool
}

type pricingGroupActivitySession struct {
	sync.Mutex
	group      string
	member     string
	userID     int
	done       chan struct{}
	heartbeat  sync.WaitGroup
	finishOnce sync.Once
	finished   bool
}

var localPricingGroupActivity = struct {
	sync.Mutex
	groups map[string]map[string]pricingGroupActivityEntry
}{groups: make(map[string]map[string]pricingGroupActivityEntry)}

var (
	pricingGroupActivityRedisOnce       sync.Once
	pricingGroupActivityRedisOperations = make(chan pricingGroupActivityRedisOperation, 4096)
)

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

func enqueuePricingGroupActivityRedisOperation(operation pricingGroupActivityRedisOperation) {
	if !common.RedisEnabled || common.RDB == nil {
		return
	}
	pricingGroupActivityRedisOnce.Do(func() { go runPricingGroupActivityRedisWriter() })
	select {
	case pricingGroupActivityRedisOperations <- operation:
	default:
		common.SysError("pricing group activity Redis queue is full")
	}
}

func runPricingGroupActivityRedisWriter() {
	for first := range pricingGroupActivityRedisOperations {
		pending := map[string]pricingGroupActivityRedisOperation{
			first.group + "\x00" + first.member: first,
		}
		for len(pending) < 512 {
			select {
			case operation := <-pricingGroupActivityRedisOperations:
				pending[operation.group+"\x00"+operation.member] = operation
			default:
				goto flush
			}
		}
	flush:
		if !common.RedisEnabled || common.RDB == nil {
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), pricingGroupActivityRedisTimeout)
		pipeline := common.RDB.TxPipeline()
		for _, operation := range pending {
			key := pricingGroupActivityRedisKey(operation.group)
			if operation.remove {
				pipeline.ZRem(ctx, key, operation.member)
				continue
			}
			pipeline.ZAdd(ctx, key, &redis.Z{
				Score:  float64(operation.expiresAt),
				Member: operation.member,
			})
			pipeline.Expire(ctx, key, 2*pricingGroupActivityLease)
		}
		_, _ = pipeline.Exec(ctx)
		cancel()
	}
}

func (session *pricingGroupActivitySession) refreshLocked() {
	if session.group == "" || session.finished {
		return
	}
	expiresAt := time.Now().Add(pricingGroupActivityLease).UnixMilli()
	refreshLocalPricingGroupActivity(session.group, session.member, session.userID, expiresAt)
	enqueuePricingGroupActivityRedisOperation(pricingGroupActivityRedisOperation{
		group: session.group, member: session.member, expiresAt: expiresAt,
	})
}

func (session *pricingGroupActivitySession) refresh() {
	session.Lock()
	defer session.Unlock()
	session.refreshLocked()
}

func (session *pricingGroupActivitySession) move(group string) {
	group = strings.TrimSpace(group)
	session.Lock()
	defer session.Unlock()
	if session.finished || group == session.group {
		return
	}
	if session.group != "" {
		removeLocalPricingGroupActivity(session.group, session.member)
		enqueuePricingGroupActivityRedisOperation(pricingGroupActivityRedisOperation{
			group: session.group, member: session.member, remove: true,
		})
	}
	session.group = group
	session.refreshLocked()
}

func (session *pricingGroupActivitySession) finish() {
	session.finishOnce.Do(func() {
		close(session.done)
		session.heartbeat.Wait()
		session.Lock()
		defer session.Unlock()
		session.finished = true
		if session.group == "" {
			return
		}
		removeLocalPricingGroupActivity(session.group, session.member)
		enqueuePricingGroupActivityRedisOperation(pricingGroupActivityRedisOperation{
			group: session.group, member: session.member, remove: true,
		})
	})
}

// BeginPricingGroupActivity registers one in-flight relay request on the Gin
// context. The returned function must be deferred so normal completions
// disappear immediately; the renewable lease also removes requests left behind
// by a crashed node.
func BeginPricingGroupActivity(ctx *gin.Context, group string, userID int, requestID string) func() {
	group = strings.TrimSpace(group)
	if group == "" {
		return func() {}
	}
	requestID = strings.TrimSpace(requestID)
	if requestID == "" {
		requestID = common.NewRequestId()
	}
	session := &pricingGroupActivitySession{
		group:  group,
		member: strconv.Itoa(userID) + "|" + common.NodeName + "|" + requestID,
		userID: userID,
		done:   make(chan struct{}),
	}
	if ctx != nil {
		ctx.Set(pricingGroupActivityContextKey, session)
	}
	session.refresh()

	session.heartbeat.Add(1)
	go func() {
		defer session.heartbeat.Done()
		ticker := time.NewTicker(pricingGroupActivityHeartbeat)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				session.refresh()
			case <-session.done:
				return
			}
		}
	}()
	return session.finish
}

// UpdatePricingGroupActivity moves the in-flight request when automatic routing
// selects a different pricing group during a retry.
func UpdatePricingGroupActivity(ctx *gin.Context, group string) {
	if ctx == nil {
		return
	}
	value, exists := ctx.Get(pricingGroupActivityContextKey)
	if !exists {
		return
	}
	session, ok := value.(*pricingGroupActivitySession)
	if ok {
		session.move(group)
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

func GetTotalPricingGroupConnections(groups []string) int {
	total := 0
	for _, activity := range GetPricingGroupActivity(groups) {
		total += activity.Connections
	}
	return total
}
