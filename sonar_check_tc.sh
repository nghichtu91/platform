#!/usr/bin/env bash
rm -rf report
mkdir report

export GO111MODULE=on
export GOPROXY=https://goproxy.cn/

# go test 
/Users/bm/bm_data_nfs/jws2/go/bin/go test ./... -json>./report/share_test.out -coverprofile=./report/share_coverage.out

# go vet
/Users/bm/bm_data_nfs/jws2/go/bin/go vet ./x/... 2>./report/share_vet.out
/Users/bm/bm_data_nfs/jws2/go/bin/go vet ./planx/... 2>>./report/share_vet.out

# golint
/Users/bm/go/bin/golint ./x/... >./report/share_lint.out
/Users/bm/go/bin/golint ./planx/... >>./report/share_lint.out

