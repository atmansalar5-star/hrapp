package contracts

// قراردادهای داشبورد «تخصیص سهمیهٔ تسهیلات».
//
// این داشبورد سه پرسش را پاسخ می‌دهد: (۱) سهمیهٔ مصوب هر مدیریت بندر امام چقدر است و
// چقدر از آن مطالبه شده، (۲) درخواست‌ها در کدام مرحله از ارسال به مدیریت مالی‌اند،
// (۳) سهم نیروی شرکتی در برابر ارکان ثالث و پراکندگی نرخ کارمزد تسهیلات چقدر است.
// همهٔ اعداد از کاتالوگ دامنه (سهمیه و مرحله) و رکوردهای واقعی وام و پروندهٔ کارکنان
// ساخته می‌شود؛ رابط وب هیچ محاسبه‌ای از خودش ندارد.

// LoanQuotaUnitRow یک ردیف نمودار تخصیص سهمیه برای یک مدیریت.
type LoanQuotaUnitRow struct {
	UnitKey          string  `json:"unitKey"`
	UnitFa           string  `json:"unitFa"`
	SharePercent     float64 `json:"sharePercent"`
	QuotaAmount      float64 `json:"quotaAmount"`
	Eligible         int     `json:"eligible"`
	Applicants       int     `json:"applicants"`
	Pending          int     `json:"pending"`
	Sent             int     `json:"sent"`
	Rejected         int     `json:"rejected"`
	RecordsCount     int     `json:"recordsCount"`
	CompanyCount     int     `json:"companyCount"`
	ThirdPartyCount  int     `json:"thirdPartyCount"`
	RequestedAmount  float64 `json:"requestedAmount"`
	SentAmount       float64 `json:"sentAmount"`
	CompanyAmount    float64 `json:"companyAmount"`
	ThirdPartyAmount float64 `json:"thirdPartyAmount"`
	// ClaimPercent درصد تعلق: مبلغ درخواستی ÷ سهمیهٔ ریالی مدیریت.
	ClaimPercent float64 `json:"claimPercent"`
	// CoveragePercent پوشش مشمولین: تعداد متقاضیان ÷ سرشماری مشمولین.
	CoveragePercent float64 `json:"coveragePercent"`
}

// LoanQuotaStageRow توزیع درخواست‌ها بین مراحل ارسال به مدیریت مالی.
type LoanQuotaStageRow struct {
	StageKey        string  `json:"stageKey"`
	StageFa         string  `json:"stageFa"`
	SentToFinance   bool    `json:"sentToFinance"`
	Count           int     `json:"count"`
	Amount          float64 `json:"amount"`
	CompanyCount    int     `json:"companyCount"`
	ThirdPartyCount int     `json:"thirdPartyCount"`
}

// LoanQuotaGroupRow سهم گروه کارگزینی (شرکتی / ارکان ثالث) از درخواست‌ها و ارسال‌ها.
type LoanQuotaGroupRow struct {
	GroupKey     string  `json:"groupKey"`
	GroupFa      string  `json:"groupFa"`
	Count        int     `json:"count"`
	Amount       float64 `json:"amount"`
	SentCount    int     `json:"sentCount"`
	SentAmount   float64 `json:"sentAmount"`
	SharePercent float64 `json:"sharePercent"`
	// SentSharePercent درصد سهمیهٔ ارسالی (بر پایهٔ مبلغ ارسال‌شده به مالی).
	SentSharePercent float64 `json:"sentSharePercent"`
}

// LoanQuotaFeeTierRow درخواست‌ها به تفکیک درصد کارمزد تسهیلات.
type LoanQuotaFeeTierRow struct {
	FeeRatePercent   float64 `json:"feeRatePercent"`
	Count            int     `json:"count"`
	Amount           float64 `json:"amount"`
	CompanyCount     int     `json:"companyCount"`
	ThirdPartyCount  int     `json:"thirdPartyCount"`
	CompanyAmount    float64 `json:"companyAmount"`
	ThirdPartyAmount float64 `json:"thirdPartyAmount"`
}

// LoanQuotaDashboard خروجی کامل داشبورد تخصیص سهمیهٔ تسهیلات.
type LoanQuotaDashboard struct {
	Units      []LoanQuotaUnitRow    `json:"units"`
	Stages     []LoanQuotaStageRow   `json:"stages"`
	Groups     []LoanQuotaGroupRow   `json:"groups"`
	FeeTiers   []LoanQuotaFeeTierRow `json:"feeTiers"`
	QuotaTotal float64               `json:"quotaTotalAmount"`
	Requested  float64               `json:"requestedTotalAmount"`
	SentAmount float64               `json:"sentTotalAmount"`
	Eligible   int                   `json:"eligibleTotal"`
	Applicants int                   `json:"applicantTotal"`
	Pending    int                   `json:"pendingTotal"`
	Sent       int                   `json:"sentTotal"`
	Rejected   int                   `json:"rejectedTotal"`
	CompanyPct float64               `json:"companySharePercent"`
	ThirdPct   float64               `json:"thirdPartySharePercent"`
	NoteFa     string                `json:"noteFa"`
}
