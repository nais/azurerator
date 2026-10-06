package credentials

import (
	"context"
	"net/http"
	"testing"
	"time"

	msgraph "github.com/nais/msgraph.go/v1.0"
)

func TestRetryableResponse(t *testing.T) {
	for _, test := range []struct {
		name      string
		err       error
		retryable bool
	}{
		{name: "throttled", err: graphError(http.StatusTooManyRequests, "throttled", ""), retryable: true},
		{name: "tenant concurrency", err: graphError(http.StatusConflict, ConcurrentRequestMessage, ""), retryable: true},
		{name: "concurrency violation code", err: &msgraph.ErrorResponse{
			ErrorObject: msgraph.ErrorObject{Code: concurrencyViolationCode, Message: "reworded"},
			Response:    &http.Response{StatusCode: http.StatusConflict},
		}, retryable: true},
		{name: "other graph error", err: graphError(http.StatusInternalServerError, "server error", "")},
		{name: "non graph error", err: context.DeadlineExceeded},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, got := retryableResponse(test.err)
			if got != test.retryable {
				t.Fatalf("retryableResponse() = %t, want %t", got, test.retryable)
			}
		})
	}
}

func TestParseRetryAfter(t *testing.T) {
	for _, test := range []struct {
		value string
		want  time.Duration
	}{
		{value: "12", want: 12 * time.Second},
		{value: "invalid"},
	} {
		response := &msgraph.ErrorResponse{Response: &http.Response{Header: http.Header{"Retry-After": []string{test.value}}}}
		if got := parseRetryAfter(response); got != test.want {
			t.Errorf("parseRetryAfter(%q) = %s, want %s", test.value, got, test.want)
		}
	}
}

func graphError(status int, message, retryAfter string) error {
	return &msgraph.ErrorResponse{
		ErrorObject: msgraph.ErrorObject{Code: "test", Message: message},
		Response:    &http.Response{StatusCode: status, Status: http.StatusText(status), Header: http.Header{"Retry-After": []string{retryAfter}}},
	}
}
