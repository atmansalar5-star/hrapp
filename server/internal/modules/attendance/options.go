// پیاده‌سازی ماژول «attendance» — فهرست‌های مرجع فیلدهای انتخابی.
//
// چرا این فایل: مخزن رکورد (recordkit) فیلد `select` را تنها وقتی به فهرست کشویی تبدیل
// می‌کند که گزینه‌هایش را داشته باشد؛ وگرنه ورودی متنی می‌ماند و کد خام (مثلاً
// `administrative`) در جدول دیده می‌شود. فهرست گزینه‌ها از کاتالوگ‌های دامنه می‌آید تا
// هیچ گزینه‌ای در رابط ساخته نشود و افزودن گزینهٔ تازه فقط در دامنه انجام گیرد.

package attendance

import (
	"net/http"

	"hrms/internal/modules/attendance/domain"
	"hrms/internal/platform/httpx"
	"hrms/internal/platform/modulekit"
)

// registerOptionsEndpoint مسیر فهرست‌های مرجع ماژول را ثبت می‌کند.
func registerOptionsEndpoint(mux *http.ServeMux, deps modulekit.Deps) {
	read := func(fn http.HandlerFunc) http.HandlerFunc { return deps.Require("attendance", "read", fn) }
	mux.HandleFunc("GET /api/v1/attendance/options", read(func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, attendanceOptions())
	}))
}

// attendanceOptions فهرست‌های مرجع ماژول: نوع مرخصی، نوع مأموریت، مشمول اعزام پزشکی،
// نوع اسکان، ارز پرداخت، نوع سند پشتیبان، پرداخت‌کننده و بله/خیر.
func attendanceOptions() map[string]any {
	return map[string]any{
		"leaveTypes":     leaveTypeOptions(),
		"missionTypes":   domain.MissionTypes(),
		"beneficiaries":  domain.MedicalBeneficiaries(),
		"accommodations": domain.Accommodations(),
		"currencies":     domain.MissionCurrencies(),
		"documentTypes":  domain.MissionDocumentTypes(),
		"payers":         payerOptions(),
		"yesNo":          []map[string]string{{"key": "yes", "titleFa": "بله"}, {"key": "no", "titleFa": "خیر"}},
		"noteFa":         "فهرست‌های مرجع از دامنهٔ ماژول می‌آیند؛ افزودن گزینهٔ تازه در دامنه انجام می‌شود.",
	}
}

// leaveTypeOptions انواع مرخصی دامنه را به شکل گزینهٔ فیلد انتخابی می‌سازد.
//
// کلید و عنوان همان دستهٔ دامنه است (پارسی)، چون کارکرد و حقوق هم با همان کلید
// مرخصی را دسته‌بندی می‌کنند؛ پس یک کلید در ذخیره، کارکرد و نمایش دیده می‌شود.
func leaveTypeOptions() []map[string]string {
	classes := domain.LeaveClasses()
	out := make([]map[string]string, 0, len(classes))
	for _, c := range classes {
		out = append(out, map[string]string{"key": c.Key, "titleFa": c.Key, "noteFa": c.NoteFa})
	}
	return out
}

// payerOptions پرداخت‌کنندهٔ هزینهٔ سند پشتیبان مأموریت.
func payerOptions() []map[string]string {
	return []map[string]string{
		{"key": "company", "titleFa": "سازمان"},
		{"key": "employee", "titleFa": "کارمند"},
		{"key": "insurer", "titleFa": "بیمه"},
	}
}
