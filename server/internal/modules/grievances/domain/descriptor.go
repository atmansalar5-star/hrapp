// Package domain دامنهٔ ماژول شکایات و امور ایثارگران.
//
// شکایت کارمند، پروندهٔ ایثارگری و تسهیلات ایثارگران هر یک یک «نوع رکورد» با گردشکار خودش
// است. قاعده‌های امتیاز ایثارگری در موتور قواعد (`internal/rules/veteran.go`) است و این بسته
// فقط شکل پرونده، گردشکار و اعتبارسنجی آن را اعلام می‌کند.
package domain

import (
	"strconv"
	"strings"

	"hrms/internal/contracts"
	"hrms/internal/platform/calendar"
	"hrms/internal/platform/recordkit"
	"hrms/internal/rules"
)

// کلیدهای نوع رکورد این ماژول.
const (
	// KindComplaints شکایت کارمند.
	KindComplaints = "complaints"
	// KindVeteranRecords پروندهٔ ایثارگری کارمند.
	KindVeteranRecords = "veteranRecords"
	// KindVeteranServices تسهیلات و معرفی‌نامه‌های ایثارگران.
	KindVeteranServices = "veteranServices"
)

// Descriptor شناسهٔ ماژول شکایات و امور ایثارگران.
func Descriptor() recordkit.Descriptor {
	return recordkit.Descriptor{
		Module:  "grievances",
		TitleFa: "شکایات و امور ایثارگران",
		Kinds: []recordkit.Kind{
			{
				Key: KindComplaints, TitleFa: "شکایت کارمند", IDPrefix: "CMP", EmployeeScoped: true,
				Fields: recordkit.Fields(
					"subject|موضوع شکایت|text|1",
					"filedDate|تاریخ ثبت|date|0",
					"channel|کانال ثبت|text|0",
					"resolution|نتیجهٔ رسیدگی|text|0",
				),
				Statuses: recordkit.Statuses(
					"received|دریافت‌شده", "investigating|در حال رسیدگی", "resolved|حل‌شده", "rejected|ردشده", "closed|بسته‌شده",
				),
				Initial: "received",
				Transitions: recordkit.Steps(
					"received|investigating,closed", "investigating|resolved,rejected,closed",
					"resolved|closed", "rejected|closed",
				),
				Actions: recordkit.Actions(
					"investigate|شروع رسیدگی|primary|investigating|0",
					"resolve|ثبت حل شکایت|primary|resolved|0",
					"reject|رد شکایت|danger|rejected|1",
					"close|بستن پرونده|ghost|closed|1",
				),
			},
			veteranRecordKind(),
			veteranServiceKind(),
		},
	}
}

// veteranRecordKind پروندهٔ ایثارگری: هم شناسنامهٔ ایثارگری کارمند است و هم جای ثبت
// امتیازهای تأییدشده (سنوات ارفاقی، افزایش سنواتی، معافیت‌ها، پایهٔ تشویقی و مرخصی).
//
// چرا امتیازها در خود پرونده ذخیره می‌شوند: موتور حقوق و پروندهٔ تحصیلی باید عدد «تصویب‌شده»
// را بخوانند، نه عدد بازمحاسبه‌شدهٔ لحظه‌ای؛ پس مقدار تأییدشده در پرونده می‌ماند و اگر سیاست
// شرکت عوض شود، پرونده‌های پیشین دست‌نخورده می‌مانند.
func veteranRecordKind() recordkit.Kind {
	return recordkit.Kind{
		Key: KindVeteranRecords, TitleFa: "پروندهٔ ایثارگری", IDPrefix: "VET", EmployeeScoped: true,
		Fields: recordkit.Fields(
			// گروه ایثارگری: کلید ماشین‌خوان (combatant/disabled/freed/martyr_family/...) یا
			// عنوان فارسی؛ پرونده‌های قدیمی همین فیلد را متنی داشته‌اند.
			"veteranType|گروه ایثارگری|text|1",
			"percent|درصد جانبازی|number|0",
			"captivityMonths|مدت اسارت (ماه)|number|0",
			"frontMonths|حضور در جبهه (ماه)|number|0",
			"relation|نسبت با شهید (پدر، مادر، همسر یا فرزند)|text|0",
			"cardNo|شمارهٔ کارت یا حکم ایثارگری|text|0",
			"issuerFa|مرجع صادرکننده|text|0",
			"issueDate|تاریخ صدور|date|0",
			// امتیازهای تأییدشده (از محاسبهٔ سرور می‌آید و کارشناس می‌تواند ویرایش کند).
			"erafaqiMonths|سنوات ارفاقی تأییدشده (ماه)|number|0",
			"extraServiceMonths|افزایش سنواتی تأییدشده (ماه)|number|0",
			"taxExemptPercent|معافیت مالیاتی تأییدشده (درصد)|number|0",
			"pensionExemptPercent|معافیت کسور بازنشستگی (درصد)|number|0",
			"educationUpgrade|یک مقطع تحصیلی بالاتر|select|0",
			"incentiveSteps|پایهٔ تشویقی تأییدشده|number|0",
			"leaveDays|مرخصی تشویقی تأییدشده (روز)|number|0",
			"benefits|سایر امتیازات|text|0",
			"calculationLog|سطرهای محاسبهٔ امتیاز|text|0",
		),
		Statuses: recordkit.Statuses(
			"draft|پیش‌نویس", "verified|بررسی‌شده", "approved|تأییدشده", "rejected|ردشده",
		),
		Initial: "draft",
		Transitions: recordkit.Steps(
			"draft|verified,rejected", "verified|approved,rejected", "rejected|draft",
		),
		Actions: recordkit.Actions(
			"verify|بررسی مدارک|primary|verified|0",
			"approve|تأیید پرونده|primary|approved|0",
			"reject|رد پرونده|danger|rejected|1",
			"reopen|بازگشت به پیش‌نویس|ghost|draft|0",
		),
		Validate: validateVeteranRecord,
		// پرونده تا پیش از تأیید نهایی ویرایش‌پذیر است؛ پس از تأیید، تغییر امتیاز باید با
		// بازگشت به پیش‌نویس و ثبت دوبارهٔ سابقه انجام شود.
		EditableStatuses: []string{"draft", "verified"},
	}
}

// veteranServiceKind تسهیلات و معرفی‌نامه‌های ایثارگران (سهمیهٔ استخدامی، تسهیلات درمانی،
// وام و مسکن): گردش‌کار صدور تا بستن، تا پیگیری مطالبات سازمان‌های حمایتی گم نشود.
func veteranServiceKind() recordkit.Kind {
	return recordkit.Kind{
		Key: KindVeteranServices, TitleFa: "تسهیلات و معرفی‌نامهٔ ایثارگری", IDPrefix: "VSRV", EmployeeScoped: true,
		Fields: recordkit.Fields(
			"serviceType|نوع خدمت|text|1",
			"organizationFa|مرجع حمایتی (بنیاد شهید و مانند آن)|text|0",
			"referenceNo|شمارهٔ نامه یا معرفی‌نامه|text|0",
			"requestDate|تاریخ درخواست|date|0",
			"amount|مبلغ تسهیلات (ریال)|money|0",
			"outcomeFa|نتیجه و پیگیری|text|0",
		),
		Statuses: recordkit.Statuses(
			"draft|پیش‌نویس", "issued|صادرشده", "closed|بسته‌شده", "rejected|ردشده",
		),
		Initial: "draft",
		Transitions: recordkit.Steps(
			"draft|issued,rejected", "issued|closed,rejected", "rejected|draft",
		),
		Actions: recordkit.Actions(
			"issue|صدور معرفی‌نامه/تسهیلات|primary|issued|0",
			"close|بستن پرونده|ghost|closed|1",
			"reject|رد درخواست|danger|rejected|1",
		),
		HistoryDate:      "requestDate",
		EditableStatuses: []string{"draft"},
	}
}

// validateVeteranRecord قاعده‌های پروندهٔ ایثارگری: هر گروه، مستندات و مبنای محاسبهٔ خودش را
// لازم دارد؛ پروندهٔ ناقص نباید وارد مرحلهٔ بررسی شود.
func validateVeteranRecord(in contracts.RecordInput) error {
	group := rules.VeteranGroupOf(in.Fields["veteranType"])
	if group == rules.VeteranGroupNone {
		return &recordkit.RuleError{Message: "گروه ایثارگری (رزمنده، جانباز، آزاده، خانوادهٔ شهید و ...) را انتخاب کنید."}
	}
	percent := numberValue(in.Fields["percent"])
	switch group {
	case rules.VeteranGroupDisabled:
		if percent <= 0 {
			return &recordkit.RuleError{Message: "برای جانباز، درصد جانبازی باید بزرگ‌تر از صفر باشد."}
		}
		if percent > 100 {
			return &recordkit.RuleError{Message: "درصد جانبازی نمی‌تواند بیشتر از ۱۰۰ باشد."}
		}
	case rules.VeteranGroupFreed:
		if numberValue(in.Fields["captivityMonths"]) <= 0 {
			return &recordkit.RuleError{Message: "برای آزاده، مدت اسارت (ماه) الزامی است."}
		}
	case rules.VeteranGroupCombatant:
		if numberValue(in.Fields["frontMonths"]) <= 0 {
			return &recordkit.RuleError{Message: "برای رزمنده، مدت حضور در جبهه (ماه) الزامی است."}
		}
	case rules.VeteranGroupMartyrFamily, rules.VeteranGroupMartyrChild:
		if strings.TrimSpace(in.Fields["relation"]) == "" && group == rules.VeteranGroupMartyrFamily {
			return &recordkit.RuleError{Message: "برای خانوادهٔ شهید، نسبت با شهید (پدر، مادر یا همسر) را ثبت کنید."}
		}
	}
	if raw := calendar.NormalizeDigits(strings.TrimSpace(in.Fields["issueDate"])); raw != "" {
		if _, err := calendar.Parse(raw); err != nil {
			return &recordkit.RuleError{Message: "تاریخ صدور کارت/حکم ایثارگری معتبر نیست (قالب YYYY/MM/DD)."}
		}
	}
	if numberValue(in.Fields["taxExemptPercent"]) > 100 || numberValue(in.Fields["pensionExemptPercent"]) > 100 {
		return &recordkit.RuleError{Message: "درصد معافیت نمی‌تواند بیشتر از ۱۰۰ باشد."}
	}
	return nil
}

// numberValue عدد یک فیلد متنی را می‌خواند (ارقام فارسی هم پذیرفته می‌شود).
func numberValue(value string) float64 {
	v := strings.TrimSpace(calendar.NormalizeDigits(value))
	v = strings.ReplaceAll(v, ",", "")
	v = strings.ReplaceAll(v, "٬", "")
	if v == "" {
		return 0
	}
	n, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0
	}
	return n
}

// Seed رکوردهای نمونهٔ سازمان.
func Seed() []contracts.RecordInput {
	return []contracts.RecordInput{
		{Kind: KindComplaints, Title: "شکایت — نحوهٔ محاسبهٔ اضافه‌کاری", EmployeeID: "EMP-007",
			Fields: map[string]string{"subject": "اختلاف در محاسبهٔ اضافه‌کاری شهریور", "filedDate": "1404/07/03", "channel": "سامانهٔ داخلی"},
			NoteFa: "در حال رسیدگی"},
		{Kind: KindComplaints, Title: "شکایت — شرایط محیط کار", EmployeeID: "EMP-004",
			Fields: map[string]string{"subject": "نبود تهویهٔ مناسب در کارگاه", "filedDate": "1404/06/18", "resolution": "رفع نقص و نصب سیستم تهویه"}},
		// جانباز ۲۵٪ تأییدشده: معافیت مالیاتی ماده ۵۶، چهار سال سنوات ارفاقی و یک مقطع بالاتر.
		{Kind: KindVeteranRecords, Title: "پروندهٔ ایثارگری — کارمند جانباز", EmployeeID: "EMP-009",
			Fields: map[string]string{
				"veteranType": rules.VeteranGroupDisabled, "percent": "25",
				"cardNo": "J-25-0142", "issuerFa": "بنیاد شهید و امور ایثارگران", "issueDate": "1389/05/12",
				"erafaqiMonths": "48", "extraServiceMonths": "48", "taxExemptPercent": "100",
				"pensionExemptPercent": "0", "educationUpgrade": "yes", "incentiveSteps": "1",
				"benefits": "معافیت مالیاتی، سنوات ارفاقی، یک مقطع تحصیلی بالاتر و مرخصی تشویقی",
			},
			SeedStatus: "approved",
			NoteFa:     "پروندهٔ تأییدشده؛ امتیازها در محاسبهٔ حقوق اعمال می‌شود"},
		// آزاده با دو سال اسارت: هر سال، دو سال سابقه و مرخصی تشویقی به نسبت مدت اسارت.
		{Kind: KindVeteranRecords, Title: "پروندهٔ ایثارگری — آزاده", EmployeeID: "EMP-005",
			Fields: map[string]string{
				"veteranType": rules.VeteranGroupFreed, "captivityMonths": "24",
				"cardNo": "A-24-0088", "issuerFa": "بنیاد شهید و امور ایثارگران", "issueDate": "1378/08/20",
				"extraServiceMonths": "48", "taxExemptPercent": "100", "educationUpgrade": "yes",
				"incentiveSteps": "1", "leaveDays": "72",
			},
			SeedStatus: "approved",
			NoteFa:     "مدت اسارت ۲۴ ماه؛ احتساب دو برابری سابقه و مرخصی تشویقی ذخیره‌شده"},
		{Kind: KindVeteranServices, Title: "معرفی‌نامهٔ تسهیلات مسکن — خانوادهٔ شهید", EmployeeID: "EMP-009",
			Fields: map[string]string{
				"serviceType": "معرفی‌نامهٔ تسهیلات مسکن", "organizationFa": "بنیاد شهید و امور ایثارگران",
				"referenceNo": "1404/م/2213", "requestDate": "1404/07/21", "amount": "3000000000",
				"outcomeFa": "به بانک عامل ارسال شد",
			},
			SeedStatus: "issued"},
	}
}
