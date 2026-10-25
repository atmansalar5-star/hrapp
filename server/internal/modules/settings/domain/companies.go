// شرکت‌ها و استقرارهای هلدینگ — فرم ورود اطلاعات شرکت تازه و ویرایش شرکت‌های موجود.
//
// قاعدهٔ معماری: نوع رکورد «شرکت/استقرار» در همین دامنه اعلام می‌شود و موتور مشترک
// `recordkit` گردشکار، اعتبارسنجی، سابقهٔ تغییرات و دسترسی را اجرا می‌کند؛ پس فرم وب و
// صفحهٔ موبایل هر دو از همین شناسه ساخته می‌شوند و قاعده‌ای در رابط تکرار نمی‌شود.
//
// تفکیک مهم:
//   - این رکوردها «دادهٔ پروفایل شرکت» هستند: نام، هلدینگ، برند، شمار کارکنان،
//     شناسه‌های قانونی (شناسهٔ ملی، کد اقتصادی، پروندهٔ مالیاتی، کد کارگاه)، تقویم و
//     سال مالی، ادارات فعال، ماژول‌های خاموش و پارامترهای بازنویسی‌شده.
//   - «استقرار فعال» خودِ سرور با پرچم `-org` هنگام اجرا تعیین می‌شود (ثبت ماژول‌ها در
//     زمان راه‌اندازی انجام می‌گیرد). فرم، شرکت تازه را ثبت می‌کند و برای فعال‌سازی
//     استقرارش، فرمان اجرا با همان کد شرکت را نشان می‌دهد؛ این کار عمداً خودکار نیست.
package domain

import (
	"regexp"
	"strconv"
	"strings"

	"hrms/internal/contracts"
	"hrms/internal/platform/calendar"
	"hrms/internal/platform/orgs"
	"hrms/internal/platform/recordkit"
)

// الگوهای اعتبارسنجی پروفایل شرکت؛ همه از دامنه اعلام می‌شوند تا فرم‌ها تکرارشان نکنند.
var (
	companyCodePattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]{1,31}$`)
	brandColorPattern  = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)
	paramKeyPattern    = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*(\.[a-zA-Z][a-zA-Z0-9_]*)+$`)
	codeListPattern    = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*$`)
)

// companyProfilesKind نوع رکورد «شرکت / استقرار».
func companyProfilesKind() recordkit.Kind {
	return recordkit.Kind{
		Key: "companyProfiles", TitleFa: "شرکت / استقرار", IDPrefix: "ORG", EmployeeScoped: false,
		Fields: recordkit.Fields(
			"code|کد شرکت (لاتین)|text|1",
			"titleFa|نام شرکت|text|1",
			"titleEn|نام لاتین|text|0",
			"holdingFa|هلدینگ|text|0",
			"employeeCodePrefix|پیشوند کد پرسنلی|text|0",
			"headcount|تعداد کارکنان|number|0",
			"brandColor|رنگ برند (#RRGGBB)|text|0",
			"logoFile|پروندهٔ نشان شرکت|text|0",
			"calendar|تقویم|text|0",
			"currencyFa|واحد پول|text|0",
			"workWeekFa|هفتهٔ کاری|text|0",
			"fiscalYearStart|آغاز سال مالی|text|0",
			"contactPhone|تلفن|text|0",
			"contactEmail|رایانامه|text|0",
			"addressFa|نشانی|text|0",
			"nationalId|شناسهٔ ملی|text|0",
			"economicCode|کد اقتصادی|text|0",
			"taxFileNumber|شمارهٔ پروندهٔ مالیاتی|text|0",
			"workshopCode|کد کارگاه تأمین اجتماعی|text|0",
			"enabledDepartments|ادارات فعال (کد با ویرگول)|text|0",
			"disabledModules|ماژول‌های خاموش (شناسه با ویرگول)|text|0",
			"paramOverrides|پارامترهای بازنویسی‌شده (هر خط: کلید=مقدار)|text|0",
			"isActiveDeployment|استقرار فعال این سرور|text|0",
			"notesFa|یادداشت|text|0",
		),
		Statuses: recordkit.Statuses(
			"draft|پیش‌نویس (در حال تکمیل اطلاعات)",
			"active|فعال",
			"archived|بایگانی‌شده",
		),
		Initial: "draft",
		// شرکت بایگانی‌شده حذف نمی‌شود؛ فقط از فهرست انتخاب‌ها کنار می‌رود تا سابقهٔ
		// اسناد و کارکنانش دست‌نخورده بماند.
		Transitions: recordkit.Steps(
			"draft|active", "active|archived", "archived|active",
		),
		Actions: recordkit.Actions(
			"activate|فعال‌سازی شرکت|primary|active|0|admin,hr_manager",
			"archive|بایگانی شرکت|ghost|archived|1|admin,hr_manager",
			"restore|بازگردانی شرکت|primary|active|0|admin,hr_manager",
		),
		Validate: validateCompanyProfile,
	}
}

// validateCompanyProfile قواعد پروفایل شرکت را می‌سنجد.
//
// اجباری‌بودن فیلدها را موتور مشترک می‌سنجد؛ اینجا فقط قواعدی است که شکل مقدار را
// تعیین می‌کنند: کد لاتین یکتا (شکل)، رنگ برند، عدد کارکنان، تقویم، فهرست کدها و
// خط‌های پارامتر.
func validateCompanyProfile(in contracts.RecordInput) error {
	code := strings.TrimSpace(calendar.NormalizeDigits(in.Fields["code"]))
	if !companyCodePattern.MatchString(code) {
		return ruleErr("کد شرکت باید با یک حرف لاتین آغاز شود و فقط حرف، رقم و زیرخط داشته باشد (۲ تا ۳۲ نویسه)؛ نمونه: bipc")
	}
	if title := strings.TrimSpace(in.Fields["titleFa"]); len([]rune(title)) < 3 {
		return ruleErr("نام شرکت باید دست‌کم سه نویسه باشد.")
	}
	if color := strings.TrimSpace(in.Fields["brandColor"]); color != "" && !brandColorPattern.MatchString(color) {
		return ruleErr("رنگ برند باید قالب #RRGGBB داشته باشد؛ نمونه: #1E88E5")
	}
	if raw := strings.TrimSpace(calendar.NormalizeDigits(in.Fields["headcount"])); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 {
			return ruleErr("تعداد کارکنان باید عدد صحیح غیرمنفی باشد.")
		}
	}
	if raw := strings.TrimSpace(calendar.NormalizeDigits(in.Fields["fiscalYearStart"])); raw != "" {
		if err := validateFiscalStart(raw); err != nil {
			return err
		}
	}
	if err := validateCodeList("ادارات فعال", in.Fields["enabledDepartments"]); err != nil {
		return err
	}
	if err := validateCodeList("ماژول‌های خاموش", in.Fields["disabledModules"]); err != nil {
		return err
	}
	return validateParamOverrides(in.Fields["paramOverrides"])
}

// validateFiscalStart آغاز سال مالی را می‌سنجد: «MM/DD» (رایج: 01/01) یا تاریخ کامل
// شمسی YYYY/MM/DD پذیرفته می‌شود.
func validateFiscalStart(raw string) error {
	if strings.Count(raw, "/") == 1 {
		parts := strings.Split(raw, "/")
		month, errM := strconv.Atoi(parts[0])
		day, errD := strconv.Atoi(parts[1])
		if errM != nil || errD != nil || month < 1 || month > 12 || day < 1 || day > 31 {
			return ruleErr("آغاز سال مالی باید MM/DD باشد (نمونه: 01/01).")
		}
		return nil
	}
	if _, err := calendar.Parse(raw); err != nil {
		return ruleErr("آغاز سال مالی باید MM/DD یا تاریخ شمسی YYYY/MM/DD باشد.")
	}
	return nil
}

// validateCodeList فهرست کدهای جداشده با ویرگول را می‌سنجد.
func validateCodeList(label, raw string) error {
	for _, item := range splitList(raw) {
		if !codeListPattern.MatchString(item) {
			return ruleErr("قلم «" + item + "» در " + label + " معتبر نیست؛ کد لاتین با ویرگول جدا شود (نمونه: staff_affairs,planning).")
		}
	}
	return nil
}

// validateParamOverrides خط‌های «کلید=مقدار» پارامترهای شرکت را می‌سنجد.
func validateParamOverrides(raw string) error {
	for _, line := range strings.Split(strings.ReplaceAll(raw, "\r\n", "\n"), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return ruleErr("هر خط پارامترها باید «کلید=مقدار» باشد؛ خط نامعتبر: " + line)
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if !paramKeyPattern.MatchString(key) {
			return ruleErr("کلید پارامتر «" + key + "» معتبر نیست؛ قالب نقطه‌دار لازم است (نمونه: payroll.monthlyHours).")
		}
		if value == "" {
			return ruleErr("مقدار پارامتر «" + key + "» خالی است.")
		}
	}
	return nil
}

// splitList رشتهٔ جداشده با ویرگول (لاتین یا فارسی) را به اقلام تمیزشده می‌شکند.
func splitList(raw string) []string {
	out := []string{}
	for _, chunk := range strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == '،' || r == ';' || r == '\n' || r == ' '
	}) {
		if item := strings.TrimSpace(chunk); item != "" {
			out = append(out, item)
		}
	}
	return out
}

// CompanySeed شرکت‌های همین استقرار را به رکورد «شرکت/استقرار» تبدیل می‌کند تا فرم
// ویرایش از همان ابتدا پر باشد و افزودن شرکت تازه کنار دادهٔ موجود بنشیند.
//
// منبع همین فهرست استقرارهاست (`platform/orgs` که از `contracts/organizations.json`
// تولید می‌شود)؛ پس رکوردهای فرم هیچ‌گاه از فهرست استقرارها جدا نمی‌افتند.
func CompanySeed() []contracts.RecordInput {
	active := orgs.Default().ID
	ids := orgs.IDs()
	out := make([]contracts.RecordInput, 0, len(ids))
	for _, id := range ids {
		org, ok := orgs.Lookup(id)
		if !ok {
			continue
		}
		isActive := "خیر"
		note := "شرکت ثبت‌شده در فهرست استقرارهای هلدینگ."
		if org.ID == active {
			isActive = "بله"
			note = "استقرار فعال همین سرور است (پرچم -org). ویرایش اطلاعات این شرکت بی‌درنگ در رابط دیده می‌شود."
		}
		out = append(out, contracts.RecordInput{
			Kind: "companyProfiles", Title: org.TitleFa, SeedStatus: "active",
			Fields: map[string]string{
				"code":               org.ID,
				"titleFa":            org.TitleFa,
				"titleEn":            org.TitleEn,
				"holdingFa":          org.HoldingFa,
				"employeeCodePrefix": org.EmployeeCodePrefix,
				"headcount":          strconv.Itoa(org.Headcount),
				"brandColor":         org.BrandColor,
				"logoFile":           org.LogoFile,
				"calendar":           org.Calendar,
				"currencyFa":         org.CurrencyFa,
				"workWeekFa":         org.WorkWeekFa,
				"fiscalYearStart":    org.FiscalYearStartFa,
				"enabledDepartments": strings.Join(org.EnabledDepartments, ","),
				"disabledModules":    strings.Join(org.DisabledModules, ","),
				"paramOverrides":     paramOverridesText(org),
				"isActiveDeployment": isActive,
				"notesFa":            strings.TrimSpace(org.NotesFa),
			},
			NoteFa: note,
		})
	}
	return out
}

// paramOverridesText پارامترهای یک شرکت را به متن «کلید=مقدار» تبدیل می‌کند.
func paramOverridesText(org orgs.Organization) string {
	lines := make([]string, 0, len(org.Params))
	for _, p := range org.Params {
		if p.Key == "" {
			continue
		}
		lines = append(lines, p.Key+"="+p.Value)
	}
	return strings.Join(lines, "\n")
}

// ruleErr خطای قاعدهٔ دامنه با پیام فارسی؛ در لایهٔ HTTP به ۴۲۲ تبدیل می‌شود.
func ruleErr(msg string) *recordkit.RuleError { return &recordkit.RuleError{Message: msg} }
