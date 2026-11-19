// آزمون کمکی‌های ابزار عملیاتی hrops (بازنویسی DSN، اعتبار نام، نقل‌قول SQL).

package main

import "testing"

func TestReplaceDatabase(t *testing.T) {
	cases := []struct{ in, name, want string }{
		{"postgres://u:p@127.0.0.1:5432/hrms?sslmode=disable", "postgres", "postgres://u:p@127.0.0.1:5432/postgres?sslmode=disable"},
		{"postgres://u:p@127.0.0.1:5432/hrms", "hrms_ops", "postgres://u:p@127.0.0.1:5432/hrms_ops"},
		{"postgres://u@host/hrms?x=1&y=2", "postgres", "postgres://u@host/postgres?x=1&y=2"},
	}
	for _, c := range cases {
		if got := replaceDatabase(c.in, c.name); got != c.want {
			t.Fatalf("replaceDatabase(%q, %q) = %q، انتظار %q", c.in, c.name, got, c.want)
		}
	}
}

func TestValidIdent(t *testing.T) {
	ok := []string{"hrms_app", "hrms_ops", "a", "h1"}
	bad := []string{"", "HRMS", "1abc", "hrms-app", "hrms app", "hrms;drop"}
	for _, v := range ok {
		if !validIdent(v) {
			t.Fatalf("«%s» باید معتبر باشد", v)
		}
	}
	for _, v := range bad {
		if validIdent(v) {
			t.Fatalf("«%s» نباید معتبر باشد", v)
		}
	}
}

func TestQuoteHelpers(t *testing.T) {
	if got := quoteIdent(`a"b`); got != `"a""b"` {
		t.Fatalf("quoteIdent نادرست: %s", got)
	}
	if got := quoteLiteral(`pa'ss`); got != `'pa''ss'` {
		t.Fatalf("quoteLiteral نادرست: %s", got)
	}
}
