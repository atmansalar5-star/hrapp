// منطق دامنهٔ ماژول «loans_retirement» — «descriptor»: قواعد، اعتبارسنجی و محاسبات خالص بدون وابستگی به زیرساخت.

package domain

import (
	"strings"

	"hrms/internal/contracts"
	"hrms/internal/platform/recordkit"
	"hrms/internal/rules"
)

// Descriptor شناسهٔ ماژول وام و بازنشستگی.
func Descriptor() recordkit.Descriptor {
	return recordkit.Descriptor{
		Module:  "loans_retirement",
		TitleFa: "امور وام و بازنشستگی",
		Kinds: []recordkit.Kind{
			{
				Key: "loans", TitleFa: "درخواست وام", IDPrefix: "LOAN", EmployeeScoped: true,
				Fields: recordkit.Fields(
					// نوع وام از فهرست اعلامی دامنه (ودیعه مسکن، متمم، بانکی، اضطراری،
					// مددکاری، فوت بستگان درجه یک و حوادث مترقبه) پر می‌شود.
					"loanType|نوع وام|select|1",
					"bank|بانک عامل|select|0",
					// مدیریت متقاضی از جدول سهمیهٔ تسهیلات (مدیریت‌های بندر امام) پر می‌شود و
					// مبنای نمودارهای تخصیص سهمیه است.
					"management|مدیریت متقاضی|select|0",
					// واحد سازمانی زیر همان مدیریت، از کاتالوگ مرجع تنظیمات؛ مشخص می‌کند وام
					// به کدام واحد رفته است.
					"unit|واحد سازمانی|select|0",
					"amount|مبلغ وام|money|1",
					"installments|تعداد اقساط|number|0",
					"installmentAmount|میزان هر قسط|money|0",
					"feeRate|نرخ کارمزد (درصد)|number|0",
					"feeAmount|مبلغ کارمزد|money|0",
					// ماه شروع کسر اقساط؛ خالی یعنی ماه بعد از تاریخ پرداخت. قاعدهٔ
					// ساخت جدول اقساط در بستهٔ rules است، نه در UI.
					"firstPeriod|شروع کسر اقساط (YYYY/MM)|text|0",
					// تسویهٔ پیش از موعد: مبلغ و تاریخ تسویه، مبنای کاهش مانده بدهی است.
					"settleDate|تاریخ تسویه|date|0",
					"settledAmount|مبلغ تسویه|money|0",
					"supplementOf|متممِ وام (شناسهٔ وام پایه)|text|0",
					"guarantorName|نام ضامن|text|0",
					"guarantorNationalId|کد ملی ضامن|text|0",
					"guarantorRelation|نسبت ضامن|text|0",
					"receivedDate|تاریخ پرداخت وام|date|0",
					"purpose|هدف وام|text|0",
				),
				Statuses: recordkit.Statuses(
					"requested|درخواست‌شده", "scored|امتیازبندی‌شده", "approved|تأییدشده",
					"rejected|ردشده", "repaying|در حال بازپرداخت", "settled|تسویه‌شده",
				),
				Initial: "requested",
				Transitions: recordkit.Steps(
					"requested|scored,approved,rejected",
					"scored|approved,rejected",
					"approved|repaying,rejected",
					"repaying|settled",
				),
				Actions: recordkit.Actions(
					"score|امتیازبندی تخصیص|ghost|scored|0|hr_spec,hr_manager",
					"approve|تأیید وام|primary|approved|0|hr_manager",
					"disburse|پرداخت تسهیلات|primary|repaying|0|hr_manager",
					"settle|تسویهٔ کامل|primary|settled|0|hr_manager",
					"reject|رد درخواست|danger|rejected|1|hr_manager",
				),
				// قاعدهٔ دامنه: نوع وام باید از فهرست اعلامی باشد و متمم باید وام پایه داشته باشد.
				Validate: validateLoan,
			},
			{
				Key: "loanInstallments", TitleFa: "قسط وام", IDPrefix: "INST", EmployeeScoped: true,
				Fields: recordkit.Fields(
					// پیوند قسط به وام پایه و شمارهٔ قسط، ستون‌های جدول اقساط را می‌سازد؛
					// بدون این دو، قسط فقط یک سررسید تنها بود و صورت حساب وام ممکن نبود.
					"loan|شناسهٔ وام|text|0",
					"seq|شمارهٔ قسط|number|0",
					"period|ماه کسر (YYYY/MM)|text|0",
					"dueDate|سررسید|date|1",
					"amount|مبلغ قسط|money|1",
					"principal|سهم اصل|money|0",
					"fee|سهم کارمزد|money|0",
					"paidDate|تاریخ کسر/پرداخت|date|0",
					"method|روش پرداخت (حقوق/نقدی/بانکی)|select|0",
				),
				Statuses:    recordkit.Statuses("pending|در انتظار پرداخت", "paid|پرداخت‌شده", "overdue|معوق"),
				Initial:     "pending",
				Transitions: recordkit.Steps("pending|paid,overdue", "overdue|paid"),
				Actions: recordkit.Actions(
					"pay|ثبت پرداخت قسط|primary|paid|0|hr_manager",
					"overdue|ثبت معوق شدن|danger|overdue|1|hr_manager",
				),
				Validate: validateInstallment,
			},
			{
				Key: "retirementFiles", TitleFa: "پروندهٔ بازنشستگی", IDPrefix: "RET", EmployeeScoped: true,
				Fields: recordkit.Fields(
					"retireDate|تاریخ بازنشستگی|date|0",
					"serviceYears|سابقهٔ خدمت|number|0",
					"pension|حقوق بازنشستگی|money|0",
				),
				Statuses:    recordkit.Statuses("draft|پیش‌نویس", "verified|بررسی‌شده", "approved|تأییدشده"),
				Initial:     "draft",
				Transitions: recordkit.Steps("draft|verified", "verified|approved"),
				Actions: recordkit.Actions(
					"verify|بررسی سوابق|primary|verified|0|hr_spec,hr_manager",
					"approve|تأیید پرونده|primary|approved|0|hr_manager",
				),
			},
		},
	}
}

// KindLoans کلید نوع رکورد درخواست وام؛ برای وصل کردن قاعدهٔ زمان‌اجرا به همان نوع.
const KindLoans = "loans"

// validateLoan قاعدهٔ دامنهٔ درخواست وام: نوع وام باید شناخته‌شده باشد و اگر وام متمم
// است، وام پایهٔ آن مشخص شده باشد. این قاعده در دامنه می‌ماند تا UI فقط فهرست انواع
// را نشان دهد و هیچ منطقی در آن تکرار نشود.
func validateLoan(in contracts.RecordInput) error { return validateLoanWith(in, nil) }

// validateLoanWith همان قاعده، با یک پذیرندهٔ اضافی برای کلیدهای مدیریت.
//
// چرا این پارامتر: مرجع مدیریت‌ها در ماژول تنظیمات قابل ویرایش است و وام نباید
// داخلی‌های آن ماژول را import کند. پس قاعدهٔ دامنه می‌ماند و فقط دامنه‌اش با اقلام
// کاتالوگ مرجع زمان‌اجرا گسترش می‌یابد (AcceptRuntimeManagements).
func validateLoanWith(in contracts.RecordInput, acceptManagement func(string) bool) error {
	loanType := in.Fields["loanType"]
	if loanType == "" {
		return &recordkit.RuleError{Message: "انتخاب نوع وام الزامی است."}
	}
	if !ValidLoanType(loanType) {
		return &recordkit.RuleError{Message: "نوع وام «" + loanType + "» در فهرست انواع وام شناخته نشد."}
	}
	if IsSupplement(loanType) && in.Fields["supplementOf"] == "" {
		return &recordkit.RuleError{Message: "برای وام متمم باید شناسهٔ وام پایه (متممِ وام) مشخص شود."}
	}
	if m := strings.TrimSpace(in.Fields["management"]); m != "" && !ValidQuotaUnit(m) {
		if acceptManagement == nil || !acceptManagement(m) {
			return &recordkit.RuleError{Message: "مدیریت «" + m + "» در جدول سهمیهٔ تسهیلات شناخته نشد."}
		}
	}

	// ---------- شرایط مالی وام: اعتبارسنجی + پرکردن از قواعد ----------
	//
	// قاعده‌های محاسبه در بستهٔ rules است؛ اینجا فقط شکل فیلد سنجیده و مقدارهای محاسباتی
	// (کارمزد، مبلغ قسط، ماه شروع کسر) نوشته می‌شوند تا فرم و فیش و صورت حساب یک عدد
	// ببینند. ورودی مرحلهٔ «درخواست» (بی‌اقساط یا بی‌تاریخ پرداخت) رد نمی‌شود؛ فقط مقدار
	// نامعتبر یا خارج از سقف رد می‌شود.
	f := in.Fields
	details := map[string]string{}
	amount := int64Field(f["amount"])
	installments := int64Field(f["installments"])
	feeRate := floatField(f["feeRate"])
	feeAmount := int64Field(f["feeAmount"])

	if strings.TrimSpace(f["amount"]) != "" && amount <= 0 {
		details["fields.amount"] = "مبلغ وام باید بزرگ‌تر از صفر باشد."
	}
	if installments < 0 {
		details["fields.installments"] = "تعداد اقساط نمی‌تواند منفی باشد."
	}
	if installments > 600 {
		details["fields.installments"] = "تعداد اقساط بیش از ۶۰۰ ماه پذیرفته نمی‌شود."
	}
	if feeRate < 0 || feeRate > 100 {
		details["fields.feeRate"] = "نرخ کارمزد باید بین صفر و ۱۰۰ درصد باشد."
	}
	if feeAmount < 0 {
		details["fields.feeAmount"] = "مبلغ کارمزد نمی‌تواند منفی باشد."
	}
	if raw := strings.TrimSpace(f["firstPeriod"]); raw != "" && rules.LoanPeriod(raw) == "" {
		details["fields.firstPeriod"] = "ماه شروع کسر اقساط باید به قالب شمسی YYYY/MM باشد."
	}
	if int64Field(f["settledAmount"]) > 0 && strings.TrimSpace(f["settleDate"]) == "" {
		details["fields.settleDate"] = "برای مبلغ تسویه، تاریخ تسویه الزامی است."
	}
	if len(details) > 0 {
		return &recordkit.ValidationError{Details: details}
	}

	// کارمزد: اگر نرخ دارد و مبلغ کارمزد خالی است، از قواعد محاسبه می‌شود.
	if amount > 0 && feeRate > 0 && strings.TrimSpace(f["feeAmount"]) == "" {
		f["feeAmount"] = itoa64(rules.LoanFeeAmount(amount, feeRate))
		feeAmount = int64Field(f["feeAmount"])
	}
	// مبلغ هر قسط: از جمع قابل بازپرداخت و تعداد اقساط (تعداد صفر = بدون جدول اقساط).
	if amount > 0 && installments > 0 {
		f["installmentAmount"] = itoa64(rules.LoanInstallmentAmount(rules.LoanTotalPayable(amount, feeAmount), int(installments)))
	}
	// ماه شروع کسر: خالی ⇒ ماه بعد از تاریخ پرداخت وام.
	if strings.TrimSpace(f["firstPeriod"]) == "" {
		if period := rules.LoanFirstPeriod(f["receivedDate"]); period != "" {
			f["firstPeriod"] = period
		}
	}
	return nil
}

// validateInstallment قاعدهٔ دامنهٔ قسط وام: ماه کسر و سررسید باید شمسی باشند، مبلغ با
// جمع سهم اصل و کارمزد بخواند و قسط بی‌پیوند به وام (پیوند خالی) هم مجاز باشد ولی در
// صورت حساب شمرده نشود.
func validateInstallment(in contracts.RecordInput) error {
	f := in.Fields
	details := map[string]string{}
	if raw := strings.TrimSpace(f["period"]); raw != "" && rules.LoanPeriod(raw) == "" {
		details["fields.period"] = "ماه کسر باید به قالب شمسی YYYY/MM باشد."
	}
	amount := int64Field(f["amount"])
	principal := int64Field(f["principal"])
	fee := int64Field(f["fee"])
	if strings.TrimSpace(f["amount"]) != "" && amount <= 0 {
		details["fields.amount"] = "مبلغ قسط باید بزرگ‌تر از صفر باشد."
	}
	if (principal > 0 || fee > 0) && amount != principal+fee {
		details["fields.amount"] = "مبلغ قسط با جمع سهم اصل و کارمزد نمی‌خواند."
	}
	if seq := int64Field(f["seq"]); seq < 0 {
		details["fields.seq"] = "شمارهٔ قسط نمی‌تواند منفی باشد."
	}
	if len(details) > 0 {
		return &recordkit.ValidationError{Details: details}
	}
	if period := rules.LoanPeriod(f["period"]); period != "" {
		f["period"] = period
		if strings.TrimSpace(f["dueDate"]) == "" {
			f["dueDate"] = period + "/01"
		}
	}
	return nil
}

// AcceptRuntimeManagements شناسهٔ ماژول را چنان برمی‌گرداند که اعتبارسنجی رکورد
// «درخواست وام» مدیریت‌های کاتالوگ مرجع زمان‌اجرا را هم بپذیرد.
//
// چرا لازم است: پنل تنظیمات می‌تواند مدیریت تازه بسازد و همان مدیریت در فرم تخصیص وام
// انتخاب شود؛ اگر قاعدهٔ رکورد فقط جدول ثابت سهمیه را می‌شناخت، ثبت تخصیص با قلم تازه
// رد می‌شد. ماژول وام این تابع را با پورت کاتالوگ وصل می‌کند، پس هیچ importی بین دو
// ماژول لازم نیست و قاعده هم در دامنه می‌ماند.
func AcceptRuntimeManagements(desc recordkit.Descriptor, accept func(string) bool) recordkit.Descriptor {
	out := desc
	out.Kinds = make([]recordkit.Kind, len(desc.Kinds))
	copy(out.Kinds, desc.Kinds)
	for i := range out.Kinds {
		if out.Kinds[i].Key != KindLoans {
			continue
		}
		out.Kinds[i].Validate = func(in contracts.RecordInput) error {
			return validateLoanWith(in, accept)
		}
	}
	return out
}

// Seed رکوردهای نمونهٔ سازمان — نمونه‌ای از هر نوع وام به همراه اطلاعات ضامن، زمان
// دریافت و «مدیریت متقاضی» (از جدول سهمیهٔ تسهیلات).
//
// SeedStatus وضعیت نمونه را روی گردش‌کار واقعی همان نوع رکورد می‌نشاند تا نمودارهای
// «وضعیت ارسال به مدیریت مالی» و «درصد سهمیهٔ ارسالی» روی دادهٔ معنادار ساخته شوند؛
// این مقدار فقط در دادهٔ نمونه اعمال می‌شود و از API قابل تنظیم نیست.
func Seed() []contracts.RecordInput {
	return []contracts.RecordInput{
		{Kind: "loans", Title: "وام ودیعهٔ مسکن — محمد احمدی", EmployeeID: "EMP-005", SeedStatus: "repaying",
			Fields: map[string]string{
				"loanType": "housingDeposit", "bank": "melli", "management": "process_one",
				"amount":       "800000000",
				"installments": "60", "installmentAmount": "14500000", "feeRate": "4",
				"feeAmount": "32000000", "guarantorName": "علی احمدی", "guarantorNationalId": "0079843211",
				"guarantorRelation": "برادر", "receivedDate": "1404/03/15", "purpose": "ودیعهٔ مسکن",
			},
			NoteFa: "در حال بازپرداخت"},
		{Kind: "loans", Title: "متمم ودیعهٔ مسکن — محمد احمدی", EmployeeID: "EMP-005", SeedStatus: "approved",
			Fields: map[string]string{
				"loanType": "housingDepositSupplement", "bank": "melli", "management": "process_one",
				"amount":       "200000000",
				"installments": "36", "installmentAmount": "5900000", "feeRate": "4",
				"feeAmount": "8000000", "supplementOf": "LOAN-0001", "guarantorName": "علی احمدی",
				"guarantorRelation": "برادر", "receivedDate": "1404/07/02", "purpose": "تکمیل ودیعهٔ مسکن",
			},
			NoteFa: "متمم وام پایه"},
		{Kind: "loans", Title: "وام بانکی — نگار حسینی", EmployeeID: "EMP-003", SeedStatus: "scored",
			Fields: map[string]string{
				"loanType": "bank", "bank": "mellat", "management": "bandar_imam", "amount": "300000000",
				"installments": "24", "installmentAmount": "13200000", "feeRate": "18",
				"feeAmount": "54000000", "guarantorName": "سارا حسینی", "guarantorRelation": "همسر",
				"receivedDate": "1404/04/10", "purpose": "هزینهٔ درمان",
			},
			NoteFa: "وام بانکی با کارمزد"},
		{Kind: "loans", Title: "وام اضطراری — رضا کریمی", EmployeeID: "EMP-007", SeedStatus: "approved",
			Fields: map[string]string{
				"loanType": "emergency", "management": "water_power", "amount": "50000000",
				"installments":      "12",
				"installmentAmount": "4200000", "feeRate": "0", "guarantorName": "حمید کریمی",
				"guarantorRelation": "پدر", "receivedDate": "1404/06/01", "purpose": "هزینهٔ اضطراری",
			},
			NoteFa: "وام اضطراری بدون کارمزد"},
		{Kind: "loans", Title: "وام مددکاری — زهرا موسوی", EmployeeID: "EMP-011",
			Fields: map[string]string{
				"loanType": "socialWork", "management": "kimia", "amount": "80000000", "installments": "20",
				"installmentAmount": "4100000", "feeRate": "2", "guarantorName": "مریم موسوی",
				"guarantorRelation": "خواهر", "receivedDate": "1404/05/20", "purpose": "حمایت مددکاری",
			},
			NoteFa: "معرفی‌شده از مددکاری اجتماعی"},
		{Kind: "loans", Title: "وام فوت بستگان درجه یک — اکبر رضایی", EmployeeID: "EMP-009", SeedStatus: "settled",
			Fields: map[string]string{
				"loanType": "firstDegreeDeath", "management": "process_two", "amount": "60000000", "installments": "18",
				"installmentAmount": "3400000", "feeRate": "0", "guarantorName": "ناصر رضایی",
				"guarantorRelation": "برادر", "receivedDate": "1404/02/11", "purpose": "هزینهٔ کفن و دفن",
			},
			NoteFa: "بدون کارمزد"},
		{Kind: "loans", Title: "وام حوادث مترقبه — سعید نادری", EmployeeID: "EMP-002", SeedStatus: "rejected",
			Fields: map[string]string{
				"loanType": "unforeseen", "bank": "mehreghtesad", "management": "repair_support",
				"amount": "120000000", "installments": "30",
				"installmentAmount": "4200000", "feeRate": "1", "guarantorName": "کاظم نادری",
				"guarantorRelation": "عمو", "receivedDate": "1404/08/05", "purpose": "جبران خسارت حادثهٔ مترقبه",
			},
			NoteFa: "وام حوادث مترقبه"},
		{Kind: "loans", Title: "وام بانکی — فاطمه صادقی (پیمانکار)", EmployeeID: "EMP-010", SeedStatus: "repaying",
			Fields: map[string]string{
				"loanType": "bank", "bank": "mehreghtesad", "management": "basparan", "amount": "250000000",
				"installments": "24", "installmentAmount": "10400000", "feeRate": "18",
				"feeAmount": "45000000", "guarantorName": "حسن صادقی", "guarantorRelation": "پدر",
				"receivedDate": "1404/05/05", "purpose": "وام بانکی نیروی پیمانکاری",
			},
			NoteFa: "نیروی ارکان ثالث"},
		{Kind: "loans", Title: "وام اضطراری — علی مرادی", EmployeeID: "EMP-004", SeedStatus: "scored",
			Fields: map[string]string{
				"loanType": "emergency", "management": "services_support", "amount": "70000000",
				"installments": "12", "installmentAmount": "5900000", "feeRate": "0",
				"guarantorName": "رسول مرادی", "guarantorRelation": "برادر",
				"receivedDate": "1404/07/10", "purpose": "هزینهٔ اضطراری",
			},
			NoteFa: "در انتظار تأیید"},
		{Kind: "loans", Title: "وام مددکاری — سارا محمدی", EmployeeID: "EMP-001", SeedStatus: "approved",
			Fields: map[string]string{
				"loanType": "socialWork", "bank": "refah", "management": "process_two", "amount": "90000000",
				"installments": "20", "installmentAmount": "4600000", "feeRate": "2",
				"feeAmount": "1800000", "guarantorName": "مینا محمدی", "guarantorRelation": "خواهر",
				"receivedDate": "1404/06/18", "purpose": "حمایت مددکاری",
			},
			NoteFa: "تأییدشده و ارسالی به مالی"},
		{Kind: "loanInstallments", Title: "قسط دوم وام مسکن — احمدی", EmployeeID: "EMP-005",
			Fields: map[string]string{"dueDate": "1404/08/01", "amount": "18300000"}},
		{Kind: "retirementFiles", Title: "پروندهٔ بازنشستگی — کارمند باسابقه", EmployeeID: "EMP-009",
			Fields: map[string]string{"retireDate": "1405/01/01", "serviceYears": "31", "pension": "27000000"},
			NoteFa: "بررسی سوابق بیمه در جریان است"},
	}
}
