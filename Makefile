.PHONY: clean deps build folders tests append create delete store concur-append concur-create concur-delete concur-store gmail plot

clean:
	go clean -i ./...

deps:
	go get -t ./...

build: folders tests plot

folders:
	if [ ! -d "results" ]; then mkdir results; fi
	if [ ! -d "private" ]; then mkdir private; fi
	chmod 0700 private

tests: append create delete store concur-append concur-create concur-delete concur-store gmail

append:
	CGO_ENABLED=0 go build -ldflags '-extldflags "-static"' test-append.go

create:
	CGO_ENABLED=0 go build -ldflags '-extldflags "-static"' test-create.go

delete:
	CGO_ENABLED=0 go build -ldflags '-extldflags "-static"' test-delete.go

store:
	CGO_ENABLED=0 go build -ldflags '-extldflags "-static"' test-store.go

concur-append:
	CGO_ENABLED=0 go build -ldflags '-extldflags "-static"' test-append-concurrent.go

concur-create:
	CGO_ENABLED=0 go build -ldflags '-extldflags "-static"' test-create-concurrent.go

concur-delete:
	CGO_ENABLED=0 go build -ldflags '-extldflags "-static"' test-delete-concurrent.go

concur-store:
	CGO_ENABLED=0 go build -ldflags '-extldflags "-static"' test-store-concurrent.go

gmail:
	CGO_ENABLED=0 go build -ldflags '-extldflags "-static"' test-append-gmail.go
	CGO_ENABLED=0 go build -ldflags '-extldflags "-static"' test-create-gmail.go
	CGO_ENABLED=0 go build -ldflags '-extldflags "-static"' test-delete-gmail.go
	CGO_ENABLED=0 go build -ldflags '-extldflags "-static"' test-store-gmail.go

plot:
	CGO_ENABLED=0 go build -ldflags '-extldflags "-static"' plot-results.go