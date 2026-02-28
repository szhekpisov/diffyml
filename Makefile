.PHONY: build coverage check-coverage bench bench-cpu bench-mem bench-compare govulncheck golangci-lint security test e2e fmt lint vet ci fixture changelog fuzz fuzz-long mutation mutation-dry

BIN = /tmp/diffyml-dev

build:
	go build -o $(BIN) .

coverage:
	go test -coverprofile=coverage.out ./pkg/...
	go tool cover -html=coverage.out

check-coverage:
	@go test ./pkg/diffyml/ -coverprofile=coverage.out
	@COVER_OUTPUT=$$(go tool cover -func=coverage.out); \
	get_file_coverage() { \
		echo "$$COVER_OUTPUT" | grep "^.*/$${1}:" | tail -1 | awk '{print $$NF}' | tr -d '%'; \
	}; \
	PARSER_COV=$$(get_file_coverage "parser.go"); \
	ORDERED_MAP_COV=$$(get_file_coverage "ordered_map.go"); \
	KUBERNETES_COV=$$(get_file_coverage "kubernetes.go"); \
	echo ""; \
	echo "=== Coverage Summary ==="; \
	printf "%-20s %8s %10s %s\n" "File" "Actual" "Required" "Status"; \
	printf "%-20s %8s %10s %s\n" "----" "------" "--------" "------"; \
	FAIL=0; \
	check_threshold() { \
		local file="$$1" actual="$$2" required="$$3"; \
		local status="PASS"; \
		if [ "$$(echo "$$actual < $$required" | bc -l)" -eq 1 ]; then \
			status="FAIL"; \
			FAIL=1; \
		fi; \
		printf "%-20s %7s%% %9s%% %s\n" "$$file" "$$actual" "$$required" "$$status"; \
	}; \
	check_threshold "parser.go"      "$$PARSER_COV"      "100.0"; \
	check_threshold "ordered_map.go" "$$ORDERED_MAP_COV" "100.0"; \
	check_threshold "kubernetes.go"  "$$KUBERNETES_COV"  "95.0"; \
	echo ""; \
	if [ "$$FAIL" -eq 1 ]; then \
		echo "Coverage threshold check FAILED"; \
		exit 1; \
	fi; \
	echo "All coverage thresholds passed"

bench:
	go test -bench=. -benchmem -count=1 ./pkg/diffyml/

bench-cpu:
	go test -bench=. -benchmem -count=1 -cpuprofile=cpu.prof ./pkg/diffyml/

bench-mem:
	go test -bench=. -benchmem -count=1 -memprofile=mem.prof ./pkg/diffyml/

bench-compare:
	bash bench/compare/run.sh

govulncheck:
	go install golang.org/x/vuln/cmd/govulncheck@latest
	$$(go env GOPATH)/bin/govulncheck ./...

golangci-lint:
	go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest run ./...

security: govulncheck golangci-lint

test:
	go test ./...

e2e: build
	go test -v -timeout 120s ./test/e2e/

fmt:
	gofmt -l -w .

lint: golangci-lint

vet:
	go vet ./...

changelog:
	git cliff --output CHANGELOG.md

fuzz:
	@for target in FuzzCompare FuzzCompareWithOptions FuzzParseWithOrder FuzzDocumentParser; do \
		echo "=== Fuzzing $$target for 30s ==="; \
		go test -fuzz="^$${target}$$" -fuzztime=30s -run='^$$' ./pkg/diffyml/; \
	done

fuzz-long:
	@for target in FuzzCompare FuzzCompareWithOptions FuzzParseWithOrder FuzzDocumentParser; do \
		echo "=== Fuzzing $$target for 5m ==="; \
		go test -fuzz="^$${target}$$" -fuzztime=5m -run='^$$' ./pkg/diffyml/; \
	done

mutation:
	gremlins unleash --workers 5 --coverpkg="./pkg/diffyml/..." --output=mutation-report.json ./pkg/diffyml/
	go clean -cache

mutation-dry:
	gremlins unleash --dry-run --coverpkg="./pkg/diffyml/..." ./pkg/diffyml/

ci: fmt vet test check-coverage security

fixture: build
	@if [ -z "$(N)" ]; then echo "Usage: make fixture N=<number>  (e.g. make fixture N=1)"; exit 1; fi
	@DIR=$$(printf "testdata/fixtures/%03d-*" $(N)); \
	DIR=$$(echo $$DIR); \
	if [ ! -d "$$DIR" ]; then \
		DIR=$$(printf "testdata/fixtures/%d-*" $(N)); \
		DIR=$$(echo $$DIR); \
	fi; \
	if [ ! -d "$$DIR" ]; then echo "Fixture $(N) not found"; exit 1; fi; \
	PARAMS=""; \
	if [ -f "$$DIR/params.cfg" ]; then PARAMS=$$(grep -v '^#' "$$DIR/params.cfg" | tr '\n' ' '); fi; \
	echo "=== Running fixture: $$DIR ==="; \
	if [ -d "$$DIR/dir1" ] && [ -d "$$DIR/dir2" ]; then \
		eval $(BIN) --color off --set-exit-code $$PARAMS "$$DIR/dir1" "$$DIR/dir2"; \
	else \
		eval $(BIN) --color off --set-exit-code $$PARAMS "$$DIR/file1.yaml" "$$DIR/file2.yaml"; \
	fi; \
	RC=$$?; echo ""; echo "exit code: $$RC"
