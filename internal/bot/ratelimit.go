package bot

import (
	"VPN-Telegram-bot/internal/admin"
	"sync"
	"time"
)

// RateLimiter implements per-user per-command in-memory rate limiting
// For production, can be swapped to Redis or similar store

type RateLimiter struct {
	mu       sync.Mutex
	lastCall map[int64]map[string]time.Time
	limits   map[string]time.Duration
}

func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		lastCall: make(map[int64]map[string]time.Time),
		limits: map[string]time.Duration{
			"/buy":             10 * time.Second,
			"/getkey":          5 * time.Second,
			"/subscriptions":   5 * time.Second,
			"/renew":           10 * time.Second,
			"/admin_broadcast": 30 * time.Second,
			// Add more commands as needed
		},
	}
}

// IsLimited returns true if user is rate-limited for this command
func (r *RateLimiter) IsLimited(userID int64, cmd string) bool {
	// Админ не лимитируется
	if userID == admin.AdminTelegramID {
		return false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now()
	if r.lastCall[userID] == nil {
		r.lastCall[userID] = make(map[string]time.Time)
	}
	limit, ok := r.limits[cmd]
	if !ok {
		limit = 2 * time.Second // default limit
	}
	last := r.lastCall[userID][cmd]
	if now.Sub(last) < limit {
		return true
	}
	r.lastCall[userID][cmd] = now
	return false
}
