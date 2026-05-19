FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY . .
RUN go mod download
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o document-quiz-backend .

FROM alpine:latest

RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/document-quiz-backend .
COPY --from=builder /app/.env* ./

EXPOSE 3000

CMD ["./document-quiz-backend"]
