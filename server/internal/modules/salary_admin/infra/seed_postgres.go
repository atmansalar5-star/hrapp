// دادهٔ اولیهٔ (seed) ماژول «salary_admin» برای محیط توسعه و آزمون.

package infra

import (
	"context"

	"hrms/internal/contracts"
	"hrms/internal/platform/calendar"
)

// SeedIfEmpty دادهٔ نمونهٔ آزمایشی احکام و گواهی‌ها را روی پایگاه‌داده می‌ریزد — ولی
// فقط اگر جدول‌های آن خالی باشند.
//
// چرا لازم است: در اجرای درون‌حافظه‌ای، انبار خودش با دادهٔ نمونه ساخته می‌شود؛ ولی
// نصب تازه روی Postgres هیچ داده‌ای ندارد و صفحه‌های «احکام حقوق» و «گواهی اشتغال»
// خالی می‌مانند. این تابع همان دادهٔ آزمایشی را در پایگاه‌داده می‌نشاند و چون
// «اگر خالی باشد» است، هیچ‌گاه دادهٔ واردشدهٔ سازمان را بازنویسی یا پاک نمی‌کند.
func (s *PostgresStore) SeedIfEmpty(ctx context.Context) (decrees, certs int, err error) {
	today, cerr := calendar.FromGregorian(s.clock())
	if cerr != nil {
		today = calendar.Date{Year: 1400, Month: 1, Day: 1}
	}

	page, err := s.ListDecrees(ctx, contracts.SalaryDecreeFilter{Size: 1})
	if err != nil {
		return 0, 0, err
	}
	if page.Total == 0 {
		for _, d := range SeedDecrees(today) {
			if _, err := s.SaveDecree(ctx, d); err != nil {
				return 0, 0, err
			}
			decrees++
		}
	}

	certPage, err := s.ListCertificates(ctx, contracts.CertificateFilter{Size: 1})
	if err != nil {
		return decrees, 0, err
	}
	if certPage.Total == 0 {
		for _, c := range SeedCertificates(today) {
			if _, err := s.SaveCertificate(ctx, c); err != nil {
				return decrees, 0, err
			}
			certs++
		}
	}
	return decrees, certs, nil
}
