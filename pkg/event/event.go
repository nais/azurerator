package event

import (
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Event struct {
	ID          string      `json:"@id"`
	EventName   Name        `json:"@event_name"`
	Application Application `json:"application"`
}

func NewEvent(ID string, eventName Name, app metav1.Object) Event {
	application := Application{
		Name:      app.GetName(),
		Namespace: app.GetNamespace(),
		Cluster:   app.GetClusterName(),
	}

	return Event{ID: ID, EventName: eventName, Application: application}
}

func (e Event) Marshal() ([]byte, error) {
	return json.Marshal(e)
}

type Name string

const (
	Created Name = "Created"
)

type Application struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Cluster   string `json:"cluster"`
}
