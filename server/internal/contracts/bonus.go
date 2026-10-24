package contracts

// قراردادهای ماژول پاداش.
//
// داشبورد مدیریتی پاداش به سه پرسش پاسخ می‌دهد: (۱) به هر مدیریت چقدر و در چند دفعه
// پاداش پرداخت شده، (۲) به هر نفر در هر مدیریت چقدر و چند بار پرداخت شده، (۳) ترکیب
// پاداش‌ها به تفکیک نوع، دوره و وضعیت. همهٔ اعداد از رکوردهای واقعی پرداخت ساخته
// می‌شود و سرور تنها منبع حقیقت است.

// BonusChoice یک گزینهٔ انتخابی در فرم پاداش (نوع پاداش، روش پرداخت).
//
// نام عمداً با `BonusOption` ماژول «پاداش تحصیلی» تفاوت دارد: آن یکی سطح تحصیلی است و
// این یکی گزینهٔ فرم پرداخت پاداش؛ یکی‌شدن آن دو، دو مفهوم مستقل را به هم می‌چسباند.
type BonusChoice struct {
	Key            string `json:"key"`
	TitleFa        string `json:"titleFa"`
	DefaultAmount  int64  `json:"defaultAmount,omitempty"`
	RequiresDecree bool   `json:"requiresDecree,omitempty"`
	NoteFa         string `json:"noteFa,omitempty"`
}

// BonusChoices فهرست‌های پایهٔ ماژول پاداش برای فرم و پنل تخصیص.
type BonusChoices struct {
	Types          []BonusChoice `json:"types"`
	Managements    []Management  `json:"managements"`
	PaymentMethods []BonusChoice `json:"paymentMethods"`
	TodayFa        string        `json:"todayFa"`
}

// BonusManagementRow جمع پاداش‌های یک مدیریت.
type BonusManagementRow struct {
	ManagementKey    string  `json:"managementKey"`
	ManagementFa     string  `json:"managementFa"`
	Payments         int     `json:"payments"`
	Beneficiaries    int     `json:"beneficiaries"`
	TotalAmount      float64 `json:"totalAmount"`
	AveragePerPerson float64 `json:"averagePerPerson"`
	MaxPayment       float64 `json:"maxPayment"`
	CompanyCount     int     `json:"companyCount"`
	ThirdPartyCount  int     `json:"thirdPartyCount"`
	FirstDate        string  `json:"firstDate,omitempty"`
	LastDate         string  `json:"lastDate,omitempty"`
}

// BonusEmployeeRow جمع پاداش‌های یک نفر در یک مدیریت (مبنای «مبالغ پرداختی به هر نفر»).
type BonusEmployeeRow struct {
	EmployeeID     string   `json:"employeeId"`
	EmployeeName   string   `json:"employeeName"`
	ManagementKey  string   `json:"managementKey"`
	ManagementFa   string   `json:"managementFa"`
	Payments       int      `json:"payments"`
	TotalAmount    float64  `json:"totalAmount"`
	AveragePayment float64  `json:"averagePayment"`
	TypeTitlesFa   []string `json:"typeTitlesFa,omitempty"`
	FirstDate      string   `json:"firstDate,omitempty"`
	LastDate       string   `json:"lastDate,omitempty"`
}

// BonusTypeRow جمع پاداش‌ها به تفکیک نوع پاداش.
type BonusTypeRow struct {
	TypeKey      string  `json:"typeKey"`
	TypeFa       string  `json:"typeFa"`
	Payments     int     `json:"payments"`
	Amount       float64 `json:"amount"`
	SharePercent float64 `json:"sharePercent"`
}

// BonusPeriodRow جمع پاداش‌ها به تفکیک دورهٔ پرداخت.
type BonusPeriodRow struct {
	Period        string  `json:"period"`
	Payments      int     `json:"payments"`
	Beneficiaries int     `json:"beneficiaries"`
	Amount        float64 `json:"amount"`
}

// BonusStatusRow جمع پاداش‌ها به تفکیک وضعیت گردش‌کار.
type BonusStatusRow struct {
	StatusKey string  `json:"statusKey"`
	StatusFa  string  `json:"statusFa"`
	Payments  int     `json:"payments"`
	Amount    float64 `json:"amount"`
}

// BonusDashboard خروجی کامل داشبورد مدیریتی پاداش.
type BonusDashboard struct {
	ManagementRows []BonusManagementRow `json:"managementRows"`
	EmployeeRows   []BonusEmployeeRow   `json:"employeeRows"`
	TypeRows       []BonusTypeRow       `json:"typeRows"`
	PeriodRows     []BonusPeriodRow     `json:"periodRows"`
	StatusRows     []BonusStatusRow     `json:"statusRows"`

	TotalPayments      int     `json:"totalPayments"`
	PaidPayments       int     `json:"paidPayments"`
	ApprovedPayments   int     `json:"approvedPayments"`
	DraftPayments      int     `json:"draftPayments"`
	CancelledPayments  int     `json:"cancelledPayments"`
	TotalAmount        float64 `json:"totalAmount"`
	PaidAmount         float64 `json:"paidAmount"`
	TotalBeneficiaries int     `json:"totalBeneficiaries"`
	AveragePerPayment  float64 `json:"averagePerPayment"`
	AveragePerPerson   float64 `json:"averagePerPerson"`
	MaxPayment         float64 `json:"maxPayment"`
	MinPayment         float64 `json:"minPayment"`
	ManagementsCovered int     `json:"managementsCovered"`
	ManagementsTotal   int     `json:"managementsTotal"`
	CompanyAmount      float64 `json:"companyAmount"`
	ThirdPartyAmount   float64 `json:"thirdPartyAmount"`
	NoteFa             string  `json:"noteFa"`
}

// BonusAllocationRequest ورودی فرم تخصیص پاداش.
//
// یک تخصیص، پرداخت‌های هم‌مبلغ برای چند کارمند یک مدیریت می‌سازد؛ پس «فرم تخصیص» هم
// پرداخت گروهی را ساده می‌کند و هم با یک تخصیص تک‌نفره کار می‌کند (EmployeeIDs یک عضو).
type BonusAllocationRequest struct {
	// Management مدیریت مقصد پرداخت — همان برچسبی که برش «پاداش به تفکیک مدیریت»
	// داشبورد و گزارش روی آن ساخته می‌شود. الزامی است؛ فهرست کارکنان هدف از
	// EmployeeIDs می‌آید و از این فیلد استنتاج نمی‌شود.
	Management string `json:"management,omitempty"`
	BonusType  string `json:"bonusType"`
	// Amount مبلغ هر نفر (ریال).
	Amount        int64  `json:"amount"`
	PaymentDate   string `json:"paymentDate"`
	PaymentMethod string `json:"paymentMethod,omitempty"`
	DecreeNo      string `json:"decreeNo,omitempty"`
	Reference     string `json:"reference,omitempty"`
	// EmployeeIDs کارکنان هدف؛ خالی یعنی همهٔ کارکنان فعال سازمان.
	EmployeeIDs []string `json:"employeeIds,omitempty"`
	NoteFa      string   `json:"noteFa,omitempty"`
}

// BonusAllocationRow یک ردیف تخصیص (پیش‌نمایش یا ساخته‌شده).
type BonusAllocationRow struct {
	EmployeeID     string  `json:"employeeId"`
	EmployeeName   string  `json:"employeeName"`
	DepartmentName string  `json:"departmentName,omitempty"`
	ManagementKey  string  `json:"managementKey"`
	ManagementFa   string  `json:"managementFa"`
	BonusTypeFa    string  `json:"bonusTypeFa"`
	Amount         float64 `json:"amount"`
	Sequence       int     `json:"sequence"`
	RecordID       string  `json:"recordId,omitempty"`
}

// BonusAllocationResult نتیجهٔ پیش‌نمایش یا ثبت تخصیص پاداش.
type BonusAllocationResult struct {
	Rows        []BonusAllocationRow `json:"rows"`
	Count       int                  `json:"count"`
	TotalAmount float64              `json:"totalAmount"`
	Created     int                  `json:"created"`
	NoteFa      string               `json:"noteFa"`
}
