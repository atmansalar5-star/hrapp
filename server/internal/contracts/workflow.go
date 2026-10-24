package contracts

import "context"

// WorkflowActor کنشگر گردش‌کار: کسی که می‌خواهد اقدامی روی یک رکورد انجام دهد. نقش و
// دامنهٔ سازمانی او تعیین می‌کند در هر مرحله کدام اقدام برایش مجاز است. این ساختار تنها
// جای «هویت تصمیم‌گیرنده» در ورک‌فلو است تا موتور هیچ‌جا نقشی را هاردکد نکند.
type WorkflowActor struct {
	Username     string
	Role         Role
	DepartmentID string
	EmployeeID   string
}

// WorkflowTask یک کار قابل انجام برای کاربر در «کارتابل»: رکوردی که در وضعیت جاری‌اش
// دست‌کم یک اقدام برای نقش او مجاز است. برای هر اقدام مجاز یک کار ساخته می‌شود تا کاربر
// بداند دقیقاً چه کاری از او انتظار می‌رود، نه فقط اینکه رکوردی «باز» است.
type WorkflowTask struct {
	Module         string `json:"module"`
	ModuleTitleFa  string `json:"moduleTitleFa"`
	Kind           string `json:"kind"`
	KindTitleFa    string `json:"kindTitleFa"`
	RecordID       string `json:"recordId"`
	Title          string `json:"title"`
	Status         string `json:"status"`
	StatusFa       string `json:"statusFa"`
	Action         string `json:"action"`
	ActionLabelFa  string `json:"actionLabelFa"`
	ActionStyle    string `json:"actionStyle,omitempty"`
	NoteRequired   bool   `json:"noteRequired,omitempty"`
	EmployeeID     string `json:"employeeId,omitempty"`
	EmployeeName   string `json:"employeeName,omitempty"`
	DepartmentName string `json:"departmentName,omitempty"`
	CreatedBy      string `json:"createdBy,omitempty"`
	UpdatedAt      string `json:"updatedAt,omitempty"`
}

// WorkflowSource منبع کارهای گردش‌کاری یک ماژول رکوردی.
//
// هر ماژول رکوردی هنگام ثبت خود را در پورت مشترک ثبت می‌کند تا «کارتابل سراسری» بدون
// شکستن مرز ماژول ساخته شود: هیچ ماژولی داخلی‌های ماژول دیگر را نمی‌بیند و فقط مالکِ
// داده، کارهای مجاز نقش را اعلام می‌کند.
type WorkflowSource interface {
	TasksFor(ctx context.Context, actor WorkflowActor) []WorkflowTask
}

// WorkflowTaskPage پاسخ کارتابل: کارها به‌همراه شمارش و تفکیک بر پایهٔ ماژول.
type WorkflowTaskPage struct {
	Items  []WorkflowTask `json:"items"`
	Total  int            `json:"total"`
	ByKind map[string]int `json:"byModule,omitempty"`
}
