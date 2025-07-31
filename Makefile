#ENVTEST_VERSION is the version of controller-runtime release branch to fetch the envtest setup script (i.e. release-0.20)
ENVTEST_VERSION ?= release-0.19
#ENVTEST_K8S_VERSION is the version of Kubernetes to use for setting up ENVTEST binaries (i.e. 1.31)
ENVTEST_K8S_VERSION ?= 1.31.0
## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

ENVTEST ?= $(LOCALBIN)/setup-envtest

# Run tests excluding integration tests
test: fmt vet
	go test ./... -coverprofile cover.out -short

# Run against the configured Kubernetes cluster in ~/.kube/config
run: fmt vet
	go run cmd/azurerator/main.go

fmt:
	go tool gofumpt -w ./

vet:
	go vet ./...

vuln:
	go tool govulncheck ./...

static:
	go tool staticcheck ./...

deadcode:
	go tool deadcode -test ./...

helm-lint:
	helm lint --strict ./charts

check: static deadcode vuln

install:
	kubectl apply -f https://raw.githubusercontent.com/nais/liberator/main/config/crd/bases/nais.io_azureadapplications.yaml
	kubectl apply -f ./hack/resources/

sample:
	kubectl apply -f ./config/samples/azureadapplication.yaml

##@ Dependencies
setup-envtest: envtest ## Download the binaries required for ENVTEST in the local bin directory.
	@echo "Setting up envtest binaries for Kubernetes version $(ENVTEST_K8S_VERSION)..."
	@$(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path || { \
		echo "Error: Failed to set up envtest binaries for version $(ENVTEST_K8S_VERSION)."; \
		exit 1; \
	}

envtest: $(ENVTEST) ## Download setup-envtest locally if necessary.
$(ENVTEST): $(LOCALBIN)
	$(call go-install-tool,$(ENVTEST),sigs.k8s.io/controller-runtime/tools/setup-envtest,$(ENVTEST_VERSION))

# go-install-tool will 'go install' any package with custom target and name of binary, if it doesn't exist
# $1 - target path with name of binary
# $2 - package url which can be installed
# $3 - specific version of package
define go-install-tool
@[ -f "$(1)-$(3)" ] || { \
set -e; \
package=$(2)@$(3) ;\
echo "Downloading $${package}" ;\
rm -f $(1) || true ;\
GOBIN=$(LOCALBIN) go install $${package} ;\
mv $(1) $(1)-$(3) ;\
} ;\
ln -sf $(1)-$(3) $(1)
endef
