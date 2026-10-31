// Package domain دامنهٔ ماژول کارگزینی و قراردادها.
//
// همهٔ قواعد اینجا زندگی می‌کنند و هیچ وابستگی به HTTP یا دیتابیس ندارند: چه قراردادی
// معتبر است، از چه وضعیتی به چه وضعیتی می‌شود رفت، تمدید چگونه آغاز و پایان می‌گیرد و
// تبدیل قرارداد مدت موقت به مدت دائم چه شرطی دارد. لایه‌های بالاتر فقط این قواعد را
// صدا می‌زنند؛ همین قواعد در هستهٔ Rust نیز بازپیاده می‌شود و آزمون‌های طلایی انحراف را
// می‌گیرند.
package domain

import (
	"strings"
	"time"

	"hrms/internal/contracts"
	"hrms/internal/platform/calendar"
	"hrms/internal/rules"
)

// Contract قرارداد کار در حافظهٔ سامانه.
type Contract struct {
	ID             string
	EmployeeID     string
	EmployeeName   string
	DepartmentName string
	Position       string
	Kind           string
	EmploymentType string
	StartDate      string
	EndDate        string
	MonthlySalary  int64
	Status         string
	SupersedesID   string
	RenewCount     int
	NoteFa         string
	CreatedBy      string
	DecidedBy      string
	Events         []Event
	Version        int

	// WageStructure اجزای ثابت مزد و فوق‌العاده‌های اقلیمی همین قرارداد؛ همان ساختاری که
	// حکم کارگزینی دارد تا فیش حقوق در نبود حکم از قرارداد هم همان مزد مبنا را بسازد.
	contracts.WageStructure
}

// Event رخداد زندگی قرارداد.
type Event struct {
	At     time.Time
	Actor  string
	Action string
	NoteFa string
}

// انواع قرارداد مجاز.
const (
	KindPermanent = "permanent"
	KindFixedTerm = "fixed_term"
)

// وضعیت‌های مجاز قرارداد.
const (
	StatusDraft      = "draft"
	StatusPending    = "pending_approval"
	StatusActive     = "active"
	StatusRejected   = "rejected"
	StatusExpired    = "expired"
	StatusTerminated = "terminated"
)

// اقدام‌های گردش‌کار.
const (
	ActionCreate    = "create"
	ActionUpdate    = "update"
	ActionSubmit    = "submit"
	ActionApprove   = "approve"
	ActionReject    = "reject"
	ActionRenew     = "renew"
	ActionConvert   = "convert_to_permanent"
	ActionTerminate = "terminate"
	ActionExpire    = "expire"
)

// Kinds انواع مجاز قرارداد.
var Kinds = []string{KindPermanent, KindFixedTerm}

// Statuses وضعیت‌های مجاز قرارداد به ترتیب گردش‌کار.
var Statuses = []string{StatusDraft, StatusPending, StatusActive, StatusRejected, StatusExpired, StatusTerminated}

// KindFa عنوان فارسی انواع قرارداد (کلید انگلیسی در داده می‌ماند).
var KindFa = map[string]string{
	KindPermanent: "مدت دائم",
	KindFixedTerm: "مدت موقت",
}

// StatusFa عنوان فارسی وضعیت‌ها.
var StatusFa = map[string]string{
	StatusDraft:      "پیش‌نویس",
	StatusPending:    "در انتظار تأیید",
	StatusActive:     "جاری",
	StatusRejected:   "رد شده",
	StatusExpired:    "پایان‌یافته",
	StatusTerminated: "خاتمه‌یافته",
}

// ActionFa عنوان فارسی رخدادهای گردش‌کار.
var ActionFa = map[string]string{
	ActionCreate:    "انعقاد قرارداد",
	ActionUpdate:    "ویرایش قرارداد",
	ActionSubmit:    "ارسال برای تأیید",
	ActionApprove:   "تأیید و ابلاغ",
	ActionReject:    "رد",
	ActionRenew:     "تمدید",
	ActionConvert:   "تبدیل به مدت دائم",
	ActionTerminate: "خاتمهٔ قرارداد",
	ActionExpire:    "پایان مدت قرارداد",
}

// MaxFixedTermDays سقف مدت قرارداد موقت (۵ سال). مقررات کار اجازه نمی‌دهد قرارداد
// موقت بی‌پایان تکرار شود؛ عبور از این سقف باید به تبدیل یا بازنگری بینجامد.
const MaxFixedTermDays = 5 * 365

// MinConversionServiceDays کمینهٔ سابقه برای تبدیل مدت موقت به مدت دائم (یک سال).
const MinConversionServiceDays = 365

// allowed انتقال‌های مجاز وضعیت. هر مسیر دیگری خطاست، حتی اگر درخواست دستی بیاید.
var allowed = map[string][]string{
	StatusDraft:      {StatusPending},
	StatusPending:    {StatusActive, StatusRejected},
	StatusActive:     {StatusExpired, StatusTerminated},
	StatusRejected:   {StatusPending, StatusDraft},
	StatusExpired:    {},
	StatusTerminated: {},
}

// CanTransition بررسی مجاز بودن انتقال وضعیت.
func CanTransition(from, to string) bool {
	for _, s := range allowed[from] {
		if s == to {
			return true
		}
	}
	return false
}

// ToContract تبدیل به شکل انتقالی قرارداد API.
func (c Contract) ToContract(today calendar.Date) contracts.Contract {
	return contracts.Contract{
		ID:               c.ID,
		EmployeeID:       c.EmployeeID,
		EmployeeName:     c.EmployeeName,
		DepartmentName:   c.DepartmentName,
		Position:         c.Position,
		Kind:             c.Kind,
		KindFa:           KindFa[c.Kind],
		EmploymentType:   c.EmploymentType,
		EmploymentTypeFa: contracts.EmploymentTypeTitleFa(c.EmploymentType),
		StartDate:        c.StartDate,
		EndDate:          c.EndDate,
		MonthlySalary:    c.MonthlySalary,
		Status:           c.Status,
		StatusFa:         StatusFa[c.Status],
		SupersedesID:     c.SupersedesID,
		RenewCount:       c.RenewCount,
		ServiceDays:      ServiceDays(c.StartDate, today),
		NoteFa:           c.NoteFa,
		CreatedBy:        c.CreatedBy,
		DecidedBy:        c.DecidedBy,
		Events:           eventsToContract(c.Events),
		Version:          c.Version,
		WageStructure:    c.WageStructure,
	}
}

func eventsToContract(events []Event) []contracts.ContractEvent {
	out := make([]contracts.ContractEvent, 0, len(events))
	for _, e := range events {
		out = append(out, contracts.ContractEvent{
			At:       e.At.UTC().Format(time.RFC3339),
			Actor:    e.Actor,
			Action:   e.Action,
			ActionFa: ActionFa[e.Action],
			NoteFa:   e.NoteFa,
		})
	}
	return out
}

// ServiceDays سابقهٔ خدمت (روز) از تاریخ شروع تا تاریخ مرجع؛ تاریخ نامعتبر صفر می‌دهد.
func ServiceDays(start string, today calendar.Date) int {
	from, err := calendar.Parse(start)
	if err != nil {
		return 0
	}
	days, err := calendar.DaysBetween(from, today)
	if err != nil || days < 0 {
		return 0
	}
	return days
}

// ContractDays طول قرارداد موقت (روز)؛ برای مدت دائم صفر.
func ContractDays(kind, start, end string) int {
	if kind != KindFixedTerm || end == "" {
		return 0
	}
	from, err1 := calendar.Parse(start)
	to, err2 := calendar.Parse(end)
	if err1 != nil || err2 != nil {
		return 0
	}
	days, err := calendar.DaysBetween(from, to)
	if err != nil || days < 0 {
		return 0
	}
	return days
}

// Overlaps بررسی هم‌پوشانی بازهٔ دو قرارداد. قرارداد مدت دائم انتهای باز دارد، پس
// هم‌پوشانی با آن تا وقتی شروع طرف مقابل بعد از سرآغاز دائم نباشد صادق است؛ همین
// قاعده جلوی دو قرارداد جاری هم‌زمان برای یک کارمند را می‌گیرد.
func Overlaps(a, b Contract) bool {
	aStart, err := calendar.Parse(a.StartDate)
	if err != nil {
		return false
	}
	bStart, err := calendar.Parse(b.StartDate)
	if err != nil {
		return false
	}
	aEnd, aOpen := endOf(a)
	bEnd, bOpen := endOf(b)
	return !endsBefore(aEnd, aOpen, bStart) && !endsBefore(bEnd, bOpen, aStart)
}

// endOf انتهای بازه؛ قرارداد دائم یا بدون تاریخ پایان، باز است (تا بی‌نهایت).
func endOf(c Contract) (string, bool) {
	if c.Kind == KindPermanent || strings.TrimSpace(c.EndDate) == "" {
		return "", true
	}
	if _, err := calendar.Parse(c.EndDate); err != nil {
		return "", true
	}
	return c.EndDate, false
}

// endsBefore آیا این انتها پیش از نقطهٔ شروع طرف مقابل تمام می‌شود (یعنی بدون هم‌پوشانی).
func endsBefore(endDate string, open bool, start calendar.Date) bool {
	if open {
		return false
	}
	end, err := calendar.Parse(endDate)
	if err != nil {
		return false
	}
	// DaysBetween(end, start) = start - end؛ مثبت یعنی start بعد از end است.
	days, err := calendar.DaysBetween(end, start)
	return err == nil && days > 0
}

// IsValid ملی‌بودن کد ملی (رقم کنترل) — همان قاعدهٔ سامانه برای داده‌های هویتی.
func IsValidNationalID(input string) bool {
	input = calendar.NormalizeDigits(strings.TrimSpace(input))
	if len(input) != 10 {
		return false
	}
	digits := make([]int, 10)
	for i, c := range input {
		if c < '0' || c > '9' {
			return false
		}
		digits[i] = int(c - '0')
	}
	allSame := true
	for _, d := range digits {
		if d != digits[0] {
			allSame = false
			break
		}
	}
	if allSame {
		return false
	}
	sum := 0
	for i := 0; i < 9; i++ {
		sum += digits[i] * (10 - i)
	}
	r := sum % 11
	if r < 2 {
		return digits[9] == r
	}
	return digits[9] == 11-r
}

// Validate اعتبارسنجی ورودی قرارداد؛ پیام‌ها فارسی و کلیدها نام فیلدهای فرم‌اند.
func Validate(in contracts.ContractInput) map[string]string {
	errs := map[string]string{}
	if strings.TrimSpace(in.EmployeeID) == "" {
		errs["employeeId"] = "انتخاب کارمند الزامی است."
	}
	if !contains(Kinds, in.Kind) {
		errs["kind"] = "نوع قرارداد باید «مدت دائم» یا «مدت موقت» باشد."
	}
	if in.EmploymentType != "" && !contracts.ValidEmploymentType(in.EmploymentType) {
		errs["employmentType"] = "نوع استخدام باید یکی از انواع فهرست رسمی باشد."
	}
	start, startErr := calendar.Parse(in.StartDate)
	if startErr != nil {
		errs["startDate"] = "تاریخ شروع باید به قالب شمسی YYYY/MM/DD باشد."
	}
	if in.MonthlySalary <= 0 {
		errs["monthlySalary"] = "مبلغ پایهٔ ماهانه باید بزرگ‌تر از صفر باشد."
	}

	switch in.Kind {
	case KindPermanent:
		if strings.TrimSpace(in.EndDate) != "" {
			errs["endDate"] = "قرارداد مدت دائم تاریخ پایان ندارد."
		}
	case KindFixedTerm:
		if strings.TrimSpace(in.EndDate) == "" {
			errs["endDate"] = "برای قرارداد مدت موقت، تاریخ پایان الزامی است."
		} else if end, err := calendar.Parse(in.EndDate); err != nil {
			errs["endDate"] = "تاریخ پایان باید به قالب شمسی YYYY/MM/DD باشد."
		} else if startErr == nil {
			days, err := calendar.DaysBetween(start, end)
			switch {
			case err != nil:
				errs["endDate"] = "بازهٔ قرارداد قابل محاسبه نیست."
			case days <= 0:
				errs["endDate"] = "تاریخ پایان باید بعد از تاریخ شروع باشد."
			case days > MaxFixedTermDays:
				errs["endDate"] = "مدت قرارداد موقت نمی‌تواند از ۵ سال بیشتر باشد."
			}
		}
	}
	// اجزای مزد و ضرایب اقلیمی همان قاعدهٔ حکم کارگزینی را دارند (۰٫۲۵ = ۲۵٪).
	for k, v := range ValidateWageStructure(in.WageStructure) {
		errs[k] = v
	}
	return errs
}

// ValidateWageStructure اعتبارسنجی اجزای مزد و ضرایب اقلیمی قرارداد.
//
// چرا در دامنهٔ ماژول قراردادها تکرار شده: هر ماژول قاعدهٔ سند خودش را دارد و مرز ماژول
// اجازه نمی‌دهد از دامنهٔ ماژول احکام import شود؛ متن پیام‌ها یکی است و هر دو از همان
// جدول مناطق اقلیمی (`rules`) می‌خوانند، پس واگرایی معنا رخ نمی‌دهد.
func ValidateWageStructure(w contracts.WageStructure) map[string]string {
	errs := map[string]string{}
	if region := strings.TrimSpace(w.ClimateRegion); region != "" {
		if _, ok := rules.ClimateRegionByKey(region); !ok {
			errs["climateRegion"] = "منطقهٔ اقلیمی انتخابی در فهرست مناطق اقلیمی نیست."
		}
	}
	for key, p := range map[string]float64{
		"badClimatePercent":     w.BadClimatePercent,
		"deprivationPercent":    w.DeprivationPercent,
		"regionHardshipPercent": w.RegionHardshipPercent,
		"extraTimePercent":      w.ExtraTimePercent,
	} {
		if p < 0 || p > 1 {
			errs[key] = "ضریب فوق‌العاده‌های اقلیمی باید نسبتی بین ۰ و ۱ باشد (۰٫۲۵ یعنی ۲۵٪)."
		}
	}
	for key, amount := range map[string]int64{
		"rankWage":           w.RankWage,
		"seniorityWage":      w.SeniorityWage,
		"postAllowance":      w.PostAllowance,
		"sacrificeAllowance": w.SacrificeAllowance,
		"climateAllowance":   w.ClimateAllowance,
		"salaryRepair":       w.SalaryRepair,
	} {
		if amount < 0 {
			errs[key] = "مبلغ این جزء نمی‌تواند منفی باشد."
		}
	}
	return errs
}

// ValidateRenew اعتبارسنجی تمدید: قرارداد باید جاری یا پایان‌یافته و مدت موقت باشد.
func ValidateRenew(c Contract, in contracts.ContractRenewInput) map[string]string {
	errs := map[string]string{}
	if c.Kind != KindFixedTerm {
		errs["kind"] = "فقط قرارداد مدت موقت تمدید می‌شود؛ برای مدت دائم از بازنگری حکم استفاده کنید."
	}
	if c.Status != StatusActive && c.Status != StatusExpired {
		errs["status"] = "تنها قرارداد جاری یا پایان‌یافته قابل تمدید است."
	}
	if in.MonthlySalary < 0 {
		errs["monthlySalary"] = "مبلغ پایه نمی‌تواند منفی باشد."
	}
	if strings.TrimSpace(in.EndDate) != "" {
		start, ok := nextStart(c)
		if !ok {
			errs["endDate"] = "تاریخ شروع تمدید از قرارداد قبلی قابل محاسبه نیست."
			return errs
		}
		end, err := calendar.Parse(in.EndDate)
		if err != nil {
			errs["endDate"] = "تاریخ پایان تمدید باید به قالب شمسی YYYY/MM/DD باشد."
			return errs
		}
		days, err := calendar.DaysBetween(start, end)
		if err != nil || days <= 0 {
			errs["endDate"] = "تاریخ پایان تمدید باید بعد از تاریخ شروع آن باشد."
		} else if days > MaxFixedTermDays {
			errs["endDate"] = "مدت قرارداد موقت نمی‌تواند از ۵ سال بیشتر باشد."
		}
	}
	return errs
}

// ValidateConversion اعتبارسنجی تبدیل مدت موقت به مدت دائم: فقط قرارداد جاری و با
// دست‌کم یک سال سابقهٔ خدمت.
func ValidateConversion(c Contract, today calendar.Date) map[string]string {
	errs := map[string]string{}
	if c.Kind != KindFixedTerm {
		errs["kind"] = "این قرارداد مدت موقت نیست."
	}
	if c.Status != StatusActive {
		errs["status"] = "تبدیل به مدت دائم تنها برای قرارداد جاری امکان دارد."
	}
	if days := ServiceDays(c.StartDate, today); days < MinConversionServiceDays {
		errs["serviceDays"] = "تبدیل به مدت دائم نیازمند دست‌کم یک سال سابقهٔ خدمت است."
	}
	return errs
}

// NextStart تاریخ شروع تمدید: روز پس از پایان قرارداد فعلی.
func NextStart(c Contract) (string, bool) {
	d, ok := nextStart(c)
	if !ok {
		return "", false
	}
	return d.String(), true
}

func nextStart(c Contract) (calendar.Date, bool) {
	if strings.TrimSpace(c.EndDate) == "" {
		return calendar.Date{}, false
	}
	end, err := calendar.Parse(c.EndDate)
	if err != nil {
		return calendar.Date{}, false
	}
	next, err := end.AddDays(1)
	if err != nil {
		return calendar.Date{}, false
	}
	return next, true
}

// DefaultRenewEnd تاریخ پایان پیش‌فرض تمدید: همان طول قرارداد قبلی.
func DefaultRenewEnd(c Contract) (string, bool) {
	start, ok := nextStart(c)
	if !ok {
		return "", false
	}
	days := ContractDays(c.Kind, c.StartDate, c.EndDate)
	if days <= 0 {
		days = 365
	}
	end, err := start.AddDays(days)
	if err != nil {
		return "", false
	}
	return end.String(), true
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}
