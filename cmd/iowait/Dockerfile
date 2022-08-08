FROM golang:1.18-bullseye as build

WORKDIR /go/src/app

COPY go.* ./
RUN go mod download

COPY . ./
RUN go build -v

FROM registry.access.redhat.com/ubi8/ubi-micro
COPY --from=build /go/src/app/iowait /
CMD ["/iowait"]