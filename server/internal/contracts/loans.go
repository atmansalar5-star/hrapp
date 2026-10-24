package contracts

// LoanOption یک گزینهٔ انتخابی در پنل وام (نوع وام یا بانک عامل) است.
type LoanOption struct {
	Key             string  `json:"key"`
	TitleFa         string  `json:"titleFa"`
	IsSupplement    bool    `json:"isSupplement,omitempty"`
	BaseKey         string  `json:"baseKey,omitempty"`
	FeeRatePercent  float64 `json:"feeRatePercent,omitempty"`
	MaxInstallments int     `json:"maxInstallments,omitempty"`
	MaxAmount       int64   `json:"maxAmount,omitempty"`
	PurposeFa       string  `json:"purposeFa,omitempty"`
	// SharePercent و Eligible فقط برای مدیریت‌های جدول سهمیه پر می‌شوند: درصد سهمیهٔ
	// مصوب مدیریت از کل صندوق تسهیلات و سرشماری مشمولین همان مدیریت.
	SharePercent float64 `json:"sharePercent,omitempty"`
	Eligible     int     `json:"eligible,omitempty"`
}

// LoanCriterionOption یک گزینهٔ معیار امتیازبندی با امتیاز پایهٔ آن.
type LoanCriterionOption struct {
	Value   string `json:"value"`
	TitleFa string `json:"titleFa"`
	Score   int    `json:"score"`
}

// LoanCriterion یک معیار امتیازبندی تخصیص وام با ضریب اهمیت و گزینه‌های امتیازدهی.
type LoanCriterion struct {
	Key     string                `json:"key"`
	TitleFa string                `json:"titleFa"`
	Weight  int                   `json:"weight"`
	Options []LoanCriterionOption `json:"options"`
}

// LoanOptions فهرست‌های پایهٔ ماژول وام: انواع وام، بانک‌های عامل و معیارهای امتیازبندی.
// رابط وب فقط نمایش می‌دهد و هیچ نرخ یا معیاری از خودش نمی‌سازد.
type LoanOptions struct {
	LoanTypes []LoanOption `json:"loanTypes"`
	Banks     []LoanOption `json:"banks"`
	// Units واحدهای سازمانی از کاتالوگ مرجع تنظیمات (هر واحد زیر یک مدیریت).
	Units []CatalogItem `json:"units"`
	// ManagementUnits مدیریت‌های بندر امام در جدول سهمیهٔ تسهیلات؛ مبنای فیلد
	// «مدیریت متقاضی» در فرم وام و نمودارهای تخصیص سهمیه است.
	ManagementUnits []LoanOption    `json:"managementUnits"`
	Criteria        []LoanCriterion `json:"criteria"`
	TodayFa         string          `json:"todayFa"`
}

// LoanScoringRequest درخواست پیش‌نمایش امتیازبندی تخصیص وام. اگر EmployeeIDs خالی باشد
// همهٔ متقاضیان نمونه امتیازبندی می‌شوند.
type LoanScoringRequest struct {
	EmployeeIDs []string `json:"employeeIds,omitempty"`
}

// LoanScoreLine ریز امتیاز یک معیار در نتیجهٔ امتیازبندی.
type LoanScoreLine struct {
	Criterion   string `json:"criterion"`
	CriterionFa string `json:"criterionFa"`
	ValueFa     string `json:"valueFa"`
	Score       int    `json:"score"`
	Weight      int    `json:"weight"`
	Points      int    `json:"points"`
}

// LoanScoreRow نتیجهٔ امتیازبندی یک متقاضی همراه رتبهٔ تخصیص.
type LoanScoreRow struct {
	EmployeeID   string          `json:"employeeId"`
	EmployeeName string          `json:"employeeName"`
	Total        int             `json:"total"`
	Rank         int             `json:"rank"`
	Breakdown    []LoanScoreLine `json:"breakdown"`
}

// LoanAllocationResult خروجی امتیازبندی و رتبه‌بندی تخصیص وام.
type LoanAllocationResult struct {
	Rows   []LoanScoreRow `json:"rows"`
	Count  int            `json:"count"`
	NoteFa string         `json:"noteFa,omitempty"`
}

// LoanGrantRequest ورودی فرم تخصیص وام: یک نوع وام با شرایط یکسان به یک یا چند کارمند.
//
// Amount مبلغ وام «هر نفر» است و Installments تعداد اقساط؛ خالی بماندن اقساط یعنی سقف
// اقساط همان نوع وام. نرخ کارمزد و مبلغ هر قسط از دامنه محاسبه می‌شود، نه از این ورودی.
type LoanGrantRequest struct {
	LoanType   string `json:"loanType"`
	Bank       string `json:"bank,omitempty"`
	Management string `json:"management,omitempty"`
	Amount     int64  `json:"amount"`
	// Installments تعداد اقساط؛ صفر یعنی سقف اقساط همان نوع وام.
	Installments        int    `json:"installments,omitempty"`
	ReceivedDate        string `json:"receivedDate"`
	Purpose             string `json:"purpose,omitempty"`
	GuarantorName       string `json:"guarantorName,omitempty"`
	GuarantorNationalID string `json:"guarantorNationalId,omitempty"`
	GuarantorRelation   string `json:"guarantorRelation,omitempty"`
	// SupplementOf شناسهٔ وام پایه؛ فقط برای انواع «متمم» الزامی است.
	SupplementOf string `json:"supplementOf,omitempty"`
	// Unit واحد سازمانی متقاضی (اختیاری) از کاتالوگ مرجع واحدها.
	Unit string `json:"unit,omitempty"`
	// EmployeeIDs کارکنان هدف؛ خالی یعنی همهٔ کارکنان فعال سازمان.
	EmployeeIDs []string `json:"employeeIds,omitempty"`
	NoteFa      string   `json:"noteFa,omitempty"`
}

// LoanGrantRow یک ردیف تخصیص وام (پیش‌نمایش یا رکورد ساخته‌شده).
type LoanGrantRow struct {
	EmployeeID     string  `json:"employeeId"`
	EmployeeName   string  `json:"employeeName"`
	DepartmentName string  `json:"departmentName,omitempty"`
	LoanTypeFa     string  `json:"loanTypeFa"`
	BankFa         string  `json:"bankFa,omitempty"`
	ManagementKey  string  `json:"managementKey,omitempty"`
	ManagementFa   string  `json:"managementFa,omitempty"`
	UnitKey        string  `json:"unitKey,omitempty"`
	UnitFa         string  `json:"unitFa,omitempty"`
	Amount         float64 `json:"amount"`
	FeeRatePercent float64 `json:"feeRatePercent"`
	FeeAmount      float64 `json:"feeAmount"`
	Installments   int     `json:"installments"`
	// InstallmentAmount مبلغ هر قسط پس از افزودن کارمزد.
	InstallmentAmount float64 `json:"installmentAmount"`
	RecordID          string  `json:"recordId,omitempty"`
}

// LoanGrantResult نتیجهٔ پیش‌نمایش یا ثبت تخصیص وام.
type LoanGrantResult struct {
	Rows []LoanGrantRow `json:"rows"`
	// Count تعداد کارکنان مشمول تخصیص.
	Count int `json:"count"`
	// TotalAmount جمع مبلغ وام‌های تخصیص‌یافته و TotalFee جمع کارمزد آن‌ها.
	TotalAmount float64 `json:"totalAmount"`
	TotalFee    float64 `json:"totalFee"`
	// Created شمار رکوردهایی که در همین فراخوانی ساخته شدند (در پیش‌نمایش صفر).
	Created int    `json:"created"`
	NoteFa  string `json:"noteFa"`
}
