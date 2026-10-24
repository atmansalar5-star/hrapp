package contracts

import "context"

// DocumentTemplate قالب متن یک سند چاپی است.
//
// متن قرارداد کار، متن فیش حقوقی، احکام حقوق و گواهی اشتغال «قابل طراحی» هستند: به‌جای
// هاردکد متن در کد سرور، متن قالب اینجا نگه‌داری می‌شود و می‌تواند با جای‌نگهدارها
// (مثل {{employee.fullName}}) ویرایش شود. موتور اسناد هنگام چاپ، قالب فعال همان نوع
// سند را می‌خواند؛ نبود قالب یعنی استفاده از متن پیش‌فرض سامانه.
type DocumentTemplate struct {
	ID        string `json:"id"`
	DocType   string `json:"docType"`
	DocTypeFa string `json:"docTypeFa"`
	Title     string `json:"title"`
	Header    string `json:"header,omitempty"`
	Body      string `json:"body"`
	Footer    string `json:"footer,omitempty"`
	Status    string `json:"status,omitempty"`
	Version   int    `json:"version,omitempty"`
}

// DocumentTemplateProvider پورت خواندن قالب‌های متن اسناد.
//
// پیاده‌سازی این پورت در ماژول تنظیمات ثبت می‌شود و موتور اسناد از آن استفاده می‌کند؛
// این‌گونه ماژول اسناد به داخلی‌های ماژول تنظیمات وابسته نمی‌شود.
type DocumentTemplateProvider interface {
	// Templates همهٔ قالب‌ها را برمی‌گرداند (برای صفحهٔ قالب‌ها و اندپوینتهای مصرفی).
	Templates(ctx context.Context) ([]DocumentTemplate, error)
	// ActiveTemplate قالب فعال یک نوع سند را برمی‌گرداند؛ نبود آن یعنی متن پیش‌فرض.
	ActiveTemplate(ctx context.Context, docType string) (DocumentTemplate, bool, error)
}
