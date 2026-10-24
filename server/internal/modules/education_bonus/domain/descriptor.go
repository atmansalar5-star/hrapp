// منطق دامنهٔ ماژول «education_bonus» — «descriptor»: قواعد، اعتبارسنجی و محاسبات خالص بدون وابستگی به زیرساخت.

package domain

import (
	"strings"

	"hrms/internal/contracts"
	"hrms/internal/platform/recordkit"
)

// Descriptor شناسهٔ ماژول پاداش تحصیلی فرزندان ممتاز (مستقل از کارت‌های رفاهی).
func Descriptor() recordkit.Descriptor {
	return recordkit.Descriptor{
		Module:  "education_bonus",
		TitleFa: "پاداش تحصیلی فرزندان ممتاز",
		Summaries: []string{
			"ثبت مستقل دانش‌آموزان ممتاز هر کارمند با مقطع، معدل و سال تحصیلی",
			"مبالغ مصوب پاداش تحصیلی به تفکیک مقطع با شمارهٔ مصوبه و تاریخ اعتبار",
			"محاسبهٔ پاداش مشمولان بر پایهٔ مبالغ مصوب و ساخت فایل پرداخت برای مالی",
		},
		Kinds: []recordkit.Kind{
			{
				Key: "outstandingStudents", TitleFa: "دانش‌آموز ممتاز و تحت تکفل", IDPrefix: "OBS",
				EmployeeScoped: true,
				Fields: recordkit.Fields(
					"relation|نسبت با کارمند|text|1",
					"nationalId|کد ملی|text|0",
					"level|مقطع تحصیلی|text|1",
					"grade|معدل|number|0",
					"school|نام مدرسه یا دانشگاه|text|0",
					"academicYear|سال تحصیلی|text|0",
					"excellent|دانش‌آموز ممتاز (۱=بله، ۰=خیر)|number|0",
					"eligible|مشمول پاداش تحصیلی (۱/۰)|number|0",
				),
				Statuses:         recordkit.Statuses("active|مشمول", "suspended|غیرمشمول"),
				Initial:          "active",
				Transitions:      recordkit.Steps("active|suspended", "suspended|active"),
				EditableStatuses: []string{"active", "suspended"},
				Actions: recordkit.Actions(
					"suspend|غیرفعال کردن تعلق|danger|suspended|1",
					"activate|فعال کردن تعلق|primary|active|0",
				),
				Validate: validateStudent,
			},
			{
				Key: "bonusRates", TitleFa: "مبلغ مصوب پاداش تحصیلی", IDPrefix: "EBR",
				Fields: recordkit.Fields(
					"level|مقطع تحصیلی|text|1",
					"amount|مبلغ مصوب (ریال)|money|1",
					"approvalRef|شمارهٔ مصوبه|text|0",
					"effectiveFrom|تاریخ اعتبار|date|0",
				),
				Statuses:         recordkit.Statuses("active|جاری", "superseded|جایگزین‌شده"),
				Initial:          "active",
				Transitions:      recordkit.Steps("active|superseded", "superseded|active"),
				EditableStatuses: []string{"active", "superseded"},
				Actions: recordkit.Actions(
					"supersede|جایگزینی با مبلغ جدید|ghost|superseded|0",
					"activate|جاری‌سازی|primary|active|0",
				),
				Validate: validateRate,
			},
			{
				Key: "bonusFiles", TitleFa: "فایل محاسباتی پاداش تحصیلی", IDPrefix: "EBF",
				Fields: recordkit.Fields(
					"period|دورهٔ پرداخت (سال تحصیلی)|text|1",
					"scopeFa|دامنهٔ محاسبه|text|1",
					"levelsCsv|مقطع‌های محاسبه‌شده|text|0",
					"employeesCsv|شناسهٔ کارکنان (محاسبهٔ انفرادی)|text|0",
					"rows|تعداد ردیف|number|0",
					"total|جمع کل (ریال)|money|0",
					"deliveryFa|مقصد ارسال|text|0",
				),
				Statuses:         recordkit.Statuses("draft|پیش‌نویس", "prepared|آمادهٔ ارسال", "sent|ارسال‌شده به مالی"),
				Initial:          "draft",
				Transitions:      recordkit.Steps("draft|prepared", "prepared|sent"),
				EditableStatuses: []string{"draft", "prepared"},
				Actions: recordkit.Actions(
					"prepare|آماده‌سازی|primary|prepared|0",
					"send|ارسال به مالی|primary|sent|0",
				),
			},
		},
	}
}

// validateStudent قاعدهٔ دامنه: دانش‌آموزی که ممتاز نیست مشمول پاداش تحصیلی نمی‌شود.
// این قاعده در دامنه می‌ماند تا نه UI و نه موتور recordkit آن را تکرار نکنند.
func validateStudent(in contracts.RecordInput) error {
	if flagValue(in.Fields["eligible"]) && !flagValue(in.Fields["excellent"]) {
		return &recordkit.RuleError{Message: "پاداش تحصیلی فقط به دانش‌آموز ممتاز تعلق می‌گیرد؛ ابتدا نشانهٔ ممتاز بودن را ثبت کنید."}
	}
	return nil
}

// validateRate قاعدهٔ دامنه: مقطع باید از فهرست اعلامی باشد.
func validateRate(in contracts.RecordInput) error {
	key := strings.TrimSpace(in.Fields["level"])
	if _, ok := LevelByKey(key); !ok {
		return &recordkit.RuleError{Message: "مقطع ناشناخته است: " + key}
	}
	return nil
}

// flagValue پرچم‌های ۱/۰ یا بله/خیر را می‌خواند.
func flagValue(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "بله", "دارد", "فعال":
		return true
	}
	return false
}

// Seed رکوردهای نمونهٔ سازمان: مبلغ مصوب هر مقطع، چند دانش‌آموز ممتاز و یک فایل
// پرداخت آماده.
func Seed() []contracts.RecordInput {
	out := []contracts.RecordInput{}

	for _, l := range Levels() {
		out = append(out, contracts.RecordInput{
			Kind: "bonusRates", Title: l.TitleFa,
			Fields: map[string]string{
				"level": l.Key, "amount": itoa(l.DefaultRate),
				"approvalRef": "مصوبهٔ نمونه", "effectiveFrom": "1404/07/01",
			},
			NoteFa: "مبلغ مصوب — قابل بازتعریف بر پایهٔ مصوبهٔ جدید هیئت‌مدیره",
		})
	}

	students := []struct {
		emp, name, relation, level, grade, school, year, excellent, eligible string
	}{
		{"EMP-005", "علی احمدی", "فرزند", "high2", "19.4", "دبیرستان نمونه", "1404/1405", "1", "1"},
		{"EMP-005", "زهرا احمدی", "فرزند", "elementary", "18.1", "دبستان شهید", "1404/1405", "0", "0"},
		{"EMP-010", "مهدی صادقی", "فرزند", "elementary", "19.9", "دبستان معرفت", "1404/1405", "1", "1"},
		{"EMP-004", "نگار مرادی", "فرزند", "middle1", "17.6", "متوسطه اول فرزانگان", "1404/1405", "0", "0"},
	}
	for _, s := range students {
		out = append(out, contracts.RecordInput{
			Kind: "outstandingStudents", Title: s.name, EmployeeID: s.emp,
			Fields: map[string]string{
				"relation": s.relation, "level": s.level, "grade": s.grade,
				"school": s.school, "academicYear": s.year,
				"excellent": s.excellent, "eligible": s.eligible,
			},
			NoteFa: "ثبت مستقل دانش‌آموز ممتاز — مبنای تعلق پاداش تحصیلی",
		})
	}

	out = append(out, contracts.RecordInput{
		Kind: "bonusFiles", Title: "فایل پاداش تحصیلی سال تحصیلی ۱۴۰۴/۱۴۰۵",
		Fields: map[string]string{
			"period": "1404/1405", "scopeFa": "گروهی — همهٔ مشمولان ممتاز",
			"levelsCsv": "high2,elementary", "rows": "2", "total": "23000000",
			"deliveryFa": "واحد مالی",
		},
		NoteFa: "خروجی قابل ارسال به مالی، مستقل از پتروکارت و نفت‌کارت",
	})
	return out
}

// itoa تبدیل سریع عدد صحیح به رشته بدون وابستگی به strconv در بذرکاری.
func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
