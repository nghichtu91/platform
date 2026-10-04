#!/bin/sh

protoc_gen_go_dir=../../bin/protoc-gen-go

../../bin/protoc --plugin=${protoc_gen_go_dir} -I .  --go_out=../planxprotogen --go_opt=paths=source_relative *.proto