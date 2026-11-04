// آزمون موتور برش‌های آماری: گروه سنی/سابقه، جدول توافقی، درصدها و شاخص‌ها.
//
// چرا این آزمون‌ها: «سکشن آمار» مبنای تصمیم‌های منابع انسانی است (هرم سنی، ریسک خروج
// تجربه، توزیع جنسیتی و تحصیلی). اگر جمع سطر/ستون یا مرز گروه‌های سنی جابه‌جا شود، عدد
// گزارش غلط می‌شود؛ پس هر دو با دادهٔ ساختگی و قطعی سنجیده می‌شوند.

package statistics

import (
	"testing"

	"hrms/internal/platform/calendar"
)

// fact یک کارمند ساختگی با مقدار بُعدهای دلخواه.
func fact(department, gender, employment, status, degree string, age int, hasAge bool, service int, hasService bool) employeeFact {
	return employeeFact{
		ID: department + gender, FullName: department + "-" + gender, Department: department,
		Gender: gender, EmploymentType: employment, Status: status, DegreeLevel: degree,
		Age: age, HasAge: hasAge, ServiceYears: service, HasService: hasService,
	}
}

// sampleFacts جمعیت نمونه: ۶ نفر با ترکیب‌های مشخص.
func sampleFacts() []employeeFact {
	return []employeeFact{
		fact("تولید", "male", "official_oil", "active", "bachelor", 45, true, 21, true),
		fact("تولید", "male", "official_worker", "active", "associate", 28, true, 4, true),
		fact("مالی", "female", "official_oil", "active", "master", 38, true, 12, true),
		fact("مالی", "female", "contractor_project", "leave", "bachelor", 33, true, 6, true),
		fact("فناوری اطلاعات", "male", "contractor_project", "probation", "diploma", 24, true, 2, true),
		// پروندهٔ ناقص: بی‌تاریخ تولد و بی‌مدرک تأییدشده.
		fact("فناوری اطلاعات", "female", "", "terminated", "", 0, false, 0, false),
	}
}

// TestWholeYearsBetween مرز سال کامل را می‌سنجد (تولد بعد از مرجع → بی‌معنا).
func TestWholeYearsBetween(t *testing.T) {
	from := calendar.Date{Year: 1370, Month: 6, Day: 15}
	cases := []struct {
		to       calendar.Date
		want     int
		wantOk   bool
		scenario string
	}{
		{calendar.Date{Year: 1404, Month: 6, Day: 15}, 34, true, "روز تولد رسیده"},
		{calendar.Date{Year: 1404, Month: 6, Day: 14}, 33, true, "یک روز قبل از تولد"},
		{calendar.Date{Year: 1404, Month: 5, Day: 1}, 33, true, "ماه تولد نرسیده"},
		{calendar.Date{Year: 1404, Month: 7, Day: 1}, 34, true, "بعد از تولد"},
		{calendar.Date{Year: 1369, Month: 1, Day: 1}, 0, false, "تاریخ پیش از تولد"},
	}
	for _, c := range cases {
		got, ok := wholeYearsBetween(from, c.to)
		if ok != c.wantOk || (ok && got != c.want) {
			t.Errorf("%s: سال کامل = %d (ok=%v)، انتظار %d (ok=%v)", c.scenario, got, ok, c.want, c.wantOk)
		}
	}
}

// TestAgeAndServiceBands گروه‌بندی سن و سابقه را روی جمعیت نمونه می‌سنجد.
func TestAgeAndServiceBands(t *testing.T) {
	facts := sampleFacts()
	ageDim, _ := dimensionByKey("age")
	serviceDim, _ := dimensionByKey("service")

	ageCounts := map[string]int{}
	for _, f := range facts {
		ageCounts[ageDim.Value(f)]++
	}
	wantAge := map[string]int{"lt30": 2, "30_39": 2, "40_49": 1, "ge50": 0, unknownKey: 1}
	for key, want := range wantAge {
		if ageCounts[key] != want {
			t.Errorf("گروه سنی %s = %d نفر، انتظار %d", key, ageCounts[key], want)
		}
	}

	serviceCounts := map[string]int{}
	for _, f := range facts {
		serviceCounts[serviceDim.Value(f)]++
	}
	wantService := map[string]int{"lt5": 2, "5_9": 1, "10_14": 1, "15_19": 0, "ge20": 1, unknownKey: 1}
	for key, want := range wantService {
		if serviceCounts[key] != want {
			t.Errorf("گروه سابقه %s = %d نفر، انتظار %d", key, serviceCounts[key], want)
		}
	}
}

// TestMatrixTotalsAndPercents جمع سطر/ستون و سه نوع درصد را می‌سنجد.
func TestMatrixTotalsAndPercents(t *testing.T) {
	facts := sampleFacts()
	ageDim, _ := dimensionByKey("age")
	genderDim, _ := dimensionByKey("gender")
	matrix := buildMatrix(facts, ageDim, genderDim, matrixTitle(ageDim, genderDim))

	if matrix.Total != 6 {
		t.Fatalf("جمع کل باید ۶ باشد، %d آمد", matrix.Total)
	}
	sumRow := 0
	for _, v := range matrix.RowTotals {
		sumRow += v
	}
	sumCol := 0
	for _, v := range matrix.ColTotals {
		sumCol += v
	}
	if sumRow != matrix.Total || sumCol != matrix.Total {
		t.Errorf("جمع سطرها (%d) و ستون‌ها (%d) باید برابر جمع کل (%d) باشد", sumRow, sumCol, matrix.Total)
	}
	// هر کارمند باید دقیقاً در یک خانه شمرده شود.
	if matrix.Mapped != matrix.Total {
		t.Errorf("شمار کارکنان شمرده‌شده %d است، انتظار %d", matrix.Mapped, matrix.Total)
	}
	// ردیف زنان: ۳ نفر (یکی پروندهٔ ناقص با جنسیت زن).
	femaleCol := -1
	for i, col := range matrix.Cols {
		if col.Key == "female" {
			femaleCol = i
		}
	}
	if femaleCol < 0 {
		t.Fatal("ستون «زن» در برش جنسیت نیست")
	}
	femaleTotal := 0
	for r := range matrix.Rows {
		femaleTotal += matrix.Cells[r][femaleCol]
	}
	if femaleTotal != 3 {
		t.Errorf("جمع ستون زنان باید ۳ باشد، %d آمد", femaleTotal)
	}
	// درست‌بودن درصدها روی یک خانهٔ مشخص: ۲ مرد زیر ۳۰ سال از ۶ نفر.
	if matrix.Peak == nil || matrix.Peak.Count != 2 {
		t.Fatalf("پرترین خانه باید ۲ نفر باشد: %#v", matrix.Peak)
	}
	if matrix.Peak.SharePct != 33.3 {
		t.Errorf("سهم خانهٔ پر = %v٪، انتظار ۳۳٫۳٪", matrix.Peak.SharePct)
	}
	if matrix.MaxCell != 2 {
		t.Errorf("بیشینهٔ خانه = %d، انتظار ۲", matrix.MaxCell)
	}
}

// TestNominalDimensionsDropEmptyCategories بررسی می‌کند بُعد اسمی دستهٔ بی‌عضو را حذف کند
// ولی بُعد ترتیبی گروه خالی را نگه دارد (تا هرم سنی کامل دیده شود).
func TestNominalDimensionsDropEmptyCategories(t *testing.T) {
	facts := sampleFacts()
	departmentDim, _ := dimensionByKey("department")
	categories := resolveCategories(departmentDim, facts)
	if len(categories) != 3 {
		t.Errorf("سه واحد در داده هست، %d دسته ساخته شد", len(categories))
	}
	ageDim, _ := dimensionByKey("age")
	ageCategories := resolveCategories(ageDim, facts)
	if len(ageCategories) != 5 {
		t.Errorf("گروه سنی باید پنج دسته (شامل گروه خالی و ثبت‌نشده) داشته باشد، %d آمد", len(ageCategories))
	}
}

// TestProfileMetrics شاخص‌های کلیدی و شاخص «پروندهٔ ناقص» را می‌سنجد.
func TestProfileMetrics(t *testing.T) {
	metrics := profileMetrics(sampleFacts())
	byKey := map[string]statMetric{}
	for _, m := range metrics {
		byKey[m.Key] = m
	}
	if got := byKey["total"].Value; got != 6 {
		t.Errorf("شمار کارکنان = %v، انتظار ۶", got)
	}
	// میانگین سن ۵ پروندهٔ دارای تاریخ تولد: (45+28+38+33+24)/5 = 33.6
	if got := byKey["avgAge"].Value; got != 33.6 {
		t.Errorf("میانگین سن = %v، انتظار ۳۳٫۶", got)
	}
	// میانگین سابقه: (21+4+12+6+2)/5 = 9
	if got := byKey["avgService"].Value; got != 9 {
		t.Errorf("میانگین سابقه = %v، انتظار ۹", got)
	}
	if got := byKey["missingBirthDate"].Value; got != 1 {
		t.Errorf("شمار بدون تاریخ تولد = %v، انتظار ۱", got)
	}
	if got := byKey["missingEducation"].Value; got != 1 {
		t.Errorf("شمار بدون مدرک تأییدشده = %v، انتظار ۱", got)
	}
	if got := byKey["femaleShare"].Value; got != 50 {
		t.Errorf("سهم بانوان = %v٪، انتظار ۵۰٪", got)
	}
	if got := byKey["topDepartment"].TextFa; got != "تولید" && got != "مالی" && got != "فناوری اطلاعات" {
		t.Errorf("پرجمعیت‌ترین واحد بی‌معنا: %q", got)
	}
}

// TestSectionPresetsCoverRequestedCombinations ترکیب‌هایی که کارفرما خواسته است.
func TestSectionPresetsCoverRequestedCombinations(t *testing.T) {
	want := map[string]bool{
		"age|service":           true,
		"education|department":  true,
		"employment|department": true,
		"gender|employment":     true,
		"gender|department":     true,
		"gender|education":      true,
	}
	presets := sectionPresets()
	found := map[string]bool{}
	for _, p := range presets {
		if _, ok := dimensionByKey(p.Rows); !ok {
			t.Errorf("بُعد سطر %s در برش %s شناخته نشد", p.Rows, p.Key)
		}
		if _, ok := dimensionByKey(p.Cols); !ok {
			t.Errorf("بُعد ستون %s در برش %s شناخته نشد", p.Cols, p.Key)
		}
		found[p.Rows+"|"+p.Cols] = true
	}
	for key := range want {
		if !found[key] {
			t.Errorf("برش «%s» در سکشن آمار نیست", key)
		}
	}
}

// TestEducationAndEmploymentCategoriesFromCatalog دسته‌های ثابت از کاتالوگ مشترک می‌آید.
func TestEducationAndEmploymentCategoriesFromCatalog(t *testing.T) {
	education := educationCategories()
	if len(education) < 6 {
		t.Errorf("دسته‌های تحصیلی از کاتالوگ مقاطع ساخته نشد: %d", len(education))
	}
	if education[0].Key != "diploma" || education[len(education)-1].Key != unknownKey {
		t.Errorf("ترتیب دسته‌های تحصیلی درست نیست: %#v", education)
	}
	employment := employmentCategories()
	if len(employment) < 3 || employment[len(employment)-1].Key != unknownKey {
		t.Errorf("دسته‌های نوع استخدام درست ساخته نشد: %d", len(employment))
	}
}
