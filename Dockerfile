# Build the project
FROM golang:alpine as build

RUN mkdir -p /project
WORKDIR /project
COPY ./ ./
RUN go build -o aka-timestamps

# Build the runtime container
FROM alpine

RUN apk add ca-certificates youtube-dl \
  && mkdir /project

COPY --from=build /project/aka-timestamps /project/aka-timestamps

WORKDIR /project
EXPOSE 9090

CMD /project/aka-timestamps
