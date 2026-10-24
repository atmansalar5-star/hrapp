// دادهٔ اولیهٔ (seed) ماژول «contracts» برای محیط توسعه و آزمون.

package infra

import (
	"context"

	"hrms/internal/contracts"
	"hrms/internal/platform/calendar"
)

// SeedIfEmpty قراردادهای نمونهٔ آزمایشی را روی پایگاه‌داده می‌ریزد — فقط اگر هیچ
// قراردادی نباشد.
//
// چرا: نصب تازه روی Postgres، صفحهٔ «قراردادها» و هر جای دیگر که قرارداد جاری کارمند
// را می‌خواند (فیش حقوق، پروندهٔ کارمند، پاداش پایان خدمت) خالی می‌ماند. این تابع همان
// دادهٔ نمونهٔ حافظه را می‌نشاند و چون «اگر خالی باشد» است، قرارداد واقعی سازمان را
// بازنویسی نمی‌کند.
func (s *PostgresStore) SeedIfEmpty(ctx context.Context) (int, error) {
	page, err := s.List(ctx, contracts.ContractFilter{Size: 1})
	if err != nil {
		return 0, err
	}
	if page.Total > 0 {
		return 0, nil
	}
	return s.saveSeed(ctx)
}

// SeedMissing قراردادهای نمونه‌ای را که هنوز در انبار نیستند می‌سازد و تعداد
// ساخته‌شده را برمی‌گرداند.
//
// چرا جدا از SeedIfEmpty: در نصب‌هایی که چند قرارداد واقعی وارد شده ولی بذرکاری کامل
// اجرا نشده بود، شرط «انبار خالی» جلوی نشستن بقیهٔ نمونه‌ها را می‌گرفت و صفحهٔ تمدید و
// چرخهٔ قرارداد بی‌محتوا می‌شد. این تابع تنها شناسه‌های غایب را می‌افزاید؛ پس نه قرارداد
// واقعی بازنویسی می‌شود و نه نمونهٔ نوبت پیش دست‌خورده می‌شود.
func (s *PostgresStore) SeedMissing(ctx context.Context) (int, error) {
	return s.saveSeed(ctx)
}

// saveSeed نمونه‌ها را می‌ریزد و هر شناسه‌ای را که از قبل هست دست‌نخورده می‌گذارد.
func (s *PostgresStore) saveSeed(ctx context.Context) (int, error) {
	today, cerr := calendar.FromGregorian(s.clock())
	if cerr != nil {
		today = calendar.Date{Year: 1400, Month: 1, Day: 1}
	}
	n := 0
	for _, c := range SeedContracts(today, s.clock) {
		if _, exists, err := s.Get(ctx, c.ID); err != nil {
			return n, err
		} else if exists {
			continue
		}
		if _, err := s.Save(ctx, c); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}
