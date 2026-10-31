// آزمون‌های دامنهٔ فایل قانونی بیمه/مالیات و فایل پرداخت بانکی: سنجش کد ملی و شبا،
// محدودشدن به سقف بیمه، جمع‌های کنترلی، گرد کردن مبلغ، مهلت و برآورد جریمه.

package domain

import (
	"fmt"
	"strings"
	"testing"
)

func TestValidateIranianNationalID(t *testing.T) {
	// ۱۲۳۴۵۶۷۸۹ با رقم کنترل ۱، کد ملی معتبر است (جمع وزن‌دار ۲۱۰ و مانده ۱).
	if err := ValidateIranianNationalID("1234567891"); err != nil {
		t.Errorf("کد ملی معتبر نباید خطا بدهد: %v", err)
	}
	if err := ValidateIranianNationalID("۱۲۳۴۵۶۷۸۹۱"); err != nil {
		t.Errorf("کد ملی با ارقام فارسی هم باید پذیرفته شود: %v", err)
	}
	cases := map[string]string{
		"1234567890": "رقم کنترل نادرست",
		"123456789":  "کمتر از ده رقم",
		"1111111111": "ارقام یکسان",
		"123456789a": "نویسهٔ غیررقمی",
		"":           "خالی",
	}
	for code, why := range cases {
		if err := ValidateIranianNationalID(code); err == nil {
			t.Errorf("کد ملی «%s» (%s) باید رد شود", code, why)
		}
	}
}

// validIBAN یک شبای معتبر با رقم کنترل درست می‌سازد (محاسبهٔ مستقل از خودِ سنجه).
func validIBAN(body string) string {
	for i := 0; i < 100; i++ {
		candidate := fmt.Sprintf("IR%02d%s", i, body)
		// پس از IR، بدنهٔ ۲۴ رقمی به پایان می‌رود و I=۱۸ و R=۲۷ به عدد می‌آید.
		rearranged := candidate[2:] + "1827"
		remainder := 0
		for _, ch := range rearranged {
			remainder = (remainder*10 + int(ch-'0')) % 97
		}
		if remainder == 1 {
			return candidate
		}
	}
	return ""
}

func TestValidateIBAN(t *testing.T) {
	ok := validIBAN("5401026800208179090020")
	if ok == "" {
		t.Fatal("ساخت شبای معتبر آزمون ناموفق بود")
	}
	if err := ValidateIBAN(ok); err != nil {
		t.Errorf("شبای معتبر نباید خطا بدهد (%s): %v", ok, err)
	}
	if err := ValidateIBAN("IR000000000000000000000000"); err == nil {
		t.Errorf("شبای با رقم کنترل نادرست باید رد شود")
	}
	if err := ValidateIBAN("IR12345"); err == nil {
		t.Errorf("شبای کوتاه باید رد شود")
	}
	if err := ValidateIBAN(""); err == nil {
		t.Errorf("شبای خالی باید رد شود")
	}
}

func TestBuildStatutoryFileInsurance(t *testing.T) {
	rates := DefaultStatutoryRates()
	rates.Ceiling = 100_000_000
	rates.TaxExemption = 24_000_000
	inputs := []StatutoryInput{
		{EmployeeID: "EMP-001", EmployeeName: "سارا محمدی", NationalID: "1234567891",
			Days: 30, Gross: 60_000_000, TaxableIncome: 20_000_000, Tax: 1_000_000},
		{EmployeeID: "EMP-002", EmployeeName: "نگار حسینی", NationalID: "1234567890",
			Days: 0, Gross: 150_000_000, TaxableIncome: 90_000_000, Tax: 6_000_000},
	}
	file, err := BuildStatutoryFile(StatutoryInsurance, "1405/08", inputs, rates, "1405/08/10")
	if err != nil {
		t.Fatalf("ساخت فایل بیمه نباید خطا بدهد: %v", err)
	}
	if file.TitleFa == "" || file.DeadlineFa != "1405/09/30" {
		t.Errorf("عنوان و مهلت فایل درست نیست: %+v", file)
	}
	// از ۱۴۰۵/۰۸/۱۰ تا مهلت ۱۴۰۵/۰۹/۳۰: ۵۰ روز باقی است و جریمه‌ای نیست.
	if file.DaysLeft != 50 || file.DaysLate != 0 || file.Penalty != 0 {
		t.Errorf("مهلت نگذشته است، پس جریمه و تأخیر باید صفر باشد: left=%d late=%d penalty=%d",
			file.DaysLeft, file.DaysLate, file.Penalty)
	}
	totals := file.Totals
	if totals.Rows != 2 {
		t.Fatalf("شمار ردیف باید ۲ باشد، %d شد", totals.Rows)
	}
	// ردیف دوم از سقف گذشته است → مبنای بیمه ۱۰۰ میلیون و یک ردیف محدودشده.
	if totals.CappedRows != 1 || totals.InsurableWage != 160_000_000 {
		t.Errorf("محدودسازی به سقف بیمه درست اعمال نشد: capped=%d insurable=%d",
			totals.CappedRows, totals.InsurableWage)
	}
	// ۷٪ + ۲۰٪ + ۳٪ روی ۱۶۰ میلیون = ۴۸ میلیون.
	if totals.EmployeeShare != 11_200_000 || totals.EmployerShare != 32_000_000 ||
		totals.Unemployment != 4_800_000 || totals.InsuranceSum != 48_000_000 {
		t.Errorf("سهم‌های بیمه درست محاسبه نشد: %+v", totals)
	}
	if file.PayableTotal != totals.InsuranceSum {
		t.Errorf("مبلغ قابل پرداخت فایل بیمه باید جمع سهم‌ها باشد: %d != %d",
			file.PayableTotal, totals.InsuranceSum)
	}
	// ردیف دوم هم مغایرت کد ملی دارد، هم روز صفر و هم عبور از سقف بیمه.
	if len(file.IssuesFa) == 0 {
		t.Fatalf("مغایرت‌های ردیف دوم باید گزارش شوند")
	}
	issues := file.IssuesFa[0]
	for _, want := range []string{"کد ملی", "روز کارکرد صفر", "سقف بیمه"} {
		if !strings.Contains(issues, want) {
			t.Errorf("مغایرت‌ها باید «%s» را داشته باشند: %s", want, issues)
		}
	}
}

func TestBuildStatutoryFileTaxAndLatePenalty(t *testing.T) {
	rates := DefaultStatutoryRates()
	rates.TaxExemption = 24_000_000
	inputs := []StatutoryInput{
		{EmployeeID: "EMP-001", EmployeeName: "سارا محمدی", NationalID: "1234567891",
			Days: 30, Gross: 100_000_000, TaxableIncome: 69_000_000, Tax: 5_000_000},
	}
	// تاریخ جاری پس از مهلت (پایان ماه بعد) است → جریمهٔ دیرکرد برآورد می‌شود.
	file, err := BuildStatutoryFile(StatutoryTax, "1405/07", inputs, rates, "1405/09/10")
	if err != nil {
		t.Fatalf("ساخت فایل مالیات نباید خطا بدهد: %v", err)
	}
	if file.DeadlineFa != "1405/08/30" {
		t.Errorf("مهلت فایل مالیات دورهٔ ۱۴۰۵/۰۷ باید پایان ۱۴۰۵/۰۸ باشد، «%s» شد", file.DeadlineFa)
	}
	if file.DaysLate != 10 {
		t.Errorf("تأخیر باید ۱۰ روز باشد، %d شد", file.DaysLate)
	}
	if file.PayableTotal != 5_000_000 {
		t.Errorf("مبلغ قابل پرداخت فایل مالیات باید جمع مالیات باشد، %d شد", file.PayableTotal)
	}
	// ۵ میلیون × ۲٫۵٪ × ۱۰/۳۰ = ۴۱٬۶۶۷ ریال (گرد شده).
	if file.Penalty != 41_667 {
		t.Errorf("برآورد جریمهٔ دیرکرد باید ۴۱٬۶۶۷ ریال باشد، %d شد", file.Penalty)
	}
	if len(file.Rows) != 1 || file.Rows[0].RowNo != 1 {
		t.Errorf("ردیف‌ها باید شماره‌گذاری شوند: %+v", file.Rows)
	}
	if len(StatutoryColumns(StatutoryTax)) != 9 || len(StatutoryRowValues(StatutoryTax, file.Rows[0])) != 9 {
		t.Errorf("ستون‌های فایل مالیات باید ۹ ستون باشد")
	}
}

func TestBuildBankFileRules(t *testing.T) {
	okIBAN := validIBAN("5401026800208179090020")
	other := validIBAN("5401026800208179090021")
	inputs := []BankInput{
		{EmployeeID: "EMP-001", EmployeeName: "سارا محمدی", NationalID: "1234567891",
			BankNameFa: "بانک ملی", IBAN: okIBAN, Amount: 53_456_789},
		{EmployeeID: "EMP-002", EmployeeName: "نگار حسینی", BankNameFa: "بانک ملی",
			IBAN: other, Amount: 30_000_000, Returned: true},
		{EmployeeID: "EMP-003", EmployeeName: "محمد احمدی", BankNameFa: "بانک ملت",
			IBAN: "", Amount: 12_000_000},
	}
	file := BuildBankFile("1405/08", "بانک ملی", inputs, 1000, "1405/08/10")
	if file.Totals.Rows != 3 {
		t.Fatalf("شمار ردیف باید ۳ باشد، %d شد", file.Totals.Rows)
	}
	// گرد کردن: ۵۳٬۴۵۶٬۷۸۹ → ۵۳٬۴۵۷٬۰۰۰ (تفاوت ۲۱۱) و ۳۰٬۰۰۰٬۰۰۰ بدون تفاوت.
	if file.Rows[0].Amount != 53_457_000 || file.Rows[0].Rounding != 211 {
		t.Errorf("گرد کردن ردیف اول درست نیست: %+v", file.Rows[0])
	}
	if file.Totals.Rounding != 211 {
		t.Errorf("جمع تفاوت گرد کردن باید ۲۱۱ باشد، %d شد", file.Totals.Rounding)
	}
	if file.Totals.Amount != 53_457_000+30_000_000+12_000_000 {
		t.Errorf("جمع مبلغ پرداختی درست نیست: %d", file.Totals.Amount)
	}
	if file.Totals.ReturnedRows != 1 || file.Totals.ReturnedSum != 30_000_000 {
		t.Errorf("برگشتی‌ها درست شمرده نشدند: %+v", file.Totals)
	}
	if file.Totals.MissingInputs != 1 {
		t.Errorf("ردیف بدون شبا باید شمرده شود: %+v", file.Totals)
	}
	if file.ControlFa == "" || file.NoteFa == "" {
		t.Errorf("جمع کنترلی و یادداشت فارسی باید ساخته شود: %+v", file)
	}
	// شبای تکراری در دو ردیف باید مغایرت بگیرد.
	dup := BuildBankFile("1405/08", "بانک ملی", []BankInput{
		{EmployeeID: "EMP-001", EmployeeName: "الف", BankNameFa: "بانک ملی", IBAN: okIBAN, Amount: 1_000_000},
		{EmployeeID: "EMP-002", EmployeeName: "ب", BankNameFa: "بانک ملی", IBAN: okIBAN, Amount: 1_000_000},
	}, 1000, "1405/08/10")
	if len(dup.IssuesFa) == 0 {
		t.Errorf("شبای تکراری باید مغایرت گزارش شود: %+v", dup.Rows)
	}
	if err := BankFileKindReady(BankFile{}); err == nil {
		t.Errorf("فایل بدون ردیف باید خطای قاعده بدهد")
	}
}

func TestRoundToAndFormat(t *testing.T) {
	cases := []struct {
		amount, unit, want, diff int64
	}{
		{53_456_789, 1000, 53_457_000, 211},
		{30_000_000, 1000, 30_000_000, 0},
		{0, 1000, 0, 0},
		{1_500, 1000, 2_000, 500},
		{1_499, 1000, 1_000, -499},
	}
	for _, c := range cases {
		got, diff := roundTo(c.amount, c.unit)
		if got != c.want || diff != c.diff {
			t.Errorf("گرد کردن %d با واحد %d: انتظار %d/%d، نتیجه %d/%d",
				c.amount, c.unit, c.want, c.diff, got, diff)
		}
	}
	if formatFa(1_234_567) != "1,234,567" || formatFa(-1_000) != "-1,000" {
		t.Errorf("قالب‌بندی عدد فارسی درست نیست: %s / %s", formatFa(1_234_567), formatFa(-1_000))
	}
}
