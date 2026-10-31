// پیاده‌سازی پشتهٔ «bootstrap» — «public»: مسیرهای عمومی نشان شرکت، QR کد و تأیید سند.
//
// تفکیک دسترسی:
//   - **ساخت QR** با احراز هویت است (کاربر واردشده کد می‌گیرد).
//   - **سرو فایل نشان** و **صفحهٔ تأیید** عمومی‌اند؛ چون نشان روی صفحهٔ ورود دیده می‌شود و
//     صفحهٔ تأیید با اسکن QR روی سند چاپی باز می‌شود، جایی که هیچ توکنی در دست نیست.
//   - محتوای صفحهٔ تأیید کمینه است (نام، واحد، شرکت، سریال) و هیچ مبلغ یا دادهٔ پرسنلی
//     حساسی نشان داده نمی‌شود.

package bootstrap

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"strings"

	"hrms/internal/contracts"
	"hrms/internal/platform/auth"
	"hrms/internal/platform/branding"
	"hrms/internal/platform/httpx"
	"hrms/internal/platform/modulekit"
	"hrms/internal/platform/orgs"
	"hrms/internal/platform/qr"
)

// verifyTokenLen طول امضای نشانی تأیید.
const verifyTokenLen = 16

// registerPublicRoutes مسیرهای نشان، QR و تأیید را سوار می‌کند.
func registerPublicRoutes(mux *http.ServeMux, deps modulekit.Deps) {
	registerBrandingRoutes(mux, deps)
	registerQRRoutes(mux, deps)
	registerVerifyRoutes(mux, deps)
}

// registerBrandingRoutes سرو عمومی فایل نشان شرکت.
func registerBrandingRoutes(mux *http.ServeMux, deps modulekit.Deps) {
	store, err := branding.New(deps.DataDir)
	if err != nil {
		deps.Logger("bootstrap").Warn("انبار نشان شرکت آماده نشد", "error", err.Error())
		return
	}
	mux.HandleFunc("GET /api/v1/branding/{file}", func(w http.ResponseWriter, r *http.Request) {
		file := r.PathValue("file")
		handle, contentType, err := store.Open(file)
		if err != nil {
			httpx.WriteError(w, http.StatusNotFound, contracts.CodeNotFound, nil)
			return
		}
		defer handle.Close()
		w.Header().Set("Content-Type", contentType)
		// نشان شرکت کم‌تغییر است؛ کش کوتاه با امکان بازبینی، بارگذاری تازه را کند نمی‌کند.
		w.Header().Set("Cache-Control", "public, max-age=300")
		w.WriteHeader(http.StatusOK)
		_, _ = io.Copy(w, handle)
	})

	// نشان شرکت فعال: رابط وب و صفحهٔ ورود همین یک نشانی را می‌خوانند. اگر برای این شرکت
	// نشان بارگذاری‌شده باشد از آن استفاده می‌شود، وگرنه نشان فهرست استقرارها برمی‌گردد.
	mux.HandleFunc("GET /api/v1/branding/active", func(w http.ResponseWriter, r *http.Request) {
		url := activeLogoURL(r, deps)
		httpx.WriteJSON(w, http.StatusOK, map[string]any{
			"org": deps.Org.ID, "titleFa": deps.Org.TitleFa,
			"logoUrl": url, "brandColor": deps.Org.BrandColor,
			"hasUploadedLogo": strings.HasPrefix(url, "/api/v1/branding/"),
			"qrCompany":       "/api/v1/qr/company/" + deps.Org.ID,
		})
	})
}

// activeLogoURL نشانی نشان شرکت فعال را برمی‌گرداند.
func activeLogoURL(r *http.Request, deps modulekit.Deps) string {
	if port, ok := deps.Ports.RecordList("settings"); ok {
		if page, err := port.List(r.Context(), contracts.RecordFilter{
			Module: "settings", Kind: "companyProfiles", Size: 200, Scope: "all",
		}); err == nil {
			for _, rec := range page.Items {
				if strings.EqualFold(strings.TrimSpace(rec.Fields["code"]), deps.Org.ID) {
					if url := strings.TrimSpace(rec.Fields["logoFile"]); url != "" {
						return url
					}
				}
			}
		}
	}
	file := strings.TrimSpace(deps.Org.LogoFile)
	if file == "" {
		return ""
	}
	if strings.HasPrefix(file, "http") || strings.HasPrefix(file, "//") || strings.HasPrefix(file, "/") {
		return file
	}
	// مسیر نسبی پروفایل شرکت (مثل assets/brand/logo.jpg) روی همین مبدأ سرو می‌شود.
	return "/" + strings.TrimPrefix(file, "./")
}

// registerQRRoutes ساخت QR کد برای سه جای مصرف: شرکت، کارمند و سند چاپی.
func registerQRRoutes(mux *http.ServeMux, deps modulekit.Deps) {
	handler := func(scope string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			link := ""
			switch scope {
			case "company":
				code := strings.TrimSpace(r.PathValue("code"))
				if _, ok := orgs.Lookup(code); !ok {
					httpx.WriteErrorMsg(w, http.StatusNotFound, contracts.CodeNotFound, "شرکت با این کد در فهرست استقرارها نیست", nil)
					return
				}
				link = qr.VerifyLink(baseURL(r), "company", code, nil)
			case "employee":
				id := strings.TrimSpace(r.PathValue("id"))
				if _, ok := employeeBrief(r, deps, id); !ok {
					httpx.WriteErrorMsg(w, http.StatusNotFound, contracts.CodeNotFound, "کارمند با این شناسه پیدا نشد", nil)
					return
				}
				link = qr.VerifyLink(baseURL(r), "employee", id, nil)
			case "document":
				kind := strings.TrimSpace(r.PathValue("kind"))
				id := strings.TrimSpace(r.PathValue("id"))
				serial := strings.TrimSpace(r.URL.Query().Get("serial"))
				if serial == "" {
					serial = id
				}
				token := qr.Sign(deps.VerifyKey, "document", kind, id, serial)
				link = qr.VerifyLink(baseURL(r), "employee", id, map[string]string{
					"kind": kind, "serial": serial, "t": token,
				})
			}
			writeQR(w, r, link)
		}
	}
	// QR شناسهٔ شرکت عمومی است: محتوایش فقط نشانی صفحهٔ تأیید شرکت است و همان صفحه هم
	// عمومی است؛ پس تصویرش را می‌توان بدون توکن در سرصفحهٔ فرم‌ها و مکاتبات نشان داد
	// (برچسب `img` نمی‌تواند هدر احراز هویت بفرستد).
	mux.HandleFunc("GET /api/v1/qr/company/{code}", handler("company"))
	// QR کارمند و سند نیازمند ورود است (به شناسهٔ کارمند و سریال سند اشاره می‌کند).
	mux.HandleFunc("GET /api/v1/qr/employee/{id}", auth.RequireAuth(handler("employee")))
	mux.HandleFunc("GET /api/v1/qr/document/{kind}/{id}", auth.RequireAuth(handler("document")))
}

// writeQR خروجی QR را به شکل SVG (پیش‌فرض) یا PNG می‌نویسد.
func writeQR(w http.ResponseWriter, r *http.Request, link string) {
	size := 240
	if raw := strings.TrimSpace(r.URL.Query().Get("size")); raw != "" {
		if v, err := parseInt(raw); err == nil {
			size = v
		}
	}
	switch strings.ToLower(strings.TrimSpace(r.URL.Query().Get("format"))) {
	case "png":
		data, err := qr.PNG(link, size)
		if err != nil {
			httpx.WriteErrorMsg(w, http.StatusInternalServerError, contracts.CodeInternal, err.Error(), nil)
			return
		}
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
	case "json":
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"link": link, "size": size})
	default:
		svg, err := qr.SVG(link, size)
		if err != nil {
			httpx.WriteErrorMsg(w, http.StatusInternalServerError, contracts.CodeInternal, err.Error(), nil)
			return
		}
		w.Header().Set("Content-Type", "image/svg+xml; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, svg)
	}
}

// registerVerifyRoutes صفحهٔ تأیید عمومی (JSON و HTML) برای شرکت و کارمند/سند.
func registerVerifyRoutes(mux *http.ServeMux, deps modulekit.Deps) {
	payload := func(r *http.Request, scope, id string) (map[string]any, int, string) {
		switch scope {
		case "company":
			org, ok := orgs.Lookup(strings.TrimSpace(id))
			if !ok {
				return nil, http.StatusNotFound, "شرکت با این کد در فهرست استقرارها نیست"
			}
			return map[string]any{
				"scope": "company", "verified": true, "code": org.ID,
				"titleFa": org.TitleFa, "holdingFa": org.HoldingFa,
				"shortCode": org.ShortCode, "headcount": org.Headcount,
				"noteFa": "این شناسهٔ شرکت در سامانهٔ منابع انسانی ثبت است.",
			}, http.StatusOK, ""
		case "employee":
			brief, ok := employeeBrief(r, deps, strings.TrimSpace(id))
			if !ok {
				return nil, http.StatusNotFound, "کارمند با این شناسه پیدا نشد"
			}
			out := map[string]any{
				"scope": "employee", "verified": true,
				"employeeId": brief.ID, "fullName": brief.FullName,
				"departmentName": brief.DepartmentName, "position": brief.Position,
				"status": brief.Status, "organization": deps.Org.TitleFa,
				"noteFa": "این کارمند در سامانهٔ منابع انسانی این شرکت ثبت است.",
			}
			// تأیید سند چاپی: نوع سند و سریال باید با امضای سرور بیاید تا دستکاری نشود.
			kind := strings.TrimSpace(r.URL.Query().Get("kind"))
			serial := strings.TrimSpace(r.URL.Query().Get("serial"))
			token := strings.TrimSpace(r.URL.Query().Get("t"))
			if kind != "" && serial != "" {
				if !qr.Check(deps.VerifyKey, token, "document", kind, brief.ID, serial) {
					out["documentVerified"] = false
					out["noteFa"] = "امضای این نشانی معتبر نیست؛ سند را از راه سامانه بازبینی کنید."
					return out, http.StatusOK, ""
				}
				out["documentVerified"] = true
				out["documentKind"] = kind
				out["serial"] = serial
				out["noteFa"] = "این سند در سامانهٔ منابع انسانی این شرکت صادر شده است."
			}
			return out, http.StatusOK, ""
		}
		return nil, http.StatusNotFound, "نوع تأیید ناشناخته است"
	}

	// خروجی JSON برای سامانه‌های دیگر و آزمون خودکار.
	jsonHandler := func(w http.ResponseWriter, r *http.Request) {
		body, code, message := payload(r, r.PathValue("scope"), r.PathValue("id"))
		if body == nil {
			httpx.WriteErrorMsg(w, code, contracts.CodeNotFound, message, nil)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, body)
	}
	mux.HandleFunc("GET /api/v1/verify/{scope}/{id}", jsonHandler)

	// صفحهٔ فارسی خوانا؛ همان چیزی که با اسکن QR روی تلفن باز می‌شود.
	mux.HandleFunc("GET /verify/{scope}/{id}", func(w http.ResponseWriter, r *http.Request) {
		body, status, message := payload(r, r.PathValue("scope"), r.PathValue("id"))
		if body == nil {
			writeVerifyPage(w, http.StatusNotFound, map[string]any{
				"ok": false, "titleFa": "تأیید نشد", "messageFa": message,
			})
			_ = status
			return
		}
		raw, _ := json.Marshal(body)
		var view map[string]any
		_ = json.Unmarshal(raw, &view)
		verified := view["verified"] == true
		if value, ok := view["documentVerified"]; ok && value != true {
			verified = false
		}
		view["ok"] = verified
		// نام شرکت با عنوان صفحه اشتباه نشود: در پاسخ تأیید شرکت، کلید «titleFa» خودِ
		// نام شرکت است؛ پس عنوان صفحه دست‌نخورده می‌ماند و در قالب کلید جدا دارد.
		if !verified {
			view["messageFa"] = view["noteFa"]
		}
		writeVerifyPage(w, http.StatusOK, view)
	})
}

// employeeBrief اطلاعات کمینهٔ یک کارمند برای تأیید (بدون مبلغ و دادهٔ حساس).
func employeeBrief(r *http.Request, deps modulekit.Deps, id string) (contracts.Employee, bool) {
	reader, ok := deps.Ports.Employees()
	if !ok {
		return contracts.Employee{}, false
	}
	employee, found, err := reader.Get(r.Context(), contracts.EmployeeFilter{Scope: "all"}, strings.TrimSpace(id))
	if err != nil || !found {
		return contracts.Employee{}, false
	}
	return employee, true
}

// verifyTemplate قالب صفحهٔ تأیید (راست‌چین، بدون وابستگی بیرونی).
var verifyTemplate = template.Must(template.New("verify").Parse(`<!doctype html>
<html lang="fa" dir="rtl">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="robots" content="noindex">
<title>{{.pageTitleFa}}</title>
<style>
  body { font-family: "Vazirmatn", "IRANSans", Tahoma, sans-serif; background: #f8fafc; color: #111827;
         margin: 0; padding: 24px; display: flex; justify-content: center; }
  .card { background: #fff; max-width: 520px; width: 100%; border-radius: 14px; padding: 24px;
          box-shadow: 0 6px 24px rgba(15,23,42,.08); }
  .badge { display: inline-block; padding: 4px 12px; border-radius: 999px; font-size: 13px; font-weight: 700; }
  .ok { background: #dcfce7; color: #166534; }
  .bad { background: #fee2e2; color: #991b1b; }
  h1 { font-size: 18px; margin: 14px 0 6px; }
  p { line-height: 2; color: #334155; font-size: 14px; margin: 6px 0; }
  table { width: 100%; border-collapse: collapse; margin-top: 14px; font-size: 13px; }
  th, td { border: 1px solid #e2e8f0; padding: 8px 10px; text-align: right; }
  th { background: #f1f5f9; width: 38%; }
  footer { margin-top: 18px; font-size: 11px; color: #64748b; text-align: center; }
</style>
</head>
<body>
<div class="card">
  {{if .ok}}<span class="badge ok">✓ تأییدشده</span>{{else}}<span class="badge bad">✖ تأیید نشد</span>{{end}}
  <h1>{{.pageTitleFa}}</h1>
  <p>{{.messageFa}}</p>
  <table>
    {{if .code}}<tr><th>کد شرکت</th><td>{{.code}}</td></tr>{{end}}
    {{if .companyTitleFa}}<tr><th>نام شرکت</th><td>{{.companyTitleFa}}</td></tr>{{end}}
    {{if .holdingFa}}<tr><th>هلدینگ</th><td>{{.holdingFa}}</td></tr>{{end}}
    {{if .fullName}}<tr><th>نام کارمند</th><td>{{.fullName}}</td></tr>{{end}}
    {{if .employeeId}}<tr><th>کد پرسنلی</th><td>{{.employeeId}}</td></tr>{{end}}
    {{if .departmentName}}<tr><th>واحد سازمانی</th><td>{{.departmentName}}</td></tr>{{end}}
    {{if .position}}<tr><th>عنوان شغل</th><td>{{.position}}</td></tr>{{end}}
    {{if .status}}<tr><th>وضعیت اشتغال</th><td>{{.status}}</td></tr>{{end}}
    {{if .serial}}<tr><th>سریال سند</th><td>{{.serial}}</td></tr>{{end}}
    {{if .organization}}<tr><th>سازمان</th><td>{{.organization}}</td></tr>{{end}}
  </table>
  <footer>سامانهٔ جامع منابع انسانی — این صفحه فقط اطلاعات تأیید را نشان می‌دهد، نه مبالغ یا داده‌های پرسنلی.</footer>
</div>
</body>
</html>
`))

// writeVerifyPage صفحهٔ تأیید را می‌نویسد.
//
// تفکیک کلیدها مهم است: «pageTitleFa» عنوان خود صفحه است و «companyTitleFa» نام شرکتی
// که در پاسخ تأیید آمده؛ در پاسخ شرکت، کلید «titleFa» همان نام شرکت است.
func writeVerifyPage(w http.ResponseWriter, status int, data map[string]any) {
	view := map[string]any{
		"pageTitleFa":    "تأیید سامانهٔ منابع انسانی",
		"messageFa":      dataString(data, "messageFa"),
		"ok":             data["ok"] == true,
		"code":           dataString(data, "code"),
		"companyTitleFa": firstNonEmpty(dataString(data, "companyTitleFa"), dataString(data, "titleFa")),
		"holdingFa":      dataString(data, "holdingFa"),
		"fullName":       dataString(data, "fullName"),
		"employeeId":     dataString(data, "employeeId"),
		"departmentName": dataString(data, "departmentName"),
		"position":       dataString(data, "position"),
		"status":         dataString(data, "status"),
		"serial":         dataString(data, "serial"),
		"organization":   dataString(data, "organization"),
	}
	if view["messageFa"] == "" {
		view["messageFa"] = dataString(data, "noteFa")
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Robots-Tag", "noindex")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = verifyTemplate.Execute(w, view)
}

// baseURL نشانی پایهٔ سرور از خود درخواست (پشت پروکسی هم درست کار می‌کند).
func baseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")); forwarded != "" {
		scheme = strings.Split(forwarded, ",")[0]
	}
	host := strings.TrimSpace(r.Host)
	if host == "" {
		host = "127.0.0.1"
	}
	return fmt.Sprintf("%s://%s", scheme, host)
}

// parseInt عدد صحیح ساده (بدون خطای پیچیده).
func parseInt(raw string) (int, error) {
	value := 0
	for _, ch := range strings.TrimSpace(raw) {
		if ch < '0' || ch > '9' {
			return 0, fmt.Errorf("عدد نامعتبر")
		}
		value = value*10 + int(ch-'0')
	}
	return value, nil
}

// dataString خواندن رشتهٔ امن از نقشهٔ پاسخ.
func dataString(data map[string]any, key string) string {
	if value, ok := data[key].(string); ok {
		return value
	}
	return ""
}
