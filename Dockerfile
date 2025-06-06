FROM golang:1.24-alpine

WORKDIR /app

# 安裝依賴
COPY go.mod go.sum ./
RUN go mod download

# 複製源碼
COPY . .

# 編譯應用
RUN go build -o app .

# 設置環境變數
ENV CGO_ENABLED=0
ENV PORT=8080

# 暴露埠口
EXPOSE 8080

# 運行應用
CMD ["./app"]