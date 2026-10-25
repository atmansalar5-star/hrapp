// پیاده‌سازی ماژول «attendance» — «roster»: نوبت مصوب کارکنان برای مصرف در کارکرد و
// ورود فایل تردد.
//
// قاعدهٔ معماری: حل نوبت قاعدهٔ دامنه است (`domain.BuildPlan`) و این فایل فقط دادهٔ
// رکوردهای همین ماژول را به آن می‌رساند. هیچ مصرف‌کنندهٔ دیگری (حقوق، گزارش) نباید
// چرخهٔ نوبت را از نو حساب کند.

package attendance

import (
	"context"
	"sort"
	"strconv"
	"strings"

	"hrms/internal/contracts"
	"hrms/internal/modules/attendance/domain"
	"hrms/internal/platform/calendar"
	"hrms/internal/platform/recordkit"
)

// roster نوبت‌های مصوب کارکنان: برای هر کارمند فهرست تخصیص‌های معتبر با چرخهٔ حل‌شده.
type roster struct {
	byEmployee  map[string][]domain.PlanInput
	minCoverage int
}

// loadRoster الگوهای چرخه و تخصیص‌های نوبت را می‌خواند و آمادهٔ حل نوبت می‌کند.
//
// الگوی بازنشسته کنار می‌رود و تخصیص غیرمصوب نوبت نمی‌سازد؛ پس برنامهٔ نوبت هرگز از
// دادهٔ تأییدنشده ساخته نمی‌شود.
func loadRoster(ctx context.Context, svc *recordkit.Service) *roster {
	out := &roster{byEmployee: map[string][]domain.PlanInput{}}
	if svc == nil {
		return out
	}
	patternPage, err := svc.List(ctx, contracts.RecordFilter{
		Module: "attendance", Kind: "shiftPatterns", Size: 500, Scope: "all"})
	if err != nil {
		return out
	}
	type pattern struct {
		cycle []string
		min   int
	}
	patterns := map[string]pattern{}
	for _, rec := range patternPage.Items {
		if rec.Status == "retired" {
			continue
		}
		cycle, err := domain.ParseCycle(rec.Fields["rotation"])
		if err != nil {
			continue
		}
		min := 0
		if raw := strings.TrimSpace(calendar.NormalizeDigits(rec.Fields["minCoverage"])); raw != "" {
			if n, err := strconv.Atoi(raw); err == nil && n > 0 {
				min = n
			}
		}
		patterns[strings.TrimSpace(rec.Title)] = pattern{cycle: cycle, min: min}
		if min > out.minCoverage {
			out.minCoverage = min
		}
	}

	assignPage, err := svc.List(ctx, contracts.RecordFilter{
		Module: "attendance", Kind: "shiftAssignments", Size: 500, Scope: "all"})
	if err != nil {
		return out
	}
	for _, rec := range assignPage.Items {
		if rec.EmployeeID == "" || !domain.PlanStatusAllowed(rec.Status) {
			continue
		}
		pat, ok := patterns[strings.TrimSpace(rec.Fields["pattern"])]
		if !ok {
			// ارجاع الگو در این استقرار پیدا نشد؛ تخصیص بی‌الگو نوبت نمی‌سازد و در
			// گزارش مغایرت پوشش دیده می‌شود.
			continue
		}
		offset := 0
		if raw := strings.TrimSpace(calendar.NormalizeDigits(rec.Fields["cycleOffset"])); raw != "" {
			if n, err := strconv.Atoi(raw); err == nil && n >= 0 {
				offset = n
			}
		}
		out.byEmployee[rec.EmployeeID] = append(out.byEmployee[rec.EmployeeID], domain.PlanInput{
			AssignmentID: rec.ID,
			EmployeeID:   rec.EmployeeID,
			EmployeeName: rec.EmployeeName,
			PatternTitle: strings.TrimSpace(rec.Fields["pattern"]),
			Cycle:        pat.cycle,
			CycleStart:   strings.TrimSpace(rec.Fields["startDate"]),
			CycleOffset:  offset,
			EndDate:      strings.TrimSpace(rec.Fields["endDate"]),
			Status:       rec.Status,
		})
	}
	return out
}

// ShiftFor نوبت یک کارمند در یک روز. تخصیص مصوبِ پوشانندهٔ آن روز برنده است؛ اگر روزی
// بیرون از همهٔ تخصیص‌ها باشد، نوبتی هم ساخته نمی‌شود.
func (r *roster) ShiftFor(employeeID, date string) (*domain.PlanDay, bool) {
	if r == nil {
		return nil, false
	}
	for _, in := range r.byEmployee[employeeID] {
		days, err := domain.BuildPlan(in, date, date)
		if err != nil || len(days) == 0 {
			continue
		}
		day := days[0]
		return &day, true
	}
	return nil, false
}

// ExpectedWorkDates روزهای موظفی یک کارمند در بازه (روز استراحت شمرده نمی‌شود).
func (r *roster) ExpectedWorkDates(employeeID, from, to string) []string {
	if r == nil {
		return nil
	}
	set := map[string]bool{}
	for _, in := range r.byEmployee[employeeID] {
		days, err := domain.BuildPlan(in, from, to)
		if err != nil {
			continue
		}
		for _, d := range days {
			if d.Rest {
				continue
			}
			set[d.Date] = true
		}
	}
	out := make([]string, 0, len(set))
	for date := range set {
		out = append(out, date)
	}
	sort.Strings(out)
	return out
}

// Employees فهرست کارمندانی که تخصیص نوبت مصوب دارند.
func (r *roster) Employees() []string {
	if r == nil {
		return nil
	}
	out := make([]string, 0, len(r.byEmployee))
	for id := range r.byEmployee {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

// Coverage حداکثر «حداقل پوشش» الگوهای این استقرار.
func (r *roster) Coverage() int {
	if r == nil {
		return 0
	}
	return r.minCoverage
}
