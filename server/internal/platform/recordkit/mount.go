// هم‌ترازی شمای رکوردکیت با دیتابیس هنگام آغاز اجرا.

package recordkit

import (
	"context"
	"net/http"
	"time"

	"hrms/internal/contracts"
	"hrms/internal/platform/modulekit"
)

// Mount یک ماژول رکورد را به‌طور کامل سوار می‌کند: انتخاب انبار (Postgres یا حافظه)،
// اتصال پورت کارکنان، ساخت سرویس، بذرکاری نمونه فقط در اجرای درون‌حافظه‌ای و ثبت
// مسیرهای استاندارد. هدف این است که هر ماژول اداری فقط «شناسهٔ دامنه» و «دامنهٔ مسیر»
// را اعلام کند و هیچ کد سیم‌کشی تکراری ننویسد.
//
// قاعدهٔ معماری: قاعدهٔ کسب‌وکار در بستهٔ domain همان ماژول اعلام می‌شود (desc)؛ این
// تابع تنها موتور را اجرا می‌کند.
// سرویس ساخته‌شده را برمی‌گرداند تا ماژول بتواند علاوه بر مسیرهای استاندارد، مسیرهای
// اختصاصی خودش (مثل محاسبهٔ حقوق یا گزارش‌ساز) را روی همان داده سوار کند.
func Mount(mux *http.ServeMux, deps modulekit.Deps, desc Descriptor, seed []contracts.RecordInput, prefix string) *Service {
	log := deps.Logger(desc.Module)
	clock := func() time.Time {
		if deps.Clock == nil {
			return time.Now()
		}
		return deps.Clock()
	}

	repo := RepositoryFor(deps, clock)
	var employees EmployeeLookup
	if reader, ok := deps.Ports.Employees(); ok {
		employees = LookupFromReader(reader)
	}
	svc := New(desc, repo, employees, deps.Audit, deps.Sync, clock)

	// بذرکاری نمونه: در اجرای درون‌حافظه‌ای همیشه، و در استقرار واقعی روی Postgres تنها
	// وقتی پرچم «بذرکاری دادهٔ نمونه» روشن باشد (نصب تازهٔ خالی). در هر دو حالت
	// بذرکاری «اگر خالی باشد» است؛ پس دادهٔ واردشدهٔ سازمان هرگز بازنویسی نمی‌شود.
	if deps.DB == nil && len(seed) > 0 {
		// اجرای درون‌حافظه‌ای: انبار تازه است و یک بار پر کردن کافی است.
		if err := svc.SeedIfEmpty(context.Background(), "system", seed); err != nil {
			log.Warn("بذرکاری نمونهٔ ماژول ناموفق بود", "error", err)
		}
	} else if deps.SeedDemo && len(seed) > 0 {
		// نصب روی پایگاه‌دادهٔ واقعی: بذرکاری افزایشی — رکوردهای نمونهٔ نبوده ساخته
		// می‌شوند و رکوردهای موجود (دادهٔ سازمان یا نمونهٔ نوبت پیش) دست‌نخورده می‌مانند.
		created, err := svc.SeedMissing(context.Background(), "system", seed)
		if err != nil {
			log.Warn("بذرکاری نمونهٔ ماژول ناموفق بود", "error", err)
		} else if created > 0 {
			log.Info("دادهٔ نمونهٔ ماژول در پایگاه‌داده بذرکاری شد", "created", created, "candidates", len(seed))
		}
	}

	Register(mux, deps, svc, prefix)
	// ثبت سرویس به‌عنوان منبع کارهای گردش‌کاری تا کارتابل سراسری بتواند کارهای مجاز
	// هر نقش را بدون شکستن مرز ماژول بخواند.
	deps.Ports.RegisterWorkflowSource(svc)
	// و به‌عنوان «منبع سابقه» تا صفحهٔ سوابق پروندهٔ کارمند، اسناد پیشین (احکام قبلی،
	// مدارک پیشین، سمت‌های پیشین…) این ماژول را بدون کد تکراری در هر ماژول ببیند.
	deps.Ports.RegisterHistorySource(svc)
	return svc
}
