.PHONY: clean deps build folders tests plot

clean:
	go clean -i ./...

deps:
	go get -t ./...

build: folders tests plot

folders:
	if [ ! -d "results" ]; then mkdir results; fi
	if [ ! -d "private" ]; then mkdir private; fi
	chmod 0700 private

tests:
	CGO_ENABLED=0 go build -ldflags '-extldflags "-static"' test-append.go
	CGO_ENABLED=0 go build -ldflags '-extldflags "-static"' test-create.go
	CGO_ENABLED=0 go build -ldflags '-extldflags "-static"' test-delete.go
	CGO_ENABLED=0 go build -ldflags '-extldflags "-static"' test-store.go
	CGO_ENABLED=0 go build -ldflags '-extldflags "-static"' test-append-concurrent.go

plot:
	CGO_ENABLED=0 go build -ldflags '-extldflags "-static"' plot-results.go

f:
	CGO_ENABLED=0 go build -ldflags '-extldflags "-static"' test-append-concurrent.go