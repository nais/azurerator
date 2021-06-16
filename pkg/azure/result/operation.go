package result

type Operation int

const (
	OperationCreated Operation = iota
	OperationUpdated
	OperationNotModified
)
