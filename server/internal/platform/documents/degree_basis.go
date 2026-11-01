// پیاده‌سازی بستهٔ «documents» — «degree_basis»: فرم اداری «تعیین مدرک ملاک حقوق».
//
// این فرم پاسخ کاغذی همان پرسشی است که کارگزینی می‌پرسد: «ملاک حقوق کدام مدرک است؟» —
// مقطع و رشتهٔ مدرک ملاک، مرتبط یا غیرمرتبط بودن و درصد ارتباط، شماره و تاریخ صورت‌جلسهٔ
// شورای تعدیل مدرک، گروه شغلی مبنای پرداخت و — برای ایثارگران — مقطع معادل‌سازی‌شدهٔ یک
// پله بالاتر. جدول سابقهٔ تحصیلی هم در همان برگه می‌آید تا تصمیم بدون مستنداتش چاپ نشود.
//
// قاعده‌ها در دامنهٔ ماژول بایگانی است؛ این لایه فقط داده را می‌چیند و قالب چاپی می‌سازد.

package documents

import (
	"context"
	"fmt"
	"strings"

	"hrms/internal/contracts"
	"hrms/internal/platform/modulekit"
)

// degreeBasisRecord رکورد «مدرک ملاک حقوق» را از ماژول مالک می‌خواند.
//
// از راه پورت قرارداد (`RecordList`) خوانده می‌شود، نه با import داخلی‌های ماژول بایگانی؛
// در فهرست با پارامتر «query» جست‌وجو و سپس با شناسه تطبیق داده می‌شود.
func degreeBasisRecord(ctx context.Context, deps modulekit.Deps, id string) (contracts.Record, bool, error) {
	port, ok := deps.Ports.RecordList("archive")
	if !ok {
		return contracts.Record{}, false, fmt.Errorf("دادهٔ بایگانی و مدرک تحصیلی در این استقرار در دسترس نیست")
	}
	page, err := port.List(ctx, contracts.RecordFilter{
		Module: "archive", Kind: "degreeBases", Query: strings.TrimSpace(id), Scope: "all", Size: 50,
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

// educationRecords سابقه‌های تحصیلی یک کارمند را از ماژول بایگانی می‌خواند.
func educationRecords(ctx context.Context, deps modulekit.Deps, employeeID string) []contracts.Record {
	port, ok := deps.Ports.RecordList("archive")
	if !ok {
		return nil
	}
	page, err := port.List(ctx, contracts.RecordFilter{
		Module: "archive", Kind: "educations", EmployeeID: employeeID, Scope: "all", Size: 100,
	})
	if err != nil {
		return nil
	}
	out := make([]contracts.Record, 0, len(page.Items))
	for _, rec := range page.Items {
		if rec.EmployeeID == employeeID {
			out = append(out, rec)
		}
	}
	return out
}

// buildDegreeBasisForm دادهٔ فرم «مدرک ملاک حقوق» را روی سند می‌نشاند.
func buildDegreeBasisForm(ctx context.Context, deps modulekit.Deps, data *Data, basis *contracts.Record) {
	if data == nil {
		return
	}
	data.FormTitleFa = "مشخصات تصمیم مدرک ملاک حقوق"
	data.Signatures = []string{
		"مهر و امضای شورای تعدیل مدرک",
		"مهر و امضای واحد کارگزینی",
		"امضای کارمند",
	}

	if basis == nil {
		// اگر تصمیم خوانده نشد، فرم خالی چاپ نمی‌شود؛ توضیح صادقانه می‌آید.
		data.NoteFa = "اطلاعات تصمیم «مدرک ملاک حقوق» قابل خواندن نبود؛ این برگه را از کارتابل بایگانی بازبینی کنید."
		return
	}
	employeeID := basis.EmployeeID
	if employeeID == "" {
		employeeID = data.Employee.ID
	}

	level := strings.TrimSpace(basis.Fields["level"])
	levelFa := contracts.DegreeLevelTitleFa(level)
	veteran := strings.TrimSpace(basis.Fields["veteranUpgrade"]) == "yes"
	upgradedLevel := strings.TrimSpace(basis.Fields["upgradedLevel"])
	upgradedFa := contracts.DegreeLevelTitleFa(upgradedLevel)
	effectiveLevelFa := levelFa
	if veteran && upgradedFa != "" {
		effectiveLevelFa = upgradedFa
	}
	relevance := strings.TrimSpace(basis.Fields["relevance"])

	// سطرهای «فرم اداری»: هر سطر یک پرسش و پاسخ روشن.
	rows := [][2]string{
		{"کارمند", strings.TrimSpace(basis.EmployeeName) + " — " + employeeID},
		{"واحد سازمانی", data.Employee.DepartmentName},
		{"پست سازمانی", data.Employee.Position},
		{"شمارهٔ تصمیم در سامانه", basis.ID},
		{"وضعیت تصمیم", basis.StatusFa},
		{"کد سابقهٔ تحصیلی ملاک", strings.TrimSpace(basis.Fields["education"])},
		{"مقطع مدرک ملاک", levelFa},
		{"رشته و گرایش مدرک ملاک", joinNonEmpty(" — ", strings.TrimSpace(basis.Fields["field"]), strings.TrimSpace(basis.Fields["orientation"]))},
		{"ارتباط مدرک با شغل", contracts.DegreeRelevanceTitleFa(relevance)},
		{"درصد ارتباط مدرک با شغل", percentFa(basis.Fields["relevancePercent"])},
		{"صورت‌جلسهٔ شورای تعدیل مدرک", joinNonEmpty(" مورخ ", strings.TrimSpace(basis.Fields["councilNo"]), strings.TrimSpace(basis.Fields["councilDate"]))},
		{"تاریخ اجرا", strings.TrimSpace(basis.Fields["effectiveDate"])},
		{"گروه شغلی مبنای حقوق", strings.TrimSpace(basis.Fields["salaryGroup"])},
		{"معادل‌سازی ایثارگران (یک مقطع بالاتر)", contracts.YesNoTitleFa(basis.Fields["veteranUpgrade"])},
		{"مقطع معادل‌سازی‌شده", upgradedFa},
		{"مقطع مبنای پرداخت", effectiveLevelFa},
		{"اثر حقوقی ثبت‌شده", strings.TrimSpace(basis.Fields["salaryEffect"])},
	}
	for _, row := range rows {
		value := strings.TrimSpace(row[1])
		if value == "" {
			value = "—"
		}
		data.FormFields = append(data.FormFields, FormField{LabelFa: row[0], ValueFa: value})
	}

	// جدول سابقهٔ تحصیلی همان کارمند: تصمیم بدون مستنداتش چاپ نشود.
	records := educationRecords(ctx, deps, employeeID)
	if len(records) > 0 {
		table := FormTable{
			TitleFa: "سابقهٔ تحصیلی ثبت‌شده در پرونده",
			Head: []string{
				"مقطع", "رشته و گرایش", "دانشگاه", "تاریخ فراغت",
				"ارتباط با شغل", "درصد ارتباط", "وضعیت شورا", "مدرک ملاک",
			},
			NoteFa: "این جدول از سابقه‌های تحصیلی پروندهٔ کارمند ساخته شده است؛ هر تغییر مدرک، تصمیم تازهٔ ملاک حقوق می‌خواهد.",
		}
		for _, rec := range records {
			isBasis := rec.ID == strings.TrimSpace(basis.Fields["education"])
			basisFa := "خیر"
			if isBasis || (strings.TrimSpace(basis.Fields["education"]) == "" &&
				rec.Fields["level"] == level && strings.TrimSpace(rec.Fields["field"]) == strings.TrimSpace(basis.Fields["field"])) {
				basisFa = "بله — ملاک این تصمیم"
			}
			table.Rows = append(table.Rows, []string{
				contracts.DegreeLevelTitleFa(rec.Fields["level"]),
				joinNonEmpty(" — ", rec.Fields["field"], rec.Fields["orientation"]),
				strings.TrimSpace(rec.Fields["university"]),
				strings.TrimSpace(rec.Fields["graduationDate"]),
				contracts.DegreeRelevanceTitleFa(rec.Fields["relevance"]),
				percentFa(rec.Fields["relevancePercent"]),
				contracts.DegreeCouncilStatusTitleFa(rec.Fields["councilStatus"]),
				basisFa,
			})
		}
		data.FormTables = append(data.FormTables, table)
	}

	// شرح فرم: تصمیم چیست و چه اثری دارد.
	body := []string{
		fmt.Sprintf("بر پایهٔ بررسی مدارک تحصیلی و صورت‌جلسهٔ شورای تعدیل مدرک، مدرک «%s %s» به‌عنوان مدرک ملاک حقوق نامبرده تعیین می‌شود.",
			levelFa, strings.TrimSpace(basis.Fields["field"])),
		fmt.Sprintf("ارتباط این مدرک با شغل «%s» و میزان آن %s ارزیابی شده است.",
			contracts.DegreeRelevanceTitleFa(relevance), percentFa(basis.Fields["relevancePercent"])),
	}
	if veteran && upgradedFa != "" {
		body = append(body, fmt.Sprintf("با احتساب ضوابط ایثارگران، این مدرک یک مقطع بالاتر معادل‌سازی شده و مقطع «%s» مبنای گروه شغلی و پرداخت است.", upgradedFa))
	}
	if group := strings.TrimSpace(basis.Fields["salaryGroup"]); group != "" {
		body = append(body, "گروه شغلی مبنای پرداخت: "+group+".")
	}
	if effect := strings.TrimSpace(basis.Fields["salaryEffect"]); effect != "" {
		body = append(body, "اثر حقوقی: "+effect+".")
	}
	data.BodyFa = body
	data.NoteFa = "این فرم مبنای تعیین گروه شغلی و پرداخت است و با هر تغییر مدرک یا نظر شورا، تصمیم تازه صادر می‌شود؛ تصمیم‌های پیشین در سابقه می‌مانند."
	if note := strings.TrimSpace(basis.NoteFa); note != "" {
		data.NoteFa = data.NoteFa + " " + note
	}
}

// joinNonEmpty اجزای غیرخالی را با جداکننده به هم می‌چسباند.
func joinNonEmpty(separator string, parts ...string) string {
	kept := make([]string, 0, len(parts))
	for _, part := range parts {
		if value := strings.TrimSpace(part); value != "" {
			kept = append(kept, value)
		}
	}
	return strings.Join(kept, separator)
}

// percentFa درصد را با علامت درصد برمی‌گرداند (خالی یعنی «—»).
func percentFa(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "—"
	}
	return trimmed + "٪"
}
