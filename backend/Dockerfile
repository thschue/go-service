# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod ./
COPY main.go ./
COPY openapi.yaml ./
COPY static/ ./static/

RUN go mod tidy
RUN go build -o backend

# Final image (Distroless)
FROM gcr.io/distroless/static:nonroot

WORKDIR /app
COPY --from=builder /app/backend .
COPY --from=builder /app/openapi.yaml .
COPY --from=builder /app/static ./static

USER nonroot:nonroot
EXPOSE 8081

ENTRYPOINT ["/app/backend"]
