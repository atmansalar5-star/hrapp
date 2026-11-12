// راهنمای «شروع به‌روزرسانی داده» برای استقرار نظام جبران خدمات ۴پی.
//
// چرا این مسیر در ماژول است و نه یک سند جدا: ترتیب تکمیل داده یک قاعدهٔ اجرایی است و
// باید همراه همان داده‌ای باشد که می‌سنجد. هر گام می‌گوید «کدام جدول/صفحه»، «چه داده‌ای
// حیاتی است»، «چرا اولویت دارد» و «اکنون چند درصدش پر است». شمارش‌ها از خودِ ماژول‌های
// مالک داده (از راه پورت قرارداد) خوانده می‌شود، پس هیچ عددی در رابط ساخته نمی‌شود.
package compensation

import (
	"context"
	"net/http"
	"strings"

	"hrms/internal/contracts"
	"hrms/internal/modules/compensation/domain"
	"hrms/internal/platform/httpx"
	"hrms/internal/platform/modulekit"
	"hrms/internal/platform/recordkit"
	"hrms/internal/rules"
)

// readinessStep یک گام از راهنمای تکمیل داده.
type readinessStep struct {
	Order int `json:"order"`
	// TitleFa عنوان گام؛ TableFa نام جدول/فهرست داده‌ای که باید پر شود.
	TitleFa string `json:"titleFa"`
	TableFa string `json:"tableFa"`
	// ModuleFa ماژول مالک داده و PageKey کلید صفحه‌ای که کاربر باید در آن کار کند.
	ModuleFa string `json:"moduleFa"`
	PageKey  string `json:"pageKey,omitempty"`
	// CriticalFa داده‌های حیاتی این گام (برچسب فارسی فیلدهای الزامی همان نوع رکورد).
	CriticalFa []string `json:"criticalFa,omitempty"`
	// WhyFa دلیل اولویت و وابستگی به گام‌های بعد.
	WhyFa string `json:"whyFa"`
	// Status یکی از ok|partial|empty و StatusFa عنوان فارسی آن.
	Status   string `json:"status"`
	StatusFa string `json:"statusFa"`
	// CountFa گزارش شمارشی گام (مثل «۲۷ پست مصوب از ۳۱ پست»).
	CountFa string `json:"countFa,omitempty"`
	// Done نشان می‌دهد گام کامل شده است (شرط عبور به گام بعدی).
	Done bool `json:"done"`
}

// registerReadinessEndpoints مسیر راهنمای شروع به‌روزرسانی داده را ثبت می‌کند.
func (m *Module) registerReadinessEndpoints(mux *http.ServeMux, deps modulekit.Deps, svc *recordkit.Service) {
	read := func(fn http.HandlerFunc) http.HandlerFunc { return deps.Require("compensation", "read", fn) }
	mux.HandleFunc("GET /api/v1/compensation/data-readiness", read(func(w http.ResponseWriter, r *http.Request) {
		steps := readinessSteps(r.Context(), deps, svc)
		done := 0
		for _, step := range steps {
			if step.Done {
				done++
			}
		}
		httpx.WriteJSON(w, http.StatusOK, map[string]any{
			"steps":       steps,
			"doneCount":   done,
			"totalCount":  len(steps),
			"progressFa":  rules.IntFa(done) + " از " + rules.IntFa(len(steps)) + " گام",
			"instruction": rules.CompensationInstructionFa(),
			// مرز مدل‌ها در همان صفحه‌ای که استقرار از آن شروع می‌شود اعلام می‌شود.
			"modelStatusFa":      modelStatusFa(compensationParams(deps).PayrollApply),
			"independenceNoteFa": rules.CompensationIndependenceFa(compensationParams(deps).PayrollApply),
			"criticalNoteFa": "دادهٔ حیاتی هر گام، همان فیلدهای الزامی اعلام‌شده در دامنهٔ سرور است؛ " +
				"رکوردی که این فیلدها را نداشته باشد ذخیره نمی‌شود، پس درصد تکمیل با همان قید سنجیده می‌شود.",
			"orderNoteFa": "ترتیب گام‌ها عمدی است: هر گام ورودی گام بعد را می‌سازد؛ " +
				"مثلاً باند پرداخت از مدل و پارامترهای مصوب ساخته می‌شود و تصمیم چهارعاملی بدون قیمت‌گذاری شغل بی‌مبنا می‌ماند.",
		})
	}))
}

// readinessSteps گام‌های تکمیل داده را به ترتیب اولویت می‌سازد و وضعیت هر گام را از
// دادهٔ واقعی می‌سنجد.
func readinessSteps(ctx context.Context, deps modulekit.Deps, svc *recordkit.Service) []readinessStep {
	params := compensationParams(deps)

	employeeTotal := 0
	if reader, ok := deps.Ports.Employees(); ok {
		if page, err := reader.List(ctx, contracts.EmployeeFilter{Scope: "all", Size: 1}); err == nil {
			employeeTotal = page.Total
		}
	}
	positionTotal := recordCount(ctx, deps, orgStructureModule, positionKind, "")
	descriptionTotal := recordCount(ctx, deps, jobClassModule, jobDescriptionKind, "")
	categoryTotal := recordCount(ctx, deps, jobClassModule, jobCategoryKind, "")

	models := listOwn(ctx, svc, domain.KindCompensationModel)
	activeModels := 0
	for _, rec := range models {
		if rec.Status == "active" {
			activeModels++
		}
	}
	pricings := listOwn(ctx, svc, domain.KindJobPricing)
	approvedPricings := countStatus(pricings, "approved")
	bands := listOwn(ctx, svc, domain.KindPayBand)
	activeBands := countStatus(bands, "active")
	competencies := listOwn(ctx, svc, domain.KindCompetencyLevel)
	confirmedCompetencies := countStatus(competencies, "confirmed")
	competencyEmployees := distinctEmployees(competencies)
	performanceRecords := listOwn(ctx, svc, domain.KindPerformanceFactor)
	approvedPerformance := countStatus(performanceRecords, "approved")
	decisions := listOwn(ctx, svc, domain.KindCompensationDecision)
	finalResults := recordCountStatus(ctx, deps, performanceModuleID, perfResultKind, "final")
	decreeSample, decreeSampleHit := decreeSampleCoverage(ctx, deps, svc)

	steps := []readinessStep{
		{
			Order: 1, TitleFa: "مدل و پارامترهای مصوب جبران خدمات", TableFa: "رکورد مدل جبران خدمات + پارامترهای شرکت",
			ModuleFa: "جبران خدمات ۴پی", PageKey: "compModel",
			CriticalFa: []string{"نام مدل", "دامنهٔ اجرا", "تاریخ اجرا", "میانهٔ باند پلهٔ ۱",
				"شیب و گسترهٔ باند", "امتیاز ارزشیابی هر پله", "کف و سقف ضریب شایستگی",
				"ضریب عملکرد هر سطح", "سقف افزایش سالانه"},
			WhyFa:   "همهٔ باندها و ضریب‌های نظام از همین مدل و پارامترها ساخته می‌شود؛ تا این گام مصوب نشود بقیهٔ محاسبه‌ها مبنای مصوب ندارند. این مدل مستقل و اختیاری است و تا پارامتر «اعمال در پرداخت» روشن نشود در حکم و فیش اثری ندارد.",
			Status:  readinessStatus(activeModels > 0, activeModels > 0, activeModels > 0),
			CountFa: countFa(rules.IntFa(activeModels)+" مدل جاری", rules.IntFa(len(models))+" مدل ثبت‌شده"),
			Done:    activeModels > 0,
		},
		{
			Order: 2, TitleFa: "ساختار سازمانی و پست‌های مصوب", TableFa: "واحدهای سازمانی و شناسنامهٔ پست‌ها",
			ModuleFa: "ساختار سازمانی", PageKey: "positions",
			CriticalFa: criticalLabels(ctx, deps, orgStructureModule, positionKind),
			WhyFa:      "قیمت‌گذاری شغل روی پست انجام می‌شود؛ بدون پست مصوب، امتیاز شغل به کسی و پستی وصل نمی‌شود.",
			Status:     readinessStatus(positionTotal > 0, positionTotal > 0, positionTotal > 0),
			CountFa:    rules.IntFa(positionTotal) + " پست مصوب",
			Done:       positionTotal > 0,
		},
		{
			Order: 3, TitleFa: "طبقه‌بندی مشاغل، شرح شغل و ارزشیابی شغل", TableFa: "گروه‌های شغلی، شرح شغل و ارزشیابی شغل",
			ModuleFa: "طبقه‌بندی مشاغل", PageKey: "jobDescriptions",
			CriticalFa: criticalLabels(ctx, deps, jobClassModule, jobDescriptionKind),
			WhyFa:      "امتیاز ارزشیابی شغل (P1) از همین ماژول می‌آید؛ شرح شغل هم مرجع شرایط احراز و پیوند با نظام پرداخت است.",
			Status:     readinessStatus(descriptionTotal > 0 && categoryTotal > 0, descriptionTotal > 0, descriptionTotal+categoryTotal > 0),
			CountFa:    rules.IntFa(categoryTotal) + " گروه شغلی، " + rules.IntFa(descriptionTotal) + " شرح شغل",
			Done:       descriptionTotal > 0 && categoryTotal > 0,
		},
		{
			Order: 4, TitleFa: "پروندهٔ کارکنان و سوابق", TableFa: "پروندهٔ پرسنلی کارکنان",
			ModuleFa: "امور کارکنان", PageKey: "employees",
			CriticalFa: []string{"کد پرسنلی", "نام و نام خانوادگی", "واحد سازمانی", "پست/سمت جاری",
				"تاریخ استخدام", "مقطع و رشتهٔ تحصیلی"},
			WhyFa:   "سطح شایستگی، سابقه و تصمیم پرداخت به کارمند وصل می‌شود؛ کارمندی که پروندهٔ کامل ندارد در بازنگری از قلم می‌افتد.",
			Status:  readinessStatus(employeeTotal > 0, employeeTotal > 0, employeeTotal > 0),
			CountFa: rules.IntFa(employeeTotal) + " کارمند فعال در پرونده",
			Done:    employeeTotal > 0,
		},
		{
			Order: 5, TitleFa: "قیمت‌گذاری شغل (P1 — شغل)", TableFa: "قیمت‌گذاری شغل",
			ModuleFa: "جبران خدمات ۴پی", PageKey: "compJobPricing",
			CriticalFa: []string{"کد پست", "عنوان پست", "خانوادهٔ شغلی", "امتیاز ارزشیابی مصوب شغل",
				"پلهٔ گروه شغلی", "تاریخ قیمت‌گذاری"},
			WhyFa:   "پلهٔ گروه شغلی از امتیاز شغل ساخته می‌شود و باند پرداخت را تعیین می‌کند؛ بدون آن، محاسبهٔ ۴پی «پلهٔ ۱» می‌گیرد.",
			Status:  readinessStatus(positionTotal > 0 && approvedPricings >= positionTotal, approvedPricings > 0, len(pricings) > 0),
			CountFa: rules.IntFa(approvedPricings) + " قیمت‌گذاری مصوب از " + rules.IntFa(positionTotal) + " پست",
			Done:    positionTotal > 0 && approvedPricings >= positionTotal,
		},
		{
			Order: 6, TitleFa: "سطح شایستگی فرد (P2 — شخص)", TableFa: "سطح شایستگی کارکنان",
			ModuleFa: "جبران خدمات ۴پی", PageKey: "compCompetency",
			CriticalFa: []string{"مدل/خانوادهٔ شغلی شایستگی", "سطح شایستگی", "تاریخ تعیین سطح", "کارمند"},
			WhyFa:      "ضریب شخص از سطح شایستگی می‌آید؛ کارمند بدون سطح تأییدشده، ضریب حداقلی می‌گیرد.",
			Status:     readinessStatus(employeeTotal > 0 && competencyEmployees >= employeeTotal, competencyEmployees > 0, len(competencies) > 0),
			CountFa:    rules.IntFa(len(competencies)) + " سطح ثبت‌شده (" + rules.IntFa(confirmedCompetencies) + " تأییدشده) برای " + rules.IntFa(competencyEmployees) + " کارمند از " + rules.IntFa(employeeTotal),
			Done:       employeeTotal > 0 && competencyEmployees >= employeeTotal,
		},
		{
			Order: 7, TitleFa: "نتیجهٔ ارزیابی عملکرد (P3 — عملکرد)", TableFa: "نتایج ارزیابی عملکرد + ضریب عملکرد دوره",
			ModuleFa: "مدیریت عملکرد", PageKey: "perfResults",
			CriticalFa: append(criticalLabels(ctx, deps, performanceModuleID, perfResultKind),
				"امتیاز نهایی عملکرد", "سطح عملکرد", "ضریب عملکرد دوره"),
			WhyFa:   "ضریب عملکرد از نتیجهٔ نهایی ارزیابی می‌آید؛ دورهٔ بسته‌نشده یعنی ضریب نامعلوم و افزایش عملکردی بی‌مبنا.",
			Status:  readinessStatus(finalResults >= employeeTotal && employeeTotal > 0, finalResults > 0, len(performanceRecords) > 0),
			CountFa: rules.IntFa(finalResults) + " نتیجهٔ نهایی ارزیابی، " + rules.IntFa(approvedPerformance) + " ضریب عملکرد تأییدشده در این ماژول",
			Done:    employeeTotal > 0 && finalResults >= employeeTotal,
		},
		{
			Order: 8, TitleFa: "باند و پله‌های پرداخت (P4 — پرداخت)", TableFa: "باند پرداخت هر گروه شغلی",
			ModuleFa: "جبران خدمات ۴پی", PageKey: "compPayBands",
			CriticalFa: []string{"پلهٔ گروه شغلی", "عنوان گروه شغلی", "کف باند", "میانهٔ باند",
				"سقف باند", "مبلغ گام پله", "تاریخ اجرا"},
			WhyFa:  "پرداخت هدف روی میانهٔ باند ساخته می‌شود و سقف باند مانع پرداخت بیرون از چهارچوب است.",
			Status: readinessStatus(activeBands >= len(rules.Grades()), activeBands > 0, len(bands) > 0),
			CountFa: rules.IntFa(activeBands) + " باند جاری از " + rules.IntFa(len(rules.Grades())) +
				" گروه شغلی (میانهٔ باند پایهٔ مصوب: " + rules.IntMoneyFa(params.BandBaseMidpoint) + ")",
			Done: activeBands >= len(rules.Grades()),
		},
		{
			Order: 9, TitleFa: "پرداخت فعلی از احکام حقوق", TableFa: "احکام حقوق ابلاغ‌شده",
			ModuleFa: "احکام حقوق و گواهی اشتغال", PageKey: "salaryDecrees",
			CriticalFa: []string{"کارمند", "پایهٔ حقوق ماهانه", "گروه شغلی", "تاریخ اجرا", "علت صدور"},
			WhyFa:      "افزایش پیشنهادی از تفاوت پرداخت فعلی و پرداخت هدف می‌آید؛ بدون حکم ابلاغ‌شده، «پرداخت فعلی» نامعلوم است.",
			Status:     readinessStatus(decreeSampleHit > 0 && decreeSampleHit >= decreeSample, decreeSampleHit > 0, decreeSample > 0),
			CountFa:    rules.IntFa(decreeSampleHit) + " حکم ابلاغ‌شده در " + rules.IntFa(decreeSample) + " کارمند نمونه",
			Done:       decreeSample > 0 && decreeSampleHit >= decreeSample,
		},
		{
			Order: 10, TitleFa: "تصمیم چهارعاملی و اعلام برای صدور حکم", TableFa: "تصمیم‌های جبران خدمات",
			ModuleFa: "جبران خدمات ۴پی", PageKey: "compDecision",
			CriticalFa: []string{"شمارهٔ تصمیم", "تاریخ تصمیم و تاریخ اجرا", "امتیاز شغل", "پلهٔ گروه شغلی",
				"سطح شایستگی", "امتیاز عملکرد", "پرداخت فعلی و پرداخت هدف", "افزایش مصوب", "خلاصهٔ تصمیم"},
			WhyFa:   "خروجی رسمی نظام ۴پی همین تصمیم است؛ واحد امور کارکنان حکم حقوق را بر پایهٔ آن صادر می‌کند.",
			Status:  readinessStatus(len(decisions) > 0 && countStatus(decisions, "applied") > 0, len(decisions) > 0, len(decisions) > 0),
			CountFa: rules.IntFa(len(decisions)) + " تصمیم ثبت‌شده (" + rules.IntFa(countStatus(decisions, "applied")) + " اجراشده)",
			Done:    len(decisions) > 0,
		},
	}
	return steps
}

// readinessStatus وضعیت گام: کامل، ناقص یا شروع‌نشده.
func readinessStatus(ok, partial, started bool) string {
	switch {
	case ok:
		return "ok"
	case partial || started:
		return "partial"
	default:
		return "empty"
	}
}

// readinessStatusFa عنوان فارسی وضعیت (برای نمایش در فرم و گزارش).
func readinessStatusFa(status string) string {
	switch status {
	case "ok":
		return "کامل"
	case "partial":
		return "ناقص — در جریان"
	default:
		return "شروع‌نشده"
	}
}

// countFa متن شمارشی گام: اگر شرط کامل برقرار نباشد، هر دو شمارش نمایش داده می‌شود.
func countFa(primary, secondary string) string {
	if strings.TrimSpace(secondary) == "" {
		return primary
	}
	return primary + " | " + secondary
}

// listOwn رکوردهای یک نوع از همین ماژول (برای سنجش پیشرفت گام‌ها).
func listOwn(ctx context.Context, svc *recordkit.Service, kind string) []contracts.Record {
	page, err := svc.List(ctx, contracts.RecordFilter{
		Module: "compensation", Kind: kind, Scope: "all", Size: 500,
	})
	if err != nil {
		return nil
	}
	return page.Items
}

// countStatus شمار رکوردهای یک وضعیت.
func countStatus(records []contracts.Record, status string) int {
	n := 0
	for _, rec := range records {
		if rec.Status == status {
			n++
		}
	}
	return n
}

// distinctEmployees شمار کارکنان یگانه در فهرست رکوردها.
func distinctEmployees(records []contracts.Record) int {
	seen := map[string]bool{}
	for _, rec := range records {
		if id := strings.TrimSpace(rec.EmployeeID); id != "" {
			seen[id] = true
		}
	}
	return len(seen)
}

// recordCount شمار رکوردهای یک نوع از ماژول دیگر، از راه پورت قرارداد.
func recordCount(ctx context.Context, deps modulekit.Deps, moduleID, kind, status string) int {
	if status == "" {
		return recordCountStatus(ctx, deps, moduleID, kind, "")
	}
	return recordCountStatus(ctx, deps, moduleID, kind, status)
}

// recordCountStatus شمار رکوردهای یک نوع (و اختیاراً یک وضعیت) در ماژول دیگر.
func recordCountStatus(ctx context.Context, deps modulekit.Deps, moduleID, kind, status string) int {
	port, ok := deps.Ports.RecordList(moduleID)
	if !ok {
		return 0
	}
	page, err := port.List(ctx, contracts.RecordFilter{
		Module: moduleID, Kind: kind, Status: status, Scope: "all", Size: 1,
	})
	if err != nil {
		return 0
	}
	return page.Total
}

// criticalLabels برچسب فارسی فیلدهای الزامی یک نوع رکورد، از شناسهٔ همان ماژول.
//
// چرا از شناسه و نه فهرست دستی: اگر دامنهٔ ماژول مالک، فیلد الزامی تازه‌ای اضافه کند،
// همین راهنما هم خودکار آن را نشان می‌دهد و «دادهٔ حیاتی» از قید واقعی ذخیره جدا نمی‌افتد.
func criticalLabels(ctx context.Context, deps modulekit.Deps, moduleID, kind string) []string {
	port, ok := deps.Ports.RecordList(moduleID)
	if !ok {
		return nil
	}
	schema := port.Schema()
	for _, k := range schema.Kinds {
		if k.Key != kind {
			continue
		}
		out := make([]string, 0, len(k.Fields))
		for _, field := range k.Fields {
			if field.Required {
				out = append(out, field.LabelFa)
			}
		}
		return out
	}
	return nil
}

// decreeSampleCoverage درصد پوشش احکام حقوق را با نمونه‌گیری از کارکنان می‌سنجد؛ پورت
// احکام فقط «حکم یک کارمند» را می‌دهد و فهرست کامل را در اختیار نمی‌گذارد.
func decreeSampleCoverage(ctx context.Context, deps modulekit.Deps, svc *recordkit.Service) (int, int) {
	const sampleSize = 20
	reader, ok := deps.Ports.Employees()
	if !ok {
		return 0, 0
	}
	page, err := reader.List(ctx, contracts.EmployeeFilter{Scope: "all", Size: sampleSize})
	if err != nil || len(page.Items) == 0 {
		return 0, 0
	}
	decrees, ok := deps.Ports.SalaryDecrees()
	if !ok {
		return len(page.Items), 0
	}
	hit := 0
	for _, emp := range page.Items {
		if _, found, err := decrees.NotifiedDecreeOf(ctx, emp.ID); err == nil && found {
			hit++
		}
	}
	_ = svc
	return len(page.Items), hit
}
