// Package orgs استقرار پارامتریک شرکت‌ها.
//
// سامانه یک محصول واحد است که برای چند شرکت هلدینگ شبیه‌سازی می‌شود: هر شرکت
// مشخص می‌کند کدام ادارات و ماژول‌ها فعال باشند، پارامترهای حقوقی-مالی‌اش چیست و
// مشخصات برندش چگونه نمایش داده شود. دادهٔ شرکت‌ها از contracts/organizations.json
// تولید می‌شود (orgs.gen.go) و این فایل فقط رفتار می‌دهد.
package orgs

import (
	"fmt"
	"sort"
	"strings"
)

// Param یک پارامتر قابل تنظیم در استقرار شرکت.
type Param struct {
	Key     string
	TitleFa string
	Value   string
	UnitFa  string
	GroupFa string
}

// Organization پروفایل کامل یک شرکت در استقرار پارامتریک.
type Organization struct {
	ID                 string
	Order              int
	TitleFa            string
	TitleEn            string
	HoldingFa          string
	HoldingEn          string
	ShortCode          string
	EmployeeCodePrefix string
	Calendar           string
	CurrencyFa         string
	Timezone           string
	WorkWeekFa         string
	FiscalYearStartFa  string
	LogoEmoji          string
	// LogoFile مسیر نسبی فایل نشان شرکت (زیر پوشهٔ assets رابط وب)؛ خالی یعنی فقط ایموجی.
	LogoFile string
	// BrandColor رنگ اصلی برند شرکت برای سرصفحه و اسناد چاپی.
	BrandColor         string
	Headcount          int
	EnabledDepartments []string
	DisabledModules    []string
	// EnabledModules ماژول‌های نهایی فعال = اداره‌های فعال منهای ماژول‌های غیرفعال.
	EnabledModules []string
	NotesFa        string
	Params         []Param
}

// Default استقرار پیش‌فرض سامانه (شرکتی که در contracts/organizations.json پیش‌فرض است).
func Default() Organization {
	for _, o := range All {
		if o.ID == DefaultID {
			return o
		}
	}
	if len(All) > 0 {
		return All[0]
	}
	return Organization{ID: DefaultID, TitleFa: DefaultID, EnabledModules: nil}
}

// Lookup یافتن شرکت با شناسه یا کد کوتاه (بدون حساسیت به بزرگی و کوچکی حروف).
// اگر شناسه خالی باشد، استقرار پیش‌فرض برمی‌گردد.
func Lookup(id string) (Organization, bool) {
	id = strings.TrimSpace(id)
	if id == "" {
		return Default(), true
	}
	for _, o := range All {
		if strings.EqualFold(o.ID, id) || strings.EqualFold(o.ShortCode, id) {
			return o, true
		}
	}
	return Organization{}, false
}

// IDs فهرست شناسهٔ شرکت‌های شناخته‌شده برای پیام خطا و کمکی CLI.
func IDs() []string {
	out := make([]string, 0, len(All))
	for _, o := range All {
		out = append(out, o.ID)
	}
	sort.Strings(out)
	return out
}

// EnabledFlags پرچم فعال/غیرفعال ماژول‌ها برای رجیستری ماژول‌ها.
func EnabledFlags(o Organization) map[string]bool {
	out := make(map[string]bool, len(o.EnabledModules))
	for _, id := range o.EnabledModules {
		out[id] = true
	}
	return out
}

// Param مقدار یک پارامتر شرکت را برمی‌گرداند.
func (o Organization) Param(key string) (string, bool) {
	for _, p := range o.Params {
		if p.Key == key {
			return p.Value, true
		}
	}
	return "", false
}

// Summary خلاصهٔ قابل‌سریال‌سازی شرکت برای پاسخ API (بدون تکرار همهٔ پارامترها).
func (o Organization) Summary() map[string]any {
	params := make([]map[string]string, 0, len(o.Params))
	for _, p := range o.Params {
		params = append(params, map[string]string{
			"key": p.Key, "titleFa": p.TitleFa, "value": p.Value, "unitFa": p.UnitFa, "groupFa": p.GroupFa,
		})
	}
	return map[string]any{
		"id":      o.ID,
		"order":   o.Order,
		"titleFa": o.TitleFa,
		"titleEn": o.TitleEn, "holdingFa": o.HoldingFa,
		"holdingEn":          o.HoldingEn,
		"shortCode":          o.ShortCode,
		"employeeCodePrefix": o.EmployeeCodePrefix,
		"calendar":           o.Calendar,
		"currencyFa":         o.CurrencyFa,
		"timezone":           o.Timezone,
		"workWeekFa":         o.WorkWeekFa,
		"fiscalYearStartFa":  o.FiscalYearStartFa,
		"logoEmoji":          o.LogoEmoji,
		"logoFile":           o.LogoFile,
		"brandColor":         o.BrandColor,
		"headcount":          o.Headcount,
		"enabledDepartments": o.EnabledDepartments,
		"disabledModules":    o.DisabledModules,
		"enabledModules":     o.EnabledModules,
		"notesFa":            o.NotesFa,
		"params":             params,
	}
}

// ListSummary خلاصهٔ همهٔ شرکت‌ها به ترتیب استقرار.
func ListSummary() []map[string]any {
	out := make([]map[string]any, 0, len(All))
	for _, o := range All {
		out = append(out, o.Summary())
	}
	return out
}

// ListSummariesOfHolding خلاصهٔ شرکت‌های یک هلدینگ به ترتیب استقرار.
func ListSummariesOfHolding(key string) []map[string]any {
	companies := CompaniesOfHolding(key)
	out := make([]map[string]any, 0, len(companies))
	for _, o := range companies {
		out = append(out, o.Summary())
	}
	return out
}

// Holding یک هلدینگ و شرکت‌های زیرمجموعهٔ آن.
//
// هلدینگ از خود شرکت‌ها استخراج می‌شود (نه فهرست جداگانه): هر شرکت می‌داند زیر کدام
// هلدینگ است، پس «باز کردن هلدینگ» یعنی دیدن همین شرکت‌های زیرمجموعه.
type Holding struct {
	// Key شناسهٔ ماشین‌خوان هلدینگ (HoldingEn) برای پارامترهای درخواست.
	Key       string
	TitleFa   string
	TitleEn   string
	Companies []Organization
}

// HoldingKey کلید ماشین‌خوان هلدینگ یک شرکت (HoldingEn با پشتیبان HoldingFa).
func HoldingKey(o Organization) string {
	if strings.TrimSpace(o.HoldingEn) != "" {
		return o.HoldingEn
	}
	return o.HoldingFa
}

// Holdings همهٔ هلدینگ‌ها را با شرکت‌های زیرمجموعه، به ترتیب نخستین ظهور شرکت‌ها،
// برمی‌گرداند.
func Holdings() []Holding {
	out := []Holding{}
	index := map[string]int{}
	for _, o := range All {
		key := HoldingKey(o)
		if key == "" {
			continue
		}
		i, ok := index[key]
		if !ok {
			i = len(out)
			index[key] = i
			out = append(out, Holding{Key: key, TitleFa: o.HoldingFa, TitleEn: o.HoldingEn})
		}
		out[i].Companies = append(out[i].Companies, o)
	}
	return out
}

// HoldingByKey یافتن هلدینگ با کلید ماشین‌خوان یا عنوان فارسی.
func HoldingByKey(key string) (Holding, bool) {
	key = strings.TrimSpace(key)
	for _, h := range Holdings() {
		if strings.EqualFold(h.Key, key) || h.TitleFa == key {
			return h, true
		}
	}
	return Holding{}, false
}

// SameHolding آیا دو شرکت زیر یک هلدینگ‌اند.
func SameHolding(a, b Organization) bool {
	ka, kb := HoldingKey(a), HoldingKey(b)
	return ka != "" && ka == kb
}

// CompaniesOfHolding شرکت‌های یک هلدینگ (با کلید یا عنوان) را برمی‌گرداند.
func CompaniesOfHolding(key string) []Organization {
	if h, ok := HoldingByKey(key); ok {
		return h.Companies
	}
	return nil
}

// UnknownError خطای خوانا برای شناسهٔ شرکت نامعتبر.
func UnknownError(id string) error {
	return fmt.Errorf("شرکت «%s» در فهرست استقرارها نیست؛ شناسه‌های مجاز: %s", id, strings.Join(IDs(), "، "))
}
