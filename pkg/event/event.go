package event

import (
	"encoding/json"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Event struct {
	ID          string      `json:"@id"`
	EventName   Name        `json:"@event_name"`
	Application Application `json:"application"`
}

func NewEvent(ID string, eventName Name, app metav1.Object, clusterName string) Event {
	application := Application{
		Name:      app.GetName(),
		Namespace: app.GetNamespace(),
		Cluster:   clusterName,
	}

	return Event{ID: ID, EventName: eventName, Application: application}
}

func (e Event) Marshal() ([]byte, error) {
	return json.Marshal(e)
}

func (e Event) IsCreated() bool {
	return e.EventName == Created
}

func (e Event) String() string {
	return fmt.Sprintf("%s (%s)", e.EventName, e.ID)
}
