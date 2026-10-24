package contracts

import "context"

// HistoryChange یک رخداد تغییر روی یک سند سابقه: چه کسی، کِی و با چه اقدامی سند را
// عوض کرده است. این «سابقهٔ تغییرات» خودِ سند است، نه سابقهٔ اسناد دیگر یک کارمند.
type HistoryChange struct {
	// At تاریخ رخداد به قالب شمسی YYYY/MM/DD.
	At       string `json:"at,omitempty"`
	Actor    string `json:"actor,omitempty"`
	Action   string `json:"action,omitempty"`
	ActionFa string `json:"actionFa,omitempty"`
	NoteFa   string `json:"noteFa,omitempty"`
}

// HistoryField یک قلم کلیدی سند سابقه (برچسب فارسی اعلامی + مقدار).
//
// اقلام از شناسهٔ دامنهٔ همان ماژول می‌آیند، پس هیچ برچسبی در رابط کاربری ساخته نمی‌شود.
type HistoryField struct {
	Key     string `json:"key"`
	LabelFa string `json:"labelFa"`
	Value   string `json:"value"`
	Type    string `json:"type,omitempty"`
}

// HistoryEntry یک سطر «سابقه»: سند یا رکوردی از یک ماژول که به یک کارمند مربوط است.
//
// این ساختار عمداً عمومی است: حکم حقوق، قرارداد، ترفیع، انتصاب، مدرک/سند بایگانی، وام،
// فرد تحت تکفل و هر رکورد گردش‌کاری دیگر با همین شکل نمایش داده می‌شود. اقلام اختصاصی
// هر سند در Fields می‌آید و سابقهٔ تغییرات خودِ سند در Changes.
//
// Latest یعنی «آخرین سند همین نوع برای این کارمند» — ویژگی نمایشی که لایهٔ چیدن سابقه
// می‌گذارد تا کاربر سند جاری را از اسناد پیشین جدا ببیند.
type HistoryEntry struct {
	Module         string `json:"module"`
	ModuleTitleFa  string `json:"moduleTitleFa"`
	Kind           string `json:"kind,omitempty"`
	KindTitleFa    string `json:"kindTitleFa,omitempty"`
	RecordID       string `json:"recordId"`
	Title          string `json:"title"`
	EmployeeID     string `json:"employeeId,omitempty"`
	EmployeeName   string `json:"employeeName,omitempty"`
	DepartmentName string `json:"departmentName,omitempty"`
	// Date تاریخ مؤثر سند (شمسی YYYY/MM/DD) و DateLabelFa برچسب همان تاریخ.
	Date        string          `json:"date,omitempty"`
	DateLabelFa string          `json:"dateLabelFa,omitempty"`
	Status      string          `json:"status,omitempty"`
	StatusFa    string          `json:"statusFa,omitempty"`
	Fields      []HistoryField  `json:"fields,omitempty"`
	Changes     []HistoryChange `json:"changes,omitempty"`
	NoteFa      string          `json:"noteFa,omitempty"`
	Latest      bool            `json:"latest,omitempty"`
}

// HistoryTimeline پاسخ «سوابق یک کارمند»: سطرهای سابقه از همهٔ ماژول‌های مجاز، به ترتیب
// نزولی تاریخ، به‌همراه شمارش هر ماژول. Warnings خطای ماژول‌هایی است که سابقهٔ خود را
// برنگردانده‌اند؛ عمدی است که خطای یک ماژول کل سابقه را از کار نیندازد، ولی هم‌زمان
// پنهان هم نماند.
type HistoryTimeline struct {
	EmployeeID   string         `json:"employeeId"`
	EmployeeName string         `json:"employeeName,omitempty"`
	Items        []HistoryEntry `json:"items"`
	Total        int            `json:"total"`
	ByModule     map[string]int `json:"byModule,omitempty"`
	Warnings     []string       `json:"warnings,omitempty"`
	NoteFa       string         `json:"noteFa,omitempty"`
}

// HistorySource منبع «سابقه» یک ماژول.
//
// هر ماژول مالک دادهٔ خودش است و سابقهٔ کارمند را خودش اعلام می‌کند؛ این‌گونه سابقهٔ
// چندپرونده‌ای (احکام، قراردادها، ترفیعات، مدارک، وام…) بدون شکستن مرز ماژول ساخته
// می‌شود و هیچ ماژولی داخلی‌های ماژول دیگر را نمی‌بیند. قاعدهٔ دسترسی اینجا اعمال
// نمی‌شود: لایهٔ تجمیع، پیش از خواندن هر منبع، دسترسی نقش به همان ماژول را می‌سنجد.
type HistorySource interface {
	Module() string
	ModuleTitleFa() string
	// HistoryOf سابقهٔ یک کارمند را برمی‌گرداند؛ ترتیب اهمیتی ندارد و لایهٔ تجمیع آن را
	// بر پایهٔ تاریخ می‌چیند.
	HistoryOf(ctx context.Context, employeeID string) ([]HistoryEntry, error)
}
