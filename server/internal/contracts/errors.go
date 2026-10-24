// Package contracts قرارداد مشترک بین ماژول‌ها و لایهٔ پلتفرم است.
// تنها چیزی که ماژول‌ها اجازه دارند از یکدیگر ببینند همین بسته است:
// DTOهای JSON (هم‌خوان با contracts/openapi/hrms.json) و portهای بین‌ماژولی.
// این بسته هیچ وابستگی داخلی ندارد تا مرز ماژول‌ها قابل‌اجرا بماند.
package contracts

// Error پاسخ استاندارد خطا؛ کدها ماشین‌خوان و پیام‌ها فارسی هستند.
type Error struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Details map[string]string `json:"details,omitempty"`
}

// MyErrors کدهای خطای شناخته‌شده سامانه.
const (
	CodeUnauthorized     = "unauthorized"
	CodeForbidden        = "forbidden"
	CodeNotFound         = "not_found"
	CodeValidationFailed = "validation_failed"
	CodeConflict         = "version_conflict"
	CodeNotImplemented   = "not_implemented"
	CodeInternal         = "internal_error"
	CodeBadRequest       = "bad_request"
)
