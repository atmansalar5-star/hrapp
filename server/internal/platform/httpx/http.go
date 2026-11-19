// Package httpx کمکی‌های HTTP مشترک: پاسخ JSON، خطای فارسی و بدنهٔ استاندارد.
package httpx

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"hrms/internal/contracts"
)

// MaxBodyBytes سقف بدنهٔ درخواست برای جلوگیری از مصرف بی‌رویه حافظه.
const MaxBodyBytes = 1 << 20

var defaultMessages = map[string]string{
	contracts.CodeUnauthorized:     "برای دسترسی باید وارد شوید.",
	contracts.CodeForbidden:        "نقش شما اجازهٔ این عملیات را ندارد.",
	contracts.CodeNotFound:         "رکورد درخواستی یافت نشد.",
	contracts.CodeValidationFailed: "اطلاعات ارسالی معتبر نیست.",
	contracts.CodeConflict:         "رکورد در سرور تغییر کرده است؛ لطفاً دوباره هم‌گام‌سازی کنید.",
	contracts.CodeNotImplemented:   "این بخش هنوز در این نسخه پیاده‌سازی نشده است.",
	contracts.CodeInternal:         "خطای داخلی سامانه رخ داد.",
	contracts.CodeBadRequest:       "درخواست نامعتبر است.",
	contracts.CodeRateLimited:      "تعداد درخواست‌ها بیش از حد مجاز است؛ کمی بعد دوباره تلاش کنید.",
}

// WriteJSON پاسخ موفق را می‌نویسد.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if v == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(v); err != nil {
		// هدر پیش‌تر نوشته شده؛ تنها راه ثبت، نادیده‌گرفتن است.
		return
	}
}

// NoContent پاسخ بدون بدنه (204).
func NoContent(w http.ResponseWriter) { w.WriteHeader(http.StatusNoContent) }

// WriteError پاسخ خطای استاندارد فارسی.
func WriteError(w http.ResponseWriter, status int, code string, details map[string]string) {
	msg, ok := defaultMessages[code]
	if !ok {
		msg = "خطای نامشخص."
	}
	WriteJSON(w, status, contracts.Error{Code: code, Message: msg, Details: details})
}

// WriteErrorMsg مانند WriteError ولی پیام دلخواه می‌فرستد.
func WriteErrorMsg(w http.ResponseWriter, status int, code, message string, details map[string]string) {
	WriteJSON(w, status, contracts.Error{Code: code, Message: message, Details: details})
}

// DecodeJSON بدنهٔ JSON را می‌خواند؛ در صورت خطا پاسخ 400 نوشته و false برمی‌گرداند.
func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		var maxErr *http.MaxBytesError
		switch {
		case errors.As(err, &maxErr):
			WriteErrorMsg(w, http.StatusRequestEntityTooLarge, contracts.CodeBadRequest, "حجم درخواست بیش از حد مجاز است.", nil)
		case errors.Is(err, io.EOF):
			WriteErrorMsg(w, http.StatusBadRequest, contracts.CodeBadRequest, "بدنهٔ درخواست خالی است.", nil)
		default:
			WriteErrorMsg(w, http.StatusBadRequest, contracts.CodeBadRequest, "ساختار JSON درخواست معتبر نیست.", nil)
		}
		return false
	}
	return true
}

// NotImplemented پاسخ ماژولی که در این نسخه پیاده‌سازی نشده، با ارجاع به شناسهٔ ماژول.
func NotImplemented(moduleID string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		WriteJSON(w, http.StatusNotImplemented, contracts.Error{
			Code:    contracts.CodeNotImplemented,
			Message: defaultMessages[contracts.CodeNotImplemented],
			Details: map[string]string{"module": moduleID},
		})
	}
}
