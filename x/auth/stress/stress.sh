#!/bin/bash

# c connections
# d durations
# t threads

case "$1" in

  "sdk_login")
    wrk -c 500 -d 10s -t 10 --latency -s sdk_login.lua http://10.0.2.193:8080/
    ;;

  "v1_login")
    wrk -c 500 -d 10s -t 10 --latency -s v1_login.lua http://10.0.2.193:8080/
    ;;

  *)
    echo -n "unknown"
    ;;

esac



