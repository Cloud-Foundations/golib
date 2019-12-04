all: test

test:
	@find * -name '*_test.go' |\
	sed -e 's@^@github.com/Cloud-Foundations/golib/@' -e 's@/[^/]*$$@@' |\
	sort -u | xargs go test
