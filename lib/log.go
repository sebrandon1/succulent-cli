package lib

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"time"
)

// ErrLogStillStreaming means the log endpoint was still sending output when
// the request timed out. The service keeps the response open while the
// playbook runs, so the output written so far is everything logged up to now.
var ErrLogStillStreaming = errors.New("log is still streaming; the playbook may still be running")

// maxFollowLogFailures is how many consecutive reconnects without any new
// output FollowLog tolerates before giving up.
const maxFollowLogFailures = 5

func (c *Client) StreamLog(ctx context.Context, env string, w io.Writer) error {
	resp, err := c.getRaw(ctx, c.endpointURL(endpointLog, env))
	if err != nil {
		return fmt.Errorf("failed to fetch log for %s: %w", env, err)
	}
	defer resp.Body.Close()

	counted := &countingWriter{w: w}
	if err := copyLimited(counted, resp.Body, MaxResponseSize); err != nil {
		if counted.n > 0 && isTimeout(err) {
			return ErrLogStillStreaming
		}

		return fmt.Errorf("error streaming log: %w", err)
	}

	return nil
}

// FollowLog writes the log as the service streams it and returns once Ansible
// prints its play recap or the context is canceled.
//
// While a playbook runs, the log endpoint holds the response open and sends
// output as it is written, so the stream uses no overall request timeout. If
// the connection ends early, FollowLog reconnects after pollInterval, skips
// the output it already wrote, and continues.
func (c *Client) FollowLog(ctx context.Context, env string, w io.Writer, pollInterval time.Duration) error {
	if pollInterval <= 0 {
		return fmt.Errorf("log poll interval must be positive")
	}

	stream := c.WithTimeout(0)

	var offset int64
	var previousTail []byte
	failures := 0

	for {
		nextOffset, nextTail, finished, err := stream.streamLogSnapshot(ctx, env, offset, previousTail, w)
		progressed := nextOffset > offset
		offset = nextOffset
		previousTail = nextTail

		if finished {
			return nil
		}

		if err != nil {
			var transient *transientLogError
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if !errors.As(err, &transient) {
				return err
			}

			if progressed {
				failures = 0
			}
			failures++
			if failures >= maxFollowLogFailures {
				return fmt.Errorf("giving up after %d consecutive log stream failures: %w", failures, transient.err)
			}
			c.logger().Warn("log stream interrupted; reconnecting", "env", env, "err", transient.err)
		} else {
			failures = 0
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

// transientLogError marks a fetch or read failure that FollowLog retries,
// as opposed to writer errors and a shrinking log, which end the follow.
type transientLogError struct {
	err error
}

func (e *transientLogError) Error() string { return e.err.Error() }

func (c *Client) streamLogSnapshot(ctx context.Context, env string, offset int64, previousTail []byte, w io.Writer) (nextOffset int64, nextTail []byte, finished bool, retErr error) {
	resp, err := c.getRaw(ctx, c.endpointURL(endpointLog, env))
	if err != nil {
		err = fmt.Errorf("failed to fetch log for %s: %w", env, err)
		// Before any output, a failed request (bad --env, server down) is not
		// worth retrying; once the follow is under way, reconnect.
		if offset == 0 {
			return offset, previousTail, false, err
		}

		return offset, previousTail, false, &transientLogError{err: err}
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			retErr = errors.Join(retErr, fmt.Errorf("closing log response: %w", err))
		}
	}()

	if offset > 0 {
		skipped, err := io.CopyN(io.Discard, resp.Body, offset)
		if err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				return offset, previousTail, false, fmt.Errorf("log output for %s shrank while following", env)
			}
			return offset, previousTail, false, &transientLogError{err: fmt.Errorf("failed to skip existing log output for %s: %w", env, err)}
		}
		if skipped != offset {
			return offset, previousTail, false, fmt.Errorf("log output for %s shrank while following", env)
		}
	}

	stream := &logFollowWriter{dst: w, tail: previousTail}
	written, err := io.Copy(stream, resp.Body)
	if err != nil {
		// Output already written stays written: advance the offset so a
		// reconnect does not print it again.
		if stream.writeErr != nil {
			return offset + written, stream.tail, stream.finished, stream.writeErr
		}

		return offset + written, stream.tail, stream.finished, &transientLogError{err: fmt.Errorf("error streaming log: %w", err)}
	}

	return offset + written, stream.tail, stream.finished, nil
}

type logFollowWriter struct {
	dst      io.Writer
	tail     []byte
	finished bool
	writeErr error
}

func (w *logFollowWriter) Write(p []byte) (int, error) {
	combined := make([]byte, 0, len(w.tail)+len(p))
	combined = append(combined, w.tail...)
	combined = append(combined, p...)
	if bytes.HasPrefix(combined, []byte("PLAY RECAP")) || bytes.Contains(combined, []byte("\nPLAY RECAP")) {
		w.finished = true
	}

	written, err := w.dst.Write(p)
	if err == nil && written != len(p) {
		err = io.ErrShortWrite
	}
	if err != nil {
		w.writeErr = err
		return written, err
	}

	const tailSize = len("\nPLAY RECAP")
	if len(combined) > tailSize {
		combined = combined[len(combined)-tailSize:]
	}
	w.tail = append(w.tail[:0], combined...)

	return written, nil
}

type countingWriter struct {
	w io.Writer
	n int64
}

func (c *countingWriter) Write(p []byte) (int, error) {
	n, err := c.w.Write(p)
	c.n += int64(n)

	return n, err
}

func isTimeout(err error) bool {
	var netErr net.Error

	return errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &netErr) && netErr.Timeout())
}
