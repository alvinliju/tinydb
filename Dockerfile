FROM golang:1.19

#TEACH:this is where we play with so that we dont have to write full paths during the build process
WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY *.go ./
