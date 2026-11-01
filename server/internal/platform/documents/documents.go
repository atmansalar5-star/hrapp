// Package documents اسناد چاپی فارسی (قرارداد، حکم حقوق، گواهی اشتغال).
//
// سند به‌صورت HTML راست‌چین با شیوه‌نامهٔ چاپ A4 ساخته می‌شود و مرورگر آن را به PDF
// تبدیل می‌کند. این انتخاب عمدی است: شکل‌دهی حروف فارسی (اتصال حروف و راست‌به‌چپ) در
// موتور چاپ مرورگر درست انجام می‌شود، درحالی‌که تولید مستقیم PDF بدون موتور متن پیچیده
// حروف فارسی را جدا و بی‌اتصال نشان می‌دهد. پس یک قالب، دو مسیر: چاپ از مرورگر برای
// کاربر و رندر دسته‌ای با مرورگر بی‌سر برای بایگانی.
//
// قالب‌ها با html/template نوشته شده‌اند تا مقادیر داده خودکار escape شوند؛ نام کارمند
// دادهٔ کاربر است و هرگز نباید در سند به‌صورت کد تفسیر شود.
package documents

import (
	"bytes"
	"crypto/sha256"
	"encoding/base32"
	"fmt"
	"html/template"
	"strings"
)

// Kind نوع سند.
type Kind string

// انواع سند پشتیبانی‌شده و ماژول مالک داده‌شان (برای بررسی دسترسی).
const (
	KindContract              Kind = "contract"
	KindSalaryDecree          Kind = "salary_decree"
	KindEmploymentCertificate Kind = "employment_certificate"
	KindPayslip               Kind = "payslip"
	// KindDegreeBasis فرم اداری «تعیین مدرک ملاک حقوق»: کدام مدرک، با چه ارتباطی و به
	// استناد کدام صورت‌جلسهٔ شورای تعدیل مدرک مبنای گروه شغلی و پرداخت است.
	KindDegreeBasis Kind = "degree_basis"
)

// ModuleOf ماژول مالک سند؛ دسترسی نقش روی همان ماژول سنجیده می‌شود.
func ModuleOf(kind Kind) (string, bool) {
	switch kind {
	case KindContract:
		return "contracts", true
	case KindSalaryDecree:
		return "salary_admin", true
	case KindEmploymentCertificate:
		return "employee", true
	case KindPayslip:
		return "payroll", true
	case KindDegreeBasis:
		return "archive", true
	default:
		return "", false
	}
}

// TitleFa عنوان فارسی سند.
func TitleFa(kind Kind) string {
	switch kind {
	case KindContract:
		return "قرارداد کار"
	case KindSalaryDecree:
		return "حکم حقوق"
	case KindEmploymentCertificate:
		return "گواهی اشتغال به کار"
	case KindPayslip:
		return "فیش حقوقی"
	case KindDegreeBasis:
		return "فرم تعیین مدرک ملاک حقوق"
	default:
		return string(kind)
	}
}

// Company مشخصات شرکت در سرصفحهٔ سند.
type Company struct {
	TitleFa           string
	HoldingFa         string
	ShortCode         string
	CurrencyFa        string
	WorkWeekFa        string
	FiscalYearStartFa string
	// LogoURL آدرس تصویر نشان شرکت روی همان مبدأ رابط وب (خالی = بدون تصویر).
	LogoURL string
	// BrandColor رنگ اصلی برند برای نوار سرصفحه (خالی = رنگ پیش‌فرض).
	BrandColor string
}

// Employee اطلاعات کارمند در سند.
type Employee struct {
	ID             string
	FullName       string
	NationalID     string
	DepartmentName string
	Position       string
	HireDate       string
	ContractType   string
	BaseSalary     int64
}

// ContractInfo خلاصهٔ قرارداد مرجع سند.
type ContractInfo struct {
	ID            string
	KindFa        string
	StatusFa      string
	StartDate     string
	EndDate       string
	MonthlySalary int64
	ServiceDays   int
	ServiceYears  int
}

// Param یک پارامتر مبنای محاسبه (از استقرار شرکت فعال، نه عدد ساختگی).
type Param struct {
	// Key کلید ماشین‌خوان پارامتر برای ارجاع در قالب متن (مثل payroll.monthlyHours).
	Key     string
	TitleFa string
	Value   string
	UnitFa  string
	GroupFa string
}

// PayslipInfo جزئیات محاسبهٔ فیش حقوقی برای سند فیش.
type PayslipInfo struct {
	Period            string
	Gross             int64
	InsuranceEmployee int64
	InsuranceEmployer int64
	Tax               int64
	Net               int64
	// MazdBase مزد مبنا: جمع اجزای ثابت مزد کارمند (از حکم یا قرارداد). مبنای اضافه‌کاری و
	// فوق‌العاده‌های درصدی است، پس روی فیش چاپی هم می‌آید.
	MazdBase int64
	// WageSourceFa منبع اجزای مزد («از حکم کارگزینی DEC-...») تا خوانندهٔ فیش بداند عدد از
	// کدام سند آمده است.
	WageSourceFa string
	// Elements تفکیک اقلام حقوق؛ همان محاسبه‌ای را نشان می‌دهد که شبیه‌ساز فیش می‌دهد.
	Elements []PayslipElement
}

// PayslipElement یک سطر از تفکیک اقلام فیش حقوقی.
type PayslipElement struct {
	TitleFa    string
	CategoryFa string
	AmountFa   string
	DetailFa   string
}

// StructureRow یک سطر از «اجزای مزد و فوق‌العاده‌ها» در حکم کارگزینی و قرارداد کار.
//
// GroupFa گروه سطر است («اجزای مزد مبنا»، «فوق‌العاده‌های اقلیمی»، …) و Total یعنی این سطر
// جمع گروه است؛ همین دو، جدول چاپی را خوانا می‌کند: خواننده می‌بیند مزد مبنا از چه ساخته
// شده و کدام اقلام روی آن سوار می‌شوند.
type StructureRow struct {
	TitleFa  string
	KindFa   string
	DetailFa string
	AmountFa string
	GroupFa  string
	Total    bool
}

// FormField یک سطر «فرم اداری» روی کاغذ: برچسب و مقدار.
//
// چرا سازوکار عمومی: سازمان فرم‌های اداری گوناگونی دارد (تعیین مدرک ملاک حقوق، تعدیل
// مدرک ایثارگران، صورت‌جلسهٔ شورا) و افزودن یک قالب HTML تازه برای هر فرم، نگهداشت را
// سنگین می‌کند. با این ساختار، هر ماژول فرم کاغذی خودش را از دادهٔ خودش می‌سازد.
type FormField struct {
	LabelFa string
	ValueFa string
}

// FormTable یک جدول «فرم اداری» روی کاغذ (سرستون‌ها و سطرها).
type FormTable struct {
	TitleFa string
	Head    []string
	Rows    [][]string
	NoteFa  string
}

// Data همهٔ دادهٔ لازم برای ساخت یک سند چاپی.
type Data struct {
	Kind     Kind
	Serial   string
	IssuedAt string
	Company  Company
	Employee Employee
	Contract *ContractInfo
	Payslip  *PayslipInfo
	// Structure اجزای مزد و فوق‌العاده‌های حکم کارگزینی و قرارداد کار؛ از همان
	// اقلام پارامتریک و همان موتور محاسبهٔ شبیه‌ساز ساخته می‌شود.
	Structure []StructureRow
	Params    []Param
	BodyFa    []string
	NoteFa    string
	// FormTitleFa/FormFields/FormTables فرم اداری سند (مثل «تعیین مدرک ملاک حقوق»).
	FormTitleFa string
	FormFields  []FormField
	FormTables  []FormTable
	// Signatures امضاهای پای سند؛ خالی یعنی سه امضای پیش‌فرض کارگزینی/کارمند/منابع انسانی.
	Signatures []string
	// CustomBody یعنی متن از قالب طراحى‌شدهٔ کاربر آمده است و به‌صورت بند‌بند نمایش
	// داده می‌شود؛ نه فهرست پیش‌فرض سامانه.
	CustomBody bool
	// VerifyURL نشانی صفحهٔ تأیید سند (مقصد QR کد چاپشده) و QRCodeSVG همان کد به شکل
	// SVG درون سند. هر دو را سرور میسازد، پس ورودی کاربر نیستند.
	VerifyURL string
	QRCodeSVG template.HTML
}

// ApplyTemplate جای‌نگهدارهای «{{key}}» را در متن قالب با مقادیر واقعی جایگزین می‌کند و
// متن را به خط‌های غیرخالی می‌شکند. مقادیر در زمان رندر توسط html/template escape
// می‌شوند، پس دادهٔ کاربر هرگز به‌صورت کد تفسیر نمی‌شود.
func ApplyTemplate(body string, vars map[string]string) []string {
	body = strings.ReplaceAll(body, "\r\n", "\n")
	for key, value := range vars {
		body = strings.ReplaceAll(body, "{{"+key+"}}", value)
	}
	lines := []string{}
	for _, line := range strings.Split(body, "\n") {
		if s := strings.TrimSpace(line); s != "" {
			lines = append(lines, s)
		}
	}
	return lines
}

// PlaceholderVars نگاشت جای‌نگهدارهای مجاز قالب متن بر پایهٔ دادهٔ سند می‌سازد.
// همین فهرست در صفحهٔ قالب‌های متن به کاربر نشان داده می‌شود.
func PlaceholderVars(d Data) map[string]string {
	vars := map[string]string{
		"company.title":         d.Company.TitleFa,
		"company.holding":       d.Company.HoldingFa,
		"company.shortCode":     d.Company.ShortCode,
		"company.currency":      d.Company.CurrencyFa,
		"company.workWeek":      d.Company.WorkWeekFa,
		"serial":                d.Serial,
		"issuedAt":              d.IssuedAt,
		"employee.id":           d.Employee.ID,
		"employee.fullName":     d.Employee.FullName,
		"employee.nationalId":   d.Employee.NationalID,
		"employee.department":   d.Employee.DepartmentName,
		"employee.position":     d.Employee.Position,
		"employee.hireDate":     d.Employee.HireDate,
		"employee.contractType": d.Employee.ContractType,
	}
	if d.Contract != nil {
		vars["contract.id"] = d.Contract.ID
		vars["contract.kind"] = d.Contract.KindFa
		vars["contract.status"] = d.Contract.StatusFa
		vars["contract.startDate"] = d.Contract.StartDate
		vars["contract.endDate"] = d.Contract.EndDate
		vars["contract.monthlySalary"] = fmt.Sprintf("%d", d.Contract.MonthlySalary)
		vars["contract.serviceYears"] = fmt.Sprintf("%d", d.Contract.ServiceYears)
	}
	if d.Payslip != nil {
		vars["payslip.period"] = d.Payslip.Period
		vars["payslip.gross"] = fmt.Sprintf("%d", d.Payslip.Gross)
		vars["payslip.insuranceEmployee"] = fmt.Sprintf("%d", d.Payslip.InsuranceEmployee)
		vars["payslip.insuranceEmployer"] = fmt.Sprintf("%d", d.Payslip.InsuranceEmployer)
		vars["payslip.tax"] = fmt.Sprintf("%d", d.Payslip.Tax)
		vars["payslip.net"] = fmt.Sprintf("%d", d.Payslip.Net)
		lines := make([]string, 0, len(d.Payslip.Elements))
		for _, el := range d.Payslip.Elements {
			lines = append(lines, fmt.Sprintf("%s: %s", el.TitleFa, el.AmountFa))
		}
		vars["payslip.elements"] = strings.Join(lines, "\n")
	}
	if len(d.Structure) > 0 {
		lines := make([]string, 0, len(d.Structure))
		for _, row := range d.Structure {
			lines = append(lines, fmt.Sprintf("%s: %s", row.TitleFa, row.AmountFa))
		}
		vars["structure.text"] = strings.Join(lines, "\n")
	}
	for _, p := range d.Params {
		if p.Key != "" {
			vars["params."+p.Key] = p.Value
		}
	}
	return vars
}

// VerifyCode کد رهگیری سند: امضای کوتاه از شمارهٔ سند و کد کارمند، تا سند کاغذی
// قابل تطبیق با نسخهٔ سامانه باشد.
func VerifyCode(serial, employeeID string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(serial) + "|" + strings.TrimSpace(employeeID)))
	return strings.TrimRight(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(sum[:8]), "=")
}

// Render سند را به HTML چاپی تبدیل می‌کند.
func Render(data Data) (string, error) {
	if _, ok := ModuleOf(data.Kind); !ok {
		return "", fmt.Errorf("نوع سند ناشناخته: %s", data.Kind)
	}
	view := struct {
		Data
		Title      string
		VerifyCode string
	}{
		Data:       data,
		Title:      TitleFa(data.Kind),
		VerifyCode: VerifyCode(data.Serial, data.Employee.ID),
	}
	var buf bytes.Buffer
	if err := documentTemplate.Execute(&buf, view); err != nil {
		return "", fmt.Errorf("ساخت سند ناموفق: %w", err)
	}
	return PersianDigits(buf.String()), nil
}

// PersianDigits ارقام لاتین را در متنِ سند به رقم فارسی برمی‌گرداند.
//
// چرا اینجا و نه سر هر جای‌نگهدار: تاریخ، کد ملی، مبلغ و شمارهٔ تماس در دادهٔ سرور
// با رقم لاتین ذخیره می‌شوند (قالب استاندارد، مرتب‌شدنی و جست‌وجوپذیر). اما سند چاپی
// فارسی است و قاطی‌شدن ارقام لاتین و فارسی روی یک برگه آن را ناموزون نشان می‌دهد.
//
// فقط متنِ بین تگ‌ها دست می‌خورد تا ویژگی‌ها (href، class، style) سالم بمانند، و
// متنی که حرف لاتین دارد دست‌نخورده می‌ماند چون شناسه است (DEC-1405-004،
// CT-1399-0004، کد رهگیری). یعنی همان سیاستی که در رابط وب هم اجرا می‌شود.
func PersianDigits(html string) string {
	var b strings.Builder
	b.Grow(len(html))
	for i := 0; i < len(html); {
		if html[i] == '<' {
			j := strings.IndexByte(html[i:], '>')
			if j < 0 {
				b.WriteString(html[i:])
				break
			}
			b.WriteString(html[i : i+j+1])
			i += j + 1
			continue
		}
		j := strings.IndexByte(html[i:], '<')
		if j < 0 {
			j = len(html) - i
		}
		b.WriteString(faDigitsText(html[i : i+j]))
		i += j
	}
	return b.String()
}

// faDigitsText یک قطعهٔ متنی (نه مارک‌آپ) را فارسی می‌کند.
//
// واژه‌به‌واژه پیش می‌رود نه کل قطعه: در سند، شناسه و تاریخ غالباً در یک گرهٔ متنی
// پشت سر هم می‌آیند («شمارهٔ سند: DEC-1405-004 تاریخ: 1405/07/12»). اگر معیار روی
// کل قطعه بود، حضور یک شناسه همهٔ ارقام آن را لاتین نگه می‌داشت. پس فقط واژه‌های
// بدون حرف لاتین فارسی می‌شوند.
func faDigitsText(s string) string {
	if !strings.ContainsFunc(s, isASCIIDigit) {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); {
		if isSpaceByte(s[i]) {
			b.WriteByte(s[i])
			i++
			continue
		}
		j := i
		for j < len(s) && !isSpaceByte(s[j]) {
			j++
		}
		word := s[i:j]
		if strings.ContainsFunc(word, isLatinLetter) {
			b.WriteString(word) // شناسه: EMP-004، CT-1399-0004، کد رهگیری
		} else {
			b.WriteString(persianizeDigits(word))
		}
		i = j
	}
	return b.String()
}

// persianizeDigits ارقام لاتین یک واژه را به رقم فارسی برمی‌گرداند.
func persianizeDigits(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune('۰' + (r - '0'))
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// isSpaceByte فقط جداکننده‌های واژه در سند است؛ نیم‌فاصله (U+200C) جزء واژه می‌ماند.
func isSpaceByte(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\f' || c == '\v'
}

func isASCIIDigit(r rune) bool { return r >= '0' && r <= '9' }

func isLatinLetter(r rune) bool {
	return (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z')
}

var documentTemplate = template.Must(template.New("document").Parse(`<!doctype html>
<html lang="fa" dir="rtl">
<head>
<meta charset="utf-8">
<title>{{.Title}} — {{.Company.TitleFa}}</title>
<style>
  /* فونت فارسی سند روی همان مبدأ رابط وب سرو می‌شود؛ مسیر مطلق است چون خود سند زیر
     /api/v1/documents/... سرو می‌شود و مسیر نسبی به آن مسیر می‌چسبید. */
  @font-face { font-family: Vazirmatn; font-style: normal; font-weight: 100 900; font-display: swap;
    src: url("/assets/fonts/Vazirmatn-var.woff2") format("woff2-variations"); }
  @page { size: A4; margin: 18mm 16mm 20mm 16mm; }
  * { box-sizing: border-box; }
  body { font-family: "Vazirmatn", "IRANSans", "Tahoma", sans-serif; color: #111827; margin: 0; padding: 24px; background: #f8fafc; font-size: 13px; line-height: 1.9; -webkit-font-smoothing: antialiased; }
  .sheet { background: #fff; max-width: 820px; margin: 0 auto; padding: 28px 30px 40px; border-radius: 10px; box-shadow: 0 4px 20px rgba(15,23,42,.08); }
  header { border-bottom: 2px solid {{if .Company.BrandColor}}{{.Company.BrandColor}}{{else}}#2563eb{{end}}; padding-bottom: 12px; margin-bottom: 18px; display: flex; justify-content: space-between; align-items: flex-start; gap: 12px; }
  .brand { font-weight: 800; font-size: 16px; display: flex; align-items: center; gap: 12px; }
  .brand-logo { width: 104px; height: 56px; object-fit: contain; object-position: right; }
  .brand-text small { display: block; font-weight: 500; color: #64748b; font-size: 11px; }
  .brand small { display: block; font-weight: 500; color: #64748b; font-size: 11px; }
  .doc-meta { text-align: left; font-size: 11px; color: #475569; direction: ltr; }
  h1 { font-size: 18px; margin: 0 0 4px; }
  .subtitle { color: #64748b; font-size: 12px; margin-bottom: 18px; }
  table { width: 100%; border-collapse: collapse; margin-bottom: 16px; }
  th, td { border: 1px solid #e2e8f0; padding: 7px 9px; text-align: right; vertical-align: top; }
  th { background: #f1f5f9; font-weight: 600; width: 22%; }
  .grid-stats { display: flex; gap: 10px; margin-bottom: 16px; }
  .stat { flex: 1; border: 1px solid #e2e8f0; border-radius: 8px; padding: 8px 10px; }
  .stat b { display: block; font-size: 14px; }
  .stat span { color: #64748b; font-size: 11px; }
  .num { font-variant-numeric: tabular-nums; }
  h2 { font-size: 14px; margin: 18px 0 8px; padding-right: 8px; border-right: 3px solid #2563eb; }
  ul { padding-right: 18px; margin: 0 0 14px; }
  .doc-body { border: 1px solid #e2e8f0; border-radius: 8px; padding: 6px 14px; margin-bottom: 16px; }
  .doc-body p { margin: 7px 0; text-align: justify; }
  .sign { display: flex; gap: 24px; margin-top: 30px; }
  .sign div { flex: 1; border-top: 1px dashed #94a3b8; padding-top: 6px; font-size: 11px; color: #475569; text-align: center; }
  footer { margin-top: 26px; border-top: 1px solid #e2e8f0; padding-top: 10px; font-size: 10px; color: #64748b; display: flex; justify-content: space-between; gap: 12px; }
  .toolbar { max-width: 820px; margin: 0 auto 12px; display: flex; gap: 8px; justify-content: flex-start; }
  .toolbar button { font-family: inherit; font-size: 13px; padding: 8px 16px; border-radius: 8px; border: 1px solid #2563eb; background: #2563eb; color: #fff; cursor: pointer; }
  .toolbar a { font-size: 12px; color: #2563eb; align-self: center; }
  @media print {
    body { background: #fff; padding: 0; }
    .sheet { box-shadow: none; border-radius: 0; padding: 0; max-width: none; }
    .toolbar { display: none; }
    h2 { break-after: avoid; }
    table { break-inside: avoid; }
  }
</style>
</head>
<body>
<div class="toolbar">
  <button onclick="window.print()">چاپ / ذخیرهٔ PDF</button>
  <a href="?">بازخوانی سند</a>
</div>
<div class="sheet">
  <header>
    <div class="brand">
      {{if .Company.LogoURL}}<img class="brand-logo" src="{{.Company.LogoURL}}" alt="{{.Company.HoldingFa}}">{{end}}
      <div class="brand-text">{{.Company.TitleFa}}
        <small>{{.Company.HoldingFa}} · کد شرکت {{.Company.ShortCode}}</small></div>
    </div>
    <div class="doc-meta">
      شمارهٔ سند: {{.Serial}}<br>
      تاریخ: {{.IssuedAt}}<br>
      کد رهگیری: {{.VerifyCode}}
    </div>
  </header>

  <h1>{{.Title}}</h1>
  <div class="subtitle">این سند از سامانهٔ جامع منابع انسانی صادر شده است.</div>

  <div class="grid-stats">
    <div class="stat"><b>{{.Employee.FullName}}</b><span>نام و نام خانوادگی</span></div>
    <div class="stat"><b class="num">{{.Employee.ID}}</b><span>کد پرسنلی</span></div>
    <div class="stat"><b>{{.Employee.DepartmentName}}</b><span>واحد سازمانی</span></div>
    <div class="stat"><b>{{.Employee.Position}}</b><span>پست سازمانی</span></div>
  </div>

  <h2>مشخصات کارمند</h2>
  <table>
    <tr><th>کد ملی</th><td class="num">{{.Employee.NationalID}}</td><th>تاریخ استخدام</th><td class="num">{{.Employee.HireDate}}</td></tr>
    <tr><th>نوع همکاری</th><td>{{.Employee.ContractType}}</td><th>هفتهٔ کاری شرکت</th><td>{{.Company.WorkWeekFa}}</td></tr>
  </table>

  {{if .Contract}}
  <h2>مشخصات قرارداد کار</h2>
  <table>
    <tr><th>شمارهٔ قرارداد</th><td class="num">{{.Contract.ID}}</td><th>نوع قرارداد</th><td>{{.Contract.KindFa}}</td></tr>
    <tr><th>تاریخ شروع</th><td class="num">{{.Contract.StartDate}}</td><th>تاریخ پایان</th><td class="num">{{if .Contract.EndDate}}{{.Contract.EndDate}}{{else}}— (مدت دائم){{end}}</td></tr>
    <tr><th>وضعیت قرارداد</th><td>{{.Contract.StatusFa}}</td><th>سابقهٔ خدمت</th><td class="num">{{.Contract.ServiceYears}} سال</td></tr>
    <tr><th>مبلغ پایهٔ ماهانه</th><td class="num">{{.Contract.MonthlySalary}} {{.Company.CurrencyFa}}</td><th>سال مالی شرکت</th><td class="num">{{.Company.FiscalYearStartFa}}</td></tr>
  </table>
  {{end}}

  {{if .Params}}
  <h2>مبنای محاسبهٔ سامانه</h2>
  <table>
    <tr><th>پارامتر</th><th>مقدار</th><th>واحد</th><th>گروه</th></tr>
    {{range .Params}}<tr><td>{{.TitleFa}}</td><td class="num">{{.Value}}</td><td>{{.UnitFa}}</td><td>{{.GroupFa}}</td></tr>{{end}}
  </table>
  {{end}}

  {{if .Payslip}}{{if .Payslip.Elements}}
  <h2>تفکیک اقلام حقوق</h2>
  {{if .Payslip.MazdBase}}
  <table>
    <tr><th>مزد مبنا (مبنای اضافه‌کاری و فوق‌العاده‌های درصدی)</th><td class="num">{{.Payslip.MazdBase}}</td>
        <th>منبع اجزای مزد</th><td>{{if .Payslip.WageSourceFa}}{{.Payslip.WageSourceFa}}{{else}}پروندهٔ پرسنلی و پارامترهای شرکت{{end}}</td></tr>
  </table>
  {{end}}
  <table>
    <tr><th>قلم حقوق</th><th>نوع</th><th>مبلغ محاسبه‌شده</th><th>شرح</th></tr>
    {{range .Payslip.Elements}}<tr><td>{{.TitleFa}}</td><td>{{.CategoryFa}}</td><td class="num">{{.AmountFa}}</td><td>{{.DetailFa}}</td></tr>{{end}}
  </table>
  {{end}}{{end}}

  {{if .Structure}}
  <h2>اجزای مزد مبنا و فوق‌العاده‌ها</h2>
  <table>
    <tr><th>جزء / فوق‌العاده</th><th>نوع</th><th>نحوهٔ محاسبه</th><th>مبلغ (ریال)</th></tr>
    {{range .Structure}}<tr{{if .Total}} class="total"{{end}}><td>{{if .Total}}<b>{{.TitleFa}}</b>{{else}}{{.TitleFa}}{{end}}</td><td>{{.KindFa}}</td><td>{{.DetailFa}}</td><td class="num">{{.AmountFa}}</td></tr>{{end}}
  </table>
  <p class="muted">مزد مبنا جمع اجزای ثابت است و مبنای محاسبهٔ اضافه‌کاری، شب‌کاری، نوبت‌کاری و فوق‌العاده‌های اقلیمی (که درصدی از همین مزد مبنا هستند و جداگانه پرداخت می‌شوند).</p>
  {{end}}

  {{if .FormFields}}
  <h2>{{if .FormTitleFa}}{{.FormTitleFa}}{{else}}مشخصات فرم{{end}}</h2>
  <table>
    {{range .FormFields}}<tr><th>{{.LabelFa}}</th><td>{{.ValueFa}}</td></tr>{{end}}
  </table>
  {{end}}

  {{range .FormTables}}
  <h2>{{.TitleFa}}</h2>
  <table>
    {{if .Head}}<tr>{{range .Head}}<th>{{.}}</th>{{end}}</tr>{{end}}
    {{range .Rows}}<tr>{{range .}}<td>{{.}}</td>{{end}}</tr>{{end}}
  </table>
  {{if .NoteFa}}<p class="muted">{{.NoteFa}}</p>{{end}}
  {{end}}

  {{if .BodyFa}}
  {{if .CustomBody}}
  <h2>متن سند</h2>
  <div class="doc-body">{{range .BodyFa}}<p>{{.}}</p>{{end}}</div>
  {{else}}
  <h2>شرح</h2>
  <ul>{{range .BodyFa}}<li>{{.}}</li>{{end}}</ul>
  {{end}}
  {{end}}

  {{if .NoteFa}}<div class="subtitle">📌 {{.NoteFa}}</div>{{end}}

  <div class="sign">
    {{if .Signatures}}{{range .Signatures}}<div>{{.}}</div>{{end}}{{else}}
    <div>مهر و امضای واحد کارگزینی</div>
    <div>امضای کارمند</div>
    <div>تأیید مدیر منابع انسانی</div>
    {{end}}
  </div>

  <footer>
    <span>سامانهٔ جامع منابع انسانی — {{.Company.TitleFa}}</span>
    <span>کد رهگیری سند: {{.VerifyCode}}</span>
    {{if .QRCodeSVG}}
    <span class="doc-qr">
      {{.QRCodeSVG}}
      <small>برای بررسی اصالت سند، این کد را با دوربین تلفن بخوانید.</small>
    </span>
    {{end}}
  </footer>
</div>
<script>
  // با باز شدن سند در قالب «چاپ»، پنجرهٔ چاپ خودکار باز می‌شود؛ این‌گونه کاربر یک
  // کلیک می‌کند و PDF ذخیره می‌شود، بدون هیچ ابزار نصب‌شدنی روی سرور.
  if (new URLSearchParams(location.search).get('print') === '1') {
    window.addEventListener('load', function () { window.print(); });
  }
</script>
</body>
</html>
`))
