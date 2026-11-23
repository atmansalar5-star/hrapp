package contracts

import "hrms/internal/rules"

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

/* ---------------------- صورت حساب و اقساط وام ---------------------- */

// LoanStatementView صورت حساب یک وام برای رابط و گزارش: سرصفحهٔ وام (بانک، مدیریت،
// ضامن) به‌همراه صورت حساب کاملِ ساخته‌شده در بستهٔ قواعد.
type LoanStatementView struct {
	LoanID             string `json:"loanId"`
	TitleFa            string `json:"titleFa"`
	EmployeeID         string `json:"employeeId,omitempty"`
	EmployeeName       string `json:"employeeName,omitempty"`
	Status             string `json:"status"`
	StatusFa           string `json:"statusFa"`
	BankFa             string `json:"bankFa,omitempty"`
	ManagementFa       string `json:"managementFa,omitempty"`
	PurposeFa          string `json:"purpose,omitempty"`
	GuarantorName      string `json:"guarantorName,omitempty"`
	GuarantorRelation  string `json:"guarantorRelation,omitempty"`
	InstallmentRecords int    `json:"installmentRecords"`
	// Statement صورت حساب کامل: شرایط، جدول اقساط، تاریخچه و مانده بدهی.
	Statement rules.LoanStatement `json:"statement"`
}

// LoanCodeGap نوع وامی که در جدول رسمی کد فیش ندارد (برای مکاتبهٔ مالی).
type LoanCodeGap struct {
	Key     string `json:"key"`
	TitleFa string `json:"titleFa"`
}

// LoanStatementTotals جمع‌های کارتابل صورت حساب وام‌ها.
type LoanStatementTotals struct {
	Loans            int   `json:"loans"`
	Active           int   `json:"active"`
	Amount           int64 `json:"amount"`
	FeeAmount        int64 `json:"feeAmount"`
	TotalPayable     int64 `json:"totalPayable"`
	PaidAmount       int64 `json:"paidAmount"`
	RemainingDebt    int64 `json:"remainingDebt"`
	OverdueAmount    int64 `json:"overdueAmount"`
	MonthlyDeduction int64 `json:"monthlyDeduction"`
}

// LoanStatementPage کارتابل صورت حساب وام‌ها: فهرست وام‌ها با مانده بدهی و جمع‌ها.
type LoanStatementPage struct {
	Items    []LoanStatementView `json:"items"`
	Count    int                 `json:"count"`
	Totals   LoanStatementTotals `json:"totals"`
	PeriodFa string              `json:"periodFa,omitempty"`
	TodayFa  string              `json:"todayFa,omitempty"`
	CodeGaps []LoanCodeGap       `json:"codeGaps,omitempty"`
	NoteFa   string              `json:"noteFa,omitempty"`
}

// LoanSchedulePreviewResult پیش‌نمایش جدول اقساط پیش از ثبت وام (بدون ذخیره‌سازی).
type LoanSchedulePreviewResult struct {
	Schedule          rules.LoanSchedule `json:"schedule"`
	PaymentCode       string             `json:"paymentCode,omitempty"`
	PaymentCodeFa     string             `json:"paymentCodeFa,omitempty"`
	InstallmentCode   string             `json:"installmentCode,omitempty"`
	InstallmentCodeFa string             `json:"installmentCodeFa,omitempty"`
	CodeNoteFa        string             `json:"codeNoteFa,omitempty"`
	Problems          []string           `json:"problems,omitempty"`
	NoteFa            string             `json:"noteFa,omitempty"`
}

// LoanScheduleBuildResult نتیجهٔ ساخت/به‌روزرسانی رکوردهای قسط یک وام.
type LoanScheduleBuildResult struct {
	LoanID    string            `json:"loanId"`
	Created   int               `json:"created"`
	Updated   int               `json:"updated"`
	Unchanged int               `json:"unchanged"`
	Statement LoanStatementView `json:"statement"`
	NoteFa    string            `json:"noteFa,omitempty"`
}

// LoanInstallmentDueRow یک قسط سررسیدشدهٔ یک دوره برای کسر از حقوق.
type LoanInstallmentDueRow struct {
	LoanID        string `json:"loanId"`
	LoanTypeFa    string `json:"loanTypeFa,omitempty"`
	EmployeeID    string `json:"employeeId,omitempty"`
	EmployeeName  string `json:"employeeName,omitempty"`
	Seq           int    `json:"seq"`
	PeriodFa      string `json:"periodFa,omitempty"`
	Amount        int64  `json:"amount"`
	Status        string `json:"status"`
	StatusFa      string `json:"statusFa"`
	CodeFa        string `json:"codeFa,omitempty"`
	CodeTitleFa   string `json:"codeTitleFa,omitempty"`
	RemainingDebt int64  `json:"remainingDebt"`
}

// LoanInstallmentDuePage فهرست اقساط یک دوره برای پردازش حقوق.
type LoanInstallmentDuePage struct {
	Rows          []LoanInstallmentDueRow `json:"rows"`
	Count         int                     `json:"count"`
	PeriodFa      string                  `json:"periodFa,omitempty"`
	TotalAmount   int64                   `json:"totalAmount"`
	PaidAmount    int64                   `json:"paidAmount"`
	PendingAmount int64                   `json:"pendingAmount"`
	NoteFa        string                  `json:"noteFa,omitempty"`
}
