// پنل پارامتریک فرم‌ها و گزارش‌ها.
//
// چرا این فایل: مدیر سامانه باید بدون تغییر کد بتواند فیلدهای یک فرم را کم و زیاد کند
// (نمایش/پنهان)، اجباری‌بودن را عوض کند، برچسب فارسی را بازنویسی کند و ستون‌های یک
// گزارش را انتخاب کند. منبع حقیقتِ «چه فیلدهایی وجود دارد» همان شناسهٔ رکورد هر ماژول
// است (recordkit.Schema) و منبع «چه ستونی برای گزارش هست» متادیتای منابع گزارش؛ این
// فایل فقط آن‌ها را می‌خواند و پارامترهای ذخیره‌شده را رویشان می‌نشاند. هیچ فهرست
// دستی‌ای در سرور یا رابط تکرار نمی‌شود.
package settings

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"hrms/internal/contracts"
	"hrms/internal/modules/settings/domain"
	"hrms/internal/platform/auth"
	"hrms/internal/platform/documents"
	"hrms/internal/platform/httpx"
	"hrms/internal/platform/modulekit"
	"hrms/internal/platform/recordkit"
)

// formFieldView فیلد یک فرم به‌همراه وضعیت نمایش پس از اعمال پارامترها.
type formFieldView struct {
	Key      string `json:"key"`
	LabelFa  string `json:"labelFa"`
	Type     string `json:"type"`
	Required bool   `json:"required"`
	Visible  bool   `json:"visible"`
	// Source منبع خودکار مقدار (employee.mobile و مانند آن)؛ خالی یعنی ورودی دستی.
	Source string `json:"source,omitempty"`
	// Custom یعنی این فیلد در دامنهٔ ماژول نیست و از پنل پارامترها به فرم افزوده شده
	// است؛ مقدارش در همان رکورد ذخیره می‌شود.
	Custom bool `json:"custom,omitempty"`
}

// formView یک فرم (نوع رکورد) با فیلدهای مؤثرش.
type formView struct {
	Module        string          `json:"module"`
	ModuleTitleFa string          `json:"moduleTitleFa"`
	Kind          string          `json:"kind"`
	TitleFa       string          `json:"titleFa"`
	Target        string          `json:"target"`
	Fields        []formFieldView `json:"fields"`
	FieldCount    int             `json:"fieldCount"`
	VisibleCount  int             `json:"visibleCount"`
	RequiredCount int             `json:"requiredCount"`
	// CustomCount شمار فیلدهای سفارشی افزوده‌شده از پنل پارامترها.
	CustomCount int `json:"customCount"`
	// ParamsID شناسهٔ رکورد پارامتر همین فرم (خالی = پارامتری ثبت نشده است).
	ParamsID string `json:"paramsId,omitempty"`
	Status   string `json:"status,omitempty"`
	StatusFa string `json:"statusFa,omitempty"`
}

// reportColumnView یک ستون قابل انتخاب در گزارش.
type reportColumnView struct {
	Key      string `json:"key"`
	LabelFa  string `json:"labelFa"`
	Type     string `json:"type"`
	Selected bool   `json:"selected"`
}

// reportView یک منبع دادهٔ گزارش با ستون‌هایش.
type reportView struct {
	Source       string             `json:"source"`
	TitleFa      string             `json:"titleFa"`
	Columns      []reportColumnView `json:"columns"`
	ColumnCount  int                `json:"columnCount"`
	SelectedCols []string           `json:"selectedColumns"`
	ParamsID     string             `json:"paramsId,omitempty"`
	Status       string             `json:"status,omitempty"`
}

// registerFormParamEndpoints پایانه‌های کاتالوگ فرم‌ها و گزارش‌ها را ثبت می‌کند.
func registerFormParamEndpoints(mux *http.ServeMux, deps modulekit.Deps, svc *recordkit.Service) {
	if svc == nil {
		return
	}
	read := func(fn http.HandlerFunc) http.HandlerFunc { return deps.Require("settings", "read", fn) }

	mux.HandleFunc("GET /api/v1/settings/forms", read(func(w http.ResponseWriter, r *http.Request) {
		forms := buildFormCatalog(r, deps, svc)
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"forms": forms, "total": len(forms)})
	}))

	mux.HandleFunc("GET /api/v1/settings/reports", read(func(w http.ResponseWriter, r *http.Request) {
		reports := buildReportCatalog(r, deps, svc)
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"reports": reports, "total": len(reports)})
	}))

	// بلوک‌های اسناد چاپی: کدام بخش‌های حکم/قرارداد/گواهی/فیش روی کاغذ بیاید و با چه
	// عنوانی. کاتالوگ از خود موتور اسناد می‌آید و پارامتر (scope=document) در همین ماژول
	// ذخیره می‌شود؛ موتور اسناد آن را در زمان چاپ می‌خواند.
	mux.HandleFunc("GET /api/v1/settings/documents", read(func(w http.ResponseWriter, r *http.Request) {
		documents := buildDocumentCatalog(r, deps, svc)
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"documents": documents, "total": len(documents)})
	}))

	// افزودن یا حذف «فیلد سفارشی» روی یک فرم: مدیر سامانه می‌تواند فیلدی را که در دامنهٔ
	// ماژول نیست به فرم بیفزاید و مقدارش در همان رکورد ذخیره شود، یا فیلد افزوده را کم کند.
	write := func(fn http.HandlerFunc) http.HandlerFunc { return deps.Require("settings", "write", fn) }
	mux.HandleFunc("POST /api/v1/settings/form-fields", write(func(w http.ResponseWriter, r *http.Request) {
		var in extraFieldRequest
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			httpx.WriteErrorMsg(w, http.StatusBadRequest, contracts.CodeBadRequest,
				"ورودی فیلد سفارشی خوانده نشد.", nil)
			return
		}
		out, msg := saveExtraField(r.Context(), deps, svc, in, settingsActor(r))
		if msg != "" {
			httpx.WriteErrorMsg(w, http.StatusBadRequest, contracts.CodeBadRequest, msg, nil)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, out)
	}))

	// فهرست تعریف گزارش‌های موجود (از پورت ماژول گزارش‌ها) تا پنل پارامترها بتواند
	// گزارش تازه بسازد یا گزارش موجود را غیرفعال/حذف کند. ساخت و تغییر با مسیر رسمی
	// همان ماژول انجام می‌شود (`/api/v1/reports/records`) تا قاعده در یک جا بماند.
	mux.HandleFunc("GET /api/v1/settings/report-defs", read(func(w http.ResponseWriter, r *http.Request) {
		defs := buildReportDefs(r.Context(), deps)
		httpx.WriteJSON(w, http.StatusOK, map[string]any{
			"reports": defs, "total": len(defs),
			"sources": reportSourceOptions(deps),
			"noteFa": "تعریف گزارش، منبع داده و ستون‌های آن را نگه می‌دارد؛ گزارش غیرفعال " +
				"پاک نمی‌شود و با «فعال‌سازی دوباره» برمی‌گردد.",
		})
	}))
}

// paramRecord پارامتر ذخیره‌شدهٔ یک هدف.
type paramRecord struct {
	ID       string
	Status   string
	Fields   map[string]string
	TitleFa  string
	StatusFa string
}

// savedParams پارامترهای ذخیره‌شده را به تفکیک هدف می‌خواند (scope:form و scope:report).
func savedParams(r *http.Request, svc *recordkit.Service, scope string) map[string]paramRecord {
	out := map[string]paramRecord{}
	page, err := svc.List(r.Context(), contracts.RecordFilter{
		Module: "settings", Kind: domain.KindFormParams, Scope: "all", Page: 1, Size: 500,
	})
	if err != nil {
		return out
	}
	for _, rec := range page.Items {
		if strings.TrimSpace(rec.Fields["scope"]) != scope {
			continue
		}
		target := strings.TrimSpace(rec.Fields["target"])
		if target == "" {
			continue
		}
		// پارامتر غیرفعال در فرم اثر نمی‌گذارد ولی پاک نمی‌شود.
		if rec.Status == "inactive" {
			continue
		}
		out[normalizeTarget(target)] = paramRecord{
			ID: rec.ID, Status: rec.Status, Fields: rec.Fields, TitleFa: rec.Title,
			StatusFa: statusTitleFa(rec.Status),
		}
	}
	return out
}

// normalizeTarget کلید یکتا برای هدف پارامتر (ماژول|نوع، بی‌فاصله و کوچک).
func normalizeTarget(target string) string {
	parts := strings.Split(target, "|")
	for i := range parts {
		parts[i] = strings.ToLower(strings.TrimSpace(parts[i]))
	}
	return strings.Join(parts, "|")
}

// csvSet رشتهٔ جداشده با کاما/ویرگول فارسی را به مجموعه تبدیل می‌کند.
func csvSet(value string) map[string]bool {
	out := map[string]bool{}
	replacer := strings.NewReplacer("،", ",", "؛", ",", "|", ",")
	for _, part := range strings.Split(replacer.Replace(value), ",") {
		key := strings.TrimSpace(part)
		if key != "" {
			out[key] = true
		}
	}
	return out
}

// csvList ستون‌ها را به ترتیب ورودی برمی‌گرداند (برای گزارش‌ها که ترتیب مهم است).
func csvList(value string) []string {
	replacer := strings.NewReplacer("،", ",", "؛", ",", "|", ",")
	out := []string{}
	for _, part := range strings.Split(replacer.Replace(value), ",") {
		key := strings.TrimSpace(part)
		if key != "" {
			out = append(out, key)
		}
	}
	return out
}

// labelOverrides از قالب «key=برچسب، key=برچسب» نگاشت برچسب می‌سازد.
func labelOverrides(value string) map[string]string {
	out := map[string]string{}
	replacer := strings.NewReplacer("،", ",", "؛", ",")
	for _, part := range strings.Split(replacer.Replace(value), ",") {
		pair := strings.SplitN(part, "=", 2)
		if len(pair) != 2 {
			continue
		}
		key := strings.TrimSpace(pair[0])
		label := strings.TrimSpace(pair[1])
		if key != "" && label != "" {
			out[key] = label
		}
	}
	return out
}

// extraField یک فیلد سفارشی افزوده‌شده از پنل پارامترها.
type extraField struct {
	Key     string
	LabelFa string
	Type    string
}

// allowedExtraTypes نوع‌های مجاز فیلد سفارشی (همان نوع‌های موتور رکورد).
var allowedExtraTypes = map[string]bool{
	"text": true, "date": true, "number": true, "money": true, "select": true,
}

// parseExtraFields رشتهٔ فیلدهای سفارشی را می‌شکند. قالب هر قلم «کلید|برچسب|نوع» و
// جداکنندهٔ قلم‌ها «؛» یا خط تازه است. قلم ناقص یا نوع ناشناخته نادیده گرفته می‌شود تا
// رکورد خراب، فرم را نشکند.
func parseExtraFields(value string) []extraField {
	out := []extraField{}
	replacer := strings.NewReplacer("\n", "؛", "\r", "؛", ";", "؛")
	for _, part := range strings.Split(replacer.Replace(value), "؛") {
		item := strings.TrimSpace(part)
		if item == "" {
			continue
		}
		pieces := strings.Split(item, "|")
		if len(pieces) < 2 {
			continue
		}
		key := strings.TrimSpace(pieces[0])
		label := strings.TrimSpace(pieces[1])
		typ := "text"
		if len(pieces) >= 3 {
			if t := strings.TrimSpace(pieces[2]); allowedExtraTypes[t] {
				typ = t
			}
		}
		if key == "" || label == "" {
			continue
		}
		out = append(out, extraField{Key: key, LabelFa: label, Type: typ})
	}
	return out
}

// formatExtraFields فیلدهای سفارشی را به همان قالب ذخیره‌شدنی برمی‌گرداند.
func formatExtraFields(fields []extraField) string {
	parts := make([]string, 0, len(fields))
	for _, f := range fields {
		parts = append(parts, f.Key+"|"+f.LabelFa+"|"+f.Type)
	}
	return strings.Join(parts, "؛")
}

// extraFieldRequest ورودی افزودن/حذف فیلد سفارشی.
type extraFieldRequest struct {
	// Target قالب «ماژول|کلید نوع رکورد» فرم هدف.
	Target  string `json:"target"`
	Key     string `json:"key"`
	LabelFa string `json:"labelFa"`
	Type    string `json:"type"`
	// Remove یعنی همین فیلد از فهرست سفارشی حذف شود (کاهش فیلد).
	Remove bool `json:"remove"`
}

// saveExtraField فیلد سفارشی را به فرم می‌افزاید یا از آن کم می‌کند.
//
// قاعده‌ها:
//   - هدف باید فرم ثبت‌شده در سامانه باشد (ماژول|نوع)؛ فیلد سفارشی روی فرم ناشناخته
//     ساخته نمی‌شود.
//   - کلید نباید با فیلد دامنهٔ همان فرم یکی باشد؛ بازنویسی فیلد رسمی مجاز نیست (برای
//     تغییر برچسب، پارامتر «labels» است).
//   - رکورد پارامتر فرم اگر نباشد ساخته و جاری می‌شود و اگر باشد به‌روزرسانی می‌شود؛ پس
//     برای هر فرم یک رکورد کافی است و سابقهٔ تغییر در آن می‌ماند.
func saveExtraField(ctx context.Context, deps modulekit.Deps, svc *recordkit.Service, in extraFieldRequest, actor string) (map[string]any, string) {
	target := normalizeTarget(in.Target)
	if target == "" {
		return nil, "فرم هدف را انتخاب کنید."
	}
	moduleID, kindKey := "", ""
	if parts := strings.SplitN(target, "|", 2); len(parts) == 2 {
		moduleID, kindKey = parts[0], parts[1]
	}
	if moduleID == "" || kindKey == "" {
		return nil, "قالب هدف باید «ماژول|کلید نوع رکورد» باشد."
	}
	port, ok := deps.Ports.RecordList(moduleID)
	if !ok {
		return nil, "ماژول انتخاب‌شده فرم رکوردی ندارد."
	}
	domainFields := map[string]bool{}
	for _, kind := range port.Schema().Kinds {
		if strings.ToLower(kind.Key) != kindKey {
			continue
		}
		for _, f := range kind.Fields {
			domainFields[f.Key] = true
		}
	}
	if len(domainFields) == 0 {
		return nil, "نوع رکورد انتخاب‌شده در آن ماژول یافت نشد."
	}

	// رکورد پارامتر همین فرم (اگر باشد) خوانده و فهرست سفارشی ویرایش می‌شود.
	var existing *contracts.Record
	if page, err := svc.List(ctx, contracts.RecordFilter{
		Module: "settings", Kind: domain.KindFormParams, Scope: "all", Size: 500,
	}); err == nil {
		for i := range page.Items {
			rec := page.Items[i]
			if strings.TrimSpace(rec.Fields["scope"]) != "form" {
				continue
			}
			if normalizeTarget(rec.Fields["target"]) != target {
				continue
			}
			existing = &rec
			break
		}
	}

	current := []extraField{}
	fields := map[string]string{"scope": "form", "target": target}
	if existing != nil {
		current = parseExtraFields(existing.Fields["extraFields"])
		for k, v := range existing.Fields {
			fields[k] = v
		}
	}

	key := strings.TrimSpace(in.Key)
	// final فهرست سفارشی پس از تغییر است؛ پاسخ سرور باید وضعیت تازه را نشان دهد، نه
	// فهرست پیش از ویرایش (وگرنه کاربر گمان می‌کند حذف انجام نشده است).
	final := current
	if in.Remove {
		if key == "" {
			return nil, "کلید فیلدی که باید حذف شود را مشخص کنید."
		}
		kept := make([]extraField, 0, len(current))
		for _, f := range current {
			if f.Key != key {
				kept = append(kept, f)
			}
		}
		if len(kept) == len(current) {
			return nil, "چنین فیلد سفارشی روی این فرم ثبت نشده است."
		}
		final = kept
		fields["extraFields"] = formatExtraFields(kept)
	} else {
		if key == "" || strings.TrimSpace(in.LabelFa) == "" {
			return nil, "کلید و برچسب فیلد سفارشی الزامی است."
		}
		if domainFields[key] {
			return nil, "این کلید فیلد رسمی همین فرم است؛ برای تغییر برچسبش از پارامتر «برچسب فارسی سفارشی» استفاده کنید."
		}
		typ := strings.TrimSpace(in.Type)
		if typ == "" {
			typ = "text"
		}
		if !allowedExtraTypes[typ] {
			return nil, "نوع فیلد سفارشی باید یکی از text، date، number، money یا select باشد."
		}
		updated := false
		for i := range current {
			if current[i].Key == key {
				current[i] = extraField{Key: key, LabelFa: strings.TrimSpace(in.LabelFa), Type: typ}
				updated = true
			}
		}
		if !updated {
			current = append(current, extraField{Key: key, LabelFa: strings.TrimSpace(in.LabelFa), Type: typ})
		}
		final = current
		fields["extraFields"] = formatExtraFields(current)
	}

	if existing == nil {
		rec, err := svc.Create(ctx, contracts.RecordInput{
			Kind: domain.KindFormParams, Title: "پارامتر فرم " + target, Fields: fields,
			NoteFa: "ساخت پارامتر فرم از پنل تنظیمات.",
		}, actor)
		if err != nil {
			return nil, err.Error()
		}
		// پارامتر تازه بلافاصله اثر کند، نه فقط پس از تأیید دستی.
		if _, err := svc.Act(ctx, rec.ID, contracts.RecordAction{
			Action: "activate", NoteFa: "جاری‌سازی پارامتر فرم.",
		}, actor); err != nil {
			return nil, err.Error()
		}
	} else if _, err := svc.Update(ctx, existing.ID, contracts.RecordInput{
		Kind: domain.KindFormParams, Title: existing.Title, Fields: fields,
	}, actor); err != nil {
		return nil, err.Error()
	}

	action := "افزوده"
	if in.Remove {
		action = "کم"
	}
	return map[string]any{
		"target": target, "key": key, "customFields": final,
		"noteFa": "فیلد «" + key + "» " + action + " شد؛ فرم‌های همین نوع از این پس با فهرست " +
			"تازه نمایش داده می‌شوند و مقدار فیلد سفارشی در همان رکورد ذخیره می‌شود.",
	}, ""
}

// reportDefView تعریف گزارش برای پنل پارامترها.
type reportDefView struct {
	ID       string `json:"id"`
	TitleFa  string `json:"titleFa"`
	Source   string `json:"source"`
	Items    string `json:"items"`
	GroupBy  string `json:"groupBy,omitempty"`
	Limit    string `json:"limit,omitempty"`
	Status   string `json:"status"`
	StatusFa string `json:"statusFa"`
}

// buildReportDefs تعریف گزارش‌های موجود را از پورت ماژول گزارش‌ها می‌خواند.
func buildReportDefs(ctx context.Context, deps modulekit.Deps) []reportDefView {
	out := []reportDefView{}
	port, ok := deps.Ports.RecordList("reports")
	if !ok {
		return out
	}
	page, err := port.List(ctx, contracts.RecordFilter{
		Module: "reports", Kind: "reportDefs", Scope: "all", Size: 300,
	})
	if err != nil {
		return out
	}
	for _, rec := range page.Items {
		out = append(out, reportDefView{
			ID: rec.ID, TitleFa: rec.Title,
			Source:  strings.TrimSpace(rec.Fields["source"]),
			Items:   strings.TrimSpace(rec.Fields["items"]),
			GroupBy: strings.TrimSpace(rec.Fields["groupBy"]),
			Limit:   strings.TrimSpace(rec.Fields["limit"]),
			Status:  rec.Status, StatusFa: rec.StatusFa,
		})
	}
	return out
}

// reportSourceOptions فهرست منابع دادهٔ گزارش برای ساخت تعریف تازه.
func reportSourceOptions(deps modulekit.Deps) []map[string]string {
	out := []map[string]string{}
	for _, source := range deps.Ports.ReportSources() {
		meta := source.Metadata()
		fields := make([]string, 0, len(meta.Fields))
		for _, f := range meta.Fields {
			fields = append(fields, f.Key+"="+f.LabelFa)
		}
		out = append(out, map[string]string{
			"key": meta.Key, "titleFa": meta.TitleFa, "fields": strings.Join(fields, "،"),
		})
	}
	return out
}

// settingsActor نام کاربر جاری برای ثبت در رکورد پارامتر.
func settingsActor(r *http.Request) string {
	if user, ok := auth.UserFrom(r.Context()); ok {
		return user.Username
	}
	return "ناشناس"
}

func statusTitleFa(status string) string {
	switch status {
	case "active":
		return "جاری"
	case "inactive":
		return "غیرفعال"
	case "draft":
		return "پیش‌نویس"
	}
	return status
}

// buildFormCatalog کاتالوگ همهٔ فرم‌های رکوردی سامانه با پارامترهای اعمال‌شده.
func buildFormCatalog(r *http.Request, deps modulekit.Deps, svc *recordkit.Service) []formView {
	params := savedParams(r, svc, "form")
	out := []formView{}
	for _, moduleID := range deps.Ports.RecordModules() {
		port, ok := deps.Ports.RecordList(moduleID)
		if !ok {
			continue
		}
		schema := port.Schema()
		for _, kind := range schema.Kinds {
			target := normalizeTarget(moduleID + "|" + kind.Key)
			record, hasParams := params[target]
			visible := map[string]bool(nil)
			required := map[string]bool{}
			labels := map[string]string{}
			if hasParams {
				if raw := strings.TrimSpace(record.Fields["visibleFields"]); raw != "" {
					visible = csvSet(raw)
				}
				required = csvSet(record.Fields["requiredFields"])
				labels = labelOverrides(record.Fields["labels"])
			}
			fields := make([]formFieldView, 0, len(kind.Fields))
			visibleCount, requiredCount := 0, 0
			for _, f := range kind.Fields {
				shown := visible == nil || visible[f.Key]
				label := f.LabelFa
				if custom, ok := labels[f.Key]; ok && custom != "" {
					label = custom
				}
				isRequired := f.Required || required[f.Key]
				if shown {
					visibleCount++
				}
				if isRequired && shown {
					requiredCount++
				}
				fields = append(fields, formFieldView{
					Key: f.Key, LabelFa: label, Type: f.Type,
					Required: isRequired, Visible: shown, Source: f.Source,
				})
			}
			view := formView{
				Module: moduleID, ModuleTitleFa: schema.ModuleTitleFa,
				Kind: kind.Key, TitleFa: kind.TitleFa, Target: target,
				Fields: fields, FieldCount: len(fields),
				VisibleCount: visibleCount, RequiredCount: requiredCount,
			}
			// فیلدهای سفارشی: افزوده‌های پنل پارامترها که در دامنهٔ ماژول نیستند. عمداً پس
			// از فیلدهای دامنه می‌آیند تا ترتیب سند رسمی به‌هم نخورد، و همیشه نمایان‌اند
			// (برای «کم کردن» یک فیلد سفارشی، همان فیلد از فهرست سفارشی حذف می‌شود).
			if hasParams {
				for _, custom := range parseExtraFields(record.Fields["extraFields"]) {
					label := custom.LabelFa
					if over, ok := labels[custom.Key]; ok && over != "" {
						label = over
					}
					view.Fields = append(view.Fields, formFieldView{
						Key: custom.Key, LabelFa: label, Type: custom.Type,
						Required: required[custom.Key], Visible: true, Custom: true,
					})
					view.CustomCount++
					view.FieldCount++
					view.VisibleCount++
					if required[custom.Key] {
						view.RequiredCount++
					}
				}
			}
			if hasParams {
				view.ParamsID = record.ID
				view.Status = record.Status
				view.StatusFa = record.StatusFa
			}
			out = append(out, view)
		}
	}
	return out
}

// buildReportCatalog کاتالوگ منابع دادهٔ گزارش با ستون‌های قابل انتخاب.
func buildReportCatalog(r *http.Request, deps modulekit.Deps, svc *recordkit.Service) []reportView {
	params := savedParams(r, svc, "report")
	out := []reportView{}
	for _, source := range deps.Ports.ReportSources() {
		meta := source.Metadata()
		target := normalizeTarget(meta.Key)
		record, hasParams := params[target]
		selected := map[string]bool{}
		selectedList := []string{}
		if hasParams {
			for _, col := range csvList(record.Fields["columns"]) {
				selected[col] = true
				selectedList = append(selectedList, col)
			}
		}
		columns := make([]reportColumnView, 0, len(meta.Fields))
		for _, f := range meta.Fields {
			// تا وقتی پارامتری ثبت نشده باشد، همهٔ ستون‌ها انتخاب‌شده نشان داده می‌شوند.
			isSelected := len(selected) == 0 || selected[f.Key]
			columns = append(columns, reportColumnView{Key: f.Key, LabelFa: f.LabelFa, Type: f.Type, Selected: isSelected})
		}
		view := reportView{
			Source: meta.Key, TitleFa: meta.TitleFa, Columns: columns,
			ColumnCount: len(columns), SelectedCols: selectedList,
		}
		if hasParams {
			view.ParamsID = record.ID
			view.Status = record.Status
		}
		out = append(out, view)
	}
	return out
}

// documentBlockView یک بلوک سند با وضعیت نمایش پس از اعمال پارامتر.
type documentBlockView struct {
	Key     string `json:"key"`
	TitleFa string `json:"titleFa"`
	Visible bool   `json:"visible"`
	// TitleFaOverridden عنوان سفارشی (خالی = عنوان پیش‌فرض موتور اسناد).
	TitleFaOverridden string `json:"customTitle,omitempty"`
}

// documentTextView یک جای متن قابل ویرایش در سند چاپی (پاراگراف متن، یادداشت پای سند).
type documentTextView struct {
	Key     string `json:"key"`
	TitleFa string `json:"titleFa"`
	HintFa  string `json:"hintFa"`
	// Text همان چیزی است که در کادر ویرایش می‌آید: متن ذخیره‌شده، وگرنه پیش‌فرض مؤثر.
	Text string `json:"text"`
	// Default متن پیش‌فرض مؤثر (قالب متن فعال اگر باشد، وگرنه پیش‌فرض موتور اسناد).
	Default string `json:"default"`
	// Custom یعنی برای این جای متن پارامتری ذخیره شده است.
	Custom bool `json:"custom"`
}

// documentView یک نوع سند چاپی با بلوک‌هایش.
type documentView struct {
	Kind      string              `json:"kind"`
	TitleFa   string              `json:"titleFa"`
	Blocks    []documentBlockView `json:"blocks"`
	Texts     []documentTextView  `json:"texts"`
	ParamsID  string              `json:"paramsId,omitempty"`
	Status    string              `json:"status,omitempty"`
	StatusFa  string              `json:"statusFa,omitempty"`
	VisibleNo int                 `json:"visibleCount"`
}

// buildDocumentCatalog کاتالوگ اسناد چاپی به‌همراه پارامتر ذخیره‌شدهٔ هر سند.
//
// چرا متن‌ها هم اینجا می‌آیند: پنل پارامتریک باید علاوه بر تیک بلوک‌ها، خودِ متن
// پاراگراف (مثلاً متن گواهی اشتغال) را در کادری نشان بدهد که همان متن مؤثر سند است —
// نه یک متن نمونهٔ جدا. اگر قالب متن فعالی برای سند ساخته شده باشد، همان قالب پیش‌فرض
// کادر است تا کاربر همان چیزی را ببیند که روی کاغذ می‌رود.
func buildDocumentCatalog(r *http.Request, deps modulekit.Deps, svc *recordkit.Service) []documentView {
	params := savedParams(r, svc, "document")
	out := []documentView{}
	for _, doc := range documents.Catalog() {
		record, hasParams := params[normalizeTarget(string(doc.Kind))]
		visible := map[string]bool{}
		custom := map[string]string{}
		if hasParams {
			for _, key := range csvList(record.Fields["visibleFields"]) {
				visible[key] = true
			}
			custom = labelOverrides(record.Fields["labels"])
		}
		blocks := make([]documentBlockView, 0, len(doc.Blocks))
		visibleCount := 0
		for _, block := range doc.Blocks {
			// پارامتر ثبت‌نشده یعنی همهٔ بلوک‌های پشتیبانی‌شده روشن‌اند.
			isVisible := len(visible) == 0 || visible[block.Key]
			if isVisible {
				visibleCount++
			}
			blocks = append(blocks, documentBlockView{
				Key: block.Key, TitleFa: block.TitleFa, Visible: isVisible,
				TitleFaOverridden: custom[block.Key],
			})
		}
		texts := buildDocumentTexts(r, deps, doc, record, hasParams)
		view := documentView{
			Kind: string(doc.Kind), TitleFa: doc.TitleFa, Blocks: blocks,
			Texts: texts, VisibleNo: visibleCount,
		}
		if hasParams {
			view.ParamsID = record.ID
			view.Status = record.Status
			view.StatusFa = record.StatusFa
		}
		out = append(out, view)
	}
	return out
}

// buildDocumentTexts جای‌های متنی یک سند را با متن مؤثرشان برمی‌گرداند.
func buildDocumentTexts(r *http.Request, deps modulekit.Deps, doc documents.DocCatalog, record paramRecord, hasParams bool) []documentTextView {
	defaults := documents.DefaultText(doc.Kind)
	if provider, ok := deps.Ports.DocumentTemplates(); ok {
		if tpl, found, err := provider.ActiveTemplate(r.Context(), string(doc.Kind)); err == nil && found {
			if body := strings.TrimSpace(tpl.Body); body != "" {
				defaults["body"] = body
			}
			if footer := strings.TrimSpace(tpl.Footer); footer != "" {
				defaults["note"] = footer
			}
		}
	}
	out := make([]documentTextView, 0, len(doc.Texts))
	for _, slot := range doc.Texts {
		fallback := defaults[slot.Key]
		stored := ""
		if hasParams {
			stored = strings.TrimSpace(record.Fields[slot.Key+"Text"])
		}
		text := stored
		if text == "" {
			text = fallback
		}
		out = append(out, documentTextView{
			Key: slot.Key, TitleFa: slot.TitleFa, HintFa: slot.HintFa,
			Text: text, Default: fallback, Custom: stored != "",
		})
	}
	return out
}
