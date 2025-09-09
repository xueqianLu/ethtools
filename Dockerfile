FROM golang:1.20-alpine AS build
  
# Set up dependencies
ENV PACKAGES build-base

# Install dependencies
RUN apk add --update $PACKAGES

# Add source files
WORKDIR /build

COPY ./ /build/ethtools

RUN cd /build/ethtools && go build -ldflags="-s -w" -o /tmp/ethtools .

FROM alpine

WORKDIR /app

COPY --from=build /tmp/ethtools /usr/bin/ethtools
