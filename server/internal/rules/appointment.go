// Package rules — قاعدهٔ «تصویب‌خواهی و اجرای انتصاب و ترفیع».
//
// چرا اینجا و نه در رابط: تصمیم «این درخواست انتصاب/ترفیع چه شرط‌هایی دارد و چه کسانی باید
// آن را تصویب کنند» یک قاعدهٔ سازمانی است، نه یک محاسبهٔ نمایشی. اگر در رابط نوشته شود، فرم
// وب و اپلیکیشن موبایل و گزارش «درخواست‌های ناتمام» هر یک عدد و ترتیب متفاوتی می‌سازند.
//
// همهٔ آستانه‌ها پارامتریک‌اند (پارامترهای شرکت) تا هیچ عددی در کد ثابت نماند:
//
//	appointment.minServiceMonths    حداقل سابقهٔ گروه فعلی برای ترفیع (ماه)
//	appointment.minEvaluationScore  حداقل امتیاز ارزیابی عملکرد
//	appointment.maxGradeStep        بیشینهٔ جهش گروه شغلی در یک ترفیع
//	appointment.commitmentMonths    حداقل تعهد خدمت برای گروه‌های مدیریتی
//	appointment.committeeRank       از این پله به بالا کمیتهٔ انتصابات تصویب می‌کند
//	appointment.ceoRank             از این پله به بالا تصویب مدیرعامل لازم است
package rules

import (
	"math"
	"strconv"
	"strings"
)

// JobGrade یک پلهٔ نردبان گروه‌های شغلی سازمان.
//
// رتبه (Rank) مبنای مقایسه است: ترفیع یعنی رفتن به رتبهٔ بالاتر، و جهش گروه یعنی اختلاف
// رتبه‌ها. عنوان فارسی همان چیزی است که در تشکیلات و ابلاغ‌ها به‌کار می‌رود.
type JobGrade struct {
	Key     string `json:"key"`
	TitleFa string `json:"titleFa"`
	Rank    int    `json:"rank"`
}

// DefaultGradeLadder نردبان پیش‌فرض گروه‌های شغلی این استقرار.
func DefaultGradeLadder() []JobGrade {
	return []JobGrade{
		{Key: "employee", TitleFa: "کارمند", Rank: 1},
		{Key: "expert", TitleFa: "کارشناس", Rank: 2},
		{Key: "senior_expert", TitleFa: "کارشناسی ارشد", Rank: 3},
		{Key: "supervisor", TitleFa: "سرپرستی", Rank: 4},
		{Key: "manager", TitleFa: "مدیریت", Rank: 5},
		{Key: "senior_manager", TitleFa: "مدیریت ارشد", Rank: 6},
	}
}

// Grades فهرست گروه‌های شغلی (کپی، تا مصرف‌کننده اصل را تغییر ندهد).
func Grades() []JobGrade {
	ladder := DefaultGradeLadder()
	out := make([]JobGrade, len(ladder))
	copy(out, ladder)
	return out
}

// GradeTitleFa عنوان فارسی یک پله از نردبان؛ پلهٔ ناشناخته همان عدد را برمی‌گرداند.
func GradeTitleFa(rank int) string {
	for _, g := range DefaultGradeLadder() {
		if g.Rank == rank {
			return g.TitleFa
		}
	}
	if rank <= 0 {
		return "نامشخص"
	}
	return "پلهٔ " + strconv.Itoa(rank)
}

// GradeRankOf رتبهٔ یک گروه از کلید آن؛ صفر یعنی ناشناخته.
func GradeRankOf(key string) int {
	value := strings.TrimSpace(key)
	for _, g := range DefaultGradeLadder() {
		if g.Key == value {
			return g.Rank
		}
	}
	return 0
}

// AppointmentKind نوع درخواست سازمانی.
type AppointmentKind struct {
	Key     string `json:"key"`
	TitleFa string `json:"titleFa"`
	NoteFa  string `json:"noteFa"`
}

// AppointmentKinds انواع درخواست‌هایی که همین فرم پوشش می‌دهد.
func AppointmentKinds() []AppointmentKind {
	return []AppointmentKind{
		{Key: "appointment", TitleFa: "انتصاب", NoteFa: "واگذاری پست سازمانی به کارمند با همان گروه یا بالاتر."},
		{Key: "promotion", TitleFa: "ترفیع", NoteFa: "ارتقای گروه شغلی بر پایهٔ سابقه، امتیاز ارزیابی و ظرفیت تشکیلات."},
		{Key: "transfer", TitleFa: "جابه‌جایی", NoteFa: "تغییر واحد یا پست بدون ارتقای گروه شغلی."},
		{Key: "acting", TitleFa: "سرپرستی موقت", NoteFa: "تصدی موقت پست تا انتخاب متصدی رسمی."},
	}
}

// AppointmentKindTitleFa عنوان فارسی نوع درخواست.
func AppointmentKindTitleFa(key string) string {
	value := strings.TrimSpace(key)
	for _, k := range AppointmentKinds() {
		if k.Key == value {
			return k.TitleFa
		}
	}
	if value == "" {
		return "انتصاب"
	}
	return value
}

// validAppointmentKind آیا نوع درخواست شناخته‌شده است.
func validAppointmentKind(key string) bool {
	value := strings.TrimSpace(key)
	for _, k := range AppointmentKinds() {
		if k.Key == value {
			return true
		}
	}
	return false
}

// ApprovalParams آستانه‌های تصویب که از پارامترهای شرکت می‌آید.
type ApprovalParams struct {
	// MinServiceMonths حداقل سابقهٔ گروه شغلی فعلی برای ترفیع (ماه).
	MinServiceMonths int `json:"minServiceMonths"`
	// MinEvaluationScore حداقل امتیاز ارزیابی عملکرد (از ۱۰۰).
	MinEvaluationScore float64 `json:"minEvaluationScore"`
	// MaxGradeStep بیشینهٔ جهش گروه در یک ترفیع.
	MaxGradeStep int `json:"maxGradeStep"`
	// CommitmentMonths حداقل تعهد خدمت برای گروه‌های مدیریتی (ماه).
	CommitmentMonths int `json:"commitmentMonths"`
	// CommitteeRank از این پله به بالا کمیتهٔ انتصابات و ترفیعات تصویب می‌کند.
	CommitteeRank int `json:"committeeRank"`
	// CEORank از این پله به بالا تصویب مدیرعامل لازم است.
	CEORank int `json:"ceoRank"`
	// CommitteeForPromotion آیا هر ترفیع بی‌قید پله، تصویب کمیته می‌خواهد.
	CommitteeForPromotion bool `json:"committeeForPromotion"`
}

// DefaultApprovalParams مقادیر پیش‌فرض وقتی پارامتر شرکت تنظیم نشده است.
func DefaultApprovalParams() ApprovalParams {
	return ApprovalParams{
		MinServiceMonths:      24,
		MinEvaluationScore:    75,
		MaxGradeStep:          1,
		CommitmentMonths:      24,
		CommitteeRank:         4,
		CEORank:               5,
		CommitteeForPromotion: true,
	}
}

// AppointmentInput ورودی فرم تصویب‌خواهی: همان فیلدهایی که کاربر پر می‌کند.
type AppointmentInput struct {
	// Kind نوع درخواست: appointment | promotion | transfer | acting.
	Kind string `json:"kind"`
	// CurrentRank و ProposedRank رتبهٔ گروه شغلی فعلی و پیشنهادی در نردبان گروه‌ها.
	CurrentRank  int `json:"currentRank"`
	ProposedRank int `json:"proposedRank"`
	// ServiceMonths سابقهٔ کارمند در گروه شغلی فعلی (ماه).
	ServiceMonths int `json:"serviceMonths"`
	// EvaluationScore امتیاز ارزیابی عملکرد؛ EvaluationRecorded یعنی ارزیابی ثبت شده است.
	EvaluationScore    float64 `json:"evaluationScore"`
	EvaluationRecorded bool    `json:"evaluationRecorded"`
	// VacancyAvailable آیا ظرفیت پست پیشنهادی خالی است؛ VacancyChecked یعنی بررسی شده است.
	VacancyAvailable bool `json:"vacancyAvailable"`
	VacancyChecked   bool `json:"vacancyChecked"`
	// CommitmentMonths تعهد خدمتی که کارمند می‌پذیرد (ماه).
	CommitmentMonths int `json:"commitmentMonths"`
	// EffectiveDate تاریخ اجرای پیشنهادی (شمسی YYYY/MM/DD).
	EffectiveDate string `json:"effectiveDate"`
}

// AppointmentCheck یک شرط بررسی‌شده در تصویب‌خواهی.
type AppointmentCheck struct {
	Key      string `json:"key"`
	TitleFa  string `json:"titleFa"`
	Passed   bool   `json:"passed"`
	Blocking bool   `json:"blocking"`
	// MessageFa نتیجه را به فارسی توضیح می‌دهد (چرا گذشت یا چرا مانع است).
	MessageFa string `json:"messageFa"`
}

// ApprovalStep یک سطح تصویب در زنجیرهٔ تصویب‌خواهی.
type ApprovalStep struct {
	Order    int    `json:"order"`
	Key      string `json:"key"`
	TitleFa  string `json:"titleFa"`
	RoleKey  string `json:"roleKey"`
	Required bool   `json:"required"`
	ReasonFa string `json:"reasonFa,omitempty"`
}

// ApprovalPlan خروجی تصویب‌خواهی: شرط‌ها، زنجیرهٔ تصویب و جمع‌بندی.
type ApprovalPlan struct {
	Kind          string `json:"kind"`
	KindFa        string `json:"kindFa"`
	Allowed       bool   `json:"allowed"`
	GradeJump     int    `json:"gradeJump"`
	CurrentGrade  string `json:"currentGradeFa"`
	ProposedGrade string `json:"proposedGradeFa"`
	// Checks شرط‌های بررسی‌شده؛ BlockingCount شمار مانع‌ها و WarningCount شمار هشدارهاست.
	Checks        []AppointmentCheck `json:"checks"`
	Approvals     []ApprovalStep     `json:"approvals"`
	BlockingCount int                `json:"blockingCount"`
	WarningCount  int                `json:"warningCount"`
	// SummaryFa یک جملهٔ آمادهٔ نمایش: «۳ سطح تصویب، ۱ مانع».
	SummaryFa string `json:"summaryFa"`
	NoteFa    string `json:"noteFa,omitempty"`
}

// PlanApproval تصویب‌خواهی انتصاب یا ترفیع را می‌سازد: شرط‌ها را می‌سنجد، مانع‌ها را
// جدا می‌کند و زنجیرهٔ تصویب لازم را به ترتیب برمی‌گرداند.
//
// قاعده‌ها (همه در همین تابع اعلام شده‌اند):
//
//	۱. تاریخ اجرا الزامی است؛ درخواست بدون تاریخ اجرا قابل تصویب نیست.
//	۲. رتبهٔ گروه‌ها باید معتبر باشد (۱ تا بالاترین پله).
//	۳. ترفیع یعنی رفتن به پلهٔ بالاتر؛ اگر گروه تغییر نکند یا پایین بیاید ترفیع نیست.
//	۴. جهش بیش از «appointment.maxGradeStep» پله در یک ترفیع مجاز نیست.
//	۵. ترفیع نیازمند حداقل سابقهٔ «appointment.minServiceMonths» در گروه فعلی است.
//	۶. ترفیع نیازمند امتیاز ارزیابی ثبت‌شده و دست‌کم «appointment.minEvaluationScore» است.
//	۷. پست پیشنهادی باید ظرفیت خالی داشته باشد (نصب بر پست پر، تصویب نمی‌شود).
//	۸. برای پله‌های کمیته و بالاتر، تعهد خدمت دست‌کم «appointment.commitmentMonths» الزامی است.
//	۹. انتصاب به گروه پایین‌تر مجاز نیست؛ جابه‌جایی مجاز است ولی هشدار می‌گیرد.
//	۱۰. سرپرستی موقت با گروه پایین‌تر هشدار می‌گیرد (تصدی موقت است، نه ترفیع).
func PlanApproval(in AppointmentInput, params ApprovalParams) ApprovalPlan {
	kind := strings.TrimSpace(in.Kind)
	if kind == "" {
		kind = "appointment"
	}
	current := in.CurrentRank
	proposed := in.ProposedRank
	jump := proposed - current

	plan := ApprovalPlan{
		Kind:          kind,
		KindFa:        AppointmentKindTitleFa(kind),
		GradeJump:     jump,
		CurrentGrade:  GradeTitleFa(current),
		ProposedGrade: GradeTitleFa(proposed),
		Checks:        []AppointmentCheck{},
		Approvals:     []ApprovalStep{},
	}

	add := func(key, title string, passed, blocking bool, message string) {
		plan.Checks = append(plan.Checks, AppointmentCheck{
			Key: key, TitleFa: title, Passed: passed, Blocking: blocking && !passed, MessageFa: message,
		})
		if !passed {
			if blocking {
				plan.BlockingCount++
			} else {
				plan.WarningCount++
			}
		}
	}

	// ۱. نوع درخواست و تاریخ اجرا.
	add("kind", "نوع درخواست", validAppointmentKind(kind), true,
		appointmentMessage(validAppointmentKind(kind), "نوع درخواست "+plan.KindFa+" است.",
			"نوع درخواست شناخته‌شده نیست؛ یکی از انتصاب، ترفیع، جابه‌جایی یا سرپرستی موقت را انتخاب کنید."))
	hasDate := strings.TrimSpace(in.EffectiveDate) != ""
	add("effectiveDate", "تاریخ اجرا", hasDate, true,
		appointmentMessage(hasDate, "اجرا از "+strings.TrimSpace(in.EffectiveDate)+" انجام می‌شود.",
			"تاریخ اجرای ابلاغ را وارد کنید؛ درخواست بدون تاریخ اجرا تصویب نمی‌شود."))

	// ۲. اعتبار پله‌های گروه شغلی.
	gradesValid := current >= 1 && proposed >= 1 && proposed <= highestRank()
	add("grades", "گروه‌های شغلی", gradesValid, true,
		appointmentMessage(gradesValid,
			"از «"+GradeTitleFa(current)+"» به «"+GradeTitleFa(proposed)+"».",
			"گروه شغلی فعلی و پیشنهادی را از فهرست نردبان گروه‌ها انتخاب کنید."))

	// ۳ و ۴. جهت حرکت و اندازهٔ جهش، به تفکیک نوع درخواست.
	switch kind {
	case "promotion":
		up := jump >= 1
		add("direction", "ارتقای گروه شغلی", up, true,
			appointmentMessage(up, "ارتقای "+strconv.Itoa(jump)+" پله در گروه شغلی.",
				"در ترفیع، گروه شغلی پیشنهادی باید بالاتر از گروه فعلی باشد."))
		stepOK := jump <= params.MaxGradeStep && jump >= 1
		add("gradeStep", "اندازهٔ جهش گروه", stepOK, true,
			appointmentMessage(stepOK, "جهش "+strconv.Itoa(jump)+" پله، در سقف مجاز ("+
				strconv.Itoa(params.MaxGradeStep)+" پله) است.",
				"جهش بیش از "+strconv.Itoa(params.MaxGradeStep)+" پله در یک ترفیع مجاز نیست؛ "+
					"برای پله‌های بیشتر، ترفیع‌های متوالی با فاصلهٔ زمانی ثبت کنید."))
	case "appointment":
		notLower := proposed >= current
		add("direction", "گروه شغلی پیشنهادی", notLower, true,
			appointmentMessage(notLower, "انتصاب در گروه «"+GradeTitleFa(proposed)+"».",
				"انتصاب به گروه پایین‌تر مجاز نیست؛ برای کاهش گروه، نوع درخواست را «جابه‌جایی» بگذارید."))
	default:
		plan.Checks = append(plan.Checks, AppointmentCheck{
			Key: "direction", TitleFa: "گروه شغلی پیشنهادی", Passed: true,
			MessageFa: "برای " + plan.KindFa + "، تغییر گروه الزامی نیست و صرفاً ثبت می‌شود.",
		})
	}

	// ۵. سابقهٔ گروه فعلی (در ترفیع مانع، در انتصاب هشدار).
	if kind == "promotion" {
		enough := in.ServiceMonths >= params.MinServiceMonths
		add("serviceMonths", "سابقهٔ گروه فعلی", enough, true,
			appointmentMessage(enough,
				"سابقهٔ "+strconv.Itoa(in.ServiceMonths)+" ماه، شرط "+strconv.Itoa(params.MinServiceMonths)+" ماه را دارد.",
				"حداقل سابقهٔ "+strconv.Itoa(params.MinServiceMonths)+" ماه در گروه فعلی لازم است؛ سابقهٔ ثبت‌شده "+
					strconv.Itoa(in.ServiceMonths)+" ماه است."))
	} else if in.ServiceMonths > 0 && in.ServiceMonths < params.MinServiceMonths {
		add("serviceMonths", "سابقهٔ گروه فعلی", false, false,
			"سابقهٔ "+strconv.Itoa(in.ServiceMonths)+" ماه کمتر از حد معمول ترفیع است؛ برای این نوع درخواست مانع نیست.")
	}

	// ۶. امتیاز ارزیابی عملکرد.
	if kind == "promotion" {
		scored := in.EvaluationRecorded && in.EvaluationScore >= params.MinEvaluationScore
		message := "امتیاز ارزیابی " + numberFa(in.EvaluationScore) + " از حداقل " + numberFa(params.MinEvaluationScore) + " بالاتر است."
		if !in.EvaluationRecorded {
			message = "امتیاز ارزیابی عملکرد ثبت نشده است؛ مبنای ترفیع باید ارزیابی مصوب دوره باشد."
		} else if in.EvaluationScore < params.MinEvaluationScore {
			message = "امتیاز ارزیابی " + numberFa(in.EvaluationScore) + " کمتر از حداقل " + numberFa(params.MinEvaluationScore) + " است."
		}
		add("evaluation", "امتیاز ارزیابی عملکرد", scored, true, message)
	} else if in.EvaluationRecorded && in.EvaluationScore < params.MinEvaluationScore {
		add("evaluation", "امتیاز ارزیابی عملکرد", false, false,
			"امتیاز ارزیابی پایین‌تر از حد معمول است؛ برای این نوع درخواست مانع نیست.")
	}

	// ۷. ظرفیت پست پیشنهادی.
	if in.VacancyChecked {
		add("vacancy", "ظرفیت پست پیشنهادی", in.VacancyAvailable, true,
			appointmentMessage(in.VacancyAvailable, "پست پیشنهادی ظرفیت خالی دارد.",
				"پست پیشنهادی ظرفیت خالی ندارد؛ تصدی هم‌زمان دو نفر در یک پست مجاز نیست."))
	} else {
		add("vacancy", "ظرفیت پست پیشنهادی", false, false,
			"ظرفیت پست پیشنهادی بررسی نشده است؛ پیش از اجرا باید تأیید شود.")
	}

	// ۸. تعهد خدمت برای گروه‌های کمیته و بالاتر.
	managerial := proposed >= params.CommitteeRank
	if managerial {
		enough := in.CommitmentMonths >= params.CommitmentMonths
		add("commitment", "تعهد خدمت", enough, true,
			appointmentMessage(enough,
				"تعهد "+strconv.Itoa(in.CommitmentMonths)+" ماه ثبت شده است.",
				"برای گروه «"+GradeTitleFa(proposed)+"» تعهد خدمت دست‌کم "+
					strconv.Itoa(params.CommitmentMonths)+" ماه لازم است."))
	}

	// زنجیرهٔ تصویب: مدیر واحد → مدیر منابع انسانی → کمیته → مدیرعامل.
	plan.Approvals = append(plan.Approvals, ApprovalStep{
		Order: 1, Key: "lineManager", TitleFa: "مدیر واحد", RoleKey: "line_manager", Required: true,
		ReasonFa: "تأیید نیاز واحد و عملکرد کارمند.",
	})
	plan.Approvals = append(plan.Approvals, ApprovalStep{
		Order: 2, Key: "hrManager", TitleFa: "مدیر منابع انسانی", RoleKey: "hr_manager", Required: true,
		ReasonFa: "انطباق با تشکیلات، پارامترهای پرداخت و قواعد منابع انسانی.",
	})
	if (kind == "promotion" && params.CommitteeForPromotion) || proposed >= params.CommitteeRank {
		plan.Approvals = append(plan.Approvals, ApprovalStep{
			Order: 3, Key: "committee", TitleFa: "کمیتهٔ انتصابات و ترفیعات", RoleKey: "hr_manager", Required: true,
			ReasonFa: "سطح گروه «" + GradeTitleFa(proposed) + "» و نوع درخواست، تصویب کمیته را لازم می‌کند.",
		})
	}
	if proposed >= params.CEORank {
		plan.Approvals = append(plan.Approvals, ApprovalStep{
			Order: len(plan.Approvals) + 1, Key: "ceo", TitleFa: "مدیرعامل", RoleKey: "admin", Required: true,
			ReasonFa: "گروه «" + GradeTitleFa(proposed) + "» در سطح تصویب مدیرعامل است.",
		})
	}

	plan.Allowed = plan.BlockingCount == 0
	plan.SummaryFa = appointmentSummaryFa(plan)
	plan.NoteFa = "ترتیب تصویب از مدیر واحد آغاز و به بالاترین سطح لازم ختم می‌شود؛ اجرای ابلاغ تنها " +
		"پس از تصویب همهٔ سطوح و با ثبت شمارهٔ ابلاغ ممکن است."
	// متن‌هایی که کاربر می‌خواند با ارقام فارسی برمی‌گردند؛ داده و کلیدهای ماشین‌خوان
	// (رتبه، شمارهٔ پله در ساختار) همان عدد لاتین باقی می‌مانند.
	for i := range plan.Checks {
		plan.Checks[i].MessageFa = persianDigits(plan.Checks[i].MessageFa)
	}
	plan.SummaryFa = persianDigits(plan.SummaryFa)
	plan.NoteFa = persianDigits(plan.NoteFa)
	return plan
}

// persianDigits ارقام لاتین یک متن فارسی را به ارقام فارسی تبدیل می‌کند.
//
// چرا اینجا: این متن‌ها جملهٔ فارسی‌اند که مستقیم در فرم تصویب‌خواهی و رکورد دیده
// می‌شوند؛ قاعدهٔ پروژه این است که عدد در متن کاربر فارسی باشد و در داده لاتین.
func persianDigits(value string) string {
	digits := []rune("۰۱۲۳۴۵۶۷۸۹")
	runes := []rune(value)
	for i, r := range runes {
		if r >= '0' && r <= '9' {
			runes[i] = digits[r-'0']
		}
	}
	return string(runes)
}

// appointmentSummaryFa جملهٔ جمع‌بندی تصویب‌خواهی برای نمایش.
func appointmentSummaryFa(plan ApprovalPlan) string {
	levels := strconv.Itoa(len(plan.Approvals))
	summary := levels + " سطح تصویب"
	if plan.BlockingCount > 0 {
		summary += " — " + strconv.Itoa(plan.BlockingCount) + " مانع"
	} else {
		summary += " — آمادهٔ تصویب"
	}
	if plan.WarningCount > 0 {
		summary += " و " + strconv.Itoa(plan.WarningCount) + " هشدار"
	}
	return summary
}

// appointmentMessage متن نتیجهٔ یک شرط: پیام موفق یا پیام مانع.
func appointmentMessage(passed bool, ok, fail string) string {
	if passed {
		return ok
	}
	return fail
}

// highestRank بالاترین پلهٔ نردبان گروه‌ها.
func highestRank() int {
	highest := 0
	for _, g := range DefaultGradeLadder() {
		if g.Rank > highest {
			highest = g.Rank
		}
	}
	return highest
}

// EvaluationPercent سهم امتیاز ارزیابی از سقف ۱۰۰ را برمی‌گرداند (برای نمودار فرم).
func EvaluationPercent(score float64) float64 {
	if score <= 0 {
		return 0
	}
	if score > 100 {
		score = 100
	}
	return math.Round(score*10) / 10
}
