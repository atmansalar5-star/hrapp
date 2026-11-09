// Package domain دامنهٔ ماژول درمان، معاینات دوره‌ای و خدمات درمانی.
//
// پروندهٔ درمان، معاینهٔ دوره‌ای و خدمت درمانی هر یک یک «نوع رکورد» با گردش‌کار خودش است.
package domain

import (
	"strconv"
	"strings"

	"hrms/internal/contracts"
	"hrms/internal/platform/calendar"
	"hrms/internal/platform/recordkit"
	"hrms/internal/rules"
)

// Descriptor شناسهٔ ماژول درمان و خدمات درمانی.
func Descriptor() recordkit.Descriptor {
	return recordkit.Descriptor{
		Module:  "medical",
		TitleFa: "درمان، معاینات دوره‌ای و خدمات درمانی",
		Kinds: []recordkit.Kind{
			{
				Key: "treatmentCases", TitleFa: "پروندهٔ درمان", IDPrefix: "MED", EmployeeScoped: true,
				Fields: recordkit.Fields(
					"diagnosis|تشخیص|text|1",
					"visitDate|تاریخ مراجعه|date|0",
					"provider|پزشک/مرکز|text|0",
					"prescription|نسخه|text|0",
				),
				Statuses:    recordkit.Statuses("open|باز", "treating|در حال درمان", "closed|بسته‌شده"),
				Initial:     "open",
				Transitions: recordkit.Steps("open|treating,closed", "treating|closed"),
				Actions: recordkit.Actions(
					"treat|ثبت درمان|primary|treating|0",
					"close|بستن پرونده|ghost|closed|0",
				),
			},
			{
				Key: "periodicExams", TitleFa: "معاینهٔ دوره‌ای", IDPrefix: "EXM", EmployeeScoped: true,
				Fields: recordkit.Fields(
					"examDate|تاریخ معاینه|date|1",
					"examType|نوع معاینه|text|0",
					"result|نتیجه|select|0",
					"nextDate|معاینهٔ بعدی|date|0",
				),
				Statuses:    recordkit.Statuses("scheduled|زمان‌بندی‌شده", "done|انجام‌شده", "passed|سالم", "failed|نیازمند پیگیری"),
				Initial:     "scheduled",
				Transitions: recordkit.Steps("scheduled|done", "done|passed,failed"),
				Actions: recordkit.Actions(
					"record|ثبت انجام معاینه|primary|done|0",
					"pass|تأیید سلامتی|primary|passed|0",
					"followup|نیازمند پیگیری|danger|failed|1",
				),
			},
			{
				Key: "supplementaryInsurance", TitleFa: "بیمهٔ تکمیلی", IDPrefix: "INS", EmployeeScoped: true,
				Fields: recordkit.Fields(
					"policyNo|شمارهٔ بیمه‌نامه|text|0",
					"insurer|شرکت بیمه‌گر|text|0",
					"coverage|درصد پوشش|number|0",
					"dependents|تعداد افراد تحت تکفل|number|0",
					"premium|حق بیمهٔ ماهانه|money|0",
					"startDate|آغاز اعتبار|date|0",
					"endDate|پایان اعتبار|date|0",
				),
				Statuses:    recordkit.Statuses("active|فعال", "suspended|معلق", "expired|منقضی"),
				Initial:     "active",
				Transitions: recordkit.Steps("active|suspended,expired", "suspended|active", "expired|active"),
				Actions: recordkit.Actions(
					"renew|تمدید اعتبار|primary|active|0",
					"activate|فعال‌سازی|primary|active|0",
					"suspend|تعلیق پوشش|danger|suspended|1",
				),
				EditableStatuses: []string{"active", "suspended", "expired"},
			},
			{
				Key: "medicalServices", TitleFa: "خدمت درمانی", IDPrefix: "MSV", EmployeeScoped: false,
				Fields: recordkit.Fields(
					"serviceName|نام خدمت|text|1",
					"coverage|پوشش بیمه|text|0",
					"cost|هزینه|money|0",
				),
				Statuses:    recordkit.Statuses("draft|پیش‌نویس", "approved|تأییدشده"),
				Initial:     "draft",
				Transitions: recordkit.Steps("draft|approved"),
				Actions: recordkit.Actions(
					"approve|تأیید خدمت|primary|approved|0",
				),
			},
			{
				Key: KindTariffs, TitleFa: "تعرفهٔ کمک هزینهٔ درمانی", IDPrefix: "MTAR", EmployeeScoped: false,
				Fields: recordkit.Fields(
					"code|کد قلم تعرفه|text|1",
					"category|دستهٔ کمک هزینه|select|1",
					"unitFa|واحد تعرفه|text|0",
					"amount|تعرفهٔ مصوب (ریال)|money|1",
					"ceilingAmount|سقف پرداخت قلم (ریال)|money|0",
					"orgSharePercent|سهم سازمان (درصد)|number|0",
					"insuranceSharePercent|سهم بیمه از پرداخت (درصد)|number|0",
					"requirePrescription|نیازمند نسخهٔ تأییدشده|select|0",
					"effectiveFrom|تاریخ اجرا|date|1",
					"refFa|ارجاع دستورالعمل (بند/جدول)|text|0",
				),
				Statuses:    recordkit.Statuses("draft|پیش‌نویس", "active|جاری", "inactive|غیرفعال"),
				Initial:     "draft",
				Transitions: recordkit.Steps("draft|active", "active|inactive", "inactive|active"),
				Actions: recordkit.Actions(
					"activate|جاری‌سازی تعرفه|primary|active|0",
					"deactivate|غیرفعال‌سازی|ghost|inactive|1",
				),
				Validate:         validateTariff,
				EditableStatuses: []string{"draft", "active", "inactive"},
			},
			{
				Key: KindClaims, TitleFa: "درخواست کمک هزینهٔ درمانی", IDPrefix: "MAID", EmployeeScoped: true,
				Fields: recordkit.Fields(
					"category|دستهٔ کمک هزینه|select|1",
					"periodYear|سال کمک هزینه (شمسی)|number|0",
					"invoiceDate|تاریخ فاکتور|date|1",
					"invoiceNo|شمارهٔ فاکتور|text|0",
					"provider|مرکز درمانی / داروخانه / پزشک|text|0",
					"prescriptionNo|شمارهٔ نسخه یا معرفی‌نامه|text|0",
					"prescriptionApproved|نسخه تأیید شده است|select|0",
					// اقلام درخواست: هر قلم «کد تعرفه|تعداد|مبلغ فاکتور»، جداکنندهٔ قلم‌ها «؛».
					// سرور از همین قلم‌ها، تعرفه و سقف را می‌خواند و مبالغ را می‌نویسد.
					"items|اقلام درخواست (کد تعرفه|تعداد|مبلغ فاکتور)|text|1",
					"invoiceTotal|جمع مبلغ فاکتور (ریال)|money|0",
					"totalBase|مبنای محاسبه پس از سقف (ریال)|money|0",
					"payableAmount|سهم سازمان (ریال)|money|0",
					"employeeAmount|سهم کارمند (ریال)|money|0",
					"insuranceAmount|سهم بیمه از پرداخت (ریال)|money|0",
					"ceilingApplied|کسر سقف قلم و سالانه (ریال)|money|0",
					"annualPaidBefore|پرداخت پیشین همان سال (ریال)|money|0",
					"annualPaidAfter|جمع پرداخت سال پس از این درخواست|money|0",
					"calculationLog|سطرهای محاسبه|text|0",
					"paymentRef|شمارهٔ سند پرداخت|text|0",
					"rejectReason|دلیل رد|text|0",
				),
				Statuses: recordkit.Statuses(
					"draft|پیش‌نویس", "submitted|ارسال‌شده", "verified|تأیید کارشناس",
					"approved|تصویب‌شده", "paid|پرداخت‌شده", "rejected|ردشده",
				),
				Initial: "draft",
				Transitions: recordkit.Steps(
					"draft|submitted,cancelled", "submitted|verified,rejected", "verified|approved,rejected",
					"approved|paid", "rejected|draft",
				),
				Actions: recordkit.Actions(
					"submit|ارسال برای بررسی|primary|submitted|0",
					"verify|تأیید کارشناس|primary|verified|0",
					"approve|تصویب نهایی|primary|approved|0",
					"reject|رد درخواست|danger|rejected|1",
					"pay|ثبت پرداخت|primary|paid|0",
					"reopen|بازگشت به پیش‌نویس|ghost|draft|1",
				),
				HistoryDate:      "invoiceDate",
				EditableStatuses: []string{"draft"},
				ActRule:          claimRule,
			},
		},
	}
}

// کلیدهای نوع رکورد کمک هزینهٔ درمانی.
const (
	// KindTariffs فهرست تعرفه‌های مصوب کمک هزینهٔ درمانی.
	KindTariffs = "medicalTariffs"
	// KindClaims درخواست‌های کمک هزینهٔ درمانی کارکنان.
	KindClaims = "medicalAidClaims"
)

// validateTariff قاعدهٔ ردیف تعرفه: دسته باید شناخته‌شده باشد، مبلغ مثبت و درصدها در بازهٔ
// صفر تا یک (یا صفر تا صد که به نسبت تبدیل می‌شود)، و تاریخ اجرا معتبر باشد.
func validateTariff(in contracts.RecordInput) error {
	key := strings.TrimSpace(in.Fields["category"])
	if key == "" {
		return &recordkit.ValidationError{Details: map[string]string{
			"fields.category": "دستهٔ کمک هزینه را انتخاب کنید.",
		}}
	}
	known := false
	for _, c := range rules.MedicalAidCategories() {
		if c.Key == key {
			known = true
			break
		}
	}
	if !known {
		return &recordkit.ValidationError{Details: map[string]string{
			"fields.category": "دستهٔ کمک هزینه در فهرست دستورالعمل نیست.",
		}}
	}
	if code := strings.TrimSpace(in.Fields["code"]); code == "" {
		return &recordkit.ValidationError{Details: map[string]string{
			"fields.code": "کد قلم تعرفه الزامی است (شناسهٔ ماندگار قلم).",
		}}
	}
	amount, err := strconv.ParseInt(calendar.NormalizeDigits(strings.TrimSpace(in.Fields["amount"])), 10, 64)
	if err != nil || amount <= 0 {
		return &recordkit.ValidationError{Details: map[string]string{
			"fields.amount": "تعرفهٔ مصوب باید عددی بزرگ‌تر از صفر باشد.",
		}}
	}
	if raw := strings.TrimSpace(in.Fields["ceilingAmount"]); raw != "" {
		if ceiling, err := strconv.ParseInt(calendar.NormalizeDigits(raw), 10, 64); err != nil || ceiling < 0 {
			return &recordkit.ValidationError{Details: map[string]string{
				"fields.ceilingAmount": "سقف پرداخت قلم باید عدد غیرمنفی باشد.",
			}}
		}
	}
	for key, raw := range map[string]string{
		"orgSharePercent":       in.Fields["orgSharePercent"],
		"insuranceSharePercent": in.Fields["insuranceSharePercent"],
	} {
		value := strings.TrimSpace(raw)
		if value == "" {
			continue
		}
		percent, err := strconv.ParseFloat(calendar.NormalizeDigits(value), 64)
		if err != nil || percent < 0 || percent > 100 {
			return &recordkit.ValidationError{Details: map[string]string{
				"fields." + key: "درصد باید عددی بین صفر تا صد باشد (مثلاً ۷۰).",
			}}
		}
	}
	if from := strings.TrimSpace(in.Fields["effectiveFrom"]); from != "" {
		if _, err := calendar.Parse(from); err != nil {
			return &recordkit.ValidationError{Details: map[string]string{
				"fields.effectiveFrom": "تاریخ اجرا باید به قالب شمسی YYYY/MM/DD باشد.",
			}}
		}
	}
	return nil
}

// claimRule قاعدهٔ اقدام‌های درخواست کمک هزینه.
//
// چرا در دامنه: «ارسال بدون قلم یا بدون فاکتور»، «تصویب بی‌بررسی کارشناس» و «پرداخت
// بی‌سهم مصوب» قاعدهٔ سازمان است؛ موتور رکورد فقط صدایش می‌زند.
func claimRule(rec recordkit.Record, action string) error {
	switch action {
	case "submit":
		if strings.TrimSpace(rec.Fields["items"]) == "" {
			return &recordkit.RuleError{Message: "اقلام درخواست را ثبت کنید؛ درخواست بدون قلم قابل بررسی نیست."}
		}
		if strings.TrimSpace(rec.Fields["invoiceDate"]) == "" {
			return &recordkit.RuleError{Message: "تاریخ فاکتور را ثبت کنید."}
		}
		if strings.TrimSpace(rec.Fields["payableAmount"]) == "" {
			return &recordkit.RuleError{Message: "پیش از ارسال، محاسبه را روی درخواست اجرا کنید."}
		}
	case "approve":
		if strings.TrimSpace(rec.Fields["payableAmount"]) == "" {
			return &recordkit.RuleError{Message: "درخواست بدون مبلغ مصوب قابل تصویب نیست."}
		}
	case "pay":
		if strings.TrimSpace(rec.Fields["paymentRef"]) == "" {
			return &recordkit.RuleError{Message: "شمارهٔ سند پرداخت را ثبت کنید؛ پرداخت بی‌سند در سامانه ثبت نمی‌شود."}
		}
	}
	return nil
}

// Seed رکوردهای نمونهٔ سازمان.
func Seed() []contracts.RecordInput {
	out := []contracts.RecordInput{
		{Kind: "treatmentCases", Title: "پروندهٔ درمان — نگار حسینی", EmployeeID: "EMP-003",
			Fields: map[string]string{"diagnosis": "کمردرد ناشی از کار", "visitDate": "1404/06/12", "provider": "درمانگاه سازمان", "prescription": "فیزیوتراپی ۱۰ جلسه"},
			NoteFa: "در حال درمان"},
		{Kind: "periodicExams", Title: "معاینهٔ دوره‌ای — محمد احمدی", EmployeeID: "EMP-005",
			Fields: map[string]string{"examDate": "1404/05/20", "examType": "معاینهٔ شغلی سالانه", "result": "سالم", "nextDate": "1405/05/20"},
			NoteFa: "معاینه انجام و تأیید شد"},
		{Kind: "medicalServices", Title: "خدمت درمانی — دندان‌پزشکی",
			Fields: map[string]string{"serviceName": "دندان‌پزشکی", "coverage": "۷۰٪ بیمهٔ تکمیلی", "cost": "4500000"}},
		{Kind: "supplementaryInsurance", Title: "بیمهٔ تکمیلی — محمد احمدی", EmployeeID: "EMP-005",
			Fields: map[string]string{"policyNo": "SUP-1404-00182", "insurer": "بیمهٔ دانا", "coverage": "80", "dependents": "4", "premium": "3200000", "startDate": "1404/01/01", "endDate": "1404/12/29"},
			NoteFa: "پوشش فعال کارمند و خانواده"},
		{Kind: "supplementaryInsurance", Title: "بیمهٔ تکمیلی — نگار حسینی", EmployeeID: "EMP-003",
			Fields: map[string]string{"policyNo": "SUP-1404-00097", "insurer": "بیمهٔ ایران", "coverage": "70", "dependents": "1", "premium": "2100000", "startDate": "1404/02/15", "endDate": "1405/02/14"},
			NoteFa: "در انتظار تمدید"},
	}
	// تعرفه‌های کمک هزینهٔ درمانی ۱۴۰۵ از فایل دادهٔ برون‌استخراج‌شدهٔ دستورالعمل می‌آید تا
	// فهرست تعرفه از روز نخست پر باشد و کاربر فقط تعرفه‌های تازه/بازنگری‌شده را ویرایش کند.
	return append(out, TariffSeed1405()...)
}
