package retry

import (
	"context"
	"time"

	"github.com/sethvargo/go-retry"
)

type Backoff struct {
	b retry.Backoff
}

func RetryableError(err error) error {
	return retry.RetryableError(err)
}

func Fibonacci(base time.Duration) Backoff {
	if base <= 0 {
		base = 1 * time.Second
	}
	b := retry.NewFibonacci(base)

	return Backoff{
		b: b,
	}
}

func (in Backoff) WithMaxDuration(timeout time.Duration) Backoff {
	in.b = retry.WithMaxDuration(timeout, in.b)
	return in
}

func (in Backoff) Do(ctx context.Context, f retry.RetryFunc) error {
	return retry.Do(ctx, in.b, f)
}
