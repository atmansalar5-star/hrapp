// قراردادها و نوع‌های مشترک دامنهٔ «report» — زبان مشترک بین ماژول‌ها، کلاینت‌ها و پایگاه داده.

package contracts

import (
	"context"
	"strings"
)

// ReportField یک قلم/ستون انتخاب‌شدنی در گزارش‌ساز است.
//
// Type یکی از text|number|money|date است تا کلاینت بتواند قالب‌بندی مناسب را نشان دهد.
type ReportField struct {
	Key     string `json:"key"`
	LabelFa string `json:"labelFa"`
	Type    string `json:"type"`
}

// ReportFilterOption یک گزینهٔ انتخابی برای فیلترهای نوع «فهرستی» (مثل نوع وام یا بانک عامل).
type ReportFilterOption struct {
	Value   string `json:"value"`
	LabelFa string `json:"labelFa"`
}

// ReportFilterDef یک فیلتر اعلامی منبع داده است.
//
// منبع دادهٔ هر ماژول فیلترهای معنادار خودش را اعلام می‌کند (نوع وام، بانک عامل، وضعیت…)
// تا گزارش‌ساز رابط را از همین بسازد و هیچ گزینه‌ای در UI هاردکد نشود. فیلترهای
// «select» فهرست گزینه‌ها را همراه دارند؛ بقیه با ورودی آزاد پر می‌شوند و می‌توان روی
// آن‌ها شرط گذاشت (برابر، بزرگ‌تر، شامل، بین…).
//
// Multiple یعنی فیلتر «چندانتخابی» است: کاربر می‌تواند چند گزینه را تیک بزند (مثلاً
// «کارگر رسمی» و «کارمند مدت معین» با هم) و رابط مقدارهای تیک‌خورده را با کاما
// می‌فرستد. سرور همان را «هر یک از این مقادیر» می‌فهمد.
type ReportFilterDef struct {
	Key      string               `json:"key"`
	LabelFa  string               `json:"labelFa"`
	Type     string               `json:"type"` // text|number|money|date|select
	Options  []ReportFilterOption `json:"options,omitempty"`
	Multiple bool                 `json:"multiple,omitempty"`
}

// MatchValue تطبیق یک مقدار با فیلتر تک‌مقداری یا چندانتخابی.
//
// قاعدهٔ مشترک همهٔ فیلترها: اگر فهرست چندانتخابی پر باشد بر فیلتر تک‌مقداری مقدم
// است و مقدار باید در همان فهرست باشد («هر یک از»)، وگرنه فیلتر تک‌مقداری خالی یعنی
// «همه». این تابع یک‌جا نوشته شده تا در انبار کارکنان، فیلتر API و گزارش‌ها تکرار نشود.
func MatchValue(single string, many []string, value string) bool {
	if len(many) > 0 {
		want := strings.TrimSpace(value)
		for _, m := range many {
			if strings.TrimSpace(m) == want {
				return true
			}
		}
		return false
	}
	return single == "" || value == single
}

// SplitValues یک مقدار فیلتر «چندانتخابی» را به اجزایش می‌شکند.
//
// رابط مقدارهای تیک‌خورده را با کاما می‌فرستد؛ این تابع همان قرارداد را در سرور
// می‌خواند و فاصله‌های اضافی و کاما/نقطه‌ویرگول فارسی را هم می‌پذیرد تا مقدارهای
// چسبیده به هم یا تایپ‌شدهٔ دستی هم درست فهمیده شوند.
func SplitValues(raw string) []string {
	out := []string{}
	for _, part := range strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == '،' || r == ';' || r == '؛'
	}) {
		if v := strings.TrimSpace(part); v != "" {
			out = append(out, v)
		}
	}
	return out
}

// ReportOperator یک عملگر شرط گزارش با برچسب فارسی.
type ReportOperator struct {
	Key     string `json:"key"`
	LabelFa string `json:"labelFa"`
}

// ReportCondition یک شرط روی یک قلم گزارش؛ اگر چند شرط بیاید، همه باید برقرار باشند.
type ReportCondition struct {
	Field  string `json:"field"`
	Op     string `json:"op"`
	Value  string `json:"value"`
	Value2 string `json:"value2,omitempty"` // فقط برای عملگر «بین دو مقدار»
}

// ReportSourceMeta معرفی یک منبع دادهٔ گزارش برای نمایش در گزارش‌ساز.
type ReportSourceMeta struct {
	Key     string        `json:"key"`
	TitleFa string        `json:"titleFa"`
	Fields  []ReportField `json:"fields"`
	// Filters فیلترهای اعلامی این منبع (نوع وام، بانک عامل، وضعیت…).
	Filters []ReportFilterDef `json:"filters,omitempty"`
}

// ReportRequest درخواست اجرای گزارش بر پایهٔ اقلام انتخابی توسط کاربر.
//
// Filterها فیلترهای سادهٔ اعلامی منبع‌اند و Conditions شرط‌های دلخواه روی هر قلم
// (با عملگر)؛ هر دو در سرور اعمال می‌شوند و UI هیچ منطقی از خودش ندارد.
type ReportRequest struct {
	Source     string            `json:"source"`
	Items      []string          `json:"items"`
	Filters    map[string]string `json:"filters,omitempty"`
	Conditions []ReportCondition `json:"conditions,omitempty"`
	Limit      int               `json:"limit,omitempty"`
}

// ReportConditionError خطای یک شرط نامعتبر در گزارش است (شمارهٔ شرط + پیام فارسی).
type ReportConditionError struct {
	Index   int
	Message string
}

func (e *ReportConditionError) Error() string { return e.Message }

// ReportSummary جمع یا میانگین یک ستون عددی.
type ReportSummary struct {
	Key     string  `json:"key"`
	LabelFa string  `json:"labelFa"`
	Total   float64 `json:"total"`
}

// ReportBrand سرصفحهٔ برند گزارش است و از پروفایل پارامتریک شرکت فعال پر می‌شود.
//
// گزارش‌ها هم مثل اسناد چاپی باید نشان و نام شرکت را روی خود داشته باشند؛ چون این
// اطلاعات یک‌جا در سرور به نتیجهٔ گزارش افزوده می‌شود، وب، موبایل و خروجی چاپی همه
// یک سرصفحه می‌بینند و هیچ کلاینتی مجبور نیست نام شرکت را از خودش بسازد.
type ReportBrand struct {
	TitleFa     string `json:"titleFa"`
	HoldingFa   string `json:"holdingFa"`
	ShortCode   string `json:"shortCode"`
	LogoURL     string `json:"logoUrl,omitempty"`
	BrandColor  string `json:"brandColor,omitempty"`
	GeneratedAt string `json:"generatedAt,omitempty"`
	GeneratedBy string `json:"generatedBy,omitempty"`
}

// ReportResult خروجی اجرای گزارش.
type ReportResult struct {
	Source  string          `json:"source"`
	TitleFa string          `json:"titleFa"`
	Columns []ReportField   `json:"columns"`
	Rows    [][]string      `json:"rows"`
	Total   int             `json:"total"`
	NoteFa  string          `json:"noteFa,omitempty"`
	Summary []ReportSummary `json:"summary,omitempty"`
	// Brand سرصفحهٔ شرکت روی گزارش (نشان، نام و رنگ برند).
	Brand *ReportBrand `json:"brand,omitempty"`
	// Items اقلام انتخاب‌شده به ترتیب ستون‌ها؛ برای بازسازی همین گزارش در چاپ/خروجی.
	Items []string `json:"items,omitempty"`
}

// ReportSource یک منبع دادهٔ گزارش است که هر ماژول می‌تواند ثبت کند.
//
// به این ترتیب گزارش‌ساز روی اقلام هر ماژول کار می‌کند، بدون آنکه ماژول گزارش‌ها به
// داخلی‌های آن ماژول وابسته شود؛ ماژول مالک، «اقلام» و «سطرها» را اعلام می‌کند.
type ReportSource interface {
	Metadata() ReportSourceMeta
	Rows(ctx context.Context, req ReportRequest) (ReportResult, error)
}
