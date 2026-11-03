// آزمون نمایش «مدرک تحصیلی، نشانی و شماره‌ها» روی اسناد چاپی و سفارشی‌سازی بلوک‌ها.
//
// چرا این آزمون‌ها: این اطلاعات از پروندهٔ کارمند و پروندهٔ تحصیلی می‌آیند و باید هم روی
// برگهٔ چاپی (جداول مشخصات و بلوک «مشخصات پایهٔ کارمند») و هم در جای‌نگهدارهای قالب متن
// دیده شوند؛ و از سوی دیگر مدیر سامانه باید بتواند بلوک‌ها را روشن/خاموش کند. اگر یکی از
// این دو مسیر جا بیفتد، سند رسمی ناقص یا ناخواسته پرنویسه چاپ می‌شود.

package documents

import (
	"strings"
	"testing"
)

// baseInfoEmployee کارمندی با اطلاعات پایهٔ کامل و یک مدرک تحصیلی.
func baseInfoEmployee() Employee {
	return Employee{
		ID: "EMP-004", FullName: "علی مرادی", NationalID: "0045678901",
		DepartmentName: "مالی و حسابداری", Position: "کارشناس ارشد مالی",
		HireDate: "1399/06/15", ContractType: "تمام\u200cوقت",
		Mobile: "09123000004", HomePhone: "06152278104", PostalCode: "6155678913",
		Address:          "بندر امام خمینی، کوی سازمانی پتروشیمی، فاز ۲، پلاک ۴۰۱",
		EducationLevelFa: "کارشناسی ارشد", EducationField: "حسابداری",
		EducationUniver: "دانشگاه شهید چمران", EducationDate: "1396/07/30",
		EducationText: "کارشناسی ارشد — حسابداری", DegreeBasisFa: "کارشناسی ارشد — حسابداری",
		HasEducationInfo: true,
		BaseSalary:       41000000,
	}
}

// certificateData سند گواهی اشتغال نمونه برای آزمون‌ها.
func certificateData() Data {
	return Data{
		Kind: KindEmploymentCertificate, Serial: "CERT-1405-0001", IssuedAt: "1405/08/12",
		Company:  Company{TitleFa: "پتروشیمی بندر امام", HoldingFa: "هلدینگ خلیج فارس", CurrencyFa: "ریال"},
		Employee: baseInfoEmployee(),
		BodyFa:   []string{"گواهی می‌شود نامبرده در این شرکت اشتغال دارد."},
	}
}

// TestPrintShowsBaseInfoAndEducation بررسی می‌کند نشانی، شماره‌ها و مدرک روی برگه بیایند.
func TestPrintShowsBaseInfoAndEducation(t *testing.T) {
	html, err := Render(certificateData())
	if err != nil {
		t.Fatalf("رندر سند ناموفق: %v", err)
	}
	// مقادیر عددی روی برگه با رقم فارسی می‌آیند (سیاست سند چاپی)، پس همان شکل را می‌سنجیم.
	checks := map[string]string{
		"شمارهٔ همراه":  faDigitsText("09123000004"),
		"تلفن منزل":     faDigitsText("06152278104"),
		"کد پستی":       faDigitsText("6155678913"),
		"نشانی":         "کوی سازمانی پتروشیمی",
		"سابقهٔ تحصیلی": "سابقهٔ تحصیلی",
		"مدرک تحصیلی":   "کارشناسی ارشد",
		"رشته":          "حسابداری",
		"دانشگاه":       "شهید چمران",
		"جدول مشخصات":   "مشخصات کارمند",
	}
	for label, needle := range checks {
		if !strings.Contains(html, needle) {
			t.Errorf("%s در سند نیامد (به‌دنبال «%s»)", label, needle)
		}
	}
	// هیچ شمارهٔ تماسی نباید با رقم لاتین روی برگه بماند.
	if strings.Contains(html, ">09123000004<") || strings.Contains(html, ">06152278104<") {
		t.Error("شمارهٔ تماس با رقم لاتین روی سند چاپ شد")
	}
}

// TestPayslipHidesContactAndEducation بررسی می‌کند فیش حقوقی این بلوک‌ها را نداشته باشد.
func TestPayslipHidesContactAndEducation(t *testing.T) {
	data := Data{
		Kind: KindPayslip, Serial: "PAY-1405-0001", IssuedAt: "1405/08/12",
		Company:  Company{TitleFa: "پتروشیمی بندر امام", CurrencyFa: "ریال"},
		Employee: baseInfoEmployee(),
		Payslip:  &PayslipInfo{Period: "1405/07", Gross: 1000, Net: 800},
		Layout:   DefaultLayout(KindPayslip),
	}
	html, err := Render(data)
	if err != nil {
		t.Fatalf("رندر فیش ناموفق: %v", err)
	}
	for _, banned := range []string{"کوی سازمانی", "سابقهٔ تحصیلی", "شمارهٔ همراه", "کد پستی"} {
		if strings.Contains(html, banned) {
			t.Errorf("فیش حقوقی نباید «%s» را داشته باشد", banned)
		}
	}
	// ولی مشخصات شغلی پایه باید بماند.
	if !strings.Contains(html, "مشخصات کارمند") {
		t.Error("جدول مشخصات کارمند از فیش حذف شده است")
	}
}

// TestLayoutCanHideAndRetitleBlocks سفارشی‌سازی پنل: خاموش‌کردن و بازعنوان‌کردن یک بلوک.
func TestLayoutCanHideAndRetitleBlocks(t *testing.T) {
	data := certificateData()
	data.Layout = Layout{
		Visible: map[string]bool{
			"employeeCard": true, "employeeInfo": true, "contactInfo": true,
			"education": false, "body": true, "signatures": true, "verify": true,
		},
		Titles: map[string]string{"contactInfo": "نشانی و تماس"},
	}
	html, err := Render(data)
	if err != nil {
		t.Fatalf("رندر سند ناموفق: %v", err)
	}
	if strings.Contains(html, "سابقهٔ تحصیلی") {
		t.Error("بلوک تحصیلی خاموش شده بود ولی روی سند آمد")
	}
	if !strings.Contains(html, "نشانی و تماس") {
		t.Error("عنوان سفارشی بلوک اعمال نشد")
	}
	if strings.Contains(html, "اطلاعات تماس و نشانی") {
		t.Error("عنوان پیش‌فرض به‌جای عنوان سفارشی چاپ شد")
	}
}

// TestKindBlocksCatalog بررسی می‌کند هر سند بلوک‌های پشتیبانی‌شدهٔ خودش را دارد.
func TestKindBlocksCatalog(t *testing.T) {
	for _, key := range KindBlocks(KindPayslip) {
		if key == "contactInfo" || key == "education" || key == "baseInfoForm" {
			t.Errorf("فیش حقوقی نباید بلوک %s را پشتیبانی کند", key)
		}
	}
	if len(KindBlocks(KindContract)) < 8 {
		t.Error("قرارداد باید بلوک‌های کامل داشته باشد")
	}
	// هر کلید کاتالوگ باید عنوان فارسی داشته باشد (برای نمایش در پنل).
	for _, key := range KindBlocks(KindSalaryDecree) {
		block, ok := BlockByKey(key)
		if !ok || strings.TrimSpace(block.TitleFa) == "" {
			t.Errorf("بلوک %s در کاتالوگ عنوان ندارد", key)
		}
	}
}

// TestPlaceholdersCoverBaseInfo بررسی می‌کند قالب‌های متنی به همهٔ مقادیر تازه دسترسی دارند.
func TestPlaceholdersCoverBaseInfo(t *testing.T) {
	vars := PlaceholderVars(Data{Employee: baseInfoEmployee()})
	want := map[string]string{
		"employee.mobile":          "09123000004",
		"employee.homePhone":       "06152278104",
		"employee.postalCode":      "6155678913",
		"employee.address":         "بندر امام خمینی، کوی سازمانی پتروشیمی، فاز ۲، پلاک ۴۰۱",
		"employee.education":       "کارشناسی ارشد — حسابداری",
		"employee.educationLevel":  "کارشناسی ارشد",
		"employee.educationField":  "حسابداری",
		"employee.educationUniver": "دانشگاه شهید چمران",
		"employee.degreeBasis":     "کارشناسی ارشد — حسابداری",
	}
	for key, value := range want {
		if got := vars[key]; got != value {
			t.Errorf("جای‌نگهدار %s = %q، انتظار %q", key, got, value)
		}
	}
	// قالب متن باید بتواند همین‌ها را در برگه بگذارد.
	lines := ApplyTemplate("نشانی: {{employee.address}}\nمدرک: {{employee.education}}", vars)
	if len(lines) != 2 || !strings.Contains(lines[0], "کوی سازمانی") || !strings.Contains(lines[1], "کارشناسی ارشد") {
		t.Errorf("جای‌گذاری در قالب متن درست نبود: %#v", lines)
	}
}

// TestBaseInfoFormSkipsEmptyValues بررسی می‌کند پروندهٔ ناقص، ردیف خالی روی سند نگذارد.
func TestBaseInfoFormSkipsEmptyValues(t *testing.T) {
	data := Data{Kind: KindSalaryDecree, Employee: Employee{ID: "EMP-001", FullName: "سارا محمدی"}}
	addBaseInfoForm(&data)
	if len(data.BaseInfoFields) != 0 {
		t.Errorf("پروندهٔ بی‌اطلاعات نباید بلوک مشخصات پایه بسازد: %#v", data.BaseInfoFields)
	}

	data.Employee.Mobile = "09123000001"
	addBaseInfoForm(&data)
	if len(data.BaseInfoFields) != 1 || data.BaseInfoTitleFa != "مشخصات پایهٔ کارمند" {
		t.Errorf("بلوک مشخصات پایه درست ساخته نشد: %#v (عنوان %q)", data.BaseInfoFields, data.BaseInfoTitleFa)
	}
}
