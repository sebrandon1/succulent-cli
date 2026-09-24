package lib

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"
)

func (c *Client) StreamLog(ctx context.Context, env string, w io.Writer) error {
	resp, err := c.getRaw(ctx, c.endpointURL(endpointLog, env))
	if err != nil {
		return fmt.Errorf("failed to fetch log for %s: %w", env, err)
	}
	defer resp.Body.Close()

	if err := copyLimited(w, resp.Body, MaxResponseSize); err != nil {
		return fmt.Errorf("error streaming log: %w", err)
	}

	return nil
}

// FollowLog polls the log endpoint and writes newly appended output until
// Ansible prints its play recap or the context is canceled.
func (c *Client) FollowLog(ctx context.Context, env string, w io.Writer, pollInterval time.Duration) error {
	if pollInterval <= 0 {
		return fmt.Errorf("log poll interval must be positive")
	}

	var offset int64
	var previousTail []byte

	for {
		resp, err := c.getRaw(ctx, c.endpointURL(endpointLog, env))
		if err != nil {
			return fmt.Errorf("failed to fetch log for %s: %w", env, err)
		}

		if offset > 0 {
			skipped, err := io.CopyN(io.Discard, resp.Body, offset)
			if err != nil {
				resp.Body.Close()
				if err == io.EOF || err == io.ErrUnexpectedEOF {
					return fmt.Errorf("log output for %s shrank while following", env)
				}
				return fmt.Errorf("failed to skip existing log output for %s: %w", env, err)
			}
			if skipped != offset {
				resp.Body.Close()
				return fmt.Errorf("log output for %s shrank while following", env)
			}
		}

		stream := &logFollowWriter{dst: w, tail: previousTail}
		written, copyErr := io.Copy(stream, resp.Body)
		closeErr := resp.Body.Close()
		if copyErr != nil {
			return fmt.Errorf("error streaming log: %w", copyErr)
		}
		if closeErr != nil {
			return fmt.Errorf("closing log response: %w", closeErr)
		}

		offset += written
		previousTail = stream.tail

		if stream.finished {
			return nil
		}

		timer := time.NewTimer(pollInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

type logFollowWriter struct {
	dst      io.Writer
	tail     []byte
	finished bool
}

func (w *logFollowWriter) Write(p []byte) (int, error) {
	combined := make([]byte, 0, len(w.tail)+len(p))
	combined = append(combined, w.tail...)
	combined = append(combined, p...)
	if bytes.HasPrefix(combined, []byte("PLAY RECAP")) || bytes.Contains(combined, []byte("\nPLAY RECAP")) {
		w.finished = true
	}

	written, err := w.dst.Write(p)
	if err != nil {
		return written, err
	}
	if written != len(p) {
		return written, io.ErrShortWrite
	}

	const tailSize = len("\nPLAY RECAP")
	if len(combined) > tailSize {
		combined = combined[len(combined)-tailSize:]
	}
	w.tail = append(w.tail[:0], combined...)

	return written, nil
}
