package lib

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"time"

	"github.com/sebrandon1/go-toolbox/jsonstore"
)

type cachedInfoEntry struct {
	Info      *ClusterInfo `json:"info"`
	FetchedAt time.Time    `json:"fetched_at"`
}

type cacheFile struct {
	Environments map[string]cachedInfoEntry `json:"environments"`
}

type Cache struct {
	dir   string
	ttl   time.Duration
	store *jsonstore.Store[cacheFile]
}

func NewCache(dir string, ttl time.Duration) *Cache {
	return &Cache{
		dir: dir,
		ttl: ttl,
		// Compact JSON and the cache.json name keep the on-disk format
		// compatible with earlier releases.
		store: jsonstore.New[cacheFile](filepath.Join(dir, "cache.json"), jsonstore.WithIndent(false)),
	}
}

func (c *Cache) load() *cacheFile {
	cf, _ := c.store.LoadOr(cacheFile{Environments: make(map[string]cachedInfoEntry)})

	if cf.Environments == nil {
		cf.Environments = make(map[string]cachedInfoEntry)
	}

	return &cf
}

func (c *Cache) save(cf *cacheFile) error {
	if err := c.store.Save(*cf); err != nil {
		return fmt.Errorf("saving cache: %w", err)
	}

	return nil
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
		slog.Default().Warn("cache save failed", "err", err)
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
		slog.Default().Warn("cache save failed", "err", err)
	}
}

func (c *Cache) Clear() error {
	return c.store.Delete()
}
