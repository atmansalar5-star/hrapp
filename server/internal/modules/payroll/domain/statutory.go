// فایل قانونی بیمه و مالیات حقوق — زیرکار ۱.۴ فاز ۱ نقشهٔ راه.
//
// این فایل قواعد «ساخت و اعتبارسنجی فایل قانونی» را در دامنه نگه می‌دارد: ستون‌های هر
// فایل، سهم بیمه‌شده/کارفرما/بیکاری، مبنای مشمول مالیات، مهلت قانونی (پایان ماه بعد) و
// برآورد جریمهٔ دیرکرد. مبلغ‌ها از موتور قواعد (`rules.ComputePayroll`) می‌آید و این لایه
// فقط آن‌ها را به ردیف‌های فایل و جمع‌های کنترلی تبدیل می‌کند.
//
// نکتهٔ پارامتریک: نرخ‌ها و ضرایب از پارامترهای شرکت خوانده می‌شوند؛ اگر شرکت نرخ یا
// قالب دیگری داشته باشد، فقط پارامتر/قالب عوض می‌شود، نه کد.
package domain

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"hrms/internal/platform/calendar"
	"hrms/internal/platform/recordkit"
)

// واژگان نوع فایل قانونی.
const (
	StatutoryInsurance = "insurance"
	StatutoryTax       = "tax"
)

// StatutoryRates نرخ‌ها و سقف‌های فایل قانونی؛ همه از پارامترهای شرکت می‌آید.
type StatutoryRates struct {
	EmployeeRate      float64 `json:"employeeRate"`
	EmployerRate      float64 `json:"employerRate"`
	UnemploymentRate  float64 `json:"unemploymentRate"`
	Ceiling           int64   `json:"ceiling"`
	TaxExemption      int64   `json:"taxExemption"`
	InsurancePenalty  float64 `json:"insurancePenaltyMonthly"`
	TaxPenalty        float64 `json:"taxPenaltyMonthly"`
	WorkshopCode      string  `json:"workshopCode"`
	TaxFileNumber     string  `json:"taxFileNumber"`
	InsuranceBranchFa string  `json:"insuranceBranchFa"`
}

// DefaultStatutoryRates نرخ‌های قانونی رایج (کارگاه مشمول قانون کار): ۷٪ سهم بیمه‌شده،
// ۲۰٪ سهم کارفرما و ۳٪ بیمهٔ بیکاری؛ جریمهٔ دیرکرد بیمه ۲٪ و مالیات ۲٫۵٪ در ماه.
func DefaultStatutoryRates() StatutoryRates {
	return StatutoryRates{
		EmployeeRate:     0.07,
		EmployerRate:     0.20,
		UnemploymentRate: 0.03,
		InsurancePenalty: 0.02,
		TaxPenalty:       0.025,
	}
}

// StatutoryRow یک ردیف فایل قانونی (هر کارمند).
type StatutoryRow struct {
	RowNo          int    `json:"rowNo"`
	EmployeeID     string `json:"employeeId"`
	EmployeeName   string `json:"employeeName"`
	NationalID     string `json:"nationalId"`
	DepartmentName string `json:"departmentName,omitempty"`
	// Days روزهای مشمول در این ماه (۱ تا ۳۱).
	Days          int   `json:"days"`
	Gross         int64 `json:"gross"`
	InsurableWage int64 `json:"insurableWage"`
	EmployeeShare int64 `json:"employeeShare"`
	EmployerShare int64 `json:"employerShare"`
	Unemployment  int64 `json:"unemploymentShare"`
	InsuranceSum  int64 `json:"insuranceTotal"`
	TaxableIncome int64 `json:"taxableIncome"`
	TaxExemption  int64 `json:"taxExemption"`
	Tax           int64 `json:"tax"`
	Net           int64 `json:"net"`
	// IssuesFa مغایرت‌های همین ردیف (کد ملی نامعتبر، روز صفر، سقف بیمه و ...).
	IssuesFa []string `json:"issuesFa,omitempty"`
}

// StatutoryTotals جمع‌های کنترلی فایل.
type StatutoryTotals struct {
	Rows          int   `json:"rows"`
	Days          int   `json:"days"`
	Gross         int64 `json:"gross"`
	InsurableWage int64 `json:"insurableWage"`
	EmployeeShare int64 `json:"employeeShare"`
	EmployerShare int64 `json:"employerShare"`
	Unemployment  int64 `json:"unemploymentShare"`
	InsuranceSum  int64 `json:"insuranceTotal"`
	TaxableIncome int64 `json:"taxableIncome"`
	TaxExemption  int64 `json:"taxExemption"`
	Tax           int64 `json:"tax"`
	Net           int64 `json:"net"`
	CappedRows    int   `json:"cappedRows"`
}

// StatutoryFile فایل قانونی یک دوره: ردیف‌ها، جمع‌ها، ستون‌ها و وضعیت مهلت.
type StatutoryFile struct {
	Kind      string          `json:"kind"`
	TitleFa   string          `json:"titleFa"`
	Period    string          `json:"period"`
	ColumnsFa []string        `json:"columnsFa"`
	Rows      []StatutoryRow  `json:"rows"`
	Totals    StatutoryTotals `json:"totals"`
	IssuesFa  []string        `json:"issuesFa,omitempty"`
	// DeadlineFa آخرین روز مهلت قانونی (پایان ماه بعد) و DaysLeft روزهای باقی‌مانده.
	DeadlineFa string `json:"deadlineFa"`
	DaysLeft   int    `json:"daysLeft"`
	DaysLate   int    `json:"daysLate"`
	// PenaltyEstimateFa برآورد جریمهٔ دیرکرد برای مبلغ قابل پرداخت.
	Penalty      int64          `json:"penaltyEstimate"`
	PenaltyDays  int            `json:"penaltyDays"`
	PayableTotal int64          `json:"payableTotal"`
	Rates        StatutoryRates `json:"rates"`
	NoteFa       string         `json:"noteFa,omitempty"`
}

// StatutoryTitles عنوان فارسی هر نوع فایل.
func StatutoryTitles(kind string) (string, bool) {
	switch kind {
	case StatutoryInsurance:
		return "لیست بیمهٔ تأمین اجتماعی", true
	case StatutoryTax:
		return "لیست مالیات حقوق", true
	}
	return "", false
}

// StatutoryColumns ستون‌های فایل به همان ترتیبی که در خروجی CSV نوشته می‌شوند.
func StatutoryColumns(kind string) []string {
	if kind == StatutoryTax {
		return []string{
			"ردیف", "کد پرسنلی", "کد ملی", "نام و نام خانوادگی", "واحد سازمانی",
			"جمع حقوق و مزایا", "معافیت مالیاتی", "درآمد مشمول", "مالیات",
		}
	}
	return []string{
		"ردیف", "کد پرسنلی", "کد ملی", "نام و نام خانوادگی", "واحد سازمانی",
		"تعداد روز کارکرد", "دستمزد و مزایای مشمول", "سهم بیمه‌شده (۷٪)",
		"سهم کارفرما (۲۰٪)", "بیمهٔ بیکاری (۳٪)", "جمع سهم‌ها",
	}
}

// StatutoryRowValues مقادیر یک ردیف به ترتیب ستون‌های همان نوع فایل.
func StatutoryRowValues(kind string, row StatutoryRow) []string {
	base := []string{
		strconv.Itoa(row.RowNo), row.EmployeeID, row.NationalID, row.EmployeeName, row.DepartmentName,
	}
	if kind == StatutoryTax {
		return append(base,
			strconv.FormatInt(row.Gross, 10),
			strconv.FormatInt(row.TaxExemption, 10),
			strconv.FormatInt(row.TaxableIncome, 10),
			strconv.FormatInt(row.Tax, 10),
		)
	}
	return append(base,
		strconv.Itoa(row.Days),
		strconv.FormatInt(row.InsurableWage, 10),
		strconv.FormatInt(row.EmployeeShare, 10),
		strconv.FormatInt(row.EmployerShare, 10),
		strconv.FormatInt(row.Unemployment, 10),
		strconv.FormatInt(row.InsuranceSum, 10),
	)
}

// StatutoryFooter رديف‌های جمع پایان فایل (تا واحد مالی جمع را خودش حساب نکند).
func StatutoryFooter(file StatutoryFile) [][]string {
	label := "جمع کل"
	if file.Kind == StatutoryTax {
		return [][]string{
			{"", "", "", label, "", strconv.FormatInt(file.Totals.Gross, 10),
				strconv.FormatInt(file.Totals.TaxExemption, 10),
				strconv.FormatInt(file.Totals.TaxableIncome, 10),
				strconv.FormatInt(file.Totals.Tax, 10)},
		}
	}
	return [][]string{
		{"", "", "", label, "", strconv.Itoa(file.Totals.Days),
			strconv.FormatInt(file.Totals.InsurableWage, 10),
			strconv.FormatInt(file.Totals.EmployeeShare, 10),
			strconv.FormatInt(file.Totals.EmployerShare, 10),
			strconv.FormatInt(file.Totals.Unemployment, 10),
			strconv.FormatInt(file.Totals.InsuranceSum, 10)},
	}
}

// ValidateIranianNationalID کد ملی ایران را می‌سنجد: ده رقم، بدون رقم تکراری یکسان،
// و رقم کنترل درست. این قاعده در دامنه است چون هم فایل بیمه و هم فایل مالیات به آن
// وابسته‌اند و نباید در دو جا تکرار شود.
func ValidateIranianNationalID(raw string) error {
	code := strings.TrimSpace(calendar.NormalizeDigits(raw))
	if code == "" {
		return fmt.Errorf("کد ملی خالی است")
	}
	if len(code) != 10 {
		return fmt.Errorf("کد ملی باید ۱۰ رقم باشد («%s»)", code)
	}
	digits := make([]int, 10)
	allSame := true
	for i, ch := range code {
		if ch < '0' || ch > '9' {
			return fmt.Errorf("کد ملی باید فقط رقم باشد («%s»)", code)
		}
		digits[i] = int(ch - '0')
		if i > 0 && digits[i] != digits[0] {
			allSame = false
		}
	}
	if allSame {
		return fmt.Errorf("کد ملی «%s» معتبر نیست (همهٔ ارقام یکسان)", code)
	}
	sum := 0
	for i := 0; i < 9; i++ {
		sum += digits[i] * (10 - i)
	}
	remainder := sum % 11
	check := digits[9]
	if remainder < 2 {
		if check != remainder {
			return fmt.Errorf("رقم کنترل کد ملی «%s» درست نیست", code)
		}
		return nil
	}
	if check != 11-remainder {
		return fmt.Errorf("رقم کنترل کد ملی «%s» درست نیست", code)
	}
	return nil
}

// StatutoryInput ورودی ساخت یک ردیف فایل قانونی از کارکرد و مبلغ محاسبه‌شده.
type StatutoryInput struct {
	EmployeeID     string
	EmployeeName   string
	NationalID     string
	DepartmentName string
	Days           int
	Gross          int64
	TaxExemption   int64
	TaxableIncome  int64
	Tax            int64
	Net            int64
}

// BuildStatutoryRow یک ردیف فایل را می‌سازد و مغایرت‌هایش را می‌نویسد.
//
// قاعدهٔ سقف بیمه: اگر دستمزد مشمول از سقف پارامتر بیشتر باشد، مبنای بیمه به سقف
// محدود می‌شود (و مغایرتش ثبت می‌گردد)؛ نرخ‌ها روی همان مبنا اعمال می‌شوند.
func BuildStatutoryRow(rowNo int, in StatutoryInput, rates StatutoryRates) StatutoryRow {
	row := StatutoryRow{
		RowNo: rowNo, EmployeeID: in.EmployeeID, EmployeeName: in.EmployeeName,
		NationalID:     strings.TrimSpace(calendar.NormalizeDigits(in.NationalID)),
		DepartmentName: in.DepartmentName,
		Days:           in.Days,
		Gross:          in.Gross,
		TaxableIncome:  in.TaxableIncome,
		TaxExemption:   in.TaxExemption,
		Tax:            in.Tax,
		Net:            in.Net,
	}

	insurable := in.Gross
	if rates.Ceiling > 0 && insurable > rates.Ceiling {
		insurable = rates.Ceiling
		row.IssuesFa = append(row.IssuesFa, fmt.Sprintf(
			"دستمزد مشمول از سقف بیمه (%s ریال) بیشتر بود؛ مبنای بیمه به سقف محدود شد.",
			formatFa(rates.Ceiling)))
	}
	row.InsurableWage = insurable
	row.EmployeeShare = rulesRound(float64(insurable) * rates.EmployeeRate)
	row.EmployerShare = rulesRound(float64(insurable) * rates.EmployerRate)
	row.Unemployment = rulesRound(float64(insurable) * rates.UnemploymentRate)
	row.InsuranceSum = row.EmployeeShare + row.EmployerShare + row.Unemployment

	if row.Days <= 0 {
		row.IssuesFa = append(row.IssuesFa, "روز کارکرد صفر است؛ ردیف بدون روز مشمول در فایل می‌آید و باید بررسی شود.")
	}
	if row.Days > 31 {
		row.IssuesFa = append(row.IssuesFa, "روز کارکرد بیش از ۳۱ است و به ۳۱ محدود شد.")
		row.Days = 31
	}
	if row.Gross <= 0 {
		row.IssuesFa = append(row.IssuesFa, "جمع حقوق و مزایا صفر است؛ ردیف در فایل بیمه/مالیات مشکوک است.")
	}
	if err := ValidateIranianNationalID(row.NationalID); err != nil {
		row.IssuesFa = append(row.IssuesFa, "مغایرت کد ملی: "+err.Error()+".")
	}
	if row.Tax < 0 || row.TaxableIncome < 0 {
		row.IssuesFa = append(row.IssuesFa, "درآمد مشمول یا مالیات منفی است؛ محاسبه بازبینی شود.")
	}
	return row
}

// BuildStatutoryFile ردیف‌ها را می‌سازد، جمع‌های کنترلی را می‌گیرد و مهلت/جریمه را
// می‌سنجد. «today» تاریخ جاری سرور به قالب شمسی است (از deps.Now می‌آید).
func BuildStatutoryFile(kind, period string, inputs []StatutoryInput, rates StatutoryRates, today string) (StatutoryFile, error) {
	title, ok := StatutoryTitles(kind)
	if !ok {
		return StatutoryFile{}, &recordkit.RuleError{Message: "نوع فایل قانونی «" + kind + "» شناخته نشد؛ «insurance» یا «tax» لازم است."}
	}
	if _, err := calendar.Parse(strings.TrimSpace(period) + "/01"); err != nil {
		return StatutoryFile{}, &recordkit.RuleError{Message: "دوره باید به قالب YYYY/MM باشد."}
	}

	sort.SliceStable(inputs, func(i, j int) bool {
		if inputs[i].EmployeeID != inputs[j].EmployeeID {
			return inputs[i].EmployeeID < inputs[j].EmployeeID
		}
		return inputs[i].EmployeeName < inputs[j].EmployeeName
	})

	file := StatutoryFile{Kind: kind, TitleFa: title, Period: period, Rates: rates}
	seenNational := map[string]bool{}
	for i, in := range inputs {
		row := BuildStatutoryRow(i+1, in, rates)
		if row.NationalID != "" {
			if seenNational[row.NationalID] {
				row.IssuesFa = append(row.IssuesFa, "کد ملی این کارمند در همین فایل تکرار شده است؛ فایل بیمه/مالیات ردیف تکراری نمی‌پذیرد.")
			}
			seenNational[row.NationalID] = true
		}
		file.Rows = append(file.Rows, row)
	}
	file.Totals = StatutoryTotalsOf(file.Rows)
	for _, row := range file.Rows {
		if len(row.IssuesFa) > 0 {
			file.IssuesFa = append(file.IssuesFa, fmt.Sprintf("ردیف %d (%s): %s",
				row.RowNo, row.EmployeeName, strings.Join(row.IssuesFa, " ")))
		}
	}
	file.ColumnsFa = StatutoryColumns(kind)
	file.DeadlineFa = statutoryDeadline(period)
	file.DaysLeft, file.DaysLate = statutoryDeadlineGap(period, today)
	file.PayableTotal = statutoryPayable(kind, file.Totals)
	file.PenaltyDays = file.DaysLate
	if file.DaysLate > 0 {
		rate := rates.InsurancePenalty
		if kind == StatutoryTax {
			rate = rates.TaxPenalty
		}
		file.Penalty = rulesRound(float64(file.PayableTotal) * rate * float64(file.DaysLate) / 30)
	}
	file.NoteFa = statutoryNote(kind, file)
	return file, nil
}

// StatutoryTotalsOf جمع‌های کنترلی فایل را از ردیف‌ها می‌گیرد.
func StatutoryTotalsOf(rows []StatutoryRow) StatutoryTotals {
	t := StatutoryTotals{Rows: len(rows)}
	for _, r := range rows {
		t.Days += r.Days
		t.Gross += r.Gross
		t.InsurableWage += r.InsurableWage
		t.EmployeeShare += r.EmployeeShare
		t.EmployerShare += r.EmployerShare
		t.Unemployment += r.Unemployment
		t.InsuranceSum += r.InsuranceSum
		t.TaxableIncome += r.TaxableIncome
		t.TaxExemption += r.TaxExemption
		t.Tax += r.Tax
		t.Net += r.Net
		if r.InsurableWage != r.Gross {
			t.CappedRows++
		}
	}
	return t
}

// statutoryPayable مبلغ قابل پرداخت هر فایل: بیمه = جمع سهم‌ها، مالیات = جمع مالیات.
func statutoryPayable(kind string, t StatutoryTotals) int64 {
	if kind == StatutoryTax {
		return t.Tax
	}
	return t.InsuranceSum
}

// statutoryDeadline آخرین روز مهلت قانونی: پایان ماه بعد از دوره.
func statutoryDeadline(period string) string {
	year, month, ok := splitPeriod(period)
	if !ok {
		return ""
	}
	nextMonth := month + 1
	nextYear := year
	if nextMonth > 12 {
		nextMonth = 1
		nextYear++
	}
	length, err := calendar.MonthLength(nextYear, nextMonth)
	if err != nil {
		length = 30
	}
	return fmt.Sprintf("%04d/%02d/%02d", nextYear, nextMonth, length)
}

// statutoryDeadlineGap روزهای باقی‌مانده تا مهلت و روزهای تأخیر از مهلت را می‌دهد؛
// عدد دوم وقتی معنا دارد که از مهلت گذشته باشیم.
func statutoryDeadlineGap(period, today string) (int, int) {
	deadline := statutoryDeadline(period)
	deadlineDate, err := calendar.Parse(deadline)
	if err != nil {
		return 0, 0
	}
	todayDate, err := calendar.Parse(strings.TrimSpace(calendar.NormalizeDigits(today)))
	if err != nil {
		return 0, 0
	}
	gap, err := calendar.DaysBetween(todayDate, deadlineDate)
	if err != nil {
		return 0, 0
	}
	if gap >= 0 {
		return gap, 0
	}
	return 0, -gap
}

// statutoryNote یادداشت فارسی فایل: مهلت، جریمه و هشدارهای مهم.
func statutoryNote(kind string, file StatutoryFile) string {
	parts := []string{fmt.Sprintf("مهلت قانونی ارسال و پرداخت: پایان ماه بعد (%s).", file.DeadlineFa)}
	switch {
	case file.DaysLate > 0:
		parts = append(parts, fmt.Sprintf("از مهلت %d روز گذشته است؛ برآورد جریمهٔ دیرکرد %s ریال (به‌نسبت روزهای تأخیر).",
			file.DaysLate, formatFa(file.Penalty)))
	case file.DaysLeft > 0:
		parts = append(parts, fmt.Sprintf("%d روز تا مهلت باقی است.", file.DaysLeft))
	}
	if len(file.IssuesFa) > 0 {
		parts = append(parts, fmt.Sprintf("%d ردیف مغایرت دارد؛ پیش از ارسال باید رفع شود.", len(file.IssuesFa)))
	}
	if file.Totals.CappedRows > 0 {
		parts = append(parts, fmt.Sprintf("در %d ردیف مبنای بیمه به سقف پارامتر محدود شد.", file.Totals.CappedRows))
	}
	if kind == StatutoryTax {
		parts = append(parts, "معافیت مالیاتی از پارامتر شرکت خوانده شد و مالیات با پله‌های پارامتریک محاسبه می‌شود.")
	}
	return strings.Join(parts, " ")
}

// splitPeriod دورهٔ YYYY/MM را به عدد می‌شکند.
func splitPeriod(period string) (int, int, bool) {
	parts := strings.Split(strings.TrimSpace(period), "/")
	if len(parts) != 2 {
		return 0, 0, false
	}
	year, errY := strconv.Atoi(strings.TrimSpace(calendar.NormalizeDigits(parts[0])))
	month, errM := strconv.Atoi(strings.TrimSpace(calendar.NormalizeDigits(parts[1])))
	if errY != nil || errM != nil || year < 1300 || month < 1 || month > 12 {
		return 0, 0, false
	}
	return year, month, true
}

// rulesRound گرد کردن ریالی نیم‌به‌بالا (هم‌رفتار با موتور قواعد).
func rulesRound(x float64) int64 {
	if x >= 0 {
		return int64(x + 0.5)
	}
	return -int64(-x + 0.5)
}

// formatFa عدد را با جداکنندهٔ هزارگان می‌نویسد (پیام‌های فارسی).
func formatFa(n int64) string {
	negative := n < 0
	if negative {
		n = -n
	}
	digits := strconv.FormatInt(n, 10)
	out := make([]byte, 0, len(digits)+len(digits)/3)
	for i, ch := range digits {
		if i > 0 && (len(digits)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, byte(ch))
	}
	if negative {
		return "-" + string(out)
	}
	return string(out)
}
