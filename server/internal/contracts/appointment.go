// قراردادهای «اجرای ابلاغ انتصاب و ترفیع» — دو پورت نوشتن بین‌ماژولی.
//
// چرا پورت و نه import مستقیم: اجرای یک ابلاغ سه اثر سازمانی دارد که هر کدام مالک جدا
// دارند — پروندهٔ کارمند (ماژول امور کارکنان) و تشکیلات/تصدی پست (ماژول ساختار سازمانی).
// ماژول انتصابات نمی‌تواند به انبار آن‌ها دست بزند (مرز ماژول)، پس هر مالک یک پورت
// باریک می‌دهد و ماژول انتصابات فقط همان را صدا می‌زند. قاعده‌ها (چه کسی تصدی می‌گیرد،
// چه زمانی، با چه شواهدی) در دامنهٔ همان مالک می‌ماند.
//
// این دو پورت همان «گام‌های اجرا»یی هستند که `career/domain.ExecutionSteps` اعلام می‌کند؛
// پیش از این اعلام می‌شدند ولی اجرا نمی‌شدند و ابلاغ فقط وضعیت رکورد را عوض می‌کرد.
package contracts

import "context"

/* ------------------------- پروندهٔ کارمند (مالک: employee) ------------------------- */

// EmployeeAssignment تغییر سازمانی‌ای که اجرای یک ابلاغ روی پروندهٔ کارمند می‌نشاند.
//
// Position پست تازه و DepartmentName نام واحد تازه است (نام، نه شناسه: رکورد ابلاغ نام
// واحد را نگه می‌دارد و تبدیل نام به شناسه کار مالک پرونده است).
type EmployeeAssignment struct {
	Position       string `json:"position,omitempty"`
	DepartmentName string `json:"departmentName,omitempty"`
	DecreeNo       string `json:"decreeNo,omitempty"`
	EffectiveDate  string `json:"effectiveDate,omitempty"`
	ReasonFa       string `json:"reasonFa,omitempty"`
}

// EmployeeAssignmentResult گزارش اعمال تغییر سازمانی روی پروندهٔ کارمند.
//
// DepartmentMatched می‌گوید نام واحد ابلاغ در فهرست واحدها پیدا شد یا نه؛ اگر نشد، واحد
// کارمند دست‌نخورده می‌ماند و این پرچم false برمی‌گردد تا گزارش اجرا صادق بماند.
type EmployeeAssignmentResult struct {
	Employee          Employee `json:"employee"`
	PositionChanged   bool     `json:"positionChanged"`
	DepartmentChanged bool     `json:"departmentChanged"`
	DepartmentMatched bool     `json:"departmentMatched"`
	NoteFa            string   `json:"noteFa,omitempty"`
}

// EmployeeAssigner پورت «ثبت تغییر سازمانی در پروندهٔ کارمند».
type EmployeeAssigner interface {
	Assign(ctx context.Context, id string, in EmployeeAssignment, actor string) (EmployeeAssignmentResult, error)
}

/* --------------------- تشکیلات و تصدی پست (مالک: orgstructure) --------------------- */

// PositionAppointment ثبت تصدی یک کارمند در یک پست سازمانی.
//
// Evidence همان «دلیل و ملاک تصویب» ابلاغ است؛ مالک تشکیلات برای آغاز تصدی شواهد
// می‌خواهد و ابلاغ مصوب همان شواهد است.
type PositionAppointment struct {
	EmployeeID    string
	PositionCode  string
	PositionTitle string
	Unit          string
	StartDate     string
	DecreeNo      string
	Evidence      string
	Commitment    int
	NoteFa        string
}

// PositionAppointmentResult نتیجهٔ ثبت تصدی در تشکیلات.
//
// ClosedIDs انتصاب‌های پیشین همان کارمند است که با این ابلاغ «پایان‌یافته» شده‌اند؛
// PositionMatched می‌گوید کد پست ابلاغ در تشکیلات پیدا شد یا نه (اگر نه، ابلاغ در پروندهٔ
// کارمند اجرا می‌شود ولی تصدی در تشکیلات ثبت نمی‌شود و گزارش اجرا همین را می‌گوید).
type PositionAppointmentResult struct {
	AssignmentID    string   `json:"assignmentId,omitempty"`
	ClosedIDs       []string `json:"closedIds,omitempty"`
	PositionMatched bool     `json:"positionMatched"`
	NoteFa          string   `json:"noteFa,omitempty"`
}

// AppointmentApplier پورت «ثبت تصدی پست در تشکیلات».
type AppointmentApplier interface {
	Appoint(ctx context.Context, in PositionAppointment, actor string) (PositionAppointmentResult, error)
}
