# syntax=docker/dockerfile:1

# Compile on the CI runner's native arch and cross-compile per TARGETARCH (CGO_ENABLED=0).
# Avoids QEMU emulation when publishing linux/amd64 + linux/arm64 images.
FROM --platform=$BUILDPLATFORM golang:1.25-bookworm AS build
ARG TARGETARCH
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
ENV CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH}
RUN go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates curl \
	&& rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=build /out/server /app/server
USER 65532:65532
EXPOSE 8080
CMD ["/app/server"]