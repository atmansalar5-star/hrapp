// آزمون زیرساخت «documents» در پلتفرم سرور — «documents_test».

package documents

import (
	"strings"
	"testing"
)

func sample(kind Kind) Data {
	return Data{
		Kind:     kind,
		Serial:   "CT-1405-0100",
		IssuedAt: "1405/07/07",
		Company: Company{
			TitleFa: "پتروشیمی بندر امام", HoldingFa: "هلدینگ خلیج فارس", ShortCode: "BIP",
			CurrencyFa: "ریال", WorkWeekFa: "شنبه تا چهارشنبه", FiscalYearStartFa: "۰۱/۰۱",
		},
		Employee: Employee{
			ID: "EMP-003", FullName: "نگار حسینی", NationalID: "0034567895",
			DepartmentName: "فروش و بازاریابی", Position: "کارشناس فروش",
			HireDate: "1401/11/20", ContractType: "تمام‌وقت", BaseSalary: 34_000_000,
		},
		Contract: &ContractInfo{
			ID: "CT-1405-0100", KindFa: "مدت موقت", StatusFa: "جاری",
			StartDate: "1405/07/01", EndDate: "1406/07/01",
			MonthlySalary: 34_000_000, ServiceDays: 1500, ServiceYears: 4,
		},
		Params: []Param{{TitleFa: "ساعت کار موظفی ماهانه", Value: "176", UnitFa: "ساعت", GroupFa: "حقوق و دستمزد"}},
		BodyFa: []string{"طرف دوم انجام وظیفه می‌کند."},
		NoteFa: "نمونه",
	}
}

func TestRenderShapeAndPrintStyles(t *testing.T) {
	for _, kind := range []Kind{KindContract, KindSalaryDecree, KindEmploymentCertificate} {
		html, err := Render(sample(kind))
		if err != nil {
			t.Fatalf("ساخت سند %s ناموفق: %v", kind, err)
		}
		for _, want := range []string{
			`<!doctype html>`, `lang="fa"`, `dir="rtl"`, "@page", `size: A4`,
			"@media print", "window.print", TitleFa(kind), "نگار حسینی",
			"پتروشیمی بندر امام", VerifyCode("CT-1405-0100", "EMP-003"),
		} {
			if !strings.Contains(html, want) {
				t.Fatalf("سند %s «%s» را ندارد", kind, want)
			}
		}
	}
}

// TestRenderEscapesUserData نام کارمند دادهٔ کاربر است؛ هرگز نباید به‌عنوان کد تفسیر شود.
func TestRenderEscapesUserData(t *testing.T) {
	data := sample(KindEmploymentCertificate)
	data.Employee.FullName = `<script>alert('x')</script>`
	data.NoteFa = `<img src=x onerror=alert(1)>`
	html, err := Render(data)
	if err != nil {
		t.Fatalf("ساخت سند ناموفق: %v", err)
	}
	// متن خطرناک باید به‌صورت متن دیده شود، نه به‌صورت تگ قابل اجرا.
	if strings.Contains(html, "<script>alert") || strings.Contains(html, "<img src=x") {
		t.Fatal("دادهٔ کاربر بدون escape در سند نشست")
	}
	if !strings.Contains(html, "&lt;script&gt;") || !strings.Contains(html, "&lt;img src=x") {
		t.Fatal("انتظار escape شدن کد بود")
	}
}

func TestRenderRejectsUnknownKind(t *testing.T) {
	if _, err := Render(sample(Kind("salary_letter"))); err == nil {
		t.Fatal("نوع سند ناشناخته باید خطا بدهد")
	}
	if _, ok := ModuleOf(Kind("salary_letter")); ok {
		t.Fatal("نوع ناشناخته نباید ماژول مالک داشته باشد")
	}
}

func TestModuleOwnerMapping(t *testing.T) {
	cases := map[Kind]string{
		KindContract:              "contracts",
		KindSalaryDecree:          "salary_admin",
		KindEmploymentCertificate: "employee",
	}
	for kind, module := range cases {
		got, ok := ModuleOf(kind)
		if !ok || got != module {
			t.Fatalf("مالک سند %s باید %s باشد، «%s» شد", kind, module, got)
		}
	}
}

// TestPersianDigitsDates: تاریخ، کد ملی و مبلغ در سند باید با رقم فارسی دیده شوند.
func TestPersianDigitsDates(t *testing.T) {
	html, err := Render(sample(KindContract))
	if err != nil {
		t.Fatalf("ساخت سند ناموفق: %v", err)
	}
	body := visibleText(html)
	for _, want := range []string{"۱۴۰۵/۰۷/۰۱", "۱۴۰۱/۱۱/۲۰", "۰۰۳۴۵۶۷۸۹۵", "۱۷۶"} {
		if !strings.Contains(body, want) {
			t.Fatalf("سند چاپی باید «%s» را با رقم فارسی داشته باشد", want)
		}
	}
	// شناسه‌ها باید لاتین بمانند تا با دادهٔ سرور و جست‌وجو یکی باشند.
	for _, want := range []string{"CT-1405-0100", "EMP-003"} {
		if !strings.Contains(body, want) {
			t.Fatalf("شناسه «%s» باید لاتین بماند", want)
		}
	}
	// ارقام لاتین نباید در متن دیده شوند.
	if off := firstLatinDigits(body); off != "" {
		t.Fatalf("متن سند هنوز رقم لاتین دارد: %q", off)
	}
}

// TestPersianDigitsKeepsMarkup: ویژگی‌ها (style/href/class) و محتوای style نباید تغییر کند.
func TestPersianDigitsKeepsMarkup(t *testing.T) {
	in := `<div class="a" style="margin:18mm"><span>1405/07/01</span><!-- DEC-1405-004 --></div>`
	out := PersianDigits(in)
	for _, want := range []string{`style="margin:18mm"`, `class="a"`, "۱۴۰۵/۰۷/۰۱", "DEC-1405-004"} {
		if !strings.Contains(out, want) {
			t.Fatalf("خروجی باید «%s» را داشته باشد: %s", want, out)
		}
	}
}

// visibleText متن دیده‌شدنی سند را جدا می‌کند (بدون تگ، style و script) تا سنجش ارقام معنا پیدا کند.
func visibleText(html string) string {
	html = stripBlock(stripBlock(html, "style"), "script")
	var b strings.Builder
	for i := 0; i < len(html); {
		if html[i] == '<' {
			j := strings.IndexByte(html[i:], '>')
			if j < 0 {
				break
			}
			i += j + 1
			b.WriteByte(' ')
			continue
		}
		b.WriteByte(html[i])
		i++
	}
	return b.String()
}

// firstLatinDigits نخستین واژهٔ عددیِ لاتین را با زمینهٔ کوتاه برمی‌گرداند؛ کمک به
// عیب‌یابی آزمون. واژه‌های دارای حرف لاتین (شناسه‌ها) عمداً لاتین می‌مانند و به‌حساب نمی‌آیند.
func firstLatinDigits(body string) string {
	fields := strings.Fields(body)
	for i, w := range fields {
		if !strings.ContainsFunc(w, isASCIIDigit) || strings.ContainsFunc(w, isLatinLetter) {
			continue
		}
		lo, hi := i-3, i+4
		if lo < 0 {
			lo = 0
		}
		if hi > len(fields) {
			hi = len(fields)
		}
		return strings.Join(fields[lo:hi], " ")
	}
	return ""
}

// stripBlock همهٔ بلوک‌های <tag>…</tag> را از HTML برمی‌دارد (بدون بازگشت‌مرجع چون
// موتور regexp زبان Go پشتیبانی نمی‌کند).
func stripBlock(html, tag string) string {
	open := "<" + tag
	closeTag := "</" + tag + ">"
	for {
		lower := strings.ToLower(html)
		i := strings.Index(lower, open)
		if i < 0 {
			return html
		}
		j := strings.Index(lower[i:], closeTag)
		if j < 0 {
			return html[:i]
		}
		html = html[:i] + " " + html[i+j+len(closeTag):]
	}
}

func TestVerifyCodeIsStableAndDistinct(t *testing.T) {
	a := VerifyCode("CT-1405-0100", "EMP-003")
	if a != VerifyCode("CT-1405-0100", "EMP-003") {
		t.Fatal("کد رهگیری باید پایدار باشد")
	}
	if a == VerifyCode("CT-1405-0101", "EMP-003") || a == VerifyCode("CT-1405-0100", "EMP-004") {
		t.Fatal("کد رهگیری باید بین اسناد متفاوت باشد")
	}
	if len(a) < 8 {
		t.Fatalf("کد رهگیری کوتاه است: %s", a)
	}
}
