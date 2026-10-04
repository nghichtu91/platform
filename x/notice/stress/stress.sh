#!/bin/bash

# c connections
# d durations
# t threads

case "$1" in

  "inner")
    wrk -c 500 -d 10s -t 10 --latency -s inner.lua http://localhost:8084/
    ;;

  "outer")
    wrk -c 500 -d 10s -t 10 --latency -s outer.lua http://121.196.106.98:8801/
    ;;

  *)
    echo -n "unknown"
    ;;

esac



