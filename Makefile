.PHONY: build clean stop start restart test

build:
	go build -o redirector ./cmd/srv

clean:
	rm -f srv

test:
	go test ./...
