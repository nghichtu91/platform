#!/bin/sh
protoc_gen_go_dir=../../bin/protoc-gen-go
../../bin/protoc --plugin=${protoc_gen_go_dir} -I .  --go_out=plugins=grpc,paths=source_relative:. *.proto
