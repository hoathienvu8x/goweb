.PHONY: build clean

build:
	@echo "Build goweb app..."
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build

run:
	@echo "Run webapp..."
	@go build && ./goweb start

clean:
	rm -rf goweb
