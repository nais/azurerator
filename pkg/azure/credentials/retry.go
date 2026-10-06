package credentials

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	msgraph "github.com/nais/msgraph.go/v1.0"
	retry "github.com/sethvargo/go-retry"
	log "github.com/sirupsen/logrus"
)

const (
	ConcurrentRequestMessage = "Error due to concurrent requests being made to the tenant. Please wait briefly and retry."
	concurrencyViolationCode = "Directory_ConcurrencyViolation"
)

// Wait pauses for duration or returns when ctx is canceled.
func Wait(ctx context.Context, duration time.Duration) error {
	if duration <= 0 {
		return nil
	}

	timer := time.NewTimer(duration)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// RetryGraph retries credential mutations for throttling and the known tenant concurrency conflict.
func RetryGraph(ctx context.Context, logger log.FieldLogger, operation string, fn func(context.Context) error) error {
	ctx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()

	var retryAfter time.Duration
	attempt := 0
	backoff := retry.WithMaxRetries(4, retry.WithCappedDuration(20*time.Second,
		retry.WithJitterPercent(20, retry.NewExponential(time.Second)),
	))
	backoffWithRetryAfter := retry.BackoffFunc(func() (time.Duration, bool) {
		next, stop := backoff.Next()
		if retryAfter > next {
			next = retryAfter
		}
		retryAfter = 0
		return next, stop
	})
	return retry.Do(ctx, backoffWithRetryAfter, func(ctx context.Context) error {
		attempt++
		err := fn(ctx)
		if err == nil {
			return nil
		}

		response, ok := retryableResponse(err)
		if !ok {
			return err
		}
		logger.Warnf("credential %s attempt %d failed (status %d, code %s)", operation, attempt, response.StatusCode(), response.ErrorObject.Code)
		retryAfter = parseRetryAfter(response)
		return retry.RetryableError(err)
	})
}

func retryableResponse(err error) (*msgraph.ErrorResponse, bool) {
	var response *msgraph.ErrorResponse
	if !errors.As(err, &response) {
		return nil, false
	}
	message := strings.TrimSpace(response.ErrorObject.Message)
	retryable := response.StatusCode() == http.StatusTooManyRequests ||
		response.ErrorObject.Code == concurrencyViolationCode ||
		message == ConcurrentRequestMessage
	return response, retryable
}

func parseRetryAfter(response *msgraph.ErrorResponse) time.Duration {
	if response.Response == nil {
		return 0
	}
	value := strings.TrimSpace(response.Response.Header.Get("Retry-After"))
	if seconds, err := strconv.ParseInt(value, 10, 64); err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second
	}
	return 0
}
