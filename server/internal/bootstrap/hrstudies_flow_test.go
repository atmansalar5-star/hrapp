// آزمون جریان سرتاسری «hrstudies»: سیم‌کشی ماژول‌ها، سیاست‌ها و گردش‌کار.

package bootstrap

import (
	"net/http"
	"testing"

	"hrms/internal/contracts"
)

// TestHRStudiesIndicesAndSurveysFlow دو تبِ واحد مطالعات را سرتاسری می‌سنجد: فهرست
// شاخص‌ها با سنجش «در محدودهٔ هدف / خارج از هدف»، تحلیل مشارکت نظرسنجی‌ها، گردش‌کار
// نوع رکورد تازه‌اعلام‌شدهٔ شاخص (ایجاد، ویرایش، حذف) و رد شدن جهت مطلوب نامعتبر.
//
// چرا این آزمون لازم است: پیش از این، مسیرهای /indices و /surveys در قرارداد اعلام
// شده بودند ولی هیچ پیاده‌سازی نداشتند؛ صفحه‌ها به دادهٔ نمایشی ثابت برمی‌گشتند. این
// آزمون تضمین می‌کند همان دو مسیر روی دادهٔ واقعی کار کنند.
func TestHRStudiesIndicesAndSurveysFlow(t *testing.T) {
	h := newTestServer(t)
	manager := login(t, h, "hr.manager", "demo")
	// حذف در سیاست نقش‌ها فقط برای مدیر سیستم مجاز است؛ همان قاعدهٔ سرور.
	admin := login(t, h, "admin", "demo")
	token := manager.AccessToken

	// ---------- شناسهٔ ماژول: نوع رکورد شاخص باید اعلام شده باشد ----------
	rec := call(t, h, http.MethodGet, "/api/v1/hrstudies/schema", token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("شناسهٔ ماژول مطالعات باید ۲۰۰ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	var schema contracts.RecordSchema
	decodeInto(t, rec, &schema)
	hasIndicatorKind := false
	for _, k := range schema.Kinds {
		if k.Key == "hrIndices" {
			hasIndicatorKind = true
		}
	}
	if !hasIndicatorKind {
		t.Fatalf("نوع رکورد «hrIndices» در شناسهٔ ماژول نیست: %+v", schema.Kinds)
	}

	// ---------- شاخص‌ها: سنجش با هدف ----------
	rec = call(t, h, http.MethodGet, "/api/v1/hrstudies/indices", token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("فهرست شاخص‌ها باید ۲۰۰ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	var indices contracts.HRIndicesView
	decodeInto(t, rec, &indices)
	if len(indices.Items) < 5 {
		t.Fatalf("شاخص‌های نمونه بارگذاری نشده‌اند (تعداد=%d)", len(indices.Items))
	}
	if indices.Summary.Total != len(indices.Items) {
		t.Fatalf("جمع شاخص‌ها با فهرست نمی‌خواند: %d در برابر %d", indices.Summary.Total, len(indices.Items))
	}
	sumVerdicts := indices.Summary.InTarget + indices.Summary.NearTarget + indices.Summary.OutOfTarget
	if sumVerdicts != indices.Summary.Total {
		t.Fatalf("جمع وضعیت‌های سنجش با کل شاخص‌ها نمی‌خواند: %d در برابر %d", sumVerdicts, indices.Summary.Total)
	}
	for _, it := range indices.Items {
		switch it.Assessment {
		case "in_target", "near_target", "out_of_target":
		default:
			t.Fatalf("وضعیت سنجش شاخص «%s» نامعتبر است: «%s»", it.TitleFa, it.Assessment)
		}
		if it.AssessmentFa == "" || it.DirectionFa == "" {
			t.Fatalf("شاخص «%s» برچسب فارسی وضعیت/جهت ندارد: %+v", it.TitleFa, it)
		}
		if it.Target == 0 && it.AchievedPct != 0 && it.AchievedPct != 100 {
			t.Fatalf("درصد تحقق شاخص بدون هدف «%s» نادرست است: %v", it.TitleFa, it.AchievedPct)
		}
	}
	// دادهٔ نمونه باید هر سه حالت سنجش را نشان دهد، وگرنه صفحهٔ تحلیل یکنواخت می‌شود.
	if indices.Summary.InTarget == 0 || indices.Summary.OutOfTarget == 0 {
		t.Fatalf("دادهٔ نمونهٔ شاخص‌ها هر سه حالت سنجش را پوشش نمی‌دهد: %+v", indices.Summary)
	}

	// ---------- نظرسنجی‌ها: تحلیل مشارکت ----------
	rec = call(t, h, http.MethodGet, "/api/v1/hrstudies/surveys", token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("فهرست نظرسنجی‌ها باید ۲۰۰ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	var surveys contracts.HRSurveysView
	decodeInto(t, rec, &surveys)
	if len(surveys.Items) < 4 {
		t.Fatalf("نظرسنجی‌های نمونه بارگذاری نشده‌اند (تعداد=%d)", len(surveys.Items))
	}
	if surveys.Summary.Total != len(surveys.Items) {
		t.Fatalf("جمع نظرسنجی‌ها با فهرست نمی‌خواند: %d در برابر %d", surveys.Summary.Total, len(surveys.Items))
	}
	if surveys.Summary.AvgParticipationPct <= 0 {
		t.Fatalf("میانگین مشارکت باید محاسبه شود: %+v", surveys.Summary)
	}
	bands := map[string]int{}
	for _, s := range surveys.Items {
		if s.ParticipationPct < 0 || s.ParticipationPct > 100 {
			t.Fatalf("نرخ مشارکت نظرسنجی «%s» خارج از بازه است: %v", s.TitleFa, s.ParticipationPct)
		}
		if s.BandFa == "" {
			t.Fatalf("نظرسنجی «%s» برچسب دستهٔ مشارکت ندارد", s.TitleFa)
		}
		bands[s.Band]++
	}
	for _, want := range []string{"good", "acceptable", "weak"} {
		if bands[want] == 0 {
			t.Fatalf("دادهٔ نمونه دستهٔ مشارکت «%s» را پوشش نمی‌دهد: %+v", want, bands)
		}
	}

	// ---------- گردش‌کار شاخص: ایجاد، ویرایش، حذف ----------
	// جهت مطلوب نامعتبر باید رد شود (قاعدهٔ دامنه، نه رابط).
	rec = call(t, h, http.MethodPost, "/api/v1/hrstudies/records", token, map[string]any{
		"kind": "hrIndices", "title": "شاخص آزمون",
		"fields": map[string]string{"formula": "الف ÷ ب", "target": "10", "current": "9", "direction": "sideways"},
	})
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("جهت مطلوب نامعتبر باید ۴۲۲ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}

	rec = call(t, h, http.MethodPost, "/api/v1/hrstudies/records", token, map[string]any{
		"kind": "hrIndices", "title": "نرخ پوشش آموزش‌های ایمنی",
		"fields": map[string]string{"formula": "نفرساعت آموزش ایمنی ÷ سرشماری", "target": "12", "current": "9",
			"direction": "higher", "unit": "ساعت", "period": "1404/07", "owner": "هماهنگی آموزش"},
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("ایجاد شاخص باید ۲۰۱ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	var indicator contracts.Record
	decodeInto(t, rec, &indicator)
	if indicator.Status != "active" {
		t.Fatalf("شاخص تازه باید در وضعیت فعال باشد، «%s» بود", indicator.Status)
	}

	// شاخص تازه باید در تحلیل هم دیده شود و «خارج از هدف» سنجیده شود (۹ در برابر ۱۲).
	rec = call(t, h, http.MethodGet, "/api/v1/hrstudies/indices?size=100", token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("فهرست شاخص‌ها پس از ایجاد باید ۲۰۰ بدهد، %d داد", rec.Code)
	}
	var after contracts.HRIndicesView
	decodeInto(t, rec, &after)
	if len(after.Items) != len(indices.Items)+1 {
		t.Fatalf("شاخص تازه در تحلیل نیامد: %d در برابر %d", len(after.Items), len(indices.Items))
	}
	found := false
	for _, it := range after.Items {
		if it.ID != indicator.ID {
			continue
		}
		found = true
		if it.Assessment != "out_of_target" || it.GapPct != 25 {
			t.Fatalf("سنجش شاخص تازه نادرست است: %+v", it)
		}
	}
	if !found {
		t.Fatalf("شاخص «%s» در فهرست تحلیل نیست", indicator.ID)
	}

	// ویرایش مقدار جاری: باید به «در محدودهٔ هدف» برگردد.
	rec = call(t, h, http.MethodPatch, "/api/v1/hrstudies/records/"+indicator.ID, token, map[string]any{
		"kind": "hrIndices", "title": "نرخ پوشش آموزش‌های ایمنی",
		"fields": map[string]string{"formula": "نفرساعت آموزش ایمنی ÷ سرشماری", "target": "12", "current": "11.8",
			"direction": "higher", "unit": "ساعت", "period": "1404/07", "owner": "هماهنگی آموزش"},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("ویرایش شاخص باید ۲۰۰ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	rec = call(t, h, http.MethodGet, "/api/v1/hrstudies/indices?size=100", token, nil)
	decodeInto(t, rec, &after)
	for _, it := range after.Items {
		if it.ID == indicator.ID && it.Assessment != "in_target" {
			t.Fatalf("شاخص ویرایش‌شده باید در هدف باشد: %+v", it)
		}
	}

	// حذف: رکورد آزمون نباید در فهرست بماند.
	rec = call(t, h, http.MethodDelete, "/api/v1/hrstudies/records/"+indicator.ID, admin.AccessToken, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("حذف شاخص باید ۲۰۴ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	rec = call(t, h, http.MethodGet, "/api/v1/hrstudies/indices?size=100", token, nil)
	decodeInto(t, rec, &after)
	if len(after.Items) != len(indices.Items) {
		t.Fatalf("پس از حذف باید %d شاخص بماند، %d شد", len(indices.Items), len(after.Items))
	}
}
