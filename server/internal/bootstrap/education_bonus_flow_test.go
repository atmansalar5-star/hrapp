// آزمون جریان سرتاسری «education_bonus»: سیم‌کشی ماژول‌ها، سیاست‌ها و گردش‌کار.

package bootstrap

import (
	"net/http"
	"strings"
	"testing"

	"hrms/internal/contracts"
)

// TestEducationBonusFlowAndFinanceFile جریان ماژول مستقل پاداش تحصیلی را از اعلام
// گزینه‌ها تا فایل پرداخت ارسالی به مالی می‌سنجد: مقاطع با مبلغ مصوب، محاسبهٔ مشمولان
// ممتاز، بازتعریف مبلغ، قاعدهٔ دامنه، ساخت فایل و خروجی CSV — همان مسیری که رابط وب
// طی می‌کند.
func TestEducationBonusFlowAndFinanceFile(t *testing.T) {
	h := newTestServer(t)
	manager := login(t, h, "hr.manager", "demo")
	token := manager.AccessToken

	// ---------- گزینه‌ها: مقاطع با مبلغ مصوب جاری ----------
	rec := call(t, h, http.MethodGet, "/api/v1/education_bonus/options", token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("گزینه‌های پاداش تحصیلی باید ۲۰۰ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	var options contracts.BonusOptions
	decodeInto(t, rec, &options)
	if len(options.Levels) != 6 {
		t.Fatalf("باید ۶ مقطع اعلام شود، %d شد", len(options.Levels))
	}
	for _, l := range options.Levels {
		if l.Amount <= 0 {
			t.Fatalf("مبلغ مصوب مقطع «%s» باید مثبت باشد", l.Key)
		}
	}
	if options.TodayFa == "" {
		t.Fatal("تاریخ شمسی امروز سرور باید اعلام شود")
	}

	// ---------- محاسبهٔ گروهی: فقط دانش‌آموز ممتاز و مشمول ----------
	preview := func() contracts.BonusResult {
		t.Helper()
		body := map[string]any{"period": "1405/1406", "scope": "group", "trialRun": true}
		rec := call(t, h, http.MethodPost, "/api/v1/education_bonus/bonus/preview", token, body)
		if rec.Code != http.StatusOK {
			t.Fatalf("پیش‌نمایش پاداش باید ۲۰۰ بدهد، %d داد: %s", rec.Code, rec.Body.String())
		}
		var out contracts.BonusResult
		decodeInto(t, rec, &out)
		return out
	}
	before := preview()
	if before.Count == 0 || before.Total <= 0 {
		t.Fatalf("پیش‌نمایش باید ردیف و جمع داشته باشد: %+v", before)
	}
	// دانش‌آموز غیرممتاز نباید در ردیف‌ها باشد.
	for _, r := range before.Rows {
		if r.Student == "زهرا احمدی" || r.Student == "نگار مرادی" {
			t.Fatalf("دانش‌آموز غیرممتاز نباید پاداش بگیرد: %q", r.Student)
		}
	}

	// ---------- بازتعریف مبلغ مصوب: مقطع متوسطه دوم گران‌تر می‌شود ----------
	rates := eduListRecords(t, h, token, "bonusRates", "high2")
	if len(rates) == 0 {
		t.Fatal("مبلغ مصوب مقطع متوسطه دوم در رکوردها پیدا نشد")
	}
	supersede := call(t, h, http.MethodPost, "/api/v1/education_bonus/records/"+rates[0].ID+"/action", token,
		map[string]any{"action": "supersede"})
	if supersede.Code != http.StatusOK {
		t.Fatalf("جایگزینی مبلغ قدیمی باید ۲۰۰ بدهد، %d داد: %s", supersede.Code, supersede.Body.String())
	}
	newRate := call(t, h, http.MethodPost, "/api/v1/education_bonus/records", token, map[string]any{
		"kind": "bonusRates", "title": "متوسطه دوم (مبلغ جدید)",
		"fields": map[string]string{
			"level": "high2", "amount": "30000000",
			"approvalRef": "مصوبهٔ آزمون", "effectiveFrom": "1405/08/01",
		},
	})
	if newRate.Code != http.StatusCreated {
		t.Fatalf("ثبت مبلغ جدید باید ۲۰۱ بدهد، %d داد: %s", newRate.Code, newRate.Body.String())
	}
	after := preview()
	if after.Total <= before.Total {
		t.Fatalf("با مبلغ جدید جمع پاداش باید بیشتر شود: قبل=%d بعد=%d", before.Total, after.Total)
	}

	// ---------- قاعدهٔ دامنه: مشمول بودن بدون نشانهٔ ممتاز رد می‌شود ----------
	bad := call(t, h, http.MethodPost, "/api/v1/education_bonus/records", token, map[string]any{
		"kind": "outstandingStudents", "title": "دانش‌آموز آزمایشی", "employeeId": "EMP-005",
		"fields": map[string]string{"relation": "فرزند", "level": "high2", "excellent": "0", "eligible": "1"},
	})
	if bad.Code != http.StatusUnprocessableEntity {
		t.Fatalf("ثبت مشمول بدون ممتاز بودن باید ۴۲۲ بدهد، %d داد: %s", bad.Code, bad.Body.String())
	}

	// ---------- محاسبهٔ انفرادی بدون انتخاب کارمند رد می‌شود ----------
	invalid := call(t, h, http.MethodPost, "/api/v1/education_bonus/bonus/preview", token, map[string]any{
		"period": "1405/1406", "scope": "individual", "trialRun": true,
	})
	if invalid.Code != http.StatusUnprocessableEntity {
		t.Fatalf("محاسبهٔ انفرادی بدون کارمند باید ۴۲۲ بدهد، %d داد: %s", invalid.Code, invalid.Body.String())
	}

	// ---------- ساخت فایل پرداخت ----------
	build := call(t, h, http.MethodPost, "/api/v1/education_bonus/bonus-files", token, map[string]any{
		"period": "1405/1406", "scope": "group",
	})
	if build.Code != http.StatusCreated {
		t.Fatalf("ساخت فایل پاداش باید ۲۰۱ بدهد، %d داد: %s", build.Code, build.Body.String())
	}
	var file contracts.BonusResult
	decodeInto(t, build, &file)
	if file.FileID == "" || file.Total <= 0 {
		t.Fatalf("فایل پاداش باید شناسه و جمع داشته باشد: %+v", file)
	}

	// ---------- فایل در فهرست رکوردها دیده می‌شود ----------
	files := eduListRecords(t, h, token, "bonusFiles", "")
	found := false
	for _, f := range files {
		if f.ID == file.FileID {
			found = true
			if f.Fields["total"] == "" {
				t.Fatal("فایل پاداش باید جمع کل را نگه دارد")
			}
		}
	}
	if !found {
		t.Fatalf("فایل پاداش %s در فهرست رکوردها نیست", file.FileID)
	}

	// ---------- خروجی CSV برای واحد مالی ----------
	export := call(t, h, http.MethodGet, "/api/v1/education_bonus/bonus-files/"+file.FileID+"/export", token, nil)
	if export.Code != http.StatusOK {
		t.Fatalf("خروجی فایل پاداش باید ۲۰۰ بدهد، %d داد: %s", export.Code, export.Body.String())
	}
	if ct := export.Header().Get("Content-Type"); !strings.Contains(ct, "text/csv") {
		t.Fatalf("نوع محتوای خروجی باید CSV باشد: %q", ct)
	}
	csv := export.Body.String()
	if !strings.HasPrefix(csv, "\ufeff") {
		t.Fatal("خروجی CSV باید BOM داشته باشد تا Excel فارسی را درست نشان دهد")
	}
	for _, want := range []string{"کد پرسنلی", "دانش‌آموز ممتاز", "مبلغ (ریال)", "جمع کل"} {
		if !strings.Contains(csv, want) {
			t.Fatalf("خروجی CSV باید «%s» را داشته باشد", want)
		}
	}
}

// eduListRecords فهرست رکوردهای ماژول پاداش تحصیلی را می‌خواند. query خالی یعنی همه.
func eduListRecords(t *testing.T, h http.Handler, token, kind, query string) []contracts.Record {
	t.Helper()
	path := "/api/v1/education_bonus/records?kind=" + kind + "&size=200"
	if query != "" {
		path += "&query=" + query
	}
	rec := call(t, h, http.MethodGet, path, token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("فهرست رکوردهای %s باید ۲۰۰ بدهد، %d داد: %s", kind, rec.Code, rec.Body.String())
	}
	var page contracts.RecordPage
	decodeInto(t, rec, &page)
	return page.Items
}
