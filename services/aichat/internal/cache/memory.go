/*
 * Copyright 2025 CloudWeGo Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package cache

import (
	"context"
	"log"
	"runtime"
	"sort"
	"sync"
	"time"

	"github.com/cloudwego/eino/schema"
)

// InMemoryCache 内存缓存实现结构体
type InMemoryCache struct {
	mutex         sync.RWMutex
	history       map[string][]*schema.Message
	userAccess    map[string]time.Time // 记录用户最后访问时间，用于LRU
	userExpiry    map[string]time.Time // 记录用户对话的过期时间
	maxMessages   int                  // 每条对话最大消息数
	maxUsers      int                  // 最大用户数
	maxMemoryMB   int                  // 最大内存使用限制（MB）
	ttl           time.Duration        // 默认过期时间
	defaultMaxMsg int                  // 默认最大消息数
	defaultMaxUsr int                  // 默认最大用户数
	defaultTTL    time.Duration        // 默认TTL
	cleanupTicker *time.Ticker         // 定期清理过期对话的定时器
	stopChan      chan struct{}        // 停止定期清理的通道
}

// NewInMemoryCache 创建内存缓存实例
func NewInMemoryCache() *InMemoryCache {
	// 设置默认值：每条对话最多100条消息，最多1000个用户，默认过期时间3小时，最大内存使用100MB
	cache := &InMemoryCache{
		history:       make(map[string][]*schema.Message),
		userAccess:    make(map[string]time.Time),
		userExpiry:    make(map[string]time.Time),
		maxMessages:   100,
		maxUsers:      1000,
		maxMemoryMB:   100,
		ttl:           3 * time.Hour,
		defaultMaxMsg: 100,
		defaultMaxUsr: 1000,
		defaultTTL:    3 * time.Hour,
		stopChan:      make(chan struct{}),
	}

	// 启动定期清理（每5分钟清理一次过期对话、检查内存使用）
	cache.startPeriodicCleanup()
	return cache
}

// updateUserMetadata 更新用户的访问时间和过期时间（内部方法，需在锁保护下调用）
func (mc *InMemoryCache) updateUserMetadata(userID string) {
	now := time.Now()
	mc.userAccess[userID] = now
	mc.userExpiry[userID] = now.Add(mc.ttl)
}

// limitHistoryLength 限制对话历史长度（内部方法，需在锁保护下调用）
func (mc *InMemoryCache) limitHistoryLength(history []*schema.Message) []*schema.Message {
	if len(history) > mc.maxMessages {
		// 只保留最近的消息
		return history[len(history)-mc.maxMessages:]
	}
	return history
}

// checkUserLimit 检查用户数限制并进行LRU淘汰（内部方法，需在锁保护下调用）
func (mc *InMemoryCache) checkUserLimit(userID string) {
	if len(mc.history) >= mc.maxUsers && !mc.userExists(userID) {
		mc.evictLRUUser()
	}
}

// SetMaxMessages 设置每条对话的最大消息数
func (mc *InMemoryCache) SetMaxMessages(max int) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()
	if max > 0 {
		mc.maxMessages = max
	} else {
		mc.maxMessages = mc.defaultMaxMsg
	}
}

// SetMaxUsers 设置最大用户数
func (mc *InMemoryCache) SetMaxUsers(max int) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()
	if max > 0 {
		mc.maxUsers = max
	} else {
		mc.maxUsers = mc.defaultMaxUsr
	}
}

// SetTTL 设置默认过期时间
func (mc *InMemoryCache) SetTTL(ttl time.Duration) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()
	if ttl > 0 {
		mc.ttl = ttl
	} else {
		mc.ttl = mc.defaultTTL
	}
}

// SaveChatHistory 保存对话历史到内存
func (mc *InMemoryCache) SaveChatHistory(ctx context.Context, userID string, history []*schema.Message) error {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	// 限制对话历史长度
	history = mc.limitHistoryLength(history)

	// 更新用户访问时间和过期时间
	mc.updateUserMetadata(userID)

	// 如果用户数超过限制，删除最久未访问的用户
	mc.checkUserLimit(userID)

	// 检查内存使用，如果超过限制，清理最旧对话
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	usedMemoryMB := float64(memStats.Alloc) / (1024 * 1024)
	if usedMemoryMB > float64(mc.maxMemoryMB) {
		mc.cleanupOldestConversations()
	}

	mc.history[userID] = history
	return nil
}

// LoadChatHistory 从内存加载对话历史
func (mc *InMemoryCache) LoadChatHistory(ctx context.Context, userID string) ([]*schema.Message, error) {
	mc.mutex.RLock()

	// 检查用户是否存在
	_, exists := mc.history[userID]
	if !exists {
		mc.mutex.RUnlock()
		return []*schema.Message{}, nil
	}

	// 检查对话是否过期
	now := time.Now()
	if expiry, ok := mc.userExpiry[userID]; ok && now.After(expiry) {
		// 对话已过期，需要删除
		mc.mutex.RUnlock()
		mc.mutex.Lock()
		mc.removeUser(userID)
		mc.mutex.Unlock()
		return []*schema.Message{}, nil
	}

	// 更新用户访问时间和过期时间
	mc.mutex.RUnlock()
	mc.mutex.Lock()
	mc.userAccess[userID] = now
	mc.userExpiry[userID] = now.Add(mc.ttl) // 刷新过期时间
	loadedHistory := mc.history[userID]     // 重新获取历史
	mc.mutex.Unlock()

	// 返回历史的副本，避免外部修改影响内部存储
	result := make([]*schema.Message, len(loadedHistory))
	copy(result, loadedHistory)
	return result, nil
}

// AddMessageToHistory 添加消息到对话历史
func (mc *InMemoryCache) AddMessageToHistory(ctx context.Context, userID string, messages ...*schema.Message) error {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	// 检查用户是否存在，不存在则检查是否需要删除最久未访问用户
	if !mc.userExists(userID) {
		mc.checkUserLimit(userID)
		// 为新用户初始化
		mc.history[userID] = []*schema.Message{}
	}

	// 获取当前历史并添加新消息
	currentHistory := mc.history[userID]
	currentHistory = append(currentHistory, messages...)

	// 限制对话历史长度
	currentHistory = mc.limitHistoryLength(currentHistory)

	// 更新历史和用户元数据
	mc.history[userID] = currentHistory
	mc.updateUserMetadata(userID)

	return nil
}

// Close 关闭内存缓存（清理资源）
func (mc *InMemoryCache) Close() error {
	// 停止定期清理
	if mc.cleanupTicker != nil {
		mc.cleanupTicker.Stop()
	}
	close(mc.stopChan)

	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	// 清理内存
	mc.history = make(map[string][]*schema.Message)
	mc.userAccess = make(map[string]time.Time)
	mc.userExpiry = make(map[string]time.Time)
	return nil
}

// userExists 检查用户是否存在（内部方法，需在锁保护下调用）
func (mc *InMemoryCache) userExists(userID string) bool {
	_, exists := mc.history[userID]
	return exists
}

// removeUser 删除用户及其所有相关数据（内部方法，需在锁保护下调用）
func (mc *InMemoryCache) removeUser(userID string) {
	delete(mc.history, userID)
	delete(mc.userAccess, userID)
	delete(mc.userExpiry, userID)
}

// evictLRUUser 淘汰最久未访问的用户（内部方法，需在锁保护下调用）
func (mc *InMemoryCache) evictLRUUser() {
	if len(mc.history) == 0 {
		return
	}

	// 找到最久未访问的用户
	var oldestUser string
	var oldestTime time.Time
	first := true

	for user, accessTime := range mc.userAccess {
		if first || accessTime.Before(oldestTime) {
			oldestUser = user
			oldestTime = accessTime
			first = false
		}
	}

	// 删除该用户
	if oldestUser != "" {
		mc.removeUser(oldestUser)
	}
}

// startPeriodicCleanup 启动定期清理过期对话的任务
func (mc *InMemoryCache) startPeriodicCleanup() {
	// 每5分钟清理一次过期对话、检查内存使用
	mc.cleanupTicker = time.NewTicker(5 * time.Minute)
	go func() {
		for {
			select {
			case <-mc.cleanupTicker.C:
				mc.cleanupExpired()
				mc.checkMemoryUsage()
			case <-mc.stopChan:
				return
			}
		}
	}()
}

// checkMemoryUsage 检查内存使用情况并在必要时清理
func (mc *InMemoryCache) checkMemoryUsage() {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	// 计算当前内存使用
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	usedMemoryMB := float64(memStats.Alloc) / (1024 * 1024)

	// 如果内存使用超过限制，清理最旧的对话
	if usedMemoryMB > float64(mc.maxMemoryMB) {
		log.Printf("memory usage %.2fMB exceeds limit %dMB, starting cleanup", usedMemoryMB, mc.maxMemoryMB)
		mc.cleanupOldestConversations()
	}
}

// cleanupOldestConversations 清理最旧对话
func (mc *InMemoryCache) cleanupOldestConversations() {
	// 如果用户数量较少，不需要清理
	if len(mc.history) < 10 {
		return
	}

	// 按最后访问时间排序用户
	type userWithTime struct {
		userID string
		time   time.Time
	}

	userTimes := make([]userWithTime, 0, len(mc.userAccess))
	for userID, accessTime := range mc.userAccess {
		userTimes = append(userTimes, userWithTime{userID: userID, time: accessTime})
	}

	// 按访问时间排序（最早访问的在前）
	sort.Slice(userTimes, func(i, j int) bool {
		return userTimes[i].time.Before(userTimes[j].time)
	})

	// 清理最旧的20%对话
	cleanupCount := len(userTimes) / 5
	if cleanupCount < 1 {
		cleanupCount = 1
	}

	for i := 0; i < cleanupCount && i < len(userTimes); i++ {
		userID := userTimes[i].userID
		delete(mc.history, userID)
		delete(mc.userAccess, userID)
		delete(mc.userExpiry, userID)
		log.Printf("deleted oldest conversation for user %s to free up memory", userID)
	}
}

// cleanupExpired 清理过期对话（内部方法，无需锁保护）
func (mc *InMemoryCache) cleanupExpired() {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	now := time.Now()
	expiredCount := 0

	// 遍历所有用户，删除过期对话
	for userID, expiryTime := range mc.userExpiry {
		if now.After(expiryTime) {
			mc.removeUser(userID)
			expiredCount++
		}
	}

	if expiredCount > 0 {
		// 可添加日志记录
		// log.Printf("清理了 %d 条过期对话记录", expiredCount)
	}
}
