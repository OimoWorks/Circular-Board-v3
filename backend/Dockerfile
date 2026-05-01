FROM golang:1.24-alpine

# air (ホットリロード)
RUN go install github.com/air-verse/air@v1.52.3

# 依存パッケージ
RUN apk --no-cache add git ca-certificates

WORKDIR /app

COPY go.mod ./
# go.sumがない場合も動作するようにする
RUN GOFLAGS="-mod=mod" go mod download

COPY . .

EXPOSE 8080

CMD ["air", "-c", ".air.toml"]
