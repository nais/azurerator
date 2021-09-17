package event

import (
	"fmt"
)

type Application struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Cluster   string `json:"cluster"`
}

func (a Application) String() string {
	return fmt.Sprintf("%s:%s:%s", a.Cluster, a.Namespace, a.Name)
}
