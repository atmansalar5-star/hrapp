// آزمون دامنهٔ ماژول «grievances» (descriptor_test). آزمون خودکار؛ هر تغییر رفتار پیاده‌سازی باید
// آزمون همین پرونده را به‌روز کند.

package domain

import (
	"strings"
	"testing"

	"hrms/internal/contracts"
	"hrms/internal/platform/recordkit"
	"hrms/internal/rules"
)

// veteranKind نوع رکورد پروندهٔ ایثارگری را از توصیفگر می‌گیرد.
func veteranKind(t *testing.T) recordkit.Kind {
	t.Helper()
	kind, ok := Descriptor().KindByKey(KindVeteranRecords)
	if !ok {
		t.Fatal("نوع رکورد پروندهٔ ایثارگری در توصیفگر نیست")
	}
	return kind
}

// record ورودی سادهٔ پرونده می‌سازد.
func record(fields map[string]string) contracts.RecordInput {
	return contracts.RecordInput{Kind: KindVeteranRecords, Title: "پروندهٔ آزمایشی", Fields: fields}
}

// TestVeteranRecordValidationRequiresGroup بررسی می‌کند پرونده بدون گروه ایثارگری پذیرفته نشود.
func TestVeteranRecordValidationRequiresGroup(t *testing.T) {
	cases := []struct {
		name   string
		fields map[string]string
		want   string
	}{
		{"بدون گروه", map[string]string{"veteranType": ""}, "گروه"},
		{"گروه ناشناس", map[string]string{"veteranType": "   "}, "گروه"},
		{"جانباز بدون درصد", map[string]string{"veteranType": rules.VeteranGroupDisabled}, "درصد جانبازی"},
		{"جانباز درصد صفر", map[string]string{"veteranType": "جانباز", "percent": "0"}, "درصد جانبازی"},
		{"جانباز درصد بیش از صد", map[string]string{"veteranType": "جانباز", "percent": "۱۲۰"}, "بیشتر از ۱۰۰"},
		{"آزاده بدون مدت اسارت", map[string]string{"veteranType": "آزاده"}, "مدت اسارت"},
		{"رزمنده بدون حضور جبهه", map[string]string{"veteranType": "رزمنده"}, "حضور در جبهه"},
		{"خانوادهٔ شهید بدون نسبت", map[string]string{"veteranType": "خانواده شهید"}, "نسبت"},
		{"تاریخ صدور نامعتبر", map[string]string{"veteranType": "آزاده", "captivityMonths": "12", "issueDate": "1389/13/40"}, "تاریخ صدور"},
		{"معافیت مالیاتی بیش از صد", map[string]string{"veteranType": "آزاده", "captivityMonths": "12", "taxExemptPercent": "150"}, "درصد معافیت"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateVeteranRecord(record(tc.fields))
			if err == nil {
				t.Fatalf("انتظار خطا برای «%s» بود، ولی پرونده پذیرفته شد", tc.name)
			}
			var ruleErr *recordkit.RuleError
			if !asRuleError(err, &ruleErr) {
				t.Fatalf("نوع خطا باید RuleError باشد، شد %T", err)
			}
			if !strings.Contains(ruleErr.Message, tc.want) {
				t.Fatalf("پیام خطا %q باید شامل %q باشد", ruleErr.Message, tc.want)
			}
		})
	}
}

// asRuleError تبدیل امن خطا به RuleError بدون وابستگی به errors.As در تست‌های فارسی.
func asRuleError(err error, target **recordkit.RuleError) bool {
	if e, ok := err.(*recordkit.RuleError); ok {
		*target = e
		return true
	}
	return false
}

// TestVeteranRecordValidationAcceptsCompleteFiles هر گروه را با مستندات خودش می‌سنجد.
func TestVeteranRecordValidationAcceptsCompleteFiles(t *testing.T) {
	cases := []struct {
		name   string
		fields map[string]string
	}{
		{"رزمندهٔ دوازده ماه جبهه", map[string]string{"veteranType": "رزمنده", "frontMonths": "13"}},
		{"جانباز ۲۵ درصد", map[string]string{"veteranType": "جانباز", "percent": "25"}},
		{"جانباز با ارقام فارسی", map[string]string{"veteranType": "جانباز", "percent": "۲۵"}},
		{"آزادهٔ دو سال اسارت", map[string]string{"veteranType": "آزاده", "captivityMonths": "24", "issueDate": "۱۳۷۸/۰۸/۲۰"}},
		{"خانوادهٔ شهید", map[string]string{"veteranType": "خانواده شهید", "relation": "همسر"}},
		{"فرزند شهید", map[string]string{"veteranType": "فرزند شهید"}},
		{"سایر ایثارگران", map[string]string{"veteranType": "سایر ایثارگران"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := validateVeteranRecord(record(tc.fields)); err != nil {
				t.Fatalf("پروندهٔ کامل نباید خطا بدهد: %v", err)
			}
		})
	}
}

// TestVeteranRecordWorkflowLocksEntitlementsAfterApproval پس از تأیید، ویرایش امتیازها فقط با
// بازگشت به پیش‌نویس ممکن است و پرش از «پیش‌نویس» به «تأییدشده» مجاز نیست.
func TestVeteranRecordWorkflowLocksEntitlementsAfterApproval(t *testing.T) {
	kind := veteranKind(t)
	approve, ok := kind.ActionByValue("approve")
	if !ok {
		t.Fatal("اقدام «تأیید پرونده» در گردشکار نیست")
	}
	if kind.CanTransition("draft", approve) {
		t.Fatal("پروندهٔ پیش‌نویس نباید یک‌مرحله‌ای تأیید شود؛ ابتدا بررسی مدارک لازم است")
	}
	if !kind.CanTransition("verified", approve) {
		t.Fatal("پروندهٔ بررسی‌شده باید قابل تأیید باشد")
	}
	editable := strings.Join(kind.EditableStatuses, ",")
	if editable != "draft,verified" {
		t.Fatalf("وضعیت‌های ویرایش‌پذیر باید پیش‌نویس و بررسی‌شده باشند، شد %q", editable)
	}
}

// TestVeteranRecordSchemaPublishesEntitlementFields امتیازهای محاسبه‌شده باید در فرم و جدولِ
// کلاینت دیده شوند تا کارشناس مقدار تأییدشده را ببیند (نه قاعده‌ای در رابط).
func TestVeteranRecordSchemaPublishesEntitlementFields(t *testing.T) {
	schema := Descriptor().Schema()
	keys := map[string]bool{}
	for _, kind := range schema.Kinds {
		if kind.Key != KindVeteranRecords {
			continue
		}
		for _, field := range kind.Fields {
			keys[field.Key] = true
		}
		for _, status := range kind.Statuses {
			if status.Value == "approved" && status.LabelFa != "تأییدشده" {
				t.Fatalf("برچسب فارسی وضعیت تأییدشده درست نیست: %q", status.LabelFa)
			}
		}
	}
	for _, want := range []string{
		"veteranType", "percent", "captivityMonths", "frontMonths",
		"erafaqiMonths", "extraServiceMonths", "taxExemptPercent", "pensionExemptPercent",
		"educationUpgrade", "incentiveSteps", "leaveDays", "calculationLog",
	} {
		if !keys[want] {
			t.Fatalf("فیلد %q در طرح پروندهٔ ایثارگری نیست", want)
		}
	}
}

// TestVeteranSeedFilesAreValid همهٔ نمونه‌های سازمان باید از اعتبارسنجی خودشان بگذرند.
func TestVeteranSeedFilesAreValid(t *testing.T) {
	kinds := map[string]recordkit.Kind{}
	for _, kind := range Descriptor().Kinds {
		kinds[kind.Key] = kind
	}
	seeds := Seed()
	if len(seeds) == 0 {
		t.Fatal("نمونهٔ سازمان برای ماژول ایثارگران تعریف نشده است")
	}
	approved := 0
	for _, in := range seeds {
		kind, ok := kinds[in.Kind]
		if !ok {
			t.Fatalf("نوع رکورد نمونه %q در توصیفگر نیست", in.Kind)
		}
		if errs := kind.ValidateInput(in, false); len(errs) > 0 {
			t.Fatalf("نمونهٔ %q اعتبارسنجی عمومی را رد نکرد: %v", in.Title, errs)
		}
		if kind.Validate != nil {
			if err := kind.Validate(in); err != nil {
				t.Fatalf("نمونهٔ %q در اعتبارسنجی دامنه رد شد: %v", in.Title, err)
			}
		}
		if in.Kind == KindVeteranRecords && in.SeedStatus == "approved" {
			approved++
			if rules.VeteranGroupOf(in.Fields["veteranType"]) == rules.VeteranGroupNone {
				t.Fatalf("گروه ایثارگری نمونهٔ %q خوانده نشد", in.Title)
			}
		}
	}
	if approved < 2 {
		t.Fatalf("برای نمایش اثر مالی، دست‌کم دو پروندهٔ تأییدشدهٔ نمونه لازم است؛ شد %d", approved)
	}
}

// TestVeteranRecordValidationReportsAllRequiredFields بررسی می‌کند فیلدهای الزامی اعلام شده‌اند.
func TestVeteranRecordValidationReportsAllRequiredFields(t *testing.T) {
	kind := veteranKind(t)
	errs := kind.ValidateInput(record(map[string]string{}), false)
	if _, ok := errs["fields.veteranType"]; !ok {
		t.Fatalf("نبودِ گروه ایثارگری باید به‌عنوان خطای فیلد الزامی برگردد، شد %v", errs)
	}
}
