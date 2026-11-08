// قراردادهای «اجرای ابلاغ انتصاب و ترفیع» — دو پورت نوشتن بین‌ماژولی.
//
// چرا پورت و نه import مستقیم: اجرای یک ابلاغ سه اثر سازمانی دارد که هر کدام مالک جدا
// دارند — پروندهٔ کارمند (ماژول امور کارکنان) و تشکیلات/تصدی پست (ماژول ساختار سازمانی).
// ماژول انتصابات نمی‌تواند به انبار آن‌ها دست بزند (مرز ماژول)، پس هر مالک یک پورت
// باریک می‌دهد و ماژول انتصابات فقط همان را صدا می‌زند. قاعده‌ها (چه کسی تصدی می‌گیرد،
// چه زمانی، با چه شواهدی) در دامنهٔ همان مالک می‌ماند.
//
// دو عملیات لازم است و هر دو در همین پورت‌ها اعلام شده‌اند:
//   - اجرا (Appoint/Assign): ابلاغ تازه را روی پرونده و تشکیلات می‌نشاند.
//   - ابطال (Release): ابلاغ اجراشده را برمی‌گرداند — تصدی بسته و پرونده به وضع پیشین.
package contracts

import "context"

/* ------------------------- پروندهٔ کارمند (مالک: employee) ------------------------- */

// EmployeeAssignment تغییر سازمانی‌ای که اجرای یک ابلاغ روی پروندهٔ کارمند می‌نشاند.
//
// Position پست تازه، DepartmentName نام واحد تازه و JobGrade گروه شغلی تازه است (نام و
// کلید، نه شناسه: رکورد ابلاغ همین‌ها را نگه می‌دارد و تبدیل به شناسه کار مالک پرونده است).
// فیلدهای خالی تغییر نمی‌کنند، پس همین ساختار برای «برگرداندن به وضع پیشین» هم به کار
// می‌رود (ابطال ابلاغ).
type EmployeeAssignment struct {
	Position       string `json:"position,omitempty"`
	DepartmentName string `json:"departmentName,omitempty"`
	JobGrade       string `json:"jobGrade,omitempty"`
	DecreeNo       string `json:"decreeNo,omitempty"`
	EffectiveDate  string `json:"effectiveDate,omitempty"`
	ReasonFa       string `json:"reasonFa,omitempty"`
}

// EmployeeAssignmentResult گزارش اعمال تغییر سازمانی روی پروندهٔ کارمند.
//
// DepartmentMatched می‌گوید نام واحد ابلاغ در فهرست واحدها پیدا شد یا نه؛ اگر نشد، واحد
// کارمند دست‌نخورده می‌ماند و این پرچم false برمی‌گردد تا گزارش اجرا صادق بماند.
type EmployeeAssignmentResult struct {
	Employee Employee `json:"employee"`
	// Before وضع کارمند پیش از این تغییر است؛ ابطال ابلاغ با همین مقدارها پرونده را
	// برمی‌گرداند و لازم نیست ماژول انتصابات وضع پیشین را خودش نگه دارد.
	Before            Employee `json:"before,omitempty"`
	PositionChanged   bool     `json:"positionChanged"`
	DepartmentChanged bool     `json:"departmentChanged"`
	GradeChanged      bool     `json:"gradeChanged"`
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

// PositionRelease درخواست ابطال یک تصدی ثبت‌شده.
//
// چرا شناسهٔ انتصاب و نه کد پست: ابطال باید همان رکوردی را ببندد که اجرای ابلاغ ساخته
// است، نه هر تصدی فعال دیگری روی همان پست.
type PositionRelease struct {
	AssignmentID string
	EmployeeID   string
	DecreeNo     string
	ReasonFa     string
}

// PositionReleaseResult نتیجهٔ ابطال تصدی.
type PositionReleaseResult struct {
	Released bool   `json:"released"`
	NoteFa   string `json:"noteFa,omitempty"`
}

// AppointmentApplier پورت «تشکیلات»: ثبت تصدی پست و ابطال آن.
type AppointmentApplier interface {
	Appoint(ctx context.Context, in PositionAppointment, actor string) (PositionAppointmentResult, error)
	Release(ctx context.Context, in PositionRelease, actor string) (PositionReleaseResult, error)
}
