FROM golang:1.17.8-alpine3.15 AS build
WORKDIR /go/src/app
COPY main.go go.mod go.sum ./
RUN go mod download
RUN CGO_ENABLED=0 go build -o /go/bin/app

FROM chromedp/headless-shell:98.0.4758.102
RUN apt update
RUN apt install tini
COPY --from=build /go/bin/app /usr/local/bin
ENTRYPOINT ["tini", "--"]
CMD ["app"]
