.PHONY: coverage check-coverage bench bench-cpu bench-mem bench-compare govulncheck golangci-lint security test fmt lint vet ci

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

fmt:
	gofmt -l -w .

lint: golangci-lint

vet:
	go vet ./...

ci: fmt vet test check-coverage security
