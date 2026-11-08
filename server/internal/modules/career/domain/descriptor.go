// Package domain دامنهٔ ماژول انتصابات، ترفیعات و کارراههٔ شغلی.
//
// قواعد این ماژول به‌صورت «شناسهٔ رکورد» اعلام می‌شود: نوع‌های رکورد، فیلدهای هر نوع،
// وضعیت‌ها، انتقال‌های مجاز و اقدام‌ها. موتور مشترک `recordkit` همین شناسه را اجرا
// می‌کند، پس قاعده اینجا می‌ماند و کد تکراری نوشته نمی‌شود.
//
// گردش‌کار انتصاب و ترفیع («تصویب‌خواهی و اجرا»):
//
//	پیش‌نویس ──ارسال برای تصویب──▶ در انتظار تصویب ──تصویب──▶ تصویب‌شده ──اجرا و ابلاغ──▶ ابلاغ‌شده
//	   └──────────────────────────── لغو ────────────────────────────┘
//
// چرا سه گام و نه دو گام: «تصویب» تصمیم است و «اجرا» اقدام سازمانی؛ میان این دو، شمارهٔ
// ابلاغ و تاریخ اجرا ثبت می‌شود. اگر اجرا بی‌ابلاغ انجام شود، سابقهٔ کارمند و حقوق
// دستمزد بی‌مستند می‌ماند.
package domain

import (
	"strconv"
	"strings"

	"hrms/internal/contracts"
	"hrms/internal/platform/calendar"
	"hrms/internal/platform/recordkit"
	"hrms/internal/rules"
)

// وضعیت‌های مشترک گردش‌کار ابلاغ سازمانی.
const (
	StatusDraft     = "draft"
	StatusPending   = "pending"
	StatusApproved  = "approved"
	StatusIssued    = "issued"
	StatusRevoked   = "revoked"
	StatusCancelled = "cancelled"
)

// کلیدهای فیلدهای گردش‌کار تصویب.
//
// چرا فیلد و نه وضعیت جدا برای هر سطح: شمار سطوح تصویب پارامتریک است (۲ تا ۴ سطح بر پایهٔ
// گروه شغلی و نوع درخواست) و وضعیت‌های ثابت نمی‌توانند شمار متغیر سطح‌ها را نشان دهند.
// این فیلدها فقط در «پیش‌نویس» قابل ویرایش‌اند (EditableStatuses) و مقدارشان را سرور
// هنگام «ارسال برای تصویب» می‌نویسد؛ پس از آن، تنها مسیر تصویب، امضای هر سطح است.
const (
	// FieldApprovalTotal شمار سطح‌های تصویب لازم برای این درخواست.
	FieldApprovalTotal = "approvalTotal"
	// FieldApprovalProgress شمار سطح‌هایی که تا کنون امضا شده‌اند.
	FieldApprovalProgress = "approvalProgress"
	// FieldApprovalLog امضاهای ثبت‌شده به ترتیب: «سطح | کاربر | تاریخ».
	FieldApprovalLog = "approvalLog"
	// FieldApprovalLevels فهرست سطوح تصویب لازم به ترتیب، برای نمایش در رکورد.
	FieldApprovalLevels = "approvalLevels"
	// FieldAssignmentID شناسهٔ انتصاب تشکیلات که اجرای ابلاغ ساخته است (برای ابطال).
	FieldAssignmentID = "assignmentId"
	// FieldBeforePosition/FieldBeforeUnit/FieldBeforeGrade وضع پیشین کارمند در لحظهٔ اجرا؛
	// ابطال ابلاغ با همین‌ها پرونده را برمی‌گرداند.
	FieldBeforePosition = "beforePosition"
	FieldBeforeUnit     = "beforeUnit"
	FieldBeforeGrade    = "beforeGrade"
)

// کلیدهای فیلدهای سنجش شرایط احراز شغل.
//
// چرا در رکورد تصویب‌خواهی: تصمیم انتصاب و ارتقا باید مستند باشد. «کد شغل» به شناسنامهٔ
// شغل گره می‌خورد و بقیهٔ شرط‌ها همیشه از همان شناسنامه خوانده می‌شود (نه از نسخهٔ کهنهٔ
// رکورد)؛ مشخصات متقاضی هم در رکورد می‌ماند تا سند تصویب بازتولیدشدنی باشد.
const (
	// FieldJobCode کد (یا عنوان) شغل مرجع در شناسنامهٔ شغل.
	FieldJobCode = "jobCode"
	// FieldRequireDegree مقطع تحصیلی لازم — از شناسنامه خوانده و در رکورد نمایش داده می‌شود.
	FieldRequireDegree = "requireDegree"
	// FieldRequireFields رشته‌های تحصیلی مرتبط شغل.
	FieldRequireFields = "requireFields"
	// FieldRequireMinExperience کمینهٔ سابقهٔ مرتبط شغل (سال).
	FieldRequireMinExperience = "requireMinExperience"
	// FieldCandidateDegree/FieldCandidateField/FieldCandidateExperience مشخصات احراز متقاضی.
	FieldCandidateDegree     = "candidateDegree"
	FieldCandidateField      = "candidateField"
	FieldCandidateExperience = "candidateExperience"
	// FieldQualificationVerdict نتیجهٔ سنجش که سرور هنگام ارسال برای تصویب می‌نویسد.
	FieldQualificationVerdict = "qualificationVerdict"
)

// انواع رکورد این ماژول.
const (
	KindAppointments = "appointments"
	KindPromotions   = "promotions"
	KindCareerPath   = "careerPath"
)

// Descriptor شناسهٔ ماژول کارراههٔ شغلی.
func Descriptor() recordkit.Descriptor {
	return recordkit.Descriptor{
		Module:  "career",
		TitleFa: "انتصابات، ترفیعات و کارراههٔ شغلی",
		Kinds: []recordkit.Kind{
			{
				Key: KindAppointments, TitleFa: "انتصاب", IDPrefix: "APPT", EmployeeScoped: true,
				Fields: careerFields(
					// وضعیت فعلی کارمند و پست پیشنهادی؛ فرم تصویب‌خواهی هر دو را نشان
					// می‌دهد تا تصمیم‌گیرنده «از کجا به کجا» را ببیند.
					"currentPosition|پست فعلی|text|0",
					"currentUnit|واحد فعلی|text|0",
					"position|پست پیشنهادی|text|1",
					"positionCode|کد پست پیشنهادی|text|0",
					"unit|واحد سازمانی پیشنهادی|text|1",
					// گروه شغلی از نردبان گروه‌ها (select) انتخاب می‌شود؛ رتبهٔ پله در
					// موتور قواعد مبنای سنجش جهش و سطح تصویب است.
					"currentGrade|گروه شغلی فعلی|select|1",
					"proposedGrade|گروه شغلی پیشنهادی|select|1",
					"serviceMonths|سابقهٔ گروه فعلی (ماه)|number|0",
					"evaluationScore|امتیاز ارزیابی عملکرد|number|0",
					"vacancy|ظرفیت پست پیشنهادی خالی است|select|0",
					"commitment|تعهد خدمت (ماه)|number|0",
					"effectiveDate|تاریخ اجرا|date|1",
					"decreeNo|شمارهٔ ابلاغ|text|0",
					"reason|دلیل و ملاک تصویب|text|0",
				),
				Statuses:    careerStatuses(),
				Initial:     StatusDraft,
				Transitions: careerTransitions(),
				Actions:     careerActions(),
				// تاریخ مؤثر سابقهٔ انتصاب، تاریخ اجراست (نه تاریخ ثبت رکورد).
				HistoryDate:      "effectiveDate",
				EditableStatuses: []string{StatusDraft},
				ActRule:          approvalRule,
			},
			{
				Key: KindPromotions, TitleFa: "ترفیع", IDPrefix: "PROMO", EmployeeScoped: true,
				Fields: careerFields(
					"fromPosition|پست فعلی|text|1",
					"toPosition|پست پیشنهادی|text|1",
					"positionCode|کد پست پیشنهادی|text|0",
					"unit|واحد سازمانی پیشنهادی|text|0",
					"currentGrade|گروه شغلی فعلی|select|1",
					"proposedGrade|گروه شغلی پیشنهادی|select|1",
					"serviceMonths|سابقهٔ گروه فعلی (ماه)|number|0",
					"evaluationScore|امتیاز ارزیابی عملکرد|number|0",
					"vacancy|ظرفیت پست پیشنهادی خالی است|select|0",
					"commitment|تعهد خدمت (ماه)|number|0",
					"effectiveDate|تاریخ اجرا|date|1",
					"decreeNo|شمارهٔ ابلاغ|text|0",
					"reason|دلیل و ملاک تصویب|text|0",
				),
				Statuses:         careerStatuses(),
				Initial:          StatusDraft,
				Transitions:      careerTransitions(),
				Actions:          careerActions(),
				HistoryDate:      "effectiveDate",
				EditableStatuses: []string{StatusDraft},
				ActRule:          approvalRule,
			},
			{
				Key: KindCareerPath, TitleFa: "کارراههٔ شغلی", IDPrefix: "CAREER", EmployeeScoped: true,
				Fields: recordkit.Fields(
					"currentPosition|پست فعلی|text|1",
					"targetPosition|پست هدف|text|1",
					"skillGap|شکاف مهارتی|text|0",
					"reviewDate|تاریخ بازنگری|date|0",
				),
				Statuses: recordkit.Statuses(
					"draft|پیش‌نویس", "active|جاری", "achieved|محقق‌شده", "archived|بایگانی‌شده",
				),
				Initial:     StatusDraft,
				Transitions: recordkit.Steps("draft|active,archived", "active|achieved,archived"),
				Actions: recordkit.Actions(
					"activate|فعال‌سازی مسیر|primary|active|0",
					"achieve|ثبت تحقق|primary|achieved|0",
					"archive|بایگانی|ghost|archived|1",
				),
			},
			{
				// پارامترهای انتصاب و ترفیع: همان کلیدهای پارامتریک موتور قواعد، ولی
				// قابل ویرایش از خود فرم تصویب‌خواهی. چرا رکورد و نه فایل استقرار:
				// آستانه‌ها سالانه تغییر می‌کنند و مدیر منابع انسانی باید بدون انتشار
				// نسخهٔ تازهٔ نرم‌افزار بتواند آنها را زیاد/کم کند و سابقهٔ تغییر بماند.
				Key: KindAppointmentParams, TitleFa: "پارامتر انتصاب و ترفیع", IDPrefix: "APARAM", EmployeeScoped: false,
				Fields: recordkit.Fields(
					// کلید پارامتر از فهرست دامنه انتخاب می‌شود؛ کلید آزاد بی‌صدا بی‌اثر
					// می‌ماند، پس اعتبارسنجی آن را می‌سنجد.
					"key|کلید پارامتر|select|1",
					"value|مقدار|text|1",
					"titleFa|عنوان پارامتر|text|0",
					"unitFa|واحد|text|0",
					"groupFa|گروه پارامتر|text|0",
					"effectiveFrom|تاریخ اجرا|date|0",
					"reason|دلیل تغییر|text|0",
				),
				Statuses:    recordkit.Statuses("draft|پیش‌نویس", "active|جاری", "inactive|غیرفعال"),
				Initial:     StatusDraft,
				Transitions: recordkit.Steps("draft|active", "active|inactive", "inactive|active"),
				Actions: recordkit.Actions(
					"activate|جاری‌سازی پارامتر|primary|active|0",
					"deactivate|غیرفعال‌سازی|ghost|inactive|1",
				),
				Validate:         validateAppointmentParam,
				EditableStatuses: []string{StatusDraft, StatusActiveParam, StatusInactiveParam},
			},
		},
	}
}

// کلیدهای نوع رکورد پارامتر در این ماژول و وضعیت‌های آن.
const (
	// KindAppointmentParams نوع رکورد پارامترهای انتصاب و ترفیع (آستانه‌ها و شرایط احراز).
	KindAppointmentParams = "appointmentParams"
	// StatusActiveParam وضعیت رکورد پارامتری که اثر عملیاتی دارد.
	StatusActiveParam = "active"
	// StatusInactiveParam وضعیت رکورد پارامتر کنارگذاشته (پاک نمی‌شود، اثر نمی‌گذارد).
	StatusInactiveParam = "inactive"
)

// AppointmentParamView یک پارامتر قابل تنظیم انتصاب و ترفیع.
//
// چرا فهرست در دامنه: فرم تصویب‌خواهی باید بداند چه پارامترهایی قابل تغییرند، عنوان و
// واحدشان چیست و مقدار پیش‌فرض سامانه چند است؛ اگر این فهرست در رابط ساخته شود، با هر
// پارامتر تازه در موتور قواعد، رابط عقب می‌ماند.
type AppointmentParamView struct {
	Key     string  `json:"key"`
	TitleFa string  `json:"titleFa"`
	UnitFa  string  `json:"unitFa"`
	GroupFa string  `json:"groupFa"`
	Default float64 `json:"default"`
	// Boolean یعنی مقدار پارامتر فقط ۰ (خیر) یا ۱ (بله) است.
	Boolean bool `json:"boolean,omitempty"`
}

// AppointmentParamCatalog فهرست پارامترهای قابل تنظیم انتصاب، ترفیع و شرایط احراز.
func AppointmentParamCatalog() []AppointmentParamView {
	defaults := rules.DefaultApprovalParams()
	qual := defaults.Qualification
	return []AppointmentParamView{
		{Key: "appointment.minServiceMonths", TitleFa: "حداقل سابقهٔ گروه فعلی برای ترفیع", UnitFa: "ماه", GroupFa: "شرط‌های ترفیع", Default: float64(defaults.MinServiceMonths)},
		{Key: "appointment.minEvaluationScore", TitleFa: "حداقل امتیاز ارزیابی عملکرد", UnitFa: "از ۱۰۰", GroupFa: "شرط‌های ترفیع", Default: defaults.MinEvaluationScore},
		{Key: "appointment.maxGradeStep", TitleFa: "بیشینهٔ جهش گروه در یک ترفیع", UnitFa: "پله", GroupFa: "شرط‌های ترفیع", Default: float64(defaults.MaxGradeStep)},
		{Key: "appointment.commitmentMonths", TitleFa: "حداقل تعهد خدمت گروه‌های مدیریتی", UnitFa: "ماه", GroupFa: "شرط‌های ترفیع", Default: float64(defaults.CommitmentMonths)},
		{Key: "appointment.committeeRank", TitleFa: "از این پله به بالا کمیتهٔ انتصابات تصویب می‌کند", UnitFa: "پلهٔ گروه", GroupFa: "سطوح تصویب", Default: float64(defaults.CommitteeRank)},
		{Key: "appointment.ceoRank", TitleFa: "از این پله به بالا تصویب مدیرعامل لازم است", UnitFa: "پلهٔ گروه", GroupFa: "سطوح تصویب", Default: float64(defaults.CEORank)},
		{Key: "appointment.committeeForPromotion", TitleFa: "هر ترفیع تصویب کمیتهٔ انتصابات بخواهد", UnitFa: "۰ = خیر، ۱ = بله", GroupFa: "سطوح تصویب", Default: boolValue(defaults.CommitteeForPromotion), Boolean: true},
		{Key: "qual.equivalence.enabled", TitleFa: "پذیرش شرایط معادل (سابقه جای مدرک)", UnitFa: "۰ = خیر، ۱ = بله", GroupFa: "شرایط احراز", Default: boolValue(qual.EquivalenceEnabled), Boolean: true},
		{Key: "qual.equivalence.yearsPerLevel", TitleFa: "سابقهٔ لازم برای جبران هر سطح کمبود مدرک", UnitFa: "سال", GroupFa: "شرایط احراز", Default: float64(qual.YearsPerDegreeLevel)},
		{Key: "qual.equivalence.maxYears", TitleFa: "سقف سابقه برای پذیرش شرایط معادل", UnitFa: "سال", GroupFa: "شرایط احراز", Default: float64(qual.MaxCompensatedYears)},
		{Key: "qual.equivalence.fieldMismatchYears", TitleFa: "سابقهٔ اضافه برای رشتهٔ ناهمسو", UnitFa: "سال", GroupFa: "شرایط احراز", Default: float64(qual.FieldMismatchExtraYears)},
		{Key: "qual.excess.monthsPerDegreeLevel", TitleFa: "تخفیف دورهٔ انتظار برای هر سطح مدرک مازاد", UnitFa: "ماه", GroupFa: "مازاد شرایط احراز", Default: float64(qual.ExcessCreditMonthsPerLevel)},
		{Key: "qual.excess.monthsPerExtraYear", TitleFa: "تخفیف دورهٔ انتظار برای هر سال سابقهٔ مازاد", UnitFa: "ماه", GroupFa: "مازاد شرایط احراز", Default: float64(qual.ExcessCreditMonthsPerYear)},
		{Key: "qual.excess.maxCreditMonths", TitleFa: "سقف تخفیف دورهٔ انتظار", UnitFa: "ماه", GroupFa: "مازاد شرایط احراز", Default: float64(qual.MaxExcessCreditMonths)},
	}
}

// AppointmentParamByKey یک پارامتر را از فهرست دامنه پیدا می‌کند.
func AppointmentParamByKey(key string) (AppointmentParamView, bool) {
	value := strings.TrimSpace(key)
	for _, p := range AppointmentParamCatalog() {
		if p.Key == value {
			return p, true
		}
	}
	return AppointmentParamView{}, false
}

// boolValue مقدار بولی را به عدد پارامتریک تبدیل می‌کند.
func boolValue(v bool) float64 {
	if v {
		return 1
	}
	return 0
}

// validateAppointmentParam قاعدهٔ رکورد پارامتر: کلید باید از فهرست دامنه باشد، مقدار باید
// عدد باشد و پارامتر بولی فقط صفر یا یک بپذیرد.
//
// چرا اینجا: پارامتری با کلید ناشناخته یا مقدار غیرعددی بی‌صدا بی‌اثر می‌ماند و کاربر گمان
// می‌کند تنظیم اعمال شده است؛ پس رکورد پیش از ذخیره رد می‌شود.
func validateAppointmentParam(in contracts.RecordInput) error {
	view, ok := AppointmentParamByKey(in.Fields["key"])
	if !ok {
		return &recordkit.ValidationError{Details: map[string]string{
			"fields.key": "کلید پارامتر از فهرست پارامترهای انتصاب و ترفیع نیست.",
		}}
	}
	raw := strings.TrimSpace(in.Fields["value"])
	if raw == "" {
		return &recordkit.ValidationError{Details: map[string]string{
			"fields.value": "مقدار پارامتر را وارد کنید.",
		}}
	}
	num, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return &recordkit.ValidationError{Details: map[string]string{
			"fields.value": "مقدار پارامتر باید عدد باشد.",
		}}
	}
	if num < 0 {
		return &recordkit.RuleError{Message: "مقدار پارامتر نمی‌تواند منفی باشد."}
	}
	if view.Boolean && num != 0 && num != 1 {
		return &recordkit.RuleError{Message: "مقدار این پارامتر فقط ۰ (خیر) یا ۱ (بله) است."}
	}
	if from := strings.TrimSpace(in.Fields["effectiveFrom"]); from != "" {
		if _, err := calendar.Parse(from); err != nil {
			return &recordkit.ValidationError{Details: map[string]string{
				"fields.effectiveFrom": "تاریخ اجرا باید به قالب شمسی YYYY/MM/DD باشد.",
			}}
		}
	}
	return nil
}

// approvalRule قاعدهٔ اقدام‌های انتصاب و ترفیع.
//
// چرا اینجا و نه در موتور: «تصویب‌خواهی ناقص»، «تصویب پیش از امضای همهٔ سطح‌ها» و «اجرای
// بی‌ابلاغ» قاعدهٔ کسب‌وکاری سازمان است، پس در دامنهٔ همین ماژول اعلام می‌شود و موتور رکورد
// فقط صدایش می‌زند. شرط‌ها:
//
//   - ارسال برای تصویب: تاریخ اجرا، گروه فعلی و پیشنهادی و «دلیل و ملاک» الزامی است؛
//     پستی که ظرفیت خالی ندارد تصویب‌خواهی نمی‌شود و ترفیع بدون امتیاز ارزیابی ارسال نمی‌شود.
//   - امضای سطح: تنها تا وقتی سطح امضانشده باقی است؛ امضای سطح تکراری مجاز نیست.
//   - تصویب نهایی: تنها وقتی همهٔ سطح‌های زنجیره امضا شده‌اند. این همان دروازه‌ای است که
//     نمی‌گذارد یک تصویب، زنجیرهٔ چندسطحی را ببندد.
//   - بازگشت برای اصلاح: دلیل بازگشت الزامی است (موتور خودش NoteRequired را می‌سنجد).
//   - اجرا و ابلاغ: شمارهٔ ابلاغ و تاریخ اجرا الزامی است؛ ابلاغ بی‌شماره سند سازمانی نیست.
//   - ابطال ابلاغ: تنها برای ابلاغ اجراشده و با دلیل.
//
// آستانه‌های عددی (سابقه، امتیاز، جهش گروه، سطح تصویب) در موتور قواعد است و مسیر
// «تصویب‌خواهی» همان‌ها را با پارامترهای شرکت می‌سنجد؛ اینجا فقط کامل‌بودن سند و ترتیب
// سطح‌های تصویب بررسی می‌شود.
func approvalRule(rec recordkit.Record, action string) error {
	effectiveDate := strings.TrimSpace(rec.Fields["effectiveDate"])
	switch action {
	case "submit":
		if effectiveDate == "" {
			return &recordkit.RuleError{Message: "تاریخ اجرای ابلاغ را ثبت کنید؛ تصویب‌خواهی بدون تاریخ اجرا ارسال نمی‌شود."}
		}
		if strings.TrimSpace(rec.Fields["currentGrade"]) == "" || strings.TrimSpace(rec.Fields["proposedGrade"]) == "" {
			return &recordkit.RuleError{Message: "گروه شغلی فعلی و پیشنهادی را انتخاب کنید تا سطح تصویب و سنجش جهش ممکن شود."}
		}
		if strings.TrimSpace(rec.Fields["reason"]) == "" {
			return &recordkit.RuleError{Message: "«دلیل و ملاک تصویب» را بنویسید؛ تصویب‌خواهی بدون ملاک قابل تصمیم‌گیری نیست."}
		}
		if strings.TrimSpace(rec.Fields["vacancy"]) == "no" {
			return &recordkit.RuleError{Message: "پست پیشنهادی ظرفیت خالی ندارد؛ تصدی هم‌زمان دو نفر در یک پست مجاز نیست."}
		}
		if rec.Kind == KindPromotions && parseNumber(rec.Fields["evaluationScore"]) <= 0 {
			return &recordkit.RuleError{Message: "برای ترفیع، امتیاز ارزیابی عملکرد دوره را ثبت کنید."}
		}
	case "sign":
		// در امضای نخست، `approvalTotal` تازه در همین اقدام نوشته می‌شود و مقدار ذخیره‌شده
		// هنوز خالی است؛ پس اینجا فقط «سطحی برای امضا مانده باشد» سنجیده می‌شود و ترتیب
		// سطح‌ها و نقش امضاکننده در مسیر قاعده‌سنج `appointment-sign` بررسی می‌شود.
		total := int(parseNumber(rec.Fields[FieldApprovalTotal]))
		progress := int(parseNumber(rec.Fields[FieldApprovalProgress]))
		if total > 0 && progress >= total {
			return &recordkit.RuleError{Message: "همهٔ سطح‌های تصویب امضا شده‌اند؛ اجرای ابلاغ را انجام دهید."}
		}
	case "approve":
		if effectiveDate == "" {
			return &recordkit.RuleError{Message: "تاریخ اجرای ابلاغ را ثبت کنید؛ تصویب بدون تاریخ اجرا معنا ندارد."}
		}
		total := int(parseNumber(rec.Fields[FieldApprovalTotal]))
		progress := int(parseNumber(rec.Fields[FieldApprovalProgress]))
		if total <= 0 {
			return &recordkit.RuleError{Message: "این درخواست سطح تصویب ندارد؛ ابتدا آن را برای تصویب ارسال کنید."}
		}
		// این اقدام با آخرین امضا انجام می‌شود؛ پس پیشرفت ذخیره‌شده باید یک پله کمتر از
		// شمار سطح‌ها باشد (فیلدهای همین اقدام پس از این سنجش نوشته می‌شوند).
		if progress+1 < total {
			return &recordkit.RuleError{Message: "تصویب نهایی پیش از امضای همهٔ سطح‌ها مجاز نیست؛ " +
				"سطح‌های امضانشده را به ترتیب امضا کنید."}
		}
	case "return":
		if strings.TrimSpace(rec.NoteFa) == "" && strings.TrimSpace(rec.Fields["returnReason"]) == "" {
			// یادداشت اقدام در موتور سنجیده می‌شود؛ این شرط برای فراخوانی داخلی است.
			return nil
		}
	case "issue":
		if effectiveDate == "" {
			return &recordkit.RuleError{Message: "تاریخ اجرای ابلاغ را ثبت کنید؛ ابلاغ بی‌تاریخ اجرا قابل اجرا نیست."}
		}
		if strings.TrimSpace(rec.Fields["decreeNo"]) == "" {
			return &recordkit.RuleError{Message: "برای اجرا، شمارهٔ ابلاغ را ثبت کنید (مثلاً ABL-1405-101)."}
		}
	case "revoke":
		if strings.TrimSpace(rec.Fields[FieldAssignmentID]) == "" {
			return &recordkit.RuleError{Message: "این ابلاغ تصدی ثبت‌شده‌ای در تشکیلات ندارد؛ ابطال آن از مسیر «لغو» انجام می‌شود."}
		}
	}
	return nil
}

// careerFields فیلدهای یک نوع انتصاب/ترفیع: اطلاعات پایهٔ کارمند + فیلدهای اختصاصی نوع +
// فیلدهای مشترک گردش‌کار تصویب.
//
// چرا یک تابع مشترک: انتصاب و ترفیع همان گردش‌کار را دارند و اگر فهرست فیلدها دو جا نوشته
// شود، با افزودن فیلد تازه (مثل شناسهٔ انتصاب تشکیلات) یکی از دو نوع عقب می‌ماند.
func careerFields(specific ...string) []recordkit.Field {
	specs := make([]string, 0, len(specific)+len(qualificationFieldSpecs)+len(workflowFieldSpecs))
	specs = append(specs, specific...)
	specs = append(specs, qualificationFieldSpecs...)
	specs = append(specs, workflowFieldSpecs...)
	return recordkit.WithEmployeeBase(recordkit.Fields(specs...))
}

// qualificationFieldSpecs فیلدهای سنجش شرایط احراز شغل (مشترک انتصاب و ترفیع).
//
// شرط شغل با «کد شغل» به شناسنامهٔ شغل گره می‌خورد و سرور بقیهٔ شرط‌ها را از همان
// شناسنامه می‌خواند؛ سه فیلد «require*» فقط برای نمایش و سند تصویب‌اند تا کاربر ببیند با
// چه شرطی سنجیده شده است.
var qualificationFieldSpecs = []string{
	"jobCode|شغل مرجع (کد یا عنوان شناسنامه)|text|0",
	"requireDegree|مقطع تحصیلی لازم (از شناسنامه)|text|0",
	"requireFields|رشته‌های مرتبط شغل|text|0",
	"requireMinExperience|کمینهٔ سابقهٔ شغل (سال)|number|0",
	"candidateDegree|مقطع تحصیلی متقاضی|select|0",
	"candidateField|رشتهٔ تحصیلی متقاضی|text|0",
	"candidateExperience|سابقهٔ مرتبط متقاضی (سال)|number|0",
	"qualificationVerdict|نتیجهٔ سنجش شرایط احراز|text|0",
}

// workflowFieldSpecs فیلدهای مشترک گردش‌کار تصویب و اجرا.
//
// approvalTotal/approvalProgress/approvalLog را سرور هنگام «ارسال برای تصویب» و در هر
// امضا می‌نویسد؛ چون این فیلدها فقط در «پیش‌نویس» قابل ویرایش‌اند، پس از ارسال دست‌کاری‌شدنی
// نیستند و مبنای سنجش «تصویب نهایی» قرار می‌گیرند.
var workflowFieldSpecs = []string{
	"approvalLevels|سطوح تصویب لازم|text|0",
	"approvalTotal|شمار سطوح تصویب|number|0",
	"approvalProgress|سطوح امضاشده|number|0",
	"approvalLog|امضاهای تصویب|text|0",
	"assignmentId|شناسهٔ انتصاب تشکیلات|text|0",
	"beforePosition|پست پیش از اجرا|text|0",
	"beforeUnit|واحد پیش از اجرا|text|0",
	"beforeGrade|گروه شغلی پیش از اجرا|text|0",
}

// careerStatuses وضعیت‌های گردش‌کار ابلاغ: پیش‌نویس، در انتظار تصویب، تصویب‌شده،
// ابلاغ‌شده، ابلاغ باطل‌شده و لغوشده.
func careerStatuses() []recordkit.Status {
	return recordkit.Statuses(
		"draft|پیش‌نویس", "pending|در انتظار تصویب", "approved|تصویب‌شده",
		"issued|ابلاغ‌شده و اجراشده", "revoked|ابلاغ باطل‌شده", "cancelled|لغوشده",
	)
}

// careerTransitions انتقال‌های مجاز.
//
// «pending|pending» عمدی است: امضای هر سطح وضعیت را عوض نمی‌کند و فقط پیشرفت تصویب را
// جلو می‌برد؛ تا وقتی همهٔ سطح‌ها امضا نکرده‌اند، رکورد در انتظار تصویب می‌ماند.
// «pending|draft» بازگشت برای اصلاح است و «issued|revoked» ابطال ابلاغ اجراشده.
func careerTransitions() map[string][]recordkit.Status {
	return recordkit.Steps(
		"draft|pending,cancelled",
		"pending|pending,approved,draft,cancelled",
		"approved|issued,draft,cancelled",
		"issued|revoked",
	)
}

// careerActions اقدام‌های گردش‌کار ابلاغ.
//
// چرا «امضای سطح» جدا از «تصویب نهایی»: هر سطح تصویب امضای خودش را می‌خواهد و نقش
// امضاکنندهٔ هر سطح از زنجیرهٔ تصویب می‌آید؛ اما تصویب نهایی تنها با آخرین امضا صادر
// می‌شود. اگر یک اقدام واحد بود، یک تصویب، زنجیرهٔ چهارسطحی را می‌بست.
func careerActions() []recordkit.Action {
	return recordkit.Actions(
		"submit|ارسال برای تصویب|primary|pending|0|employee,hr_spec,hr_manager,line_manager",
		"sign|امضای سطح تصویب|primary|pending|0|line_manager,hr_manager,admin",
		"approve|تصویب نهایی|primary|approved|0|hr_manager,admin",
		"return|بازگشت برای اصلاح|ghost|draft|1|line_manager,hr_manager,admin",
		"issue|اجرا و ابلاغ|primary|issued|0|hr_manager",
		"revoke|ابطال ابلاغ|danger|revoked|1|hr_manager,admin",
		"cancel|لغو|danger|cancelled|1|hr_manager,line_manager",
	)
}

// parseNumber عدد یک فیلد متنی را می‌خواند (صفر اگر نامعتبر باشد).
func parseNumber(value string) float64 {
	num, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil {
		return 0
	}
	return num
}

// ExecutionStep یک گام از «اجرا و ابلاغ» سازمانی.
//
// چرا در دامنه: ترتیب گام‌های اجرا (ابلاغ ← سابقه ← تشکیلات ← حقوق ← اطلاع و بایگانی)
// رویهٔ سازمان است، نه چیدمان صفحه؛ فرم، راهنمای کاربر و مستند استقرار باید یک ترتیب
// ببینند و اگر گامی اضافه شد، فقط همین‌جا اعلام شود.
type ExecutionStep struct {
	Order    int    `json:"order"`
	Key      string `json:"key"`
	TitleFa  string `json:"titleFa"`
	DetailFa string `json:"detailFa"`
}

// ExecutionSteps گام‌های اجرای ابلاغ انتصاب یا ترفیع.
func ExecutionSteps(kind string) []ExecutionStep {
	target := "پست پیشنهادی"
	if strings.TrimSpace(kind) == "promotion" {
		target = "پست و گروه شغلی جدید"
	}
	return []ExecutionStep{
		{Order: 1, Key: "decree", TitleFa: "صدور ابلاغ",
			DetailFa: "شمارهٔ ابلاغ، تاریخ اجرا و مقام تصویب‌کننده در رکورد ثبت می‌شود."},
		{Order: 2, Key: "history", TitleFa: "ثبت سابقهٔ کارمند",
			DetailFa: target + " از تاریخ اجرا در سابقهٔ سازمانی کارمند ثبت می‌شود."},
		{Order: 3, Key: "orgchart", TitleFa: "به‌روزرسانی تشکیلات",
			DetailFa: "پست پیشنهادی متصدی می‌گیرد و وضعیت پست پیشین بررسی و در صورت خالی‌شدن بلا تصدی می‌شود."},
		{Order: 4, Key: "payroll", TitleFa: "اعلام به حقوق و دستمزد",
			DetailFa: "گروه شغلی و مبنای پرداخت از تاریخ اجرا در دورهٔ حقوق همان ماه اعمال می‌شود."},
		{Order: 5, Key: "notify", TitleFa: "اطلاع به کارمند و بایگانی",
			DetailFa: "ابلاغ به کارمند اطلاع داده و نسخهٔ امضاشده در پرونده بایگانی می‌شود."},
	}
}

// Seed رکوردهای نمونهٔ سازمان برای نمایش اولیهٔ صفحه‌ها.
func Seed() []contracts.RecordInput {
	return []contracts.RecordInput{
		{
			Kind: KindAppointments, Title: "انتصاب مدیر تولید", EmployeeID: "EMP-009",
			Fields: map[string]string{
				"currentPosition": "کارشناس ارشد فرآیند", "currentUnit": "تولید و عملیات",
				"position": "مدیر تولید", "unit": "تولید و عملیات",
				"currentGrade": "senior_expert", "proposedGrade": "manager",
				"serviceMonths": "42", "evaluationScore": "91", "vacancy": "yes", "commitment": "36",
				"effectiveDate": "1404/07/01", "decreeNo": "ABL-1404-118",
				"reason":         "بازنشستگی مدیر پیشین و احراز شرایط مدیریت تولید در ارزیابی سالانه",
				"approvalLevels": "۴ سطح: مدیر واحد، مدیر منابع انسانی، کمیتهٔ انتصابات، مدیرعامل",
			},
			SeedStatus: StatusIssued,
			NoteFa:     "ابلاغ سازمانی در پی بازنشستگی مدیر پیشین",
		},
		{
			Kind: KindAppointments, Title: "انتصاب سرپرست فروش", EmployeeID: "EMP-005",
			Fields: map[string]string{
				"currentPosition": "کارشناس فروش", "currentUnit": "فروش و بازاریابی",
				"position": "سرپرست فروش", "unit": "فروش و بازاریابی",
				"currentGrade": "expert", "proposedGrade": "supervisor",
				"serviceMonths": "28", "evaluationScore": "83", "vacancy": "yes", "commitment": "24",
				"effectiveDate":  "1405/09/01",
				"reason":         "خالی‌بودن پست سرپرست فروش و پیشنهاد مدیر واحد با تأیید ارزیابی عملکرد",
				"approvalLevels": "۳ سطح: مدیر واحد، مدیر منابع انسانی، کمیتهٔ انتصابات",
			},
			SeedStatus: StatusPending,
			NoteFa:     "در انتظار تصویب کمیته",
		},
		{
			Kind: KindPromotions, Title: "ترفیع به کارشناس ارشد مالی", EmployeeID: "EMP-004",
			Fields: map[string]string{
				"fromPosition": "کارشناس مالی", "toPosition": "کارشناس ارشد مالی",
				"unit":         "امور مالی",
				"currentGrade": "expert", "proposedGrade": "senior_expert",
				"serviceMonths": "30", "evaluationScore": "92", "vacancy": "yes", "commitment": "0",
				"effectiveDate": "1404/07/01", "decreeNo": "ABL-1404-201",
				"reason":         "ارزیابی عملکرد دورهٔ اول سال و سه سال سابقهٔ کارشناسی مالی",
				"approvalLevels": "سه سطح: مدیر واحد، مدیر منابع انسانی، کمیتهٔ انتصابات و ترفیعات",
			},
			SeedStatus: StatusIssued,
			NoteFa:     "بر پایهٔ ارزیابی عملکرد دورهٔ اول سال",
		},
		{
			Kind: KindPromotions, Title: "ترفیع گروه شغلی تولید", EmployeeID: "EMP-007",
			Fields: map[string]string{
				"fromPosition": "سرپرست تولید", "toPosition": "کارشناس ارشد تولید",
				"unit":         "تولید و عملیات",
				"currentGrade": "supervisor", "proposedGrade": "senior_expert",
				"serviceMonths": "26", "evaluationScore": "88", "vacancy": "yes", "commitment": "0",
				"effectiveDate":  "1405/09/01",
				"reason":         "کسب امتیاز ارزیابی بالای حد مصوب و تأیید کمیتهٔ ترفیعات",
				"approvalLevels": "سه سطح: مدیر واحد، مدیر منابع انسانی، کمیتهٔ انتصابات و ترفیعات",
			},
			SeedStatus: StatusApproved,
			NoteFa:     "تصویب‌شده؛ در انتظار ثبت شمارهٔ ابلاغ",
		},
		{
			Kind: KindCareerPath, Title: "مسیر رشد به مدیریت منابع انسانی", EmployeeID: "EMP-008",
			Fields: map[string]string{
				"currentPosition": "کارشناس منابع انسانی", "targetPosition": "مدیر منابع انسانی",
				"skillGap":   "دورهٔ مدیریت منابع انسانی راهبردی — ۶۰ ساعت",
				"reviewDate": "1405/01/15",
			},
			NoteFa: "نیازمند تکمیل دوره‌های آموزشی و دو سال سابقهٔ سرپرستی",
		},
	}
}
