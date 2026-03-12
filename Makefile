.PHONY: build coverage check-coverage check-doc bench bench-cpu bench-mem bench-compare govulncheck golangci-lint security test e2e fmt lint vet ci fixture changelog fuzz fuzz-long mutation mutation-dry

BIN = /tmp/diffyml-dev

build:
	go build -o $(BIN) .

coverage:
	go test -coverprofile=coverage.out ./pkg/...
	go tool cover -html=coverage.out

check-coverage:
	@go test ./pkg/diffyml/ -coverprofile=coverage.out
	@COVER_OUTPUT=$$(go tool cover -func=coverage.out); \
	TOTAL_COV=$$(echo "$$COVER_OUTPUT" | grep '^total:' | awk '{print $$NF}' | tr -d '%'); \
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
	check_threshold "TOTAL" "$$TOTAL_COV" "99.0"; \
	echo ""; \
	if [ "$$FAIL" -eq 1 ]; then \
		echo "Coverage threshold check FAILED"; \
		exit 1; \
	fi; \
	echo "All coverage thresholds passed"

check-doc:
	@DOC=pkg/diffyml/doc.go; \
	EXCLUDE=pkg/diffyml/.doc-exclude; \
	FAIL=0; \
	echo ""; \
	echo "=== doc.go Sync Check ==="; \
	DOC_LINKS=$$(grep -oE '\[[A-Z][A-Za-z0-9]*\]' "$$DOC" | tr -d '[]' | sort -u); \
	for link in $$DOC_LINKS; do \
		if ! go doc ./pkg/diffyml/ "$$link" >/dev/null 2>&1; then \
			echo "  BROKEN: [$$link] references a non-existent symbol"; \
			FAIL=1; \
		fi; \
	done; \
	EXCLUDES=""; \
	if [ -f "$$EXCLUDE" ]; then EXCLUDES=$$(grep -vE '^(#|$$)' "$$EXCLUDE"); fi; \
	for sym in $$(find pkg/diffyml -maxdepth 1 -type f -name '*.go' ! -name '*_test.go' -exec grep -hE '^(type|func) [A-Z][A-Za-z0-9]+' {} + | sed -E 's/^(type|func) ([A-Z][A-Za-z0-9]*).*/\2/' | sort -u); do \
		if echo "$$DOC_LINKS" | grep -qx "$$sym"; then continue; fi; \
		if [ -n "$$EXCLUDES" ] && echo "$$EXCLUDES" | grep -qx "$$sym"; then continue; fi; \
		echo "  UNDOCUMENTED: $$sym (add [$$sym] to doc.go or $$sym to .doc-exclude)"; \
		FAIL=1; \
	done; \
	echo ""; \
	if [ "$$FAIL" -eq 1 ]; then echo "doc.go sync check FAILED"; exit 1; fi; \
	echo "doc.go sync check passed"

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
	gremlins unleash --workers 10 --coverpkg="./pkg/diffyml/..." --output=mutation-report.json ./pkg/diffyml/
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
