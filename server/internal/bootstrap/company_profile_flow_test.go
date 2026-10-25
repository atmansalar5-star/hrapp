// آزمون جریان سرتاسری «شرکت‌ها و استقرارها»: فرم ورود شرکت تازه و ویرایش شرکت‌های
// موجود از راه API واقعی — همان مسیری که صفحهٔ «شرکت‌ها و استقرارها» در مرورگر می‌خواند.

package bootstrap

import (
	"net/http"
	"strings"
	"testing"

	"hrms/internal/contracts"
)

func TestCompanyProfileFlow(t *testing.T) {
	h := newTestServer(t)
	hr := login(t, h, "hr.manager", "demo").AccessToken
	emp := login(t, h, "employee", "demo").AccessToken

	// ---------- شناسهٔ ماژول: نوع رکورد شرکت با پیشوند ORG ----------
	rec := call(t, h, http.MethodGet, "/api/v1/settings/schema", hr, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("شناسهٔ ماژول تنظیمات باید ۲۰۰ بدهد، %d داد", rec.Code)
	}
	var schema contracts.RecordSchema
	decodeInto(t, rec, &schema)
	found := false
	for _, k := range schema.Kinds {
		if k.Key == "companyProfiles" {
			found = true
			if k.IDPrefix != "ORG" {
				t.Errorf("پیشوند شناسهٔ شرکت باید ORG باشد، «%s» است", k.IDPrefix)
			}
		}
	}
	if !found {
		t.Fatalf("نوع رکورد «companyProfiles» در شناسهٔ ماژول تنظیمات نیست")
	}

	// ---------- شرکت‌های موجود از فهرست استقرارها بذر شده‌اند ----------
	list := call(t, h, http.MethodGet, "/api/v1/settings/records?kind=companyProfiles&size=100", hr, nil)
	if list.Code != http.StatusOK {
		t.Fatalf("فهرست شرکت‌ها باید ۲۰۰ بدهد، %d داد: %s", list.Code, list.Body.String())
	}
	var page contracts.RecordPage
	decodeInto(t, list, &page)
	if page.Total < 2 {
		t.Fatalf("شرکت‌های استقرار باید بذر شده باشند، %d رکورد آمد", page.Total)
	}
	activeSeen := 0
	for _, item := range page.Items {
		if !strings.HasPrefix(item.ID, "ORG-") {
			t.Errorf("شناسهٔ شرکت باید با ORG- آغاز شود، «%s» است", item.ID)
		}
		if item.Fields["isActiveDeployment"] == "بله" {
			activeSeen++
			if item.Fields["code"] == "" || item.Fields["holdingFa"] == "" {
				t.Errorf("شرکت فعال باید کد و هلدینگ داشته باشد: %+v", item.Fields)
			}
		}
	}
	if activeSeen != 1 {
		t.Errorf("فقط یک شرکت باید «استقرار فعال این سرور» باشد، %d بود", activeSeen)
	}

	// ---------- قاعده‌های دامنه: کد، رنگ برند و پارامترها ----------
	badCases := []map[string]any{
		{"kind": "companyProfiles", "title": "کد نامعتبر",
			"fields": map[string]string{"code": "12 شرکت", "titleFa": "شرکت آزمون"}},
		{"kind": "companyProfiles", "title": "رنگ نامعتبر",
			"fields": map[string]string{"code": "sample_co", "titleFa": "شرکت آزمون", "brandColor": "آبی"}},
		{"kind": "companyProfiles", "title": "پارامتر نامعتبر",
			"fields": map[string]string{"code": "sample_co", "titleFa": "شرکت آزمون", "paramOverrides": "monthlyHours بدون علامت"}},
		{"kind": "companyProfiles", "title": "اداره نامعتبر",
			"fields": map[string]string{"code": "sample_co", "titleFa": "شرکت آزمون", "enabledDepartments": "staff_affairs,ادارهٔ نامعتبر"}},
	}
	for i, body := range badCases {
		got := call(t, h, http.MethodPost, "/api/v1/settings/records", hr, body)
		if got.Code != http.StatusUnprocessableEntity {
			t.Errorf("حالت نامعتبر %d باید ۴۲۲ بدهد، %d داد: %s", i+1, got.Code, got.Body.String())
		}
	}

	// ---------- ورود شرکت تازه ----------
	rec = call(t, h, http.MethodPost, "/api/v1/settings/records", hr, map[string]any{
		"kind": "companyProfiles", "title": "شرکت پتروشیمی نمونهٔ آزمون",
		"fields": map[string]string{
			"code":               "sample_petro",
			"titleFa":            "شرکت پتروشیمی نمونهٔ آزمون",
			"titleEn":            "Sample Petro",
			"holdingFa":          "هلدینگ نمونه",
			"headcount":          "850",
			"brandColor":         "#1E88E5",
			"calendar":           "شمسی",
			"currencyFa":         "ریال",
			"fiscalYearStart":    "01/01",
			"nationalId":         "10101010101",
			"economicCode":       "411111111111",
			"taxFileNumber":      "123456789",
			"workshopCode":       "99887766",
			"enabledDepartments": "staff_affairs,planning,platform",
			"disabledModules":    "sports,reports",
			"paramOverrides":     "payroll.monthlyHours=176\nnightShift.factor=1.35",
			"notesFa":            "شرکت تازه‌ای که از فرم وارد شده است.",
		},
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("ثبت شرکت تازه باید ۲۰۱ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	var company contracts.Record
	decodeInto(t, rec, &company)
	if company.Status != "draft" {
		t.Errorf("شرکت تازه باید پیش‌نویس باشد تا اطلاعاتش کامل شود، «%s» شد", company.Status)
	}
	if company.Fields["code"] != "sample_petro" {
		t.Errorf("کد شرکت باید ذخیره شود، «%s» شد", company.Fields["code"])
	}

	// ---------- ویرایش همان شرکت (فرم ویرایش) ----------
	rec = call(t, h, http.MethodPatch, "/api/v1/settings/records/"+company.ID, hr, map[string]any{
		"kind": "companyProfiles", "title": "شرکت پتروشیمی نمونهٔ آزمون (ویرایش‌شده)",
		"fields": map[string]string{
			"code":               "sample_petro",
			"titleFa":            "شرکت پتروشیمی نمونهٔ آزمون (ویرایش‌شده)",
			"headcount":          "900",
			"paramOverrides":     "payroll.monthlyHours=180",
			"enabledDepartments": "staff_affairs,planning,platform,relations",
		},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("ویرایش شرکت باید ۲۰۰ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	decodeInto(t, rec, &company)
	if company.Fields["headcount"] != "900" {
		t.Errorf("تعداد کارکنان ویرایش‌شده باید ۹۰۰ باشد، «%s» شد", company.Fields["headcount"])
	}
	if !strings.Contains(company.Fields["paramOverrides"], "180") {
		t.Errorf("پارامتر ویرایش‌شده باید ذخیره شود: %q", company.Fields["paramOverrides"])
	}

	// ---------- فعال‌سازی شرکت و دیده‌شدنش در فهرست ----------
	act := call(t, h, http.MethodPost, "/api/v1/settings/records/"+company.ID+"/action", hr,
		map[string]any{"action": "activate"})
	if act.Code != http.StatusOK {
		t.Fatalf("فعال‌سازی شرکت باید ۲۰۰ بدهد، %d داد: %s", act.Code, act.Body.String())
	}
	got := call(t, h, http.MethodGet, "/api/v1/settings/records/"+company.ID, hr, nil)
	if got.Code != http.StatusOK {
		t.Fatalf("خواندن شرکت باید ۲۰۰ بدهد، %d داد", got.Code)
	}
	decodeInto(t, got, &company)
	if company.Status != "active" {
		t.Errorf("وضعیت پایانی شرکت باید active باشد، «%s» شد", company.Status)
	}

	// ---------- دسترسی: کارمند نباید شرکت اضافه کند ----------
	denied := call(t, h, http.MethodPost, "/api/v1/settings/records", emp, map[string]any{
		"kind": "companyProfiles", "title": "شرکت غیرمجاز",
		"fields": map[string]string{"code": "denied_co", "titleFa": "شرکت غیرمجاز"},
	})
	if denied.Code != http.StatusForbidden {
		t.Errorf("افزودن شرکت توسط کارمند باید ۴۰۳ بدهد، %d داد", denied.Code)
	}
}
