../bin/protoc --plugin=../bin/protoc-gen-go --go_out=plugins=grpc:./planxprotogen/ ./proto/planxproto/*.proto
mv ./planxprotogen/github.com/nghichtu91/platform/share/planx/servers/planxprotogen/*.go ./planxprotogen
rm -rf ./planxprotogen/gitlab.taiyouxi.cn