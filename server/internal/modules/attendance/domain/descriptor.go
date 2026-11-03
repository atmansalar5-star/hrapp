// Package domain دامنهٔ ماژول حضور و غیاب، مرخصی و مأموریت.
//
// هر یک از تردد، درخواست مرخصی، مأموریت، اضافه‌کاری و نوبت‌کاری (الگوی چرخه، تخصیص
// نوبت، تعویض شیفت) یک «نوع رکورد» با گردش‌کار تأیید خودش است؛ قواعد اینجا اعلام
// می‌شود و موتور مشترک `recordkit` آن را اجرا می‌کند. قاعده‌های نوبت‌کاری در shifts.go.
package domain

import (
	"hrms/internal/contracts"
	"hrms/internal/platform/recordkit"
)

// Descriptor شناسهٔ ماژول حضور و غیاب.
func Descriptor() recordkit.Descriptor {
	return recordkit.Descriptor{
		Module:  "attendance",
		TitleFa: "حضور و غیاب، مرخصی و مأموریت",
		Kinds: []recordkit.Kind{
			{
				Key: "leaveRequests", TitleFa: "درخواست مرخصی", IDPrefix: "LV", EmployeeScoped: true,
				Fields: recordkit.Fields(
					"leaveType|نوع مرخصی|select|1",
					"fromDate|از تاریخ|date|1",
					"toDate|تا تاریخ|date|1",
					"days|تعداد روز|number|0",
				),
				Statuses: recordkit.Statuses(
					"draft|پیش‌نویس", "pending|در انتظار تأیید مدیر واحد", "unitApproved|تأیید مدیر واحد",
					"approved|تأیید نهایی منابع انسانی", "rejected|ردشده", "cancelled|لغوشده",
				),
				Initial: "draft",
				// زنجیرهٔ دومرحله‌ای مرخصی: مدیر واحد شایستگی و جانشینی را تأیید می‌کند و
				// سپس منابع انسانی سوابق و ماندهٔ مرخصی را نهایی می‌کند. این تفکیک وظایف در
				// دامنه اعلام می‌شود و موتور آن را اجبار می‌کند.
				Transitions: recordkit.Steps(
					"draft|pending,cancelled",
					"pending|unitApproved,rejected,cancelled",
					"unitApproved|approved,rejected",
					"approved|cancelled",
				),
				Actions: recordkit.Actions(
					"submit|ارسال برای تأیید|primary|pending|0|employee,hr_spec,hr_manager,line_manager",
					"endorse|تأیید مدیر واحد|primary|unitApproved|0|line_manager",
					"approve|تأیید نهایی منابع انسانی|primary|approved|0|hr_manager",
					"reject|رد درخواست|danger|rejected|1|line_manager,hr_manager",
					"cancel|لغو|ghost|cancelled|1|employee,hr_spec,hr_manager,line_manager",
				),
			},
			{
				Key: "missions", TitleFa: "مأموریت", IDPrefix: "MSN", EmployeeScoped: true,
				Fields: recordkit.Fields(
					"destination|مقصد|text|1",
					"fromDate|از تاریخ|date|1",
					"toDate|تا تاریخ|date|1",
					"purpose|هدف مأموریت|text|0",
				),
				Statuses: recordkit.Statuses(
					"draft|پیش‌نویس", "pending|در انتظار تأیید", "approved|تأییدشده", "settled|تسویه‌شده", "rejected|ردشده",
				),
				Initial: "draft",
				Transitions: recordkit.Steps(
					"draft|pending", "pending|approved,rejected", "approved|settled",
				),
				Actions: recordkit.Actions(
					"submit|ارسال برای تأیید|primary|pending|0|employee,hr_spec,hr_manager,line_manager",
					"approve|تأیید|primary|approved|0|hr_manager,line_manager",
					"settle|ثبت تسویه|ghost|settled|0|hr_manager",
					"reject|رد درخواست|danger|rejected|1|hr_manager,line_manager",
				),
			},
			{
				Key: "overtime", TitleFa: "اضافه‌کاری", IDPrefix: "OT", EmployeeScoped: true,
				Fields: recordkit.Fields(
					"date|تاریخ|date|1",
					"hours|ساعت اضافه‌کاری|number|1",
					"rate|ضریب|number|0",
				),
				Statuses: recordkit.Statuses(
					"draft|ثبت‌شده", "approved|تأییدشده", "rejected|ردشده",
				),
				Initial:     "draft",
				Transitions: recordkit.Steps("draft|approved,rejected"),
				Actions: recordkit.Actions(
					"approve|تأیید|primary|approved|0|hr_manager,line_manager",
					"reject|رد|danger|rejected|1|hr_manager,line_manager",
				),
			},
			{
				Key: "attendanceRecords", TitleFa: "تردد روزانه", IDPrefix: "ATT", EmployeeScoped: true,
				Fields: recordkit.Fields(
					"date|تاریخ|date|1",
					"checkIn|ورود|text|0",
					"checkOut|خروج|text|0",
					"workedHours|ساعت کارکرد|number|0",
					// فیلدهای زیر را «ورود فایل دستگاه تردد» پر می‌کند: نوبت همان روز از
					// چرخهٔ مصوب حل می‌شود و تأخیر/تعجیل/شب‌کاری/جمعه‌کاری در دامنه سنجیده
					// و همین‌جا ذخیره می‌شود تا کارکرد دوره و تطبیق حقوق از همان بخوانند.
					"shift|نوبت روز|text|0",
					"lateMinutes|تأخیر (دقیقه)|number|0",
					"earlyMinutes|تعجیل (دقیقه)|number|0",
					"nightHours|ساعت شب‌کاری|number|0",
					"fridayHours|ساعت جمعه/تعطیل‌کاری|number|0",
					"deviceCode|کد دستگاه تردد|text|0",
					"sourceFile|فایل مبدأ|text|0",
				),
				Statuses: recordkit.Statuses(
					"recorded|ثبت‌شده", "confirmed|تأییدشده", "disputed|دارای اعتراض",
				),
				Initial:     "recorded",
				Transitions: recordkit.Steps("recorded|confirmed,disputed"),
				Actions: recordkit.Actions(
					"confirm|تأیید تردد|primary|confirmed|0",
					"dispute|ثبت اعتراض|danger|disputed|1",
				),
			},
			{
				// سند گردش ورود فایل تردد: چه فایلی، چند خط، چند رکورد ساخته/ویرایش شد و
				// چند ردیف ثبت نشد. برای ردیابی «این تردد از کدام فایل آمده» نگه داشته می‌شود.
				Key: "deviceImports", TitleFa: "ورود فایل دستگاه تردد", IDPrefix: "DVI", EmployeeScoped: false,
				Fields: recordkit.Fields(
					"fileName|نام فایل|text|0",
					"importedAt|تاریخ ورود|text|0",
					"lines|خطوط فایل|number|0",
					"accepted|ردیف پذیرفته|number|0",
					"rejected|ردیف ردشده|number|0",
					"duplicate|ردیف تکراری|number|0",
					"created|رکورد ساخته‌شده|number|0",
					"updated|رکورد ویرایش‌شده|number|0",
					"skipped|ردیف ثبت‌نشده|number|0",
				),
				Statuses: recordkit.Statuses(
					"recorded|ثبت‌شده", "confirmed|تأییدشده توسط کارگزینی",
				),
				Initial:     "recorded",
				Transitions: recordkit.Steps("recorded|confirmed"),
				Actions: recordkit.Actions(
					"confirm|تأیید فایل|primary|confirmed|0|hr_spec,hr_manager",
				),
			},
			{
				Key: "shiftPatterns", TitleFa: "الگوی چرخهٔ نوبت", IDPrefix: "SHP", EmployeeScoped: false,
				Fields: recordkit.Fields(
					"rotation|ترتیب چرخه (کلیدهای شیفت جداشده با «،»)|text|1",
					"minCoverage|حداقل پوشش هر شیفت (نفر)|number|0",
				),
				Statuses: recordkit.Statuses(
					"draft|پیش‌نویس", "active|فعال", "retired|بازنشسته",
				),
				Initial: "draft",
				// چرخهٔ نوبت دادهٔ پایه است: تا فعال‌سازی مرجع تخصیص نیست و بازنشسته‌سازی فقط
				// تخصیص‌های تازه را از انتخاب بیرون می‌برد؛ تخصیص‌های قبلی از بین نمی‌روند.
				Transitions: recordkit.Steps(
					"draft|active", "active|retired", "retired|active",
				),
				Actions: recordkit.Actions(
					"activate|فعال‌سازی الگو|primary|active|0|hr_manager",
					"retire|بازنشسته‌سازی الگو|ghost|retired|1|hr_manager",
					"restore|بازگردانی الگو|primary|active|0|hr_manager",
				),
				Validate: validateShiftPattern,
			},
			{
				Key: "shiftAssignments", TitleFa: "تخصیص نوبت", IDPrefix: "SHA", EmployeeScoped: true,
				Fields: recordkit.Fields(
					"pattern|الگوی چرخه (عنوان)|text|1",
					"startDate|تاریخ شروع (مبنای چرخه)|date|1",
					"endDate|تاریخ پایان|date|0",
					"cycleOffset|جایگاه روز شروع در چرخه|number|0",
				),
				Statuses: recordkit.Statuses(
					"draft|پیش‌نویس", "pending|در انتظار تأیید مدیر واحد", "approved|مصوب",
					"ended|خاتمه‌یافته", "rejected|ردشده", "cancelled|لغوشده",
				),
				Initial: "draft",
				Transitions: recordkit.Steps(
					"draft|pending,cancelled",
					"pending|approved,rejected,cancelled",
					"approved|ended,cancelled",
				),
				Actions: recordkit.Actions(
					"submit|ارسال برای تأیید|primary|pending|0|employee,hr_spec,hr_manager,line_manager",
					"approve|تأیید تخصیص نوبت|primary|approved|0|line_manager,hr_manager",
					"end|خاتمهٔ تخصیص|ghost|ended|0|hr_manager",
					"reject|رد تخصیص|danger|rejected|1|line_manager,hr_manager",
					"cancel|لغو|ghost|cancelled|1|employee,hr_spec,hr_manager,line_manager",
				),
				Validate: validateShiftAssignment,
				// تاریخ مؤثر سابقهٔ کارمند همان آغاز چرخه است؛ تا این تاریخ نوبتی معلق نمی‌شود.
				HistoryDate: "startDate",
			},
			{
				Key: "shiftSwaps", TitleFa: "تعویض نوبت", IDPrefix: "SSW", EmployeeScoped: true,
				Fields: recordkit.Fields(
					"swapDate|تاریخ تعویض|date|1",
					"partnerEmployee|شناسهٔ کارمند مقابل|text|1",
					"myShift|شیفت من در آن روز|text|0",
					"partnerShift|شیفت او در آن روز|text|0",
					"reason|دلیل تعویض|text|0",
				),
				Statuses: recordkit.Statuses(
					"draft|پیش‌نویس", "pending|در انتظار تأیید مدیر واحد", "approved|تأییدشده",
					"rejected|ردشده", "cancelled|لغوشده",
				),
				Initial: "draft",
				Transitions: recordkit.Steps(
					"draft|pending,cancelled",
					"pending|approved,rejected,cancelled",
				),
				Actions: recordkit.Actions(
					"submit|ارسال برای تأیید|primary|pending|0|employee,hr_spec,hr_manager,line_manager",
					"approve|تأیید تعویض|primary|approved|0|line_manager,hr_manager",
					"reject|رد تعویض|danger|rejected|1|line_manager,hr_manager",
					"cancel|لغو|ghost|cancelled|1|employee,hr_spec,hr_manager,line_manager",
				),
				Validate:    validateShiftSwap,
				HistoryDate: "swapDate",
			},
		},
	}
}

// Seed رکوردهای نمونهٔ سازمان.
func Seed() []contracts.RecordInput {
	return []contracts.RecordInput{
		{Kind: "leaveRequests", Title: "مرخصی استحقاقی — سارا محمدی", EmployeeID: "EMP-001",
			Fields: map[string]string{"leaveType": "استحقاقی", "fromDate": "1404/07/12", "toDate": "1404/07/15", "days": "4"}},
		{Kind: "leaveRequests", Title: "مرخصی بدون حقوق — نگار حسینی", EmployeeID: "EMP-003",
			Fields: map[string]string{"leaveType": "بدون حقوق", "fromDate": "1404/07/20", "toDate": "1404/07/22", "days": "3"},
			NoteFa: "در انتظار تأیید مدیر واحد"},
		{Kind: "missions", Title: "مأموریت اداری — محمد احمدی", EmployeeID: "EMP-005",
			Fields: map[string]string{"destination": "تهران", "fromDate": "1404/06/05", "toDate": "1404/06/07", "purpose": "جلسهٔ هماهنگی ستاد"}},
		{Kind: "overtime", Title: "اضافه‌کاری مهر — مریم نوری", EmployeeID: "EMP-008",
			Fields: map[string]string{"date": "1404/07/08", "hours": "12", "rate": "1.4"}},
		{Kind: "attendanceRecords", Title: "تردد — احمدی", EmployeeID: "EMP-005",
			Fields: map[string]string{"date": "1404/07/09", "checkIn": "07:55", "checkOut": "16:30", "workedHours": "8.5"}},
		{Kind: "shiftPatterns", Title: "الگوی چهارشیفتی اورژانس",
			Fields: map[string]string{"rotation": "صبح,عصر,شب,استراحت", "minCoverage": "2"},
			NoteFa: "چرخهٔ چهارروزهٔ بخش اورژانس؛ مرجع تخصیص نوبت پس از فعال‌سازی."},
		{Kind: "shiftAssignments", Title: "تخصیص نوبت — مریم نوری", EmployeeID: "EMP-008",
			Fields: map[string]string{"pattern": "الگوی چهارشیفتی اورژانس", "startDate": "1404/07/01", "cycleOffset": "0"}},
		// سند ورود فایل دستگاه تردد: نمونهٔ آزمایشی «موقت» تا این نوع رکورد در استقرار
		// تازه بی‌داده نماند و آزمون دود Postgres (که برای هر نوع رکورد یک رکورد نمونه
		// می‌خواهد) سبز شود. با ورود فایل واقعی دستگاه، این سند در فهرست کنار آن می‌ماند
		// و کاملاً قابل حذف است.
		{Kind: "deviceImports", Title: "ورود فایل تردد — مهر ۱۴۰۴",
			Fields: map[string]string{
				"fileName": "device-1404-07.csv", "importedAt": "1404/07/30",
				"lines": "1240", "accepted": "1218", "rejected": "22", "duplicate": "6",
				"created": "980", "updated": "238", "skipped": "22",
			},
			NoteFa: "نمونهٔ موقت ورود فایل دستگاه تردد؛ با فایل واقعی جایگزین یا حذف شود."},
		{Kind: "shiftSwaps", Title: "تعویض نوبت — احمدی با نوری", EmployeeID: "EMP-005",
			Fields: map[string]string{"swapDate": "1404/07/10", "partnerEmployee": "EMP-008",
				"myShift": "صبح", "partnerShift": "عصر", "reason": "علت خانوادگی"}},
	}
}
