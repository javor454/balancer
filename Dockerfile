FROM golang:1.23.6-alpine as development

ARG PROJECT_ROOT
WORKDIR ${PROJECT_ROOT}

COPY go.mod ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -buildvcs=false -o target/balancer .

FROM alpine:latest as production

WORKDIR $PROJECT_ROOT
COPY --from=development /target/balancer .

CMD ["./balancer"]
