all:
	@cd $(GOPATH)/src; go install github.com/Cloud-Foundations/golib/cmd/*

format:
	gofmt -s -w .

format-imports:
	goimports -w .

get-deps:
	go get -t ./...

test:
	@find * -name '*_test.go' |\
	sed -e 's@^@github.com/Cloud-Foundations/golib/@' -e 's@/[^/]*$$@@' |\
	sort -u | xargs go test
