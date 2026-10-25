package contracts

import "context"

// WorkTimeRow کارکرد یک کارمند در یک دورهٔ پرداخت.
//
// این قرارداد میان ماژول حضور (مالک رکوردها و قواعد کارکرد) و ماژول حقوق بسته
// می‌شود؛ هیچ ماژولی داخلی‌های ماژول دیگر را import نمی‌کند.
type WorkTimeRow struct {
	EmployeeID     string `json:"employeeId"`
	EmployeeName   string `json:"employeeName,omitempty"`
	DepartmentName string `json:"departmentName,omitempty"`
	Period         string `json:"period"`

	// WorkedHours ساعت کارکرد از رکوردهای تردد روزانه (بدون رکوردهای دارای اعتراض).
	WorkedHours float64 `json:"workedHours"`
	// WorkedDays ساعت کارکرد تبدیل‌شده به روز.
	WorkedDays float64 `json:"workedDays"`

	// OvertimeHours ساعت اضافه‌کاریِ تأییدشده (مبنای پرداخت).
	OvertimeHours float64 `json:"overtimeHours"`
	// OvertimeFactor ضریب مصوب اضافه‌کاری این کارمند در دوره.
	OvertimeFactor float64 `json:"overtimeFactor"`
	// PendingOvertimeHours ساعت اضافه‌کاری در انتظار تأیید (اثری در پرداخت ندارد).
	PendingOvertimeHours float64 `json:"pendingOvertimeHours"`

	// روزهای مرخصی از رکوردهای تأییدشده.
	PaidLeaveDays   float64 `json:"paidLeaveDays"`
	UnpaidLeaveDays float64 `json:"unpaidLeaveDays"`
	SickLeaveDays   float64 `json:"sickLeaveDays"`
	// PendingLeaveDays مرخصی در گردش (پیش‌نویس/در انتظار/تأیید مدیر واحد).
	PendingLeaveDays float64 `json:"pendingLeaveDays"`

	// MissionDays روزهای مأموریت تأییدشده.
	MissionDays float64 `json:"missionDays"`
	// AbsenceDays روزهای غیبت ثبت‌شده.
	AbsenceDays float64 `json:"absenceDays"`

	// ShiftHours ساعت کارکرد در روزهایی که نوبت مصوب دارند.
	ShiftHours float64 `json:"shiftHours"`
	// NightHours و FridayHours ساعت‌های شب‌کاری و کار در روز تعطیل/پایان هفته.
	NightHours  float64 `json:"nightHours"`
	FridayHours float64 `json:"fridayHours"`
	// LateMinutes و EarlyLeaveMinutes مجموع تأخیر و تعجیل دوره به دقیقه.
	LateMinutes       int `json:"lateMinutes"`
	EarlyLeaveMinutes int `json:"earlyLeaveMinutes"`
	// ExpectedDays روزهای موظفی از نوبت مصوب و MissingDays روزهای بی‌تردد و بی‌مرخصی.
	ExpectedDays float64 `json:"expectedDays"`
	MissingDays  float64 `json:"missingDays"`

	// PayableDays روزهای قابل پرداخت و DeductibleDays روزهای کسری.
	PayableDays    float64 `json:"payableDays"`
	DeductibleDays float64 `json:"deductibleDays"`
	// PresenceRatio نسبت حضور بین ۰ و ۱.
	PresenceRatio float64 `json:"presenceRatio"`

	// RecordsCount شمار رکوردهای مؤثرِ شمرده‌شده و PendingRecords در گردش.
	RecordsCount   int `json:"recordsCount"`
	PendingRecords int `json:"pendingRecords"`
}

// WorkTimeQuery پرس‌وجوی کارکرد یک دوره.
type WorkTimeQuery struct {
	// Period دورهٔ پرداخت به قالب YYYY/MM؛ خالی یعنی همهٔ دوره‌ها.
	Period string
	// EmployeeIDs اگر پر باشد، فقط همان کارکنان برمی‌گردند.
	EmployeeIDs []string
}

// WorkTimeProvider پورت کارکرد دوره.
//
// ماژول حضور هنگام Register خودش را اینجا ثبت می‌کند و ماژول حقوق (و هر مصرف‌کنندهٔ
// دیگر) از راه همین قرارداد می‌خواند.
type WorkTimeProvider interface {
	// WorkTime کارکرد دوره را به تفکیک کارمند برمی‌گرداند.
	WorkTime(ctx context.Context, q WorkTimeQuery) ([]WorkTimeRow, error)
	// Periods دوره‌های دارای کارکرد را برمی‌گرداند (برای انتخاب دوره در رابط).
	Periods(ctx context.Context) ([]string, error)
}
