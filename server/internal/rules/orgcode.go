// کدگذاری نمودار سازمانی — شمارهٔ سمت، پایهٔ سمت سازمانی، پایهٔ شخصی و سه کد همراه.
//
// چرا در موتور قواعد: این کدها شناسهٔ ماشین‌خوانِ تشکیلات‌اند و باید در همهٔ مصرف‌کننده‌ها
// (شناسنامهٔ پست، انتصاب، کارت ساختار سازمانی، سند چاپی و خروجی داده) یک قالب داشته باشند.
//
// قالب: «پیش‌شمارهٔ سه رقمی + نقطه + ده رقم»
//
//	080.0000000001   شمارهٔ سمت            (شناسهٔ یکتای پست سازمانی)
//	081.0000000001   پایهٔ سمت سازمانی     (پایهٔ مصوب خودِ پست)
//	082.0000000001   پایهٔ شخصی            (پایهٔ فرد از حکم ابلاغ‌شده)
//	083.0000000001   شمارهٔ واحد سازمانی   (کد واحد در نمودار)
//	084.0000000001   شمارهٔ انتصاب          (شناسهٔ یکتای رابطهٔ کارمند و پست)
//	085.0000000001   کد خانوادهٔ شغلی      (گروه/خانوادهٔ شغلی پست)
//
// **پیش‌شماره‌ها متغیرند، نه سخت‌کدشده:** مقدار پیش‌فرض هر پیش‌شماره همان فهرست بالاست و
// شرکت می‌تواند هر شش پیش‌شماره را در پارامترهای سازمانی (`orgcode.prefix.<کلید>`) عوض کند؛
// قالب، یکتایی و اعتبارسنجی از همان یک جا خوانده می‌شود.
package rules

import (
	"sort"
	"strconv"
	"strings"
)

// طول اجزای کد: سه رقم پیش‌شماره و ده رقم سریال.
const (
	OrgCodePrefixDigits = 3
	OrgCodeSerialDigits = 10
	// OrgCodeLength طول کامل کد با نقطهٔ جداکننده.
	OrgCodeLength = OrgCodePrefixDigits + 1 + OrgCodeSerialDigits
	// OrgCodeSerialMax بیشترین سریال پذیرفتنی.
	OrgCodeSerialMax = 9_999_999_999
)

// شناسهٔ نوع‌های کد نمودار سازمانی.
const (
	OrgCodeKindPost       = "post"
	OrgCodeKindOrgGrade   = "orgGrade"
	OrgCodeKindPersonal   = "personal"
	OrgCodeKindUnit       = "unit"
	OrgCodeKindAssignment = "assignment"
	OrgCodeKindJobFamily  = "jobFamily"
)

// OrgCodeKind یک نوع کد با پیش‌شمارهٔ مؤثر و عنوان فارسی.
type OrgCodeKind struct {
	Key string `json:"key"`
	// Prefix پیش‌شمارهٔ مؤثر؛ مقدار پیش‌فرض ۰۸۰ تا ۰۸۵ است و از پارامترهای سازمانی
	// (`orgcode.prefix.<کلید>`) قابل تغییر است.
	Prefix  string `json:"prefix"`
	TitleFa string `json:"titleFa"`
	NoteFa  string `json:"noteFa"`
}

// OrgCodeKindParam پیش‌شمارهٔ پیش‌فرض و کلید پارامتر هر نوع کد.
type OrgCodeKindParam struct {
	Key           string
	DefaultPrefix string
}

// orgCodeKindCatalog فهرست شش نوع کد با پیش‌شمارهٔ پیش‌فرض ۰۸۰ تا ۰۸۵.
var orgCodeKindCatalog = []OrgCodeKindParam{
	{Key: OrgCodeKindPost, DefaultPrefix: "080"},
	{Key: OrgCodeKindOrgGrade, DefaultPrefix: "081"},
	{Key: OrgCodeKindPersonal, DefaultPrefix: "082"},
	{Key: OrgCodeKindUnit, DefaultPrefix: "083"},
	{Key: OrgCodeKindAssignment, DefaultPrefix: "084"},
	{Key: OrgCodeKindJobFamily, DefaultPrefix: "085"},
}

// orgCodeTitles عنوان و توضیح فارسی هر نوع کد (کلید → متن).
var orgCodeTitles = map[string][2]string{
	OrgCodeKindPost: {"شمارهٔ سمت",
		"شناسهٔ یکتای پست سازمانی در تشکیلات؛ در شناسنامهٔ پست ثبت می‌شود."},
	OrgCodeKindOrgGrade: {"پایهٔ سمت سازمانی",
		"پایهٔ مصوب خودِ پست (مستقل از متصدی)؛ از نردبان پایه‌های سازمانی."},
	OrgCodeKindPersonal: {"پایهٔ شخصی",
		"پایهٔ فرد که از حکم ابلاغ‌شده خوانده می‌شود و با جابه‌جایی پست تغییر نمی‌کند."},
	OrgCodeKindUnit: {"شمارهٔ واحد سازمانی",
		"کد واحد در نمودار سازمانی؛ مبنای گروه‌بندی پست‌ها و گزارش‌های تشکیلات."},
	OrgCodeKindAssignment: {"شمارهٔ انتصاب",
		"شناسهٔ یکتای رابطهٔ کارمند و پست؛ با پایان تصدی پاک نمی‌شود."},
	OrgCodeKindJobFamily: {"کد خانوادهٔ شغلی",
		"کد خانواده/گروه شغلی پست؛ مبنای طبقه‌بندی مشاغل است."},
}

// OrgCodeKindKeys کلیدهای شش نوع کد به ترتیب پیش‌شماره.
func OrgCodeKindKeys() []string {
	out := make([]string, 0, len(orgCodeKindCatalog))
	for _, kind := range orgCodeKindCatalog {
		out = append(out, kind.Key)
	}
	return out
}

// OrgCodeKindParamKey کلید پارامتر پیش‌شمارهٔ یک نوع کد.
func OrgCodeKindParamKey(kind string) string {
	return "orgcode.prefix." + strings.TrimSpace(kind)
}

// OrgCodeKindTitleFa عنوان فارسی یک نوع کد؛ کلید ناشناخته خودش برمی‌گردد.
func OrgCodeKindTitleFa(kind string) string {
	if title, ok := orgCodeTitles[strings.TrimSpace(kind)]; ok {
		return title[0]
	}
	return strings.TrimSpace(kind)
}

// OrgCodeKindNoteFa توضیح فارسی یک نوع کد.
func OrgCodeKindNoteFa(kind string) string {
	if title, ok := orgCodeTitles[strings.TrimSpace(kind)]; ok {
		return title[1]
	}
	return ""
}

// OrgCodeScheme پیش‌شمارهٔ مؤثر شش نوع کد در یک استقرار.
type OrgCodeScheme struct {
	prefixes map[string]string
}

// DefaultOrgCodeScheme پیش‌شماره‌های پیش‌فرض (۰۸۰ تا ۰۸۵).
func DefaultOrgCodeScheme() OrgCodeScheme {
	scheme := OrgCodeScheme{prefixes: map[string]string{}}
	for _, kind := range orgCodeKindCatalog {
		scheme.prefixes[kind.Key] = kind.DefaultPrefix
	}
	return scheme
}

// OrgCodeSchemeFromValues پیش‌شماره‌ها را از جفت‌های «کلید پارامتر → مقدار» می‌سازد.
//
// کلید پارامتر: `orgcode.prefix.<کلید نوع کد>` (مثلاً orgcode.prefix.post). مقدار نامعتبر
// (غیررقم، غیرسه‌رقمی یا تکراری) نادیده گرفته می‌شود و پیش‌فرض می‌ماند، تا یک مقدار اشتباه
// همهٔ کدهای سازمان را خراب نکند.
func OrgCodeSchemeFromValues(values map[string]string) OrgCodeScheme {
	scheme := DefaultOrgCodeScheme()
	taken := map[string]bool{}
	for _, prefix := range scheme.prefixes {
		taken[prefix] = true
	}
	for _, kind := range orgCodeKindCatalog {
		raw, ok := values[OrgCodeKindParamKey(kind.Key)]
		if !ok {
			continue
		}
		value := strings.TrimSpace(NormalizeOrgCodeDigits(raw))
		if !validOrgPrefix(value) {
			continue
		}
		current := scheme.prefixes[kind.Key]
		if value != current && taken[value] {
			continue
		}
		delete(taken, current)
		scheme.prefixes[kind.Key] = value
		taken[value] = true
	}
	return scheme
}

// Prefix پیش‌شمارهٔ مؤثر یک نوع کد.
func (s OrgCodeScheme) Prefix(kind string) string {
	if s.prefixes == nil {
		return DefaultOrgCodeScheme().Prefix(kind)
	}
	return s.prefixes[strings.TrimSpace(kind)]
}

// View فهرست شش نوع با پیش‌شمارهٔ مؤثر (برای کارت مقادیر مجاز، فرم و مستندات).
func (s OrgCodeScheme) View() []OrgCodeKind {
	out := make([]OrgCodeKind, 0, len(orgCodeKindCatalog))
	for _, kind := range orgCodeKindCatalog {
		out = append(out, OrgCodeKind{
			Key: kind.Key, Prefix: s.Prefix(kind.Key),
			TitleFa: OrgCodeKindTitleFa(kind.Key), NoteFa: OrgCodeKindNoteFa(kind.Key),
		})
	}
	return out
}

// validOrgPrefix پیش‌شماره باید دقیقاً سه رقم باشد.
func validOrgPrefix(value string) bool {
	if len(value) != OrgCodePrefixDigits {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// kindOfPrefix نوع کد از پیش‌شمارهٔ مؤثر.
func (s OrgCodeScheme) kindOfPrefix(prefix string) (string, bool) {
	for _, kind := range orgCodeKindCatalog {
		if s.Prefix(kind.Key) == prefix {
			return kind.Key, true
		}
	}
	return "", false
}

// Format ساخت کد از نوع و سریال: «080.0000000001».
//
// سریال باید در بازهٔ ۱ تا ۹٬۹۹۹٬۹۹۹٬۹۹۹ باشد؛ صفر رزرو است و کد بی‌سریال معنا ندارد.
func (s OrgCodeScheme) Format(kind string, serial int64) (string, bool) {
	prefix := s.Prefix(kind)
	if !validOrgPrefix(prefix) {
		return "", false
	}
	if serial < 1 || serial > OrgCodeSerialMax {
		return "", false
	}
	digits := strconv.FormatInt(serial, 10)
	if len(digits) > OrgCodeSerialDigits {
		return "", false
	}
	return prefix + "." + strings.Repeat("0", OrgCodeSerialDigits-len(digits)) + digits, true
}

// OrgCode کد تجزیه‌شدهٔ نمودار سازمانی.
type OrgCode struct {
	Prefix string `json:"prefix"`
	Serial int64  `json:"serial"`
	Kind   string `json:"kind,omitempty"`
	// TitleFa عنوان فارسی نوع کد (برای پیام خطا و چاپ).
	TitleFa string `json:"titleFa,omitempty"`
}

// Parse تجزیهٔ کد با پیش‌شماره‌های همین استقرار.
func (s OrgCodeScheme) Parse(code string) (OrgCode, bool) {
	value := strings.TrimSpace(NormalizeOrgCodeDigits(code))
	if len(value) != OrgCodeLength || value[OrgCodePrefixDigits] != '.' {
		return OrgCode{}, false
	}
	prefix := value[:OrgCodePrefixDigits]
	kind, ok := s.kindOfPrefix(prefix)
	if !ok {
		return OrgCode{}, false
	}
	serialText := value[OrgCodePrefixDigits+1:]
	for _, r := range serialText {
		if r < '0' || r > '9' {
			return OrgCode{}, false
		}
	}
	serial, err := strconv.ParseInt(serialText, 10, 64)
	if err != nil || serial < 1 {
		return OrgCode{}, false
	}
	return OrgCode{Prefix: prefix, Serial: serial, Kind: kind, TitleFa: OrgCodeKindTitleFa(kind)}, true
}

// Valid آیا رشته یک کد معتبر این استقرار است.
func (s OrgCodeScheme) Valid(code string) bool {
	_, ok := s.Parse(code)
	return ok
}

// ValidFor آیا کد از نوع خواسته‌شده است (مثلاً شمارهٔ سمت با پیش‌شمارهٔ ۰۸۰).
func (s OrgCodeScheme) ValidFor(code, kind string) bool {
	parsed, ok := s.Parse(code)
	return ok && parsed.Kind == strings.TrimSpace(kind)
}

// Sample یک کد نمونه از یک نوع (برای پیام خطا و راهنما).
func (s OrgCodeScheme) Sample(kind string) string {
	code, ok := s.Format(kind, 1)
	if !ok {
		return s.Prefix(kind) + ".0000000000"
	}
	return code
}

// Next سریال بعدی یک نوع کد بر پایهٔ کدهای موجود (کدهای نامعتبر نادیده گرفته می‌شوند).
func (s OrgCodeScheme) Next(kind string, existing []string) (string, bool) {
	next := int64(1)
	for _, raw := range existing {
		parsed, ok := s.Parse(raw)
		if !ok || parsed.Kind != strings.TrimSpace(kind) {
			continue
		}
		if parsed.Serial >= next {
			next = parsed.Serial + 1
		}
	}
	return s.Format(kind, next)
}

// PersonalForEmployee کد پایهٔ شخصی از شمارهٔ پرسنلی.
//
// چرا از شمارهٔ پرسنلی: پایهٔ شخصی به «فرد» تعلق دارد نه به پست؛ ساختنش از شناسهٔ کارمند،
// یکتا و پایدار می‌ماند و با تغییر سمت عوض نمی‌شود. شناسهٔ بدون رقم، عدد پایدار می‌گیرد.
func (s OrgCodeScheme) PersonalForEmployee(employeeID string) string {
	return s.ForRecord(OrgCodeKindPersonal, employeeID)
}

// ForRecord کد یکتای یک رکورد (سمت، انتصاب، واحد، خانوادهٔ شغلی) از شناسهٔ آن.
func (s OrgCodeScheme) ForRecord(kind, recordID string) string {
	id := strings.TrimSpace(recordID)
	if id == "" {
		return ""
	}
	serial := int64(0)
	if text := digitsOf(id); text != "" {
		if value, err := strconv.ParseInt(text, 10, 64); err == nil {
			serial = value
		}
	}
	if serial < 1 {
		serial = stableSerial(id)
	}
	code, _ := s.Format(kind, serial)
	return code
}

// orgGradeLadder پایه‌های سازمانی رایج به ترتیب نزولی، برای ساختن کد پایه از گروه شغلی.
var orgGradeLadder = []string{
	"مدیریت عامل", "مدیریت", "سرپرستی", "کارشناسی ارشد", "کارشناسی", "کاردانی", "دیپلم",
}

// OrgGradeFor کد پایهٔ سمت سازمانی از عنوان گروه شغلی.
//
// گروه شناخته‌شده: شمارهٔ ردیف همان نردبان. گروه ناشناخته: یک عدد پایدار از متن، تا کد
// معتبر و تکرارشدنی بماند و دو پست هم‌عنوان یک کد بگیرند.
func (s OrgCodeScheme) OrgGradeFor(title string) string {
	value := strings.TrimSpace(title)
	if value == "" {
		return ""
	}
	serial := int64(0)
	for i, grade := range orgGradeLadder {
		if grade == value {
			serial = int64(i + 1)
			break
		}
	}
	if serial == 0 {
		serial = stableSerial(value)
	}
	code, _ := s.Format(OrgCodeKindOrgGrade, serial)
	return code
}

// JobFamilyFor کد خانوادهٔ شغلی از عنوان آن.
func (s OrgCodeScheme) JobFamilyFor(title string) string {
	value := strings.TrimSpace(title)
	if value == "" {
		return ""
	}
	code, _ := s.Format(OrgCodeKindJobFamily, stableSerial(value))
	return code
}

// UnitFor کد واحد سازمانی از نام (و در نبود آن، از شناسهٔ) واحد.
func (s OrgCodeScheme) UnitFor(unit string) string {
	value := strings.TrimSpace(unit)
	if value == "" {
		return ""
	}
	code, _ := s.Format(OrgCodeKindUnit, stableSerial(value))
	return code
}

// digitsOf تنها ارقام لاتین یک رشته را برمی‌گرداند (ارقام فارسی هم پذیرفته است).
func digitsOf(value string) string {
	var b strings.Builder
	for _, r := range NormalizeOrgCodeDigits(value) {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// stableSerial یک عدد پایدار از متن می‌سازد (برای شناسه‌های بدون رقم).
func stableSerial(value string) int64 {
	sum := int64(0)
	for _, r := range NormalizeOrgCodeDigits(value) {
		sum = (sum*31 + int64(r)) % OrgCodeSerialMax
	}
	return sum + 1
}

// NormalizeOrgCodeDigits ارقام فارسی/عربی کد را به لاتین برمی‌گرداند.
//
// کد در داده و API با ارقام لاتین ذخیره می‌شود؛ ولی کاربر در فرم ممکن است با صفحه‌کلید
// فارسی تایپ کند و کد نباید به همین دلیل رد شود.
func NormalizeOrgCodeDigits(code string) string {
	var b strings.Builder
	b.Grow(len(code))
	for _, r := range code {
		switch {
		case r >= '۰' && r <= '۹':
			b.WriteRune('0' + (r - '۰'))
		case r >= '٠' && r <= '٩':
			b.WriteRune('0' + (r - '٠'))
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// OrgCodeFa کد را با ارقام فارسی برای نمایش برمی‌گرداند (کد ماشین‌خوان دست‌نخورده می‌ماند).
func OrgCodeFa(code string) string {
	value := strings.TrimSpace(code)
	if value == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(value))
	for _, r := range value {
		if r >= '0' && r <= '9' {
			b.WriteRune('۰' + (r - '0'))
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// OrgCodePrefixParamKeys کلیدهای پارامتر شش پیش‌شماره (برای پنل پارامترهای سازمانی).
func OrgCodePrefixParamKeys() []string {
	keys := make([]string, 0, len(orgCodeKindCatalog))
	for _, kind := range orgCodeKindCatalog {
		keys = append(keys, OrgCodeKindParamKey(kind.Key))
	}
	sort.Strings(keys)
	return keys
}

// OrgCodePrefixValues جفت‌های «کلید پارامتر → پیش‌شمارهٔ مؤثر» برای نمایش و ذخیره.
func (s OrgCodeScheme) ParamValues() map[string]string {
	out := map[string]string{}
	for _, kind := range orgCodeKindCatalog {
		out[OrgCodeKindParamKey(kind.Key)] = s.Prefix(kind.Key)
	}
	return out
}
