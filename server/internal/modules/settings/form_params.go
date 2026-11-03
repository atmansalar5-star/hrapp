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
	"net/http"
	"strings"

	"hrms/internal/contracts"
	"hrms/internal/modules/settings/domain"
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
		documents := buildDocumentCatalog(r, svc)
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"documents": documents, "total": len(documents)})
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

// statusTitleFa عنوان فارسی وضعیت پارامتر.
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

// documentView یک نوع سند چاپی با بلوک‌هایش.
type documentView struct {
	Kind      string              `json:"kind"`
	TitleFa   string              `json:"titleFa"`
	Blocks    []documentBlockView `json:"blocks"`
	ParamsID  string              `json:"paramsId,omitempty"`
	Status    string              `json:"status,omitempty"`
	StatusFa  string              `json:"statusFa,omitempty"`
	VisibleNo int                 `json:"visibleCount"`
}

// buildDocumentCatalog کاتالوگ اسناد چاپی به‌همراه پارامتر ذخیره‌شدهٔ هر سند.
func buildDocumentCatalog(r *http.Request, svc *recordkit.Service) []documentView {
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
		view := documentView{
			Kind: string(doc.Kind), TitleFa: doc.TitleFa, Blocks: blocks, VisibleNo: visibleCount,
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
