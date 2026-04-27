BINARY := ./bin/agent
PKG    := ./...

.PHONY: build test test-verbose lint fmt tidy coverage coverage-report run clean

build:
	@mkdir -p bin
	go build -o $(BINARY) ./cmd/agent

test:
	go test $(PKG)

test-verbose:
	go test -v $(PKG)

lint:
	go vet $(PKG)

fmt:
	gofmt -s -w .

tidy:
	go mod tidy

coverage:
	go test -coverprofile=coverage.out $(PKG)
	go tool cover -html=coverage.out -o coverage.html

coverage-report:
	go test -cover $(PKG)

run: build
	$(BINARY)

clean:
	rm -rf bin coverage.out coverage.html
