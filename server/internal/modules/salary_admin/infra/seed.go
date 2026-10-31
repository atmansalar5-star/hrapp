// دادهٔ اولیهٔ (seed) ماژول «salary_admin» برای محیط توسعه و آزمون.

package infra

import (
	"hrms/internal/contracts"
	"hrms/internal/modules/salary_admin/domain"
	"hrms/internal/platform/calendar"
)

// SeedDecrees احکام حقوق نمونه در همهٔ وضعیت‌های گردشکار — همخوان با کارکنان نمونه.
// تاریخ اجرا نسبت به امروز ساخته می‌شود تا نمونهٔ نمایشی همیشه معنادار بماند.
//
// هر حکم اجزای مزد و منطقهٔ اقلیمی خودش را دارد: نمونهٔ نمایشی باید همان چیزی را نشان
// دهد که در عمل ثبت می‌شود (مزد مبنا از اجزای حکم + فوق‌العاده‌های اقلیمی همان منطقه).
func SeedDecrees(today calendar.Date) []domain.SalaryDecree {
	day := func(offset int) string {
		d, err := today.AddDays(offset)
		if err != nil {
			return today.String()
		}
		return d.String()
	}
	return []domain.SalaryDecree{
		{
			ID: "DEC-1404-0214", EmployeeID: "EMP-021", EmployeeName: "رضا کریمی",
			EmploymentType: "official_worker",
			DepartmentName: "تولید و عملیات", Position: "کارشناس ارشد تولید",
			JobGroup: "کارشناس ارشد", BaseSalary: 318_000_000, EffectiveDate: day(-5),
			ReasonFa: "ترفیع گروه شغلی", Status: domain.DecreeIssued,
			CreatedBy: "hr.spec", NoteFa: "در انتظار ابلاغ به کارمند",
			WageStructure: contracts.WageStructure{
				ClimateRegion: "south_coast",
				RankWage:      28_000_000, SeniorityWage: 34_000_000,
				PostAllowance: 12_000_000, ClimateAllowance: 18_000_000,
				BadClimatePercent: 0.25, DeprivationPercent: 0.25,
				RegionHardshipPercent: 0.30, ExtraTimePercent: 0.15,
			},
		},
		{
			ID: "DEC-1404-0213", EmployeeID: "EMP-005", EmployeeName: "محمد احمدی",
			EmploymentType: "fixed_1662",
			DepartmentName: "فروش و بازاریابی", Position: "مدیر فروش",
			JobGroup: "مدیریت", BaseSalary: 246_000_000, EffectiveDate: day(-40),
			ReasonFa: "افزایش سنواتی", Status: domain.DecreeNotified,
			CreatedBy: "hr.spec", DecidedBy: "hr.manager",
			WageStructure: contracts.WageStructure{
				ClimateRegion: "south_coast",
				RankWage:      42_000_000, SeniorityWage: 31_000_000,
				PostAllowance: 20_000_000, SacrificeAllowance: 5_000_000,
				ClimateAllowance: 22_000_000,
				// ضرایب اقلیمی این کارمند بالاتر از پیش‌فرض شرکت است (منطقهٔ عملیاتی).
				BadClimatePercent: 0.30, RegionHardshipPercent: 0.35,
			},
		},
		{
			ID: "DEC-1404-0212", EmployeeID: "EMP-011", EmployeeName: "امیر قاسمی",
			EmploymentType: "long_term",
			DepartmentName: "فناوری اطلاعات", Position: "کارشناس شبکه",
			JobGroup: "تکنسین ارشد", BaseSalary: 198_000_000, EffectiveDate: day(-40),
			ReasonFa: "تمدید قرارداد", Status: domain.DecreeNotified,
			CreatedBy: "hr.spec", DecidedBy: "hr.manager",
			WageStructure: contracts.WageStructure{
				ClimateRegion: "central",
				RankWage:      16_000_000, SeniorityWage: 12_000_000,
				PostAllowance: 8_000_000,
			},
		},
		{
			ID: "DEC-1404-0211", EmployeeID: "EMP-032", EmployeeName: "سعید مرادی",
			EmploymentType: "contractor_project",
			DepartmentName: "پشتیبانی", Position: "کارشناس پشتیبانی",
			JobGroup: "کارشناس", BaseSalary: 231_000_000, EffectiveDate: day(-2),
			ReasonFa: "اصلاح مأخذ پرداخت", Status: domain.DecreeDraft,
			CreatedBy: "hr.spec", NoteFa: "نیازمند بازنگری گروه شغلی پیش از صدور",
			WageStructure: contracts.WageStructure{
				ClimateRegion: "desert",
				RankWage:      19_000_000, SeniorityWage: 14_000_000,
				ClimateAllowance: 15_000_000,
			},
		},
		{
			ID: "DEC-1403-0198", EmployeeID: "EMP-005", EmployeeName: "محمد احمدی",
			EmploymentType: "fixed_1662",
			DepartmentName: "فروش و بازاریابی", Position: "مدیر فروش",
			JobGroup: "مدیریت", BaseSalary: 231_000_000, EffectiveDate: day(-400),
			ReasonFa: "افزایش سنواتی سالانه", Status: domain.DecreeSuperseded,
			CreatedBy: "hr.spec", DecidedBy: "hr.manager",
			NoteFa: "با صدور حکم DEC-1404-0213 منسوخ شد",
			WageStructure: contracts.WageStructure{
				ClimateRegion: "south_coast",
				RankWage:      38_000_000, SeniorityWage: 27_000_000,
				PostAllowance: 18_000_000, ClimateAllowance: 20_000_000,
			},
		},
	}
}

// SeedCertificates گواهی‌های نمونه در همهٔ وضعیت‌های گردشکار.
func SeedCertificates(today calendar.Date) []domain.EmploymentCertificate {
	day := func(offset int) string {
		d, err := today.AddDays(offset)
		if err != nil {
			return today.String()
		}
		return d.String()
	}
	return []domain.EmploymentCertificate{
		{
			ID: "CRT-0312", Serial: "CRT-1404-0312", EmployeeID: "EMP-005", EmployeeName: "محمد احمدی",
			DepartmentName: "فروش و بازاریابی", Position: "مدیر فروش",
			Kind: domain.CertBank, PurposeFa: "بانک ملت — افتتاح حساب",
			RequestedAt: day(-12), IssuedAt: day(-11), DeliveredAt: day(-10),
			Status: domain.CertDelivered, DecidedBy: "hr.manager",
		},
		{
			ID: "CRT-0311", Serial: "CRT-1404-0311", EmployeeID: "EMP-008", EmployeeName: "مریم نوری",
			DepartmentName: "منابع انسانی", Position: "کارشناس منابع انسانی",
			Kind: domain.CertEmbassy, PurposeFa: "سفارت آلمان — درخواست ویزا",
			RequestedAt: day(-9), IssuedAt: day(-8),
			Status: domain.CertIssued, DecidedBy: "hr.manager",
		},
		{
			ID: "CRT-0310", Serial: "CRT-1404-0310", EmployeeID: "EMP-011", EmployeeName: "امیر قاسمی",
			DepartmentName: "فناوری اطلاعات", Position: "کارشناس شبکه",
			Kind: domain.CertSimple, PurposeFa: "ارائه به مراجع ذی‌ربط",
			RequestedAt: day(-3), Status: domain.CertRequested,
		},
		{
			ID: "CRT-0309", Serial: "CRT-1404-0309", EmployeeID: "EMP-032", EmployeeName: "سعید مرادی",
			DepartmentName: "پشتیبانی", Position: "کارشناس پشتیبانی",
			Kind: domain.CertBank, PurposeFa: "بانک صادرات — تسهیلات",
			RequestedAt: day(-6), Status: domain.CertApproved, DecidedBy: "hr.manager",
		},
	}
}
