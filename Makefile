.PHONY: example

test:
	go test -v *.go

example:
	go run example.go -- in.log
