// آزمون دامنهٔ ماژول «compensation» (descriptor_test). آزمون خودکار؛ هر تغییر رفتار پیاده‌سازی باید
// آزمون همین پرونده را به‌روز کند.

package domain

import (
	"strings"
	"testing"

	"hrms/internal/contracts"
	"hrms/internal/rules"
)

// kindOf نوع رکورد را از شناسهٔ ماژول بیرون می‌کشد.
func kindOf(t *testing.T, key string) (string, bool) {
	t.Helper()
	for _, kind := range Descriptor().Kinds {
		if kind.Key == key {
			return kind.IDPrefix, true
		}
	}
	return "", false
}

// TestCompensationKindsAreDefined هشت نوع رکورد (شش نوع ۴پی و دو نوع جی‌بیست) با پیشوند
// شناسهٔ یکتا اعلام شده‌اند.
func TestCompensationKindsAreDefined(t *testing.T) {
	want := map[string]string{
		KindCompensationModel:    "CMOD",
		KindJobPricing:           "CJPR",
		KindCompetencyLevel:      "CCMP",
		KindPerformanceFactor:    "CPRF",
		KindPayBand:              "CBND",
		KindCompensationDecision: "CDEC",
		KindG20Index:             "CG20",
		KindG20Assessment:        "CG2A",
	}
	seen := map[string]string{}
	for key, prefix := range want {
		got, ok := kindOf(t, key)
		if !ok {
			t.Fatalf("نوع رکورد %s در شناسهٔ ماژول نیست", key)
		}
		if got != prefix {
			t.Fatalf("پیشوند شناسهٔ %s باید %s باشد، بود %s", key, prefix, got)
		}
		if other, dup := seen[prefix]; dup {
			t.Fatalf("پیشوند %s هم به %s و هم به %s می‌رسد", prefix, other, key)
		}
		seen[prefix] = key
	}
	if len(Descriptor().Kinds) != len(want) {
		t.Fatalf("شمار نوع‌های رکورد %d است، انتظار %d", len(Descriptor().Kinds), len(want))
	}
}

// TestEmployeeScopedKinds کارمندمحور بودن سه نوع سرآمد و نبودن آن در مدل و باند.
func TestEmployeeScopedKinds(t *testing.T) {
	wantScoped := map[string]bool{
		KindCompetencyLevel:      true,
		KindPerformanceFactor:    true,
		KindCompensationDecision: true,
		KindCompensationModel:    false,
		KindJobPricing:           false,
		KindPayBand:              false,
	}
	for _, kind := range Descriptor().Kinds {
		want, known := wantScoped[kind.Key]
		if !known {
			continue
		}
		if kind.EmployeeScoped != want {
			t.Fatalf("کارمندمحور بودن %s باید %v باشد", kind.Key, want)
		}
	}
}

// TestCompensationModelValidation پارامترهای مدل در بازهٔ معقول پذیرفته و بیرون آن رد شوند.
func TestCompensationModelValidation(t *testing.T) {
	base := map[string]string{
		"modelName": "مدل ۱۴۰۵", "policyScope": "organization", "effectiveDate": "1405/01/01",
		"bandBaseMidpoint": "120000000", "bandSlopePercent": "18", "bandSpreadPercent": "40",
		"bandStepPercent": "3", "jobPointsPerGrade": "60", "personMinFactor": "0.9", "personMaxFactor": "1.1",
		"perfFactorExcellent": "1.15", "perfFactorGood": "1.08", "perfFactorMeets": "1",
		"perfFactorBelow": "0.92", "marketAdjustPercent": "10", "maxAnnualIncreasePercent": "25",
		"payrollApply": "yes",
	}
	valid := contracts.RecordInput{Kind: KindCompensationModel, Fields: base}
	if err := validateCompensationModel(valid); err != nil {
		t.Fatalf("مدل کامل نباید خطا بدهد: %v", err)
	}

	// ارقام فارسی هم باید پذیرفته شوند (کاربر در پنل فارسی می‌نویسد).
	fa := map[string]string{}
	for k, v := range base {
		fa[k] = v
	}
	fa["bandBaseMidpoint"] = "۱۲۰٬۰۰۰٬۰۰۰"
	if err := validateCompensationModel(contracts.RecordInput{Fields: fa}); err != nil {
		t.Fatalf("مبلغ با ارقام فارسی باید پذیرفته شود: %v", err)
	}

	cases := []struct {
		name  string
		patch map[string]string
		want  string
	}{
		{"بدون نام", map[string]string{"modelName": ""}, "نام مدل"},
		{"دامنهٔ نامعلوم", map[string]string{"policyScope": "world"}, "دامنهٔ اجرا"},
		{"بدون تاریخ اجرا", map[string]string{"effectiveDate": ""}, "تاریخ اجرا"},
		{"تاریخ نادرست", map[string]string{"effectiveDate": "1405/13/40"}, "تاریخ اجرا"},
		{"میانهٔ صفر", map[string]string{"bandBaseMidpoint": "0"}, "میانهٔ باند"},
		{"شیب بیرون بازه", map[string]string{"bandSlopePercent": "150"}, "شیب میانهٔ باند"},
		{"گسترهٔ بیرون بازه", map[string]string{"bandSpreadPercent": "500"}, "گسترهٔ باند"},
		{"ضریب معکوس", map[string]string{"personMinFactor": "1.2", "personMaxFactor": "1.1"}, "کف ضریب شایستگی"},
		{"ضریب عملکرد صفر", map[string]string{"perfFactorGood": "0"}, "ضرایب عملکرد"},
		{"سقف افزایش بیرون بازه", map[string]string{"maxAnnualIncreasePercent": "180"}, "سقف افزایش سالانه"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fields := map[string]string{}
			for k, v := range base {
				fields[k] = v
			}
			for k, v := range tc.patch {
				fields[k] = v
			}
			err := validateCompensationModel(contracts.RecordInput{Fields: fields})
			if err == nil {
				t.Fatal("انتظار خطا بود ولی مدل پذیرفته شد")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("پیام خطا باید شامل «%s» باشد، بود: %s", tc.want, err.Error())
			}
		})
	}
}

// TestJobPricingValidation امتیاز شغل، سازگاری عوامل و پلهٔ گروه شغلی سنجیده شود.
func TestJobPricingValidation(t *testing.T) {
	ok := contracts.RecordInput{Fields: map[string]string{
		"positionCode": "POS-0142", "positionTitle": "سرپرست بهره‌برداری", "jobFamily": "operations",
		"evaluationPoints": "240", "factorKnowledge": "70", "factorProblemSolving": "65",
		"factorResponsibility": "65", "factorWorkingConditions": "40", "gradeRank": "5",
		"pricingDate": "1405/01/12",
	}}
	if err := validateJobPricing(ok); err != nil {
		t.Fatalf("قیمت‌گذاری کامل نباید خطا بدهد: %v", err)
	}
	// بدون عامل‌های تفصیلی هم باید پذیرفته شود (امتیاز مصوب از طبقه‌بندی مشاغل می‌آید).
	noFactors := contracts.RecordInput{Fields: map[string]string{
		"positionTitle": "کارشناس تعمیرات", "jobFamily": "maintenance",
		"evaluationPoints": "180", "gradeRank": "4", "pricingDate": "1405/01/12",
	}}
	if err := validateJobPricing(noFactors); err != nil {
		t.Fatalf("قیمت‌گذاری بدون عامل تفصیلی نباید خطا بدهد: %v", err)
	}

	bad := []struct {
		name  string
		patch map[string]string
		want  string
	}{
		{"بدون عنوان پست", map[string]string{"positionTitle": ""}, "عنوان پست"},
		{"خانوادهٔ نامعلوم", map[string]string{"jobFamily": "space"}, "خانوادهٔ شغلی"},
		{"بدون امتیاز", map[string]string{"evaluationPoints": ""}, "امتیاز ارزشیابی"},
		{"جمع ناهمخوان", map[string]string{"evaluationPoints": "300"}, "جمع عامل‌های شغل"},
		{"پلهٔ بیرون نردبان", map[string]string{"gradeRank": "12"}, "پلهٔ گروه شغلی"},
		{"بدون تاریخ", map[string]string{"pricingDate": ""}, "تاریخ قیمت‌گذاری"},
	}
	for _, tc := range bad {
		t.Run(tc.name, func(t *testing.T) {
			fields := map[string]string{}
			for k, v := range ok.Fields {
				fields[k] = v
			}
			for k, v := range tc.patch {
				fields[k] = v
			}
			err := validateJobPricing(contracts.RecordInput{Fields: fields})
			if err == nil {
				t.Fatal("انتظار خطا بود")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("پیام خطا باید شامل «%s» باشد، بود: %s", tc.want, err.Error())
			}
		})
	}
}

// TestCompetencyAndPerformanceValidation سطح شایستگی و ضریب عملکرد سنجیده شوند.
func TestCompetencyAndPerformanceValidation(t *testing.T) {
	competency := contracts.RecordInput{Fields: map[string]string{
		"competencyModel": "operations", "level": "4", "levelDate": "1405/02/15", "levelScore": "84",
	}}
	if err := validateCompetencyLevel(competency); err != nil {
		t.Fatalf("سطح شایستگی معتبر نباید خطا بدهد: %v", err)
	}
	for _, level := range []string{"0", "9"} {
		fields := map[string]string{"competencyModel": "hr", "level": level, "levelDate": "1405/02/15"}
		if err := validateCompetencyLevel(contracts.RecordInput{Fields: fields}); err == nil {
			t.Fatalf("سطح %s باید رد شود", level)
		}
	}

	perf := contracts.RecordInput{Fields: map[string]string{
		"cycleTitle": "ارزیابی ۱۴۰۴", "cycleYear": "1404", "score": "86", "level": "good", "factor": "1.08",
	}}
	if err := validatePerformanceFactor(perf); err != nil {
		t.Fatalf("ضریب عملکرد معتبر نباید خطا بدهد: %v", err)
	}
	mismatch := map[string]string{
		"cycleTitle": "ارزیابی ۱۴۰۴", "cycleYear": "1404", "score": "86", "level": "excellent", "factor": "1.08",
	}
	err := validatePerformanceFactor(contracts.RecordInput{Fields: mismatch})
	if err == nil || !strings.Contains(err.Error(), "همخوان نیست") {
		t.Fatalf("ناهمخوانی سطح و امتیاز باید خطا بدهد: %v", err)
	}
}

// TestPayBandValidation ترتیب کف، میانه و سقف باند سنجیده شود.
func TestPayBandValidation(t *testing.T) {
	ok := contracts.RecordInput{Fields: map[string]string{
		"gradeRank": "5", "gradeTitle": "مدیریت", "minPay": "200000000", "midPay": "230000000",
		"maxPay": "270000000", "stepAmount": "6900000", "effectiveDate": "1405/01/01",
	}}
	if err := validatePayBand(ok); err != nil {
		t.Fatalf("باند معتبر نباید خطا بدهد: %v", err)
	}
	broken := map[string]string{
		"gradeRank": "5", "gradeTitle": "مدیریت", "minPay": "230000000", "midPay": "230000000",
		"maxPay": "270000000", "stepAmount": "6900000", "effectiveDate": "1405/01/01",
	}
	err := validatePayBand(contracts.RecordInput{Fields: broken})
	if err == nil || !strings.Contains(err.Error(), "کف < میانه < سقف") {
		t.Fatalf("ترتیب نادرست باند باید خطا بدهد: %v", err)
	}
}

// TestCompensationDecisionValidation تصمیم چهارعاملی: افزایش از تفاوت هدف و فعلی بیشتر نشود.
func TestCompensationDecisionValidation(t *testing.T) {
	ok := contracts.RecordInput{
		EmployeeID: "EMP-001",
		Fields: map[string]string{
			"decisionNo": "۱۴۰۵/ج‌خ/۰۲۷", "decisionDate": "1405/02/25", "effectiveDate": "1405/03/01",
			"jobScore": "240", "gradeRank": "5", "competencyLevel": "4", "performanceScore": "86",
			"currentPay": "268000000", "targetPay": "285000000", "increaseAmount": "17000000",
			"marketAdjust": "no", "decisionFa": "افزایش بر پایهٔ شغل، شایستگی و عملکرد",
		},
	}
	if err := validateCompensationDecision(ok); err != nil {
		t.Fatalf("تصمیم کامل نباید خطا بدهد: %v", err)
	}

	over := map[string]string{}
	for k, v := range ok.Fields {
		over[k] = v
	}
	over["increaseAmount"] = "50000000"
	err := validateCompensationDecision(contracts.RecordInput{EmployeeID: "EMP-001", Fields: over})
	if err == nil || !strings.Contains(err.Error(), "بیشتر") {
		t.Fatalf("افزایش بیش از تفاوت هدف و فعلی باید خطا بدهد: %v", err)
	}

	noText := map[string]string{}
	for k, v := range ok.Fields {
		noText[k] = v
	}
	noText["decisionFa"] = ""
	if err := validateCompensationDecision(contracts.RecordInput{Fields: noText}); err == nil {
		t.Fatal("تصمیم بدون متن مبنا باید رد شود")
	}
}

// TestCompensationSeedCoversAllKinds بذر نمونه همهٔ نوع‌های رکورد را پوشش دهد و با پارامترها یک‌سان باشد.
func TestCompensationSeedCoversAllKinds(t *testing.T) {
	counts := map[string]int{}
	for _, item := range Seed() {
		counts[item.Kind]++
	}
	for _, key := range []string{
		KindCompensationModel, KindJobPricing, KindCompetencyLevel, KindPerformanceFactor,
		KindPayBand, KindCompensationDecision,
	} {
		if counts[key] == 0 {
			t.Fatalf("بذر نمونه برای نوع %s ساخته نشده است", key)
		}
	}
	params := rules.DefaultCompensationParams()
	if counts[KindPayBand] != params.JobMaxGrade {
		t.Fatalf("بذر باید برای هر %d پله یک باند بسازد، ساخت: %d", params.JobMaxGrade, counts[KindPayBand])
	}
	// باندهای نمونه باید همان باندهای موتور قواعد باشند (نه عدد دستی).
	bands := rules.CompensationBands(params)
	var seeded contracts.RecordInput
	for _, item := range Seed() {
		if item.Kind == KindPayBand && item.Fields["gradeRank"] == "3" {
			seeded = item
		}
	}
	if seeded.Kind == "" {
		t.Fatal("باند پلهٔ ۳ در بذر نبود")
	}
	if seeded.Fields["midPay"] != itoa(bands[2].Midpoint) {
		t.Fatalf("میانهٔ باند بذر با موتور قواعد یکی نیست: %s در برابر %d",
			seeded.Fields["midPay"], bands[2].Midpoint)
	}
}

// itoa تبدیل سادهٔ عدد به رشتهٔ لاتین (بدون وابستگی تازه).
func itoa(v int64) string {
	if v == 0 {
		return "0"
	}
	digits := ""
	for v > 0 {
		digits = string(rune('0'+v%10)) + digits
		v /= 10
	}
	return digits
}
