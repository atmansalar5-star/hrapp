// پیاده‌سازی ماژول «settings» — «logos»: بارگذاری، خواندن و حذف نشان (لوگوی) شرکت‌ها.
//
// قاعده‌ها:
//   - فایل نشان در پوشهٔ دادهٔ سرور ذخیره می‌شود (`<data>/branding/`) و با مسیر عمومی
//     `/api/v1/branding/<file>` سرو می‌شود؛ پس روی صفحهٔ ورود (پیش از احراز هویت) هم دیده
//     می‌شود و به باینری برنامه وابسته نیست.
//   - سنجش نوع فایل با امضای دودویی و پاک‌سازی نام در بستهٔ `platform/branding` است.
//   - نشانی نشان روی خودِ رکورد شرکت (`logoFile`) می‌نشیند تا هر استقرار نشان خودش را
//     داشته باشد و سندهای چاپی همان را بخوانند.

package settings

import (
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"hrms/internal/contracts"
	"hrms/internal/platform/auth"
	"hrms/internal/platform/branding"
	"hrms/internal/platform/calendar"
	"hrms/internal/platform/httpx"
	"hrms/internal/platform/modulekit"
	"hrms/internal/platform/recordkit"
)

// registerLogoEndpoints مسیرهای نشان شرکت را ثبت می‌کند.
func registerLogoEndpoints(mux *http.ServeMux, deps modulekit.Deps, svc *recordkit.Service) {
	read := func(fn http.HandlerFunc) http.HandlerFunc { return deps.Require("settings", "read", fn) }
	write := func(fn http.HandlerFunc) http.HandlerFunc { return deps.Require("settings", "write", fn) }

	// وضعیت نشان فعلی شرکت (برای فرم و برای سرصفحهٔ رابط).
	mux.HandleFunc("GET /api/v1/settings/companies/{id}/logo", read(func(w http.ResponseWriter, r *http.Request) {
		rec, ok := companyRecord(r, svc)
		if !ok {
			httpx.WriteErrorMsg(w, http.StatusNotFound, contracts.CodeNotFound, "شرکت با این شناسه پیدا نشد", nil)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, logoState(rec))
	}))

	// بارگذاری نشان تازه (multipart با فیلد «file»).
	mux.HandleFunc("POST /api/v1/settings/companies/{id}/logo", write(func(w http.ResponseWriter, r *http.Request) {
		rec, ok := companyRecord(r, svc)
		if !ok {
			httpx.WriteErrorMsg(w, http.StatusNotFound, contracts.CodeNotFound, "شرکت با این شناسه پیدا نشد", nil)
			return
		}
		store, err := branding.New(deps.DataDir)
		if err != nil {
			httpx.WriteErrorMsg(w, http.StatusConflict, contracts.CodeConflict, err.Error(), nil)
			return
		}
		if err := r.ParseMultipartForm(branding.MaxBytes + (1 << 20)); err != nil {
			httpx.WriteErrorMsg(w, http.StatusBadRequest, contracts.CodeBadRequest,
				"بدنهٔ درخواست باید multipart/form-data با فیلد «file» باشد.", nil)
			return
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			httpx.WriteErrorMsg(w, http.StatusBadRequest, contracts.CodeBadRequest,
				"فیلد فایل («file») در درخواست نبود.", nil)
			return
		}
		defer file.Close()
		data, err := io.ReadAll(io.LimitReader(file, branding.MaxBytes+1))
		if err != nil {
			httpx.WriteErrorMsg(w, http.StatusBadRequest, contracts.CodeBadRequest, "خواندن فایل نشان ممکن نشد.", nil)
			return
		}
		name := ""
		if header != nil {
			name = header.Filename
		}
		code := strings.TrimSpace(rec.Fields["code"])
		if code == "" {
			code = rec.ID
		}
		saved, err := store.Save(code, name, data)
		if err != nil {
			httpx.WriteErrorMsg(w, http.StatusUnprocessableEntity, contracts.CodeConflict, err.Error(), nil)
			return
		}

		// نشان پیشین پاک می‌شود تا فایل‌های بی‌استفاده در پوشهٔ داده نمانند.
		if previous := branding.NameFromURL(rec.Fields["logoFile"]); previous != "" && previous != saved.Name {
			_ = store.Remove(previous)
		}

		updated, err := saveCompanyFields(r, svc, rec, map[string]string{
			"logoFile":      saved.URL,
			"logoHash":      saved.SHA256,
			"logoSizeKb":    strconv.Itoa((saved.SizeBytes + 1023) / 1024),
			"logoUpdatedAt": todayFa(deps),
		}, "نشان شرکت بارگذاری شد")
		if err != nil {
			httpx.WriteErrorMsg(w, http.StatusUnprocessableEntity, contracts.CodeConflict, err.Error(), nil)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, map[string]any{
			"company": updated, "logo": saved,
			"noteFa": "نشان شرکت ذخیره شد و از همین حالا در رابط، صفحهٔ ورود و سندهای چاپی همین شرکت دیده می‌شود.",
		})
	}))

	// حذف نشان شرکت.
	mux.HandleFunc("DELETE /api/v1/settings/companies/{id}/logo", write(func(w http.ResponseWriter, r *http.Request) {
		rec, ok := companyRecord(r, svc)
		if !ok {
			httpx.WriteErrorMsg(w, http.StatusNotFound, contracts.CodeNotFound, "شرکت با این شناسه پیدا نشد", nil)
			return
		}
		if store, err := branding.New(deps.DataDir); err == nil {
			if name := branding.NameFromURL(rec.Fields["logoFile"]); name != "" {
				if err := store.Remove(name); err != nil {
					httpx.WriteErrorMsg(w, http.StatusInternalServerError, contracts.CodeInternal, err.Error(), nil)
					return
				}
			}
		}
		updated, err := saveCompanyFields(r, svc, rec, map[string]string{
			"logoFile": "", "logoHash": "", "logoSizeKb": "", "logoUpdatedAt": "",
		}, "نشان شرکت حذف شد")
		if err != nil {
			httpx.WriteErrorMsg(w, http.StatusUnprocessableEntity, contracts.CodeConflict, err.Error(), nil)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, map[string]any{
			"company": updated, "noteFa": "نشان شرکت حذف شد؛ سندها بدون تصویر نشان چاپ می‌شوند.",
		})
	}))
}

// companyRecord رکورد «شرکت/استقرار» را از شناسه می‌خواند.
func companyRecord(r *http.Request, svc *recordkit.Service) (contracts.Record, bool) {
	if svc == nil {
		return contracts.Record{}, false
	}
	rec, ok, err := svc.Get(r.Context(), contracts.RecordFilter{
		Module: "settings", Kind: "companyProfiles", Scope: "all",
	}, r.PathValue("id"))
	if err != nil || !ok || rec.Kind != "companyProfiles" {
		return contracts.Record{}, false
	}
	return rec, true
}

// saveCompanyFields فیلدهای داده‌شده را روی رکورد شرکت می‌نویسد.
//
// نکته: ویرایش رکورد، کل مجموعهٔ فیلدها را جایگزین می‌کند؛ پس فیلدهای فعلی خوانده و
// همان‌ها با تغییرهای خواسته‌شده فرستاده می‌شوند تا بقیهٔ اطلاعات شرکت از دست نرود.
func saveCompanyFields(r *http.Request, svc *recordkit.Service, rec contracts.Record, changes map[string]string, noteFa string) (contracts.Record, error) {
	fields := map[string]string{}
	for key, value := range rec.Fields {
		fields[key] = value
	}
	for key, value := range changes {
		fields[key] = value
	}
	actor := actorNameOf(r)
	return svc.Update(r.Context(), rec.ID, contracts.RecordInput{
		Kind: "companyProfiles", Title: rec.Title, Fields: fields,
		NoteFa: noteFa,
	}, actor)
}

// actorNameOf نام کاربر جاری برای ثبت رخداد ویرایش.
func actorNameOf(r *http.Request) string {
	if user, ok := auth.UserFrom(r.Context()); ok && user.Username != "" {
		return user.Username
	}
	return "settings"
}

// logoState وضعیت نشان یک شرکت را برای رابط می‌سازد.
func logoState(rec contracts.Record) map[string]any {
	url := strings.TrimSpace(rec.Fields["logoFile"])
	state := map[string]any{
		"companyId": rec.ID,
		"code":      rec.Fields["code"],
		"titleFa":   rec.Title,
		"url":       url,
		"hasLogo":   url != "",
		"hash":      rec.Fields["logoHash"],
		"updatedAt": rec.Fields["logoUpdatedAt"],
		"sizeKb":    rec.Fields["logoSizeKb"],
		"qrCompany": "/api/v1/qr/company/" + rec.Fields["code"],
	}
	if url == "" {
		state["noteFa"] = "برای این شرکت نشانی ثبت نشده است؛ می‌توانید نشان رسمی را بارگذاری کنید."
	} else {
		state["noteFa"] = "نشان ثبت‌شدهٔ این شرکت."
	}
	return state
}

// todayFa تاریخ جاری سرور به تقویم شمسی.
func todayFa(deps modulekit.Deps) string {
	now := time.Now()
	if deps.Clock != nil {
		now = deps.Clock()
	}
	today, err := calendar.FromGregorian(now)
	if err != nil {
		return ""
	}
	return today.String()
}
