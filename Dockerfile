# Build the manager binary
FROM golang:1.16 as builder

ENV os "linux"
ENV arch "amd64"

# download kubebuilder and extract it to tmp
RUN curl -L https://go.kubebuilder.io/dl/2.3.1/${os}/${arch} | tar -xz -C /tmp/

# move to a long-term location and put it on your path
# (you'll need to set the KUBEBUILDER_ASSETS env var if you put it somewhere else)
RUN mv /tmp/kubebuilder_2.3.1_${os}_${arch} /usr/local/kubebuilder
RUN export PATH=$PATH:/usr/local/kubebuilder/bin

COPY . /workspace
WORKDIR /workspace
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download
# Run tests
RUN make test
# Build
RUN CGO_ENABLED=0 GOOS=${os} GOARCH=${arch} GO111MODULE=on go build -a -installsuffix cgo -o azurerator cmd/azurerator/main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM alpine:3
WORKDIR /
COPY --from=builder /workspace/azurerator /azurerator
RUN apk add --no-cache ca-certificates

CMD ["/azurerator"]
