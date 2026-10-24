package contracts

import "context"

// Contract قرارداد کار یک کارمند در شکل انتقالی API.
//
// قرارداد سند مبنای رابطهٔ کار است؛ هر تمدید یا تبدیل یک قرارداد تازه می‌سازد و
// قرارداد پیشین را «منسوخ» می‌کند تا سابقه قابل ردیابی بماند. نوع استخدام (EmploymentType)
// شاکلهٔ کارگزینی است و تعیین می‌کند کدام حکم و کدام جدول مزد اعمال شود.
type Contract struct {
	ID               string          `json:"id"`
	EmployeeID       string          `json:"employeeId"`
	EmployeeName     string          `json:"employeeName"`
	DepartmentName   string          `json:"departmentName,omitempty"`
	Position         string          `json:"position,omitempty"`
	Kind             string          `json:"kind"`
	KindFa           string          `json:"kindFa"`
	EmploymentType   string          `json:"employmentType,omitempty"`
	EmploymentTypeFa string          `json:"employmentTypeFa,omitempty"`
	StartDate        string          `json:"startDate"`
	EndDate          string          `json:"endDate,omitempty"`
	MonthlySalary    int64           `json:"monthlySalary"`
	Status           string          `json:"status"`
	StatusFa         string          `json:"statusFa"`
	SupersedesID     string          `json:"supersedesId,omitempty"`
	RenewCount       int             `json:"renewCount"`
	ServiceDays      int             `json:"serviceDays"`
	NoteFa           string          `json:"noteFa,omitempty"`
	CreatedBy        string          `json:"createdBy,omitempty"`
	DecidedBy        string          `json:"decidedBy,omitempty"`
	Events           []ContractEvent `json:"events,omitempty"`
	Version          int             `json:"version"`
}

// ContractEvent یک رخداد در زندگی قرارداد (انعقاد، تأیید، تمدید، خاتمه).
type ContractEvent struct {
	At       string `json:"at"`
	Actor    string `json:"actor,omitempty"`
	Action   string `json:"action"`
	ActionFa string `json:"actionFa,omitempty"`
	NoteFa   string `json:"noteFa,omitempty"`
}

// ContractInput ورودی انعقاد یا ویرایش پیش‌نویس قرارداد.
type ContractInput struct {
	EmployeeID string `json:"employeeId"`
	Kind       string `json:"kind"`
	// EmploymentType نوع استخدام؛ اگر خالی بماند از نوع قرارداد استنتاج می‌شود.
	EmploymentType string `json:"employmentType,omitempty"`
	StartDate      string `json:"startDate"`
	EndDate        string `json:"endDate,omitempty"`
	MonthlySalary  int64  `json:"monthlySalary"`
	NoteFa         string `json:"noteFa,omitempty"`
}

// ContractDecision ورودی تأیید/رد یا خاتمهٔ قرارداد.
type ContractDecision struct {
	Approve bool   `json:"approve"`
	NoteFa  string `json:"noteFa,omitempty"`
}

// ContractRenewInput ورودی تمدید یا تبدیل قرارداد.
type ContractRenewInput struct {
	EndDate       string `json:"endDate,omitempty"`
	MonthlySalary int64  `json:"monthlySalary,omitempty"`
	NoteFa        string `json:"noteFa,omitempty"`
}

// ContractFilter فیلترهای فهرست قراردادها؛ Scope محدودهٔ دادهٔ نقش را حمل می‌کند.
type ContractFilter struct {
	Query           string
	EmployeeID      string
	Kind            string
	Status          string
	EmploymentType  string
	Sort            string
	Page            int
	Size            int
	ExpiringDays    int
	Today           string
	Scope           string
	ActorEmployeeID string
	ActorDepartment string
}

// ContractPage صفحهٔ قراردادها.
type ContractPage struct {
	Items []Contract `json:"items"`
	Total int        `json:"total"`
	Page  int        `json:"page"`
	Size  int        `json:"size"`
}

// ContractDocumentProvider پورت خواندن قرارداد برای ساخت اسناد چاپی (قرارداد، حکم
// کارگزینی، گواهی اشتغال). ماژول قراردادها آن را پیاده می‌کند و موتور اسناد مصرف.
type ContractDocumentProvider interface {
	// ContractByID خواندن یک قرارداد با شناسه.
	ContractByID(ctx context.Context, id string) (Contract, bool, error)
	// ActiveContractOf قرارداد جاری یک کارمند.
	ActiveContractOf(ctx context.Context, employeeID string) (Contract, bool, error)
}
