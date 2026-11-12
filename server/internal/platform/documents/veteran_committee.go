// پیاده‌سازی بستهٔ «documents» — «veteran_committee»: صورت‌جلسه و مصوبهٔ کمیتهٔ ایثارگری.
//
// این برگه پاسخ کاغذی همان پرسشی است که کمیتهٔ ایثارگری باید برای هر فرد روشن کند: کدام
// امتیاز تعلق می‌گیرد و به چه میزان؟ یک مقطع تحصیلی بالاتر؟ معافیت مالیاتی؟ معافیت کسور
// بازنشستگی؟ کاهش ساعت کار جانباز؟ چند پایهٔ تشویقی؟ افزایش سنواتی و بر چه مبنایی؟ مرخصی
// تشویقی؟ سهمیهٔ استخدامی؟ کنار هر بند، «پیش‌فرض قانونی» همان کارمند چاپ می‌شود تا اعضای
// کمیته بدانند تصمیمشان کجا از قاعدهٔ عمومی فاصله گرفته است.
//
// قاعده‌ها در دامنهٔ ماژول امور رفاهی (`grievances`) و موتور قواعد (`internal/rules`) است؛ این
// لایه فقط داده را می‌چیند و برگهٔ امضاشدنی می‌سازد.
package documents

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"hrms/internal/contracts"
	"hrms/internal/platform/modulekit"
	"hrms/internal/rules"
)

// veteranCommitteeRecord رکورد مصوبهٔ کمیته را از ماژول مالک (امور رفاهی و ایثارگران) می‌خواند.
func veteranCommitteeRecord(ctx context.Context, deps modulekit.Deps, id string) (contracts.Record, bool, error) {
	port, ok := deps.Ports.RecordList("grievances")
	if !ok {
		return contracts.Record{}, false, fmt.Errorf("دادهٔ امور ایثارگران در این استقرار در دسترس نیست")
	}
	page, err := port.List(ctx, contracts.RecordFilter{
		Module: "grievances", Kind: "veteranCommittee", Query: strings.TrimSpace(id), Scope: "all", Size: 50,
	})
	if err != nil {
		return contracts.Record{}, false, err
	}
	for _, rec := range page.Items {
		if rec.ID == strings.TrimSpace(id) {
			return rec, true, nil
		}
	}
	return contracts.Record{}, false, nil
}

// veteranCommitteeSignatures بلوک امضای پایین برگه را از «اعضای تعریف‌شدهٔ» کمیته می‌سازد.
//
// چرا از رکوردهای عضو و نه از فهرست ثابت: ترکیب امضاکنندگان را سازمان تعریف می‌کند (پنل
// اعضای کمیته) و برگهٔ چاپی باید همان ترکیب را با نام و نقش نشان دهد. اگر هیچ عضوی تعریف
// نشده باشد، سه امضای متعارف برمی‌گردد تا برگه بی‌امضا نماند؛ مهر مدیریت منابع انسانی در
// هر حالت آخرین امضاست و اگر عضوی با نقش «مهر و امضای مدیریت» تعریف شده باشد همان می‌آید.
func veteranCommitteeSignatures(ctx context.Context, deps modulekit.Deps) []string {
	body := []string{}
	seal := "مهر و امضای مدیریت منابع انسانی"
	if port, ok := deps.Ports.RecordList("grievances"); ok {
		page, err := port.List(ctx, contracts.RecordFilter{
			Module: "grievances", Kind: "veteranCommitteeMember", Scope: "all", Size: 200,
		})
		if err == nil {
			active := make([]contracts.Record, 0, len(page.Items))
			for _, rec := range page.Items {
				if rec.Status == "active" {
					active = append(active, rec)
				}
			}
			sort.SliceStable(active, func(i, j int) bool {
				oi, oj := committeeMemberSignatureSort(active[i].Fields), committeeMemberSignatureSort(active[j].Fields)
				if oi != oj {
					return oi < oj
				}
				return strings.TrimSpace(active[i].Fields["fullName"]) < strings.TrimSpace(active[j].Fields["fullName"])
			})
			for _, rec := range active {
				role := rules.VeteranCommitteeRoleOf(rec.Fields["role"])
				label := rules.VeteranCommitteeSignatureFa(role, rec.Fields["fullName"], rec.Fields["signatureTitle"])
				if role == rules.VeteranCommitteeHRSeal {
					seal = label
					continue
				}
				body = append(body, label)
			}
		}
	}
	if len(body) == 0 {
		body = []string{"رئیس کمیته", "دبیر کمیته", "نمایندهٔ امور ایثارگران"}
	}
	return append(body, seal)
}

// committeeMemberSignatureSort ترتیب نشستن امضا: «ترتیب امضا»ی ثبت‌شده، وگرنه ترتیب نقش.
func committeeMemberSignatureSort(f map[string]string) int {
	if value := strings.TrimSpace(f["sortOrder"]); value != "" {
		n := 0
		for _, ch := range value {
			if ch < '0' || ch > '9' {
				return rules.VeteranCommitteeRoleOrder(f["role"])
			}
			n = n*10 + int(ch-'0')
		}
		if n > 0 {
			return n
		}
	}
	return 100 + rules.VeteranCommitteeRoleOrder(f["role"])
}

// buildVeteranCommitteeForm دادهٔ برگهٔ «مصوبهٔ کمیتهٔ ایثارگری» را روی سند می‌نشاند.
func buildVeteranCommitteeForm(ctx context.Context, deps modulekit.Deps, data *Data, committee *contracts.Record) {
	if data == nil {
		return
	}
	data.FormTitleFa = "مشخصات جلسهٔ کمیته و متن مصوبه"
	data.Signatures = veteranCommitteeSignatures(ctx, deps)
	if committee == nil {
		data.NoteFa = "اطلاعات مصوبهٔ کمیته قابل خواندن نبود؛ این برگه را از کارتابل امور ایثارگران بازبینی کنید."
		return
	}
	f := committee.Fields
	employeeID := strings.TrimSpace(committee.EmployeeID)
	group := rules.VeteranGroupOf(f["veteranGroup"])
	groupFa := rules.VeteranGroupTitleFa(group)

	rows := [][2]string{
		{"کارمند", strings.TrimSpace(committee.EmployeeName) + " — " + employeeID},
		{"واحد سازمانی", data.Employee.DepartmentName},
		{"پست سازمانی", data.Employee.Position},
		{"شناسهٔ مصوبه در سامانه", committee.ID},
		{"وضعیت مصوبه", committee.StatusFa},
		{"شمارهٔ صورت‌جلسه یا مصوبه", strings.TrimSpace(f["committeeNo"])},
		{"تاریخ جلسهٔ کمیته", strings.TrimSpace(f["meetingDate"])},
		{"اعضای کمیته", strings.TrimSpace(f["members"])},
		{"گروه ایثارگری", groupFa},
		{"مستندات بررسی‌شده", strings.TrimSpace(f["basisDocs"])},
		{"تاریخ اجرا", strings.TrimSpace(f["effectiveDate"])},
		{"امضاکنندگان", strings.TrimSpace(f["signOff"])},
	}
	for _, row := range rows {
		value := strings.TrimSpace(row[1])
		if value == "" {
			value = "—"
		}
		data.FormFields = append(data.FormFields, FormField{LabelFa: row[0], ValueFa: value})
	}

	// جدول بندهای مصوبه: تصمیم کمیته در کنار پیش‌فرض قانونی همان کارمند.
	decision := rules.VeteranDecisionFromFields(f)
	base, hasBase := committeeLegalBase(ctx, deps, employeeID)
	head := []string{"بند", "پیش‌فرض قانونی / سیاست شرکت", "مصوبهٔ کمیته", "اثر در فیش و پرونده"}
	table := FormTable{
		TitleFa: "بندهای تصمیم کمیته",
		Head:    head,
		NoteFa: "ستون «پیش‌فرض قانونی» از پروندهٔ تأییدشدهٔ همان کارمند و موتور قواعد ساخته می‌شود؛ " +
			"مصوبهٔ کمیته بر آن مقدم است و در پروندهٔ کارمند ثبت می‌شود.",
	}
	for _, line := range committeeDecisionRows(decision, f, base, hasBase) {
		table.Rows = append(table.Rows, line)
	}
	if len(table.Rows) == 0 {
		table.Rows = append(table.Rows, []string{"—", "—", "کمیته هیچ بندی را تغییر نداد", "بدون تغییر"})
	}
	data.FormTables = append(data.FormTables, table)

	// شرح برگه: متن مصوبه و مغایرت‌های آن با پیش‌فرض قانونی.
	body := []string{}
	if decisionFa := strings.TrimSpace(f["decisionFa"]); decisionFa != "" {
		body = append(body, "متن مصوبه: "+decisionFa)
	}
	if hasBase {
		_, changes := rules.ApplyVeteranDecision(base, decision)
		if len(changes) > 0 {
			body = append(body, "بندهایی که مصوبهٔ کمیته خلاف پیش‌فرض قانونی یا کم‌تر و بیش‌تر از آن تعیین کرده است:")
			body = append(body, changes...)
		} else {
			body = append(body, "مصوبهٔ کمیته در همهٔ بندها با پیش‌فرض قانونی همین کارمند هم‌خوان است.")
		}
	} else {
		body = append(body, "پروندهٔ تأییدشدهٔ ایثارگری برای این کارمند یافت نشد؛ اثر مصوبه پس از ثبت و تأیید پرونده اعمال می‌شود.")
	}
	if other := strings.TrimSpace(f["otherBenefits"]); other != "" {
		body = append(body, "سایر امتیازات مصوب: "+other)
	}
	data.BodyFa = body
	data.NoteFa = "این صورت‌جلسه مبنای اعمال امتیازهای ایثارگری در پرونده و فیش حقوق است؛ تغییر آن نیازمند مصوبهٔ تازهٔ کمیته است."
	if note := strings.TrimSpace(committee.NoteFa); note != "" {
		data.NoteFa = data.NoteFa + " " + note
	}
}

// committeeLegalBase پیش‌فرض قانونی همین کارمند را از پروندهٔ ایثارگری می‌سازد.
func committeeLegalBase(ctx context.Context, deps modulekit.Deps, employeeID string) (rules.VeteranEntitlement, bool) {
	port, ok := deps.Ports.RecordList("grievances")
	if !ok || strings.TrimSpace(employeeID) == "" {
		return rules.VeteranEntitlement{}, false
	}
	page, err := port.List(ctx, contracts.RecordFilter{
		Module: "grievances", Kind: "veteranRecords", EmployeeID: employeeID, Scope: "all", Size: 50,
	})
	if err != nil {
		return rules.VeteranEntitlement{}, false
	}
	best := contracts.Record{}
	bestRank := -1
	for _, rec := range page.Items {
		if rec.EmployeeID != employeeID {
			continue
		}
		rank := 0
		switch rec.Status {
		case "approved":
			rank = 2
		case "verified":
			rank = 1
		}
		if rank > bestRank {
			best, bestRank = rec, rank
		}
	}
	if bestRank < 0 {
		return rules.VeteranEntitlement{}, false
	}
	values := map[string]float64{}
	for _, param := range deps.Org.Params {
		if !strings.HasPrefix(param.Key, "veteran.") {
			continue
		}
		values[param.Key] = rules.ParseFaNumber(param.Value)
	}
	params := rules.VeteranParamsFromValues(values)
	profile := rules.VeteranProfileFromFields(best.Fields, best.Status == "approved")
	return rules.ComputeVeteranEntitlements(profile, params), true
}

// committeeDecisionRows سطرهای جدول بندها: پیش‌فرض قانونی در برابر مصوبهٔ کمیته.
func committeeDecisionRows(dec rules.VeteranDecision, f map[string]string, base rules.VeteranEntitlement, hasBase bool) [][]string {
	rows := [][]string{}
	legal := func(text string) string {
		if !hasBase {
			return "پروندهٔ تأییدشده یافت نشد"
		}
		return text
	}
	percentFa := func(v float64) string { return rules.IntFa(v) + "٪" }
	yesNo := func(v bool) string {
		if v {
			return "بله"
		}
		return "خیر"
	}
	decided := func(current string) string {
		if strings.TrimSpace(current) == "" {
			return "پیروی از پیش‌فرض (تصمیم تازه‌ای گرفته نشد)"
		}
		return current
	}

	rows = append(rows, []string{
		"یک مقطع تحصیلی بالاتر",
		legal(yesNo(base.EducationUpgrade)),
		decided(rules.DecisionYesNoFa(dec.EduUpgrade)),
		"مدرک ملاک حقوق و گروه شغلی",
	})
	taxCurrent := "تعلق نمی‌گیرد"
	if hasBase && base.TaxExempt > 0 {
		taxCurrent = "تعلق می‌گیرد — " + percentFa(base.TaxExempt)
	}
	taxDecided := decided(decisionPercentText(dec.TaxExempt, dec.TaxExemptPercent))
	rows = append(rows, []string{"معافیت مالیاتی حقوق و فوق‌العاده‌ها", legal(taxCurrent), taxDecided, "مالیات فیش حقوق"})

	pensionCurrent := "تعلق نمی‌گیرد"
	if hasBase && base.PensionExempt > 0 {
		pensionCurrent = "تعلق می‌گیرد — " + percentFa(base.PensionExempt)
	}
	rows = append(rows, []string{
		"معافیت کسور بازنشستگی",
		legal(pensionCurrent),
		decided(decisionPercentText(dec.PensionExempt, dec.PensionExemptPercent)),
		"کسور سهم کارمند در فیش",
	})

	hoursDecided := ""
	if dec.WorkHourReduction {
		hours := dec.ReducedWeeklyHours
		if hours <= 0 {
			hours = 36
		}
		hoursDecided = rules.IntFa(hours) + " ساعت در هفته"
	}
	rows = append(rows, []string{
		"کاهش ساعت کار جانباز",
		"سیاست شرکت (بدون مصوبه اعمال نمی‌شود)",
		decided(hoursDecided),
		"حضور و کارکرد ماهانه",
	})

	rows = append(rows, []string{
		"پایهٔ تشویقی",
		legal(rules.IntFa(base.IncentiveSteps) + " پایه"),
		decided(decisionIntText(dec.IncentiveSteps, " پایه")),
		"پایهٔ تشویقی حکم و فیش نفت",
	})
	rows = append(rows, []string{
		"افزایش سنواتی",
		legal(rules.IntFa(base.TotalExtraServiceMonths) + " ماه"),
		decided(decisionIntText(dec.SeniorityMonths, " ماه") + seniorityBasisSuffix(dec)),
		"سابقهٔ خدمت و بازنشستگی",
	})
	rows = append(rows, []string{
		"مرخصی تشویقی",
		legal(rules.IntFa(base.LeaveDays) + " روز"),
		decided(decisionIntText(dec.LeaveDays, " روز")),
		"ماندهٔ مرخصی کارمند",
	})
	quotaCurrent := "تعلق نمی‌گیرد"
	if hasBase && base.EmploymentQuotaPercent > 0 {
		quotaCurrent = "تعلق می‌گیرد — " + percentFa(base.EmploymentQuotaPercent)
	}
	rows = append(rows, []string{
		"سهمیهٔ استخدامی",
		legal(quotaCurrent),
		decided(rules.DecisionYesNoFa(dec.EmploymentQuota)),
		"جذب و استخدام، گزارش جامعهٔ ایثارگران",
	})
	if other := strings.TrimSpace(f["otherBenefits"]); other != "" {
		rows = append(rows, []string{"سایر امتیازات", "—", other, "بر پایهٔ مصوبه"})
	}
	return rows
}

// decisionPercentText متن بند درصدی مصوبه.
func decisionPercentText(flag *bool, percent *float64) string {
	if flag == nil && percent == nil {
		return ""
	}
	if flag != nil && !*flag {
		return "تعلق نمی‌گیرد"
	}
	if percent != nil {
		return "تعلق می‌گیرد — " + rules.IntFa(*percent) + "٪"
	}
	return "تعلق می‌گیرد"
}

// decisionIntText متن بند عددی مصوبه.
func decisionIntText(value *int, unitFa string) string {
	if value == nil {
		return ""
	}
	return rules.IntFa(*value) + unitFa
}

// seniorityBasisSuffix مبنای افزایش سنواتی را به متن اضافه می‌کند.
func seniorityBasisSuffix(dec rules.VeteranDecision) string {
	if dec.SeniorityMonths == nil || strings.TrimSpace(dec.SeniorityBasis) == "" {
		return ""
	}
	return " — مبنا: " + rules.VeteranSeniorityBasisTitleFa(dec.SeniorityBasis)
}
