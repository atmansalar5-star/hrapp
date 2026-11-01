// پیاده‌سازی ماژول «archive» — «education»: اطلاعات پایهٔ تحصیلی و مدرک ملاک حقوق.
//
// چرا این لایه: پروندهٔ پرسنلی باید یک نگاه کامل به تحصیلات بدهد — همهٔ مقاطع، **آخرین
// مدرک**، **مدرک ملاک حقوق** (با صورت‌جلسهٔ شورای تعدیل مدرک، درصد ارتباط و معادل‌سازی
// ایثارگران) و هشدارهای صادقانه. این خواندن ترکیبی از رکوردهای همین ماژول است، پس در
// ماژول مالکِ همان داده ساخته می‌شود و رابط فقط نمایش می‌دهد.
//
// قاعده‌ها هیچ‌کدام اینجا نیستند: مرجع مقطع‌ها و ارتباط تحصیلات در بستهٔ `contracts`
// اعلام شده و قواعد گردشکار (تأیید شورا، معادل‌سازی ایثارگران) در `domain` همین ماژول.
package archive

import (
	"net/http"
	"sort"
	"strings"

	"hrms/internal/contracts"
	"hrms/internal/modules/archive/domain"
	"hrms/internal/platform/httpx"
	"hrms/internal/platform/modulekit"
	"hrms/internal/platform/recordkit"
)

// educationItem یک رکورد تحصیلی در پاسخ خلاصهٔ پرونده.
type educationItem struct {
	ID             string `json:"id"`
	Title          string `json:"title"`
	Level          string `json:"level"`
	LevelFa        string `json:"levelFa"`
	LevelOrder     int    `json:"levelOrder"`
	Field          string `json:"field,omitempty"`
	Orientation    string `json:"orientation,omitempty"`
	University     string `json:"university,omitempty"`
	GraduationDate string `json:"graduationDate,omitempty"`
	DiplomaNo      string `json:"diplomaNo,omitempty"`
	Average        string `json:"average,omitempty"`
	Relevance      string `json:"relevance,omitempty"`
	RelevanceFa    string `json:"relevanceFa,omitempty"`
	// RelevancePercent درصد ارتباط تحصیلات با شغل (میزان ارتباط مدرک).
	RelevancePercent string `json:"relevancePercent,omitempty"`
	CouncilStatus    string `json:"councilStatus,omitempty"`
	CouncilStatusFa  string `json:"councilStatusFa,omitempty"`
	CouncilNo        string `json:"councilNo,omitempty"`
	CouncilDate      string `json:"councilDate,omitempty"`
	Status           string `json:"status"`
	StatusFa         string `json:"statusFa"`
	// LatestDegree آیا این رکورد «آخرین مدرک تحصیلی» کارمند است.
	LatestDegree bool `json:"latestDegree"`
	// IsSalaryBasis آیا همین مدرک، ملاک جاری حقوق است.
	IsSalaryBasis bool `json:"isSalaryBasis"`
}

// degreeBasisItem تصمیم «مدرک ملاک حقوق».
type degreeBasisItem struct {
	ID             string `json:"id"`
	Title          string `json:"title"`
	Education      string `json:"education,omitempty"`
	Level          string `json:"level,omitempty"`
	LevelFa        string `json:"levelFa,omitempty"`
	Field          string `json:"field,omitempty"`
	Orientation    string `json:"orientation,omitempty"`
	Relevance      string `json:"relevance,omitempty"`
	RelevanceFa    string `json:"relevanceFa,omitempty"`
	RelevancePct   string `json:"relevancePercent,omitempty"`
	CouncilNo      string `json:"councilNo,omitempty"`
	CouncilDate    string `json:"councilDate,omitempty"`
	EffectiveDate  string `json:"effectiveDate,omitempty"`
	SalaryGroup    string `json:"salaryGroup,omitempty"`
	VeteranUpgrade string `json:"veteranUpgrade,omitempty"`
	VeteranFa      string `json:"veteranUpgradeFa,omitempty"`
	UpgradedLevel  string `json:"upgradedLevel,omitempty"`
	// UpgradedLevelFa مقطع معادل‌سازی‌شده برای ایثارگران (یک مقطع بالاتر).
	UpgradedLevelFa string `json:"upgradedLevelFa,omitempty"`
	SalaryEffect    string `json:"salaryEffect,omitempty"`
	Status          string `json:"status"`
	StatusFa        string `json:"statusFa"`
	// EffectiveLevelFa مقطعی که واقعاً مبنای حقوق است: برای ایثارگران همان مقطع
	// معادل‌سازی‌شده و در غیر آن، مقطع خودِ مدرک.
	EffectiveLevelFa string `json:"effectiveLevelFa,omitempty"`
}

// veteranInfo وضعیت ایثارگری کارمند (از ماژول امور ایثارگران خوانده می‌شود).
type veteranInfo struct {
	IsVeteran bool `json:"isVeteran"`
	// Confirmed یعنی پروندهٔ ایثارگری تأیید نهایی شده است (نه فقط بررسی‌شده).
	Confirmed  bool   `json:"confirmed"`
	TypeFa     string `json:"typeFa,omitempty"`
	StatusFa   string `json:"statusFa,omitempty"`
	Percent    string `json:"percent,omitempty"`
	RecordID   string `json:"recordId,omitempty"`
	NoteFa     string `json:"noteFa,omitempty"`
	UpgradeFa  string `json:"upgradeNoteFa,omitempty"`
	IsVerified bool   `json:"verified"`
}

// educationView پاسخ «اطلاعات پایهٔ تحصیلی» یک کارمند.
type educationView struct {
	Module      string            `json:"module"`
	EmployeeID  string            `json:"employeeId"`
	Records     []educationItem   `json:"records"`
	Latest      *educationItem    `json:"latest,omitempty"`
	Bases       []degreeBasisItem `json:"bases"`
	ActiveBasis *degreeBasisItem  `json:"activeBasis,omitempty"`
	BasisNoteFa string            `json:"basisNoteFa"`
	Veteran     veteranInfo       `json:"veteran"`
	Summary     map[string]any    `json:"summary"`
	Options     map[string]any    `json:"options"`
	Warnings    []string          `json:"warnings,omitempty"`
	NoteFa      string            `json:"noteFa"`
}

// registerEducationEndpoints گزینه‌های فرم تحصیلی و خلاصهٔ پروندهٔ تحصیلی را ثبت می‌کند.
func registerEducationEndpoints(mux *http.ServeMux, deps modulekit.Deps, svc *recordkit.Service) {
	if svc == nil {
		return
	}
	read := func(fn http.HandlerFunc) http.HandlerFunc { return deps.Require("archive", "read", fn) }

	// گزینه‌های فرم تحصیلی: مقطع‌ها، ارتباط تحصیلات، وضعیت شورای تعدیل و بله/خیر.
	mux.HandleFunc("GET /api/v1/archive/options", read(func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, educationOptions())
	}))

	// خلاصهٔ تحصیلی یک کارمند: همهٔ مقاطع، آخرین مدرک، مدرک ملاک (جاری و پیشین) و
	// وضعیت ایثارگری — همان چیزی که پروندهٔ پرسنلی و ساختار سازمانی می‌خوانند.
	mux.HandleFunc("GET /api/v1/archive/education/{employeeId}", read(func(w http.ResponseWriter, r *http.Request) {
		view, err := buildEducationView(r, deps, svc, r.PathValue("employeeId"))
		if err != nil {
			httpx.WriteErrorMsg(w, http.StatusInternalServerError, contracts.CodeInternal, err.Error(), nil)
			return
		}
		if view.EmployeeID == "" {
			httpx.WriteErrorMsg(w, http.StatusBadRequest, contracts.CodeBadRequest, "کد پرسنلی در نشانی نیامده است.", nil)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, view)
	}))
}

// educationOptions فهرست گزینه‌های فرم‌های تحصیلی (همه از دامنهٔ مشترک).
func educationOptions() map[string]any {
	return map[string]any{
		"degreeLevels":         contracts.DegreeLevels(),
		"relevances":           contracts.DegreeRelevances(),
		"councilStatuses":      contracts.DegreeCouncilStatuses(),
		"yesNo":                contracts.YesNos(),
		"educationKind":        domain.KindEducations,
		"educationKindTitleFa": "سابقهٔ تحصیلی",
		"basisKind":            domain.KindDegreeBases,
		"basisKindTitleFa":     "مدرک ملاک حقوق",
		"noteFa": "مقطع‌ها، ارتباط تحصیلات و وضعیت شورای تعدیل مدرک از دامنهٔ سرور می‌آید؛ " +
			"برای ایثارگران، «یک مقطع بالاتر» با همان فهرست مقاطع معادل‌سازی می‌شود.",
	}
}

// buildEducationView پروندهٔ تحصیلی یک کارمند را می‌سازد.
func buildEducationView(r *http.Request, deps modulekit.Deps, svc *recordkit.Service, employeeID string) (educationView, error) {
	id := strings.TrimSpace(employeeID)
	view := educationView{
		Module:     svc.Module(),
		EmployeeID: id,
		Records:    []educationItem{},
		Bases:      []degreeBasisItem{},
		Options:    educationOptions(),
		Summary:    map[string]any{},
		NoteFa: "سابقهٔ تحصیلی هر کارمند یک رکورد جداگانه است (مقطع، رشته، گرایش و ارتباط با شغل) " +
			"و تصمیم «کدام مدرک ملاک حقوق است» در فرم جداگانهٔ خودش ثبت می‌شود؛ هر دو در سوابق " +
			"پروندهٔ کارمند می‌مانند.",
	}
	if id == "" {
		return view, nil
	}

	// رکوردهای تحصیلی همین ماژول.
	educations, err := listModuleRecords(r, svc, domain.KindEducations, id)
	if err != nil {
		return view, err
	}
	bases, err := listModuleRecords(r, svc, domain.KindDegreeBases, id)
	if err != nil {
		return view, err
	}

	// آخرین مدرک: بالاترین مقطع (و در مقطع مساوی، نوترین تاریخ فراغت).
	levels := make([]string, 0, len(educations))
	for _, rec := range educations {
		levels = append(levels, rec.Fields["level"])
	}
	latestLevel, hasLatest := contracts.HighestDegreeLevel(levels)
	latestIndex := -1
	for i, rec := range educations {
		if !hasLatest || rec.Fields["level"] != latestLevel.Key {
			continue
		}
		if latestIndex < 0 || rec.Fields["graduationDate"] > educations[latestIndex].Fields["graduationDate"] {
			latestIndex = i
		}
	}

	activeBasisID := ""
	for _, rec := range bases {
		if rec.Status == domain.BasisActive {
			activeBasisID = rec.Fields["education"]
			break
		}
	}
	basisByEducation := map[string]bool{}
	if activeBasisID != "" {
		basisByEducation[activeBasisID] = true
	}

	for i, rec := range educations {
		level, _ := contracts.DegreeLevelByKey(rec.Fields["level"])
		item := educationItem{
			ID: rec.ID, Title: rec.Title,
			Level: rec.Fields["level"], LevelFa: contracts.DegreeLevelTitleFa(rec.Fields["level"]),
			LevelOrder: level.Order,
			Field:      rec.Fields["field"], Orientation: rec.Fields["orientation"],
			University: rec.Fields["university"], GraduationDate: rec.Fields["graduationDate"],
			DiplomaNo: rec.Fields["diplomaNo"], Average: rec.Fields["average"],
			Relevance: rec.Fields["relevance"], RelevanceFa: contracts.DegreeRelevanceTitleFa(rec.Fields["relevance"]),
			RelevancePercent: rec.Fields["relevancePercent"],
			CouncilStatus:    rec.Fields["councilStatus"], CouncilStatusFa: contracts.DegreeCouncilStatusTitleFa(rec.Fields["councilStatus"]),
			CouncilNo: rec.Fields["councilNo"], CouncilDate: rec.Fields["councilDate"],
			Status: rec.Status, StatusFa: rec.StatusFa,
			LatestDegree: i == latestIndex,
			// ملاک جاری با کد سابقهٔ تحصیلی روی تصمیم ثبت می‌شود و اگر خالی باشد، با
			// همان مقطع و رشتهٔ مدرک تطبیق داده می‌شود تا دادهٔ قدیمی هم گم نشود.
			IsSalaryBasis: basisByEducation[rec.ID] || basisMatchesRecord(rec, bases),
		}
		if item.LevelFa == "" {
			item.LevelFa = rec.Fields["level"]
		}
		view.Records = append(view.Records, item)
		if item.LatestDegree {
			copy := item
			view.Latest = &copy
		}
	}

	for _, rec := range bases {
		item := degreeBasisItem{
			ID: rec.ID, Title: rec.Title, Education: rec.Fields["education"],
			Level: rec.Fields["level"], LevelFa: contracts.DegreeLevelTitleFa(rec.Fields["level"]),
			Field: rec.Fields["field"], Orientation: rec.Fields["orientation"],
			Relevance: rec.Fields["relevance"], RelevanceFa: contracts.DegreeRelevanceTitleFa(rec.Fields["relevance"]),
			RelevancePct: rec.Fields["relevancePercent"],
			CouncilNo:    rec.Fields["councilNo"], CouncilDate: rec.Fields["councilDate"],
			EffectiveDate: rec.Fields["effectiveDate"], SalaryGroup: rec.Fields["salaryGroup"],
			VeteranUpgrade: rec.Fields["veteranUpgrade"],
			VeteranFa:      contracts.YesNoTitleFa(rec.Fields["veteranUpgrade"]),
			UpgradedLevel:  rec.Fields["upgradedLevel"],
			SalaryEffect:   rec.Fields["salaryEffect"],
			Status:         rec.Status, StatusFa: rec.StatusFa,
		}
		item.UpgradedLevelFa = contracts.DegreeLevelTitleFa(rec.Fields["upgradedLevel"])
		item.EffectiveLevelFa = item.LevelFa
		if rec.Fields["veteranUpgrade"] == "yes" && item.UpgradedLevelFa != "" {
			item.EffectiveLevelFa = item.UpgradedLevelFa
		}
		view.Bases = append(view.Bases, item)
		if rec.Status == domain.BasisActive && view.ActiveBasis == nil {
			copy := item
			view.ActiveBasis = &copy
		}
	}
	// نوترین تصمیم بالا می‌آید؛ ترتیب نمایشی، نه قاعدهٔ کسب‌وکاری.
	sort.SliceStable(view.Bases, func(i, j int) bool {
		return view.Bases[i].EffectiveDate > view.Bases[j].EffectiveDate
	})

	view.Veteran = veteranOf(r, deps, id)
	view.Warnings = educationWarnings(view)
	view.Summary = map[string]any{
		"records":      len(view.Records),
		"bases":        len(view.Bases),
		"hasLatest":    view.Latest != nil,
		"hasActive":    view.ActiveBasis != nil,
		"verifiedOnly": countVerified(view.Records),
	}
	if view.ActiveBasis == nil {
		view.BasisNoteFa = "برای این کارمند «مدرک ملاک حقوق» فعالی ثبت نشده است؛ تا زمانی که ثبت نشود، " +
			"گروه شغلی و مبنای پرداخت از حکم جاری خوانده می‌شود."
	} else {
		view.BasisNoteFa = "مدرک ملاک حقوق این کارمند: " + view.ActiveBasis.EffectiveLevelFa +
			" " + view.ActiveBasis.Field + " — به استناد صورت‌جلسهٔ " + view.ActiveBasis.CouncilNo +
			" مورخ " + view.ActiveBasis.CouncilDate + "."
	}
	return view, nil
}

// basisMatchesRecord تطبیق تصمیم ملاک با یک سابقهٔ تحصیلی، وقتی کد سابقه خالی است.
func basisMatchesRecord(rec contracts.Record, bases []contracts.Record) bool {
	for _, base := range bases {
		if base.Status != domain.BasisActive {
			continue
		}
		if strings.TrimSpace(base.Fields["education"]) != "" {
			continue
		}
		if base.Fields["level"] == rec.Fields["level"] &&
			strings.TrimSpace(base.Fields["field"]) == strings.TrimSpace(rec.Fields["field"]) {
			return true
		}
	}
	return false
}

// listModuleRecords رکوردهای یک نوع از همین ماژول برای یک کارمند، به ترتیب مقطع.
//
// فیلتر کارمند در انبار یک قید است و اینجا دوباره سنجیده می‌شود تا سابقهٔ کارمند دیگر
// هرگز داخل پروندهٔ این کارمند نیاید.
func listModuleRecords(r *http.Request, svc *recordkit.Service, kind, employeeID string) ([]contracts.Record, error) {
	page, err := svc.List(r.Context(), contracts.RecordFilter{
		Module: svc.Module(), Kind: kind, EmployeeID: employeeID, Scope: "all", Size: 500,
	})
	if err != nil {
		return nil, err
	}
	records := make([]contracts.Record, 0, len(page.Items))
	for _, rec := range page.Items {
		if rec.EmployeeID != employeeID {
			continue
		}
		records = append(records, rec)
	}
	sort.SliceStable(records, func(i, j int) bool {
		return contracts.DegreeLevelOrder(records[i].Fields["level"]) <
			contracts.DegreeLevelOrder(records[j].Fields["level"])
	})
	return records, nil
}

// veteranOf وضعیت ایثارگری کارمند را از ماژول امور ایثارگران می‌خواند.
//
// از راه پورت قرارداد خوانده می‌شود (نه import بین ماژولی) و فقط وقتی پروندهٔ ایثارگری
// «تأییدشده» باشد، «ایثارگر» شمرده می‌شود؛ پروندهٔ در حال بررسی فقط نمایش داده می‌شود.
func veteranOf(r *http.Request, deps modulekit.Deps, employeeID string) veteranInfo {
	info := veteranInfo{}
	port, ok := deps.Ports.RecordList("grievances")
	if !ok {
		return info
	}
	page, err := port.List(r.Context(), contracts.RecordFilter{
		Module: "grievances", Kind: "veteranRecords", EmployeeID: employeeID, Scope: "all", Size: 50,
	})
	if err != nil {
		return info
	}
	for _, rec := range page.Items {
		if rec.EmployeeID != employeeID {
			continue
		}
		info.RecordID = rec.ID
		info.TypeFa = rec.Fields["veteranType"]
		info.Percent = rec.Fields["percent"]
		info.StatusFa = rec.StatusFa
		info.IsVerified = rec.Status == "verified"
		// «تأییدشده» قطعی و «بررسی‌شده» در انتظار تأیید نهایی است؛ هر دو یعنی پروندهٔ
		// ایثارگری این کارمند ثبت شده و معادل‌سازی «یک مقطع بالاتر» موضوعیت دارد.
		info.IsVeteran = rec.Status == "approved" || rec.Status == "verified"
		info.Confirmed = rec.Status == "approved"
		break
	}
	switch {
	case info.Confirmed:
		info.NoteFa = "پروندهٔ ایثارگری این کارمند تأیید نهایی شده است."
		info.UpgradeFa = "بر پایهٔ ضوابط ایثارگران، مدرک ملاک می‌تواند یک مقطع بالاتر معادل‌سازی شود."
	case info.IsVeteran:
		info.NoteFa = "پروندهٔ ایثارگری این کارمند بررسی شده و در انتظار تأیید نهایی است."
		info.UpgradeFa = "معادل‌سازی «یک مقطع بالاتر» پس از تأیید نهایی پروندهٔ ایثارگری اعمال می‌شود."
	default:
		info.NoteFa = "پروندهٔ ایثارگری برای این کارمند ثبت نشده است."
	}
	return info
}

// countVerified شمار سابقه‌های تأییدشدهٔ مدارک.
func countVerified(items []educationItem) int {
	count := 0
	for _, item := range items {
		if item.Status == domain.EducationVerified || item.Status == domain.EducationArchived {
			count++
		}
	}
	return count
}

// educationWarnings هشدارهای صادقانه دربارهٔ پروندهٔ تحصیلی.
func educationWarnings(view educationView) []string {
	warnings := []string{}
	if len(view.Records) == 0 {
		return append(warnings, "برای این کارمند هیچ سابقهٔ تحصیلی ثبت نشده است؛ مدرک تحصیلی باید در پرونده باشد.")
	}
	unverified := []string{}
	for _, item := range view.Records {
		if item.Status == domain.EducationDraft {
			unverified = append(unverified, item.LevelFa+" "+item.Field)
		}
	}
	if len(unverified) > 0 {
		warnings = append(warnings, "مدارک این مقاطع هنوز تأیید نشده است: "+strings.Join(unverified, "، ")+".")
	}
	if view.ActiveBasis == nil {
		warnings = append(warnings, "«مدرک ملاک حقوق» فعالی ثبت نشده است؛ تا تعیین آن، مبنای گروه شغلی روشن نیست.")
	} else {
		pending := 0
		for _, base := range view.Bases {
			if base.Status == domain.BasisDraft || base.Status == domain.BasisApproved {
				pending++
			}
		}
		if pending > 0 {
			warnings = append(warnings, "تصمیم ملاک حقوق این کارمند "+itoa(pending)+
				" رکورد در جریان دارد (پیش‌نویس یا تأییدشدهٔ شورا)؛ وضعیت آن را تعیین کنید.")
		}
		if view.ActiveBasis.Relevance == "not_related" {
			warnings = append(warnings, "مدرک ملاک جاری «غیرمرتبط» ارزیابی شده است؛ این مورد را با شورای تعدیل مدرک بازبینی کنید.")
		}
		if view.Veteran.IsVeteran && view.ActiveBasis.VeteranUpgrade != "yes" {
			warnings = append(warnings, "این کارمند ایثارگر است ولی معادل‌سازی «یک مقطع بالاتر» روی مدرک ملاک ثبت نشده است.")
		}
		if view.Veteran.IsVeteran && view.ActiveBasis.VeteranUpgrade == "yes" &&
			strings.TrimSpace(view.ActiveBasis.UpgradedLevel) == "" {
			warnings = append(warnings, "معادل‌سازی ایثارگری روی مدرک ملاک ثبت شده ولی «مقطع معادل‌سازی‌شده» خالی است.")
		}
	}
	if view.Latest == nil {
		warnings = append(warnings, "آخرین مدرک تحصیلی از سابقه‌ها قابل تشخیص نبود.")
	}
	return warnings
}

// itoa عدد را به رشتهٔ فارسی برمی‌گرداند (عدد داخل پیام فارسی کاربر).
func itoa(n int) string {
	if n <= 0 {
		return "۰"
	}
	digits := []rune("۰۱۲۳۴۵۶۷۸۹")
	out := []rune{}
	for n > 0 {
		out = append([]rune{digits[n%10]}, out...)
		n /= 10
	}
	return string(out)
}
