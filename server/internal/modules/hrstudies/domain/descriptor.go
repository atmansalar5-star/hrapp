// Package domain دامنهٔ ماژول مطالعات منابع انسانی.
//
// پروژهٔ مطالعاتی، نظرسنجی و پیشنهاد کارکنان هر یک یک «نوع رکورد» با گردش‌کار خودش است.
package domain

import (
	"hrms/internal/contracts"
	"hrms/internal/platform/recordkit"
)

// Descriptor شناسهٔ ماژول مطالعات منابع انسانی.
func Descriptor() recordkit.Descriptor {
	return recordkit.Descriptor{
		Module:  "hrstudies",
		TitleFa: "مطالعات منابع انسانی",
		Kinds: []recordkit.Kind{
			{
				Key: "studies", TitleFa: "پروژهٔ مطالعاتی", IDPrefix: "STD", EmployeeScoped: false,
				Fields: recordkit.Fields(
					"topic|موضوع پژوهش|text|1",
					"startDate|تاریخ شروع|date|0",
					"endDate|تاریخ پایان|date|0",
					"researcher|پژوهشگر|text|0",
				),
				Statuses:    recordkit.Statuses("draft|پیش‌نویس", "running|در حال اجرا", "completed|تمام‌شده", "archived|بایگانی‌شده"),
				Initial:     "draft",
				Transitions: recordkit.Steps("draft|running", "running|completed", "completed|archived"),
				Actions: recordkit.Actions(
					"start|آغاز پژوهش|primary|running|0",
					"complete|پایان پژوهش|primary|completed|0",
					"archive|بایگانی|ghost|archived|1",
				),
			},
			{
				Key: "surveys", TitleFa: "نظرسنجی", IDPrefix: "SRV", EmployeeScoped: false,
				Fields: recordkit.Fields(
					"title|عنوان نظرسنجی|text|1",
					"startDate|تاریخ شروع|date|0",
					"endDate|تاریخ پایان|date|0",
					"participants|تعداد شرکت‌کننده|number|0",
					"population|جامعهٔ آماری (نفر)|number|0",
					"satisfactionIndex|شاخص رضایت (٪)|number|0",
					"weakAxis|محور ضعیف|text|0",
					"actionsFa|اقدام‌های اصلاحی|text|0",
				),
				Statuses:    recordkit.Statuses("draft|پیش‌نویس", "open|در حال اجرا", "closed|بسته‌شده"),
				Initial:     "draft",
				Transitions: recordkit.Steps("draft|open", "open|closed"),
				Actions: recordkit.Actions(
					"open|آغاز نظرسنجی|primary|open|0",
					"close|بستن نظرسنجی|ghost|closed|0",
				),
			},
			{
				// شاخص سرمایهٔ انسانی: هر شاخص یک رکورد است که فرمول، هدف، جهت مطلوب و
				// مقدار جاری دوره را نگه می‌دارد. سنجش «در محدودهٔ هدف / خارج از هدف»
				// قاعدهٔ دامنه است (domain.EvaluateIndicator) و در مسیر /indices اعمال
				// می‌شود؛ پس رابط هیچ آستانهٔ سخت‌شده‌ای ندارد.
				Key: "hrIndices", TitleFa: "شاخص سرمایهٔ انسانی", IDPrefix: "IND", EmployeeScoped: false,
				Fields: recordkit.Fields(
					"formula|فرمول محاسبه|text|1",
					"target|مقدار هدف|number|1",
					"current|مقدار جاری دوره|number|1",
					"direction|جهت مطلوب|select|1",
					"unit|واحد سنجش|text|0",
					"period|دورهٔ سنجش|text|0",
					"owner|مالک شاخص|text|0",
				),
				// قاعدهٔ اختصاصی همین نوع: جهت مطلوب باید یکی از دو مقدار اعلامی باشد تا
				// سنجش «در محدودهٔ هدف» هرگز وارونه محاسبه نشود.
				Validate: func(in contracts.RecordInput) error {
					if !ValidIndicatorDirection(in.Fields["direction"]) {
						return &recordkit.RuleError{Message: "جهت مطلوب شاخص باید «بیشتر بهتر» (higher) یا «کمتر بهتر» (lower) باشد."}
					}
					return nil
				},
				Statuses: recordkit.Statuses(
					"active|فعال", "under_review|نیازمند بازنگری تعریف", "archived|بایگانی‌شده"),
				Initial:     "active",
				Transitions: recordkit.Steps("active|under_review,archived", "under_review|active,archived"),
				Actions: recordkit.Actions(
					"review|نیازمند بازنگری تعریف|danger|under_review|1",
					"activate|فعال‌سازی|primary|active|0",
					"archive|بایگانی|ghost|archived|1",
				),
			},
			{
				Key: "suggestions", TitleFa: "پیشنهاد کارکنان", IDPrefix: "SUG", EmployeeScoped: true,
				Fields: recordkit.Fields(
					"subject|موضوع پیشنهاد|text|1",
					"submitDate|تاریخ ثبت|date|0",
					"evaluation|ارزیابی|text|0",
				),
				Statuses:    recordkit.Statuses("submitted|ثبت‌شده", "reviewing|در حال بررسی", "accepted|پذیرفته‌شده", "rejected|ردشده"),
				Initial:     "submitted",
				Transitions: recordkit.Steps("submitted|reviewing", "reviewing|accepted,rejected"),
				Actions: recordkit.Actions(
					"review|شروع بررسی|primary|reviewing|0",
					"accept|پذیرش پیشنهاد|primary|accepted|0",
					"reject|رد پیشنهاد|danger|rejected|1",
				),
			},
		},
	}
}

// Seed رکوردهای نمونهٔ سازمان.
//
// نظرسنجی‌ها و شاخص‌ها عمداً چند وضعیت و چند حالت سنجش را پوشش می‌دهند: یک شاخص در
// هدف، یکی نزدیک هدف و یکی خارج از هدف؛ یک نظرسنجی مشارکت خوب، یکی قابل قبول و یکی
// ضعیف. اگر همه در یک حالت بودند، صفحهٔ تحلیل خالی چشم می‌زد و آزمون هم چیزی نمی‌سنجید.
func Seed() []contracts.RecordInput {
	return []contracts.RecordInput{
		{Kind: "studies", Title: "بررسی علل ترک خدمت کارکنان فنی",
			Fields: map[string]string{"topic": "علل ترک خدمت کارکنان فنی", "startDate": "1404/05/01", "researcher": "کارشناس مطالعات"},
			NoteFa: "در حال اجرا"},

		// نظرسنجی‌ها: دوره، جامعهٔ آماری، مشارکت، شاخص رضایت، محور ضعیف و اقدام اصلاحی.
		{Kind: "surveys", Title: "نظرسنجی جو سازمانی ۱۴۰۴",
			SeedStatus: "closed",
			Fields: map[string]string{"title": "جو سازمانی ۱۴۰۴", "startDate": "1404/06/01", "endDate": "1404/06/20",
				"participants": "312", "population": "380", "satisfactionIndex": "74", "weakAxis": "ارتباط با سرپرست",
				"actionsFa": "دورهٔ مهارت‌های ارتباطی سرپرستان"}},
		{Kind: "surveys", Title: "نظرسنجی رضایت از سرویس ایاب و ذهاب",
			SeedStatus: "open",
			Fields: map[string]string{"title": "رضایت از سرویس ایاب و ذهاب", "startDate": "1404/06/25", "endDate": "1404/07/10",
				"participants": "156", "population": "240", "satisfactionIndex": "61", "weakAxis": "زمان‌بندی سرویس",
				"actionsFa": "بازنگری ساعت حرکت سرویس شیفتی"}},
		{Kind: "surveys", Title: "نظرسنجی نظام پیشنهادها",
			SeedStatus: "closed",
			Fields: map[string]string{"title": "نظام پیشنهادها", "startDate": "1404/05/10", "endDate": "1404/05/25",
				"participants": "270", "population": "380", "satisfactionIndex": "79", "weakAxis": "سرعت پاسخ‌دهی",
				"actionsFa": "تعیین مهلت ۱۵ روزهٔ پاسخ به پیشنهاد"}},
		{Kind: "surveys", Title: "نظرسنجی بازنشستگی و رضایت شغلی",
			Fields: map[string]string{"title": "بازنشستگی و رضایت شغلی", "startDate": "1404/10/01",
				"population": "95", "participants": "0", "weakAxis": "—"},
			NoteFa: "در حال طراحی پرسش‌نامه"},

		// شاخص‌های سرمایهٔ انسانی: فرمول، هدف، مقدار جاری و جهت مطلوب.
		{Kind: "hrIndices", Title: "نرخ ترک خدمت",
			Fields: map[string]string{"formula": "خروج ÷ میانگین سرشماری", "target": "5", "current": "4.2",
				"direction": "lower", "unit": "٪", "period": "1404/06", "owner": "برنامه‌ریزی نیروی انسانی"}},
		{Kind: "hrIndices", Title: "هزینهٔ جبران خدمات به ازای هر کارمند",
			Fields: map[string]string{"formula": "جمع هزینه ÷ سرشماری", "target": "648000000", "current": "760000000",
				"direction": "lower", "unit": "ریال", "period": "1404/06", "owner": "امور کارکنان"}},
		{Kind: "hrIndices", Title: "ساعت آموزش به ازای هر کارمند",
			Fields: map[string]string{"formula": "جمع ساعت آموزش ÷ سرشماری", "target": "24", "current": "27",
				"direction": "higher", "unit": "ساعت", "period": "1404/06", "owner": "هماهنگی آموزش"}},
		{Kind: "hrIndices", Title: "نرخ تصدی پست‌های کلیدی",
			Fields: map[string]string{"formula": "پست‌های دارای تصدی ÷ پست‌های کلیدی", "target": "100", "current": "88",
				"direction": "higher", "unit": "٪", "period": "1404/06", "owner": "برنامه‌ریزی نیروی انسانی"}},
		{Kind: "hrIndices", Title: "نرخ تحقق برنامه‌های آموزشی",
			Fields: map[string]string{"formula": "دوره‌های اجراشده ÷ دوره‌های مصوب", "target": "90", "current": "68",
				"direction": "higher", "unit": "٪", "period": "1404/06", "owner": "هماهنگی آموزش"}},
		{Kind: "hrIndices", Title: "شاخص رضایت کارکنان",
			Fields: map[string]string{"formula": "میانگین شاخص رضایت نظرسنجی‌های دوره", "target": "75", "current": "74",
				"direction": "higher", "unit": "٪", "period": "1404/06", "owner": "واحد مطالعات"},
			SeedStatus: "under_review", NoteFa: "نیازمند بازنگری تعریف: وزن محورهای نظرسنجی"},

		{Kind: "suggestions", Title: "پیشنهاد بهبود چیدمان خط تولید", EmployeeID: "EMP-005",
			Fields: map[string]string{"subject": "بهبود چیدمان خط تولید", "submitDate": "1404/06/22", "evaluation": "صرفه‌جویی ۱۲٪ در زمان"},
			NoteFa: "در حال بررسی"},
	}
}
