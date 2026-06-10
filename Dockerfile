FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o proxy-server .

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /app/proxy-server .
COPY static/ ./static/
EXPOSE 8080
ENV PORT=8080
ENV PHISH_DOMAIN=your-domain.onrender.com
CMD ["./proxy-server"]
