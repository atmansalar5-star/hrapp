package contracts

// Dependent فرد تحت تکفل یک کارمند (همسر، فرزند، فرزند دانشجو).
//
// تکفل پایهٔ پوشش بیمهٔ درمانی است؛ پس هر تغییر وضعیت یک رخداد با تصمیم‌گیرنده
// دارد و از «در انتظار بررسی» شروع می‌شود، نه اینکه مستقیم تأییدشده ثبت شود.
type Dependent struct {
	ID             string `json:"id"`
	EmployeeID     string `json:"employeeId"`
	EmployeeName   string `json:"employeeName"`
	FullName       string `json:"fullName"`
	Relation       string `json:"relation"`
	RelationFa     string `json:"relationFa"`
	NationalID     string `json:"nationalId,omitempty"`
	BirthDate      string `json:"birthDate,omitempty"`
	CoverageStart  string `json:"coverageStart,omitempty"`
	Insurance      bool   `json:"insurance"`
	InsuranceFa    string `json:"insuranceFa,omitempty"`
	DocumentsFa    string `json:"documentsFa,omitempty"`
	Status         string `json:"status"`
	StatusFa       string `json:"statusFa"`
	NoteFa         string `json:"noteFa,omitempty"`
	DecidedBy      string `json:"decidedBy,omitempty"`
	CoverageMonths int    `json:"coverageMonths"`
	Version        int    `json:"version"`
}

// DependentInput ورودی ثبت یا ویرایش فرد تحت تکفل.
type DependentInput struct {
	EmployeeID    string `json:"employeeId"`
	FullName      string `json:"fullName"`
	Relation      string `json:"relation"`
	NationalID    string `json:"nationalId,omitempty"`
	BirthDate     string `json:"birthDate,omitempty"`
	CoverageStart string `json:"coverageStart,omitempty"`
	Insurance     bool   `json:"insurance"`
	DocumentsFa   string `json:"documentsFa,omitempty"`
	NoteFa        string `json:"noteFa,omitempty"`
}

// DependentPage صفحهٔ افراد تحت تکفل.
type DependentPage struct {
	Items []Dependent `json:"items"`
	Total int         `json:"total"`
	Page  int         `json:"page"`
	Size  int         `json:"size"`
}

// DependentFilter فیلترهای فهرست تکفل‌شدگان.
type DependentFilter struct {
	Query      string
	EmployeeID string
	Relation   string
	Status     string
	Page       int
	Size       int
	// Today تاریخ مرجع محاسبهٔ مدت پوشش (شمسی)؛ از سرور پر می‌شود.
	Today           string
	Scope           string
	ActorEmployeeID string
	ActorDepartment string
}

// FlightRequest فرم درخواست پرواز کارمند (و در صورت نیاز همراهان تحت تکفل).
type FlightRequest struct {
	ID            string `json:"id"`
	EmployeeID    string `json:"employeeId"`
	EmployeeName  string `json:"employeeName"`
	OriginFa      string `json:"originFa"`
	DestinationFa string `json:"destinationFa"`
	DepartDate    string `json:"departDate"`
	ReturnDate    string `json:"returnDate,omitempty"`
	Purpose       string `json:"purpose"`
	PurposeFa     string `json:"purposeFa"`
	Passengers    int    `json:"passengers"`
	CompanionsFa  string `json:"companionsFa,omitempty"`
	EstimatedCost int64  `json:"estimatedCost"`
	Status        string `json:"status"`
	StatusFa      string `json:"statusFa"`
	NoteFa        string `json:"noteFa,omitempty"`
	DecidedBy     string `json:"decidedBy,omitempty"`
	TicketNo      string `json:"ticketNo,omitempty"`
	Version       int    `json:"version"`
}

// FlightRequestInput ورودی ثبت یا ویرایش فرم درخواست پرواز.
type FlightRequestInput struct {
	EmployeeID    string `json:"employeeId"`
	OriginFa      string `json:"originFa"`
	DestinationFa string `json:"destinationFa"`
	DepartDate    string `json:"departDate"`
	ReturnDate    string `json:"returnDate,omitempty"`
	Purpose       string `json:"purpose"`
	Passengers    int    `json:"passengers"`
	CompanionsFa  string `json:"companionsFa,omitempty"`
	EstimatedCost int64  `json:"estimatedCost,omitempty"`
	NoteFa        string `json:"noteFa,omitempty"`
}

// FlightDecision ورودی تأیید/رد، صدور بلیت، تسویه یا لغو درخواست پرواز.
type FlightDecision struct {
	Action   string `json:"action"`
	TicketNo string `json:"ticketNo,omitempty"`
	NoteFa   string `json:"noteFa,omitempty"`
}

// FlightRequestPage صفحهٔ درخواست‌های پرواز.
type FlightRequestPage struct {
	Items []FlightRequest `json:"items"`
	Total int             `json:"total"`
	Page  int             `json:"page"`
	Size  int             `json:"size"`
}

// FlightRequestFilter فیلترهای فهرست درخواست‌های پرواز.
type FlightRequestFilter struct {
	Query           string
	EmployeeID      string
	Status          string
	Purpose         string
	Page            int
	Size            int
	Scope           string
	ActorEmployeeID string
	ActorDepartment string
}
