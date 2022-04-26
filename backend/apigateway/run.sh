#!/bin/sh

ACCESS_SECRET=uV3gqXaPOQrvJCQ1CgB2QML3ZhQ66cNk
RETRY_TIMEOUT=5000

go run main.go --http.addr=127.0.0.1:8000 --consul.addr=127.0.0.1:8500
