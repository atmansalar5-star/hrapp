// Package domain دامنهٔ ماژول احکام حقوق و گواهی اشتغال.
//
// قواعد اینجا زندگی می‌کنند و هیچ وابستگی به HTTP یا دیتابیس ندارند: چه حکمی معتبر
// است، از چه وضعیتی به چه وضعیتی می‌رود، اصلاح حکم چگونه حکم تازه می‌سازد و گواهی
// از کدام مرحله‌ها می‌گذرد. همین قواعد در هستهٔ Rust هم بازپیاده می‌شود.
package domain

import (
	"strings"

	"hrms/internal/contracts"
	"hrms/internal/platform/calendar"
	"hrms/internal/rules"
)

// SalaryDecree حکم حقوق در حافظهٔ سامانه.
//
// WageStructure اجزای ثابت مزد و فوق‌العاده‌های اقلیمی همین حکم را نگه می‌دارد؛ فیش حقوق و
// اسناد چاپی مزد مبنا را از همین اجزا می‌سازند.
type SalaryDecree struct {
	ID             string
	EmployeeID     string
	EmployeeName   string
	DepartmentName string
	Position       string
	EmploymentType string
	JobGroup       string
	BaseSalary     int64
	EffectiveDate  string
	ReasonFa       string
	Status         string
	SupersedesID   string
	NoteFa         string
	CreatedBy      string
	DecidedBy      string
	Version        int

	contracts.WageStructure
}

// EmploymentCertificate گواهی اشتغال در حافظهٔ سامانه.
type EmploymentCertificate struct {
	ID             string
	Serial         string
	EmployeeID     string
	EmployeeName   string
	DepartmentName string
	Position       string
	Kind           string
	PurposeFa      string
	RequestedAt    string
	IssuedAt       string
	DeliveredAt    string
	Status         string
	NoteFa         string
	DecidedBy      string
	Version        int
}

// وضعیت‌های حکم حقوق.
const (
	DecreeDraft      = "draft"
	DecreeIssued     = "issued"
	DecreeNotified   = "notified"
	DecreeSuperseded = "superseded"
)

// اقدام‌های گردش‌کار حکم.
const (
	ActionCreate  = "create"
	ActionUpdate  = "update"
	ActionIssue   = "issue"
	ActionNotify  = "notify"
	ActionRevise  = "revise"
	ActionDelete  = "delete"
	ActionRequest = "request"
	ActionApprove = "approve"
	ActionReject  = "reject"
	ActionDeliver = "deliver"
	// ActionSupersede اقدام خودکاری است که هنگام ابلاغ حکم تازه روی حکم پیشین ثبت
	// می‌شود؛ در «سابقهٔ تغییرات» حکم منسوخ همین رخداد دیده می‌شود.
	ActionSupersede = "supersede"
)

// DecreeActionFa عنوان فارسی اقدام‌های ثبت‌شدهٔ حکم.
//
// چرا این‌جا: رخدادهای حکم با کلید لاتین در زنجیرهٔ حسابرسی ثبت می‌شوند و نمایش
// «سابقهٔ تغییرات» به برچسب فارسی نیاز دارد؛ برچسب مالِ دامنهٔ همین ماژول است، پس
// رابط کاربری هیچ عنوانی نمی‌سازد و کلید ناشناخته بدون برچسب (فقط با شرح رخداد)
// نمایش داده می‌شود.
var DecreeActionFa = map[string]string{
	ActionCreate:    "ثبت حکم",
	ActionUpdate:    "ویرایش پیش‌نویس",
	ActionRevise:    "اصلاح حکم",
	ActionIssue:     "صدور حکم",
	ActionNotify:    "ابلاغ حکم",
	ActionDelete:    "حذف پیش‌نویس",
	ActionSupersede: "منسوخ‌سازی با حکم تازه",
}

// DecreeStatuses وضعیت‌های مجاز حکم حقوق به ترتیب گردش‌کار.
var DecreeStatuses = []string{DecreeDraft, DecreeIssued, DecreeNotified, DecreeSuperseded}

// DecreeStatusFa عنوان فارسی وضعیت‌های حکم.
var DecreeStatusFa = map[string]string{
	DecreeDraft:      "پیش‌نویس",
	DecreeIssued:     "صادرشده — در انتظار ابلاغ",
	DecreeNotified:   "ابلاغ‌شده",
	DecreeSuperseded: "منسوخ‌شده",
}

var decreeAllowed = map[string][]string{
	DecreeDraft:      {DecreeIssued},
	DecreeIssued:     {DecreeNotified},
	DecreeNotified:   {DecreeSuperseded},
	DecreeSuperseded: {},
}

// CanDecreeTransition بررسی مجاز بودن انتقال وضعیت حکم.
func CanDecreeTransition(from, to string) bool {
	for _, s := range decreeAllowed[from] {
		if s == to {
			return true
		}
	}
	return false
}

// ToSalaryDecree تبدیل به شکل انتقالی API.
func (d SalaryDecree) ToSalaryDecree() contracts.SalaryDecree {
	return contracts.SalaryDecree{
		ID:               d.ID,
		EmployeeID:       d.EmployeeID,
		EmployeeName:     d.EmployeeName,
		DepartmentName:   d.DepartmentName,
		Position:         d.Position,
		EmploymentType:   d.EmploymentType,
		EmploymentTypeFa: contracts.EmploymentTypeTitleFa(d.EmploymentType),
		JobGroup:         d.JobGroup,
		BaseSalary:       d.BaseSalary,
		EffectiveDate:    d.EffectiveDate,
		ReasonFa:         d.ReasonFa,
		Status:           d.Status,
		StatusFa:         DecreeStatusFa[d.Status],
		SupersedesID:     d.SupersedesID,
		NoteFa:           d.NoteFa,
		CreatedBy:        d.CreatedBy,
		DecidedBy:        d.DecidedBy,
		Version:          d.Version,
		WageStructure:    d.WageStructure,
	}
}

// WithWage مزد مبنا و عنوان منطقهٔ اقلیمی را روی شکل انتقالی حکم می‌گذارد.
//
// چرا اینجا و نه در رابط: کاربر باید همان عددی را ببیند که فیش و سند چاپی حساب می‌کنند؛
// پس مزد مبنا یک‌بار با همان قاعدهٔ مشترک (بستهٔ rules) ساخته می‌شود و در پاسخ می‌آید.
func (d SalaryDecree) WithWage(companyPercent map[string]float64) contracts.SalaryDecree {
	return d.ToSalaryDecree().WithWageView(companyPercent)
}

// Wage اجزای مزد این حکم را با پیش‌فرض‌های سازمانی می‌سازد (همان تابعی که نمایش می‌دهد).
func (d SalaryDecree) Wage(companyPercent map[string]float64) rules.WageComponents {
	return d.WageStructure.Components(d.BaseSalary, companyPercent)
}

// Climate ضرایب اقلیمی این حکم؛ ضریب ثبت‌نشده از پارامتر شرکت و بعد از منطقه می‌آید.
func (d SalaryDecree) Climate(companyPercent map[string]float64) rules.ClimateAllowances {
	c := d.WageStructure.Climate(companyPercent)
	c.SourceFa = "حکم کارگزینی " + d.ID
	return c
}

// نحوهٔ صدور گواهی اشتغال.
const (
	CertSimple  = "simple"
	CertBank    = "bank"
	CertEmbassy = "embassy"
	CertOther   = "other"
)

// وضعیت‌های گواهی اشتغال.
const (
	CertRequested = "requested"
	CertApproved  = "approved"
	CertRejected  = "rejected"
	CertIssued    = "issued"
	CertDelivered = "delivered"
)

// CertKinds انواع مجاز گواهی.
var CertKinds = []string{CertSimple, CertBank, CertEmbassy, CertOther}

// CertStatuses وضعیت‌های مجاز گواهی به ترتیب گردش‌کار.
var CertStatuses = []string{CertRequested, CertApproved, CertRejected, CertIssued, CertDelivered}

// CertKindFa عنوان فارسی انواع گواهی.
var CertKindFa = map[string]string{
	CertSimple:  "گواهی سادهٔ اشتغال",
	CertBank:    "گواهی اشتغال جهت بانک",
	CertEmbassy: "گواهی اشتغال جهت سفارت",
	CertOther:   "سایر",
}

// CertStatusFa عنوان فارسی وضعیت‌های گواهی.
var CertStatusFa = map[string]string{
	CertRequested: "در انتظار تأیید مدیر واحد",
	CertApproved:  "تأییدشده — در انتظار صدور",
	CertRejected:  "رد شده",
	CertIssued:    "صادرشده",
	CertDelivered: "تحویل‌شده",
}

var certAllowed = map[string][]string{
	CertRequested: {CertApproved, CertRejected},
	CertApproved:  {CertIssued},
	CertIssued:    {CertDelivered},
	CertRejected:  {},
	CertDelivered: {},
}

// CanCertTransition بررسی مجاز بودن انتقال وضعیت گواهی.
func CanCertTransition(from, to string) bool {
	for _, s := range certAllowed[from] {
		if s == to {
			return true
		}
	}
	return false
}

// ToCertificate تبدیل به شکل انتقالی API.
func (c EmploymentCertificate) ToCertificate() contracts.EmploymentCertificate {
	return contracts.EmploymentCertificate{
		ID:             c.ID,
		Serial:         c.Serial,
		EmployeeID:     c.EmployeeID,
		EmployeeName:   c.EmployeeName,
		DepartmentName: c.DepartmentName,
		Position:       c.Position,
		Kind:           c.Kind,
		KindFa:         CertKindFa[c.Kind],
		PurposeFa:      c.PurposeFa,
		RequestedAt:    c.RequestedAt,
		IssuedAt:       c.IssuedAt,
		DeliveredAt:    c.DeliveredAt,
		Status:         c.Status,
		StatusFa:       CertStatusFa[c.Status],
		NoteFa:         c.NoteFa,
		DecidedBy:      c.DecidedBy,
		Version:        c.Version,
	}
}

// ValidateSalaryDecree اعتبارسنجی ورودی حکم حقوق؛ کلیدها نام فیلدهای فرم‌اند.
func ValidateSalaryDecree(in contracts.SalaryDecreeInput) map[string]string {
	errs := map[string]string{}
	if strings.TrimSpace(in.EmployeeID) == "" {
		errs["employeeId"] = "انتخاب کارمند الزامی است."
	}
	if strings.TrimSpace(in.JobGroup) == "" {
		errs["jobGroup"] = "گروه شغلی الزامی است."
	}
	if in.EmploymentType != "" && !contracts.ValidEmploymentType(in.EmploymentType) {
		errs["employmentType"] = "نوع استخدام باید یکی از انواع فهرست رسمی باشد."
	}
	if in.BaseSalary <= 0 {
		errs["baseSalary"] = "مبلغ پایهٔ ماهانه باید بزرگ‌تر از صفر باشد."
	}
	if _, err := calendar.Parse(in.EffectiveDate); err != nil {
		errs["effectiveDate"] = "تاریخ اجرا باید به قالب شمسی YYYY/MM/DD باشد."
	}
	if strings.TrimSpace(in.ReasonFa) == "" {
		errs["reasonFa"] = "علت صدور حکم الزامی است."
	}
	for k, v := range ValidateWageStructure(in.WageStructure) {
		errs[k] = v
	}
	return errs
}

// ValidateWageStructure اعتبارسنجی اجزای مزد و ضرایب اقلیمی سند کارگزینی (حکم/قرارداد).
//
// قاعده: ضریب نسبتی است (۰٫۲۵ = ۲۵٪) و بیرون بازهٔ صفر تا یک معنا ندارد؛ مقدار صفر
// «ثبت نشده» است و به پیش‌فرض سازمانی می‌رسد، پس خطا شمرده نمی‌شود. مبلغ ریالی هم
// منفی نمی‌تواند باشد. منطقهٔ اقلیمی باید در فهرست مناطق باشد تا ضرایبش قابل استخراج بماند.
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
		"payGap":             w.PayGap,
	} {
		if amount < 0 {
			errs[key] = "مبلغ این جزء نمی‌تواند منفی باشد."
		}
	}
	// تاریخ شروع تفاوت تطبیق اگر ثبت شده باشد باید تاریخ شمسی معتبر باشد؛ مبدأ محاسبهٔ
	// جذب و سابقهٔ حکم است، پس تاریخ نامعتبر نباید بی‌صدا بماند.
	if since := strings.TrimSpace(w.PayGapSince); since != "" {
		if _, err := calendar.Parse(since); err != nil {
			errs["payGapSince"] = "تاریخ شروع تفاوت تطبیق باید به قالب شمسی YYYY/MM/DD باشد."
		}
	}
	return errs
}

// ValidateRevise اعتبارسنجی اصلاح حکم: حکم مبنا باید ابلاغ‌شده باشد.
func ValidateRevise(current SalaryDecree, in contracts.SalaryDecreeDecision) map[string]string {
	errs := map[string]string{}
	if current.Status != DecreeNotified {
		errs["status"] = "تنها حکم ابلاغ‌شده قابل اصلاح است."
	}
	if in.BaseSalary < 0 {
		errs["baseSalary"] = "مبلغ پایه نمی‌تواند منفی باشد."
	}
	if strings.TrimSpace(in.ReasonFa) == "" {
		errs["reasonFa"] = "علت اصلاح حکم الزامی است."
	}
	for k, v := range ValidateWageStructure(in.WageStructure) {
		errs[k] = v
	}
	return errs
}

// ValidateCertificate اعتبارسنجی ورودی درخواست گواهی.
func ValidateCertificate(in contracts.CertificateInput) map[string]string {
	errs := map[string]string{}
	if strings.TrimSpace(in.EmployeeID) == "" {
		errs["employeeId"] = "انتخاب کارمند الزامی است."
	}
	if !contains(CertKinds, in.Kind) {
		errs["kind"] = "نوع گواهی نامعتبر است."
	}
	return errs
}

// ValidateGroupDecree اعتبارسنجی صدور گروهی حکم: باید دست‌کم یک معیار هدف یا افزایش
// مشخص باشد و تاریخ اجرا معتبر باشد.
func ValidateGroupDecree(in contracts.SalaryDecreeGroupInput) map[string]string {
	errs := map[string]string{}
	if _, err := calendar.Parse(in.EffectiveDate); err != nil {
		errs["effectiveDate"] = "تاریخ اجرا باید به قالب شمسی YYYY/MM/DD باشد."
	}
	if strings.TrimSpace(in.ReasonFa) == "" {
		errs["reasonFa"] = "علت صدور حکم الزامی است."
	}
	if in.EmploymentType != "" && !contracts.ValidEmploymentType(in.EmploymentType) {
		errs["employmentType"] = "نوع استخدام باید یکی از انواع فهرست رسمی باشد."
	}
	if in.IncreasePercent == 0 && in.IncreaseAmount == 0 && len(in.EmployeeIDs) == 0 {
		errs["increasePercent"] = "درصد یا مبلغ افزایش (یا انتخاب دستی کارکنان) الزامی است."
	}
	if in.IncreasePercent < 0 || in.IncreaseAmount < 0 {
		errs["increasePercent"] = "افزایش نمی‌تواند منفی باشد."
	}
	return errs
}

// GroupNewSalary مبلغ پایهٔ تازه را با درصد/مبلغ افزایش و گرد کردن محاسبه می‌کند.
// این تنها نقطهٔ محاسبهٔ افزایش گروهی است تا رابط و سرور یک عدد ببینند.
func GroupNewSalary(current int64, percent float64, amount, roundTo int64) int64 {
	base := float64(current)
	next := base + base*percent/100 + float64(amount)
	if next < 0 {
		next = 0
	}
	out := int64(next + 0.5)
	if roundTo <= 0 {
		roundTo = 10000
	}
	rem := out % roundTo
	if rem != 0 {
		out += roundTo - rem
	}
	return out
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}
