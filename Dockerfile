FROM golang as builder
ENV GOOS=linux
ENV GOARCH=amd64
ENV CGO_ENABLED=0

WORKDIR /build

COPY go.mod ./
COPY go.sum ./
RUN go mod download

ADD . src

WORKDIR /build/src

RUN go build -a -tags netgo -ldflags '-w' -o encarno cmd/encarno/main.go

FROM python

RUN pip install bzt # to cache the step

COPY --from=builder /build/src/encarno /usr/bin/encarno

ADD taurus /tmp/taurus

RUN pip install /tmp/taurus

RUN bzt /tmp/taurus/dummy.yml # sanity test

ENTRYPOINT TODO