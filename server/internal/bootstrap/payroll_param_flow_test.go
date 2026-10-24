// آزمون جریان سرتاسری «payroll_param»: قواعد مؤثر بودن پارامترهای حقوق از راه API.
//
// این آزمون همان قراردادی را می‌سنجد که سامانه‌های پیشرو روی پارامتر مالیات/بیمه
// دارند: پیش‌نویس اثر عملیاتی ندارد، تاریخ اجرا رعایت می‌شود، نسخهٔ جاریِ تازه‌تر
// برنده است بدون اینکه نسخهٔ قدیمی از بین برود، و مقدار باید با واحدش بخواند.

package bootstrap

import (
	"net/http"
	"testing"

	"hrms/internal/contracts"
)

// TestPayrollParameterEffectiveness تغذیهٔ پارامترهای حقوق را می‌سنجد: مقدار فعال
// اعمال می‌شود؛ پیش‌نویس و پارامتر آینده اثر ندارند؛ با رسیدن تاریخ اجرا، نسخهٔ
// تازه‌تر جایگزین می‌شود و با غیرفعال‌سازی، نسخهٔ پیشین برمی‌گردد.
func TestPayrollParameterEffectiveness(t *testing.T) {
	h := newTestServer(t)
	hr := login(t, h, "hr.manager", "demo").AccessToken
	// مدیر سیستم برای حذف و اقدام‌های حساس؛ اینجا فقط برای خواندن مکمل به کار نمی‌رود.

	type paramView struct {
		Values map[string]string `json:"values"`
		Items  []struct {
			Key string `json:"key"`
		} `json:"items"`
	}
	readParams := func() paramView {
		t.Helper()
		rec := call(t, h, http.MethodGet, "/api/v1/payroll/parameters", hr, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("خواندن پارامترها باید ۲۰۰ بدهد، %d داد: %s", rec.Code, rec.Body.String())
		}
		var view paramView
		decodeInto(t, rec, &view)
		return view
	}
	uniqueItems := func(view paramView) {
		t.Helper()
		seen := map[string]int{}
		for _, item := range view.Items {
			seen[item.Key]++
			if seen[item.Key] > 1 {
				t.Errorf("فهرست /parameters برای کلید «%s» ردیف تکراری دارد؛ هر پارامتر باید یک ردیف مؤثر باشد", item.Key)
			}
		}
	}

	// ---------- ۱) پارامتر فعالِ بذرشده اعمال شده و فهرست بدون ردیف تکراری است ----------
	base := readParams()
	if base.Values["insurance.employeeRate"] != "0.07" {
		t.Fatalf("نرخ بیمهٔ سهم کارمند باید از پارامتر فعال خوانده شود، %q آمد", base.Values["insurance.employeeRate"])
	}
	if base.Values["allowance.housing"] == "" {
		t.Fatal("حق مسکن از پارامترهای فعال نمی‌آید؛ تغذیهٔ پارامترها قطع است")
	}
	housingBase := base.Values["allowance.housing"]
	uniqueItems(base)

	// ---------- ۲) پیش‌نویس نباید اثر عملیاتی داشته باشد ----------
	create := call(t, h, http.MethodPost, "/api/v1/payroll/records", hr, map[string]any{
		"kind": "payParams", "title": "نسخهٔ پیش‌نویس حق مسکن (آزمون)",
		"fields": map[string]string{"key": "allowance.housing", "value": "99000000",
			"unitFa": "ریال", "groupFa": "مزایا"},
	})
	if create.Code != http.StatusCreated {
		t.Fatalf("ایجاد پارامتر باید ۲۰۱ بدهد، %d داد: %s", create.Code, create.Body.String())
	}
	var param contracts.Record
	decodeInto(t, create, &param)
	if param.Status != "draft" {
		t.Fatalf("پارامتر تازه باید پیش‌نویس باشد، «%s» بود", param.Status)
	}
	if got := readParams().Values["allowance.housing"]; got != housingBase {
		t.Fatalf("پیش‌نویس نباید در محاسبه اعمال شود؛ حق مسکن %q شد (انتظار %q)", got, housingBase)
	}

	// ---------- ۳) تاریخ اجرای آینده حتی پس از فعال‌سازی اثر ندارد ----------
	edit := call(t, h, http.MethodPatch, "/api/v1/payroll/records/"+param.ID, hr, map[string]any{
		"kind": "payParams", "title": "نسخهٔ پیش‌نویس حق مسکن (آزمون)",
		"fields": map[string]string{"key": "allowance.housing", "value": "99000000",
			"unitFa": "ریال", "groupFa": "مزایا", "effectiveFrom": jdate(t, 30)},
	})
	if edit.Code != http.StatusOK {
		t.Fatalf("ویرایش پیش‌نویس باید ۲۰۰ بدهد، %d داد: %s", edit.Code, edit.Body.String())
	}
	if code := call(t, h, http.MethodPost, "/api/v1/payroll/records/"+param.ID+"/action", hr,
		map[string]any{"action": "activate"}).Code; code != http.StatusOK {
		t.Fatalf("جاری‌سازی پارامتر باید ۲۰۰ بدهد، %d داد", code)
	}
	if got := readParams().Values["allowance.housing"]; got != housingBase {
		t.Fatalf("پارامتر با تاریخ اجرای آینده نباید اعمال شود؛ حق مسکن %q شد", got)
	}

	// ---------- ۴) با رسیدن تاریخ اجرا، نسخهٔ تازه‌تر برنده می‌شود ----------
	// ویرایش در وضعیت «جاری» هم باید مجاز باشد (پارامتر دادهٔ پیکربندی است).
	apply := call(t, h, http.MethodPatch, "/api/v1/payroll/records/"+param.ID, hr, map[string]any{
		"kind": "payParams", "title": "نسخهٔ پیش‌نویس حق مسکن (آزمون)",
		"fields": map[string]string{"key": "allowance.housing", "value": "99000000",
			"unitFa": "ریال", "groupFa": "مزایا", "effectiveFrom": jdate(t, 0)},
	})
	if apply.Code != http.StatusOK {
		t.Fatalf("ویرایش پارامتر جاری باید ۲۰۰ بدهد، %d داد: %s", apply.Code, apply.Body.String())
	}
	afterApply := readParams()
	if afterApply.Values["allowance.housing"] != "99000000" {
		t.Fatalf("نسخهٔ جاریِ رسیده به تاریخ اجرا باید برنده شود، %q آمد", afterApply.Values["allowance.housing"])
	}
	uniqueItems(afterApply)

	// ---------- ۵) غیرفعال‌سازی، نسخهٔ پیشین را بدون حذف برمی‌گرداند ----------
	if code := call(t, h, http.MethodPost, "/api/v1/payroll/records/"+param.ID+"/action", hr,
		map[string]any{"action": "deactivate"}).Code; code != http.StatusOK {
		t.Fatalf("غیرفعال‌سازی باید ۲۰۰ بدهد، %d داد", code)
	}
	if got := readParams().Values["allowance.housing"]; got != housingBase {
		t.Fatalf("با غیرفعال شدن نسخه باید مقدار پیشین برگردد؛ %q آمد (انتظار %q)", got, housingBase)
	}
}

// TestPayrollParameterValidation فرم ورود پارامتر را در برابر مقدار ناسازگار با واحد
// می‌سنجد: سرور باید ۴۲۲ و پیام فارسی کنار همان فیلد بدهد تا رابط خطا را کنار مقدار
// نشان دهد (هیچ قاعده‌ای در UI نیست).
func TestPayrollParameterValidation(t *testing.T) {
	h := newTestServer(t)
	hr := login(t, h, "hr.manager", "demo").AccessToken

	cases := []struct {
		name   string
		fields map[string]string
	}{
		{name: "مقدار غیرعددی با واحد ریال",
			fields: map[string]string{"key": "test.rials.value", "value": "زیاد", "unitFa": "ریال"}},
		{name: "مقدار غیرنسبتی با واحد نسبت",
			fields: map[string]string{"key": "test.ratio.value", "value": "یک‌چهارم", "unitFa": "نسبت"}},
		{name: "پله بدون «سقف:نرخ»",
			fields: map[string]string{"key": "test.brackets.value", "value": "50000000", "unitFa": "پله"}},
		{name: "پله با جزء غیرعددی",
			// ارقام فارسی در سمت عدد پذیرفته می‌شوند؛ «زیاد» نه.
			fields: map[string]string{"key": "test.brackets2.value", "value": "50000000:زیاد", "unitFa": "پله"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := call(t, h, http.MethodPost, "/api/v1/payroll/records", hr, map[string]any{
				"kind": "payParams", "title": "پارامتر نامعتبر (آزمون)", "fields": tc.fields,
			})
			if rec.Code != http.StatusUnprocessableEntity {
				t.Fatalf("مقدار ناسازگار با واحد باید ۴۲۲ بدهد، %d داد: %s", rec.Code, rec.Body.String())
			}
		})
	}

	// مقدار معتبر با ارقام فارسی باید پذیرفته شود (یکسان‌سازی ورودی کاربر).
	ok := call(t, h, http.MethodPost, "/api/v1/payroll/records", hr, map[string]any{
		"kind": "payParams", "title": "پارامتر معتبر (آزمون)",
		"fields": map[string]string{"key": "test.valid.farsi", "value": "۱۲۰۰۰۰۰", "unitFa": "ریال",
			"effectiveFrom": jdate(t, 1)},
	})
	if ok.Code != http.StatusCreated {
		t.Fatalf("پارامتر معتبر با ارقام فارسی باید ۲۰۱ بدهد، %d داد: %s", ok.Code, ok.Body.String())
	}
	// پارامتر پیش‌نویس با تاریخ اجرای آینده نباید در /parameters دیده شود.
	rec := call(t, h, http.MethodGet, "/api/v1/payroll/parameters", hr, nil)
	var view struct {
		Values map[string]string `json:"values"`
	}
	decodeInto(t, rec, &view)
	if _, exists := view.Values["test.valid.farsi"]; exists {
		t.Fatal("پارامتر پیش‌نویس نباید در /parameters اعمال شود")
	}
}
