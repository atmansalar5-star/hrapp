package contracts

import "context"

// Record رکورد گردش‌کاری یک ماژول اداری (انتصاب، ترفیع، پروندهٔ بایگانی، حکم انضباطی…).
//
// ماژول‌هایی مثل انتصابات، بایگانی و کمیتهٔ انضباط کار یک الگوی مشترک دارند: یک رکورد
// با چند فیلد، یک وضعیت، گردش‌کار تأیید و سابقهٔ تغییرات. به‌جای نوشتن یک برش کامل برای
// هر کدام، قواعد هر ماژول در بستهٔ domain خودش اعلام می‌شود (فیلدها، وضعیت‌ها، انتقال‌ها)
// و لایهٔ پلتفرم همان الگو را اجرا می‌کند. این‌گونه قاعده در دامنه می‌ماند و کد تکراری
// حذف می‌شود. Fields کلیدهای ماشین‌خوان (لاتین) و مقادیر نمایشی هستند.
type Record struct {
	ID             string            `json:"id"`
	Kind           string            `json:"kind"`
	KindFa         string            `json:"kindFa"`
	Title          string            `json:"title"`
	EmployeeID     string            `json:"employeeId,omitempty"`
	EmployeeName   string            `json:"employeeName,omitempty"`
	DepartmentName string            `json:"departmentName,omitempty"`
	Status         string            `json:"status"`
	StatusFa       string            `json:"statusFa"`
	Fields         map[string]string `json:"fields,omitempty"`
	NoteFa         string            `json:"noteFa,omitempty"`
	CreatedBy      string            `json:"createdBy,omitempty"`
	DecidedBy      string            `json:"decidedBy,omitempty"`
	Events         []RecordEvent     `json:"events,omitempty"`
	Version        int               `json:"version"`
}

// RecordEvent یک رخداد گردش‌کار روی رکورد.
type RecordEvent struct {
	At       string `json:"at"`
	Actor    string `json:"actor"`
	Action   string `json:"action"`
	ActionFa string `json:"actionFa"`
	NoteFa   string `json:"noteFa,omitempty"`
}

// RecordInput ورودی ایجاد یا ویرایش رکورد.
type RecordInput struct {
	Kind       string            `json:"kind"`
	Title      string            `json:"title"`
	EmployeeID string            `json:"employeeId,omitempty"`
	Fields     map[string]string `json:"fields,omitempty"`
	NoteFa     string            `json:"noteFa,omitempty"`
	// SeedStatus وضعیت دلخواه برای «دادهٔ نمونه»؛ فقط در recordkit.SeedIfEmpty به کار
	// می‌رود تا نمونهٔ نمایشی وضعیت‌های واقعی گردش‌کار را نشان دهد. از JSON خوانده
	// نمی‌شود تا کلاینت نتواند با فرستادن این کلید، وضعیت رکورد را دور بزند.
	SeedStatus string `json:"-"`
}

// RecordAction ورودی انجام یک اقدام گردش‌کار (تأیید، رد، بستن، اجرا…).
type RecordAction struct {
	Action string            `json:"action"`
	NoteFa string            `json:"noteFa,omitempty"`
	Fields map[string]string `json:"fields,omitempty"`
}

// RecordPage صفحهٔ رکوردها.
type RecordPage struct {
	Items []Record `json:"items"`
	Total int      `json:"total"`
	Page  int      `json:"page"`
	Size  int      `json:"size"`
}

// RecordFilter فیلترهای فهرست رکوردها.
type RecordFilter struct {
	// Module ماژول مالک؛ در انبار مشترک Postgres برای جداسازی دادهٔ ماژول‌ها استفاده می‌شود.
	Module          string
	Kind            string
	Status          string
	Query           string
	EmployeeID      string
	Page            int
	Size            int
	Scope           string
	ActorEmployeeID string
	ActorDepartment string
}

// RecordListPort خواندن فقط‌خواندنی رکوردهای یک ماژول دیگر از راه قرارداد.
//
// ماژول‌هایی که باید روی دادهٔ ماژول دیگری آمار بگیرند (مثل ماژول آمار روی حضور و غیاب)
// همین قرارداد را مصرف می‌کنند؛ این‌گونه مرز ماژول شکسته نمی‌شود و مالکِ داده، دسترسی
// و دامنهٔ نقش را همان‌جا اعمال می‌کند.
type RecordListPort interface {
	// List رکوردهای یک نوع را در ماژول خودش برمی‌گرداند. امضای آن عیناً با
	// `recordkit.Service.List` یکی است تا سرویس هر ماژول بدون پوستهٔ اضافه این قرارداد
	// را برآورده کند.
	List(ctx context.Context, f RecordFilter) (RecordPage, error)
	// Schema شناسهٔ نوع‌های رکورد همین ماژول (فیلدها، وضعیت‌ها و اقدام‌ها). مصرف‌کنندهٔ
	// آن لایهٔ یکپارچه‌سازی است: از همین شناسه، «ساختار داده» و قالب بارگذاری هر جدول
	// ساخته می‌شود، پس هیچ فهرستی در کد تکرار نمی‌شود.
	Schema() RecordSchema
}

// RecordField تعریف یک فیلد در شناسهٔ رکورد (برای ساخت خودکار فرم در کلاینت).
type RecordField struct {
	Key      string `json:"key"`
	LabelFa  string `json:"labelFa"`
	Type     string `json:"type"`
	Required bool   `json:"required"`
	// Source منبع خودکار مقدار (مثل employee.mobile)؛ کلاینت از همین می‌فهمد فیلد را
	// از پروندهٔ کارمند پیش‌پر و تغییرناپذیر نشان دهد. خالی یعنی ورودی دستی.
	Source string `json:"source,omitempty"`
}

// RecordStatus وضعیت رکورد به‌همراه عنوان فارسی.
type RecordStatus struct {
	Value   string `json:"value"`
	LabelFa string `json:"labelFa"`
}

// RecordActionDef اقدام ممکن روی رکورد (تأیید، رد، بستن…) برای ساخت خودکار دکمه‌ها.
//
// Roles نقش‌هایی است که مجاز به انجام این اقدام در این مرحله‌اند (تفکیک وظایف)؛ خالی
// یعنی هر نقشی که در سطح ماژول اجازهٔ اقدام دارد. Scope حداقل دامنهٔ دادهٔ لازم برای
// اقدام است (all|own_unit|own) تا مثلاً «تأیید» فقط روی رکورد واحد خودِ کاربر مجاز باشد.
type RecordActionDef struct {
	Value        string   `json:"value"`
	LabelFa      string   `json:"labelFa"`
	Style        string   `json:"style"`
	To           string   `json:"to"`
	NoteRequired bool     `json:"noteRequired"`
	Roles        []string `json:"roles,omitempty"`
	Scope        string   `json:"scope,omitempty"`
}

// RecordWorkflowStep یک مرحله از گردش‌کار تأیید: وضعیتی که در آن یک گروه نقش تصمیم
// می‌گیرد. کلاینت از همین زنجیره، نمودار مرحله‌ها و «چه کسی این‌جاست» را می‌سازد؛ پس
// هیچ مرحله‌ای در UI بازنویسی نمی‌شود. Actions اقدام‌هایی است که از این وضعیت بیرون
// می‌رود و Roles اجتماع نقش‌های مجاز همین اقدام‌هاست.
type RecordWorkflowStep struct {
	Status   string   `json:"status"`
	StatusFa string   `json:"statusFa"`
	Roles    []string `json:"roles,omitempty"`
	Actions  []string `json:"actions,omitempty"`
}

// RecordKindSchema شناسهٔ یک نوع رکورد در ماژول.
//
// Transitions نگاشت «وضعیت فعلی → مقصدهای مجاز» است تا کلاینت فقط اقدام‌های مجاز را
// نشان دهد؛ این هم از همان اعلام دامنه می‌آید و در UI بازنویسی نمی‌شود.
type RecordKindSchema struct {
	Key         string              `json:"key"`
	TitleFa     string              `json:"titleFa"`
	IDPrefix    string              `json:"idPrefix"`
	Fields      []RecordField       `json:"fields"`
	Statuses    []RecordStatus      `json:"statuses"`
	Actions     []RecordActionDef   `json:"actions"`
	Transitions map[string][]string `json:"transitions"`
	// Workflow زنجیرهٔ مرحله‌های تأیید این نوع رکورد، به ترتیب وضعیت‌ها.
	Workflow      []RecordWorkflowStep `json:"workflow,omitempty"`
	InitialStatus string               `json:"initialStatus"`
	// EditableStatuses وضعیت‌هایی که ویرایش در آن‌ها مجاز است. کلاینت از همین فهرست
	// دکمهٔ ویرایش را نشان می‌دهد تا قاعدهٔ «کجا می‌شود ویرایش کرد» در UI بازنویسی نشود.
	EditableStatuses []string `json:"editableStatuses,omitempty"`
	EmployeeScoped   bool     `json:"employeeScoped"`
}

// RecordSchema شناسهٔ کامل یک ماژول: انواع رکورد و فیلدهایشان.
//
// کلاینت همین را می‌خواند و فرم و جدول را می‌سازد؛ پس هیچ برچسب یا قاعده‌ای در UI
// تکرار نمی‌شود و تغییر فیلد در دامنه، خودکار در رابط دیده می‌شود.
type RecordSchema struct {
	Module        string             `json:"module"`
	ModuleTitleFa string             `json:"moduleTitleFa"`
	Kinds         []RecordKindSchema `json:"kinds"`
}
