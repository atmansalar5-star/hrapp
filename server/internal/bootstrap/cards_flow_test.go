// آزمون جریان سرتاسری «cards»: سیم‌کشی ماژول‌ها، سیاست‌ها و گردش‌کار.

package bootstrap

import (
	"net/http"
	"strings"
	"testing"

	"hrms/internal/contracts"
)

// TestCardChargeFlowAndFinanceFile جریان کارت‌های رفاهی را از پنل تخصیص تا فایل شارژ
// ارسالی به مالی می‌سنجد: گزینه‌ها، پیش‌نمایش، بازتعریف نرخ، مناسبت تقویمی، فایل
// شارژ و خروجی CSV — همان مسیری که رابط وب طی می‌کند.
func TestCardChargeFlowAndFinanceFile(t *testing.T) {
	h := newTestServer(t)
	manager := login(t, h, "hr.manager", "demo")
	token := manager.AccessToken

	// ---------- گزینه‌ها: کارت‌ها، بخش‌ها با نرخ جاری و مناسبت‌های فعال ----------
	rec := call(t, h, http.MethodGet, "/api/v1/cards/options", token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("گزینه‌های کارت‌ها باید ۲۰۰ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	var options contracts.CardOptions
	decodeInto(t, rec, &options)
	if len(options.Cards) != 2 {
		t.Fatalf("باید دو کارت (پتروکارت و نفت‌کارت) اعلام شود، %d شد", len(options.Cards))
	}
	// پتروکارت: فرهنگی، رستوران، پاداش تحصیلی — نفت‌کارت: چهار قلم رفاهی.
	if len(options.Sections) != 7 {
		t.Fatalf("باید ۷ بخش شارژ اعلام شود (۳ پتروکارت + ۴ قلم رفاهی)، %d شد", len(options.Sections))
	}
	if len(options.Occasions) < 5 {
		t.Fatalf("مناسبت‌های نمونهٔ تقویمی بارگذاری نشده‌اند (%d)", len(options.Occasions))
	}
	for _, opt := range options.Sections {
		if opt.Amount <= 0 {
			t.Fatalf("نرخ جاری «%s» باید مثبت باشد", opt.Key)
		}
	}

	// ---------- پیش‌نمایش شارژ گروهی نفت‌کارت ----------
	preview := func() contracts.CardChargeResult {
		t.Helper()
		body := map[string]any{"card": "naftmart", "scope": "group", "period": "1404/07", "trialRun": true}
		rec := call(t, h, http.MethodPost, "/api/v1/cards/charge/preview", token, body)
		if rec.Code != http.StatusOK {
			t.Fatalf("پیش‌نمایش شارژ باید ۲۰۰ بدهد، %d داد: %s", rec.Code, rec.Body.String())
		}
		var out contracts.CardChargeResult
		decodeInto(t, rec, &out)
		return out
	}
	before := preview()
	if before.Count == 0 || before.Total <= 0 {
		t.Fatalf("پیش‌نمایش باید ردیف و جمع داشته باشد: %+v", before)
	}
	// چهار قلم رفاهی باید هر یک ردیف داشته باشند.
	naftItems := 0
	for _, s := range before.BySection {
		if strings.HasPrefix(s.Section, "naftmart_") && s.Count > 0 {
			naftItems++
		}
	}
	if naftItems != 4 {
		t.Fatalf("هر چهار قلم رفاهی نفت‌کارت باید شارژ شوند، %d قلم شارژ شد", naftItems)
	}

	// ---------- بازتعریف نرخ: نرخ جاری جایگزین می‌شود و مبلغ بالا می‌رود ----------
	rates := listRecords(t, h, token, "cardRates", "", "naftmart_food")
	if len(rates) == 0 {
		t.Fatal("نرخ قلم «کالای اساسی» در رکوردها پیدا نشد")
	}
	supersede := call(t, h, http.MethodPost, "/api/v1/cards/records/"+rates[0].ID+"/action", token,
		map[string]any{"action": "supersede"})
	if supersede.Code != http.StatusOK {
		t.Fatalf("جایگزینی نرخ قدیمی باید ۲۰۰ بدهد، %d داد: %s", supersede.Code, supersede.Body.String())
	}
	newRate := call(t, h, http.MethodPost, "/api/v1/cards/records", token, map[string]any{
		"kind": "cardRates", "title": "کالای اساسی و خواروبار (نرخ جدید)",
		"fields": map[string]string{
			"card": "naftmart", "section": "naftmart_food", "amount": "25000000",
			"unitFa": "ماهانه به ازای هر نفر", "effectiveFrom": "1404/08/01",
		},
	})
	if newRate.Code != http.StatusCreated {
		t.Fatalf("ثبت نرخ جدید باید ۲۰۱ بدهد، %d داد: %s", newRate.Code, newRate.Body.String())
	}
	after := preview()
	if after.Total <= before.Total {
		t.Fatalf("با نرخ جدید جمع شارژ باید بیشتر شود: قبل=%d بعد=%d", before.Total, after.Total)
	}

	// ---------- قاعدهٔ دامنه: پاداش تحصیلی بدون فرزند ممتاز رد می‌شود ----------
	bad := call(t, h, http.MethodPost, "/api/v1/cards/records", token, map[string]any{
		"kind": "cardBeneficiaries", "title": "فرزند آزمایشی", "employeeId": "EMP-005",
		"fields": map[string]string{"relation": "فرزند", "excellent": "0", "educationBonus": "1"},
	})
	if bad.Code != http.StatusUnprocessableEntity {
		t.Fatalf("تخصیص پاداش تحصیلی بدون ممتاز بودن باید ۴۲۲ بدهد، %d داد: %s", bad.Code, bad.Body.String())
	}

	// ---------- مناسبت تقویمی: تعریف مناسبت تازه و شارژ مناسبتی ----------
	// بدون کارت هدف یعنی مناسبت برای هر دو کارت قابل شارژ است.
	occasion := call(t, h, http.MethodPost, "/api/v1/cards/records", token, map[string]any{
		"kind": "occasions", "title": "سالگرد تأسیس شرکت",
		"fields": map[string]string{
			"date": jdate(t, 30), "categoryFa": "سازمانی", "card": "",
			"amount": "3000000", "scopeFa": "همهٔ افراد مشمول کارت",
		},
	})
	if occasion.Code != http.StatusCreated {
		t.Fatalf("تعریف مناسبت تازه باید ۲۰۱ بدهد، %d داد: %s", occasion.Code, occasion.Body.String())
	}
	var occasionRec contracts.Record
	decodeInto(t, occasion, &occasionRec)

	occasionPreview := func() contracts.CardChargeResult {
		t.Helper()
		body := map[string]any{
			"card": "naftmart", "scope": "group", "period": "1404/08",
			"occasionId": occasionRec.ID, "trialRun": true,
		}
		rec := call(t, h, http.MethodPost, "/api/v1/cards/charge/preview", token, body)
		if rec.Code != http.StatusOK {
			t.Fatalf("پیش‌نمایش مناسبتی باید ۲۰۰ بدهد، %d داد: %s", rec.Code, rec.Body.String())
		}
		var out contracts.CardChargeResult
		decodeInto(t, rec, &out)
		return out
	}
	occ := occasionPreview()
	occRows := 0
	for _, r := range occ.Rows {
		if r.Section == "occasion" && r.OccasionFa == "سالگرد تأسیس شرکت" {
			occRows++
		}
	}
	if occRows == 0 {
		t.Fatal("شارژ مناسبتی برای مناسبت تازه ساخته نشد")
	}

	// ---------- شارژ انفرادی بدون انتخاب کارمند رد می‌شود ----------
	invalid := call(t, h, http.MethodPost, "/api/v1/cards/charge/preview", token, map[string]any{
		"card": "petrokart", "scope": "individual", "period": "1404/07", "trialRun": true,
	})
	if invalid.Code != http.StatusUnprocessableEntity {
		t.Fatalf("شارژ انفرادی بدون کارمند باید ۴۲۲ بدهد، %d داد: %s", invalid.Code, invalid.Body.String())
	}

	// ---------- ساخت فایل شارژ پتروکارت با شارژ مناسبتی ----------
	build := call(t, h, http.MethodPost, "/api/v1/cards/charge-files", token, map[string]any{
		"card": "petrokart", "scope": "group", "period": "1404/08",
		"occasionId": occasionRec.ID,
	})
	if build.Code != http.StatusCreated {
		t.Fatalf("ساخت فایل شارژ باید ۲۰۱ بدهد، %d داد: %s", build.Code, build.Body.String())
	}
	var file contracts.CardChargeResult
	decodeInto(t, build, &file)
	if file.FileID == "" || file.Total <= 0 {
		t.Fatalf("فایل شارژ باید شناسه و جمع داشته باشد: %+v", file)
	}

	// ---------- فایل شارژ در فهرست دیده می‌شود ----------
	files := listRecords(t, h, token, "chargeFiles", "all", "")
	found := false
	for _, f := range files {
		if f.ID == file.FileID {
			found = true
			if f.Fields["total"] == "" {
				t.Fatal("فایل شارژ باید جمع کل را نگه دارد")
			}
		}
	}
	if !found {
		t.Fatalf("فایل شارژ %s در فهرست رکوردها نیست", file.FileID)
	}

	// ---------- خروجی CSV برای واحد مالی ----------
	export := call(t, h, http.MethodGet, "/api/v1/cards/charge-files/"+file.FileID+"/export", token, nil)
	if export.Code != http.StatusOK {
		t.Fatalf("خروجی فایل شارژ باید ۲۰۰ بدهد، %d داد: %s", export.Code, export.Body.String())
	}
	if ct := export.Header().Get("Content-Type"); !strings.Contains(ct, "text/csv") {
		t.Fatalf("نوع محتوای خروجی باید CSV باشد: %q", ct)
	}
	csv := export.Body.String()
	if !strings.HasPrefix(csv, "\ufeff") {
		t.Fatal("خروجی CSV باید BOM داشته باشد تا Excel فارسی را درست نشان دهد")
	}
	for _, want := range []string{"کد پرسنلی", "مبلغ (ریال)", "جمع کل", "سالگرد تأسیس شرکت"} {
		if !strings.Contains(csv, want) {
			t.Fatalf("خروجی CSV باید «%s» را داشته باشد", want)
		}
	}
}

// listRecords فهرست رکوردهای یک نوع را می‌خواند. status خالی یعنی همهٔ وضعیت‌ها.
func listRecords(t *testing.T, h http.Handler, token, kind, status, query string) []contracts.Record {
	t.Helper()
	path := "/api/v1/cards/records?kind=" + kind + "&size=200"
	if status != "" && status != "all" {
		path += "&status=" + status
	}
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
