.PHONY: clean deps build folders tests plot

clean:
	go clean -i ./...

deps:
	go get -t ./...

build: folders tests plot

folders:
	if [ ! -d "results" ]; then mkdir results; fi

tests:
	CGO_ENABLED=0 go build -ldflags '-extldflags "-static"' test-append.go

plot:
	CGO_ENABLED=0 go build -ldflags '-extldflags "-static"' plot-results.go