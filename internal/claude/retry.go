package claude

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

// RetryExecutor wraps an [Executor] to automatically retry on rate limit errors.
//
// When Claude exits due to a usage quota limit, RetryExecutor detects the rate
// limit from the streaming events, waits until the quota resets, and retries
// the same prompt. It passes through all events to the original handler so the
// caller still sees the full output.
//
// Rate limits are detected via two mechanisms:
//  1. Structured rate_limit_event in the JSON stream (primary)
//  2. Text pattern matching for "You've hit your limit" messages (fallback)
//
// Non-rate-limit failures are passed through immediately without retry.
type RetryExecutor struct {
	inner      Executor
	maxRetries int
	sleepFunc  func(context.Context, time.Duration) error
}

// NewRetryExecutor creates a [RetryExecutor] wrapping the given executor.
//
// maxRetries controls how many times to retry on rate limit (default 10 if <= 0).
// Each retry waits until the quota reset time plus a 60-second buffer.
func NewRetryExecutor(inner Executor, maxRetries int) *RetryExecutor {
	if maxRetries <= 0 {
		maxRetries = 10
	}
	return &RetryExecutor{
		inner:      inner,
		maxRetries: maxRetries,
		sleepFunc:  defaultSleep,
	}
}

// Execute delegates to the inner executor without retry.
//
// The channel-based Execute mode does not support retry because the caller
// consumes events as they stream. Use [ExecuteWithResult] for retry support.
func (r *RetryExecutor) Execute(ctx context.Context, prompt string) (<-chan Event, error) {
	return r.inner.Execute(ctx, prompt)
}

// ExecuteWithResult runs the inner executor and retries on rate limit errors.
//
// On each attempt, events are intercepted to detect rate limits but still
// passed through to the original handler. If the execution fails and a rate
// limit is detected, the executor waits until the reset time and retries.
//
// Non-rate-limit failures return immediately without retry.
func (r *RetryExecutor) ExecuteWithResult(ctx context.Context, prompt string, handler EventHandler) (int, error) {
	var exitCode int
	var execErr error

	for attempt := 0; attempt <= r.maxRetries; attempt++ {
		var rateLimitResetAt time.Time
		var rateLimited bool
		var capturedTexts []string

		wrappedHandler := func(event Event) {
			// Capture rate limit info from structured events
			if event.IsRateLimited() && !event.RateLimitResetAt.IsZero() {
				rateLimited = true
				rateLimitResetAt = event.RateLimitResetAt
			}

			// Capture text for fallback detection
			if event.IsText() {
				capturedTexts = append(capturedTexts, event.Text)
			}

			// Always pass through to original handler
			if handler != nil {
				handler(event)
			}
		}

		exitCode, execErr = r.inner.ExecuteWithResult(ctx, prompt, wrappedHandler)

		// Success — return immediately
		if exitCode == 0 && execErr == nil {
			return 0, nil
		}

		// Last attempt — don't try to detect rate limit, just return
		if attempt == r.maxRetries {
			break
		}

		// Check for rate limit via structured event
		if !rateLimited {
			// Fallback: check text for rate limit pattern
			if resetTime, found := parseRateLimitFromText(capturedTexts); found {
				rateLimited = true
				rateLimitResetAt = resetTime
			}
		}

		// Not a rate limit — fail immediately
		if !rateLimited {
			return exitCode, execErr
		}

		// Calculate wait duration
		waitDuration := time.Until(rateLimitResetAt) + 60*time.Second
		if waitDuration <= 0 {
			waitDuration = 60 * time.Second
		}

		fmt.Fprintf(os.Stderr, "\n⏳ Rate limit hit. Waiting until %s (retry %d/%d)...\n",
			rateLimitResetAt.Local().Format("15:04 MST"), attempt+1, r.maxRetries)

		if err := r.sleepFunc(ctx, waitDuration); err != nil {
			return exitCode, err
		}

		fmt.Fprintf(os.Stderr, "🔄 Resuming workflow...\n\n")
	}

	return exitCode, execErr
}

func defaultSleep(ctx context.Context, d time.Duration) error {
	select {
	case <-time.After(d):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// rateLimitTextPattern matches the text message Claude displays when rate limited.
// Handles both ASCII apostrophe and Unicode right single quotation mark.
// Examples:
//
//	"You've hit your limit · resets 4pm (Europe/Berlin)"
//	"You've hit your limit · resets 7:20pm (Europe/Berlin)"
var rateLimitTextPattern = regexp.MustCompile(
	`You[''\x{2019}]ve hit your limit.*resets\s+(\d{1,2}(?::\d{2})?\s*[aApP][mM])\s*\(([^)]+)\)`,
)

// parseRateLimitFromText checks captured text events for rate limit messages
// and parses the reset time. This is a fallback for when the structured
// rate_limit_event is not available or doesn't indicate the limit.
func parseRateLimitFromText(texts []string) (time.Time, bool) {
	for _, text := range texts {
		matches := rateLimitTextPattern.FindStringSubmatch(text)
		if matches == nil {
			continue
		}

		timeStr := strings.TrimSpace(strings.ToLower(matches[1]))
		tzName := strings.TrimSpace(matches[2])

		loc, err := time.LoadLocation(tzName)
		if err != nil {
			continue
		}

		now := time.Now().In(loc)

		// Parse time — try "3:04pm" and "3pm" formats
		var parsed time.Time
		for _, layout := range []string{"3:04pm", "3pm"} {
			parsed, err = time.Parse(layout, timeStr)
			if err == nil {
				break
			}
		}
		if err != nil {
			continue
		}

		// Build full timestamp: today in that timezone
		resetTime := time.Date(
			now.Year(), now.Month(), now.Day(),
			parsed.Hour(), parsed.Minute(), 0, 0, loc,
		)

		// If reset time is in the past by more than 5 minutes, assume tomorrow
		if resetTime.Before(now.Add(-5 * time.Minute)) {
			resetTime = resetTime.Add(24 * time.Hour)
		}

		return resetTime, true
	}

	return time.Time{}, false
}
