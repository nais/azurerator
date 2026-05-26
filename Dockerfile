FROM --platform=$BUILDPLATFORM golang:1.26.3 AS builder
ENV CGO_ENABLED=0
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
ARG TARGETOS
ARG TARGETARCH
RUN GOOS=$TARGETOS GOARCH=$TARGETARCH go build -a -o bin/azurerator ./cmd/azurerator

FROM cgr.dev/chainguard/static:latest
WORKDIR /app
COPY --from=builder /src/bin/azurerator /app/azurerator
ENTRYPOINT ["/app/azurerator"]
