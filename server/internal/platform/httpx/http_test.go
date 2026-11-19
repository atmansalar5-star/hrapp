// آزمون سقف بدنهٔ درخواست — بند REQ-09 اسپک امنیتی.

package httpx

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestREQ09RequestBodyLimit می‌سنجد بدنهٔ بزرگ‌تر از سقف با ۴۱۳ رد شود و JSON بدشکل
// با ۴۰۰؛ یعنی هیچ مسیر API بدنهٔ بی‌سقف نمی‌خواند.
func TestREQ09RequestBodyLimit(t *testing.T) {
	if MaxBodyBytes != 1<<20 {
		t.Fatalf("سقف بدنه باید ۱ مگابایت باشد، %d است", MaxBodyBytes)
	}

	oversize := `{"pad":"` + strings.Repeat("x", MaxBodyBytes) + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/x", strings.NewReader(oversize))
	rec := httptest.NewRecorder()
	var body map[string]string
	if DecodeJSON(rec, req, &body) {
		t.Fatal("بدنهٔ بزرگ‌تر از سقف نباید پذیرفته شود")
	}
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("پاسخ باید ۴۱۳ باشد، %d شد", rec.Code)
	}

	bad := httptest.NewRequest(http.MethodPost, "/api/v1/x", strings.NewReader("{not json"))
	badRec := httptest.NewRecorder()
	if DecodeJSON(badRec, bad, &body) {
		t.Fatal("JSON بدشکل نباید پذیرفته شود")
	}
	if badRec.Code != http.StatusBadRequest {
		t.Fatalf("پاسخ باید ۴۰۰ باشد، %d شد", badRec.Code)
	}

	// فیلد ناشناخته رد می‌شود (جلوگیری از تزریق فیلدهای ناخواسته).
	unknown := httptest.NewRequest(http.MethodPost, "/api/v1/x", strings.NewReader(`{"a":1,"evil":true}`))
	unknownRec := httptest.NewRecorder()
	if DecodeJSON(unknownRec, unknown, &struct{ A int }{}) {
		t.Fatal("فیلد ناشناخته نباید پذیرفته شود")
	}
}
