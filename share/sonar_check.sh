#!/usr/bin/env bash
rm -rf report
mkdir report

# go test 
go test ./... -json>./report/share_test.out -coverprofile=./report/share_coverage.out

# go vet
go vet ./x/... 2>./report/share_vet.out
go vet ./planx/... 2>>./report/share_vet.out

# golint
golint ./x/... >./report/share_lint.out
golint ./planx/... >>./report/share_lint.out

