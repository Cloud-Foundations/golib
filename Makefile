all:
	@cd $(GOPATH)/src; go install github.com/Cloud-Foundations/golib/cmd/*

format:
	gofmt -s -w .

format-imports:
	goimports -w .

get-deps:
	go get -t ./...

update-deps:
	go get -u ./...
	go mod tidy

test:
	@find * -name '*_test.go' |\
	sed -e 's@^@github.com/Cloud-Foundations/golib/@' -e 's@/[^/]*$$@@' |\
	sort -u | xargs go test

test-uncached:
	@find * -name '*_test.go' |\
	sed -e 's@^@github.com/Cloud-Foundations/golib/@' -e 's@/[^/]*$$@@' |\
	sort -u | xargs go test -count=1
