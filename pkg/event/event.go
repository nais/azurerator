package event

import (
	"encoding/json"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Event struct {
	ID          string      `json:"@id"`
	Name        Name        `json:"@event_name"`
	Application Application `json:"application"`
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

func (a Application) String() string {
	return fmt.Sprintf("%s:%s:%s", a.Cluster, a.Namespace, a.Name)
}

func New(ID string, eventName Name, app metav1.Object, clusterName string) Event {
	application := Application{
		Name:      app.GetName(),
		Namespace: app.GetNamespace(),
		Cluster:   clusterName,
	}

	return Event{ID: ID, Name: eventName, Application: application}
}

func (e Event) Marshal() ([]byte, error) {
	return json.Marshal(e)
}

func (e Event) IsCreated() bool {
	return e.Name == Created
}

func (e Event) String() string {
	return fmt.Sprintf("%s (%s)", e.Name, e.ID)
}
