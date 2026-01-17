// generate_testdata generates YAML test file pairs at various sizes
// for benchmarking diffyml against alternative YAML diff tools.
//
// Usage:
//
//	go run generate_testdata.go --size small --output-dir /tmp/bench-data/small
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var sizes = map[string]int{
	"small":  4,
	"medium": 42,
	"large":  420,
	"xlarge": 4200,
}

func main() {
	size := flag.String("size", "medium", "Size preset: small, medium, large, xlarge")
	outputDir := flag.String("output-dir", ".", "Directory to write from.yaml and to.yaml")
	flag.Parse()

	n, ok := sizes[*size]
	if !ok {
		fmt.Fprintf(os.Stderr, "unknown size %q (valid: small, medium, large, xlarge)\n", *size)
		os.Exit(1)
	}

	if err := os.MkdirAll(*outputDir, 0o750); err != nil {
		fmt.Fprintf(os.Stderr, "mkdir: %v\n", err)
		os.Exit(1)
	}

	from := generateFrom(n)
	to := generateTo(n)

	if err := os.WriteFile(filepath.Join(*outputDir, "from.yaml"), from, 0o600); err != nil {
		fmt.Fprintf(os.Stderr, "write from.yaml: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(filepath.Join(*outputDir, "to.yaml"), to, 0o600); err != nil {
		fmt.Fprintf(os.Stderr, "write to.yaml: %v\n", err)
		os.Exit(1)
	}

	fi, _ := os.Stat(filepath.Join(*outputDir, "from.yaml"))
	fmt.Printf("Generated %s: %d services, from.yaml=%d bytes, to.yaml=%d bytes\n",
		*size, n, fi.Size(), len(to))
}

// header writes a realistic metadata + config block matching the style
// found in testdata/perf/test1/file1.yaml.
func header() string {
	return `metadata:
  name: performance-test
  version: 1.0.0
  environment: production
  region: us-east-1
  replicas: 3

config:
  database:
    host: db.example.com
    port: 5432
    username: admin
    password: secret123
    pool_size: 10
    timeout: 30
  cache:
    enabled: true
    ttl: 3600
    max_size: 1000
  logging:
    level: info
    format: json
    output: stdout

`
}

// headerModified writes a slightly modified metadata + config block.
func headerModified() string {
	return `metadata:
  name: performance-test
  version: 1.1.0
  environment: production
  region: us-west-2
  replicas: 5

config:
  database:
    host: db.example.com
    port: 5432
    username: admin
    password: secret123
    pool_size: 20
    timeout: 30
  cache:
    enabled: true
    ttl: 7200
    max_size: 2000
  logging:
    level: warn
    format: json
    output: stdout

`
}

// generateFrom generates the "from" YAML file with n services.
// Ported from pkg/diffyml/benchmark_test.go:generateServiceList.
func generateFrom(n int) []byte {
	var b strings.Builder
	b.WriteString(header())
	b.WriteString("services:\n")
	for i := 0; i < n; i++ {
		fmt.Fprintf(&b, "- name: service-%03d\n", i)
		fmt.Fprintf(&b, "  version: 1.0.%d\n", i%10)
		fmt.Fprintf(&b, "  replicas: %d\n", 1+(i%5))
		fmt.Fprintf(&b, "  memory: %dMi\n", 256+(i%4)*128)
		fmt.Fprintf(&b, "  cpu: %dm\n", 100+(i%4)*50)
		b.WriteString("  enabled: true\n")
		fmt.Fprintf(&b, "  port: %d\n", 8000+i)
		b.WriteString("  protocol: http\n")
		fmt.Fprintf(&b, "  timeout: %d\n", 30+(i%3)*10)
		b.WriteString("  labels:\n")
		b.WriteString("    tier: backend\n")
		fmt.Fprintf(&b, "    team: team-%d\n", i%5)
	}
	return []byte(b.String())
}

// generateTo generates the "to" YAML file with ~20% modifications.
// Ported from pkg/diffyml/benchmark_test.go:generateServiceListModified.
func generateTo(n int) []byte {
	var b strings.Builder
	b.WriteString(headerModified())
	b.WriteString("services:\n")

	// Skip first 2 services (removed), add the rest with modifications.
	removed := 2
	added := n / 10
	if added < 1 {
		added = 1
	}
	for i := removed; i < n; i++ {
		fmt.Fprintf(&b, "- name: service-%03d\n", i)
		if i%5 == 0 {
			fmt.Fprintf(&b, "  version: 2.0.%d\n", i%10)
		} else {
			fmt.Fprintf(&b, "  version: 1.0.%d\n", i%10)
		}
		if i%5 == 1 {
			fmt.Fprintf(&b, "  replicas: %d\n", 3+(i%3))
		} else {
			fmt.Fprintf(&b, "  replicas: %d\n", 1+(i%5))
		}
		fmt.Fprintf(&b, "  memory: %dMi\n", 256+(i%4)*128)
		fmt.Fprintf(&b, "  cpu: %dm\n", 100+(i%4)*50)
		b.WriteString("  enabled: true\n")
		fmt.Fprintf(&b, "  port: %d\n", 8000+i)
		b.WriteString("  protocol: http\n")
		fmt.Fprintf(&b, "  timeout: %d\n", 30+(i%3)*10)
		b.WriteString("  labels:\n")
		b.WriteString("    tier: backend\n")
		fmt.Fprintf(&b, "    team: team-%d\n", i%5)
	}
	// Add new services
	for i := n; i < n+added; i++ {
		fmt.Fprintf(&b, "- name: service-%03d\n", i)
		fmt.Fprintf(&b, "  version: 1.0.%d\n", i%10)
		fmt.Fprintf(&b, "  replicas: %d\n", 1+(i%5))
		fmt.Fprintf(&b, "  memory: %dMi\n", 256+(i%4)*128)
		fmt.Fprintf(&b, "  cpu: %dm\n", 100+(i%4)*50)
		b.WriteString("  enabled: true\n")
		fmt.Fprintf(&b, "  port: %d\n", 8000+i)
		b.WriteString("  protocol: http\n")
		fmt.Fprintf(&b, "  timeout: %d\n", 30+(i%3)*10)
		b.WriteString("  labels:\n")
		b.WriteString("    tier: backend\n")
		fmt.Fprintf(&b, "    team: team-%d\n", i%5)
	}
	return []byte(b.String())
}
