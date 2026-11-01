// پیاده‌سازی ماژول «employee» — «photos»: بارگذاری، خواندن و حذف عکس کارگزینی.
//
// قاعده‌ها:
//   - فایل عکس در پوشهٔ دادهٔ سرور ذخیره می‌شود (`<data>/photos/`) و از مسیر امضاشدهٔ
//     `/api/v1/photos/<file>?e=<کد>&t=<امضا>` سرو می‌شود؛ پس برچسب <img> هم می‌تواند
//     آن را نشان دهد، ولی حدس نام فایل بدون امضا عکس کارمند را باز نمی‌کند.
//   - سنجش نوع و حجم فایل، ساخت نام امن و نگهداشت نگاشت در بستهٔ `platform/photos` است؛
//     این لایه فقط ترجمهٔ درخواست/پاسخ است و هیچ قاعده‌ای اینجا نوشته نمی‌شود.
//   - خواندن/حذف عکس با همان محدودهٔ دسترسی پروندهٔ کارمند انجام می‌شود (فیلتر نقش).

package api

import (
	"errors"
	"io"
	"net/http"

	"hrms/internal/contracts"
	"hrms/internal/modules/employee/app"
	"hrms/internal/modules/employee/infra"
	"hrms/internal/platform/httpx"
	"hrms/internal/platform/photos"
)

// GetPhoto وضعیت عکس کارگزینی یک کارمند.
func (h *Handler) GetPhoto(w http.ResponseWriter, r *http.Request) {
	state, err := h.svc.PhotoState(r.Context(), h.filterFrom(r), r.PathValue("id"))
	if err != nil {
		h.writePhotoError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, state)
}

// UploadPhoto بارگذاری عکس کارگزینی (multipart با فیلد «file»).
func (h *Handler) UploadPhoto(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	// پیش از خواندن بدنه، وجود کارمند و محدودهٔ دسترسی سنجیده می‌شود تا فایل بی‌مقصد
	// روی دیسک نوشته نشود.
	if _, ok, err := h.svc.Get(r.Context(), h.filterFrom(r), id); err != nil {
		h.writePhotoError(w, err)
		return
	} else if !ok {
		httpx.WriteError(w, http.StatusNotFound, contracts.CodeNotFound, nil)
		return
	}
	if err := r.ParseMultipartForm(photos.MaxBytes + (1 << 20)); err != nil {
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
	data, err := io.ReadAll(io.LimitReader(file, photos.MaxBytes+1))
	if err != nil {
		httpx.WriteErrorMsg(w, http.StatusBadRequest, contracts.CodeBadRequest, "خواندن فایل عکس ممکن نشد.", nil)
		return
	}
	name := ""
	if header != nil {
		name = header.Filename
	}
	state, err := h.svc.SavePhoto(r.Context(), id, name, data, h.actor(r))
	if err != nil {
		h.writePhotoError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, state)
}

// DeletePhoto حذف عکس کارگزینی.
func (h *Handler) DeletePhoto(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.DeletePhoto(r.Context(), r.PathValue("id"), h.actor(r)); err != nil {
		h.writePhotoError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, contracts.EmployeePhoto{
		EmployeeID: r.PathValue("id"),
		NoteFa:     "عکس کارگزینی حذف شد؛ پرونده بدون عکس نمایش داده می‌شود.",
	})
}

// writePhotoError خطاهای مسیرهای عکس را به پاسخ فارسی تبدیل می‌کند.
func (h *Handler) writePhotoError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, infra.ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, contracts.CodeNotFound, nil)
	case errors.Is(err, app.ErrPhotoStoreUnavailable):
		httpx.WriteErrorMsg(w, http.StatusConflict, contracts.CodeConflict,
			"انبار عکس کارگزینی در این اجرا آماده نیست؛ سرور را با پوشهٔ داده اجرا کنید.", nil)
	default:
		var validation *app.ValidationError
		if errors.As(err, &validation) {
			httpx.WriteError(w, http.StatusUnprocessableEntity, contracts.CodeValidationFailed, validation.Details)
			return
		}
		// خطاهای سنجش فایل (نوع نامعتبر، حجم بیش از حد، نام ناامن) پیام فارسی خودشان را
		// دارند؛ همان پیام به کاربر نشان داده می‌شود.
		httpx.WriteErrorMsg(w, http.StatusUnprocessableEntity, contracts.CodeValidationFailed, err.Error(), nil)
	}
}
