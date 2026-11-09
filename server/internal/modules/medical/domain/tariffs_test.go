// آزمون دادهٔ تعرفهٔ ۱۴۰۵ — نگهبان فایل بذرکاری تعرفه‌ها.
//
// چرا این آزمون: تعرفه‌ها «دادهٔ مصوب» هستند و از فایل JSON بذر می‌شوند؛ اگر این فایل خراب،
// ناقص یا دارای کد تکراری شود، سامانه بی‌صدا با فهرست ناقص بالا می‌آید. پس داده باید در آزمون
// بررسی شود: شمار ردیف، یکتایی کد، دستهٔ شناس‌شناخته و مبلغ مثبت.
package domain

import (
	"regexp"
	"testing"

	"hrms/internal/rules"
)

var tariffCodePattern = regexp.MustCompile(`^MT-[A-Z]{3}-[0-9]{3}$`)

func TestTariffSeed1405LoadsAllRows(t *testing.T) {
	rows := TariffSeed1405()
	if len(rows) != 87 {
		t.Fatalf("شمار ردیف تعرفهٔ ۱۴۰۵: %d، انتظار ۸۷ ردیف", len(rows))
	}
	categories := map[string]bool{}
	for _, c := range rules.MedicalAidCategories() {
		categories[c.Key] = true
	}
	codes := map[string]bool{}
	seen := map[string]int{}
	for i, row := range rows {
		if row.Kind != KindTariffs {
			t.Errorf("ردیف %d: نوع رکورد %q است، انتظار %q", i, row.Kind, KindTariffs)
		}
		if row.Title == "" {
			t.Errorf("ردیف %d: عنوان خالی است", i)
		}
		code := row.Fields["code"]
		if !tariffCodePattern.MatchString(code) {
			t.Errorf("ردیف %d: کد تعرفه نامعتبر %q", i, code)
		}
		if codes[code] {
			t.Errorf("ردیف %d: کد تکراری %q", i, code)
		}
		codes[code] = true
		if !categories[row.Fields["category"]] {
			t.Errorf("ردیف %s: دستهٔ ناشناخته %q", code, row.Fields["category"])
		}
		seen[row.Fields["category"]]++
		if row.Fields["amount"] == "" || row.Fields["amount"] == "0" {
			t.Errorf("ردیف %s: مبلغ تعرفه خالی یا صفر است", code)
		}
		if row.Fields["effectiveFrom"] != "1405/01/01" {
			t.Errorf("ردیف %s: تاریخ اجرا %q است، انتظار 1405/01/01", code, row.Fields["effectiveFrom"])
		}
		// تعرفه باید «جاری» بذر شود؛ تعرفهٔ پیش‌نویس در محاسبه نادیده گرفته می‌شود.
		if row.SeedStatus != "active" {
			t.Errorf("ردیف %s: وضعیت بذر %q است، انتظار active", code, row.SeedStatus)
		}
	}
	// هر هشت دستهٔ کمک هزینه باید در دستورالعمل ۱۴۰۵ ردیف داشته باشند یا آگاهانه خالی باشند.
	for _, c := range rules.MedicalAidCategories() {
		if seen[c.Key] == 0 {
			t.Logf("دستهٔ %q در دستورالعمل ۱۴۰۵ ردیف تعرفه ندارد (بر پایهٔ فاکتور پرداخت می‌شود)", c.Key)
		}
	}
}

func TestTariffSeed1405FlagValues(t *testing.T) {
	for _, row := range TariffSeed1405() {
		if v, ok := row.Fields["requirePrescription"]; ok && v != "yes" {
			t.Errorf("ردیف %s: مقدار پرچم نسخه %q است، انتظار yes", row.Fields["code"], v)
		}
	}
}
