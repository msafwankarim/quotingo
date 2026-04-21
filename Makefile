BINARY=quotingo
IMAGE=msafwankarim/quotingo:latest
CONTAINER=quotingo
PORT=8080

.PHONY: build test run docker-build docker-run docker-start docker-stop clean

build:
	mkdir -p bin
	go build -o bin/$(BINARY) cmd/main.go

test:
	go test ./...

run: build
	./bin/$(BINARY)

docker-build:
	docker build -t $(IMAGE) .

docker-run: docker-build
	docker run --rm -p $(PORT):$(PORT) $(IMAGE)

docker-start: docker-build
	docker run -d --name $(CONTAINER) -p $(PORT):$(PORT) $(IMAGE)

docker-stop:
	docker stop $(CONTAINER) || true
	docker rm $(CONTAINER) || true

clean:
	rm -rf bin
