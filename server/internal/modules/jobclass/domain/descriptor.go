// Package domain دامنهٔ ماژول طبقه‌بندی مشاغل.
//
// طبقه‌بندی شغلی، ارزشیابی مشاغل و شرح شغل هر یک یک «نوع رکورد» است؛ قواعد اینجا اعلام
// می‌شود و موتور مشترک `recordkit` آن را اجرا می‌کند.
//
// سه سند منابع انسانی که این ماژول می‌سازد از هم جدا هستند و نباید در هم بیامیزند:
//
//	شرح شغل      ← رکورد همین ماژول (نوع «jobDescriptions»): هدف شغل، وظایف، مسئولیت‌ها،
//	               اختیارات، شاخص‌ها، شرایط کار و شرایط احراز.
//	شرح مشاغل    ← مجموع شرح شغل‌های سازمان؛ فهرست آن از همین رکوردها ساخته می‌شود
//	               (هیچ دادهٔ تازه‌ای ذخیره نمی‌کند، فقط نمای فهرستی است).
//	شناسنامهٔ شغل ← کارت یک‌صفحه‌ای که از شرح شغل + طبقه + ارزشیابی + شرایط احراز ساخته
//	               می‌شود؛ همیشه از دادهٔ جاری خوانده می‌شود تا کارت و شرح شغل واگرا نشوند.
package domain

import (
	"errors"
	"sort"
	"strconv"
	"strings"

	"hrms/internal/contracts"
	"hrms/internal/platform/calendar"
	"hrms/internal/platform/recordkit"
	"hrms/internal/rules"
)

// کلیدهای نوع رکورد این ماژول.
const (
	KindCategories   = "jobCategories"
	KindEvaluations  = "jobEvaluations"
	KindDescriptions = "jobDescriptions"
)

// کلیدهای فیلد شرح شغل که بیرون از این بسته هم خوانده می‌شوند (کارت شغل، انتصاب).
const (
	FieldJobCode           = "jobCode"
	FieldJobTitle          = "jobTitle"
	FieldJobGroup          = "jobGroup"
	FieldJobRank           = "jobRank"
	FieldUnit              = "unit"
	FieldLocation          = "location"
	FieldReportsTo         = "reportsTo"
	FieldSubordinates      = "subordinates"
	FieldPurpose           = "purpose"
	FieldDuties            = "duties"
	FieldResponsibilities  = "responsibilities"
	FieldAuthorities       = "authorities"
	FieldKPIs              = "kpis"
	FieldTools             = "tools"
	FieldWorkingConditions = "workingConditions"
	FieldHazards           = "hazards"
	FieldDegreeLevel       = "degreeLevel"
	FieldFields            = "fields"
	FieldMinExperience     = "minExperienceYears"
	FieldSkills            = "skills"
	FieldOtherRequirements = "otherRequirements"
	FieldRequirements      = "requirements"
	FieldCareerPath        = "careerPath"
	FieldRevisionDate      = "revisionDate"
)

// وضعیت‌های شرح شغل.
const (
	StatusDraft    = "draft"
	StatusApproved = "approved"
	StatusRevised  = "revised"
)

// RuleErr خطای نقض قاعدهٔ دامنه (بدون وابستگی به لایهٔ HTTP).
func ruleErr(msg string) *recordkit.RuleError { return &recordkit.RuleError{Message: msg} }

// Descriptor شناسهٔ ماژول طبقه‌بندی مشاغل.
func Descriptor() recordkit.Descriptor {
	return recordkit.Descriptor{
		Module:  "jobclass",
		TitleFa: "طبقه‌بندی مشاغل",
		Kinds: []recordkit.Kind{
			{
				Key: KindCategories, TitleFa: "طبقه‌بندی شغلی", IDPrefix: "JOB", EmployeeScoped: false,
				Fields: recordkit.Fields(
					"code|کد طبقه|text|1",
					"groupName|گروه شغلی|text|1",
					"grade|رتبه|text|0",
				),
				Statuses:    recordkit.Statuses("draft|پیش‌نویس", "approved|تأییدشده", "active|جاری"),
				Initial:     "draft",
				Transitions: recordkit.Steps("draft|approved", "approved|active"),
				Actions: recordkit.Actions(
					"approve|تأیید طبقه|primary|approved|0",
					"activate|جاری‌سازی|primary|active|0",
				),
			},
			{
				Key: KindEvaluations, TitleFa: "ارزشیابی شغل", IDPrefix: "EVA", EmployeeScoped: false,
				Fields: recordkit.Fields(
					"jobTitle|عنوان شغل|text|1",
					"points|امتیاز ارزشیابی|number|0",
					"factor|عوامل ارزشیابی|text|0",
				),
				Statuses:    recordkit.Statuses("draft|پیش‌نویس", "scored|امتیازدهی‌شده", "approved|تأییدشده"),
				Initial:     "draft",
				Transitions: recordkit.Steps("draft|scored", "scored|approved"),
				Actions: recordkit.Actions(
					"score|ثبت امتیاز|primary|scored|0",
					"approve|تأیید ارزشیابی|primary|approved|0",
				),
			},
			{
				// شرح شغل: سند مرجع شناسنامهٔ شغل و سنجش شرایط احراز در انتصاب و ارتقا.
				// ترتیب فیلدها همان ترتیب سند چاپی است: شناسهٔ شغل → جایگاه → وظایف →
				// شرایط کار → شرایط احراز → مسیر ارتقا.
				Key: KindDescriptions, TitleFa: "شرح شغل", IDPrefix: "JDD", EmployeeScoped: false,
				Fields: recordkit.Fields(
					FieldJobCode+"|کد شغل|text|0",
					FieldJobTitle+"|عنوان شغل|text|1",
					FieldJobGroup+"|گروه شغلی (نردبان سازمان)|select|0",
					FieldJobRank+"|رتبهٔ طبقه|text|0",
					FieldUnit+"|واحد سازمانی|text|0",
					FieldLocation+"|محل استقرار|text|0",
					FieldReportsTo+"|گزارش‌دهی به|text|0",
					FieldSubordinates+"|تعداد زیرمجموعه|number|0",
					FieldPurpose+"|هدف و جایگاه شغل|text|0",
					FieldDuties+"|شرح وظایف|text|1",
					FieldResponsibilities+"|مسئولیت‌ها|text|0",
					FieldAuthorities+"|اختیارات|text|0",
					FieldKPIs+"|شاخص‌های کلیدی عملکرد|text|0",
					FieldTools+"|ابزار و تجهیزات|text|0",
					FieldWorkingConditions+"|شرایط کار|text|0",
					FieldHazards+"|مخاطرات شغلی|text|0",
					FieldDegreeLevel+"|مقطع تحصیلی لازم|select|1",
					FieldFields+"|رشته‌های تحصیلی مرتبط|text|0",
					FieldMinExperience+"|کمینهٔ سابقهٔ مرتبط (سال)|number|1",
					FieldSkills+"|مهارت‌ها و دوره‌های لازم|text|0",
					FieldOtherRequirements+"|سایر شرایط احراز|text|0",
					FieldRequirements+"|شرایط احراز — متن تکمیلی|text|0",
					FieldCareerPath+"|مسیر ارتقا (پست‌های بعدی)|text|0",
					FieldRevisionDate+"|تاریخ بازنگری|date|0",
				),
				Statuses:    recordkit.Statuses("draft|پیش‌نویس", "approved|تأییدشده", "revised|بازنگری‌شده"),
				Initial:     StatusDraft,
				Transitions: recordkit.Steps("draft|approved", "approved|revised", "revised|approved"),
				Actions: recordkit.Actions(
					"approve|تأیید شرح شغل|primary|approved|0",
					"revise|ثبت بازنگری|ghost|revised|0",
				),
				Validate:         validateDescription,
				EditableStatuses: []string{StatusDraft, StatusRevised},
			},
		},
	}
}

// validateDescription قاعدهٔ شرح شغل: مقطع تحصیلی باید از نردبان مدرک باشد، رشته‌ها
// فهرست‌شدنی بمانند (با ویرگول از هم جدا)، و اعداد شرط احراز عدد صحیح غیرمنفی باشند.
//
// چرا اینجا: شرایط احراز مبنای سنجش انتصاب و ارتقاست؛ اگر مقطع ناشناخته یا سابقهٔ
// منفی ذخیره شود، فرم محاسباتی عدد بی‌معنا می‌سازد.
func validateDescription(in contracts.RecordInput) error {
	if strings.TrimSpace(in.Fields[FieldDuties]) == "" {
		return ruleErr("شرح وظایف شغل الزامی است.")
	}
	degree := strings.TrimSpace(in.Fields[FieldDegreeLevel])
	if degree == "" {
		return ruleErr("مقطع تحصیلی لازم را از نردبان مدرک انتخاب کنید.")
	}
	if _, ok := rules.DegreeLevelByKey(degree); !ok {
		return ruleErr("مقطع تحصیلی لازم در نردبان مدرک شناخته‌شده نیست.")
	}
	if raw := strings.TrimSpace(in.Fields[FieldMinExperience]); raw != "" {
		n, err := strconv.Atoi(calendar.NormalizeDigits(raw))
		if err != nil || n < 0 {
			return ruleErr("کمینهٔ سابقهٔ مرتبط باید عدد صحیح غیرمنفی (سال) باشد.")
		}
	}
	if raw := strings.TrimSpace(in.Fields[FieldSubordinates]); raw != "" {
		n, err := strconv.Atoi(calendar.NormalizeDigits(raw))
		if err != nil || n < 0 {
			return ruleErr("تعداد زیرمجموعه باید عدد صحیح غیرمنفی باشد.")
		}
	}
	if raw := strings.TrimSpace(in.Fields[FieldJobGroup]); raw != "" && rules.GradeRankOf(raw) == 0 {
		return ruleErr("گروه شغلی باید از نردبان گروه‌های سازمان انتخاب شود.")
	}
	if raw := strings.TrimSpace(in.Fields[FieldRevisionDate]); raw != "" {
		if _, err := calendar.Parse(raw); err != nil {
			return ruleErr("تاریخ بازنگری باید به قالب ششمسی YYYY/MM/DD باشد.")
		}
	}
	return nil
}

// Requirement شرایط احراز یک شرح شغل را از فیلدهای آن می‌خواند.
//
// چرا این تابع در دامنه: هم فرم محاسباتی شرایط احراز، هم کارت شغل و هم تصویب‌خواهی انتصاب
// باید یک شرط واحد ببینند؛ اگر هر کدام فیلدها را خودش بخواند، یک روز دو عدد می‌سازند.
func Requirement(fields map[string]string) rules.JobRequirement {
	req := rules.JobRequirement{
		DegreeLevel: strings.TrimSpace(fields[FieldDegreeLevel]),
		Fields:      FieldList(fields[FieldFields]),
		Skills:      FieldList(fields[FieldSkills]),
	}
	if raw := strings.TrimSpace(fields[FieldMinExperience]); raw != "" {
		if n, err := strconv.Atoi(calendar.NormalizeDigits(raw)); err == nil && n > 0 {
			req.MinExperienceYears = n
		}
	}
	return req
}

// FieldList فهرست نوشته‌شده در یک فیلد متنی را به برش تبدیل می‌کند. جداکننده‌ها: ویرگول
// فارسی و لاتین، نقطه‌ویرگول و خط تیره‌های فهرست‌ساز.
func FieldList(raw string) []string {
	normalized := strings.NewReplacer("،", ",", "؛", ",", "\n", ",", "\r", ",", "٬", ",").Replace(raw)
	parts := strings.Split(normalized, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	return out
}

// CardBlock یک بخش از شناسنامهٔ شغل (عنوان + سطرهای «برچسب: مقدار»).
type CardBlock struct {
	Key     string    `json:"key"`
	TitleFa string    `json:"titleFa"`
	Rows    []CardRow `json:"rows"`
}

// CardRow یک سطر شناسنامه.
type CardRow struct {
	LabelFa string `json:"labelFa"`
	ValueFa string `json:"valueFa"`
	// Long یعنی مقدار متنی بلند است و در چاپ تمام‌عرض نمایش داده شود.
	Long bool `json:"long,omitempty"`
}

// JobCard شناسنامهٔ شغل: کارت یک‌صفحه‌ای یک شغل با بخش‌های آمادهٔ چاپ.
type JobCard struct {
	RecordID   string `json:"recordId"`
	JobCode    string `json:"jobCode,omitempty"`
	TitleFa    string `json:"titleFa"`
	GroupFa    string `json:"groupFa,omitempty"`
	RankFa     string `json:"rankFa,omitempty"`
	UnitFa     string `json:"unitFa,omitempty"`
	LocationFa string `json:"locationFa,omitempty"`
	Status     string `json:"status"`
	StatusFa   string `json:"statusFa"`
	// RequirementFa خلاصهٔ فارسی شرایط احراز و Assessment راهنمای فرم محاسباتی است.
	RequirementFa string `json:"requirementFa"`
	// Approved آیا این شناسنامه مبنای انتصاب و ارتقا است (شرح شغل تأییدشده).
	Approved bool        `json:"approved"`
	Blocks   []CardBlock `json:"blocks"`
	// ExecutionPlanFa گام‌های اجرای انتصاب بر پایهٔ همین شناسنامه.
	ExecutionPlanFa []string `json:"executionPlanFa"`
	NoteFa          string   `json:"noteFa,omitempty"`
}

// BuildCard شناسنامهٔ شغل را از یک رکورد شرح شغل می‌سازد.
//
// چرا از رکورد و نه دادهٔ جداگانه: شناسنامه یک «نما» است نه یک سند مستقل؛ اگر جدا ذخیره
// شود، پس از هر بازنگری شرح شغل یک نسخهٔ کهنه از کارت در سازمان می‌ماند.
func BuildCard(rec contracts.Record) JobCard {
	fields := rec.Fields
	groupFa := ""
	if group := strings.TrimSpace(fields[FieldJobGroup]); group != "" {
		if rank := rules.GradeRankOf(group); rank > 0 {
			groupFa = group + " — " + rules.GradeTitleFa(rank)
		} else {
			groupFa = group
		}
	}
	card := JobCard{
		RecordID:      rec.ID,
		JobCode:       strings.TrimSpace(fields[FieldJobCode]),
		TitleFa:       strings.TrimSpace(rec.Title),
		GroupFa:       groupFa,
		RankFa:        strings.TrimSpace(fields[FieldJobRank]),
		UnitFa:        strings.TrimSpace(fields[FieldUnit]),
		LocationFa:    strings.TrimSpace(fields[FieldLocation]),
		Status:        rec.Status,
		StatusFa:      rec.StatusFa,
		RequirementFa: rules.RequirementFa(Requirement(fields)),
		Approved:      rec.Status == StatusApproved || rec.Status == StatusRevised,
		NoteFa:        rec.NoteFa,
	}

	identity := []CardRow{
		{LabelFa: "عنوان شغل", ValueFa: card.TitleFa},
		{LabelFa: "کد شغل", ValueFa: orDash(card.JobCode)},
		{LabelFa: "گروه شغلی", ValueFa: orDash(groupFa)},
		{LabelFa: "رتبهٔ طبقه", ValueFa: orDash(card.RankFa)},
		{LabelFa: "واحد سازمانی", ValueFa: orDash(card.UnitFa)},
		{LabelFa: "محل استقرار", ValueFa: orDash(card.LocationFa)},
		{LabelFa: "گزارش‌دهی به", ValueFa: orDash(fields[FieldReportsTo])},
		{LabelFa: "تعداد زیرمجموعه", ValueFa: orDash(fields[FieldSubordinates])},
		{LabelFa: "وضعیت شناسنامه", ValueFa: rec.StatusFa},
	}
	card.Blocks = append(card.Blocks, CardBlock{Key: "identity", TitleFa: "شناسهٔ شغل", Rows: identity})

	purpose := []CardRow{{LabelFa: "هدف و جایگاه شغل", ValueFa: orDash(fields[FieldPurpose]), Long: true}}
	card.Blocks = append(card.Blocks, CardBlock{Key: "purpose", TitleFa: "هدف شغل", Rows: purpose})

	card.Blocks = append(card.Blocks, CardBlock{Key: "duties", TitleFa: "وظایف و مسئولیت‌ها", Rows: []CardRow{
		{LabelFa: "شرح وظایف", ValueFa: orDash(fields[FieldDuties]), Long: true},
		{LabelFa: "مسئولیت‌ها", ValueFa: orDash(fields[FieldResponsibilities]), Long: true},
		{LabelFa: "اختیارات", ValueFa: orDash(fields[FieldAuthorities]), Long: true},
		{LabelFa: "شاخص‌های کلیدی عملکرد", ValueFa: orDash(fields[FieldKPIs]), Long: true},
	}})

	card.Blocks = append(card.Blocks, CardBlock{Key: "conditions", TitleFa: "شرایط کار", Rows: []CardRow{
		{LabelFa: "ابزار و تجهیزات", ValueFa: orDash(fields[FieldTools]), Long: true},
		{LabelFa: "شرایط محیط کار", ValueFa: orDash(fields[FieldWorkingConditions]), Long: true},
		{LabelFa: "مخاطرات شغلی", ValueFa: orDash(fields[FieldHazards]), Long: true},
	}})

	req := Requirement(fields)
	requirementRows := []CardRow{
		{LabelFa: "مقطع تحصیلی", ValueFa: rules.DegreeTitleFa(req.DegreeLevel)},
		{LabelFa: "رشته‌های تحصیلی", ValueFa: orDash(strings.Join(req.Fields, " / ")), Long: true},
		{LabelFa: "کمینهٔ سابقهٔ مرتبط", ValueFa: fa(rules.PersianDigitsInt(req.MinExperienceYears)) + " سال"},
		{LabelFa: "مهارت‌ها و دوره‌ها", ValueFa: orDash(strings.Join(req.Skills, " / ")), Long: true},
		{LabelFa: "سایر شرایط", ValueFa: orDash(fields[FieldOtherRequirements]), Long: true},
		{LabelFa: "شرایط احراز — متن تکمیلی", ValueFa: orDash(fields[FieldRequirements]), Long: true},
	}
	card.Blocks = append(card.Blocks, CardBlock{Key: "requirements", TitleFa: "شرایط احراز", Rows: requirementRows})

	card.Blocks = append(card.Blocks, CardBlock{Key: "career", TitleFa: "مسیر ارتقا و بازنگری", Rows: []CardRow{
		{LabelFa: "مسیر ارتقا", ValueFa: orDash(fields[FieldCareerPath]), Long: true},
		{LabelFa: "تاریخ بازنگری", ValueFa: orDash(fields[FieldRevisionDate])},
	}})

	card.ExecutionPlanFa = []string{
		"۱. احراز تأیید شرح شغل: شناسنامهٔ تأییدنشده مبنای انتصاب یا ارتقا نیست.",
		"۲. سنجش شرایط احراز با فرم محاسباتی: مدرک، رشته و سابقهٔ مرتبط متقاضی با همین شناسنامه سنجیده می‌شود.",
		"۳. اگر شرط‌ها بی‌واسطه برآورده نشد، شرایط معادل (سابقهٔ مازاد) در همان فرم محاسبه و مستند می‌شود.",
		"۴. تصویب‌خواهی انتصاب یا ارتقا با همین شناسنامه و نتیجهٔ فرم محاسباتی ساخته می‌شود.",
		"۵. پس از تصویب همهٔ سطوح، ابلاغ صادر و پروندهٔ کارمند به‌روز می‌شود.",
	}
	return card
}

// QualificationNoteFa یادداشت مشترک فرم محاسباتی و کارت شغل: چه چیزی مبنا است و چه چیزی
// نیست. چرا در دامنه: همین جمله در فرم وب، نسخهٔ چاپی و کارتابل تصویب دیده می‌شود و باید
// یکی باشد.
func QualificationNoteFa() string {
	return "این سنجش بر پایهٔ شناسنامهٔ شغل مصوب و پارامترهای شرکت انجام شده است؛ " +
		"نتیجهٔ «برآورده با شرایط معادل» باید در تصویب‌خواهی مستند شود و کمبود جبران‌نشده " +
		"مانع انتصاب و ارتقاست. شرح شغل پیش‌نویس مبنای سنجش نیست و باید نخست تأیید شود."
}

// orDash مقدار خالی را با خط تیرهٔ نمایشی پر می‌کند.
func orDash(value string) string {
	if v := strings.TrimSpace(value); v != "" {
		return v
	}
	return "—"
}

// fa رقم‌های لاتین یک متن را فارسی می‌کند (نمایش).
func fa(value string) string { return rules.PersianDigits(value) }

// ApprovedDescriptions شرح شغل‌های تأییدشده را از میان رکوردها برمی‌گرداند، مرتب بر
// اساس عنوان. مبنای «شرح مشاغل» و انتخاب شغل در فرم انتصاب همین فهرست است.
func ApprovedDescriptions(items []contracts.Record) []contracts.Record {
	out := make([]contracts.Record, 0, len(items))
	for _, rec := range items {
		if rec.Kind != KindDescriptions {
			continue
		}
		if rec.Status != StatusApproved && rec.Status != StatusRevised {
			continue
		}
		out = append(out, rec)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Title < out[j].Title })
	return out
}

// FindDescription شرح شغل یک شغل را با کد شغل یا عنوان پیدا می‌کند. اولویت با کد است،
// چون عنوان در سازمان ممکن است به دو صورت نوشته شود.
func FindDescription(items []contracts.Record, codeOrTitle string) (contracts.Record, bool) {
	target := normalizeText(codeOrTitle)
	if target == "" {
		return contracts.Record{}, false
	}
	for _, rec := range items {
		if rec.Kind != KindDescriptions {
			continue
		}
		if normalizeText(rec.Fields[FieldJobCode]) == target {
			return rec, true
		}
	}
	for _, rec := range items {
		if rec.Kind != KindDescriptions {
			continue
		}
		if normalizeText(rec.Title) == target || normalizeText(rec.Fields[FieldJobTitle]) == target {
			return rec, true
		}
	}
	return contracts.Record{}, false
}

// normalizeText متن را برای مقایسه یکدست می‌کند (حروف عربی/فارسی، نیم‌فاصله، فاصله).
func normalizeText(value string) string {
	replacer := strings.NewReplacer(
		"ي", "ی", "ك", "ک", "ۀ", "ه", "ة", "ه", "\u200c", " ", "\u200f", "", "\u200e", "",
	)
	return strings.Join(strings.Fields(replacer.Replace(strings.TrimSpace(value))), " ")
}

// ErrNoDescription خطای «شرح شغل یافت نشد» برای مصرف‌کنندگان دامنه.
var ErrNoDescription = errors.New("شرح شغل تأییدشده‌ای با این کد یا عنوان یافت نشد")

// Seed رکوردهای نمونهٔ سازمان.
func Seed() []contracts.RecordInput {
	return []contracts.RecordInput{
		{Kind: KindCategories, Title: "گروه شغلی مالی و حسابداری",
			Fields: map[string]string{"code": "FIN", "groupName": "مالی و حسابداری", "grade": "کارشناسی تا ارشد"}},
		{Kind: KindCategories, Title: "گروه شغلی تولید",
			Fields: map[string]string{"code": "PRD", "groupName": "تولید و عملیات", "grade": "عملیاتی تا مدیریت"}},
		{Kind: KindEvaluations, Title: "ارزشیابی شغل کارشناس ارشد مالی",
			Fields: map[string]string{"jobTitle": "کارشناس ارشد مالی", "points": "720", "factor": "دانش، پیچیدگی، مسئولیت"},
			NoteFa: "امتیاز پایهٔ ترفیع"},
		{
			Kind: KindDescriptions, Title: "سرپرست فروش", SeedStatus: StatusApproved,
			Fields: map[string]string{
				FieldJobCode:           "SAL-SUP-01",
				FieldJobTitle:          "سرپرست فروش",
				FieldJobGroup:          "supervisor",
				FieldJobRank:           "سرپرستی فروش",
				FieldUnit:              "بازرگانی و فروش",
				FieldLocation:          "ستاد — بندر امام",
				FieldReportsTo:         "مدیر بازرگانی و فروش",
				FieldSubordinates:      "6",
				FieldPurpose:           "هدایت تیم فروش برای تحقق برنامهٔ فروش، حفظ سهم بازار و وصول مطالبات.",
				FieldDuties:            "هدف‌گذاری فروش ماهانه و سالانه، پایش عملکرد کارشناسان فروش، مذاکره با مشتریان کلیدی، پیگیری وصول مطالبات، تهیهٔ گزارش فروش دوره‌ای.",
				FieldResponsibilities:  "پاسخ‌گویی دربارهٔ تحقق برنامهٔ فروش، صحت قراردادهای فروش و رعایت سیاست‌های قیمت‌گذاری.",
				FieldAuthorities:       "تأیید تخفیف در سقف مصوب، تخصیص مشتریان به کارشناسان، پیشنهاد استخدام کارشناس فروش.",
				FieldKPIs:              "درصد تحقق برنامهٔ فروش، دورهٔ وصول مطالبات، سهم بازار، رضایت مشتریان کلیدی.",
				FieldTools:             "سامانهٔ فروش و مشتریان، اتومبیل سازمانی، خط تلفن مستقیم.",
				FieldWorkingConditions: "کار در دفتر با سفرهای دوره‌ای به سایت مشتریان؛ فشار زمانی پایان دوره.",
				FieldHazards:           "رانندگی در مسیرهای بین‌شهری.",
				FieldDegreeLevel:       "bachelor",
				FieldFields:            "مدیریت, مدیریت بازرگانی, حسابداری, مهندسی صنایع",
				FieldMinExperience:     "5",
				FieldSkills:            "دورهٔ مذاکرهٔ حرفه‌ای, نرم‌افزار فروش, اکسل پیشرفته",
				FieldOtherRequirements: "گواهی سلامت جسمانی، عدم سوءپیشینه.",
				FieldCareerPath:        "مدیر بازرگانی و فروش، کارشناس ارشد فروش",
				FieldRevisionDate:      "1404/06/01",
			},
			NoteFa: "شناسنامهٔ شغل نمونه برای سنجش شرایط احراز انتصاب و ارتقا.",
		},
		{
			Kind: KindDescriptions, Title: "کارشناس ارشد مالی", SeedStatus: StatusApproved,
			Fields: map[string]string{
				FieldJobCode:           "FIN-EXP-03",
				FieldJobTitle:          "کارشناس ارشد مالی",
				FieldJobGroup:          "senior_expert",
				FieldJobRank:           "کارشناسی ارشد مالی",
				FieldUnit:              "امور مالی",
				FieldLocation:          "ستاد — بندر امام",
				FieldReportsTo:         "مدیر امور مالی",
				FieldSubordinates:      "0",
				FieldPurpose:           "تحلیل و کنترل صورت‌های مالی، تهیهٔ گزارش‌های مدیریتی و کنترل انطباق با استانداردهای حسابداری.",
				FieldDuties:            "تهیهٔ صورت‌های مالی دوره‌ای، تحلیل انحراف بودجه، کنترل اسناد حسابداری، پاسخ به حسابرسان، تهیهٔ گزارش مدیریتی ماهانه.",
				FieldResponsibilities:  "صحت و کامل‌بودن صورت‌های مالی و رعایت مهلت‌های قانونی گزارش‌دهی.",
				FieldAuthorities:       "رد اسناد مالی مغایر، درخواست اصلاح سند از واحدهای دیگر.",
				FieldKPIs:              "به‌موقع‌بودن صورت‌های مالی، تعداد مغایرت حسابرسي، دقت تحلیل انحراف بودجه.",
				FieldTools:             "نرم‌افزار حسابداری، اکسل پیشرفته.",
				FieldWorkingConditions: "کار در دفتر؛ اوج کار در پایان دورهٔ مالی.",
				FieldHazards:           "ندارد.",
				FieldDegreeLevel:       "master",
				FieldFields:            "حسابداری, مدیریت مالی, اقتصاد",
				FieldMinExperience:     "6",
				FieldSkills:            "استانداردهای حسابداری, گزارش‌گیری مالی, اکسل پیشرفته",
				FieldOtherRequirements: "تسلط بر استانداردهای حسابداری ایران.",
				FieldCareerPath:        "مدیر امور مالی، رئیس حسابداری",
				FieldRevisionDate:      "1404/06/01",
			},
		},
	}
}
