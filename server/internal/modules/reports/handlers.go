// پایانه‌های HTTP ماژول «reports».

package reports

import (
	"context"
	"net/http"
	"strings"
	"time"

	"hrms/internal/contracts"
	"hrms/internal/platform/auth"
	"hrms/internal/platform/calendar"
	"hrms/internal/platform/datasource"
	"hrms/internal/platform/httpx"
	"hrms/internal/platform/modulekit"
	"hrms/internal/platform/repfilter"
)

// registerBuilderEndpoints مسیرهای گزارش‌ساز را ثبت می‌کند: فهرست منابع داده و اجرای
// گزارش بر پایهٔ اقلام انتخاب‌شده. منابع داده از پورت مشترک می‌آید؛ هر ماژول منبع خودش
// را ثبت می‌کند و این ماژول هیچ داده‌ای را مستقیم از ماژول دیگر نمی‌خواند.
func registerBuilderEndpoints(mux *http.ServeMux, deps modulekit.Deps) {
	report := func(fn http.HandlerFunc) http.HandlerFunc { return deps.Require("reports", "report", fn) }

	// قالب ورود اطلاعات دستی ستون‌های یک منبع گزارش: نوع، الگو، مقادیر مجاز و نمونه.
	// منبع همان ساختار داده است که قالب اکسل فرم‌ها از آن ساخته می‌شود.
	mux.HandleFunc("GET /api/v1/reports/entry-format", report(func(w http.ResponseWriter, r *http.Request) {
		source := strings.TrimSpace(r.URL.Query().Get("source"))
		if source == "" {
			list := make([]map[string]any, 0)
			for _, s := range deps.Ports.ReportSources() {
				meta := s.Metadata()
				list = append(list, map[string]any{
					"key": meta.Key, "titleFa": meta.TitleFa, "fields": len(meta.Fields),
				})
			}
			httpx.WriteJSON(w, http.StatusOK, map[string]any{"sources": list, "total": len(list)})
			return
		}
		s, ok := deps.Ports.ReportSource(source)
		if !ok {
			httpx.WriteErrorMsg(w, http.StatusNotFound, contracts.CodeNotFound,
				"منبع دادهٔ گزارش با این شناسه پیدا نشد؛ فهرست را از همین مسیر بدون پارامتر بگیرید.", nil)
			return
		}
		meta := s.Metadata()
		format := datasource.EntryFormatFromReportFields(meta.Fields)
		filters := make([]map[string]string, 0, len(meta.Filters))
		for _, f := range meta.Filters {
			filters = append(filters, map[string]string{"key": f.Key, "titleFa": f.LabelFa, "type": f.Type})
		}
		httpx.WriteJSON(w, http.StatusOK, map[string]any{
			"source": meta.Key, "titleFa": meta.TitleFa,
			"format": format, "sampleRow": datasource.SampleRow(format),
			"filters": filters,
			"noteFa": "این برگه می‌گوید ستون‌های این گزارش چه نوعی دارند و مقدار را چه شکلی وارد کنید؛ " +
				"همان الگوی قالب دستی فرم‌ها است.",
		})
	}))

	mux.HandleFunc("GET /api/v1/reports/sources", report(func(w http.ResponseWriter, r *http.Request) {
		sources := deps.Ports.ReportSources()
		metas := make([]contracts.ReportSourceMeta, 0, len(sources))
		for _, s := range sources {
			metas = append(metas, s.Metadata())
		}
		// عملگرهای شرط یک‌بار و مشترک اعلام می‌شوند؛ رابط شرط‌ساز را از همین می‌سازد
		// تا برچسب عملگرها در UI هاردکد نشود.
		httpx.WriteJSON(w, http.StatusOK, map[string]any{
			"sources": metas, "operators": repfilter.Operators(), "total": len(metas),
		})
	}))

	mux.HandleFunc("POST /api/v1/reports/run", report(func(w http.ResponseWriter, r *http.Request) {
		var req contracts.ReportRequest
		if !httpx.DecodeJSON(w, r, &req) {
			return
		}
		if err := repfilter.Validate(req.Conditions); err != nil {
			httpx.WriteErrorMsg(w, http.StatusBadRequest, contracts.CodeValidationFailed, err.Error(), nil)
			return
		}
		result, err := runReport(r.Context(), deps, req)
		if err != nil {
			httpx.WriteErrorMsg(w, http.StatusBadRequest, contracts.CodeValidationFailed, err.Error(), nil)
			return
		}
		deps.RecordAudit(r, "reports", "report-run", req.Source, "اجرای گزارش بر پایهٔ اقلام انتخابی")
		httpx.WriteJSON(w, http.StatusOK, result)
	}))
}

// runReport منبع دادهٔ خواسته‌شده را پیدا و گزارش را اجرا می‌کند و سرصفحهٔ برند شرکت
// فعال را روی نتیجه می‌نشاند.
func runReport(ctx context.Context, deps modulekit.Deps, req contracts.ReportRequest) (contracts.ReportResult, error) {
	source, ok := deps.Ports.ReportSource(req.Source)
	if !ok {
		// نبود منبع: به‌جای خطای مبهم، فهرست منابع موجود را برمی‌گردانیم.
		return contracts.ReportResult{}, &unknownSourceError{source: req.Source, available: availableSources(deps)}
	}
	result, err := source.Rows(ctx, req)
	if err != nil {
		return contracts.ReportResult{}, err
	}
	// اقلام انتخابی به همان ترتیب ستون‌ها؛ تا خروجی چاپی/ذخیره‌شده بتواند همین گزارش را
	// بازسازی کند، بدون آنکه کلاینت مجبور باشد انتخاب‌ها را جدا نگه دارد.
	items := make([]string, 0, len(result.Columns))
	for _, c := range result.Columns {
		items = append(items, c.Key)
	}
	result.Items = items
	result.Brand = brandOf(ctx, deps)
	result.Print = reportPrintChoice(deps, req.Print)
	return result, nil
}

// reportPrintChoice تصمیم «نمایش توضیحات در برگهٔ چاپی» را می‌سازد.
//
// اولویت: انتخاب صریح همین اجرا، سپس پارامتر شرکت `reports.print.showDescription`
// (پیش‌فرض: روشن). قاعده در سرور است تا رابط کاربری فقط نمایش دهد و رفتار در همهٔ
// خروجی‌ها (چاپ، PDF، خروجی سروری) یکی بماند.
func reportPrintChoice(deps modulekit.Deps, choice *contracts.ReportPrintChoice) *contracts.ReportPrint {
	show := true
	source := "پارامتر شرکت reports.print.showDescription"
	for _, p := range deps.Org.Params {
		if p.Key != "reports.print.showDescription" {
			continue
		}
		v := strings.TrimSpace(p.Value)
		show = !(v == "0" || strings.EqualFold(v, "no") || strings.EqualFold(v, "false"))
	}
	if choice != nil && choice.ShowDescription != nil {
		show = *choice.ShowDescription
		source = "انتخاب همین اجرا"
	}
	note := "بخش توضیحات در برگهٔ چاپی نمایش داده می‌شود."
	if !show {
		note = "بخش توضیحات در برگهٔ چاپی نمایش داده نمی‌شود."
	}
	return &contracts.ReportPrint{ShowDescription: show, SourceFa: source, NoteFa: note}
}

// brandOf سرصفحهٔ برند گزارش را از استقرار پارامتریک شرکت فعال می‌سازد.
// هیچ نام یا نشانی اینجا ساخته نمی‌شود؛ همه از پروفایل شرکت می‌آید، پس گزارش شرکت‌های
// مختلف با هویت خودشان چاپ می‌شود.
func brandOf(ctx context.Context, deps modulekit.Deps) *contracts.ReportBrand {
	brand := &contracts.ReportBrand{
		TitleFa:    deps.Org.TitleFa,
		HoldingFa:  deps.Org.HoldingFa,
		ShortCode:  deps.Org.ShortCode,
		LogoURL:    logoURL(deps.Org.LogoFile),
		BrandColor: deps.Org.BrandColor,
	}
	now := time.Now()
	if deps.Clock != nil {
		now = deps.Clock()
	}
	if d, err := calendar.FromGregorian(now); err == nil {
		brand.GeneratedAt = d.String()
	}
	if user, ok := auth.UserFrom(ctx); ok {
		brand.GeneratedBy = user.Username
	}
	return brand
}

// logoURL مسیر نشان شرکت را به آدرس قابل مصرف در خروجی گزارش تبدیل می‌کند. گزارش روی
// همان مبدأ رابط وب سرو می‌شود، پس مسیر نسبی پروفایل شرکت کافی است.
func logoURL(file string) string {
	file = strings.TrimSpace(file)
	if file == "" {
		return ""
	}
	return "/" + strings.TrimPrefix(file, "/")
}

func availableSources(deps modulekit.Deps) []string {
	out := []string{}
	for _, s := range deps.Ports.ReportSources() {
		out = append(out, s.Metadata().Key)
	}
	return out
}

type unknownSourceError struct {
	source    string
	available []string
}

func (e *unknownSourceError) Error() string {
	msg := "منبع دادهٔ گزارش «" + e.source + "» شناخته نشد."
	if len(e.available) > 0 {
		msg += " منابع موجود: "
		for i, k := range e.available {
			if i > 0 {
				msg += "، "
			}
			msg += k
		}
	}
	return msg
}
