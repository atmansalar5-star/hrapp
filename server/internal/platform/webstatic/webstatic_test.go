// آزمون مسیر فایل ایستا: پیمایش مسیر و فایل پنهان باید رد شود.
// ارجاع: docs/security-spec-fa.md بند REQ-10.

package webstatic

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestREQ10StaticPathTraversal می‌سنجد فایل عمومی سرو می‌شود، ولی `..`، فایل پنهان و
// کاراکتر تهی رد می‌شوند.
func TestREQ10StaticPathTraversal(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<h1>ok</h1>"), 0o600); err != nil {
		t.Fatalf("ساخت فایل آزمون ناموفق: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "js"), 0o755); err != nil {
		t.Fatalf("ساخت پوشهٔ آزمون ناموفق: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "js", "app.js"), []byte("console.log(1)"), 0o600); err != nil {
		t.Fatalf("ساخت فایل js ناموفق: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("HRMS_JWT_SECRET=leak"), 0o600); err != nil {
		t.Fatalf("ساخت فایل پنهان ناموفق: %v", err)
	}

	handler := Handler(dir, nil)

	allowed := []string{"/", "/js/app.js"}
	for _, path := range allowed {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("مسیر مجاز %s کد %d داد", path, rec.Code)
		}
	}
	// http.FileServer برای /index.html به ریشه هدایت می‌کند؛ ۳۰۱ رفتار درست است.
	recIndex := httptest.NewRecorder()
	handler.ServeHTTP(recIndex, httptest.NewRequest(http.MethodGet, "/index.html", nil))
	if recIndex.Code != http.StatusMovedPermanently {
		t.Fatalf("درخواست /index.html باید به ریشه هدایت شود، کد %d", recIndex.Code)
	}

	blocked := []string{"/.env", "/../.env", "/js/../../.env", "/%2e%2e/.env"}
	for _, path := range blocked {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != http.StatusNotFound {
			t.Fatalf("مسیر ناامن %s باید ۴۰۴ بگیرد، کد %d گرفت (بدنه: %s)", path, rec.Code, rec.Body.String())
		}
		if body := rec.Body.String(); strings.Contains(body, "leak") {
			t.Fatalf("محتوای فایل پنهان از مسیر %s نشت کرد", path)
		}
	}

	// کاراکتر تهی در مسیر.
	if safePath("/x\x00y") {
		t.Fatal("مسیر دارای کاراکتر تهی باید رد شود")
	}
}
