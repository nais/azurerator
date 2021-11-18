KUBEBUILDER_VERSION := 3.2.0
K8S_VERSION         := 1.22.1
arch                := amd64
os                  := $(shell uname -s | tr '[:upper:]' '[:lower:]')

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

kubebuilder:
	test -d /usr/local/kubebuilder || (sudo mkdir -p /usr/local/kubebuilder && sudo chown "${USER}" /usr/local/kubebuilder)
	curl -L "https://storage.googleapis.com/kubebuilder-tools/kubebuilder-tools-${K8S_VERSION}-$(os)-$(arch).tar.gz" | tar -xz -C /usr/local
	curl -L -o /usr/local/kubebuilder/bin/kubebuilder https://github.com/kubernetes-sigs/kubebuilder/releases/download/v${KUBEBUILDER_VERSION}/kubebuilder_$(os)_$(arch)
	chmod +x /usr/local/kubebuilder/bin/*

install:
	kubectl apply -f https://raw.githubusercontent.com/nais/liberator/main/config/crd/bases/nais.io_azureadapplications.yaml
	kubectl apply -f ./hack/resources/

sample:
	kubectl apply -f ./config/samples/azureadapplication.yaml
