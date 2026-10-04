#!/bin/sh

../../../planx/bin/protoc --plugin=../../../planx/bin/protoc-gen-go --proto_path=. --go_out=plugins=grpc,paths=source_relative:. proto/chat/*.proto
mv -f proto/chat/*.pb.go protogen

