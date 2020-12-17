EXECUTABLE:=load-balancer
BUILD_DIR:=./bin

.PHONY: build test clean
build: test
	mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build -o $(BUILD_DIR)/$(EXECUTABLE) cmd/balancer/main.go

test:
	go test ./...

clean:
	rm -rf $(BUILD_DIR)/
