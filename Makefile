SERVICES := auth catalog booking payment notification gateway

.PHONY: help run web build test tidy fmt vet clean

help:
	@echo "make run s=auth   - run one Go service"
	@echo "make web          - run the React dev server"
	@echo "make build        - build all Go services into bin/"
	@echo "make test         - run Go tests with the race detector"
	@echo "make tidy         - go mod tidy"
	@echo "make fmt          - gofmt all Go code"
	@echo "make vet          - go vet all packages"
	@echo "make clean        - remove build output"
	@echo ""
	@echo "services: $(SERVICES)"

run:
	go run ./services/$(s)/cmd/server

web:
	cd frontend && npm run dev

build:
	@for s in $(SERVICES); do \
		echo "building $$s"; \
		go build -o bin/$$s ./services/$$s/cmd/server || exit 1; \
	done

test:
	go test -race ./...

tidy:
	go mod tidy

fmt:
	gofmt -l -w .

vet:
	go vet ./...

clean:
	rm -rf bin/
