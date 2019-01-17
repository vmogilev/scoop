FROM golang:1.11 as builder
ARG SEMVER=undefined
ARG RELEASE_VER=undefined
WORKDIR /go/src/github.com/vmogilev/scoop
COPY . ./
RUN GOOS=linux CGO_ENABLED=0 go build -a -installsuffix cgo -ldflags "-X main.relver=${RELEASE_VER} -X main.semver=${SEMVER}" -o scoop .

FROM alpine:latest
WORKDIR /
COPY --from=builder /go/src/github.com/vmogilev/scoop/scoop .
EXPOSE 8080
ENTRYPOINT ["/scoop"]
