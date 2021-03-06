package options

import (
	v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/liberator/pkg/finalizer"

	"github.com/nais/azureator/pkg/annotations"
)

const (
	FinalizerName string = "finalizer.azurerator.nais.io"
)

type FinalizerOptions struct {
	Finalize        bool
	Register        bool
	DeleteFromAzure bool
}

func (b optionsBuilder) Finalizer() FinalizerOptions {
	hasFinalizer := finalizer.HasFinalizer(&b.instance, FinalizerName)
	finalize := hasFinalizer && finalizer.IsBeingDeleted(&b.instance)
	shouldPreserve := ShouldPreserve(&b.instance)

	return FinalizerOptions{
		Finalize:        finalize,
		Register:        !hasFinalizer,
		DeleteFromAzure: !shouldPreserve,
	}
}

func ShouldPreserve(instance *v1.AzureAdApplication) bool {
	_, found := annotations.HasAnnotation(instance, annotations.PreserveKey)
	return found
}
