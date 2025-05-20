# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod ./
COPY main.go ./
COPY templates/ ./templates/

RUN go mod tidy
RUN go build -o frontend

# Final image (Distroless)
FROM gcr.io/distroless/static:nonroot

WORKDIR /app
COPY --from=builder /app/frontend .
COPY --from=builder /app/templates ./templates

USER nonroot:nonroot
EXPOSE 8080

ENTRYPOINT ["/app/frontend"]
