FROM golang:1.25.13
WORKDIR /app
ENV CGO_ENABLED=0 GO111MODULE=on GOPROXY=https://proxy.golang.org,direct
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build ./...
RUN CGO_ENABLED=0 go test -run '^$' ./...
CMD ["go", "run", "./cmd/alertd"]
