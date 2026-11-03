// قرارداد مشترک «خدمات و تسهیلات افراد تحت تکفل».
//
// چرا یک مرجع مشترک: «تعلق» فرد تحت تکفل به خدمات درمانی، بیمهٔ تکمیلی، سفر کارت و
// کارت‌های رفاهی (پتروکارت/نفتکارت) در یک جا اعلام می‌شود و سه مصرف‌کننده از همان
// می‌خوانند: فرم تکفل (دامنه و رابط)، موتور شارژ کارت‌ها (که باید فقط افراد با تعلق
// ثبت‌شده را شارژ کند) و گزارش‌ها. اگر هر جا فهرست جدا ساخته می‌شد، کلید «petrokart»
// در یک جا و «پتروکارت» در جای دیگر معنا پیدا می‌کرد.
package contracts

import (
	"context"
	"strings"
)

// DependentService یک خدمت یا تسهیلاتی که فرد تحت تکفل می‌تواند مشمول آن باشد.
//
// Key کلید ماشین‌خوان (لاتین) و TitleFa برچسب فارسی است. CardKey (اختیاری) پل به
// کارت‌های رفاهی است: اگر پر باشد، تعلق این خدمت مبنای شارژ همان کارت می‌شود.
type DependentService struct {
	Key     string `json:"key"`
	TitleFa string `json:"titleFa"`
	CardKey string `json:"cardKey,omitempty"`
	// GroupFa گروه نمایشی در فرم تکفل (بیمه و درمان / رفاه و کارت).
	GroupFa string `json:"groupFa,omitempty"`
	// Master پرچمِ «تکفل» است: تیک اصلی شناخته‌شدن فرد به‌عنوان فرد تحت تکفل. فرم‌ها و
	// گزارش‌های بعدی با همین پرچم می‌فهمند کدام چک‌باکس کلید اصلی است، نه با مقایسهٔ رشته.
	Master bool `json:"master,omitempty"`
	// ApprovalKey پرچم مصوبه/مجوز: خدماتی که بدون مصوبهٔ جداگانه تعلق نمی‌گیرند
	// (نمونه: «مصوبهٔ پرواز») با این پرچم علامت می‌خورند.
	ApprovalKey bool `json:"approvalKey,omitempty"`
}

// dependentServiceCatalog فهرست خدمات قابل تعلق برای فرد تحت تکفل.
//
// دو تیک تازه هم اینجاست: «تکفل» (تیک اصلی شناخته‌شدن فرد به‌عنوان تحت تکفل) و
// «مصوبهٔ پرواز» (مجوز استفاده از بلیت/پرواز). این دو فعلاً ذخیره می‌شوند تا
// فرم‌های بعدی (درخواست پرواز، گزارش‌ها و شارژ) از همین داده بخوانند.
var dependentServiceCatalog = []DependentService{
	{Key: "dependency", TitleFa: "تکفل", GroupFa: "تکفل و وضعیت", Master: true},
	{Key: "medical", TitleFa: "خدمات درمانی", GroupFa: "بیمه و درمان"},
	{Key: "supplementary_insurance", TitleFa: "بیمهٔ تکمیلی", GroupFa: "بیمه و درمان"},
	{Key: "travel_card", TitleFa: "سفر کارت", GroupFa: "رفاه و سفر"},
	{Key: "flight_approval", TitleFa: "مصوبهٔ پرواز", GroupFa: "رفاه و سفر", ApprovalKey: true},
	{Key: "petrokart", TitleFa: "پتروکارت (بخش فرهنگی و رستوران)", CardKey: "petrokart", GroupFa: "کارت رفاهی"},
	{Key: "naftcard", TitleFa: "نفت‌کارت (نفتمارت / فارس‌کارت / سامان‌کارت)", CardKey: "naftmart", GroupFa: "کارت رفاهی"},
}

// DependentServices فهرست خدمات تعلق (کپی، تا مصرف‌کننده اصل را تغییر ندهد).
func DependentServices() []DependentService {
	out := make([]DependentService, len(dependentServiceCatalog))
	copy(out, dependentServiceCatalog)
	return out
}

// DependentServiceByKey یافتن یک خدمت با کلید.
func DependentServiceByKey(key string) (DependentService, bool) {
	value := strings.TrimSpace(key)
	for _, service := range dependentServiceCatalog {
		if service.Key == value {
			return service, true
		}
	}
	return DependentService{}, false
}

// DependentServiceTitleFa عنوان فارسی یک خدمت؛ کلید ناشناخته خودش برمی‌گردد.
func DependentServiceTitleFa(key string) string {
	if service, ok := DependentServiceByKey(key); ok {
		return service.TitleFa
	}
	return strings.TrimSpace(key)
}

// ValidDependentService بررسی معتبر بودن کلید خدمت.
func ValidDependentService(key string) bool {
	_, ok := DependentServiceByKey(key)
	return ok
}

// DependentServiceKeys کلیدهای مجاز خدمات (برای اعتبارسنجی و فهرست گزینه‌ها).
func DependentServiceKeys() []string {
	out := make([]string, 0, len(dependentServiceCatalog))
	for _, service := range dependentServiceCatalog {
		out = append(out, service.Key)
	}
	return out
}

// ServiceForCard کلید خدمت متناظر یک کارت رفاهی.
//
// مصرف‌کننده: موتور شارژ کارت‌ها — می‌پرسد «برای شارژ این کارت، تعلقِ کدام خدمت لازم
// است؟» و پاسخ از همین مرجع می‌آید، نه از یک switch در لایهٔ شارژ.
func ServiceForCard(card string) string {
	value := strings.TrimSpace(card)
	for _, service := range dependentServiceCatalog {
		if service.CardKey != "" && service.CardKey == value {
			return service.Key
		}
	}
	return ""
}

// ServiceLabelsFa عنوان‌های فارسی یک مجموعهٔ کلید خدمت (به ترتیب خودِ فهرست).
func ServiceLabelsFa(keys []string) []string {
	wanted := map[string]bool{}
	for _, key := range keys {
		wanted[strings.TrimSpace(key)] = true
	}
	out := []string{}
	for _, service := range dependentServiceCatalog {
		if wanted[service.Key] {
			out = append(out, service.TitleFa)
		}
	}
	return out
}

// NormalizeDependentServices کلیدهای خدمات را پاک‌سازی و یکتا می‌کند و ترتیب فهرست
// مرجع را برمی‌گرداند؛ کلید ناشناخته حذف می‌شود تا دادهٔ نامعتبر ذخیره نشود.
func NormalizeDependentServices(keys []string) []string {
	wanted := map[string]bool{}
	for _, key := range keys {
		value := strings.TrimSpace(key)
		if ValidDependentService(value) {
			wanted[value] = true
		}
	}
	out := []string{}
	for _, service := range dependentServiceCatalog {
		if wanted[service.Key] {
			out = append(out, service.Key)
		}
	}
	return out
}

// HasDependentService آیا کلید داده‌شده در فهرست خدمات این فرد هست.
func HasDependentService(services []string, key string) bool {
	wanted := strings.TrimSpace(key)
	if wanted == "" {
		return false
	}
	for _, service := range services {
		if strings.TrimSpace(service) == wanted {
			return true
		}
	}
	return false
}

// DependentReader پورت خواندن افراد تحت تکفل برای ماژول‌های دیگر.
//
// چرا فقط تکفل‌های تأییدشده: مصرف‌کننده (موتور شارژ کارت رفاهی و گزارش‌های بیمه) نباید
// فردی را که تکفلش تأیید نشده مشمول کند. اعمال محدودهٔ دسترسی نقش هم داخل همین پورت
// انجام می‌شود و مصرف‌کننده فقط قرارداد را می‌بیند.
type DependentReader interface {
	// ApprovedDependents تکفل‌های «تأییدشدهٔ» یک کارمند را برمی‌گرداند.
	ApprovedDependents(ctx context.Context, employeeID string) ([]Dependent, error)
}
