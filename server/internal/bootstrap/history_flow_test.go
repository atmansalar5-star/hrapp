// آزمون جریان سرتاسری «history»: سیم‌کشی ماژول‌ها، سیاست‌ها و گردش‌کار.

package bootstrap

import (
	"net/http"
	"strings"
	"testing"

	"hrms/internal/contracts"
)

// TestEmployeeHistoryAggregatesModules می‌سنجد که سوابق یک کارمند از چند ماژول می‌آید،
// اسناد پیشین (مثل حکم منسوخ) هم دیده می‌شوند، ترتیب نزولی تاریخ است و برای هر
// (ماژول، نوع) تنها تازه‌ترین سند نشانهٔ «آخرین سند» می‌گیرد.
func TestEmployeeHistoryAggregatesModules(t *testing.T) {
	h := newTestServer(t)
	hr := login(t, h, "hr.manager", "demo").AccessToken

	rec := call(t, h, http.MethodGet, "/api/v1/history?employeeId=EMP-005", hr, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("سوابق کارمند باید ۲۰۰ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	var tl contracts.HistoryTimeline
	decodeInto(t, rec, &tl)

	if tl.EmployeeID != "EMP-005" {
		t.Fatalf("شناسهٔ کارمند در پاسخ نادرست است: %s", tl.EmployeeID)
	}
	if tl.EmployeeName == "" {
		t.Fatal("نام کارمند در سرصفحهٔ سوابق نیامد")
	}
	if len(tl.Warnings) != 0 {
		t.Fatalf("هیچ ماژولی نباید در خواندن سابقه خطا بدهد: %v", tl.Warnings)
	}

	byModule := map[string][]contracts.HistoryEntry{}
	for _, e := range tl.Items {
		byModule[e.Module] = append(byModule[e.Module], e)
	}
	// سه ماژولی که سابقهٔ خود را با منبع اختصاصی اعلام می‌کنند (احکام، قراردادها، تکفل).
	for _, mod := range []string{"salary_admin", "contracts", "staff_services"} {
		if len(byModule[mod]) == 0 {
			t.Errorf("سابقهٔ ماژول %s در پرونده نیامده است (ماژول‌های حاضر: %v)", mod, moduleKeys(byModule))
		}
	}
	if tl.Total != len(tl.Items) {
		t.Fatalf("شمارش کل با تعداد سطرها نمی‌خواند: %d ≠ %d", tl.Total, len(tl.Items))
	}
	if tl.ByModule["salary_admin"] != len(byModule["salary_admin"]) {
		t.Fatalf("شمارش به تفکیک ماژول نادرست است: %v", tl.ByModule)
	}

	// «احکام قبلی»: حکم منسوخ هم باید در سابقه باشد، نه اینکه پاک شده باشد.
	superseded := false
	for _, e := range byModule["salary_admin"] {
		if e.RecordID == "DEC-1403-0198" && e.Status == "superseded" {
			superseded = true
		}
	}
	if !superseded {
		t.Fatal("حکم منسوخ پیشین در سوابق کارمند دیده نشد")
	}

	// ترتیب نزولی تاریخ: سطرهای بی‌تاریخ باید در پایان بیایند.
	seenEmptyOrNonDescending := ""
	for i, e := range tl.Items {
		if e.Date == "" {
			seenEmptyOrNonDescending = "سطر بی‌تاریخ در پرونده‌ای که همهٔ اسنادش تاریخ دارند"
			continue
		}
		if i == 0 {
			continue
		}
		if tl.Items[i-1].Date != "" && e.Date > tl.Items[i-1].Date {
			t.Fatalf("ترتیب نزولی تاریخ نقض شد: %s پیش از %s", tl.Items[i-1].Date, e.Date)
		}
	}
	if seenEmptyOrNonDescending != "" {
		t.Fatalf("سابقه بدون تاریخ مؤثر ماند: %s", seenEmptyOrNonDescending)
	}

	// «آخرین سند» برای هر (ماژول، نوع) دقیقاً یک سند.
	latest := map[string]int{}
	for _, e := range tl.Items {
		if e.Latest {
			latest[e.Module+"|"+e.Kind]++
		}
	}
	if len(latest) == 0 {
		t.Fatal("هیچ سندی نشانهٔ «آخرین سند» نگرفت")
	}
	for key, n := range latest {
		if n != 1 {
			t.Fatalf("برای %s تعداد %d سند نشانهٔ آخرین گرفته‌اند", key, n)
		}
	}

	// فیلتر ماژول روی سرور اعمال می‌شود.
	only := call(t, h, http.MethodGet, "/api/v1/history?employeeId=EMP-005&module=salary_admin", hr, nil)
	if only.Code != http.StatusOK {
		t.Fatalf("سوابق فیلترشده باید ۲۰۰ بدهد، %d داد", only.Code)
	}
	var filtered contracts.HistoryTimeline
	decodeInto(t, only, &filtered)
	if filtered.Total == 0 {
		t.Fatal("فیلتر ماژول همهٔ سطرهای احکام را حذف کرد")
	}
	for _, e := range filtered.Items {
		if e.Module != "salary_admin" {
			t.Fatalf("سطر ماژول دیگری در فیلتر احکام آمد: %s", e.Module)
		}
	}
}

// TestHistoryRespectsRoleAndScope می‌سنجد که سوابق پرونده هم‌قاعده با دامنهٔ دادهٔ نقش
// محدود می‌شود: کارمند فقط سابقهٔ خودش، مدیر واحد کارکنان واحد خودش، و مدیر سیستم همه.
func TestHistoryRespectsRoleAndScope(t *testing.T) {
	h := newTestServer(t)
	admin := login(t, h, "admin", "demo").AccessToken
	hr := login(t, h, "hr.manager", "demo").AccessToken
	lineMgr := login(t, h, "line.mgr", "demo").AccessToken
	emp := login(t, h, "employee", "demo").AccessToken

	if rec := call(t, h, http.MethodGet, "/api/v1/history", hr, nil); rec.Code != http.StatusBadRequest {
		t.Fatalf("نبود شناسهٔ کارمند باید ۴۰۰ بدهد، %d داد", rec.Code)
	}

	// کارمند ساده: پروندهٔ خودش مجاز، پروندهٔ دیگری ممنوع.
	if rec := call(t, h, http.MethodGet, "/api/v1/history?employeeId=EMP-010", emp, nil); rec.Code != http.StatusOK {
		t.Fatalf("کارمند باید سابقهٔ پروندهٔ خودش را ببیند، %d داد: %s", rec.Code, rec.Body.String())
	}
	if rec := call(t, h, http.MethodGet, "/api/v1/history?employeeId=EMP-005", emp, nil); rec.Code != http.StatusForbidden {
		t.Fatalf("سابقهٔ کارمند دیگر برای نقش کارمند باید ۴۰۳ بدهد، %d داد", rec.Code)
	}

	// مدیر واحد: کارکنان واحد خودش مجاز، واحد دیگر ممنوع.
	if rec := call(t, h, http.MethodGet, "/api/v1/history?employeeId=EMP-002", lineMgr, nil); rec.Code != http.StatusOK {
		t.Fatalf("مدیر واحد باید سابقهٔ کارمند واحد خودش را ببیند، %d داد", rec.Code)
	}
	if rec := call(t, h, http.MethodGet, "/api/v1/history?employeeId=EMP-005", lineMgr, nil); rec.Code != http.StatusForbidden {
		t.Fatalf("سابقهٔ کارمند واحد دیگر برای مدیر واحد باید ۴۰۳ بدهد، %d داد", rec.Code)
	}

	// مدیر سیستم: کارمند ناشناخته خطا نیست؛ سابقهٔ خالی با یادداشت برمی‌گردد.
	rec := call(t, h, http.MethodGet, "/api/v1/history?employeeId=EMP-999", admin, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("سابقهٔ کارمند ناشناخته باید ۲۰۰ بدهد، %d داد", rec.Code)
	}
	var empty contracts.HistoryTimeline
	decodeInto(t, rec, &empty)
	if empty.Total != 0 || len(empty.Items) != 0 {
		t.Fatalf("کارمند ناشناخته نباید سابقه بدهد: %d سطر", empty.Total)
	}
	if !strings.Contains(empty.NoteFa, "احکام قبلی") {
		t.Fatal("یادداشت توضیحی سوابق در پاسخ نیامد")
	}
}

// TestHistoryIncludesChangeTrailFromAudit می‌سنجد که سابقهٔ تغییرات اسنادی که رخداد
// تغییر را در خودِ سند نگه نمی‌دارند (مانند تکفل) از زنجیرهٔ حسابرسی می‌آید: پس از یک
// اقدام واقعی روی سند، همان اقدام با کاربر و برچسب فارسی در سطر سابقهٔ پرونده دیده می‌شود.
func TestHistoryIncludesChangeTrailFromAudit(t *testing.T) {
	h := newTestServer(t)
	spec := login(t, h, "hr.spec", "demo")
	manager := login(t, h, "hr.manager", "demo")

	// یک تکفل تازه برای کارمندی که در آزمون سابقهٔ دیگری دیده نشده است.
	create := map[string]any{
		"employeeId":    "EMP-012",
		"fullName":      "نسترن کیانی",
		"relation":      "child",
		"nationalId":    "0087654326",
		"coverageStart": jdate(t, 3),
		"insurance":     true,
		"documentsFa":   "کپی شناسنامه و کارت ملی",
	}
	rec := call(t, h, http.MethodPost, "/api/v1/personnel/dependents", spec.AccessToken, create)
	if rec.Code != http.StatusCreated {
		t.Fatalf("ثبت تکفل باید ۲۰۱ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	var dependent contracts.Dependent
	decodeInto(t, rec, &dependent)

	rec = call(t, h, http.MethodPost, "/api/v1/personnel/dependents/"+dependent.ID+"/decision", manager.AccessToken, map[string]any{"approve": true})
	if rec.Code != http.StatusOK {
		t.Fatalf("تأیید تکفل باید ۲۰۰ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}

	rec = call(t, h, http.MethodGet, "/api/v1/history?employeeId=EMP-012", manager.AccessToken, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("سوابق پرونده باید ۲۰۰ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	var tl contracts.HistoryTimeline
	decodeInto(t, rec, &tl)

	var row *contracts.HistoryEntry
	for i := range tl.Items {
		if tl.Items[i].Module == "staff_services" && tl.Items[i].RecordID == dependent.ID {
			row = &tl.Items[i]
		}
	}
	if row == nil {
		t.Fatalf("سطر تکفل تازه در سوابق پرونده نیامد (سطرها: %v)", moduleKeys2(tl.Items))
	}
	if len(row.Changes) == 0 {
		t.Fatalf("سابقهٔ تغییرات تکفل از حسابرسی پر نشد: %+v", row)
	}
	actions := map[string]string{}
	actors := map[string]string{}
	for _, c := range row.Changes {
		actions[c.Action] = c.ActionFa
		actors[c.Action] = c.Actor
		if c.At == "" {
			t.Errorf("زمان رخداد باید پر شود: %+v", c)
		}
		if c.NoteFa == "" {
			t.Errorf("شرح فارسی رخداد نباید خالی باشد: %+v", c)
		}
	}
	// کاربر هر رخداد از خود لاگ می‌آید: ثبت‌کنندهٔ تکفل hr.spec و تصمیم‌گیرنده hr.manager.
	if actors["dependent-create"] != "hr.spec" {
		t.Errorf("ثبت‌کنندهٔ تکفل باید hr.spec باشد: %v", actors)
	}
	if actors["dependent-decide"] != "hr.manager" {
		t.Errorf("تصمیم‌گیرنده باید hr.manager باشد: %v", actors)
	}
	if actions["dependent-create"] != "ثبت تکفل" {
		t.Errorf("برچسب فارسی «ثبت تکفل» از دامنه نیامد: %v", actions)
	}
	if actions["dependent-decide"] != "تصمیم دربارهٔ تکفل" {
		t.Errorf("برچسب فارسی «تصمیم دربارهٔ تکفل» از دامنه نیامد: %v", actions)
	}
}

// moduleKeys2 ماژول/شناسهٔ سطرهای سابقه (برای پیام خطای خوانا).
func moduleKeys2(items []contracts.HistoryEntry) []string {
	out := make([]string, 0, len(items))
	for _, e := range items {
		out = append(out, e.Module+"/"+e.RecordID)
	}
	return out
}

// moduleKeys کلید ماژول‌های حاضر در سابقه (برای پیام خطای خوانا).
func moduleKeys(byModule map[string][]contracts.HistoryEntry) []string {
	out := make([]string, 0, len(byModule))
	for k := range byModule {
		out = append(out, k)
	}
	return out
}
