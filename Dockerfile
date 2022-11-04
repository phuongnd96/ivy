# Build the manager binary
FROM golang:1.18 as builder

WORKDIR /go/src/github.com/phuongnd96/ivy
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY main.go main.go
COPY api/ api/
COPY controllers/ controllers/
COPY pkg/ pkg/
COPY helper/ helper/
# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o manager main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM golang:1.18-alpine
WORKDIR /
COPY --from=builder /go/src/github.com/phuongnd96/ivy/manager .
COPY config.yaml .
USER 65532:65532

ENTRYPOINT ["/manager"]
