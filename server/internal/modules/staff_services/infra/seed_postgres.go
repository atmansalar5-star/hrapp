// دادهٔ اولیهٔ (seed) ماژول «staff_services» برای محیط توسعه و آزمون.

package infra

import (
	"context"

	"hrms/internal/contracts"
	"hrms/internal/platform/calendar"
)

// SeedIfEmpty دادهٔ نمونهٔ آزمایشی افراد تحت تکفل و درخواست پرواز را روی پایگاه‌داده
// می‌ریزد — ولی جدول‌های خالی را؛ دادهٔ واردشده را دست نمی‌زند.
func (s *PostgresStore) SeedIfEmpty(ctx context.Context) (dependents, flights int, err error) {
	today, cerr := calendar.FromGregorian(s.clock())
	if cerr != nil {
		today = calendar.Date{Year: 1400, Month: 1, Day: 1}
	}

	page, err := s.ListDependents(ctx, contracts.DependentFilter{Size: 1})
	if err != nil {
		return 0, 0, err
	}
	if page.Total == 0 {
		for _, d := range SeedDependents(today) {
			if _, err := s.SaveDependent(ctx, d); err != nil {
				return dependents, 0, err
			}
			dependents++
		}
	}

	flightPage, err := s.ListFlights(ctx, contracts.FlightRequestFilter{Size: 1})
	if err != nil {
		return dependents, 0, err
	}
	if flightPage.Total == 0 {
		for _, f := range SeedFlights(today) {
			if _, err := s.SaveFlight(ctx, f); err != nil {
				return dependents, flights, err
			}
			flights++
		}
	}
	return dependents, flights, nil
}
