# Build the project
FROM golang:alpine as build

RUN mkdir -p /project
WORKDIR /project
COPY ./ ./
RUN go build -o aka-timestamps

# Build the runtime container
FROM python:alpine

RUN apk add ca-certificates \
  && mkdir /project

COPY --from=build /project/aka-timestamps /project/aka-timestamps
COPY ./youtube-dl /project/youtube-dl

WORKDIR /project
EXPOSE 9090

CMD /project/aka-timestamps
