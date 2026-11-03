package lib

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	// logPrefixSize is how much of the log identifies a run. The first lines
	// carry the run's start timestamps, so a new run's prefix differs from
	// the previous run's even though the task names are the same.
	logPrefixSize = 4096

	// playbookTailSize is how much of the end of the log WaitForPlaybook keeps
	// for the recap and the failing task.
	playbookTailSize = 64 * 1024

	maxFatalExcerpt = 2000
)

// logPrefixTimeout bounds a prefix read. The endpoint streams while a
// playbook runs, so a short log can hold the response open indefinitely.
var logPrefixTimeout = 15 * time.Second

var recapHostLine = regexp.MustCompile(`^(\S+)\s*:\s*ok=\d+.*`)
var recapCounter = regexp.MustCompile(`\b(failed|unreachable)=(\d+)`)

// LogPrefix returns up to the first 4 KiB of the environment's Ansible log.
// It returns the partial prefix if the log is still streaming.
func (c *Client) LogPrefix(ctx context.Context, env string) ([]byte, error) {
	resp, err := c.WithTimeout(logPrefixTimeout).getRaw(ctx, c.endpointURL(endpointLog, env))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch log for %s: %w", env, err)
	}
	defer resp.Body.Close()

	prefix, err := io.ReadAll(io.LimitReader(resp.Body, logPrefixSize))
	// A timeout after some output means the log is still streaming: keep what
	// arrived.
	if err != nil && (len(prefix) == 0 || !isTimeout(err)) {
		return nil, fmt.Errorf("failed to read log for %s: %w", env, err)
	}

	return prefix, nil
}

// WaitForPlaybook waits for a playbook run that started after baseline was
// taken (see LogPrefix), follows it until PLAY RECAP, and reports whether it
// failed. Progress is written to w. The context bounds the whole wait.
func (c *Client) WaitForPlaybook(ctx context.Context, env string, baseline []byte, pollInterval time.Duration, w io.Writer) error {
	if err := c.waitForNewRun(ctx, env, baseline, pollInterval); err != nil {
		return err
	}

	tail := &tailWriter{max: playbookTailSize}
	if err := c.FollowLog(ctx, env, io.MultiWriter(w, tail), pollInterval); err != nil {
		return fmt.Errorf("following the playbook log: %w", err)
	}

	return ParsePlayRecap(tail.buf)
}

func (c *Client) waitForNewRun(ctx context.Context, env string, baseline []byte, pollInterval time.Duration) error {
	for {
		prefix, err := c.LogPrefix(ctx, env)
		if err == nil && len(prefix) > 0 && !bytes.Equal(prefix, commonPrefix(baseline, len(prefix))) {
			return nil
		}
		if err != nil {
			c.logger().Debug("waiting for the new playbook run", "env", env, "err", err)
		}

		timer := time.NewTimer(pollInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return fmt.Errorf("waiting for the playbook to start: %w", ctx.Err())
		case <-timer.C:
		}
	}
}

// commonPrefix returns the first n bytes of b, or all of b if it is shorter.
// A run that has only written part of its prefix is compared against the same
// length of the baseline.
func commonPrefix(b []byte, n int) []byte {
	if len(b) < n {
		return b
	}

	return b[:n]
}

// ParsePlayRecap checks the PLAY RECAP at the end of an Ansible log. It
// returns an error naming the hosts with failed or unreachable tasks and the
// last failing task, or an error if the log has no recap.
func ParsePlayRecap(log []byte) error {
	text := string(log)
	idx := strings.LastIndex(text, "PLAY RECAP")
	if idx < 0 {
		return errors.New("playbook log has no PLAY RECAP")
	}

	var failedHosts []string
	for _, line := range strings.Split(text[idx:], "\n")[1:] {
		line = strings.TrimSpace(line)
		host := recapHostLine.FindStringSubmatch(line)
		if host == nil {
			continue
		}
		for _, counter := range recapCounter.FindAllStringSubmatch(line, -1) {
			if n, _ := strconv.Atoi(counter[2]); n > 0 {
				failedHosts = append(failedHosts, host[1])
				break
			}
		}
	}

	if len(failedHosts) == 0 {
		return nil
	}

	msg := fmt.Sprintf("playbook failed on %s", strings.Join(failedHosts, ", "))
	if excerpt := lastFatal(text[:idx]); excerpt != "" {
		msg += ":\n" + excerpt
	}

	return errors.New(msg)
}

// lastFatal returns the last failing task: its TASK header through the end of
// its error output, trimmed to a readable length.
func lastFatal(text string) string {
	fatal := strings.LastIndex(text, "fatal: [")
	if fatal < 0 {
		return ""
	}

	start := fatal
	if task := strings.LastIndex(text[:fatal], "TASK ["); task >= 0 {
		start = task
	}

	excerpt := strings.TrimSpace(text[start:])
	if len(excerpt) > maxFatalExcerpt {
		excerpt = excerpt[:maxFatalExcerpt] + "\n..."
	}

	return excerpt
}

// tailWriter keeps the last max bytes written to it.
type tailWriter struct {
	buf []byte
	max int
}

func (t *tailWriter) Write(p []byte) (int, error) {
	t.buf = append(t.buf, p...)
	if extra := len(t.buf) - t.max; extra > 0 {
		t.buf = append(t.buf[:0], t.buf[extra:]...)
	}

	return len(p), nil
}
