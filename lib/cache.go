package lib

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type cachedInfoEntry struct {
	Info      *ClusterInfo `json:"info"`
	FetchedAt time.Time    `json:"fetched_at"`
}

type cacheFile struct {
	Environments map[string]cachedInfoEntry `json:"environments"`
}

type Cache struct {
	dir string
	ttl time.Duration
}

func NewCache(dir string, ttl time.Duration) *Cache {
	return &Cache{dir: dir, ttl: ttl}
}

func (c *Cache) path() string {
	return filepath.Join(c.dir, "cache.json")
}

func (c *Cache) load() *cacheFile {
	data, err := os.ReadFile(c.path())
	if err != nil {
		return &cacheFile{Environments: make(map[string]cachedInfoEntry)}
	}

	var cf cacheFile
	if err := json.Unmarshal(data, &cf); err != nil {
		return &cacheFile{Environments: make(map[string]cachedInfoEntry)}
	}

	if cf.Environments == nil {
		cf.Environments = make(map[string]cachedInfoEntry)
	}

	return &cf
}

func (c *Cache) save(cf *cacheFile) error {
	if err := os.MkdirAll(c.dir, 0o750); err != nil {
		return fmt.Errorf("creating cache directory: %w", err)
	}

	data, err := json.Marshal(cf)
	if err != nil {
		return fmt.Errorf("marshaling cache: %w", err)
	}

	return os.WriteFile(c.path(), data, 0o600)
}

func (c *Cache) GetInfo(env string) (*ClusterInfo, bool) {
	cf := c.load()

	entry, ok := cf.Environments[env]
	if !ok {
		return nil, false
	}

	if time.Since(entry.FetchedAt) > c.ttl {
		return nil, false
	}

	return entry.Info, true
}

func (c *Cache) SetInfo(env string, info *ClusterInfo) {
	cf := c.load()

	cf.Environments[env] = cachedInfoEntry{
		Info:      info,
		FetchedAt: time.Now(),
	}

	if err := c.save(cf); err != nil {
		fmt.Fprintf(os.Stderr, "warning: cache save failed: %v\n", err)
	}
}

func (c *Cache) SetMultipleInfo(entries map[string]*ClusterInfo) {
	cf := c.load()

	now := time.Now()

	for env, info := range entries {
		cf.Environments[env] = cachedInfoEntry{
			Info:      info,
			FetchedAt: now,
		}
	}

	if err := c.save(cf); err != nil {
		fmt.Fprintf(os.Stderr, "warning: cache save failed: %v\n", err)
	}
}

func (c *Cache) Clear() error {
	return os.Remove(c.path())
}
