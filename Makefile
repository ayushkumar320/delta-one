SERVICES := auth catalog booking payment notification gateway

.PHONY: help run run-all web databases build test tidy fmt vet clean

help:
	@echo "make databases    - create the per-service Postgres databases"
	@echo "make run s=auth   - run one Go service"
	@echo "make run-all      - run every Go service in one terminal"
	@echo "make web          - run the React dev server"
	@echo "make build        - build all Go services into bin/"
	@echo "make test         - run Go tests with the race detector"
	@echo "make tidy         - go mod tidy"
	@echo "make fmt          - gofmt all Go code"
	@echo "make vet          - go vet all packages"
	@echo "make clean        - remove build output"
	@echo ""
	@echo "services: $(SERVICES)"

DATABASES := delta_auth delta_catalog delta_booking delta_payment

# Each service owns a database. Creating them is the only setup step Postgres
# needs; the services apply their own migrations when they start.
databases:
	@for db in $(DATABASES); do \
		echo "creating $$db"; \
		docker exec -i delta-postgres createdb -U postgres $$db 2>/dev/null \
			|| echo "  $$db already exists"; \
	done

run:
	go run ./services/$(s)/cmd/server

# Runs every service in the foreground and stops them all on Ctrl-C.
run-all:
	@trap 'kill 0' INT TERM; \
	for s in $(SERVICES); do \
		go run ./services/$$s/cmd/server & \
	done; \
	wait

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
