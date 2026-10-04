package comparators

import "strings"

// Split modes for the unordered comparator.
const (
	SplitLines  = "lines"
	SplitTokens = "tokens"
)

// UnorderedParams holds the unordered comparator flags.
type UnorderedParams struct {
	Split            string
	IgnoreDuplicates bool
	IgnoreCase       bool
}

// ParseUnorderedParams reads the unordered comparator flags.
func ParseUnorderedParams(p map[string]any) (UnorderedParams, error) {
	params := UnorderedParams{
		Split:            SplitLines,
		IgnoreDuplicates: paramBool(p, "ignore_duplicates"),
		IgnoreCase:       paramBool(p, "ignore_case"),
	}
	if raw := paramString(p, "split"); raw != "" {
		if raw != SplitLines && raw != SplitTokens {
			return params, &ComparatorError{Code: CodeParamOutOfRange, Field: "split", Detail: raw}
		}
		params.Split = raw
	}
	return params, nil
}

// CompareUnordered compares both sides as multisets, either per line or per
// token. With IgnoreDuplicates each side collapses to its distinct set.
func CompareUnordered(expected, actual string, params UnorderedParams) Result {
	want := splitItems(expected, params)
	got := splitItems(actual, params)
	if params.IgnoreDuplicates {
		want = distinctItems(want)
		got = distinctItems(got)
	}
	counts := make(map[string]int, len(want))
	for _, item := range want {
		counts[item]++
	}
	for _, item := range got {
		counts[item]--
		if counts[item] < 0 {
			return Result{Verdict: VerdictWA, Message: "unordered output differs"}
		}
	}
	for _, left := range counts {
		if left != 0 {
			return Result{Verdict: VerdictWA, Message: "unordered output differs"}
		}
	}
	return Result{Verdict: VerdictAC}
}

func splitItems(s string, params UnorderedParams) []string {
	var items []string
	if params.Split == SplitTokens {
		items = strings.Fields(s)
	} else {
		items = strings.Split(s, "\n")
	}
	out := items[:0]
	for _, item := range items {
		if params.IgnoreCase {
			item = strings.ToLower(item)
		}
		if item == "" {
			continue
		}
		out = append(out, item)
	}
	return out
}

func distinctItems(items []string) []string {
	seen := make(map[string]struct{}, len(items))
	out := make([]string, 0, len(items))
	for _, item := range items {
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}
