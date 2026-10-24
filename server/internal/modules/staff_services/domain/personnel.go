// Package domain دامنهٔ ماژول افراد تحت تکفل و درخواست پرواز.
//
// دو موجودیت، دو گردش‌کار کوچک: تکفل (ثبت → تأیید/رد → خروج) و درخواست پرواز
// (ثبت → تأیید مدیر واحد → صدور بلیت → تسویه). قواعد اینجا خالص‌اند و هیچ وابستگی به
// HTTP یا دیتابیس ندارند.
package domain

import (
	"strings"

	"hrms/internal/contracts"
	"hrms/internal/platform/calendar"
)

// وضعیت‌های فرد تحت تکفل.
const (
	DependentPending  = "pending"
	DependentApproved = "approved"
	DependentRejected = "rejected"
	DependentRemoved  = "removed"
)

// نسبت‌های مجاز تکفل.
const (
	RelationSpouse       = "spouse"
	RelationChild        = "child"
	RelationStudentChild = "student_child"
	RelationParent       = "parent"
)

// وضعیت‌های درخواست پرواز.
const (
	FlightPending   = "pending"
	FlightApproved  = "approved"
	FlightRejected  = "rejected"
	FlightIssued    = "issued"
	FlightSettled   = "settled"
	FlightCancelled = "cancelled"
)

// هدف‌های سفر مجاز.
const (
	PurposeMission = "mission"
	PurposeLeave   = "leave"
	PurposeFamily  = "family"
)

// اقدام‌های گردش‌کار درخواست پرواز.
const (
	FlightActionApprove = "approve"
	FlightActionReject  = "reject"
	FlightActionIssue   = "issue"
	FlightActionSettle  = "settle"
	FlightActionCancel  = "cancel"
)

// اقدام‌های ثبت‌شدهٔ تکفل در زنجیرهٔ حسابرسی.
//
// چرا کلید کامل (پیشوند + اقدام): لاگ حسابرسی این ماژول هم تکفل و هم درخواست پرواز را
// ثبت می‌کند و پیشوند، سند مرجع را جدا می‌کند. همین کلید، کلید برچسب فارسی هم است،
// پس نمایش «سابقهٔ تغییرات» هیچ عنوانی نمی‌سازد.
const (
	DependentActionCreate = "dependent-create"
	DependentActionUpdate = "dependent-update"
	DependentActionDecide = "dependent-decide"
	DependentActionRemove = "dependent-remove"
)

// DependentActionFa عنوان فارسی اقدام‌های تکفل برای نمایش در سابقهٔ تغییرات پرونده.
var DependentActionFa = map[string]string{
	DependentActionCreate: "ثبت تکفل",
	DependentActionUpdate: "ویرایش اطلاعات تکفل",
	DependentActionDecide: "تصمیم دربارهٔ تکفل",
	DependentActionRemove: "خروج از تکفل",
}

// Relations نسبت‌های مجاز تکفل.
var Relations = []string{RelationSpouse, RelationChild, RelationStudentChild, RelationParent}

// RelationFa عنوان فارسی نسبت‌ها.
var RelationFa = map[string]string{
	RelationSpouse:       "همسر",
	RelationChild:        "فرزند",
	RelationStudentChild: "فرزند دانشجو",
	RelationParent:       "والد تحت تکفل",
}

// DependentStatusFa عنوان فارسی وضعیت تکفل.
var DependentStatusFa = map[string]string{
	DependentPending:  "در انتظار بررسی",
	DependentApproved: "تکفل تأییدشده",
	DependentRejected: "رد شده",
	DependentRemoved:  "خارج‌شده از تکفل",
}

// Purposes هدف‌های مجاز سفر.
var Purposes = []string{PurposeMission, PurposeLeave, PurposeFamily}

// PurposeFa عنوان فارسی هدف سفر.
var PurposeFa = map[string]string{
	PurposeMission: "ماموریت اداری",
	PurposeLeave:   "مرخصی و سفر شخصی",
	PurposeFamily:  "همراهی خانواده",
}

// FlightStatusFa عنوان فارسی وضعیت درخواست پرواز.
var FlightStatusFa = map[string]string{
	FlightPending:   "در انتظار تأیید مدیر واحد",
	FlightApproved:  "تأییدشده — در انتظار صدور بلیت",
	FlightRejected:  "رد شده",
	FlightIssued:    "بلیت صادرشده",
	FlightSettled:   "تسویه‌شده",
	FlightCancelled: "لغو شده",
}

// flightAllowed انتقال‌های مجاز وضعیت درخواست پرواز.
var flightAllowed = map[string][]string{
	FlightPending:  {FlightApproved, FlightRejected, FlightCancelled},
	FlightApproved: {FlightIssued, FlightCancelled},
	FlightIssued:   {FlightSettled},
}

// Dependent فرد تحت تکفل در حافظهٔ سامانه.
//
// DepartmentName داخلی است و به DTO نمی‌رود؛ برای اعمال محدودهٔ دسترسی «واحد خودم»
// و برای سرصفحهٔ فرم‌های چاپی لازم است.
type Dependent struct {
	ID             string
	EmployeeID     string
	EmployeeName   string
	DepartmentName string
	FullName       string
	Relation       string
	NationalID     string
	BirthDate      string
	CoverageStart  string
	Insurance      bool
	DocumentsFa    string
	Status         string
	NoteFa         string
	DecidedBy      string
	Version        int
}

// ToContract تبدیل به شکل انتقالی قرارداد API.
func (d Dependent) ToContract(today calendar.Date) contracts.Dependent {
	months := 0
	if start, err := calendar.Parse(d.CoverageStart); err == nil {
		if days, err := calendar.DaysBetween(start, today); err == nil && days > 0 {
			months = days / 30
		}
	}
	return contracts.Dependent{
		ID:             d.ID,
		EmployeeID:     d.EmployeeID,
		EmployeeName:   d.EmployeeName,
		FullName:       d.FullName,
		Relation:       d.Relation,
		RelationFa:     RelationFa[d.Relation],
		NationalID:     d.NationalID,
		BirthDate:      d.BirthDate,
		CoverageStart:  d.CoverageStart,
		Insurance:      d.Insurance,
		InsuranceFa:    insuranceFa(d.Insurance),
		DocumentsFa:    d.DocumentsFa,
		Status:         d.Status,
		StatusFa:       DependentStatusFa[d.Status],
		NoteFa:         d.NoteFa,
		DecidedBy:      d.DecidedBy,
		CoverageMonths: months,
		Version:        d.Version,
	}
}

func insuranceFa(on bool) string {
	if on {
		return "تحت پوشش بیمهٔ درمانی"
	}
	return "بدون پوشش بیمه"
}

// FromDependentInput ساخت تکفل از ورودی قرارداد.
func FromDependentInput(id string, in contracts.DependentInput, employeeName, departmentName string) Dependent {
	return Dependent{
		ID:             id,
		EmployeeID:     strings.TrimSpace(in.EmployeeID),
		EmployeeName:   employeeName,
		DepartmentName: departmentName,
		FullName:       strings.TrimSpace(in.FullName),
		Relation:       in.Relation,
		NationalID:     calendar.NormalizeDigits(strings.TrimSpace(in.NationalID)),
		BirthDate:      strings.TrimSpace(in.BirthDate),
		CoverageStart:  strings.TrimSpace(in.CoverageStart),
		Insurance:      in.Insurance,
		DocumentsFa:    strings.TrimSpace(in.DocumentsFa),
		Status:         DependentPending,
		NoteFa:         strings.TrimSpace(in.NoteFa),
	}
}

// ApplyDependentInput اعمال ویرایش روی تکفل موجود.
func ApplyDependentInput(d Dependent, in contracts.DependentInput, employeeName, departmentName string) Dependent {
	if departmentName != "" {
		d.DepartmentName = departmentName
	}
	if strings.TrimSpace(in.FullName) != "" {
		d.FullName = strings.TrimSpace(in.FullName)
	}
	if in.Relation != "" {
		d.Relation = in.Relation
	}
	if strings.TrimSpace(in.NationalID) != "" {
		d.NationalID = calendar.NormalizeDigits(strings.TrimSpace(in.NationalID))
	}
	d.BirthDate = strings.TrimSpace(in.BirthDate)
	d.CoverageStart = strings.TrimSpace(in.CoverageStart)
	d.Insurance = in.Insurance
	d.DocumentsFa = strings.TrimSpace(in.DocumentsFa)
	d.NoteFa = strings.TrimSpace(in.NoteFa)
	d.EmployeeName = employeeName
	return d
}

// ValidateDependent اعتبارسنجی ورودی تکفل.
func ValidateDependent(in contracts.DependentInput) map[string]string {
	errs := map[string]string{}
	if strings.TrimSpace(in.EmployeeID) == "" {
		errs["employeeId"] = "انتخاب کارمند الزامی است."
	}
	if len([]rune(strings.TrimSpace(in.FullName))) < 3 {
		errs["fullName"] = "نام فرد تحت تکفل باید حداقل ۳ نویسه باشد."
	}
	if !contains(Relations, in.Relation) {
		errs["relation"] = "نسبت باید یکی از همسر، فرزند، فرزند دانشجو یا والد باشد."
	}
	if id := calendar.NormalizeDigits(strings.TrimSpace(in.NationalID)); id != "" && !validNationalID(id) {
		errs["nationalId"] = "کد ملی فرد تحت تکفل باید ۱۰ رقم با رقم کنترل معتبر باشد."
	}
	if in.BirthDate != "" {
		if _, err := calendar.Parse(in.BirthDate); err != nil {
			errs["birthDate"] = "تاریخ تولد باید به قالب شمسی YYYY/MM/DD باشد."
		}
	}
	if in.CoverageStart != "" {
		if _, err := calendar.Parse(in.CoverageStart); err != nil {
			errs["coverageStart"] = "تاریخ آغاز پوشش باید به قالب شمسی YYYY/MM/DD باشد."
		}
	}
	return errs
}

// FlightRequest درخواست پرواز در حافظهٔ سامانه.
type FlightRequest struct {
	ID             string
	EmployeeID     string
	EmployeeName   string
	DepartmentName string
	OriginFa       string
	DestinationFa  string
	DepartDate     string
	ReturnDate     string
	Purpose        string
	Passengers     int
	CompanionsFa   string
	EstimatedCost  int64
	Status         string
	NoteFa         string
	DecidedBy      string
	TicketNo       string
	Version        int
}

// ToContract تبدیل به شکل انتقالی قرارداد API.
func (f FlightRequest) ToContract() contracts.FlightRequest {
	return contracts.FlightRequest{
		ID:            f.ID,
		EmployeeID:    f.EmployeeID,
		EmployeeName:  f.EmployeeName,
		OriginFa:      f.OriginFa,
		DestinationFa: f.DestinationFa,
		DepartDate:    f.DepartDate,
		ReturnDate:    f.ReturnDate,
		Purpose:       f.Purpose,
		PurposeFa:     PurposeFa[f.Purpose],
		Passengers:    f.Passengers,
		CompanionsFa:  f.CompanionsFa,
		EstimatedCost: f.EstimatedCost,
		Status:        f.Status,
		StatusFa:      FlightStatusFa[f.Status],
		NoteFa:        f.NoteFa,
		DecidedBy:     f.DecidedBy,
		TicketNo:      f.TicketNo,
		Version:       f.Version,
	}
}

// FromFlightInput ساخت درخواست پرواز از ورودی.
func FromFlightInput(id string, in contracts.FlightRequestInput, employeeName, departmentName string) FlightRequest {
	passengers := in.Passengers
	if passengers < 1 {
		passengers = 1
	}
	return FlightRequest{
		ID:             id,
		EmployeeID:     strings.TrimSpace(in.EmployeeID),
		EmployeeName:   employeeName,
		DepartmentName: departmentName,
		OriginFa:       strings.TrimSpace(in.OriginFa),
		DestinationFa:  strings.TrimSpace(in.DestinationFa),
		DepartDate:     strings.TrimSpace(in.DepartDate),
		ReturnDate:     strings.TrimSpace(in.ReturnDate),
		Purpose:        in.Purpose,
		Passengers:     passengers,
		CompanionsFa:   strings.TrimSpace(in.CompanionsFa),
		EstimatedCost:  in.EstimatedCost,
		Status:         FlightPending,
		NoteFa:         strings.TrimSpace(in.NoteFa),
	}
}

// ApplyFlightInput اعمال ویرایش روی درخواست موجود.
func ApplyFlightInput(f FlightRequest, in contracts.FlightRequestInput, employeeName, departmentName string) FlightRequest {
	if departmentName != "" {
		f.DepartmentName = departmentName
	}
	f.OriginFa = strings.TrimSpace(in.OriginFa)
	f.DestinationFa = strings.TrimSpace(in.DestinationFa)
	f.DepartDate = strings.TrimSpace(in.DepartDate)
	f.ReturnDate = strings.TrimSpace(in.ReturnDate)
	f.Purpose = in.Purpose
	if in.Passengers > 0 {
		f.Passengers = in.Passengers
	}
	f.CompanionsFa = strings.TrimSpace(in.CompanionsFa)
	f.EstimatedCost = in.EstimatedCost
	f.NoteFa = strings.TrimSpace(in.NoteFa)
	f.EmployeeName = employeeName
	return f
}

// ValidateFlight اعتبارسنجی ورودی درخواست پرواز؛ تاریخ پرواز نباید در گذشته باشد.
func ValidateFlight(in contracts.FlightRequestInput, today calendar.Date) map[string]string {
	errs := map[string]string{}
	if strings.TrimSpace(in.EmployeeID) == "" {
		errs["employeeId"] = "انتخاب کارمند الزامی است."
	}
	if strings.TrimSpace(in.OriginFa) == "" {
		errs["originFa"] = "مبدأ پرواز الزامی است."
	}
	if strings.TrimSpace(in.DestinationFa) == "" {
		errs["destinationFa"] = "مقصد پرواز الزامی است."
	}
	if strings.EqualFold(strings.TrimSpace(in.OriginFa), strings.TrimSpace(in.DestinationFa)) &&
		strings.TrimSpace(in.OriginFa) != "" {
		errs["destinationFa"] = "مقصد باید با مبدأ متفاوت باشد."
	}
	if !contains(Purposes, in.Purpose) {
		errs["purpose"] = "هدف سفر باید ماموریت، مرخصی یا همراهی خانواده باشد."
	}
	depart, departErr := calendar.Parse(in.DepartDate)
	if departErr != nil {
		errs["departDate"] = "تاریخ پرواز باید به قالب شمسی YYYY/MM/DD باشد."
	} else if days, err := calendar.DaysBetween(today, depart); err == nil && days < 0 {
		errs["departDate"] = "تاریخ پرواز نمی‌تواند پیش از امروز باشد."
	}
	if strings.TrimSpace(in.ReturnDate) != "" {
		ret, err := calendar.Parse(in.ReturnDate)
		switch {
		case err != nil:
			errs["returnDate"] = "تاریخ بازگشت باید به قالب شمسی YYYY/MM/DD باشد."
		case departErr == nil:
			if days, err := calendar.DaysBetween(depart, ret); err != nil || days < 0 {
				errs["returnDate"] = "تاریخ بازگشت نمی‌تواند پیش از تاریخ پرواز باشد."
			}
		}
	}
	if in.Passengers < 1 {
		errs["passengers"] = "شمار مسافران باید حداقل یک نفر باشد."
	}
	if in.Passengers > 10 {
		errs["passengers"] = "برای بیش از ۱۰ مسافر، درخواست گروهی جداگانه ثبت کنید."
	}
	if in.EstimatedCost < 0 {
		errs["estimatedCost"] = "برآورد هزینه نمی‌تواند منفی باشد."
	}
	return errs
}

// CanTransitionFlight بررسی مجاز بودن انتقال وضعیت درخواست پرواز.
func CanTransitionFlight(from, to string) bool {
	for _, s := range flightAllowed[from] {
		if s == to {
			return true
		}
	}
	return false
}

// TargetOfFlightAction وضعیت مقصد هر اقدام.
func TargetOfFlightAction(action string) (string, bool) {
	switch action {
	case FlightActionApprove:
		return FlightApproved, true
	case FlightActionReject:
		return FlightRejected, true
	case FlightActionIssue:
		return FlightIssued, true
	case FlightActionSettle:
		return FlightSettled, true
	case FlightActionCancel:
		return FlightCancelled, true
	default:
		return "", false
	}
}

// validNationalID بررسی رقم کنترل کد ملی (همان قاعدهٔ سامانه).
func validNationalID(input string) bool {
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

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}
