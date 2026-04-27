package synchronizer

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestEvent_Validate(t *testing.T) {
	app := &metav1.ObjectMeta{Name: "some-app", Namespace: "some-ns"}

	for _, test := range []struct {
		name        string
		event       Event
		wantMissing []string // empty means valid
	}{
		{
			name:  "fully populated",
			event: NewCreatedEvent("1", app, "some-cluster", "some-client-id"),
		},
		{
			name:        "missing ClientID",
			event:       NewCreatedEvent("1", app, "some-cluster", ""),
			wantMissing: []string{"clientID"},
		},
		{
			name:        "missing cluster",
			event:       NewCreatedEvent("1", app, "", "some-client-id"),
			wantMissing: []string{"cluster"},
		},
		{
			name:        "missing namespace",
			event:       NewCreatedEvent("1", &metav1.ObjectMeta{Name: "some-app"}, "some-cluster", "some-client-id"),
			wantMissing: []string{"namespace"},
		},
		{
			name:        "missing name",
			event:       NewCreatedEvent("1", &metav1.ObjectMeta{Namespace: "some-ns"}, "some-cluster", "some-client-id"),
			wantMissing: []string{"name"},
		},
		{
			name:        "multiple missing fields are all reported",
			event:       NewCreatedEvent("1", &metav1.ObjectMeta{}, "", ""),
			wantMissing: []string{"name", "namespace", "cluster", "clientID"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := test.event.Validate()
			if len(test.wantMissing) == 0 {
				assert.NoError(t, err)
				return
			}
			assert.Error(t, err)
			for _, field := range test.wantMissing {
				assert.True(t, strings.Contains(err.Error(), field),
					"expected error to mention %q, got: %v", field, err)
			}
		})
	}
}
