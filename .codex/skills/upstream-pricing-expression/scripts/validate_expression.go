// Validate a generated billing expression against the repository's actual
// compiler and a small set of finite, non-negative token vectors.
//
// Run from the repository root, for example:
//
//	go run .codex/skills/upstream-pricing-expression/scripts/validate_expression.go \
//	  -expr 'tier("base", p * 1.5 + cr * 0.05 + c * 4.5)'
//
// This validates syntax and arithmetic only. Time functions still use the
// current wall clock; boundary behavior must be checked separately.
package main

import (
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"strings"

	"github.com/QuantumNous/new-api/pkg/billingexpr"
)

func main() {
	var exprText string
	var exprFile string
	flag.StringVar(&exprText, "expr", "", "billing expression text")
	flag.StringVar(&exprFile, "file", "", "read the billing expression from a file")
	flag.Parse()

	if exprText != "" && exprFile != "" {
		fail("use either -expr or -file, not both")
	}
	if exprFile != "" {
		data, err := os.ReadFile(exprFile)
		if err != nil {
			fail("read expression file: %v", err)
		}
		exprText = string(data)
	} else if exprText == "" {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			fail("read expression from stdin: %v", err)
		}
		exprText = string(data)
	}

	exprText = strings.TrimSpace(exprText)
	if exprText == "" {
		fail("expression is empty")
	}
	if _, err := billingexpr.CompileFromCache(exprText); err != nil {
		fail("compile: %v", err)
	}

	type vector struct {
		name    string
		params  billingexpr.TokenParams
		request billingexpr.RequestInput
	}
	vectors := []vector{
		{name: "zero", params: billingexpr.TokenParams{}},
		{name: "basic", params: billingexpr.TokenParams{P: 1_000, C: 500, Len: 1_000}},
		{name: "cache", params: billingexpr.TokenParams{P: 100_000, C: 10_000, Len: 100_000, CR: 20_000, CC: 5_000, CC1h: 2_000}},
		{
			name: "multimodal",
			params: billingexpr.TokenParams{
				P: 1_000_000, C: 100_000, Len: 1_000_000, CR: 300_000,
				CC: 10_000, CC1h: 5_000, Img: 2_000, ImgO: 500, AI: 1_000, AO: 500,
			},
		},
		{
			name:   "request",
			params: billingexpr.TokenParams{P: 2_000, C: 1_000, Len: 2_000},
			request: billingexpr.RequestInput{
				Headers: map[string]string{"x-billing-test": "enabled"},
				Body:    []byte(`{"service_tier":"priority","stream":true}`),
			},
		},
	}
	for i, vector := range vectors {
		cost, trace, err := billingexpr.RunExprWithRequest(exprText, vector.params, vector.request)
		if err != nil {
			fail("vector %d (%s): run: %v", i, vector.name, err)
		}
		if cost < 0 || math.IsNaN(cost) || math.IsInf(cost, 0) {
			fail("vector %d (%s): invalid cost %v", i, vector.name, cost)
		}
		fmt.Printf("vector=%d name=%s cost=%g tier=%s\n", i, vector.name, cost, trace.MatchedTier)
	}
	fmt.Println("expression validation: ok")
}

func fail(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "expression validation: "+format+"\n", args...)
	os.Exit(1)
}
