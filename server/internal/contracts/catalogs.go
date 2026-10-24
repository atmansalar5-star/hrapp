package contracts

import "context"

// قرارداد کاتالوگ‌های مرجع زمان‌اجرا.
//
// چرا اینجا و چرا پورت: ماژول «تنظیمات» مرجع قابل ویرایش مدیریت‌ها، بانک‌ها و واحدها را
// نگه می‌دارد و ماژول‌های وام و پاداش باید همان فهرست را ببینند. هیچ ماژولی نباید
// داخلی‌های ماژول دیگر را import کند، پس مثل «پورت کارکنان» و «منبع گزارش»، اینجا فقط
// قرارداد اعلام می‌شود و ماژول تنظیمات خودش را به‌عنوان تأمین‌کننده ثبت می‌کند.
//
// کلیدها لاتین و پایدارند (داده و API)، عنوان‌ها فارسی و نمایشی.

// کلیدهای کاتالوگ مرجع. همان مقدار در فیلد `kind` رکورد تنظیمات هم به‌کار می‌رود.
const (
	CatalogManagements = "managements"
	CatalogBanks       = "banks"
	CatalogUnits       = "units"
)

// CatalogItem یک قلم کاتالوگ مرجع.
//
// Key شناسهٔ ماشین‌خوان و پایدار است (در رکورد و API ذخیره می‌شود)، TitleFa عنوان فارسی
// نمایشی، ParentKey والد قلم (برای واحد: کلید مدیریت بالادست) و Active یعنی قلم در
// فرم‌ها و اعتبارسنجی دیده شود؛ قلم غیرفعال حذف نمی‌شود تا دادهٔ قدیمی به آن اشاره کند.
type CatalogItem struct {
	Key       string `json:"key"`
	TitleFa   string `json:"titleFa"`
	NoteFa    string `json:"noteFa,omitempty"`
	ParentKey string `json:"parentKey,omitempty"`
	Active    bool   `json:"active"`
	// RecordID شناسهٔ رکورد کاتالوگ در ماژول تنظیمات (برای ویرایش از پنل).
	RecordID string `json:"recordId,omitempty"`
}

// CatalogReader پورت خواندن کاتالوگ‌های مرجع. ماژول تنظیمات آن را فراهم می‌کند و
// مصرف‌کننده‌ها (وام، پاداش، گزارش‌ها) فقط همین قرارداد را می‌شناسند.
type CatalogReader interface {
	// Catalog فهرست اقلام یک کاتالوگ را برمی‌گرداند. kind ناشناخته فهرست خالی می‌دهد.
	Catalog(ctx context.Context, kind string) ([]CatalogItem, error)
}

// MergeManagementKeys کاتالوگ مرجع را روی فهرست پایهٔ مدیریت‌های بندر امام سوار می‌کند و
// نقشهٔ «کلید ← عنوان فارسی» می‌سازد.
//
// قاعده: کاتالوگ زمان‌اجرا بر پایه مقدم است (کلید تکراری، عنوان تازه می‌گیرد) و اقلام
// غیرفعال کنار گذاشته می‌شوند. اگر کاتالوگ خالی باشد، همان فهرست پایه می‌ماند؛ پس
// رفتار پیش‌فرض سامانه با این تغییر عوض نمی‌شود.
func MergeManagementKeys(base []Management, items []CatalogItem) map[string]string {
	out := make(map[string]string, len(base)+len(items))
	for _, m := range base {
		out[m.Key] = m.TitleFa
	}
	for _, it := range items {
		key := it.Key
		if key == "" {
			continue
		}
		if !it.Active {
			delete(out, key)
			continue
		}
		out[key] = it.TitleFa
	}
	return out
}
