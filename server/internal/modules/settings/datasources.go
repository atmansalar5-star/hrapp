// پیاده‌سازی ماژول «settings» — «datasources»: رابط اتصال به منابع داده بیرونی،
// ساختار دادهٔ آماده‌سازی و قالب‌های بارگذاری.
//
// قاعدهٔ معماری: درایورها در `platform/datasource` هستند و این فایل فقط پایانهٔ HTTP
// می‌سازد (سنجش ورودی، دسترسی، پاسخ فارسی). ساختار داده هم از شناسهٔ خودِ ماژول‌ها ساخته
// می‌شود، پس هیچ فهرست جدولی در کد تکرار نمی‌شود.

package settings

import (
	"encoding/json"
	"io"
	"net/http"
	"sort"
	"strings"

	"hrms/internal/contracts"
	"hrms/internal/platform/datasource"
	"hrms/internal/platform/httpx"
	"hrms/internal/platform/modulekit"
	"hrms/internal/platform/recordkit"
)

// registerDataSourceEndpoints مسیرهای تبادل داده را ثبت می‌کند.
func registerDataSourceEndpoints(mux *http.ServeMux, deps modulekit.Deps) {
	read := func(fn http.HandlerFunc) http.HandlerFunc { return deps.Require("settings", "read", fn) }
	write := func(fn http.HandlerFunc) http.HandlerFunc { return deps.Require("settings", "write", fn) }

	// فهرست انواع منبع داده (اکسل، CSV، Postgres، SQL Server، MySQL، Oracle) با فیلدهای هرکدام.
	mux.HandleFunc("GET /api/v1/settings/data-sources/kinds", read(func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, map[string]any{
			"kinds": datasource.Catalog(),
			"noteFa": "هر نوع منبع، فیلدهای پیکربندی خودش را دارد؛ فیلدهای رمز در رکورد ذخیره نمی‌شوند." +
				" برای فایل‌های اکسس (mdb/accdb) از خروجی CSV یا Excel همان سامانه استفاده کنید،" +
				" چون اتصال مستقیم نیازمند درایور ODBC روی سرور است.",
		})
	}))

	// آزمون اتصال.
	mux.HandleFunc("POST /api/v1/settings/data-sources/test", write(func(w http.ResponseWriter, r *http.Request) {
		cfg, ok := decodeSourceConfig(w, r)
		if !ok {
			return
		}
		source, ok := datasource.Get(cfg.Kind)
		if !ok {
			httpx.WriteErrorMsg(w, http.StatusUnprocessableEntity, contracts.CodeConflict,
				"نوع منبع داده پشتیبانی نمی‌شود.", nil)
			return
		}
		if err := datasource.Validate(cfg); err != nil {
			httpx.WriteErrorMsg(w, http.StatusUnprocessableEntity, contracts.CodeConflict, err.Error(), nil)
			return
		}
		ctx, cancel := datasource.WithTimeout(r.Context(), cfg)
		defer cancel()
		report, err := source.Test(ctx, cfg)
		if err != nil {
			httpx.WriteJSON(w, http.StatusOK, map[string]any{
				"ok": false, "messageFa": err.Error(), "kind": cfg.Kind,
			})
			return
		}
		deps.Audit.Append(actorNameOf(r), "settings", "data-source-test", string(cfg.Kind),
			"آزمون اتصال منبع داده: "+strings.TrimSpace(cfg.Name))
		httpx.WriteJSON(w, http.StatusOK, report)
	}))

	// فهرست جدول‌ها/برگه‌های منبع.
	mux.HandleFunc("POST /api/v1/settings/data-sources/tables", write(func(w http.ResponseWriter, r *http.Request) {
		cfg, ok := decodeSourceConfig(w, r)
		if !ok {
			return
		}
		source, ok := datasource.Get(cfg.Kind)
		if !ok {
			httpx.WriteErrorMsg(w, http.StatusUnprocessableEntity, contracts.CodeConflict, "نوع منبع داده پشتیبانی نمی‌شود.", nil)
			return
		}
		ctx, cancel := datasource.WithTimeout(r.Context(), cfg)
		defer cancel()
		tables, err := source.Tables(ctx, cfg)
		if err != nil {
			httpx.WriteErrorMsg(w, http.StatusConflict, contracts.CodeConflict, err.Error(), nil)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"tables": tables, "total": len(tables)})
	}))

	// پیش‌نمایش ردیف‌ها (پیش از هر بارگذاری).
	mux.HandleFunc("POST /api/v1/settings/data-sources/preview", write(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Config   datasource.Config `json:"config"`
			Table    string            `json:"table"`
			Query    string            `json:"query"`
			RowLimit int               `json:"rowLimit"`
		}
		if !decodeBody(w, r, &body) {
			return
		}
		source, ok := datasource.Get(body.Config.Kind)
		if !ok {
			httpx.WriteErrorMsg(w, http.StatusUnprocessableEntity, contracts.CodeConflict, "نوع منبع داده پشتیبانی نمی‌شود.", nil)
			return
		}
		// سنجش «فقط‌خواندنی» پیش از رسیدن به درایور انجام می‌گیرد تا برای همهٔ منابع
		// (فایلی و پایگاه‌داده) یکسان باشد؛ درایور فایلی هم پرس‌وجوی دستی نمی‌پذیرد.
		if strings.TrimSpace(body.Query) != "" {
			if err := datasource.IsReadOnlyStatement(body.Query); err != nil {
				httpx.WriteErrorMsg(w, http.StatusConflict, contracts.CodeConflict, err.Error(), nil)
				return
			}
		}
		ctx, cancel := datasource.WithTimeout(r.Context(), body.Config)
		defer cancel()
		var (
			result datasource.Result
			err    error
		)
		if strings.TrimSpace(body.Query) != "" {
			result, err = source.Query(ctx, body.Config, body.Query, body.RowLimit)
		} else {
			result, err = source.Preview(ctx, body.Config, body.Table, body.RowLimit)
		}
		if err != nil {
			httpx.WriteErrorMsg(w, http.StatusConflict, contracts.CodeConflict, err.Error(), nil)
			return
		}
		// یک‌کاسه‌سازی خروجی: شمار ردیف و نشانهٔ «بریده‌شدن» همیشه در پاسخ باشد.
		result = datasource.LimitResult(result, body.RowLimit)
		httpx.WriteJSON(w, http.StatusOK, result)
	}))

	// بارگذاری فایل (CSV/Excel) در پوشهٔ دادهٔ سرور تا در پیش‌نمایش و بارگذاری مصرف شود.
	mux.HandleFunc("POST /api/v1/settings/data-sources/upload", write(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(datasource.MaxUploadBytes + (1 << 20)); err != nil {
			httpx.WriteErrorMsg(w, http.StatusBadRequest, contracts.CodeBadRequest,
				"بدنهٔ درخواست باید multipart/form-data با فیلد «file» باشد.", nil)
			return
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			httpx.WriteErrorMsg(w, http.StatusBadRequest, contracts.CodeBadRequest, "فیلد فایل («file») در درخواست نبود.", nil)
			return
		}
		defer file.Close()
		data, err := io.ReadAll(io.LimitReader(file, datasource.MaxUploadBytes+1))
		if err != nil {
			httpx.WriteErrorMsg(w, http.StatusBadRequest, contracts.CodeBadRequest, "خواندن فایل ممکن نشد.", nil)
			return
		}
		name := ""
		if header != nil {
			name = header.Filename
		}
		upload, err := datasource.SaveUpload(deps.DataDir, name, data)
		if err != nil {
			httpx.WriteErrorMsg(w, http.StatusUnprocessableEntity, contracts.CodeConflict, err.Error(), nil)
			return
		}
		deps.Audit.Append(actorNameOf(r), "settings", "data-source-upload", upload.Name, "بارگذاری فایل داده")
		httpx.WriteJSON(w, http.StatusOK, map[string]any{
			"upload": upload,
			"noteFa": "فایل ذخیره شد؛ برای پیش‌نمایش، همین مسیر را در پیکربندی منبع فایلی بگذارید.",
		})
	}))

	// ساختار داده و راهنما: همهٔ جدول‌های هدف با ستون‌ها، قالب‌ها و مقادیر مجاز.
	mux.HandleFunc("GET /api/v1/settings/data-sources/manifest", read(func(w http.ResponseWriter, r *http.Request) {
		entities := manifestEntities(deps, r)
		httpx.WriteJSON(w, http.StatusOK, map[string]any{
			"entities": entities,
			"total":    len(entities),
			"noteFa": "این ساختار از شناسهٔ خودِ ماژول‌های فعال ساخته شده است؛ با افزودن ماژول یا فیلد تازه،" +
				" خودبه‌خود تازه می‌شود. ترتیب فهرست، ترتیب توصیه‌شدهٔ بارگذاری است.",
		})
	}))

	// راهنمای فارسی آماده‌سازی داده (متن قابل مطالعه و ذخیره).
	mux.HandleFunc("GET /api/v1/settings/data-sources/guide", read(func(w http.ResponseWriter, r *http.Request) {
		entities := manifestEntities(deps, r)
		guide := datasource.GuideFa(entities, deps.Org.TitleFa)
		if strings.EqualFold(r.URL.Query().Get("format"), "json") {
			httpx.WriteJSON(w, http.StatusOK, map[string]any{"guideFa": guide})
			return
		}
		w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, guide)
	}))

	// قالب ورود اطلاعات دستی: الگو، مقادیر مجاز و یک نمونه برای هر ستون یک جدول/فرم.
	//
	// چرا جدا از قالب اکسل: قالب اکسل فایل ساخته می‌شود تا پر و بارگذاری شود، ولی کارشناس
	// باید پیش از ورود داده ببیند هر ستون را «چه شکلی» بنویسد. این مسیر همان ساختار را
	// به‌صورت برگهٔ قابل نمایش و چاپ برمی‌گرداند؛ منبع هر دو یکی است.
	mux.HandleFunc("GET /api/v1/settings/data-sources/entry-format", read(func(w http.ResponseWriter, r *http.Request) {
		entities := manifestEntities(deps, r)
		key := strings.TrimSpace(r.URL.Query().Get("entity"))
		if key == "" {
			// بدون شناسه: فهرست جدول‌های موجود برای انتخاب در فرم.
			list := make([]map[string]any, 0, len(entities))
			for _, entity := range entities {
				list = append(list, map[string]any{
					"key": entity.Key, "titleFa": entity.TitleFa, "moduleFa": entity.ModuleFa,
					"groupFa": entity.GroupFa, "fields": len(entity.Fields),
				})
			}
			httpx.WriteJSON(w, http.StatusOK, map[string]any{"entities": list, "total": len(list)})
			return
		}
		entity, ok := datasource.FindEntity(entities, key)
		if !ok {
			httpx.WriteErrorMsg(w, http.StatusNotFound, contracts.CodeNotFound,
				"جدول درخواستی در ساختار داده پیدا نشد؛ فهرست را از همین مسیر بدون پارامتر بگیرید.", nil)
			return
		}
		sheet := datasource.EntrySheetOf(entity)
		httpx.WriteJSON(w, http.StatusOK, map[string]any{
			"sheet": sheet,
			"noteFa": "هر ستون با الگوی ورود و یک مقدار نمونه آمده است؛ سطر نمونه همان ترتیب ستون‌هاست. " +
				"داده‌های الزامی علامتدارند و رکورد بی‌آن‌ها ذخیره نمی‌شود.",
		})
	}))

	// قالب اکسل: یک کارپوشه با برگهٔ «راهنما» و یک برگه برای هر جدول.
	mux.HandleFunc("GET /api/v1/settings/data-sources/template.xlsx", read(func(w http.ResponseWriter, r *http.Request) {
		entities := datasource.SelectEntities(manifestEntities(deps, r), splitQueryList(r.URL.Query().Get("entities")))
		if len(entities) == 0 {
			httpx.WriteErrorMsg(w, http.StatusNotFound, contracts.CodeNotFound, "جدولی برای ساخت قالب پیدا نشد.", nil)
			return
		}
		content, err := datasource.TemplateXLSX(entities, deps.Org.TitleFa)
		if err != nil {
			httpx.WriteErrorMsg(w, http.StatusInternalServerError, contracts.CodeInternal, err.Error(), nil)
			return
		}
		deps.Audit.Append(actorNameOf(r), "settings", "data-template-xlsx", r.URL.Query().Get("entities"),
			"دریافت قالب اکسل بارگذاری")
		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		w.Header().Set("Content-Disposition", `attachment; filename="hrms-data-template.xlsx"`)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(content)
	}))

	// قالب CSV یک جدول (برای بارگذاری سبک یا سامانه‌های قدیمی).
	mux.HandleFunc("GET /api/v1/settings/data-sources/template.csv", read(func(w http.ResponseWriter, r *http.Request) {
		entities := manifestEntities(deps, r)
		entity, ok := datasource.FindEntity(entities, r.URL.Query().Get("entity"))
		if !ok {
			httpx.WriteErrorMsg(w, http.StatusNotFound, contracts.CodeNotFound,
				"جدول درخواستی در ساختار داده پیدا نشد؛ فهرست را از مسیر manifest بگیرید.", nil)
			return
		}
		content, err := datasource.TemplateCSV(entity)
		if err != nil {
			httpx.WriteErrorMsg(w, http.StatusInternalServerError, contracts.CodeInternal, err.Error(), nil)
			return
		}
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="hrms-`+entity.Key+`.csv"`)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(content)
	}))
}

// manifestEntities ساختار داده را از شناسهٔ همهٔ ماژول‌های فعال می‌سازد.
//
// قاعده: هر ماژول رکوردی، شناسهٔ نوع‌های رکوردش را روی پورت قراردادی منتشر می‌کند؛ پس
// فهرست جدول‌ها و ستون‌ها «تولیدشده» است و با تغییر سامانه خودکار تازه می‌شود.
func manifestEntities(deps modulekit.Deps, r *http.Request) []datasource.Entity {
	schemas := []contracts.RecordSchema{}
	titles := map[string]string{}
	if deps.Registry != nil {
		for _, module := range deps.Registry.All() {
			titles[module.ID] = module.TitleFa
			port, ok := deps.Ports.RecordList(module.ID)
			if !ok {
				continue
			}
			schemas = append(schemas, port.Schema())
		}
	}
	sort.SliceStable(schemas, func(i, j int) bool {
		return titles[schemas[i].Module] < titles[schemas[j].Module]
	})
	entities := datasource.Manifest(schemas, titles)
	if key := strings.TrimSpace(r.URL.Query().Get("entity")); key != "" {
		if entity, ok := datasource.FindEntity(entities, key); ok {
			return []datasource.Entity{entity}
		}
		return nil
	}
	return entities
}

// decodeSourceConfig پیکربندی منبع را از بدنهٔ درخواست می‌خواند.
func decodeSourceConfig(w http.ResponseWriter, r *http.Request) (datasource.Config, bool) {
	var body struct {
		Config datasource.Config `json:"config"`
	}
	if !decodeBody(w, r, &body) {
		return datasource.Config{}, false
	}
	return body.Config, true
}

// decodeBody بدنهٔ JSON را می‌خواند و در خطا پیام فارسی می‌دهد.
func decodeBody(w http.ResponseWriter, r *http.Request, target any) bool {
	raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		httpx.WriteErrorMsg(w, http.StatusBadRequest, contracts.CodeBadRequest, "خواندن بدنهٔ درخواست ممکن نشد.", nil)
		return false
	}
	if err := json.Unmarshal(raw, target); err != nil {
		httpx.WriteErrorMsg(w, http.StatusBadRequest, contracts.CodeBadRequest, "ساختار JSON درخواست معتبر نیست.", nil)
		return false
	}
	return true
}

// splitQueryList رشتهٔ جداشده با ویرگول را می‌شکند.
func splitQueryList(raw string) []string {
	out := []string{}
	for _, chunk := range strings.Split(raw, ",") {
		if value := strings.TrimSpace(chunk); value != "" {
			out = append(out, value)
		}
	}
	return out
}

// keepRecordkitImport جلوگیری از حذف ناخواستهٔ import در ویرایش‌های بعدی (امضای سرویس).
var _ = recordkit.Service{}
