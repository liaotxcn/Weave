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
	"fmt"
	"log"
	"reflect"
	"runtime"
	"sort"
	"sync"
	"time"
)

// InMemoryCache 内存缓存实现结构体
type InMemoryCache struct {
	mutex sync.RWMutex
	// 结构化对话存储
	conversations     map[string]interface{} // 存储结构化对话
	userConversations map[string][]string    // 存储用户与对话的关联
	userAccess        map[string]time.Time   // 记录用户最后访问时间，用于LRU
	userExpiry        map[string]time.Time   // 记录用户对话的过期时间
	maxUsers          int                    // 最大用户数
	maxMemoryMB       int                    // 最大内存使用限制（MB）
	ttl               time.Duration          // 默认过期时间
	defaultMaxUsr     int                    // 默认最大用户数
	defaultTTL        time.Duration          // 默认TTL
	cleanupTicker     *time.Ticker           // 定期清理过期对话的定时器
	stopChan          chan struct{}          // 停止定期清理的通道
}

// NewInMemoryCache 创建内存缓存实例
func NewInMemoryCache() *InMemoryCache {
	config := GetDefaultCacheConfig()
	return NewInMemoryCacheWithConfig(&config)
}

// NewInMemoryCacheWithConfig 基于配置创建内存缓存实例
func NewInMemoryCacheWithConfig(config *CacheConfig) *InMemoryCache {
	maxUsers := config.MaxUsers
	maxMemoryMB := config.MaxMemoryMB
	ttl := config.TTL
	cache := &InMemoryCache{
		conversations:     make(map[string]interface{}),
		userConversations: make(map[string][]string),
		userAccess:        make(map[string]time.Time),
		userExpiry:        make(map[string]time.Time),
		maxUsers:          maxUsers,
		maxMemoryMB:       maxMemoryMB,
		ttl:               ttl,
		defaultMaxUsr:     maxUsers,
		defaultTTL:        ttl,
		stopChan:          make(chan struct{}),
	}

	// 启动定期清理
	cache.startPeriodicCleanup()
	return cache
}

// updateUserMetadata 更新用户的访问时间和过期时间（内部方法，需在锁保护下调用）
func (mc *InMemoryCache) updateUserMetadata(userID string) {
	now := time.Now()
	mc.userAccess[userID] = now
	mc.userExpiry[userID] = now.Add(mc.ttl)
}

// checkUserLimit 检查用户数限制并进行LRU淘汰（内部方法，需在锁保护下调用）
func (mc *InMemoryCache) checkUserLimit(userID string) {
	if len(mc.userConversations) >= mc.maxUsers && !mc.userExists(userID) {
		mc.evictLRUUser()
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

// SaveConversation 保存结构化对话
func (mc *InMemoryCache) SaveConversation(ctx context.Context, conversation interface{}) error {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	// 获取对话ID和用户ID
	convValue := reflect.ValueOf(conversation)
	if convValue.Kind() == reflect.Ptr {
		convValue = convValue.Elem()
	}

	idField := convValue.FieldByName("ID")
	userIDField := convValue.FieldByName("UserID")

	if !idField.IsValid() || !userIDField.IsValid() {
		return fmt.Errorf("conversation must have ID and UserID fields")
	}

	convID := idField.String()
	userID := userIDField.String()

	// 保存对话
	mc.conversations[convID] = conversation

	// 关联用户和对话
	if _, exists := mc.userConversations[userID]; !exists {
		mc.userConversations[userID] = []string{}
	}

	// 检查对话是否已存在
	found := false
	for _, id := range mc.userConversations[userID] {
		if id == convID {
			found = true
			break
		}
	}

	// 如果不存在，添加到列表
	if !found {
		mc.userConversations[userID] = append(mc.userConversations[userID], convID)
	}

	// 如果用户数超过限制，删除最久未访问的用户
	mc.checkUserLimit(userID)

	// 检查内存使用，如果超过限制，清理最旧对话
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	usedMemoryMB := float64(memStats.Alloc) / (1024 * 1024)
	if usedMemoryMB > float64(mc.maxMemoryMB) {
		mc.cleanupOldestConversations()
	}

	// 更新用户的访问时间和过期时间
	mc.updateUserMetadata(userID)

	return nil
}

// LoadConversation 加载结构化对话
func (mc *InMemoryCache) LoadConversation(ctx context.Context, conversationID string) (interface{}, error) {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()

	conversation, exists := mc.conversations[conversationID]
	if !exists {
		return nil, fmt.Errorf("conversation not found")
	}

	return conversation, nil
}

// LoadUserConversations 加载用户的所有结构化对话
func (mc *InMemoryCache) LoadUserConversations(ctx context.Context, userID string) ([]interface{}, error) {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()

	convIDs, exists := mc.userConversations[userID]
	if !exists {
		return []interface{}{}, nil
	}

	conversations := make([]interface{}, 0, len(convIDs))
	for _, convID := range convIDs {
		if conv, exists := mc.conversations[convID]; exists {
			conversations = append(conversations, conv)
		}
	}

	return conversations, nil
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
	mc.userAccess = make(map[string]time.Time)
	mc.userExpiry = make(map[string]time.Time)
	mc.conversations = make(map[string]interface{})
	mc.userConversations = make(map[string][]string)
	return nil
}

// userExists 检查用户是否存在（内部方法，需在锁保护下调用）
func (mc *InMemoryCache) userExists(userID string) bool {
	_, exists := mc.userConversations[userID]
	return exists
}

// removeUser 删除用户及其所有相关数据（内部方法，需在锁保护下调用）
func (mc *InMemoryCache) removeUser(userID string) {
	delete(mc.userAccess, userID)
	delete(mc.userExpiry, userID)
	// 清理用户的结构化对话
	if convIDs, exists := mc.userConversations[userID]; exists {
		for _, convID := range convIDs {
			delete(mc.conversations, convID)
		}
		delete(mc.userConversations, userID)
	}
}

// evictLRUUser 淘汰最久未访问的用户（内部方法，需在锁保护下调用）
func (mc *InMemoryCache) evictLRUUser() {
	if len(mc.userConversations) == 0 {
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
	if len(mc.userConversations) < 10 {
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
		delete(mc.userAccess, userID)
		delete(mc.userExpiry, userID)
		// 清理用户的结构化对话
		if convIDs, exists := mc.userConversations[userID]; exists {
			for _, convID := range convIDs {
				delete(mc.conversations, convID)
			}
			delete(mc.userConversations, userID)
		}
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
		log.Printf("cleaned up %d expired conversations", expiredCount)
	}
}
