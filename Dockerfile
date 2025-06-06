FROM golang:1.24-alpine

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o app .

ENV CGO_ENABLED=0
ENV PORT=8080

EXPOSE 8080

CMD ["./app"]