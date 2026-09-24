package lib

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// AuditEntry records the outcome of one CLI operation that changes remote or
// local state.
type AuditEntry struct {
	Timestamp   time.Time         `json:"timestamp"`
	UserID      string            `json:"user_id"`
	Operation   string            `json:"operation"`
	Environment string            `json:"environment"`
	Parameters  map[string]string `json:"parameters,omitempty"`
	Result      string            `json:"result"`
	Error       string            `json:"error,omitempty"`
}

// AuditLog appends JSON Lines entries to a local file.
type AuditLog struct {
	path string
	mu   sync.Mutex
}

func NewAuditLog(path string) (*AuditLog, error) {
	if path == "" {
		return nil, errors.New("audit log path is empty")
	}

	return &AuditLog{path: path}, nil
}

func (l *AuditLog) Record(entry AuditEntry) error {
	if l == nil {
		return errors.New("audit log is not initialized")
	}
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC()
	} else {
		entry.Timestamp = entry.Timestamp.UTC()
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("encoding audit entry: %w", err)
	}
	data = append(data, '\n')

	l.mu.Lock()
	defer l.mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(l.path), 0o750); err != nil {
		return fmt.Errorf("creating audit log directory: %w", err)
	}

	file, err := os.OpenFile(l.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("opening audit log: %w", err)
	}

	if err := file.Chmod(0o600); err != nil {
		_ = file.Close()
		return fmt.Errorf("setting audit log permissions: %w", err)
	}

	written, writeErr := file.Write(data)
	if writeErr == nil && written != len(data) {
		writeErr = errors.New("short write")
	}
	if writeErr == nil {
		writeErr = file.Sync()
	}
	closeErr := file.Close()
	if writeErr != nil || closeErr != nil {
		return errors.Join(writeErr, closeErr)
	}

	return nil
}

func (l *AuditLog) ReadLast(limit int) ([]AuditEntry, error) {
	if l == nil {
		return nil, errors.New("audit log is not initialized")
	}
	if limit <= 0 {
		return []AuditEntry{}, nil
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	file, err := os.Open(l.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []AuditEntry{}, nil
		}
		return nil, fmt.Errorf("opening audit log: %w", err)
	}
	defer file.Close()

	entries := make([]AuditEntry, 0, limit)
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	line := 0
	for scanner.Scan() {
		line++
		var entry AuditEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			return nil, fmt.Errorf("decoding audit entry on line %d: %w", line, err)
		}

		if len(entries) < limit {
			entries = append(entries, entry)
		} else {
			copy(entries, entries[1:])
			entries[len(entries)-1] = entry
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading audit log: %w", err)
	}

	return entries, nil
}
