package lib

import (
	"testing"
	"time"
)

func TestCacheSetAndGet(t *testing.T) {
	dir := t.TempDir()
	cache := NewCache(dir, 60*time.Second)

	info := &ClusterInfo{
		Environment: "testenv",
		PlanName:    "testenv",
		InstallerIP: "192.168.1.100",
		Nodes: []NodeInfo{
			{Name: "testenv-installer", Status: StatusUp, IP: "192.168.1.100"},
		},
	}

	cache.SetInfo("testenv", info)

	got, ok := cache.GetInfo("testenv")
	if !ok {
		t.Fatal("Expected cache hit, got miss")
	}

	if got.InstallerIP != "192.168.1.100" {
		t.Errorf("Expected installer IP 192.168.1.100, got %s", got.InstallerIP)
	}

	if len(got.Nodes) != 1 {
		t.Errorf("Expected 1 node, got %d", len(got.Nodes))
	}
}

func TestCacheMiss(t *testing.T) {
	dir := t.TempDir()
	cache := NewCache(dir, 60*time.Second)

	_, ok := cache.GetInfo("nonexistent")
	if ok {
		t.Fatal("Expected cache miss, got hit")
	}
}

func TestCacheExpiry(t *testing.T) {
	dir := t.TempDir()
	cache := NewCache(dir, 1*time.Millisecond)

	info := &ClusterInfo{Environment: "testenv"}
	cache.SetInfo("testenv", info)

	time.Sleep(5 * time.Millisecond)

	_, ok := cache.GetInfo("testenv")
	if ok {
		t.Fatal("Expected cache miss after TTL, got hit")
	}
}

func TestCacheSetMultiple(t *testing.T) {
	dir := t.TempDir()
	cache := NewCache(dir, 60*time.Second)

	entries := map[string]*ClusterInfo{
		"env1": {Environment: "env1", InstallerIP: "192.168.1.1"},
		"env2": {Environment: "env2", InstallerIP: "192.168.1.2"},
	}

	cache.SetMultipleInfo(entries)

	got1, ok1 := cache.GetInfo("env1")
	if !ok1 {
		t.Fatal("Expected cache hit for env1")
	}

	if got1.InstallerIP != "192.168.1.1" {
		t.Errorf("Expected env1 IP 192.168.1.1, got %s", got1.InstallerIP)
	}

	got2, ok2 := cache.GetInfo("env2")
	if !ok2 {
		t.Fatal("Expected cache hit for env2")
	}

	if got2.InstallerIP != "192.168.1.2" {
		t.Errorf("Expected env2 IP 192.168.1.2, got %s", got2.InstallerIP)
	}
}

func TestCacheClear(t *testing.T) {
	dir := t.TempDir()
	cache := NewCache(dir, 60*time.Second)

	cache.SetInfo("testenv", &ClusterInfo{Environment: "testenv"})

	if err := cache.Clear(); err != nil {
		t.Fatalf("Clear failed: %v", err)
	}

	_, ok := cache.GetInfo("testenv")
	if ok {
		t.Fatal("Expected cache miss after clear, got hit")
	}
}

func TestCacheEmptyDir(t *testing.T) {
	dir := t.TempDir()
	cache := NewCache(dir, 60*time.Second)

	_, ok := cache.GetInfo("anything")
	if ok {
		t.Fatal("Expected cache miss on empty cache dir")
	}
}
