// قرارداد مشترک «شناسنامهٔ شغل» — زبان مشترک ماژول طبقه‌بندی مشاغل و مصرف‌کنندگانش.
//
// چرا در بستهٔ contracts: تصویب‌خواهی انتصاب و ارتقا (ماژول career) باید شرایط احراز شغل
// را از شناسنامهٔ شغل بخواند و فرم انتخاب شغل هم باید فهرست شناسنامه‌ها را ببیند. اگر
// career مستقیم به دامنهٔ jobclass وابسته شود، مرز ماژول می‌شکند (دروازهٔ معماری همین را
// می‌سنجد)؛ و اگر کلید فیلدها را در دو جا بنویسد، یک روز واگرا می‌شوند. پس واژگان مشترک
// اینجا اعلام می‌شود و هر ماژول پیاده‌سازی/مصرف خودش را دارد.
package contracts

import (
	"context"

	"hrms/internal/rules"
)

// JobRequirementView شرایط احراز یک شغل از شناسنامهٔ شغل، به‌همراه شناسهٔ سند.
//
// Approved یعنی شرح شغل تأیید شده است؛ شناسنامهٔ تأییدنشده مبنای انتصاب و ارتقا نیست.
type JobRequirementView struct {
	RecordID    string               `json:"recordId"`
	JobCode     string               `json:"jobCode,omitempty"`
	TitleFa     string               `json:"titleFa"`
	UnitFa      string               `json:"unitFa,omitempty"`
	GroupFa     string               `json:"groupFa,omitempty"`
	Approved    bool                 `json:"approved"`
	StatusFa    string               `json:"statusFa,omitempty"`
	Requirement rules.JobRequirement `json:"requirement"`
	// DegreeFa عنوان فارسی مقطع لازم، برای نمایش در فرم‌ها بدون تکرار فهرست مقاطع.
	DegreeFa string `json:"degreeFa,omitempty"`
}

// JobRegistryReader پورت خواندن شناسنامه‌های شغل (پیاده‌سازی: ماژول طبقه‌بندی مشاغل).
type JobRegistryReader interface {
	// Jobs فهرست شناسنامه‌های شغل سازمان را برمی‌گرداند (برای فهرست‌ها و انتخابگر فرم).
	Jobs(ctx context.Context) ([]JobRequirementView, error)
	// Requirement شرایط احراز یک شغل را با «کد شغل» یا «عنوان شغل» پیدا می‌کند.
	Requirement(ctx context.Context, codeOrTitle string) (JobRequirementView, bool, error)
}
