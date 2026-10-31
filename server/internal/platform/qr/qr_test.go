// آزمون‌های QR کد: خروجی SVG/PNG، امضای نشانی تأیید و ساخت نشانی با کدگذاری امن.

package qr

import (
	"bytes"
	"image/png"
	"strings"
	"testing"
)

func TestSVGAndPNG(t *testing.T) {
	link := "https://hrms.local/verify/employee/EMP-001"
	svg, err := SVG(link, 200)
	if err != nil {
		t.Fatalf("ساخت SVG نباید خطا بدهد: %v", err)
	}
	if !strings.HasPrefix(svg, "<?xml") || !strings.Contains(svg, "<svg") || !strings.Contains(svg, "<rect") {
		t.Errorf("خروجی SVG نامعتبر است: %.80s", svg)
	}
	if !strings.Contains(svg, `width="200"`) {
		t.Errorf("اندازهٔ SVG باید ۲۰۰ باشد: %.120s", svg)
	}

	data, err := PNG(link, 200)
	if err != nil {
		t.Fatalf("ساخت PNG نباید خطا بدهد: %v", err)
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("خروجی باید PNG معتبر باشد: %v", err)
	}
	if b := img.Bounds(); b.Dx() < 100 || b.Dy() < 100 {
		t.Errorf("اندازهٔ تصویر QR کوچک است: %v", b)
	}

	if _, err := SVG("", 200); err == nil {
		t.Errorf("محتوای خالی باید خطا بدهد")
	}
	if _, err := SVG(link, 10); err != nil {
		t.Errorf("اندازهٔ کوچک باید به کران پایین بسته شود، نه خطا: %v", err)
	}
}

func TestSignAndCheck(t *testing.T) {
	key := []byte("test-key")
	token := Sign(key, "document", "employment_certificate", "EMP-001", "CRT-1405-1")
	if len(token) != 16 {
		t.Fatalf("طول امضا باید ۱۶ نویسه باشد، %d شد", len(token))
	}
	if !Check(key, token, "document", "employment_certificate", "EMP-001", "CRT-1405-1") {
		t.Errorf("امضای درست باید پذیرفته شود")
	}
	if Check(key, token, "document", "employment_certificate", "EMP-001", "CRT-1405-2") {
		t.Errorf("امضای سریال دیگر نباید پذیرفته شود")
	}
	if Check([]byte("other-key"), token, "document", "employment_certificate", "EMP-001", "CRT-1405-1") {
		t.Errorf("امضا با کلید دیگر نباید پذیرفته شود")
	}
}

func TestVerifyLinkEscaping(t *testing.T) {
	link := VerifyLink("https://hrms.local/", "employee", "EMP-001", map[string]string{
		"kind": "employment_certificate", "serial": "CRT 1405/1", "t": "abc",
	})
	if !strings.HasPrefix(link, "https://hrms.local/verify/employee/EMP-001?") {
		t.Fatalf("ابتدای نشانی درست نیست: %s", link)
	}
	if strings.Contains(link, " ") || !strings.Contains(link, "CRT%201405%2F1") {
		t.Errorf("مقدارهای پرس‌وجو باید کدگذاری شوند: %s", link)
	}
	if !strings.HasSuffix(link, "t=abc") {
		t.Errorf("کلیدهای پرس‌وجو باید مرتب باشند: %s", link)
	}
	if got := VerifyLink("https://hrms.local", "company", "bipc", nil); got != "https://hrms.local/verify/company/bipc" {
		t.Errorf("نشانی بدون پرس‌وجو ساده است: %s", got)
	}
}
