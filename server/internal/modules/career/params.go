// پیاده‌سازی ماژول «انتصابات، ترفیعات و کارراههٔ شغلی» — پارامترهای قابل ویرایش.
//
// چرا این فایل: آستانه‌های تصویب و شرایط احراز سالانه عوض می‌شوند (سابقهٔ لازم، امتیاز
// ارزیابی، ضریب جبران مدرک، سقف تخفیف مازاد). پیش از این، تنها راه تغییر، ویرایش
// پارامترهای استقرار شرکت (فایل تولیدشده) بود که انتشار نسخهٔ تازه می‌خواست. اکنون مدیر
// منابع انسانی از **خود فرم انتصابات و ترفیعات** مقدار را زیاد/کم می‌کند و سابقهٔ تغییر در
// رکورد پارامتر می‌ماند.
//
// ترتیب تقدم مقدار (همان الگوی پارامترهای حقوق):
//
//	۱. پیش‌فرض موتور قواعد (rules)
//	۲. پارامتر استقرار شرکت (deps.Org.Params)
//	۳. رکورد پارامتر «جاری» با تازه‌ترین تاریخ اجرا (این ماژول)
//
// رکورد «غیرفعال» اثر نمی‌گذارد ولی پاک نمی‌شود؛ پس بازگشت به مقدار پیشین همیشه ممکن است.
package career

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"hrms/internal/contracts"
	"hrms/internal/modules/career/domain"
	"hrms/internal/platform/calendar"
	"hrms/internal/platform/httpx"
	"hrms/internal/platform/modulekit"
	"hrms/internal/platform/recordkit"
	"hrms/internal/rules"
)

// paramSourceView یک پارامتر با مقدار مؤثر، منبع مقدار و امکان ویرایش.
type paramSourceView struct {
	Key     string  `json:"key"`
	TitleFa string  `json:"titleFa"`
	UnitFa  string  `json:"unitFa"`
	GroupFa string  `json:"groupFa"`
	Default float64 `json:"default"`
	Value   float64 `json:"value"`
	// Source یکی از default|company|record است: پیش‌فرض سامانه، پارامتر شرکت، مقدار ثبت‌شده.
	Source   string `json:"source"`
	SourceFa string `json:"sourceFa"`
	// RecordID شناسهٔ رکورد پارامتر جاری (خالی = مقداری ثبت نشده است).
	RecordID string `json:"recordId,omitempty"`
	// EffectiveFrom تاریخ اجرای مقدار ثبت‌شده.
	EffectiveFrom string `json:"effectiveFrom,omitempty"`
	Boolean       bool   `json:"boolean,omitempty"`
	Changed       bool   `json:"changed"`
}

// registerAppointmentParamEndpoints مسیرهای خواندن و تغییر پارامترهای انتصاب و ترفیع.
func registerAppointmentParamEndpoints(mux *http.ServeMux, deps modulekit.Deps, svc *recordkit.Service) {
	read := func(fn http.HandlerFunc) http.HandlerFunc { return deps.Require("career", "read", fn) }
	write := func(fn http.HandlerFunc) http.HandlerFunc { return deps.Require("career", "write", fn) }

	// فهرست پارامترها با مقدار مؤثر و منبع آن (فرم انتصابات این را نشان می‌دهد).
	mux.HandleFunc("GET /api/v1/career/appointment-params", read(func(w http.ResponseWriter, r *http.Request) {
		params := appointmentParamViews(r.Context(), deps, svc)
		httpx.WriteJSON(w, http.StatusOK, map[string]any{
			"params": params,
			"total":  len(params),
			"noteFa": appointmentParamNoteFa(),
		})
	}))

	// ثبت مقدار تازه (افزایش/تغییر) یا کنارگذاشتن مقدار ثبت‌شده (کاهش).
	mux.HandleFunc("POST /api/v1/career/appointment-params", write(func(w http.ResponseWriter, r *http.Request) {
		var in appointmentParamRequest
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			httpx.WriteErrorMsg(w, http.StatusBadRequest, contracts.CodeBadRequest,
				"ورودی پارامتر خوانده نشد.", nil)
			return
		}
		out, msg := saveAppointmentParam(r.Context(), deps, svc, in, actorOf(r))
		if msg != "" {
			httpx.WriteErrorMsg(w, http.StatusBadRequest, contracts.CodeBadRequest, msg, nil)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, out)
	}))
}

// appointmentParamRequest ورودی تغییر یک پارامتر.
type appointmentParamRequest struct {
	Key string `json:"key"`
	// Value مقدار تازه؛ در حالت Remove نادیده گرفته می‌شود.
	Value string `json:"value"`
	// ReasonFa دلیل تغییر (در رکورد پارامتر می‌ماند).
	ReasonFa string `json:"reasonFa"`
	// EffectiveFrom تاریخ اجرا (اختیاری، شمسی).
	EffectiveFrom string `json:"effectiveFrom"`
	// Remove یعنی مقدار ثبت‌شده کنار گذاشته شود و مقدار شرکت/پیش‌فرض برگردد.
	Remove bool `json:"remove"`
}

// appointmentParamViews مقدار مؤثر همهٔ پارامترها را با منبعشان برمی‌گرداند.
func appointmentParamViews(ctx context.Context, deps modulekit.Deps, svc *recordkit.Service) []paramSourceView {
	company := companyParamValues(deps)
	records := activeParamRecords(ctx, svc)
	catalog := domain.AppointmentParamCatalog()
	out := make([]paramSourceView, 0, len(catalog))
	for _, p := range catalog {
		view := paramSourceView{
			Key: p.Key, TitleFa: p.TitleFa, UnitFa: p.UnitFa, GroupFa: p.GroupFa,
			Default: p.Default, Value: p.Default, Source: "default", SourceFa: "پیش‌فرض سامانه",
			Boolean: p.Boolean,
		}
		if v, ok := company[p.Key]; ok {
			view.Value = v
			view.Source = "company"
			view.SourceFa = "پارامتر شرکت"
		}
		if rec, ok := records[p.Key]; ok {
			if v, err := strconv.ParseFloat(strings.TrimSpace(rec.Fields["value"]), 64); err == nil {
				view.Value = v
				view.Source = "record"
				view.SourceFa = "مقدار ثبت‌شده در فرم"
				view.RecordID = rec.ID
				view.EffectiveFrom = strings.TrimSpace(rec.Fields["effectiveFrom"])
			}
		}
		view.Changed = view.Value != view.Default
		out = append(out, view)
	}
	return out
}

// saveAppointmentParam مقدار یک پارامتر را ثبت یا کنار می‌گذارد.
//
// قاعده‌ها:
//   - کلید باید از فهرست پارامترهای دامنه باشد.
//   - مقدار باید عدد نامنفی باشد و پارامتر بولی فقط صفر یا یک بپذیرد (همان قاعدهٔ رکورد).
//   - هر ثبت، رکورد پیشین همان کلید را «غیرفعال» می‌کند تا سابقه بماند و تازه‌ترین رکورد
//     جاری، مقدار مؤثر شود (تاریخ‌گذاری مؤثر بدون بازنویسی سابقه).
//   - حالت Remove همان کنارگذاشتن است: مقدار شرکت/پیش‌فرض برمی‌گردد.
func saveAppointmentParam(ctx context.Context, deps modulekit.Deps, svc *recordkit.Service, in appointmentParamRequest, actor string) (map[string]any, string) {
	key := strings.TrimSpace(in.Key)
	view, ok := domain.AppointmentParamByKey(key)
	if !ok {
		return nil, "کلید پارامتر از فهرست پارامترهای انتصاب و ترفیع نیست."
	}
	existing := activeParamRecords(ctx, svc)

	if in.Remove {
		rec, has := existing[key]
		if !has {
			return nil, "برای این پارامتر مقداری ثبت نشده است که کنار گذاشته شود."
		}
		reason := strings.TrimSpace(in.ReasonFa)
		if reason == "" {
			reason = "بازگشت به مقدار پارامتر شرکت/پیش‌فرض سامانه."
		}
		if _, err := svc.Act(ctx, rec.ID, contracts.RecordAction{Action: "deactivate", NoteFa: reason}, actor); err != nil {
			return nil, err.Error()
		}
		return map[string]any{
			"key": key, "removed": true,
			"params": appointmentParamViews(ctx, deps, svc),
			"noteFa": "مقدار ثبت‌شدهٔ «" + view.TitleFa + "» کنار گذاشته شد؛ از این پس مقدار پارامتر شرکت " +
				"یا پیش‌فرض سامانه به کار می‌رود. سابقهٔ تغییر پاک نشد.",
		}, ""
	}

	raw := strings.TrimSpace(in.Value)
	if raw == "" {
		return nil, "مقدار پارامتر را وارد کنید."
	}
	num, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return nil, "مقدار پارامتر باید عدد باشد."
	}
	if num < 0 {
		return nil, "مقدار پارامتر نمی‌تواند منفی باشد."
	}
	if view.Boolean && num != 0 && num != 1 {
		return nil, "مقدار این پارامتر فقط ۰ (خیر) یا ۱ (بله) است."
	}
	if from := strings.TrimSpace(in.EffectiveFrom); from != "" {
		if _, err := calendar.Parse(from); err != nil {
			return nil, "تاریخ اجرا باید به قالب شمسی YYYY/MM/DD باشد."
		}
	}

	// رکوردهای جاری همین کلید کنار گذاشته می‌شوند تا فقط یک مقدار جاری بماند.
	for _, rec := range existing {
		if strings.TrimSpace(rec.Fields["key"]) != key {
			continue
		}
		if _, err := svc.Act(ctx, rec.ID, contracts.RecordAction{
			Action: "deactivate", NoteFa: "جایگزینی با مقدار تازه.",
		}, actor); err != nil {
			return nil, err.Error()
		}
	}

	rec, err := svc.Create(ctx, contracts.RecordInput{
		Kind:  domain.KindAppointmentParams,
		Title: "پارامتر " + view.TitleFa + " = " + raw,
		Fields: map[string]string{
			"key": key, "value": raw, "titleFa": view.TitleFa,
			"unitFa": view.UnitFa, "groupFa": view.GroupFa,
			"effectiveFrom": strings.TrimSpace(in.EffectiveFrom),
			"reason":        strings.TrimSpace(in.ReasonFa),
		},
		NoteFa: strings.TrimSpace(in.ReasonFa),
	}, actor)
	if err != nil {
		return nil, err.Error()
	}
	if _, err := svc.Act(ctx, rec.ID, contracts.RecordAction{
		Action: "activate", NoteFa: "جاری‌سازی مقدار تازهٔ پارامتر.",
	}, actor); err != nil {
		return nil, err.Error()
	}

	return map[string]any{
		"key": key, "value": num, "recordId": rec.ID, "removed": false,
		"params": appointmentParamViews(ctx, deps, svc),
		"noteFa": "مقدار «" + view.TitleFa + "» ثبت و جاری شد؛ همین مقدار در سنجش تصویب‌خواهی " +
			"انتصاب و ترفیع به کار می‌رود و سابقهٔ تغییر در رکورد پارامتر می‌ماند.",
	}, ""
}

// activeParamRecords رکوردهای پارامتر «جاری» را به تفکیک کلید برمی‌گرداند.
//
// قاعده: رکورد غیرفعال و پیش‌نویس اثر ندارد. اگر چند رکورد جاری از یک کلید باشد، تازه‌ترین
// تاریخ اجرا برنده است و در نبود تاریخ، تازه‌ترین شناسه (شمارندهٔ رکورد) ملاک است.
func activeParamRecords(ctx context.Context, svc *recordkit.Service) map[string]contracts.Record {
	out := map[string]contracts.Record{}
	if svc == nil {
		return out
	}
	page, err := svc.List(ctx, contracts.RecordFilter{
		Module: "career", Kind: domain.KindAppointmentParams, Scope: "all", Size: 500,
	})
	if err != nil {
		return out
	}
	for _, rec := range page.Items {
		if rec.Status != domain.StatusActiveParam {
			continue
		}
		key := strings.TrimSpace(rec.Fields["key"])
		if key == "" {
			continue
		}
		current, has := out[key]
		if !has || paramRecordNewer(rec, current) {
			out[key] = rec
		}
	}
	return out
}

// paramRecordNewer آیا رکورد تازه از رکورد جاری جدیدتر است؟ (تاریخ اجرا و سپس شناسه)
func paramRecordNewer(candidate, current contracts.Record) bool {
	from := strings.TrimSpace(candidate.Fields["effectiveFrom"])
	to := strings.TrimSpace(current.Fields["effectiveFrom"])
	if from != to {
		return from > to
	}
	return candidate.ID > current.ID
}

// appointmentParamOverrides مقدار پارامترهای ثبت‌شده در فرم (رکوردهای جاری) را برمی‌گرداند.
func appointmentParamOverrides(ctx context.Context, svc *recordkit.Service) map[string]float64 {
	out := map[string]float64{}
	for key, rec := range activeParamRecords(ctx, svc) {
		if v, err := strconv.ParseFloat(strings.TrimSpace(rec.Fields["value"]), 64); err == nil {
			out[key] = v
		}
	}
	return out
}

// companyParamValues پارامترهای عددی همهٔ گروه‌های استقرار شرکت (کلید → مقدار).
func companyParamValues(deps modulekit.Deps) map[string]float64 {
	out := map[string]float64{}
	for _, p := range deps.Org.Params {
		if v, err := strconv.ParseFloat(strings.TrimSpace(p.Value), 64); err == nil {
			out[p.Key] = v
		}
	}
	return out
}

// appointmentParamNoteFa یادداشت مشترک پنل پارامترها.
func appointmentParamNoteFa() string {
	return "ترتیب تقدم مقدار: مقدار ثبت‌شده در همین فرم ← پارامتر استقرار شرکت ← پیش‌فرض " +
		"سامانه. کنارگذاشتن یک مقدار، سابقهٔ آن را پاک نمی‌کند و مقدار شرکت/پیش‌فرض برمی‌گردد. " +
		"تغییر پارامتر، تصویب‌خواهی‌های جدید را با مقدار تازه می‌سنجد."
}

// rulesQualificationParams پارامترهای شرایط احراز را از لایه‌های سه‌گانه می‌سازد.
//
// چرا همین تابع برای سنجش: فرم، امضای سطح تصویب و دروازهٔ اجرا باید یک عدد ببینند؛ پس همه
// از همین یک مسیر مقدار می‌خوانند.
func rulesQualificationParams(deps modulekit.Deps, overrides map[string]float64) rules.QualificationParams {
	values := map[string]float64{}
	company := companyParamValues(deps)
	for k, v := range company {
		if strings.HasPrefix(k, "qual.") {
			values[k] = v
		}
	}
	for k, v := range overrides {
		if strings.HasPrefix(k, "qual.") {
			values[k] = v
		}
	}
	return rules.QualificationParamsFromValues(values)
}
