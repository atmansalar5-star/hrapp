package main

import "testing"

// lanURLs فقط وقتی نشانی شبکه می‌سازد که برنامه روی همهٔ کارت‌های شبکه شنود کند؛
// برای نشانی حلقه‌ای یا یک کارت مشخص باید خالی باشد تا در لاگ گمراه‌کننده نباشد.
func TestLanURLsOnlyForAllInterfaces(t *testing.T) {
	for _, addr := range []string{"127.0.0.1:8099", "192.168.1.5:8099", "[::1]:8099", "bad-addr", ""} {
		if got := lanURLs(addr); got != nil {
			t.Fatalf("lanURLs(%q) باید خالی باشد، شد %v", addr, got)
		}
	}
}

func TestDisplayURL(t *testing.T) {
	cases := map[string]string{
		"127.0.0.1:8099": "http://127.0.0.1:8099",
		"0.0.0.0:8099":   "http://127.0.0.1:8099",
		":8099":          "http://127.0.0.1:8099",
	}
	for in, want := range cases {
		if got := displayURL(in); got != want {
			t.Fatalf("displayURL(%q)=%q، انتظار %q", in, got, want)
		}
	}
}
