// برش‌های آماری کارکنان — جدول‌های توافقی (cross-tab)، توزیع‌ها و شاخص‌ها.
//
// چرا این فایل: کارفرما برای «سکشن آمار» برش‌های ترکیبی می‌خواهد — سن در برابر سابقه،
// تحصیلات در برابر واحد، نوع استخدام در برابر واحد، جنسیت در برابر نوع استخدام، واحد و
// تحصیلات. هر برش یک جدول دوبُعدی است که هم جدول می‌شود هم نمودار؛ پس به‌جای یک تابع
// برای هر ترکیب، یک موتور می‌نویسیم که «بُعد» را می‌شناسد و هر جفت بُعدی را جمع می‌زند.
//
// قاعدهٔ معماری: هیچ قاعدهٔ کسب‌وکاری اینجا ساخته نمی‌شود. بُعدها فقط دادهٔ پرونده را
// گروه می‌کنند (واحد، تحصیلات، نوع استخدام، جنسیت، وضعیت) و گروه سنی/سابقه از همان
// تاریخ‌های پرونده ساخته می‌شود. کارکنان از پورت قراردادی کارکنان و تحصیلات از پورت
// رکوردهای بایگانی خوانده می‌شوند؛ هیچ import مستقیمی بین ماژول‌ها نیست.
//
// قاعدهٔ ارقام: این لایه همهٔ اعداد را لاتین و «عدد» برمی‌گرداند (نه رشتهٔ آمادهٔ نمایش)؛
// فارسی‌سازی ارقام و ساختن متن جمله‌ها کار رابط است.

package statistics

import (
	"context"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"hrms/internal/contracts"
	"hrms/internal/platform/auth"
	"hrms/internal/platform/calendar"
	"hrms/internal/platform/httpx"
	"hrms/internal/platform/modulekit"
)

// registerCrosstabEndpoints مسیرهای «سکشن آمار» را ثبت می‌کند.
func registerCrosstabEndpoints(mux *http.ServeMux, deps modulekit.Deps) {
	read := func(fn http.HandlerFunc) http.HandlerFunc { return deps.Require("statistics", "read", fn) }

	// فهرست بُعدها و دسته‌هایشان: رابط وب فهرست‌های انتخابی را از همین‌جا می‌سازد.
	mux.HandleFunc("GET /api/v1/statistics/dimensions", read(func(w http.ResponseWriter, r *http.Request) {
		facts := loadFacts(r.Context(), deps, r)
		httpx.WriteJSON(w, http.StatusOK, map[string]any{
			"dimensions": allDimensions(facts),
			"presets":    sectionPresets(),
			"total":      len(facts),
			"scopeFa":    scopeFa(r),
		})
	}))

	// یک برش دلخواه: هر جفت بُعدی که کاربر از فهرست انتخاب کند.
	mux.HandleFunc("GET /api/v1/statistics/crosstab", read(func(w http.ResponseWriter, r *http.Request) {
		facts := loadFacts(r.Context(), deps, r)
		rowsDim, okRows := dimensionByKey(r.URL.Query().Get("rows"))
		colsDim, okCols := dimensionByKey(r.URL.Query().Get("cols"))
		if !okRows || !okCols {
			httpx.WriteErrorMsg(w, http.StatusBadRequest, contracts.CodeValidationFailed,
				"بُعد سطر یا ستون نامعتبر است.", map[string]string{
					"rows": "بُعد سطر باید از فهرست بُعدهای آماری باشد (مثل age یا department).",
					"cols": "بُعد ستون باید از فهرست بُعدهای آماری باشد (مثل education یا gender).",
				})
			return
		}
		matrix := buildMatrix(facts, rowsDim, colsDim, matrixTitle(rowsDim, colsDim))
		httpx.WriteJSON(w, http.StatusOK, map[string]any{
			"matrix": matrix, "total": len(facts), "scopeFa": scopeFa(r),
		})
	}))

	// کل سکشن: شاخص‌های کلیدی + توزیع‌های تک‌بُعدی + برش‌های آماده.
	mux.HandleFunc("GET /api/v1/statistics/section", read(func(w http.ResponseWriter, r *http.Request) {
		facts := loadFacts(r.Context(), deps, r)
		presets := sectionPresets()
		limit := 0
		if raw := strings.TrimSpace(r.URL.Query().Get("count")); raw != "" {
			if n, err := strconv.Atoi(raw); err == nil {
				limit = n
			}
		}
		matrices := make([]statMatrix, 0, len(presets))
		for _, preset := range presets {
			if limit > 0 && len(matrices) >= limit {
				break
			}
			rowsDim, okRows := dimensionByKey(preset.Rows)
			colsDim, okCols := dimensionByKey(preset.Cols)
			if !okRows || !okCols {
				continue
			}
			matrix := buildMatrix(facts, rowsDim, colsDim, preset.TitleFa)
			matrix.Key = preset.Key
			matrix.NoteFa = preset.NoteFa
			matrices = append(matrices, matrix)
		}
		httpx.WriteJSON(w, http.StatusOK, map[string]any{
			"metrics":       profileMetrics(facts),
			"dimensions":    allDimensions(facts),
			"distributions": buildDistributions(facts),
			"matrices":      matrices,
			"presets":       presets,
			"total":         len(facts),
			"scopeFa":       scopeFa(r),
			"noteFa":        "همهٔ اعداد از پروندهٔ کارکنان همین لحظه ساخته می‌شود: گروه سنی و سابقه از تاریخ تولد و تاریخ استخدام، تحصیلات از بالاترین مدرک تأییدشدهٔ بایگانی، و واحد/نوع استخدام/وضعیت از رکورد کارمند. هر برش هم جدول است هم نمودار و هم قابل خروجی‌گرفتن.",
		})
	}))
}

/* ----------------------------- مدل پاسخ ----------------------------- */

// statCategory یک دستهٔ آماری (یک گروه سنی، یک واحد سازمانی، یک مقطع تحصیلی...).
type statCategory struct {
	Key      string  `json:"key"`
	TitleFa  string  `json:"titleFa"`
	Count    int     `json:"count"`
	SharePct float64 `json:"sharePct"`
	// Order ترتیب نمایش؛ برای گروه‌های سنی و مقاطع تحصیلی معنادار است.
	Order int `json:"order"`
}

// statDimension یک بُعد آماری با دسته‌هایش.
type statDimension struct {
	Key        string         `json:"key"`
	TitleFa    string         `json:"titleFa"`
	Kind       string         `json:"kind"` // ordinal (ترتیبی) | nominal (اسمی)
	NoteFa     string         `json:"noteFa,omitempty"`
	Categories []statCategory `json:"categories"`
	Total      int            `json:"total"`
}

// statCell یک خانهٔ جدول توافقی با سه نوع درصد (سهم کل، درصد سطر، درصد ستون).
type statCell struct {
	RowKey   string  `json:"rowKey"`
	ColKey   string  `json:"colKey"`
	Count    int     `json:"count"`
	SharePct float64 `json:"sharePct"`
	RowPct   float64 `json:"rowPct"`
	ColPct   float64 `json:"colPct"`
}

// statMatrixRow یک سطر جدول توافقی.
type statMatrixRow struct {
	Key      string     `json:"key"`
	TitleFa  string     `json:"titleFa"`
	Count    int        `json:"count"`
	SharePct float64    `json:"sharePct"`
	Cells    []statCell `json:"cells"`
	// Series همان شمار هر خانه به ترتیب ستون‌ها — آمادهٔ نمودار میله‌ای انباشته.
	Series []int `json:"series"`
}

// statMatrix یک برش دوبُعدی کامل.
type statMatrix struct {
	Key       string          `json:"key"`
	TitleFa   string          `json:"titleFa"`
	NoteFa    string          `json:"noteFa,omitempty"`
	RowsDim   statDimension   `json:"rowsDim"`
	ColsDim   statDimension   `json:"colsDim"`
	Rows      []statMatrixRow `json:"rows"`
	Cols      []statCategory  `json:"cols"`
	Cells     [][]int         `json:"cells"`
	RowTotals []int           `json:"rowTotals"`
	ColTotals []int           `json:"colTotals"`
	Total     int             `json:"total"`
	MaxCell   int             `json:"maxCell"`
	// Mapped شمار کارکنانی که در این برش شمرده شده‌اند (برابر جمع همهٔ خانه‌ها).
	Mapped int `json:"mapped"`
	// Peak پرترین خانهٔ جدول؛ رابط از همین، جملهٔ «بیشترین تمرکز» را می‌سازد.
	Peak *statCell `json:"peak,omitempty"`
}

// statMetric یک شاخص کلیدی سکشن (میانگین سن، میانگین سابقه، سهم بانوان...).
//
// TextFa برای شاخص‌هایی است که مقدارشان «نام» است (پرجمعیت‌ترین واحد)؛ عدد جدا می‌آید تا
// فارسی‌سازی ارقام در رابط انجام شود.
type statMetric struct {
	Key     string  `json:"key"`
	TitleFa string  `json:"titleFa"`
	Value   float64 `json:"value"`
	TextFa  string  `json:"textFa,omitempty"`
	UnitFa  string  `json:"unitFa,omitempty"`
}

// sectionPreset یک برش آمادهٔ سکشن (ترکیب‌هایی که کارفرما خواسته است).
type sectionPreset struct {
	Key     string `json:"key"`
	Rows    string `json:"rows"`
	Cols    string `json:"cols"`
	TitleFa string `json:"titleFa"`
	NoteFa  string `json:"noteFa,omitempty"`
}

/* ----------------------------- دادهٔ پایه ----------------------------- */

// employeeFact یک کارمند در شکل آماری: مقدار هر بُعد آماده است.
type employeeFact struct {
	ID             string
	FullName       string
	Department     string
	Gender         string
	EmploymentType string
	Status         string
	ContractType   string
	Age            int
	HasAge         bool
	ServiceYears   int
	HasService     bool
	// DegreeLevel کلید بالاترین مدرک تأییدشدهٔ بایگانی؛ خالی یعنی ثبت‌نشده.
	DegreeLevel string
}

// loadFacts کارکنان را با محدودهٔ دسترسی نقش جاری می‌خواند و به شکل آماری تبدیل می‌کند.
//
// چرا از پورت: ماژول آمار مالک دادهٔ کارکنان نیست؛ فقط از فیلتر قراردادی همان پورت
// می‌خواند و محدودهٔ نقش را خودِ پورت اعمال می‌کند، پس مدیر واحد فقط کارکنان واحد خودش
// را در آمار می‌بیند.
func loadFacts(ctx context.Context, deps modulekit.Deps, r *http.Request) []employeeFact {
	reader, ok := deps.Ports.Employees()
	if !ok {
		return nil
	}
	filter := contracts.EmployeeFilter{Scope: "all", Size: 500}
	if user, ok := auth.UserFrom(r.Context()); ok {
		switch user.Role {
		case contracts.RoleEmployee:
			filter.Scope = "own"
			filter.ActorID = user.EmployeeID
		case contracts.RoleLineManager:
			filter.Scope = "own_unit"
			filter.ActorDepartment = user.DepartmentID
		}
	}
	var people []contracts.Employee
	for page := 1; page <= 20; page++ {
		filter.Page = page
		res, err := reader.List(ctx, filter)
		if err != nil {
			break
		}
		people = append(people, res.Items...)
		if len(res.Items) < filter.Size {
			break
		}
	}

	today, _ := calendar.FromGregorian(factsClock(deps))
	degrees := educationLevelIndex(ctx, deps)

	facts := make([]employeeFact, 0, len(people))
	for _, e := range people {
		fact := employeeFact{
			ID: e.ID, FullName: e.FullName, Department: strings.TrimSpace(e.DepartmentName),
			Gender: strings.TrimSpace(e.Gender), EmploymentType: strings.TrimSpace(e.EmploymentType),
			Status: strings.TrimSpace(e.Status), ContractType: strings.TrimSpace(e.ContractType),
			DegreeLevel: degrees[e.ID],
		}
		if fact.Department == "" {
			fact.Department = unknownKey
		}
		// سن: از تاریخ تولد شمسی تا امروز.
		if birth, err := calendar.Parse(strings.TrimSpace(e.BirthDate)); err == nil {
			if years, ok := wholeYearsBetween(birth, today); ok {
				fact.Age, fact.HasAge = years, true
			}
		}
		// سابقهٔ خدمت: از تاریخ استخدام تا امروز.
		if hire, err := calendar.Parse(strings.TrimSpace(e.HireDate)); err == nil {
			if years, ok := wholeYearsBetween(hire, today); ok {
				fact.ServiceYears, fact.HasService = years, true
			}
		}
		facts = append(facts, fact)
	}
	sort.SliceStable(facts, func(i, j int) bool { return facts[i].FullName < facts[j].FullName })
	return facts
}

// factsClock زمان مرجع آمار؛ برای آزمون‌پذیری از ساعت استقرار (در صورت وجود) می‌آید.
func factsClock(deps modulekit.Deps) time.Time {
	if deps.Clock != nil {
		return deps.Clock()
	}
	return time.Now()
}

// wholeYearsBetween سال کامل بین دو تاریخ شمسی (سن/سابقه). تاریخ بعد از مرجع → ok=false.
func wholeYearsBetween(from, to calendar.Date) (int, bool) {
	years := to.Year - from.Year
	if to.Month < from.Month || (to.Month == from.Month && to.Day < from.Day) {
		years--
	}
	if years < 0 {
		return 0, false
	}
	return years, true
}

// educationLevelIndex بالاترین مقطع تأییدشدهٔ هر کارمند را از رکوردهای بایگانی می‌سازد.
//
// چه مدرکی شمرده می‌شود: همان قاعدهٔ سند چاپی — فقط مدرک «تأییدشده» یا «بایگانیشده»؛
// مدرکی که شورای تعدیل مدرک تأییدش نکرده ملاک آمار نمی‌شود و در دستهٔ «ثبت‌نشده» می‌ماند
// تا پروندهٔ ناقص دیده شود.
func educationLevelIndex(ctx context.Context, deps modulekit.Deps) map[string]string {
	out := map[string]string{}
	port, ok := deps.Ports.RecordList("archive")
	if !ok {
		return out
	}
	orders := map[string]int{}
	for page := 1; page <= 20; page++ {
		res, err := port.List(ctx, contracts.RecordFilter{
			Module: "archive", Kind: "educations", Scope: "all", Page: page, Size: 500,
		})
		if err != nil {
			break
		}
		for _, rec := range res.Items {
			status := strings.TrimSpace(rec.Status)
			if status != "" && status != "verified" && status != "archived" {
				continue
			}
			level := strings.TrimSpace(rec.Fields["level"])
			order := contracts.DegreeLevelOrder(level)
			if order == 0 {
				continue
			}
			if order > orders[rec.EmployeeID] {
				orders[rec.EmployeeID] = order
				out[rec.EmployeeID] = level
			}
		}
		if len(res.Items) < 500 {
			break
		}
	}
	return out
}

/* ----------------------------- ابعاد آماری ----------------------------- */

const unknownKey = "unknown"

// dimSpec تعریف یک بُعد: چطور از کارمند مقدار بگیرد و دسته‌هایش چیست.
type dimSpec struct {
	Key     string
	TitleFa string
	Kind    string
	NoteFa  string
	// Value کلید دستهٔ یک کارمند (خالی یعنی «ثبت‌نشده»).
	Value func(f employeeFact) string
	// Fixed دسته‌های ثابت (ترتیبی یا فهرست قراردادی). خالی یعنی دسته‌ها از داده ساخته شود.
	Fixed []statCategory
}

// unknownCategory دستهٔ «ثبت‌نشده» — در بُعدهای ثابت هست تا دادهٔ ناقص پنهان نشود.
func unknownCategory(title string) statCategory {
	return statCategory{Key: unknownKey, TitleFa: title, Order: 999}
}

// dimensionSpecs فهرست بُعدهای آماری سامانه.
func dimensionSpecs() []dimSpec {
	return []dimSpec{
		{
			Key: "gender", TitleFa: "جنسیت", Kind: "nominal",
			NoteFa: "از پروندهٔ کارمند (مرد/زن)؛ خالی یعنی ثبت نشده.",
			Value:  func(f employeeFact) string { return f.Gender },
			Fixed: []statCategory{
				{Key: "male", TitleFa: "مرد", Order: 1},
				{Key: "female", TitleFa: "زن", Order: 2},
				unknownCategory("ثبت‌نشده"),
			},
		},
		{
			Key: "age", TitleFa: "گروه سنی", Kind: "ordinal",
			NoteFa: "از تاریخ تولد پروندهٔ کارمند تا امروز؛ گروه‌بندی در همین موتور آماری اعلام شده است.",
			Value: func(f employeeFact) string {
				if !f.HasAge {
					return unknownKey
				}
				switch {
				case f.Age < 30:
					return "lt30"
				case f.Age < 40:
					return "30_39"
				case f.Age < 50:
					return "40_49"
				default:
					return "ge50"
				}
			},
			Fixed: []statCategory{
				{Key: "lt30", TitleFa: "زیر ۳۰ سال", Order: 1},
				{Key: "30_39", TitleFa: "۳۰ تا ۳۹ سال", Order: 2},
				{Key: "40_49", TitleFa: "۴۰ تا ۴۹ سال", Order: 3},
				{Key: "ge50", TitleFa: "۵۰ سال و بیشتر", Order: 4},
				unknownCategory("تاریخ تولد ثبت‌نشده"),
			},
		},
		{
			Key: "service", TitleFa: "سابقهٔ خدمت", Kind: "ordinal",
			NoteFa: "از تاریخ استخدام پروندهٔ کارمند تا امروز.",
			Value: func(f employeeFact) string {
				if !f.HasService {
					return unknownKey
				}
				switch {
				case f.ServiceYears < 5:
					return "lt5"
				case f.ServiceYears < 10:
					return "5_9"
				case f.ServiceYears < 15:
					return "10_14"
				case f.ServiceYears < 20:
					return "15_19"
				default:
					return "ge20"
				}
			},
			Fixed: []statCategory{
				{Key: "lt5", TitleFa: "زیر ۵ سال", Order: 1},
				{Key: "5_9", TitleFa: "۵ تا ۹ سال", Order: 2},
				{Key: "10_14", TitleFa: "۱۰ تا ۱۴ سال", Order: 3},
				{Key: "15_19", TitleFa: "۱۵ تا ۱۹ سال", Order: 4},
				{Key: "ge20", TitleFa: "۲۰ سال و بیشتر", Order: 5},
				unknownCategory("تاریخ استخدام ثبت‌نشده"),
			},
		},
		{
			Key: "education", TitleFa: "تحصیلات", Kind: "ordinal",
			NoteFa: "بالاترین مدرک تأییدشدهٔ پروندهٔ بایگانی؛ مدرک در دست بررسی شورای تعدیل مدرک ملاک آمار نمی‌شود.",
			Value:  func(f employeeFact) string { return f.DegreeLevel },
			Fixed:  educationCategories(),
		},
		{
			Key: "department", TitleFa: "واحد سازمانی", Kind: "nominal",
			NoteFa: "واحد سازمانی همان پروندهٔ کارمند؛ دستهٔ خالی نمایش داده نمی‌شود.",
			Value:  func(f employeeFact) string { return f.Department },
		},
		{
			Key: "employment", TitleFa: "نوع استخدام", Kind: "nominal",
			NoteFa: "کلید کاتالوگ انواع استخدام سازمان.",
			Value:  func(f employeeFact) string { return f.EmploymentType },
			Fixed:  employmentCategories(),
		},
		{
			Key: "status", TitleFa: "وضعیت اشتغال", Kind: "nominal",
			NoteFa: "وضعیت کارمند در پرونده (شاغل، مرخصی، آزمایشی، پایان خدمت).",
			Value:  func(f employeeFact) string { return f.Status },
			Fixed:  statusCategories(),
		},
	}
}

// educationCategories دسته‌های تحصیلی از کاتالوگ مشترک مقاطع + «ثبت‌نشده».
func educationCategories() []statCategory {
	out := []statCategory{}
	for _, level := range contracts.DegreeLevels() {
		out = append(out, statCategory{Key: level.Key, TitleFa: level.TitleFa, Order: level.Order})
	}
	return append(out, unknownCategory("مدرک تأییدشده ثبت‌نشده"))
}

// employmentCategories دسته‌های نوع استخدام از کاتالوگ مشترک + «ثبت‌نشده».
func employmentCategories() []statCategory {
	out := []statCategory{}
	for i, t := range contracts.EmploymentTypes() {
		out = append(out, statCategory{Key: t.Key, TitleFa: t.TitleFa, Order: i + 1})
	}
	return append(out, unknownCategory("ثبت‌نشده"))
}

// statusCategories دسته‌های وضعیت از کاتالوگ مشترک + «ثبت‌نشده».
func statusCategories() []statCategory {
	out := []statCategory{}
	for i, s := range contracts.EmployeeStatuses() {
		out = append(out, statCategory{Key: s.Key, TitleFa: s.TitleFa, Order: i + 1})
	}
	return append(out, unknownCategory("ثبت‌نشده"))
}

// dimensionByKey یافتن بُعد با کلید.
func dimensionByKey(key string) (dimSpec, bool) {
	value := strings.TrimSpace(key)
	for _, spec := range dimensionSpecs() {
		if spec.Key == value {
			return spec, true
		}
	}
	return dimSpec{}, false
}

// allDimensions بُعدها با دسته‌هایشان روی دادهٔ فعلی (برای فهرست‌های انتخابی رابط).
func allDimensions(facts []employeeFact) []statDimension {
	out := make([]statDimension, 0, len(dimensionSpecs()))
	for _, spec := range dimensionSpecs() {
		out = append(out, dimWithCounts(spec, resolveCategories(spec, facts), len(facts)))
	}
	return out
}

// resolveCategories دسته‌های یک بُعد را با شمارش روی دادهٔ فعلی برمی‌گرداند.
//
// قاعده: بُعدهای ترتیبی (سن، سابقه، تحصیلات) همهٔ دسته‌ها را نگه می‌دارند تا گروه خالی هم
// دیده شود؛ بُعدهای اسمی دستهٔ بی‌عضو را حذف می‌کنند تا جدول شلوغ نشود.
func resolveCategories(spec dimSpec, facts []employeeFact) []statCategory {
	counts := map[string]int{}
	for _, f := range facts {
		key := strings.TrimSpace(spec.Value(f))
		if key == "" {
			key = unknownKey
		}
		counts[key]++
	}
	categories := []statCategory{}
	if len(spec.Fixed) > 0 {
		for _, cat := range spec.Fixed {
			cat.Count = counts[cat.Key]
			cat.SharePct = sharePct(float64(cat.Count), float64(len(facts)))
			if spec.Kind == "nominal" && cat.Count == 0 && cat.Key != unknownKey {
				continue
			}
			categories = append(categories, cat)
		}
		return categories
	}
	for key, count := range counts {
		title := key
		if key == unknownKey {
			title = "ثبت‌نشده"
		}
		categories = append(categories, statCategory{
			Key: key, TitleFa: title, Count: count,
			SharePct: sharePct(float64(count), float64(len(facts))),
		})
	}
	sort.SliceStable(categories, func(i, j int) bool {
		if categories[i].Count != categories[j].Count {
			return categories[i].Count > categories[j].Count
		}
		return categories[i].TitleFa < categories[j].TitleFa
	})
	for i := range categories {
		categories[i].Order = i + 1
	}
	return categories
}

/* ----------------------------- جدول توافقی ----------------------------- */

// buildMatrix برش دوبُعدی «سطرها × ستون‌ها» را روی دادهٔ فعلی می‌سازد.
func buildMatrix(facts []employeeFact, rowsSpec, colsSpec dimSpec, titleFa string) statMatrix {
	rowsCats := resolveCategories(rowsSpec, facts)
	colsCats := resolveCategories(colsSpec, facts)

	rowIndex := map[string]int{}
	for i, cat := range rowsCats {
		rowIndex[cat.Key] = i
	}
	colIndex := map[string]int{}
	for i, cat := range colsCats {
		colIndex[cat.Key] = i
	}

	cells := make([][]int, len(rowsCats))
	for i := range cells {
		cells[i] = make([]int, len(colsCats))
	}
	skipped := 0
	for _, f := range facts {
		rowKey := strings.TrimSpace(rowsSpec.Value(f))
		if rowKey == "" {
			rowKey = unknownKey
		}
		colKey := strings.TrimSpace(colsSpec.Value(f))
		if colKey == "" {
			colKey = unknownKey
		}
		r, okR := rowIndex[rowKey]
		c, okC := colIndex[colKey]
		if !okR || !okC {
			skipped++
			continue
		}
		cells[r][c]++
	}

	matrix := statMatrix{
		Key: rowsSpec.Key + "_" + colsSpec.Key, TitleFa: titleFa,
		RowsDim: dimWithCounts(rowsSpec, rowsCats, len(facts)),
		ColsDim: dimWithCounts(colsSpec, colsCats, len(facts)),
		Cols:    colsCats, Cells: cells, Total: len(facts),
	}
	rowTotals := make([]int, len(rowsCats))
	colTotals := make([]int, len(colsCats))
	maxCell := 0
	for r := range rowsCats {
		row := statMatrixRow{
			Key: rowsCats[r].Key, TitleFa: rowsCats[r].TitleFa,
			Series: make([]int, len(colsCats)),
		}
		rowTotal := 0
		for c := range colsCats {
			count := cells[r][c]
			rowTotal += count
			colTotals[c] += count
			row.Series[c] = count
			if count > maxCell {
				maxCell = count
			}
			row.Cells = append(row.Cells, statCell{
				RowKey: rowsCats[r].Key, ColKey: colsCats[c].Key, Count: count,
			})
		}
		row.Count = rowTotal
		rowTotals[r] = rowTotal
		row.SharePct = sharePct(float64(rowTotal), float64(len(facts)))
		matrix.Rows = append(matrix.Rows, row)
	}
	// درصدها پس از جمع کل سطر/ستون حساب می‌شوند (وابسته به هم‌اند).
	var peak *statCell
	mapped := len(facts) - skipped
	for r := range matrix.Rows {
		for c := range matrix.Rows[r].Cells {
			cell := &matrix.Rows[r].Cells[c]
			cell.SharePct = sharePct(float64(cell.Count), float64(len(facts)))
			cell.RowPct = sharePct(float64(cell.Count), float64(rowTotals[r]))
			cell.ColPct = sharePct(float64(cell.Count), float64(colTotals[c]))
			if peak == nil || cell.Count > peak.Count {
				snapshot := *cell
				peak = &snapshot
			}
		}
	}
	matrix.RowTotals = rowTotals
	matrix.ColTotals = colTotals
	matrix.MaxCell = maxCell
	matrix.Mapped = mapped
	matrix.Peak = peak
	return matrix
}

// dimWithCounts بُعد را با جمع کل دسته‌هایش برمی‌گرداند.
func dimWithCounts(spec dimSpec, categories []statCategory, total int) statDimension {
	return statDimension{
		Key: spec.Key, TitleFa: spec.TitleFa, Kind: spec.Kind, NoteFa: spec.NoteFa,
		Categories: categories, Total: total,
	}
}

// matrixTitle عنوان خودکار یک برش: «سطر در برابر ستون».
func matrixTitle(rowsSpec, colsSpec dimSpec) string {
	if rowsSpec.Key == colsSpec.Key {
		return rowsSpec.TitleFa
	}
	return rowsSpec.TitleFa + " در برابر " + colsSpec.TitleFa
}

/* ----------------------------- شاخص‌ها و توزیع‌ها ----------------------------- */

// buildDistributions توزیع تک‌بُعدی همهٔ بُعدها (مبنای نمودارهای دایره‌ای و میله‌ای).
func buildDistributions(facts []employeeFact) []statDimension {
	return allDimensions(facts)
}

// profileMetrics شاخص‌های کلیدی سکشن آماری.
func profileMetrics(facts []employeeFact) []statMetric {
	total := len(facts)
	ageSum, ageCount := 0, 0
	serviceSum, serviceCount := 0, 0
	female, male := 0, 0
	noBirth, noEducation := 0, 0
	byDepartment := map[string]int{}
	byEducation := map[string]int{}
	for _, f := range facts {
		if f.HasAge {
			ageSum += f.Age
			ageCount++
		} else {
			noBirth++
		}
		if f.HasService {
			serviceSum += f.ServiceYears
			serviceCount++
		}
		switch f.Gender {
		case "female":
			female++
		case "male":
			male++
		}
		if strings.TrimSpace(f.DegreeLevel) == "" {
			noEducation++
		} else {
			byEducation[f.DegreeLevel]++
		}
		byDepartment[f.Department]++
	}

	metrics := []statMetric{
		{Key: "total", TitleFa: "کارکنان پرونده", Value: float64(total), UnitFa: "نفر"},
		{Key: "avgAge", TitleFa: "میانگین سن", Value: average(ageSum, ageCount), UnitFa: "سال"},
		{Key: "avgService", TitleFa: "میانگین سابقهٔ خدمت", Value: average(serviceSum, serviceCount), UnitFa: "سال"},
		{Key: "femaleShare", TitleFa: "سهم بانوان از کارکنان", Value: sharePct(float64(female), float64(total)), UnitFa: "٪"},
		{Key: "male", TitleFa: "آقایان", Value: float64(male), UnitFa: "نفر"},
		{Key: "female", TitleFa: "بانوان", Value: float64(female), UnitFa: "نفر"},
	}
	if top, count := topEntry(byDepartment); top != "" {
		metrics = append(metrics, statMetric{
			Key: "topDepartment", TitleFa: "پرجمعیت‌ترین واحد سازمانی",
			Value: float64(count), TextFa: titleOfDepartment(top), UnitFa: "نفر",
		})
	}
	if top, count := topEntry(byEducation); top != "" {
		metrics = append(metrics, statMetric{
			Key: "topEducation", TitleFa: "پرتکرارترین مدرک تأییدشده",
			Value: float64(count), TextFa: contracts.DegreeLevelTitleFa(top), UnitFa: "نفر",
		})
	}
	// شاخص «پروندهٔ ناقص» عمدی است: پرونده‌ای که تاریخ تولد یا مدرک تأییدشده ندارد باید
	// دیده شود، وگرنه آمار سازمان بیش از واقع کامل به‌نظر می‌رسد.
	metrics = append(metrics,
		statMetric{Key: "missingBirthDate", TitleFa: "بدون تاریخ تولد", Value: float64(noBirth), UnitFa: "نفر"},
		statMetric{Key: "missingEducation", TitleFa: "بدون مدرک تأییدشده", Value: float64(noEducation), UnitFa: "نفر"},
	)
	return metrics
}

// titleOfDepartment عنوان نمایشی واحد (کلید نامعلوم → «ثبت‌نشده»).
func titleOfDepartment(key string) string {
	if key == unknownKey {
		return "ثبت‌نشده"
	}
	return key
}

// average میانگین با یک رقم اعشار (صفر اگر داده‌ای نباشد).
func average(sum, count int) float64 {
	if count == 0 {
		return 0
	}
	return math.Round(float64(sum)/float64(count)*10) / 10
}

// topEntry پرتکرارترین کلید نگاشت را برمی‌گرداند.
func topEntry(counts map[string]int) (string, int) {
	bestKey, bestCount := "", 0
	for key, count := range counts {
		if count > bestCount || (count == bestCount && key < bestKey) {
			bestKey, bestCount = key, count
		}
	}
	return bestKey, bestCount
}

// sectionPresets برش‌های آمادهٔ سکشن — همان ترکیب‌هایی که کارفرما خواسته است، به ترتیبی
// که از ساده به ترکیبی می‌روند.
func sectionPresets() []sectionPreset {
	return []sectionPreset{
		{Key: "age_service", Rows: "age", Cols: "service", TitleFa: "گروه سنی × سابقهٔ خدمت",
			NoteFa: "تصویر تجربهٔ سازمان: هر گروه سنی چند سال سابقه دارد."},
		{Key: "education_department", Rows: "education", Cols: "department", TitleFa: "تحصیلات × واحد سازمانی",
			NoteFa: "ترکیب دانش و واحد؛ مبنای جانشین‌پروری و برنامهٔ آموزش."},
		{Key: "employment_department", Rows: "employment", Cols: "department", TitleFa: "نوع استخدام × واحد سازمانی",
			NoteFa: "پایهٔ گزارش‌های بیمه و برنامهٔ نیروی انسانی."},
		{Key: "gender_employment", Rows: "gender", Cols: "employment", TitleFa: "جنسیت × نوع استخدام",
			NoteFa: "تنوع جنسیتی در هر شکل به‌کارگیری."},
		{Key: "gender_department", Rows: "gender", Cols: "department", TitleFa: "جنسیت × واحد سازمانی",
			NoteFa: "توزیع جنسیتی واحدها؛ مبنای سیاست‌های عدالت سازمانی."},
		{Key: "gender_education", Rows: "gender", Cols: "education", TitleFa: "جنسیت × تحصیلات",
			NoteFa: "سطح سواد هر جنسیت؛ مبنای برنامهٔ ارتقای مدرک."},
		{Key: "age_gender", Rows: "age", Cols: "gender", TitleFa: "گروه سنی × جنسیت",
			NoteFa: "هرم سنی–جنسیتی سازمان."},
		{Key: "service_department", Rows: "service", Cols: "department", TitleFa: "سابقهٔ خدمت × واحد سازمانی",
			NoteFa: "ریسک خروج تجربه از هر واحد."},
	}
}

// scopeFa توضیح محدودهٔ دادهٔ کاربر جاری (برای شفاف‌بودن آمار).
func scopeFa(r *http.Request) string {
	user, ok := auth.UserFrom(r.Context())
	if !ok {
		return "همهٔ کارکنان"
	}
	switch user.Role {
	case contracts.RoleEmployee:
		return "فقط پروندهٔ خودِ کاربر"
	case contracts.RoleLineManager:
		return "فقط کارکنان واحد کاربر"
	}
	return "همهٔ کارکنان"
}
