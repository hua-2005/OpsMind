// Token 黑名单 — 退出登录后短期失效，防止 token 重用
package auth

import (
	"sync"
	"time"
)

type TokenBlacklist struct {
	mu    sync.RWMutex
	store map[string]time.Time
}

var blacklist = &TokenBlacklist{store: make(map[string]time.Time)}

func init() {
	// 每 5 分钟清理过期条目
	go func() {
		for {
			time.Sleep(5 * time.Minute)
			blacklist.cleanup()
		}
	}()
}

// BlacklistToken 将 token 加入黑名单，有效期取 token 剩余时间
func BlacklistToken(token string) {
	blacklist.mu.Lock()
	blacklist.store[token] = time.Now().Add(24 * time.Hour)
	blacklist.mu.Unlock()
}

// IsBlacklisted 检查 token 是否在黑名单中
func IsBlacklisted(token string) bool {
	blacklist.mu.RLock()
	_, ok := blacklist.store[token]
	blacklist.mu.RUnlock()
	return ok
}

func (b *TokenBlacklist) cleanup() {
	b.mu.Lock()
	now := time.Now()
	for k, v := range b.store {
		if now.After(v) {
			delete(b.store, k)
		}
	}
	b.mu.Unlock()
}
