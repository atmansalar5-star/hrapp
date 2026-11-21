// آزمون API فرهنگ داده: کنترل دسترسی و رفتار روی استقرار بدون پایگاه‌داده.
// ارجاع: docs/data-dictionary-fa.md بند DD-04.

package bootstrap

import (
	"net/http"
	"strings"
	"testing"
)

// TestDD04DictionaryAPIAdminOnly می‌سنجد مسیر فرهنگ داده بدون نشست ۴۰۱ بدهد، برای نقش
// غیرمدیر ۴۰۳ و روی استقرار درون‌حافظه‌ای (بدون پایگاه‌داده) با پیام روشن ۵۰۳.
func TestDD04DictionaryAPIAdminOnly(t *testing.T) {
	handler := newTestServer(t)
	const path = "/api/v1/platform/data-dictionary"

	if rec := call(t, handler, http.MethodGet, path, "", nil); rec.Code != http.StatusUnauthorized {
		t.Fatalf("بدون نشست باید ۴۰۱ باشد، %d شد", rec.Code)
	}

	// نقش غیرمدیر: «کارشناس منابع انسانی» اجازهٔ دیدن نقشهٔ داده را ندارد.
	specialist := login(t, handler, "hr.spec", "demo")
	if rec := call(t, handler, http.MethodGet, path, specialist.AccessToken, nil); rec.Code != http.StatusForbidden {
		t.Fatalf("نقش غیرمدیر باید ۴۰۳ بگیرد، %d گرفت: %s", rec.Code, rec.Body.String())
	}

	// مدیر سیستم: روی انبار درون‌حافظه‌ای (بدون پایگاه‌داده) پاسخ روشن ۵۰۳ می‌گیرد.
	admin := login(t, handler, "admin", "demo")
	rec := call(t, handler, http.MethodGet, path, admin.AccessToken, nil)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("بدون پایگاه‌داده باید ۵۰۳ باشد، %d شد: %s", rec.Code, rec.Body.String())
	}
	if body := rec.Body.String(); !strings.Contains(body, "پایگاه‌داده") {
		t.Fatalf("پیام باید توضیح دهد فرهنگ داده به پایگاه‌داده نیاز دارد: %s", body)
	}

	// قالب ناشناخته روی همان مسیر: پیش از رسیدن به پایگاه‌داده هم نادیده گرفته نمی‌شود.
	rec = call(t, handler, http.MethodGet, path+"?format=nope", admin.AccessToken, nil)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("قالب ناشناخته روی استقرار بدون پایگاه‌داده باید ۵۰۳ بدهد، %d شد", rec.Code)
	}
}
