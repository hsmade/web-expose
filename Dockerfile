FROM golang:1.15.3
WORKDIR /app
ENV PATH=$PATH:/app
COPY go.mod go.sum /app/
RUN go mod download
COPY . /app/
RUN go build cmd/client/client.go
RUN go build cmd/server/server.go
