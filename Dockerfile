FROM golang:1.27.1-alpine AS builder

# The official golang images set GOTOOLCHAIN=local, so the base image tag has to
# be at least as new as go.mod's go directive or `go mod download` refuses to
# run. Renovate bumps the tag and the directive from different datasources in
# different PRs, so they drift, and the drift breaks the image build rather than
# conflicting honestly. Letting Go fetch the toolchain go.mod asks for makes the
# build follow the module instead of racing it; the pinned tag still fixes the
# base OS and the usual case downloads nothing.
ENV GOTOOLCHAIN=auto

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
FROM alpine:3.24

# Install dependencies
RUN apk --no-cache add ca-certificates git

# Copy binary
COPY --from=builder /workspace/xp-provider-gen /usr/local/bin/xp-provider-gen
RUN chmod +x /usr/local/bin/xp-provider-gen

# Set working directory
WORKDIR /workspace

# Entry point
ENTRYPOINT ["xp-provider-gen"]