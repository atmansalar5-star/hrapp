package contracts

// CardChargeRequest درخواست ساخت شارژ یا فایل شارژ کارت‌های رفاهی.
//
// یک درخواست هم شارژ انفرادی و هم گروهی را پوشش می‌دهد: `scope=group` یعنی همهٔ
// مشمولان فعال کارت و `scope=individual` یعنی فقط کارکنان فهرست‌شده در EmployeeIDs.
// `TrialRun` فقط پیش‌نمایش می‌سازد و هیچ رکوردی ذخیره نمی‌کند.
type CardChargeRequest struct {
	Card        string   `json:"card"`     // petrokart | naftmart
	Scope       string   `json:"scope"`    // group | individual
	Sections    []string `json:"sections"` // بخش‌های خواسته‌شده؛ خالی یعنی همهٔ بخش‌های کارت
	Period      string   `json:"period"`   // دورهٔ شارژ، قالب شمسی YYYY/MM
	OccasionID  string   `json:"occasionId"`
	EmployeeIDs []string `json:"employeeIds"`
	TrialRun    bool     `json:"trialRun"`
	// MaxHeads سقف «سر» های شارژ برای هر کارمند: خودِ کارمند یک سر و هر فرد تحت تکفل
	// یک سر. صفر یعنی بی‌سقف (رفتار پیش از این). افراد بیش از سقف شارژ نمی‌شوند و در
	// هشدارهای نتیجه با عدد فارسی گزارش می‌شوند.
	MaxHeads int `json:"maxHeads,omitempty"`
}

// CardChargeRow یک ردیف شارژ برای یک ذی‌نفع (کارمند یا فرد تحت تکفل) در یک بخش.
type CardChargeRow struct {
	EmployeeID   string `json:"employeeId"`
	EmployeeName string `json:"employeeName"`
	Beneficiary  string `json:"beneficiary"`
	Relation     string `json:"relation"`
	Section      string `json:"section"`
	SectionFa    string `json:"sectionFa"`
	OccasionFa   string `json:"occasionFa,omitempty"`
	Amount       int64  `json:"amount"`
	BasisFa      string `json:"basisFa"`
}

// CardChargeTotal جمع یک بخش در فایل شارژ.
type CardChargeTotal struct {
	Section   string `json:"section"`
	SectionFa string `json:"sectionFa"`
	Count     int    `json:"count"`
	Total     int64  `json:"total"`
}

// CardChargeResult نتیجهٔ ساخت شارژ: ردیف‌ها، جمع کل، جمع هر بخش و (در حالت غیر
// آزمایشی) شناسهٔ فایل شارژ ثبت‌شده.
type CardChargeResult struct {
	Card      string            `json:"card"`
	CardFa    string            `json:"cardFa"`
	Period    string            `json:"period"`
	ScopeFa   string            `json:"scopeFa"`
	Rows      []CardChargeRow   `json:"rows"`
	Count     int               `json:"count"`
	Total     int64             `json:"total"`
	BySection []CardChargeTotal `json:"bySection"`
	FileID    string            `json:"fileId,omitempty"`
	FileNoFa  string            `json:"fileNoFa,omitempty"`
	Warnings  []string          `json:"warnings,omitempty"`
}

// CardOption یک گزینهٔ انتخابی (کارت، بخش، مناسبت) برای فرم‌های رابط وب.
type CardOption struct {
	Key     string `json:"key"`
	TitleFa string `json:"titleFa"`
	Card    string `json:"card,omitempty"`
	Amount  int64  `json:"amount,omitempty"`
	DateFa  string `json:"dateFa,omitempty"`
}

// CardOptions فهرست‌های پایهٔ ماژول کارت‌ها: کارت‌ها، بخش‌ها با نرخ جاری و
// مناسبت‌های فعال. رابط وب از همین می‌سازد و هیچ برچسبی را از خودش نمی‌نویسد.
type CardOptions struct {
	Cards       []CardOption            `json:"cards"`
	Sections    []CardOption            `json:"sections"`
	Occasions   []CardOption            `json:"occasions"`
	RatesByCard map[string][]CardOption `json:"ratesByCard"`
	// TodayFa تاریخ شمسی امروز سرور؛ فرم شارژ دورهٔ جاری را از همین پیشنهاد می‌دهد تا
	// ساعت کلاینت مرجع نباشد.
	TodayFa string `json:"todayFa"`
}
