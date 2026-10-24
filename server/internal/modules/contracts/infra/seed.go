// دادهٔ اولیهٔ (seed) ماژول «contracts» برای محیط توسعه و آزمون.

package infra

import (
	"time"

	"hrms/internal/modules/contracts/domain"
	"hrms/internal/platform/calendar"
)

// SeedContracts قراردادهای نمونهٔ سازمان — هم‌خوان با کارکنان نمونهٔ ماژول کارکنان
// (همان ۱۲ نفر با همان واحد و پست).
//
// تاریخ‌ها نسبت به «امروز» ساخته می‌شوند، نه ثابت: نمونهٔ نمایشی باید هر روز که باز
// شود معنادار بماند — یک قرارداد در آستانهٔ سرآمد، یکی تازه تمدیدشده، یکی خاتمه‌یافته.
// اگر تاریخ‌ها ثابت بودند، چند ماه بعد همه «منقضی» به نظر می‌رسیدند و صفحهٔ تمدید
// بی‌محتوا می‌شد.
func SeedContracts(today calendar.Date, clock func() time.Time) []domain.Contract {
	day := func(offset int) string {
		d, err := today.AddDays(offset)
		if err != nil {
			return today.String()
		}
		return d.String()
	}
	at := func(daysAgo int) time.Time {
		base := clock()
		return base.AddDate(0, 0, -daysAgo)
	}
	created := func(actor, note string, daysAgo int) []domain.Event {
		return []domain.Event{{At: at(daysAgo), Actor: actor, Action: domain.ActionCreate, NoteFa: note}}
	}

	return []domain.Contract{
		{
			ID: "CT-1394-0009", EmployeeID: "EMP-009", EmployeeName: "حسین موسوی",
			EmploymentType: "official_oil",
			DepartmentName: "تولید و عملیات", Position: "مدیر تولید",
			Kind: domain.KindPermanent, StartDate: day(-3600), MonthlySalary: 54_000_000,
			Status: domain.StatusActive, CreatedBy: "hr.manager", DecidedBy: "hr.manager",
			Events: created("hr.manager", "انعقاد قرارداد مدت دائم", 3600),
		},
		{
			ID: "CT-1395-0006", EmployeeID: "EMP-006", EmployeeName: "نرگس کریمی",
			EmploymentType: "official_oil",
			DepartmentName: "مالی و حسابداری", Position: "مدیر مالی",
			Kind: domain.KindPermanent, StartDate: day(-3300), MonthlySalary: 56_000_000,
			Status: domain.StatusActive, CreatedBy: "hr.manager", DecidedBy: "hr.manager",
			Events: created("hr.manager", "انعقاد قرارداد مدت دائم", 3300),
		},
		{
			ID: "CT-1397-0002", EmployeeID: "EMP-002", EmployeeName: "آرش رضایی",
			EmploymentType: "official_oil",
			DepartmentName: "فناوری اطلاعات", Position: "مدیر فناوری اطلاعات",
			Kind: domain.KindPermanent, StartDate: day(-2700), MonthlySalary: 58_000_000,
			Status: domain.StatusActive, CreatedBy: "hr.manager", DecidedBy: "hr.manager",
			Events: created("hr.manager", "انعقاد قرارداد مدت دائم", 2700),
		},
		{
			ID: "CT-1398-0001", EmployeeID: "EMP-001", EmployeeName: "سارا محمدی",
			EmploymentType: "official_oil",
			DepartmentName: "منابع انسانی", Position: "مدیر منابع انسانی",
			Kind: domain.KindPermanent, StartDate: day(-2500), MonthlySalary: 52_000_000,
			Status: domain.StatusActive, CreatedBy: "hr.manager", DecidedBy: "hr.manager",
			Events: created("hr.manager", "انعقاد قرارداد مدت دائم", 2500),
		},
		{
			ID: "CT-1398-0003", EmployeeID: "EMP-003", EmployeeName: "نگار حسینی",
			EmploymentType: "fixed_1662",
			DepartmentName: "فروش و بازاریابی", Position: "کارشناس فروش",
			Kind: domain.KindPermanent, StartDate: day(-2400), MonthlySalary: 34_000_000,
			Status: domain.StatusActive, CreatedBy: "hr.manager", DecidedBy: "hr.manager",
			Events: created("hr.manager", "انعقاد قرارداد مدت دائم", 2400),
		},
		{
			ID: "CT-1399-0004", EmployeeID: "EMP-004", EmployeeName: "علی مرادی",
			EmploymentType: "official_worker",
			DepartmentName: "مالی و حسابداری", Position: "کارشناس ارشد مالی",
			Kind: domain.KindPermanent, StartDate: day(-2100), MonthlySalary: 41_000_000,
			Status: domain.StatusActive, CreatedBy: "hr.manager", DecidedBy: "hr.manager",
			Events: created("hr.manager", "انعقاد قرارداد مدت دائم", 2100),
		},
		{
			ID: "CT-1399-0012", EmployeeID: "EMP-012", EmployeeName: "الهام رستمی",
			EmploymentType: "fixed_1662",
			DepartmentName: "فروش و بازاریابی", Position: "کارشناس بازاریابی",
			Kind: domain.KindFixedTerm, StartDate: day(-2300), EndDate: day(-400), MonthlySalary: 33_000_000,
			Status: domain.StatusTerminated, CreatedBy: "hr.manager", DecidedBy: "hr.manager",
			NoteFa: "پایان همکاری و تسویه‌حساب",
			Events: []domain.Event{
				{At: at(2300), Actor: "hr.manager", Action: domain.ActionCreate, NoteFa: "انعقاد قرارداد مدت موقت"},
				{At: at(400), Actor: "hr.manager", Action: domain.ActionTerminate, NoteFa: "خاتمهٔ قرارداد و تسویه‌حساب"},
			},
		},
		{
			ID: "CT-1400-0007", EmployeeID: "EMP-007", EmployeeName: "رضا کاظمی",
			EmploymentType: "official_worker",
			DepartmentName: "تولید و عملیات", Position: "سرپرست تولید",
			Kind: domain.KindPermanent, StartDate: day(-1600), MonthlySalary: 38_000_000,
			Status: domain.StatusActive, CreatedBy: "hr.spec", DecidedBy: "hr.manager",
			Events: created("hr.spec", "انعقاد قرارداد مدت دائم", 1600),
		},
		{
			ID: "CT-1402-0008", EmployeeID: "EMP-008", EmployeeName: "مریم نوری",
			EmploymentType: "fixed_holding",
			DepartmentName: "منابع انسانی", Position: "کارشناس منابع انسانی",
			Kind: domain.KindPermanent, StartDate: day(-830), MonthlySalary: 31_000_000,
			Status: domain.StatusActive, CreatedBy: "hr.spec", DecidedBy: "hr.manager",
			Events: created("hr.spec", "انعقاد قرارداد مدت دائم", 830),
		},
		{
			ID: "CT-1404-0005", EmployeeID: "EMP-005", EmployeeName: "محمد احمدی",
			EmploymentType: "fixed_1662",
			DepartmentName: "فروش و بازاریابی", Position: "مدیر فروش",
			Kind: domain.KindFixedTerm, StartDate: day(-180), EndDate: day(53), MonthlySalary: 55_000_000,
			Status: domain.StatusActive, CreatedBy: "hr.spec", DecidedBy: "hr.manager",
			NoteFa: "قرارداد یک‌ساله با امکان تمدید",
			Events: created("hr.spec", "انعقاد قرارداد مدت موقت", 180),
		},
		{
			ID: "CT-1404-0010", EmployeeID: "EMP-010", EmployeeName: "فاطمه صادقی",
			EmploymentType: "contractor_project",
			DepartmentName: "فناوری اطلاعات", Position: "برنامه‌نویس",
			Kind: domain.KindFixedTerm, StartDate: day(-210), EndDate: day(68), MonthlySalary: 36_000_000,
			Status: domain.StatusActive, CreatedBy: "hr.spec", DecidedBy: "hr.manager",
			NoteFa: "قرارداد پیمانکاری مدت موقت",
			Events: created("hr.spec", "انعقاد قرارداد مدت موقت", 210),
		},
		{
			ID: "CT-1404-0011", EmployeeID: "EMP-011", EmployeeName: "امیر قاسمی",
			EmploymentType: "long_term",
			DepartmentName: "فناوری اطلاعات", Position: "کارشناس شبکه",
			Kind: domain.KindFixedTerm, StartDate: day(-120), EndDate: day(245), MonthlySalary: 29_000_000,
			Status: domain.StatusActive, CreatedBy: "hr.spec", DecidedBy: "hr.manager",
			NoteFa: "دورهٔ آزمایشی؛ ارزیابی پیش از تبدیل به مدت دائم",
			Events: created("hr.spec", "انعقاد قرارداد مدت موقت", 120),
		},
		{
			ID: "CT-1404-0105", EmployeeID: "EMP-005", EmployeeName: "محمد احمدی",
			EmploymentType: "fixed_1662",
			DepartmentName: "فروش و بازاریابی", Position: "مدیر فروش",
			Kind: domain.KindFixedTerm, StartDate: day(54), EndDate: day(419), MonthlySalary: 62_000_000,
			Status: domain.StatusPending, SupersedesID: "CT-1404-0005", RenewCount: 1,
			CreatedBy: "hr.spec", NoteFa: "تمدید یک‌ساله با بازنگری مبلغ پایه",
			Events: []domain.Event{{At: at(3), Actor: "hr.spec", Action: domain.ActionRenew, NoteFa: "پیش‌نویس تمدید برای تأیید"}},
		},
	}
}
