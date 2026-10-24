// پیاده‌سازی ماژول «reports» — «holding_source».

package reports

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"hrms/internal/contracts"
	"hrms/internal/platform/auth"
	"hrms/internal/platform/modulekit"
	"hrms/internal/platform/orgs"
	"hrms/internal/platform/repfilter"
)

// holdingHrSource منبع دادهٔ گزارش «شاخص‌های منابع انسانی شرکت‌های هلدینگ».
//
// این منبع برای هر شرکت زیرمجموعهٔ هلدینگ یک سطر می‌سازد و شاخص‌های سازمانی همان شرکت را
// از پروفایل پارامتریک آن می‌خواند (تعداد کل کارکنان، ادارات/ماژول‌های فعال و پارامترهای
// کلیدی منابع انسانی مثل مرخصی، ساعت موظفی و سن بازنشستگی). به این ترتیب نمای هلدینگ
// می‌تواند شرکت‌های زیرمجموعه را کنار هم ببیند و مقایسه کند.
//
// محدودیت صریح: شاخص‌هایی که به رکورد کارکنان نیاز دارند (تعداد فعال، میانگین حقوق پایه)
// فقط برای شرکتِ فعالِ همین استقرار محاسبه می‌شوند؛ چون هر فرآیند یک شرکت را اجرا می‌کند و
// نگهدارندهٔ دادهٔ کارکنان شرکت‌های دیگر نیست. این محدودیت در NoteFa گزارش هم می‌آید تا
// کاربر «—» را با «صفر» اشتباه نگیرد.
type holdingHrSource struct{ deps modulekit.Deps }

const (
	holdingHrKey  = "holdingHr"
	holdingAllKey = "__all__"
)

// holdingHrFields اقلام قابل‌انتخاب گزارش هلدینگ.
func holdingHrFields() []contracts.ReportField {
	return []contracts.ReportField{
		{Key: "company", LabelFa: "شرکت", Type: "text"},
		{Key: "holding", LabelFa: "هلدینگ", Type: "text"},
		{Key: "headcount", LabelFa: "تعداد کل کارکنان (سازمانی)", Type: "number"},
		{Key: "activeEmployees", LabelFa: "کارکنان فعال", Type: "number"},
		{Key: "avgSalary", LabelFa: "میانگین حقوق پایه", Type: "money"},
		{Key: "payrollMass", LabelFa: "جمع حقوق پایه ماهانه", Type: "money"},
		{Key: "departments", LabelFa: "ادارات فعال", Type: "number"},
		{Key: "modules", LabelFa: "ماژول‌های فعال", Type: "number"},
		{Key: "annualLeave", LabelFa: "مرخصی استحقاقی سالانه (روز)", Type: "number"},
		{Key: "sickLeave", LabelFa: "مرخصی استعلاجی سالانه (روز)", Type: "number"},
		{Key: "monthlyHours", LabelFa: "ساعت کار موظفی ماهانه", Type: "number"},
		{Key: "overtimeFactor", LabelFa: "ضریب اضافه‌کاری", Type: "number"},
		{Key: "retireAgeMen", LabelFa: "سن بازنشستگی مردان", Type: "number"},
		{Key: "retireAgeWomen", LabelFa: "سن بازنشستگی زنان", Type: "number"},
	}
}

// holdingHrFilters فیلترهای اعلامی منبع: انتخاب هلدینگ و جست‌وجوی نام شرکت. فهرست هلدینگ‌ها
// از پروفایل استقرار ساخته می‌شود تا هیچ گزینه‌ای در UI هاردکد نشود.
func holdingHrFilters() []contracts.ReportFilterDef {
	opts := []contracts.ReportFilterOption{{Value: holdingAllKey, LabelFa: "همهٔ هلدینگ‌ها"}}
	for _, h := range orgs.Holdings() {
		opts = append(opts, contracts.ReportFilterOption{Value: h.Key, LabelFa: h.TitleFa})
	}
	return []contracts.ReportFilterDef{
		{Key: "holding", LabelFa: "هلدینگ", Type: "select", Options: opts},
		{Key: "company", LabelFa: "نام شرکت (شامل)", Type: "text"},
	}
}

// Metadata معرفی منبع هلدینگ.
func (s holdingHrSource) Metadata() contracts.ReportSourceMeta {
	return contracts.ReportSourceMeta{
		Key: holdingHrKey, TitleFa: "شاخص‌های منابع انسانی شرکت‌های هلدینگ",
		Fields: holdingHrFields(), Filters: holdingHrFilters(),
	}
}

// companyMetrics شاخص‌های مبتنی بر رکورد کارکنان برای یک شرکت.
type companyMetrics struct {
	employees   int
	active      int
	avgSalary   float64
	payrollMass float64
	ok          bool
}

// Rows سطرهای گزارش هلدینگ را می‌سازد: یک سطر برای هر شرکت در محدودهٔ دسترسی/فیلتر.
func (s holdingHrSource) Rows(ctx context.Context, req contracts.ReportRequest) (contracts.ReportResult, error) {
	fields := holdingHrFields()
	byKey := make(map[string]contracts.ReportField, len(fields))
	for _, f := range fields {
		byKey[f.Key] = f
	}
	cols := selectReportColumns(fields, req.Items, byKey)

	companies := s.companiesFor(ctx, req.Filters["holding"])
	if len(companies) == 0 {
		return contracts.ReportResult{}, fmt.Errorf("شرکتی برای این محدوده یافت نشد")
	}
	metrics := s.activeMetrics(ctx)

	nameFilter := strings.TrimSpace(req.Filters["company"])
	rows := make([][]string, 0, len(companies))
	var totalHead, totalActive, companyCount int
	var totalMass float64
	for _, o := range companies {
		m := metrics[o.ID]
		values := map[string]string{
			"company":         o.TitleFa,
			"holding":         o.HoldingFa,
			"headcount":       strconv.Itoa(o.Headcount),
			"departments":     strconv.Itoa(len(o.EnabledDepartments)),
			"modules":         strconv.Itoa(len(o.EnabledModules)),
			"annualLeave":     companyParam(o, "leave.annualDays"),
			"sickLeave":       companyParam(o, "leave.sickDays"),
			"monthlyHours":    companyParam(o, "payroll.monthlyHours"),
			"overtimeFactor":  companyParam(o, "payroll.overtimeFactor"),
			"retireAgeMen":    companyParam(o, "retirement.ageMen"),
			"retireAgeWomen":  companyParam(o, "retirement.ageWomen"),
			"activeEmployees": "",
			"avgSalary":       "",
			"payrollMass":     "",
		}
		if m.ok {
			values["activeEmployees"] = strconv.Itoa(m.active)
			values["avgSalary"] = strconv.FormatFloat(m.avgSalary, 'f', 0, 64)
			values["payrollMass"] = strconv.FormatFloat(m.payrollMass, 'f', 0, 64)
		}
		if nameFilter != "" && !strings.Contains(o.TitleFa, nameFilter) && !strings.Contains(o.ID, nameFilter) {
			continue
		}
		if !repfilter.Matches(values, req.Conditions) {
			continue
		}
		row := make([]string, 0, len(cols))
		for _, c := range cols {
			row = append(row, values[c.Key])
		}
		rows = append(rows, row)
		totalHead += o.Headcount
		totalActive += m.active
		totalMass += m.payrollMass
		companyCount++
	}

	summary := []contracts.ReportSummary{
		{Key: "headcount", LabelFa: "جمع کل کارکنان (سازمانی)", Total: float64(totalHead)},
	}
	if totalActive > 0 {
		summary = append(summary, contracts.ReportSummary{Key: "activeEmployees", LabelFa: "جمع کارکنان فعال (شرکت فعال)", Total: float64(totalActive)})
	}
	if totalMass > 0 {
		summary = append(summary, contracts.ReportSummary{Key: "payrollMass", LabelFa: "جمع حقوق پایه ماهانه (شرکت فعال)", Total: totalMass})
	}

	note := fmt.Sprintf("شاخص‌های سازمانی هر شرکت از پروفایل پارامتریک همان شرکت خوانده شده است (%d شرکت در این گزارش). شاخص‌های مبتنی بر رکورد کارکنان — «کارکنان فعال»، «میانگین حقوق پایه» و «جمع حقوق پایه» — فقط برای شرکت فعال در این استقرار («%s») محاسبه می‌شود؛ برای سایر شرکت‌ها این اقلام خالی نمایش داده می‌شوند.", companyCount, s.deps.Org.TitleFa)

	return contracts.ReportResult{
		Source:  holdingHrKey,
		TitleFa: "داشبورد منابع انسانی هلدینگ",
		Columns: cols,
		Rows:    rows,
		Total:   len(rows),
		NoteFa:  note,
		Summary: summary,
	}, nil
}

// companiesFor محدودهٔ شرکت‌ها را تعیین می‌کند. کاربر غیرمدیر همیشه فقط شرکت‌های هلدینگ
// خودش را می‌بیند (مستقل از فیلتر درخواست)؛ مدیر می‌تواند یک هلدینگ یا همهٔ هلدینگ‌ها را
// انتخاب کند. پیش‌فرض، هلدینگ شرکت فعال است.
func (s holdingHrSource) companiesFor(ctx context.Context, key string) []orgs.Organization {
	key = strings.TrimSpace(key)
	own := orgs.HoldingKey(s.deps.Org)
	if user, ok := auth.UserFrom(ctx); ok && user.Role != contracts.RoleAdmin {
		// محدودهٔ دسترسی بر فیلتر مقدم است؛ هیچ کاربری از هلدینگ خودش فراتر نمی‌رود.
		return orgs.CompaniesOfHolding(own)
	}
	switch {
	case key == holdingAllKey:
		return orgs.All
	case key == "":
		return orgs.CompaniesOfHolding(own)
	default:
		return orgs.CompaniesOfHolding(key)
	}
}

// activeMetrics شاخص‌های رکوردی شرکت فعال را از پورت کارکنان می‌خواند.
func (s holdingHrSource) activeMetrics(ctx context.Context) map[string]companyMetrics {
	out := map[string]companyMetrics{}
	if s.deps.Ports == nil {
		return out
	}
	reader, ok := s.deps.Ports.Employees()
	if !ok {
		return out
	}
	page, err := reader.List(ctx, contracts.EmployeeFilter{Scope: "all", Size: 5000})
	if err != nil {
		return out
	}
	var sum int64
	active := 0
	for _, e := range page.Items {
		sum += e.BaseSalary
		if e.Status == "active" {
			active++
		}
	}
	n := len(page.Items)
	avg := 0.0
	if n > 0 {
		avg = float64(sum) / float64(n)
	}
	out[s.deps.Org.ID] = companyMetrics{employees: n, active: active, avgSalary: avg, payrollMass: float64(sum), ok: true}
	return out
}

// companyParam مقدار یک پارامتر شرکت را به‌صورت رشته برمی‌گرداند (خالی اگر نباشد).
func companyParam(o orgs.Organization, key string) string {
	if v, ok := o.Param(key); ok {
		return strings.TrimSpace(v)
	}
	return ""
}

// selectReportColumns اقلام انتخابی را به ترتیب درخواست برمی‌گرداند؛ اگر انتخابی نبود یا
// هیچ قلم معتبری نداشت، همهٔ اقلام منبع برمی‌گردد.
func selectReportColumns(all []contracts.ReportField, items []string, byKey map[string]contracts.ReportField) []contracts.ReportField {
	if len(items) == 0 {
		return all
	}
	out := make([]contracts.ReportField, 0, len(items))
	for _, key := range items {
		if f, ok := byKey[key]; ok {
			out = append(out, f)
		}
	}
	if len(out) == 0 {
		return all
	}
	return out
}
