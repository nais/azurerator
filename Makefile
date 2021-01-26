# Run tests excluding integration tests
test: fmt vet
	go test ./... -coverprofile cover.out -short

# Run against the configured Kubernetes cluster in ~/.kube/config
run: fmt vet
	go run cmd/azurerator/main.go

# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	go vet ./...
