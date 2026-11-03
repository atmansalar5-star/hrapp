// دادهٔ اولیهٔ (seed) ماژول «staff_services» برای محیط توسعه و آزمون.

package infra

import (
	"hrms/internal/modules/staff_services/domain"
	"hrms/internal/platform/calendar"
)

// SeedDependents افراد تحت تکفل نمونه — هم‌خوان با کارکنان نمونهٔ سامانه.
// کدهای ملی عمداً دارای رقم کنترل معتبرند تا سنجهٔ اعتبارسنجی روی داده خودمان هم
// واقعاً آزموده شود.
func SeedDependents(today calendar.Date) []domain.Dependent {
	day := func(offset int) string {
		d, err := today.AddDays(offset)
		if err != nil {
			return today.String()
		}
		return d.String()
	}
	return []domain.Dependent{
		{
			ID: "DEP-0101", EmployeeID: "EMP-005", EmployeeName: "محمد احمدی",
			DepartmentName: "فروش و بازاریابی", FullName: "فاطمه احمدی", Relation: domain.RelationSpouse,
			NationalID: "0079845630", BirthDate: day(-13500), CoverageStart: day(-1800),
			Insurance: true, DocumentsFa: "کپی شناسنامه و عقدنامه",
			Services: []string{"dependency", "medical", "supplementary_insurance", "travel_card", "flight_approval", "petrokart", "naftcard"},
			Status:   domain.DependentApproved, DecidedBy: "hr.manager",
		},
		{
			ID: "DEP-0102", EmployeeID: "EMP-008", EmployeeName: "مریم نوری",
			DepartmentName: "منابع انسانی", FullName: "زهرا نوری", Relation: domain.RelationStudentChild,
			NationalID: "0119988771", BirthDate: day(-8200), CoverageStart: day(-700),
			Insurance: true, DocumentsFa: "شناسنامه و گواهی اشتغال به تحصیل",
			Services: []string{"dependency", "medical", "supplementary_insurance", "travel_card"},
			Status:   domain.DependentApproved, DecidedBy: "hr.manager",
			NoteFa: "پوشش تا پایان تحصیل؛ تمدید سالانهٔ گواهی لازم است",
		},
		{
			ID: "DEP-0103", EmployeeID: "EMP-011", EmployeeName: "امیر قاسمی",
			DepartmentName: "فناوری اطلاعات", FullName: "امیرعلی کریمی", Relation: domain.RelationChild,
			NationalID: "0069123454", BirthDate: day(-1500), CoverageStart: day(20),
			Insurance: false, DocumentsFa: "شناسنامه",
			Services: []string{"dependency", "medical"},
			Status:   domain.DependentPending,
			NoteFa:   "در انتظار کنترل مدرک هویتی",
		},
		{
			ID: "DEP-0104", EmployeeID: "EMP-010", EmployeeName: "فاطمه صادقی",
			DepartmentName: "فناوری اطلاعات", FullName: "لیلا صادقی", Relation: domain.RelationParent,
			NationalID: "1270456784", CoverageStart: day(-400),
			Insurance: true, DocumentsFa: "شناسنامه و تعهد کفالت",
			Services: []string{"dependency", "medical", "supplementary_insurance"},
			Status:   domain.DependentApproved, DecidedBy: "hr.spec",
		},
		{
			ID: "DEP-0105", EmployeeID: "EMP-005", EmployeeName: "محمد احمدی",
			DepartmentName: "فروش و بازاریابی", FullName: "سارا احمدی", Relation: domain.RelationChild,
			NationalID: "0087654326", CoverageStart: day(-3000),
			Insurance: false, Status: domain.DependentRemoved, DecidedBy: "hr.manager",
			NoteFa: "خروج از تکفل به دلیل اشتغال و پوشش بیمهٔ مستقل",
		},
		{
			ID: "DEP-0106", EmployeeID: "EMP-005", EmployeeName: "محمد احمدی",
			DepartmentName: "فروش و بازاریابی", FullName: "علی احمدی", Relation: domain.RelationChild,
			NationalID: "0084512376", BirthDate: day(-6200), CoverageStart: day(-2400),
			Insurance: true, DocumentsFa: "شناسنامه و کارنامهٔ تحصیلی",
			Services: []string{"dependency", "medical", "supplementary_insurance", "travel_card", "flight_approval", "petrokart", "naftcard"},
			Status:   domain.DependentApproved, DecidedBy: "hr.manager",
			NoteFa: "فرزند ممتاز؛ مشمول همهٔ خدمات و کارت‌های رفاهی",
		},
		{
			ID: "DEP-0107", EmployeeID: "EMP-005", EmployeeName: "محمد احمدی",
			DepartmentName: "فروش و بازاریابی", FullName: "زهرا احمدی", Relation: domain.RelationChild,
			NationalID: "0085632174", BirthDate: day(-3200), CoverageStart: day(-1500),
			Insurance: true, DocumentsFa: "شناسنامه",
			Services: []string{"dependency", "medical", "supplementary_insurance"},
			Status:   domain.DependentApproved, DecidedBy: "hr.manager",
			NoteFa: "تعلق پتروکارت و نفت‌کارت ثبت نشده است؛ در فایل شارژ شارژ نمی‌شود",
		},
		{
			ID: "DEP-0108", EmployeeID: "EMP-010", EmployeeName: "فاطمه صادقی",
			DepartmentName: "فناوری اطلاعات", FullName: "مهدی صادقی", Relation: domain.RelationChild,
			NationalID: "0068745123", BirthDate: day(-5100), CoverageStart: day(-2000),
			Insurance: true, DocumentsFa: "شناسنامه و کارنامهٔ تحصیلی",
			Services: []string{"dependency", "medical", "travel_card", "flight_approval", "petrokart", "naftcard"},
			Status:   domain.DependentApproved, DecidedBy: "hr.spec",
		},
		{
			ID: "DEP-0109", EmployeeID: "EMP-004", EmployeeName: "علی مرادی",
			DepartmentName: "مالی و حسابداری", FullName: "نگار مرادی", Relation: domain.RelationChild,
			NationalID: "0059874512", BirthDate: day(-4600), CoverageStart: day(-1700),
			Insurance: true, DocumentsFa: "شناسنامه",
			Services: []string{"dependency", "medical", "petrokart"},
			Status:   domain.DependentApproved, DecidedBy: "hr.manager",
			NoteFa: "تعلق بیمهٔ تکمیلی و نفت‌کارت ثبت نشده است",
		},
	}
}

// SeedFlights فرم‌های درخواست پرواز نمونه در همهٔ وضعیت‌های گردش‌کار.
func SeedFlights(today calendar.Date) []domain.FlightRequest {
	day := func(offset int) string {
		d, err := today.AddDays(offset)
		if err != nil {
			return today.String()
		}
		return d.String()
	}
	return []domain.FlightRequest{
		{
			ID: "FL-0184", EmployeeID: "EMP-005", EmployeeName: "محمد احمدی",
			DepartmentName: "فروش و بازاریابی", OriginFa: "ماهشهر", DestinationFa: "تهران",
			DepartDate: day(9), ReturnDate: day(12), Purpose: domain.PurposeMission, Passengers: 1,
			EstimatedCost: 18_500_000, Status: domain.FlightPending,
			NoteFa: "جلسهٔ قرارداد با مشتری اصلی",
		},
		{
			ID: "FL-0183", EmployeeID: "EMP-008", EmployeeName: "مریم نوری",
			DepartmentName: "منابع انسانی", OriginFa: "ماهشهر", DestinationFa: "مشهد",
			DepartDate: day(4), ReturnDate: day(9), Purpose: domain.PurposeFamily, Passengers: 3,
			CompanionsFa: "همراه با دو نفر تحت تکفل", EstimatedCost: 52_000_000,
			Status: domain.FlightApproved, DecidedBy: "hr.manager",
			NoteFa: "تأییدشده؛ در انتظار رزرو و صدور بلیت",
		},
		{
			ID: "FL-0182", EmployeeID: "EMP-011", EmployeeName: "امیر قاسمی",
			DepartmentName: "فناوری اطلاعات", OriginFa: "تهران", DestinationFa: "ماهشهر",
			DepartDate: day(-6), Purpose: domain.PurposeMission, Passengers: 1,
			EstimatedCost: 16_200_000, Status: domain.FlightSettled, DecidedBy: "hr.manager",
			TicketNo: "IR-7421", NoteFa: "بازگشت از مأموریت آموزشی؛ تسویه انجام شد",
		},
		{
			ID: "FL-0181", EmployeeID: "EMP-010", EmployeeName: "فاطمه صادقی",
			DepartmentName: "فناوری اطلاعات", OriginFa: "ماهشهر", DestinationFa: "اصفهان",
			DepartDate: day(-14), Purpose: domain.PurposeLeave, Passengers: 1,
			EstimatedCost: 14_800_000, Status: domain.FlightCancelled, DecidedBy: "hr.spec",
			NoteFa: "لغو به دلیل تغییر برنامهٔ مأموریت",
		},
		{
			ID: "FL-0185", EmployeeID: "EMP-003", EmployeeName: "نگار حسینی",
			DepartmentName: "فروش و بازاریابی", OriginFa: "ماهشهر", DestinationFa: "شیراز",
			DepartDate: day(21), ReturnDate: day(23), Purpose: domain.PurposeMission, Passengers: 1,
			EstimatedCost: 21_400_000, Status: domain.FlightIssued, DecidedBy: "hr.manager",
			TicketNo: "IR-8813", NoteFa: "نمایشگاه صنعت؛ بلیت صادر شد",
		},
	}
}
