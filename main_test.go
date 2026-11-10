package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestParseTarget(t *testing.T) {
	tests := []struct {
		in      string
		want    target
		wantErr bool
	}{
		{in: "built-fast", want: target{Owner: "built-fast"}},
		{in: "built-fast/foo", want: target{Owner: "built-fast", Repo: "foo"}},
		{in: "built-fast/foo.bar_baz", want: target{Owner: "built-fast", Repo: "foo.bar_baz"}},
		{in: "-bad", wantErr: true},
		{in: "built-fast/", wantErr: true},
		{in: "built-fast/..", wantErr: true},
		{in: "a/b/c", wantErr: true},
		{in: "a?b", wantErr: true},
	}

	for _, tt := range tests {
		got, err := parseTarget(tt.in)
		if (err != nil) != tt.wantErr {
			t.Errorf("parseTarget(%q) err = %v, wantErr %v", tt.in, err, tt.wantErr)
			continue
		}
		if got != tt.want {
			t.Errorf("parseTarget(%q) = %+v, want %+v", tt.in, got, tt.want)
		}
	}
}

func TestParseMonth(t *testing.T) {
	now := time.Date(2026, time.November, 10, 0, 0, 0, 0, time.UTC)

	got, err := parseMonth("", now)
	if err != nil || got != (period{2026, time.November}) {
		t.Errorf("parseMonth(\"\") = %+v, %v", got, err)
	}

	got, err = parseMonth("2025-09", now)
	if err != nil || got != (period{2025, time.September}) {
		t.Errorf("parseMonth(2025-09) = %+v, %v", got, err)
	}

	for _, bad := range []string{"2025-13", "2025-9", "09-2025", "2025-09-01"} {
		if _, err := parseMonth(bad, now); err == nil {
			t.Errorf("parseMonth(%q) expected error", bad)
		}
	}
}

type fakeClient struct {
	responses map[string]string
	paths     []string
}

func (f *fakeClient) Get(path string, resp interface{}) error {
	f.paths = append(f.paths, path)
	body, ok := f.responses[path]
	if !ok {
		return &json.UnsupportedValueError{Str: "unexpected path " + path}
	}
	return json.Unmarshal([]byte(body), resp)
}

func TestFetchSummary(t *testing.T) {
	p := period{2026, time.November}

	tests := []struct {
		name     string
		target   target
		userType string
		wantPath string
	}{
		{"org", target{Owner: "built-fast"}, "Organization",
			"organizations/built-fast/settings/billing/usage/summary?month=11&product=Actions&year=2026"},
		{"org repo", target{Owner: "built-fast", Repo: "foo"}, "Organization",
			"organizations/built-fast/settings/billing/usage/summary?month=11&product=Actions&repository=built-fast%2Ffoo&year=2026"},
		{"user", target{Owner: "octocat"}, "User",
			"users/octocat/settings/billing/usage/summary?month=11&product=Actions&year=2026"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &fakeClient{responses: map[string]string{
				"users/" + tt.target.Owner: `{"type":"` + tt.userType + `"}`,
				tt.wantPath:                `{"usageItems":[{"sku":"actions_linux","grossQuantity":10}]}`,
			}}
			s, err := fetchSummary(c, tt.target, p)
			if err != nil {
				t.Fatalf("fetchSummary: %v (paths: %v)", err, c.paths)
			}
			if len(s.UsageItems) != 1 || s.UsageItems[0].SKU != "actions_linux" {
				t.Errorf("unexpected summary: %+v", s)
			}
		})
	}
}

func TestRender(t *testing.T) {
	s := usageSummary{UsageItems: []usageItem{
		{SKU: "actions_linux", UnitType: "minutes", GrossQuantity: 1234, GrossAmount: 7.404, DiscountAmount: 7.404},
		{SKU: "actions_macos", UnitType: "minutes", GrossQuantity: 602.6774, GrossAmount: 37.366, DiscountAmount: 30, NetAmount: 7.366},
	}}

	var buf bytes.Buffer
	if err := render(&buf, target{Owner: "built-fast"}, period{2026, time.November}, s); err != nil {
		t.Fatal(err)
	}
	out := buf.String()

	for _, want := range []string{
		"GitHub Actions usage for built-fast (Nov 2026)",
		"1,234", "602.68", "$7.40", "$37.37", "$7.37",
		"Total", "$44.77", "$37.40",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

func TestRenderEmpty(t *testing.T) {
	var buf bytes.Buffer
	if err := render(&buf, target{Owner: "o", Repo: "r"}, period{2026, time.January}, usageSummary{}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "No usage.") {
		t.Errorf("unexpected output: %s", buf.String())
	}
}

func TestFormatMoney(t *testing.T) {
	for in, want := range map[float64]string{
		0: "$0.00", 0.001: "$0.00", -0.001: "$0.00", 1234567.891: "$1,234,567.89", -5: "-$5.00",
	} {
		if got := formatMoney(in); got != want {
			t.Errorf("formatMoney(%v) = %q, want %q", in, got, want)
		}
	}
}

func TestFetchByRepo(t *testing.T) {
	c := &fakeClient{responses: map[string]string{
		"users/built-fast": `{"type":"Organization"}`,
		"organizations/built-fast/settings/billing/usage?month=11&product=actions&year=2026": `{"usageItems":[
			{"product":"actions","unitType":"Minutes","quantity":100,"grossAmount":0.6,"discountAmount":0.6,"netAmount":0,"repositoryName":"a"},
			{"product":"actions","unitType":"Minutes","quantity":50,"grossAmount":0.3,"discountAmount":0,"netAmount":0.3,"repositoryName":"a"},
			{"product":"actions","unitType":"GigabyteHours","quantity":9,"grossAmount":0.01,"discountAmount":0.01,"netAmount":0,"repositoryName":"a"},
			{"product":"actions","unitType":"Minutes","quantity":10,"grossAmount":1,"discountAmount":0,"netAmount":1,"repositoryName":"b"},
			{"product":"actions","unitType":"Minutes","quantity":5,"grossAmount":0.03,"discountAmount":0.03,"netAmount":0},
			{"product":"packages","unitType":"GigabyteHours","quantity":5,"grossAmount":9,"discountAmount":0,"netAmount":9,"repositoryName":"c"}
		]}`,
	}}

	got, err := fetchByRepo(c, "built-fast", period{2026, time.November})
	if err != nil {
		t.Fatalf("fetchByRepo: %v (paths: %v)", err, c.paths)
	}

	want := []repoUsage{
		{Repo: "b", Minutes: 10, GrossAmount: 1, NetAmount: 1},
		{Repo: "a", Minutes: 150, GrossAmount: 0.91, DiscountAmount: 0.61, NetAmount: 0.3},
		{Repo: noRepo, Minutes: 5, GrossAmount: 0.03, DiscountAmount: 0.03},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d repos, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		g, w := got[i], want[i]
		if g.Repo != w.Repo || g.Minutes != w.Minutes || roundTo(g.GrossAmount, 2) != w.GrossAmount ||
			roundTo(g.DiscountAmount, 2) != w.DiscountAmount || roundTo(g.NetAmount, 2) != w.NetAmount {
			t.Errorf("repo %d = %+v, want %+v", i, g, w)
		}
	}
}

func TestRenderByRepo(t *testing.T) {
	usage := []repoUsage{
		{Repo: "quark", Minutes: 1500, GrossAmount: 9, DiscountAmount: 4, NetAmount: 5},
		{Repo: "vector", Minutes: 20, GrossAmount: 0.12, DiscountAmount: 0.12},
	}

	var buf bytes.Buffer
	if err := renderByRepo(&buf, target{Owner: "built-fast"}, period{2026, time.November}, usage); err != nil {
		t.Fatal(err)
	}
	out := buf.String()

	for _, want := range []string{
		"built-fast by repository (Nov 2026)", "quark", "vector", "1,500", "1,520", "$9.12", "$4.12", "$5.00",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

func TestRunByRepoRejectsRepoTarget(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := run([]string{"--by-repo", "built-fast/foo"}, &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), "--by-repo requires an OWNER") {
		t.Errorf("expected --by-repo error, got %v", err)
	}
}
