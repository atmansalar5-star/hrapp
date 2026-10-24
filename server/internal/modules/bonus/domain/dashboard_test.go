// آزمون دامنهٔ ماژول «bonus» (dashboard_test).

package domain

import (
	"testing"

	"hrms/internal/contracts"
)

// TestBonusCatalogIsComplete تضمین می‌کند کاتالوگ انواع پاداش و روش‌های پرداخت کامل و
// بی‌تکرار است: هر نوع پاداش در سازمان مصوبه‌محور است، پس همه باید RequiresDecree
// داشته باشند و مبلغ پیشنهادی مثبت باشد. اگر این‌ها به‌هم بریزد، فرم تخصیص و گزارش‌ها
// بی‌سروصدا غلط می‌شوند.
func TestBonusCatalogIsComplete(t *testing.T) {
	types := BonusTypes()
	if len(types) != 6 {
		t.Fatalf("باید ۶ نوع پاداش باشد، %d شد", len(types))
	}
	seen := map[string]bool{}
	for _, b := range types {
		if b.Key == "" || b.TitleFa == "" {
			t.Fatalf("نوع پاداش باید کلید و عنوان فارسی داشته باشد: %+v", b)
		}
		if seen[b.Key] {
			t.Fatalf("کلید نوع پاداش تکراری است: %s", b.Key)
		}
		seen[b.Key] = true
		if b.DefaultAmount <= 0 {
			t.Fatalf("مبلغ پیشنهادی نوع %s باید مثبت باشد", b.Key)
		}
		if !b.RequiresDecree {
			t.Fatalf("هر پرداخت پاداش مصوبه‌محور است؛ %s بدون الزام مصوبه تعریف شده", b.Key)
		}
	}

	if methods := PaymentMethods(); len(methods) != 4 {
		t.Fatalf("باید ۴ روش پرداخت باشد، %d شد", len(methods))
	}
	if !ValidPaymentMethod("") {
		t.Fatal("روش پرداخت خالی باید مجاز باشد (اختیاری است)")
	}
	if ValidPaymentMethod("bitcoin") {
		t.Fatal("روش پرداخت ناشناخته نباید مجاز باشد")
	}
	if !ValidBonusType("occasional") || ValidBonusType("unknown") {
		t.Fatal("اعتبارسنجی نوع پاداش درست کار نمی‌کند")
	}
	if BonusTypeTitleFa("performance") != "پاداش عملکرد" {
		t.Fatalf("عنوان نوع پاداش درست نیست: %s", BonusTypeTitleFa("performance"))
	}
}

// TestBuildDashboardSplitsByManagementAndPerson همان چیزی را می‌سنجد که داشبورد مدیریتی
// وعده می‌دهد: «مبلغ و تعداد دفعات پرداختی به هر نفر در هر مدیریت» و جمع هر مدیریت.
// قاعده‌های کلیدی: هر هشت مدیریت حتی بدون پرداخت حاضرند، و «دفعات» شمارش رویدادهاست نه
// نفرات.
func TestBuildDashboardSplitsByManagementAndPerson(t *testing.T) {
	facts := []PaymentFact{
		{EmployeeID: "EMP-1", EmployeeName: "الف", Management: "bandar_imam", BonusType: "occasional",
			Amount: 20_000_000, Date: "1404/07/10", Period: PeriodOf("1404/07/10"), Status: "paid", Group: EmploymentGroupCompany},
		{EmployeeID: "EMP-1", EmployeeName: "الف", Management: "bandar_imam", BonusType: "occasional",
			Amount: 30_000_000, Date: "1404/08/10", Period: PeriodOf("1404/08/10"), Status: "paid", Group: EmploymentGroupCompany},
		{EmployeeID: "EMP-2", EmployeeName: "ب", Management: "bandar_imam", BonusType: "performance",
			Amount: 20_000_000, Date: "1404/07/15", Period: PeriodOf("1404/07/15"), Status: "approved", Group: EmploymentGroupThirdParty},
		{EmployeeID: "EMP-3", EmployeeName: "پ", Management: "kimia", BonusType: "project",
			Amount: 40_000_000, Date: "1404/07/20", Period: PeriodOf("1404/07/20"), Status: "cancelled", Group: EmploymentGroupCompany},
	}
	d := BuildDashboard(facts)

	if len(d.ManagementRows) != 8 {
		t.Fatalf("هر ۸ مدیریت باید در داشبورد حاضر باشند، %d شد", len(d.ManagementRows))
	}
	if d.ManagementsTotal != 8 || d.ManagementsCovered != 1 {
		t.Fatalf("پوشش مدیریت‌ها غلط است: covered=%d total=%d", d.ManagementsCovered, d.ManagementsTotal)
	}

	row := managementRow(t, d, "bandar_imam")
	if row.Payments != 3 || row.Beneficiaries != 2 {
		t.Fatalf("ردیف بندر امام باید ۳ پرداخت به ۲ نفر باشد: %+v", row)
	}
	if row.TotalAmount != 70_000_000 {
		t.Fatalf("جمع مبلغ بندر امام باید ۷۰٬۰۰۰٬۰۰۰ باشد: %v", row.TotalAmount)
	}
	if row.AveragePerPerson != 35_000_000 {
		t.Fatalf("میانگین هر نفر بندر امام باید ۳۵٬۰۰۰٬۰۰۰ باشد: %v", row.AveragePerPerson)
	}
	if row.MaxPayment != 30_000_000 {
		t.Fatalf("بیشترین پرداخت بندر امام باید ۳۰٬۰۰۰٬۰۰۰ باشد: %v", row.MaxPayment)
	}
	if row.CompanyCount != 2 || row.ThirdPartyCount != 1 {
		t.Fatalf("تفکیک شرکتی/ارکان ثالث بندر امام غلط است: %+v", row)
	}
	if row.FirstDate != "1404/07/10" || row.LastDate != "1404/08/10" {
		t.Fatalf("بازهٔ پرداخت بندر امام غلط است: %s … %s", row.FirstDate, row.LastDate)
	}

	// مدیریت بدون پرداخت هم ردیف دارد، ولی صفر.
	if empty := managementRow(t, d, "kimia"); empty.Payments != 0 || empty.TotalAmount != 0 {
		t.Fatalf("کیمیا باید ردیف صفر داشته باشد (تنها پرداختش لغو شده): %+v", empty)
	}

	// «مبلغ و تعداد دفعات پرداختی به هر نفر در هر مدیریت».
	if len(d.EmployeeRows) != 2 {
		t.Fatalf("باید ۲ ردیف نفر-مدیریت باشد، %d شد", len(d.EmployeeRows))
	}
	emp := employeeRow(t, d, "EMP-1", "bandar_imam")
	if emp.Payments != 2 || emp.TotalAmount != 50_000_000 || emp.AveragePayment != 25_000_000 {
		t.Fatalf("ردیف EMP-1 در بندر امام غلط است: %+v", emp)
	}
	if len(emp.TypeTitlesFa) != 1 || emp.TypeTitlesFa[0] != "پاداش مناسبتی" {
		t.Fatalf("انواع پاداش EMP-1 غلط است: %v", emp.TypeTitlesFa)
	}
	if emp.ManagementFa != "بندر امام" {
		t.Fatalf("عنوان فارسی مدیریت در ردیف نفر غلط است: %s", emp.ManagementFa)
	}
	for _, e := range d.EmployeeRows {
		if e.EmployeeID == "EMP-3" {
			t.Fatal("پرداخت لغوشده نباید ردیف نفر بسازد")
		}
	}
}

// TestBuildDashboardExcludesCancelledFromSums ولیCountsThemInStatus تضمین می‌کند پرداخت
// لغوشده در هیچ جمعی نمی‌آید، اما در برش وضعیت دیده می‌شود تا کارتابل و حسابرسی گم نشود.
func TestBuildDashboardExcludesCancelledFromSums(t *testing.T) {
	facts := []PaymentFact{
		{EmployeeID: "EMP-1", Management: "bandar_imam", BonusType: "occasional", Amount: 20_000_000, Date: "1404/07/10", Status: "paid", Group: EmploymentGroupCompany},
		{EmployeeID: "EMP-2", Management: "bandar_imam", BonusType: "performance", Amount: 35_000_000, Date: "1404/07/12", Status: "draft", Group: EmploymentGroupCompany},
		{EmployeeID: "EMP-3", Management: "kimia", BonusType: "project", Amount: 99_000_000, Date: "1404/07/20", Status: "cancelled", Group: EmploymentGroupCompany},
	}
	d := BuildDashboard(facts)

	if d.TotalPayments != 2 || d.TotalAmount != 55_000_000 {
		t.Fatalf("پرداخت لغوشده نباید در جمع‌ها بیاید: payments=%d amount=%v", d.TotalPayments, d.TotalAmount)
	}
	if d.PaidPayments != 1 || d.DraftPayments != 1 || d.ApprovedPayments != 0 {
		t.Fatalf("تفکیک وضعیت غلط است: paid=%d draft=%d approved=%d", d.PaidPayments, d.DraftPayments, d.ApprovedPayments)
	}
	if d.CancelledPayments != 1 {
		t.Fatalf("لغوشده باید شمرده شود: %d", d.CancelledPayments)
	}
	if d.TotalBeneficiaries != 2 {
		t.Fatalf("نفرات بهره‌مند باید ۲ باشد: %d", d.TotalBeneficiaries)
	}
	if d.MaxPayment != 35_000_000 || d.MinPayment != 20_000_000 {
		t.Fatalf("کمینه/بیشینه نباید لغوشده را ببیند: max=%v min=%v", d.MaxPayment, d.MinPayment)
	}
	if d.AveragePerPayment != 27_500_000 || d.AveragePerPerson != 27_500_000 {
		t.Fatalf("میانگین‌ها غلط است: perPayment=%v perPerson=%v", d.AveragePerPayment, d.AveragePerPerson)
	}

	// برش وضعیت همهٔ وضعیت‌های گردش‌کار را نشان می‌دهد، حتی صفرها.
	if len(d.StatusRows) != len(BonusStatuses()) {
		t.Fatalf("برش وضعیت باید همهٔ وضعیت‌ها را داشته باشد: %d", len(d.StatusRows))
	}
	cancelledFound := false
	for _, s := range d.StatusRows {
		if s.StatusKey == "cancelled" && s.Payments == 1 && s.StatusFa != "" {
			cancelledFound = true
		}
	}
	if !cancelledFound {
		t.Fatal("وضعیت لغوشده در برش وضعیت پیدا نشد")
	}
}

// TestBuildDashboardTypeSharesAndPeriods سهم انواع پاداش (که باید روی ۱۰۰٪ جمع شود) و
// روند دوره‌ای را می‌سنجد؛ این‌ها مبنای دو نمودار داشبوردند.
func TestBuildDashboardTypeSharesAndPeriods(t *testing.T) {
	facts := []PaymentFact{
		{EmployeeID: "EMP-1", Management: "bandar_imam", BonusType: "occasional", Amount: 20_000_000, Date: "1404/07/10", Period: PeriodOf("1404/07/10"), Status: "paid", Group: EmploymentGroupCompany},
		{EmployeeID: "EMP-1", Management: "bandar_imam", BonusType: "occasional", Amount: 30_000_000, Date: "1404/08/10", Period: PeriodOf("1404/08/10"), Status: "paid", Group: EmploymentGroupCompany},
		{EmployeeID: "EMP-2", Management: "bandar_imam", BonusType: "performance", Amount: 20_000_000, Date: "1404/07/15", Period: PeriodOf("1404/07/15"), Status: "approved", Group: EmploymentGroupThirdParty},
	}
	d := BuildDashboard(facts)

	// همهٔ انواع کاتالوگ در گزارش حاضرند، حتی آن‌ها که پرداختی ندارند.
	if len(d.TypeRows) != len(BonusTypes()) {
		t.Fatalf("برش نوع باید همهٔ انواع کاتالوگ را داشته باشد: %d", len(d.TypeRows))
	}
	var shareSum float64
	byKey := map[string]float64{}
	for _, r := range d.TypeRows {
		shareSum += r.SharePercent
		byKey[r.TypeKey] = r.SharePercent
	}
	if byKey["occasional"] != 71.4 || byKey["performance"] != 28.6 {
		t.Fatalf("سهم انواع پاداش غلط است: occasional=%v performance=%v", byKey["occasional"], byKey["performance"])
	}
	if shareSum < 99.9 || shareSum > 100.1 {
		t.Fatalf("مجموع سهم انواع باید حدود ۱۰۰٪ باشد: %v", shareSum)
	}

	if len(d.PeriodRows) != 2 {
		t.Fatalf("باید ۲ دوره باشد، %d شد", len(d.PeriodRows))
	}
	if d.PeriodRows[0].Period != "1404/07" || d.PeriodRows[0].Payments != 2 || d.PeriodRows[0].Beneficiaries != 2 || d.PeriodRows[0].Amount != 40_000_000 {
		t.Fatalf("دورهٔ ۱۴۰۴/۰۷ غلط است: %+v", d.PeriodRows[0])
	}
	if d.PeriodRows[1].Period != "1404/08" || d.PeriodRows[1].Payments != 1 || d.PeriodRows[1].Beneficiaries != 1 {
		t.Fatalf("دورهٔ ۱۴۰۴/۰۸ غلط است: %+v", d.PeriodRows[1])
	}

	if d.CompanyAmount != 50_000_000 || d.ThirdPartyAmount != 20_000_000 {
		t.Fatalf("ترکیب شرکتی/ارکان ثالث غلط است: company=%v thirdParty=%v", d.CompanyAmount, d.ThirdPartyAmount)
	}
}

// TestBuildDashboardEmptyIsSafe تضمین می‌کند داشبورد روی دادهٔ خالی خراب نمی‌شود (تقسیم
// بر صفر و ردیف nil)، چون هر نصب تازه همین وضعیت را دارد.
func TestBuildDashboardEmptyIsSafe(t *testing.T) {
	d := BuildDashboard(nil)
	if d.TotalPayments != 0 || d.TotalAmount != 0 || d.AveragePerPayment != 0 || d.AveragePerPerson != 0 {
		t.Fatalf("داشبورد خالی باید همه‌جا صفر باشد: %+v", d)
	}
	if len(d.ManagementRows) != 8 {
		t.Fatalf("ردیف مدیریت‌ها حتی روی دادهٔ خالی لازم است: %d", len(d.ManagementRows))
	}
	if d.ManagementsCovered != 0 || d.TotalBeneficiaries != 0 {
		t.Fatalf("پوشش روی دادهٔ خالی باید صفر باشد: covered=%d beneficiaries=%d", d.ManagementsCovered, d.TotalBeneficiaries)
	}
}

// TestAmountAndPeriodParsing ارقام فارسی و جداکنندهٔ هزارگان را می‌سنجد؛ فرم و فایل
// واردشده این‌ها را تولید می‌کند و اشتباه در تفسیر مبلغ، خطای پولی است.
func TestAmountAndPeriodParsing(t *testing.T) {
	cases := []struct {
		in   string
		want int64
		ok   bool
	}{
		{"20000000", 20_000_000, true},
		{"۲۰٬۰۰۰٬۰۰۰", 20_000_000, true},
		{"۲۰,۰۰۰,۰۰۰", 20_000_000, true},
		{"", 0, false},
		{"abc", 0, false},
	}
	for _, c := range cases {
		got, ok := AmountOf(c.in)
		if ok != c.ok || (ok && got != c.want) {
			t.Fatalf("AmountOf(%q) = (%d, %v)، انتظار (%d, %v)", c.in, got, ok, c.want, c.ok)
		}
	}

	if PeriodOf("1404/07/20") != "1404/07" {
		t.Fatalf("PeriodOf درست کار نمی‌کند: %s", PeriodOf("1404/07/20"))
	}
	if PeriodOf("۱۴۰۴/۰۷/۲۰") != "1404/07" {
		t.Fatalf("PeriodOf باید ارقام فارسی را هم بفهمد: %s", PeriodOf("۱۴۰۴/۰۷/۲۰"))
	}
	if PeriodOf("1404") != "" {
		t.Fatalf("تاریخ ناقص باید دورهٔ خالی بدهد: %s", PeriodOf("1404"))
	}
	if SequenceOf("") != 0 || SequenceOf("۳") != 3 {
		t.Fatalf("SequenceOf غلط است: %d", SequenceOf("۳"))
	}
}

// TestValidateBonusPaymentEnforcesDomainRules تضمین می‌کند قواعد پاداش در دامنه و پیش از
// ساخت هر رکورد اعمال می‌شوند: نوع/مدیریت معتبر، مبلغ مثبت، تاریخ شمسی و — مهم‌تر از
// همه — شمارهٔ مصوبه برای پرداخت مصوبه‌محور.
func TestValidateBonusPaymentEnforcesDomainRules(t *testing.T) {
	valid := contracts.RecordInput{Fields: map[string]string{
		"bonusType": "occasional", "management": "kimia", "amount": "20000000",
		"paymentDate": "1404/09/10", "paymentMethod": "bank_transfer", "decreeNo": "1404/م/0151",
	}}
	if err := validateBonusPayment(valid); err != nil {
		t.Fatalf("ورودی معتبر نباید خطا بدهد: %v", err)
	}

	bad := []struct {
		name   string
		mutate func(m map[string]string)
	}{
		{"نوع پاداش ناشناخته", func(m map[string]string) { m["bonusType"] = "unknown" }},
		{"نوع پاداش خالی", func(m map[string]string) { m["bonusType"] = "" }},
		{"مدیریت ناشناخته", func(m map[string]string) { m["management"] = "tehran" }},
		{"مبلغ صفر", func(m map[string]string) { m["amount"] = "0" }},
		{"شمارهٔ ماه نامعتبر", func(m map[string]string) { m["paymentDate"] = "1404/13/01" }},
		{"روز نامعتبر برای ماه", func(m map[string]string) { m["paymentDate"] = "1404/12/31" }},
		{"تاریخ ناقص", func(m map[string]string) { m["paymentDate"] = "1404/07" }},
		{"تاریخ غیرعددی", func(m map[string]string) { m["paymentDate"] = "not-a-date" }},
		{"روش پرداخت نامجاز", func(m map[string]string) { m["paymentMethod"] = "crypto" }},
		{"مصوبهٔ خالی", func(m map[string]string) { m["decreeNo"] = "" }},
	}
	for _, c := range bad {
		fields := map[string]string{}
		for k, v := range valid.Fields {
			fields[k] = v
		}
		c.mutate(fields)
		if err := validateBonusPayment(contracts.RecordInput{Fields: fields}); err == nil {
			t.Fatalf("«%s» باید رد می‌شد ولی رد نشد", c.name)
		}
	}
}

func managementRow(t *testing.T, d contracts.BonusDashboard, key string) contracts.BonusManagementRow {
	t.Helper()
	for _, r := range d.ManagementRows {
		if r.ManagementKey == key {
			return r
		}
	}
	t.Fatalf("ردیف مدیریت %s در داشبورد نیست", key)
	return contracts.BonusManagementRow{}
}

func employeeRow(t *testing.T, d contracts.BonusDashboard, empID, management string) contracts.BonusEmployeeRow {
	t.Helper()
	for _, r := range d.EmployeeRows {
		if r.EmployeeID == empID && r.ManagementKey == management {
			return r
		}
	}
	t.Fatalf("ردیف نفر %s در مدیریت %s پیدا نشد", empID, management)
	return contracts.BonusEmployeeRow{}
}
