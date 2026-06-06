# Stage 1: Build execution setup
FROM golang:1.22-alpine AS builder
WORKDIR /app

RUN go mod init quikdb-frame-app && go mod tidy
RUN go get github.com/gofiber/fiber/v2 github.com/gofiber/contrib/websocket

COPY main.go .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o app main.go

# Stage 2: Empty scratch deployment to keep size under 15MB
FROM scratch
COPY --from=builder /app/app /app
EXPOSE 8080
ENTRYPOINT ["/app"]