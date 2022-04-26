#!/bin/sh

ACCESS_SECRET=uV3gqXaPOQrvJCQ1CgB2QML3ZhQ66cNk
RETRY_TIMEOUT=5000

go run main.go --grpc.addr=127.0.0.1:8082 --consul.addr=127.0.0.1:8500
