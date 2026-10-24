// منطق دامنهٔ ماژول «cards» — «descriptor»: قواعد، اعتبارسنجی و محاسبات خالص بدون وابستگی به زیرساخت.

package domain

import (
	"strings"

	"hrms/internal/contracts"
	"hrms/internal/platform/recordkit"
)

// Descriptor شناسهٔ ماژول کارت‌های رفاهی و شارژ.
func Descriptor() recordkit.Descriptor {
	return recordkit.Descriptor{
		Module:  "cards",
		TitleFa: "کارت‌های رفاهی و شارژ (پتروکارت، نفت‌کارت)",
		Summaries: []string{
			"شارژ ماهیانهٔ پتروکارت در بخش‌های فرهنگی و رستوران",
			"شارژ نفت‌کارت (نفت‌مارت / فارس‌کارت / سامان‌کارت) با چهار قلم رفاهی",
			"تخصیص یا عدم تخصیص به افراد تحت تکفل و پاداش تحصیلی فرزندان ممتاز",
			"بازتعریف نرخ‌ها و مناسبت‌های تقویمی و ساخت فایل شارژ برای مالی",
		},
		Kinds: []recordkit.Kind{
			{
				Key: "cardBeneficiaries", TitleFa: "فرد مشمول و تحت تکفل", IDPrefix: "CBN",
				EmployeeScoped: true,
				Fields: recordkit.Fields(
					"relation|نسبت با کارمند|text|1",
					"nationalId|کد ملی|text|0",
					"birthDate|تاریخ تولد|date|0",
					"educationLevel|مقطع تحصیلی|text|0",
					"gpa|معدل|number|0",
					"excellent|فرزند ممتاز (۱=بله، ۰=خیر)|number|0",
					"petrokartCultural|تخصیص بخش فرهنگی پتروکارت (۱/۰)|number|0",
					"petrokartRestaurant|تخصیص رستوران پتروکارت (۱/۰)|number|0",
					"educationBonus|تخصیص پاداش تحصیلی (۱/۰)|number|0",
					"naftmart|تخصیص نفت‌کارت (۱/۰)|number|0",
				),
				Statuses:         recordkit.Statuses("active|مشمول", "suspended|غیرمشمول"),
				Initial:          "active",
				Transitions:      recordkit.Steps("active|suspended", "suspended|active"),
				EditableStatuses: []string{"active", "suspended"},
				Actions: recordkit.Actions(
					"suspend|عدم تخصیص|danger|suspended|1",
					"activate|تخصیص مجدد|primary|active|0",
				),
				Validate: validateBeneficiary,
			},
			{
				Key: "cardRates", TitleFa: "نرخ و قلم شارژ", IDPrefix: "CRT",
				Fields: recordkit.Fields(
					"card|کارت|text|1",
					"section|بخش یا قلم|text|1",
					"amount|مبلغ (ریال)|money|1",
					"unitFa|واحد محاسبه|text|0",
					"effectiveFrom|تاریخ اعتبار|date|0",
				),
				Statuses:         recordkit.Statuses("active|جاری", "superseded|جایگزین‌شده"),
				Initial:          "active",
				Transitions:      recordkit.Steps("active|superseded", "superseded|active"),
				EditableStatuses: []string{"active", "superseded"},
				Actions: recordkit.Actions(
					"supersede|جایگزینی با نرخ جدید|ghost|superseded|0",
					"activate|جاری‌سازی|primary|active|0",
				),
				Validate: validateRate,
			},
			{
				Key: "occasions", TitleFa: "مناسبت تقویمی", IDPrefix: "OCC",
				Fields: recordkit.Fields(
					"date|تاریخ شمسی مناسبت|date|1",
					"categoryFa|نوع مناسبت|text|0",
					"card|کارت هدف|text|0",
					"amount|مبلغ شارژ مناسبتی (ریال)|money|0",
					"scopeFa|دامنهٔ مشمولان|text|0",
				),
				Statuses:         recordkit.Statuses("active|فعال", "inactive|غیرفعال"),
				Initial:          "active",
				Transitions:      recordkit.Steps("active|inactive", "inactive|active"),
				EditableStatuses: []string{"active", "inactive"},
				Actions: recordkit.Actions(
					"deactivate|غیرفعال‌سازی|ghost|inactive|0",
					"activate|فعال‌سازی|primary|active|0",
				),
				Validate: validateOccasion,
			},
			{
				Key: "petrokartCharges", TitleFa: "شارژ پتروکارت", IDPrefix: "PKC",
				EmployeeScoped: true,
				Fields: recordkit.Fields(
					"period|دورهٔ شارژ (YYYY/MM)|text|1",
					"cultural|مبلغ بخش فرهنگی|money|0",
					"restaurant|مبلغ بخش رستوران|money|0",
					"education|پاداش تحصیلی|money|0",
					"beneficiaries|تعداد افراد مشمول|number|0",
					"total|جمع شارژ دوره|money|0",
					"channel|نحوهٔ شارژ|text|0",
				),
				Statuses:         recordkit.Statuses("draft|پیش‌نویس", "issued|شارژشده", "reversed|برگشتی"),
				Initial:          "draft",
				Transitions:      recordkit.Steps("draft|issued", "issued|reversed"),
				EditableStatuses: []string{"draft"},
				Actions: recordkit.Actions(
					"issue|ثبت شارژ|primary|issued|0",
					"reverse|برگشت شارژ|danger|reversed|1",
				),
			},
			{
				Key: "naftmartCharges", TitleFa: "شارژ نفت‌کارت", IDPrefix: "NMC",
				EmployeeScoped: true,
				Fields: recordkit.Fields(
					"period|دورهٔ شارژ (YYYY/MM)|text|1",
					"food|۱) کالای اساسی و خواروبار|money|0",
					"protein|۲) گوشت، مرغ و پروتئین|money|0",
					"dairy|۳) لبنیات|money|0",
					"hygiene|۴) شوینده و بهداشتی|money|0",
					"beneficiaries|تعداد افراد مشمول|number|0",
					"total|جمع شارژ دوره|money|0",
					"channel|نحوهٔ شارژ|text|0",
				),
				Statuses:         recordkit.Statuses("draft|پیش‌نویس", "issued|شارژشده", "reversed|برگشتی"),
				Initial:          "draft",
				Transitions:      recordkit.Steps("draft|issued", "issued|reversed"),
				EditableStatuses: []string{"draft"},
				Actions: recordkit.Actions(
					"issue|ثبت شارژ|primary|issued|0",
					"reverse|برگشت شارژ|danger|reversed|1",
				),
			},
			{
				Key: "chargeFiles", TitleFa: "فایل شارژ", IDPrefix: "CHF",
				Fields: recordkit.Fields(
					"card|کارت|text|1",
					"period|دورهٔ شارژ|text|1",
					"scopeFa|دامنهٔ شارژ|text|1",
					"occasionFa|مناسبت|text|0",
					"rows|تعداد ردیف|number|0",
					"total|جمع کل (ریال)|money|0",
					"deliveryFa|مقصد ارسال|text|0",
					// معیارهای ساخت فایل، تا خروجی گرفتن از همان پرونده همیشه همان ردیف‌ها را بازتولید کند.
					"scopeKey|دامنه (گروهی/انفرادی)|text|0",
					"sectionsCsv|بخش‌های شارژ|text|0",
					"employeesCsv|شناسهٔ کارکنان (شارژ انفرادی)|text|0",
					"occasionId|شناسهٔ مناسبت|text|0",
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

// validateBeneficiary قاعدهٔ دامنه: پاداش تحصیلی بدون نشانهٔ فرزند ممتاز بی‌معنا است.
// این قاعده در دامنه می‌ماند تا نه UI و نه موتور recordkit آن را تکرار نکنند.
func validateBeneficiary(in contracts.RecordInput) error {
	if flagValue(in.Fields["educationBonus"]) && !flagValue(in.Fields["excellent"]) {
		return &recordkit.RuleError{Message: "پاداش تحصیلی فقط برای فرزند ممتاز قابل تخصیص است؛ ابتدا نشانهٔ ممتاز بودن را ثبت کنید."}
	}
	return nil
}

// validateRate قاعدهٔ دامنه: کارت و بخش باید از فهرست اعلامی همان کارت باشند.
func validateRate(in contracts.RecordInput) error {
	card := strings.TrimSpace(in.Fields["card"])
	if !ValidCard(card) {
		return &recordkit.RuleError{Message: "کارت نامعتبر است؛ «petrokart» یا «naftmart» را وارد کنید."}
	}
	key := strings.TrimSpace(in.Fields["section"])
	sec, ok := SectionByKey(key)
	if !ok {
		return &recordkit.RuleError{Message: "بخش ناشناخته است: " + key}
	}
	if sec.Card != card {
		return &recordkit.RuleError{Message: "بخش «" + sec.TitleFa + "» به این کارت تعلق ندارد."}
	}
	return nil
}

// validateOccasion قاعدهٔ دامنه: کارت هدف (اگر داده شده) باید معتبر باشد.
func validateOccasion(in contracts.RecordInput) error {
	card := strings.TrimSpace(in.Fields["card"])
	if card != "" && !ValidCard(card) {
		return &recordkit.RuleError{Message: "کارت هدف نامعتبر است؛ «petrokart» یا «naftmart» را وارد کنید."}
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

// Seed رکوردهای نمونهٔ سازمان: نرخ‌ها از فهرست اعلامی، چند فرد مشمول، مناسبت‌های
// تقویمی و دو شارژ ثبت‌شده.
func Seed() []contracts.RecordInput {
	out := []contracts.RecordInput{}

	for _, s := range Sections() {
		kind := "cardRates"
		out = append(out, contracts.RecordInput{
			Kind: kind, Title: s.TitleFa,
			Fields: map[string]string{
				"card": s.Card, "section": s.Key, "amount": itoa(s.DefaultRate),
				"unitFa": s.UnitFa, "effectiveFrom": "1404/07/01",
			},
			NoteFa: "نرخ جاری — قابل بازتعریف برای نرخ‌های جدید",
		})
	}

	beneficiaries := []struct {
		emp, name, relation, edu, gpa, excellent string
		cultural, restaurant, bonus, naftmart    string
	}{
		{"EMP-005", "محمد احمدی", "خود کارمند", "", "", "0", "1", "1", "0", "1"},
		{"EMP-005", "علی احمدی", "فرزند", "متوسطه دوم", "19.4", "1", "1", "1", "1", "1"},
		{"EMP-005", "زهرا احمدی", "فرزند", "ابتدایی", "18.1", "0", "1", "1", "0", "0"},
		{"EMP-010", "فاطمه صادقی", "خود کارمند", "", "", "0", "1", "1", "0", "1"},
		{"EMP-010", "مهدی صادقی", "فرزند", "ابتدایی", "19.9", "1", "1", "0", "1", "1"},
		{"EMP-004", "علی مرادی", "خود کارمند", "", "", "0", "1", "1", "0", "1"},
		{"EMP-004", "نگار مرادی", "فرزند", "متوسطه اول", "17.6", "0", "1", "0", "0", "0"},
	}
	for _, b := range beneficiaries {
		out = append(out, contracts.RecordInput{
			Kind: "cardBeneficiaries", Title: b.name, EmployeeID: b.emp,
			Fields: map[string]string{
				"relation": b.relation, "educationLevel": b.edu, "gpa": b.gpa, "excellent": b.excellent,
				"petrokartCultural": b.cultural, "petrokartRestaurant": b.restaurant,
				"educationBonus": b.bonus, "naftmart": b.naftmart,
			},
			NoteFa: "تخصیص‌های ثبت‌شده در پنل افراد تحت تکفل",
		})
	}

	// مناسبت‌های تقویمی برگرفته از «جدول پیشنهادی پرداخت‌های غیرنقدی مناسبت‌ها و
	// اعیاد اسلامی و مذهبی» سازمان برای بازهٔ ۱۴۰۴/۱۰/۰۱ تا ۱۴۰۵/۰۹/۳۰.
	// مبلغ مصوب در سند اعلام نشده بود و پس از تصویب هیئت‌مدیره ثبت می‌شود؛
	// تاریخ‌ها بر پایهٔ تقویم همان سال محاسبه شده و قابل ویرایش‌اند.
	scopeAll := "کلیهٔ کارکنان تابع صندوق‌های بازنشستگی نفت و تأمین اجتماعی و پیمانکاران مشاغل مستمر"
	occasions := []struct{ title, date, cat, card, scope string }{
		{"روز صنعت پتروشیمی", "1404/10/08", "ملی", "petrokart", scopeAll},
		{"ولادت حضرت علی (ع) — روز پدر", "1404/10/12", "مذهبی", "petrokart", "کارکنان مرد"},
		{"عید مبعث", "1404/10/26", "مذهبی", "petrokart", scopeAll},
		{"ولادت امام حسین (ع)", "1404/11/02", "مذهبی", "petrokart", scopeAll},
		{"ولادت حضرت ابوالفضل (ع) و روز جانباز", "1404/11/03", "مذهبی", "petrokart", "کارکنان جانباز، آزادگان و رزمندگان"},
		{"ولادت امام سجاد (ع)", "1404/11/04", "مذهبی", "petrokart", scopeAll},
		{"ولادت امام زمان (عج) — نیمه شعبان", "1404/11/14", "مذهبی", "petrokart", scopeAll},
		{"۲۲ بهمن سالروز پیروزی شکوهمند انقلاب اسلامی", "1404/11/22", "ملی", "petrokart", scopeAll},
		{"کمک هزینه اقلام مصرفی ماه مبارک رمضان", "1404/11/29", "مذهبی", "naftmart", scopeAll},
		{"عید سعید فطر", "1404/12/29", "مذهبی", "petrokart", scopeAll},
		{"کمک هزینه اقلام مصرفی عید نوروز", "1405/01/01", "ملی", "naftmart", scopeAll},
		{"ولادت امام رضا (ع) — آغاز دهه کرامت", "1405/02/08", "مذهبی", "petrokart", scopeAll},
		{"روز کارگر", "1405/02/11", "ملی", "petrokart", scopeAll},
		{"روز اول ذی‌الحجه — روز ازدواج", "1405/02/28", "فرهنگی", "naftmart", "کارکنان متأهل"},
		{"عید سعید قربان", "1405/03/06", "مذهبی", "petrokart", scopeAll},
		{"عید سعید غدیر", "1405/03/14", "مذهبی", "petrokart", scopeAll},
		{"ولادت حضرت رسول اکرم (ص)", "1405/06/08", "مذهبی", "petrokart", scopeAll},
		{"ولادت امام جعفر صادق (ع)", "1405/06/08", "مذهبی", "petrokart", scopeAll},
		{"گرامیداشت هفته دفاع مقدس", "1405/06/31", "ملی", "petrokart", "کارکنان شاغل در مناطق عملیاتی دفاع مقدس (ماهشهر و ایلام) و سایر شرکت‌های گروه"},
		{"کمک هزینه خرید کتاب و لوازم‌التحریر", "1405/07/01", "فرهنگی", "naftmart", scopeAll},
		{"روز ایمنی و آتش‌نشانی", "1405/07/07", "سازمانی", "petrokart", scopeAll},
		{"ولادت حضرت فاطمه زهرا (س) — روز مادر و روز زن", "1405/09/09", "مذهبی", "petrokart", "کارکنان زن"},
		{"کمک هزینه اقلام مصرفی شب یلدا", "1405/09/30", "فرهنگی", "naftmart", scopeAll},
	}
	for _, o := range occasions {
		out = append(out, contracts.RecordInput{
			Kind: "occasions", Title: o.title,
			Fields: map[string]string{
				"date": o.date, "categoryFa": o.cat, "card": o.card,
				"amount": "", "scopeFa": o.scope,
			},
			NoteFa: "مبلغ مصوب پس از تصویب هیئت‌مدیره ثبت می‌شود؛ تاریخ و دامنه قابل ویرایش است",
		})
	}

	out = append(out,
		contracts.RecordInput{
			Kind: "petrokartCharges", Title: "شارژ پتروکارت مهر ۱۴۰۴ — محمد احمدی", EmployeeID: "EMP-005",
			Fields: map[string]string{
				"period": "1404/07", "cultural": "15000000", "restaurant": "24000000", "education": "12000000",
				"beneficiaries": "3", "total": "51000000", "channel": "گروهی",
			},
			NoteFa: "شارژ دوره‌ای پتروکارت",
		},
		contracts.RecordInput{
			Kind: "naftmartCharges", Title: "شارژ نفت‌کارت مهر ۱۴۰۴ — فاطمه صادقی", EmployeeID: "EMP-010",
			Fields: map[string]string{
				"period": "1404/07", "food": "20000000", "protein": "12000000", "dairy": "6000000",
				"hygiene": "5000000", "beneficiaries": "2", "total": "43000000", "channel": "گروهی",
			},
			NoteFa: "اقلام چهارگانه رفاهی نفت‌کارت",
		},
		contracts.RecordInput{
			Kind: "chargeFiles", Title: "فایل شارژ مهر ۱۴۰۴ — پتروکارت و نفت‌کارت",
			Fields: map[string]string{
				"card": "petrokart", "period": "1404/07", "scopeFa": "گروهی — همهٔ مشمولان",
				"occasionFa": "—", "rows": "26", "total": "318000000", "deliveryFa": "واحد مالی",
			},
			NoteFa: "خروجی قابل ارسال به مالی",
		},
	)
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
