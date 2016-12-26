.PHONY: clean deps build folders pluto dovecot

clean:
	go clean -i ./...

deps:
	go get -t ./...

build: folders pluto dovecot

folders:
	if [ ! -d "results" ]; then mkdir results; fi

pluto:
	CGO_ENABLED=0 go build -ldflags '-extldflags "-static"' test-pluto-append.go
	CGO_ENABLED=0 go build -ldflags '-extldflags "-static"' test-pluto-concurrent.go
	CGO_ENABLED=0 go build -ldflags '-extldflags "-static"' test-pluto-failover.go

dovecot:
	CGO_ENABLED=0 go build -ldflags '-extldflags "-static"' test-dovecot-append.go
	CGO_ENABLED=0 go build -ldflags '-extldflags "-static"' test-dovecot-concurrent.go