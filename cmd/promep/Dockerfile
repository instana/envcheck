FROM golang:latest as build

WORKDIR /go/src/app

COPY go.* ./
RUN go mod download

COPY . ./
RUN go build -o promep -v ./cmd/promep 

FROM registry.access.redhat.com/ubi8/ubi-micro
COPY --from=build /go/src/app/promep /
CMD ["/promep"]
