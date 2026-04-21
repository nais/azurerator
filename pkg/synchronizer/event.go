package synchronizer

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
	Updated Name = "Updated"
)

type Application struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Cluster   string `json:"cluster"`
	ClientID  string `json:"clientId"`
}

func (a Application) String() string {
	return fmt.Sprintf("%s:%s:%s", a.Cluster, a.Namespace, a.Name)
}

func NewEvent(ID string, eventName Name, app metav1.Object, clusterName, clientID string) Event {
	application := Application{
		Name:      app.GetName(),
		Namespace: app.GetNamespace(),
		Cluster:   clusterName,
		ClientID:  clientID,
	}

	return Event{ID: ID, Name: eventName, Application: application}
}

func NewCreatedEvent(ID string, app metav1.Object, clusterName, clientID string) Event {
	return NewEvent(ID, Created, app, clusterName, clientID)
}

func NewUpdatedEvent(ID string, app metav1.Object, clusterName, clientID string) Event {
	return NewEvent(ID, Updated, app, clusterName, clientID)
}

func (e Event) Marshal() ([]byte, error) {
	return json.Marshal(e)
}

func (e Event) IsCreated() bool {
	return e.Name == Created
}

func (e Event) IsUpdated() bool {
	return e.Name == Updated
}

func (e Event) String() string {
	return fmt.Sprintf("%s (%s)", e.Name, e.ID)
}
