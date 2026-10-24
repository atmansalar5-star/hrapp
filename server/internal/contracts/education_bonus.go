package contracts

// BonusRequest درخواست محاسبهٔ پاداش تحصیلی فرزندان ممتاز.
//
// `scope=group` یعنی همهٔ دانش‌آموزان ممتاز مشمول و `scope=individual` یعنی فقط
// کارکنان فهرست‌شده در EmployeeIDs. `Levels` خالی یعنی همهٔ مقاطع محاسبه شوند.
// `TrialRun` فقط پیش‌نمایش می‌سازد و هیچ رکوردی ذخیره نمی‌کند.
type BonusRequest struct {
	Levels      []string `json:"levels"` // مقطع‌های خواسته‌شده؛ خالی یعنی همه
	Scope       string   `json:"scope"`  // group | individual
	Period      string   `json:"period"` // سال تحصیلی، مثلاً 1404/1405
	EmployeeIDs []string `json:"employeeIds"`
	TrialRun    bool     `json:"trialRun"`
}

// BonusRow یک ردیف پاداش تحصیلی برای یک دانش‌آموز ممتاز.
type BonusRow struct {
	EmployeeID   string `json:"employeeId"`
	EmployeeName string `json:"employeeName"`
	Student      string `json:"student"`
	Relation     string `json:"relation"`
	Level        string `json:"level"`
	LevelFa      string `json:"levelFa"`
	Grade        string `json:"grade,omitempty"`
	Amount       int64  `json:"amount"`
	BasisFa      string `json:"basisFa"`
}

// BonusTotal جمع یک مقطع در فایل پاداش.
type BonusTotal struct {
	Level   string `json:"level"`
	LevelFa string `json:"levelFa"`
	Count   int    `json:"count"`
	Total   int64  `json:"total"`
}

// BonusResult نتیجهٔ محاسبهٔ پاداش: ردیف‌ها، جمع کل، جمع هر مقطع و (در حالت غیر
// آزمایشی) شناسهٔ فایل ثبت‌شده.
type BonusResult struct {
	Period   string       `json:"period"`
	ScopeFa  string       `json:"scopeFa"`
	Rows     []BonusRow   `json:"rows"`
	Count    int          `json:"count"`
	Total    int64        `json:"total"`
	ByLevel  []BonusTotal `json:"byLevel"`
	FileID   string       `json:"fileId,omitempty"`
	FileNoFa string       `json:"fileNoFa,omitempty"`
	Warnings []string     `json:"warnings,omitempty"`
}

// BonusOption یک گزینهٔ انتخابی (مقطع یا مبلغ مصوب) برای فرم‌های رابط وب.
type BonusOption struct {
	Key     string `json:"key"`
	TitleFa string `json:"titleFa"`
	Amount  int64  `json:"amount,omitempty"`
}

// BonusOptions فهرست‌های پایهٔ ماژول پاداش تحصیلی: مقاطع با مبلغ مصوب جاری و
// تاریخ شمسی امروز سرور.
type BonusOptions struct {
	Levels  []BonusOption `json:"levels"`
	TodayFa string        `json:"todayFa"`
}
