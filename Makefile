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
	CGO_ENABLED=0 go build -ldflags '-extldflags "-static"' test-create.go
	CGO_ENABLED=0 go build -ldflags '-extldflags "-static"' test-delete.go

plot:
	CGO_ENABLED=0 go build -ldflags '-extldflags "-static"' plot-results.go