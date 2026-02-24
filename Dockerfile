FROM golang:1.21-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=dev
RUN CGO_ENABLED=0 go build -ldflags="-s -w -X main.version=${VERSION}" -o /spectrehub ./cmd/spectrehub

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /spectrehub /usr/local/bin/spectrehub
ENTRYPOINT ["spectrehub"]
