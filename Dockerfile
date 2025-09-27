FROM golang:1.24.5-alpine AS builder

WORKDIR /workspace

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build args
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags "-X github.com/cychiang/xp-provider-gen/pkg/version.Version=${VERSION} -X github.com/cychiang/xp-provider-gen/pkg/version.GitCommit=${COMMIT} -X github.com/cychiang/xp-provider-gen/pkg/version.BuildDate=${BUILD_DATE}" \
    -trimpath \
    -o xp-provider-gen \
    ./cmd/xp-provider-gen

# Final image
FROM alpine:3.22

# Install dependencies
RUN apk --no-cache add ca-certificates git

# Copy binary
COPY --from=builder /workspace/xp-provider-gen /usr/local/bin/xp-provider-gen
RUN chmod +x /usr/local/bin/xp-provider-gen

# Set working directory
WORKDIR /workspace

# Entry point
ENTRYPOINT ["xp-provider-gen"]