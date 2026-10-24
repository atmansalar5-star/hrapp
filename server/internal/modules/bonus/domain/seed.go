package domain

import "hrms/internal/contracts"

// Seed رکوردهای نمونهٔ پرداخت پاداش.
//
// چیدمان نمونه عمدی است: هر هشت مدیریت بندر امام دست‌کم یک پرداخت دارند، چند نفر در
// یک مدیریت چند پرداخت پیاپی دارند (تا «تعداد دفعات پرداخت به هر نفر» معنا پیدا کند)،
// و هر چهار وضعیت گردش‌کار (پیش‌نویس، تأییدشده، پرداخت‌شده، لغوشده) در داده هست تا
// داشبورد و کارتابل روی دادهٔ واقعی معنا داشته باشند.
//
// SeedStatus فقط در دادهٔ نمونه اعمال می‌شود و از API قابل تنظیم نیست.
func Seed() []contracts.RecordInput {
	return []contracts.RecordInput{
		// — مدیریت بندر امام —
		bonusPay("EMP-002", "bandar_imam", "performance", "35000000", "1404/04/20", "1", "1404/م/0112", "bank_transfer", "پاداش عملکرد بهار", "paid"),
		bonusPay("EMP-002", "bandar_imam", "performance", "35000000", "1404/07/20", "2", "1404/م/0148", "bank_transfer", "پاداش عملکرد تابستان", "paid"),
		bonusPay("EMP-004", "bandar_imam", "production", "25000000", "1404/05/10", "1", "1404/م/0131", "payroll", "پاداش تحقق تولید", "paid"),

		// — مدیریت پشتیبانی تعمیرات —
		bonusPay("EMP-007", "repair_support", "production", "25000000", "1404/03/25", "1", "1404/م/0095", "payroll", "پاداش تعمیرات اساسی", "paid"),
		bonusPay("EMP-007", "repair_support", "production", "30000000", "1404/06/25", "2", "1404/م/0140", "bank_transfer", "پاداش توقف اضطراری", "approved"),
		bonusPay("EMP-009", "repair_support", "project", "40000000", "1404/06/15", "1", "1404/م/0137", "bank_transfer", "پاداش پروژهٔ اورهال", "paid"),

		// — آب و نیرو —
		bonusPay("EMP-010", "water_power", "occasional", "20000000", "1404/06/01", "1", "1404/م/0135", "gift_card", "پاداش مناسبتی روز کارگر", "paid"),

		// — بسپاران —
		bonusPay("EMP-003", "basparan", "occasional", "20000000", "1404/03/01", "1", "1404/م/0088", "bank_transfer", "پاداش مناسبتی نوروز", "paid"),
		bonusPay("EMP-012", "basparan", "retention", "15000000", "1404/07/05", "1", "1404/م/0145", "bank_transfer", "پاداش نگهداشت نیرو", "draft"),

		// — فراورش یک —
		bonusPay("EMP-005", "process_one", "performance", "35000000", "1404/03/20", "1", "1404/م/0093", "bank_transfer", "پاداش عملکرد زمستان", "paid"),
		bonusPay("EMP-005", "process_one", "performance", "35000000", "1404/05/20", "2", "1404/م/0133", "bank_transfer", "پاداش عملکرد بهار", "paid"),
		bonusPay("EMP-005", "process_one", "special", "60000000", "1404/07/15", "3", "1404/م/0147", "bank_transfer", "پاداش ویژهٔ بهبود فرایند", "approved"),

		// — فراورش دو —
		bonusPay("EMP-006", "process_two", "performance", "35000000", "1404/04/20", "1", "1404/م/0113", "payroll", "پاداش عملکرد بهار", "paid"),
		bonusPay("EMP-006", "process_two", "occasional", "20000000", "1404/07/01", "2", "1404/م/0144", "gift_card", "پاداش مناسبتی جشن پایان نیم‌سال", "paid"),
		bonusPay("EMP-011", "process_two", "retention", "15000000", "1404/08/01", "1", "1404/م/0152", "bank_transfer", "پاداش نگهداشت نیروی متخصص", "draft"),

		// — معاونت پشتیبانی خدمات —
		bonusPay("EMP-001", "services_support", "occasional", "20000000", "1404/03/01", "1", "1404/م/0087", "bank_transfer", "پاداش مناسبتی نوروز", "paid"),
		bonusPay("EMP-001", "services_support", "performance", "40000000", "1404/06/20", "2", "1404/م/0139", "bank_transfer", "پاداش عملکرد نیم‌سال اول", "paid"),
		bonusPay("EMP-008", "services_support", "project", "40000000", "1404/05/15", "1", "1404/م/0132", "bank_transfer", "پاداش استقرار سامانه", "paid"),

		// — کیمیا —
		bonusPay("EMP-009", "kimia", "project", "40000000", "1404/07/25", "1", "1404/م/0149", "bank_transfer", "پاداش پروژهٔ کیمیا", "approved"),

		// پرداخت لغوشده: باید در جمع‌ها نیاید و فقط در برش وضعیت شمرده شود.
		bonusPay("EMP-010", "water_power", "occasional", "20000000", "1404/04/05", "1", "1404/م/0104", "gift_card", "لغو شد — ابطال مصوبه", "cancelled"),
	}
}

// bonusPay یک رکورد نمونهٔ پرداخت پاداش می‌سازد تا چیدمان نمونه خوانا و بی‌تکرار بماند.
func bonusPay(employeeID, management, bonusType, amount, date, sequence, decreeNo, method, reference, status string) contracts.RecordInput {
	return contracts.RecordInput{
		Kind:       "bonusPayments",
		Title:      BonusTypeTitleFa(bonusType) + " — " + contracts.ManagementTitleFa(management),
		EmployeeID: employeeID,
		SeedStatus: status,
		Fields: map[string]string{
			"bonusType":     bonusType,
			"management":    management,
			"amount":        amount,
			"paymentDate":   date,
			"sequence":      sequence,
			"decreeNo":      decreeNo,
			"paymentMethod": method,
			"reference":     reference,
		},
	}
}
