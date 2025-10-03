package repl

import (
	"crypto/md5"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// PerformanceOptimizer handles caching and optimization for REPL commands
type PerformanceOptimizer struct {
	mu             sync.RWMutex
	commandCache   map[string]*CachedResult
	parseCache     map[string]*ParsedCommand
	enabled        bool
	maxCacheSize   int
	cacheHitCount  int
	cacheMissCount int
}

// CachedResult stores the result of a command execution
type CachedResult struct {
	Result    interface{}
	Error     error
	Timestamp time.Time
	HitCount  int
}

// ParsedCommand stores pre-parsed command information
type ParsedCommand struct {
	Language  string
	Command   string
	Arguments []string
	Timestamp time.Time
}

// NewPerformanceOptimizer creates a new performance optimizer
func NewPerformanceOptimizer(enabled bool) *PerformanceOptimizer {
	optimizer := &PerformanceOptimizer{
		commandCache: make(map[string]*CachedResult),
		parseCache:   make(map[string]*ParsedCommand),
		enabled:      enabled,
		maxCacheSize: 1000,
	}

	return optimizer
}

// CacheCommand caches the result of a command execution
func (po *PerformanceOptimizer) CacheCommand(command string, result interface{}, err error) {
	if !po.enabled {
		return
	}

	po.mu.Lock()
	defer po.mu.Unlock()

	// Generate cache key
	key := po.generateCacheKey(command)

	// Clean cache if it's getting too large
	if len(po.commandCache) >= po.maxCacheSize {
		po.cleanOldestEntries()
	}

	po.commandCache[key] = &CachedResult{
		Result:    result,
		Error:     err,
		Timestamp: time.Now(),
		HitCount:  0,
	}
}

// GetCachedCommand retrieves a cached command result
func (po *PerformanceOptimizer) GetCachedCommand(command string) (interface{}, error, bool) {
	if !po.enabled {
		return nil, nil, false
	}

	po.mu.RLock()
	defer po.mu.RUnlock()

	key := po.generateCacheKey(command)

	if cached, exists := po.commandCache[key]; exists {
		// Check if cache entry is still valid (not older than 5 minutes)
		if time.Since(cached.Timestamp) < 5*time.Minute {
			cached.HitCount++
			po.cacheHitCount++
			return cached.Result, cached.Error, true
		}

		// Clean expired entry
		delete(po.commandCache, key)
	}

	po.cacheMissCount++
	return nil, nil, false
}

// PreParseCommand parses and caches command structure for faster execution
func (po *PerformanceOptimizer) PreParseCommand(command string) *ParsedCommand {
	if !po.enabled {
		return po.parseCommandBasic(command)
	}

	po.mu.RLock()
	key := po.generateCacheKey(command)
	if parsed, exists := po.parseCache[key]; exists {
		po.mu.RUnlock()
		return parsed
	}
	po.mu.RUnlock()

	// Parse command
	parsed := po.parseCommandBasic(command)

	po.mu.Lock()
	po.parseCache[key] = parsed
	po.mu.Unlock()

	return parsed
}

// parseCommandBasic performs basic command parsing
func (po *PerformanceOptimizer) parseCommandBasic(command string) *ParsedCommand {
	// Simple parsing logic - can be enhanced
	parsed := &ParsedCommand{
		Timestamp: time.Now(),
	}

	if strings.HasPrefix(command, "lua.") {
		parsed.Language = "lua"
		parsed.Command = strings.TrimPrefix(command, "lua.")
	} else if strings.HasPrefix(command, "py.") || strings.HasPrefix(command, "python.") {
		prefix := "python."
		if strings.HasPrefix(command, "py.") {
			prefix = "py."
		} else {
			prefix = "python."
		}
		parsed.Language = "python"
		parsed.Command = strings.TrimPrefix(command, prefix)
	} else {
		parsed.Language = ""
		parsed.Command = command
	}

	return parsed
}

// GetCacheStats returns cache performance statistics
func (po *PerformanceOptimizer) GetCacheStats() map[string]interface{} {
	po.mu.RLock()
	defer po.mu.RUnlock()

	totalRequests := po.cacheHitCount + po.cacheMissCount
	hitRate := 0.0
	if totalRequests > 0 {
		hitRate = float64(po.cacheHitCount) / float64(totalRequests) * 100
	}

	return map[string]interface{}{
		"enabled":         po.enabled,
		"cache_hits":      po.cacheHitCount,
		"cache_misses":    po.cacheMissCount,
		"hit_rate":        hitRate,
		"cache_size":      len(po.commandCache),
		"parsed_commands": len(po.parseCache),
	}
}

// ClearCache clears all cached data
func (po *PerformanceOptimizer) ClearCache() {
	po.mu.Lock()
	defer po.mu.Unlock()

	po.commandCache = make(map[string]*CachedResult)
	po.parseCache = make(map[string]*ParsedCommand)
	po.cacheHitCount = 0
	po.cacheMissCount = 0
}

// SetEnabled enables or disables the performance optimizer
func (po *PerformanceOptimizer) SetEnabled(enabled bool) {
	po.enabled = enabled
}

// IsEnabled returns whether the optimizer is enabled
func (po *PerformanceOptimizer) IsEnabled() bool {
	return po.enabled
}

// generateCacheKey generates a unique key for caching
func (po *PerformanceOptimizer) generateCacheKey(command string) string {
	hash := md5.Sum([]byte(command))
	return fmt.Sprintf("%x", hash)
}

// cleanOldestEntries removes old cache entries to maintain cache size
func (po *PerformanceOptimizer) cleanOldestEntries() {
	// Remove 20% of oldest entries
	entriesToRemove := len(po.commandCache) / 5

	type entry struct {
		key       string
		timestamp time.Time
	}

	var entries []entry
	for key, cached := range po.commandCache {
		entries = append(entries, entry{key: key, timestamp: cached.Timestamp})
	}

	// Sort by timestamp (oldest first)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].timestamp.Before(entries[j].timestamp)
	})

	// Remove oldest entries
	for i := 0; i < entriesToRemove && i < len(entries); i++ {
		delete(po.commandCache, entries[i].key)
	}
}

// Cleanup releases all resources used by the performance optimizer
func (po *PerformanceOptimizer) Cleanup() {
	po.ClearCache()
	po.enabled = false
}
