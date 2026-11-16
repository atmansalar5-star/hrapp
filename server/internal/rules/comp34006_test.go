// آزمون قاعدهٔ «comp34006» در موتور قواعد.

package rules

import (
	"math"
	"strconv"
	"strings"
	"testing"
)

// TestComp34006LaddersMatchDocument جدول‌های سند را با همان اعدادی می‌سنجد که در فایل
// IMS-HR-34006 چاپ شده‌اند. تلورانس ۱ امتیاز عمدی است: خودِ جدول‌های Word در چند خانه
// یک واحد با قاعدهٔ رشد گرد شده‌اند (مثلاً گرید ۲۳ جدول ۲: سند ۷٬۵۶۲ و قاعده ۷٬۵۶۱).
func TestComp34006LaddersMatchDocument(t *testing.T) {
	p := DefaultComp34006Params()
	cases := []struct {
		ladder string
		grade  int
		want   float64
		// tol تلورانس امتیاز؛ فقط خانه‌هایی که خودِ سند در آن‌ها با قاعدهٔ رشدش نمی‌خواند
		// مقدار مثبت دارند (گردکردن دستی خانه‌های جدول Word). تفاوت‌ها در بزرگ‌ترین حالت
		// ۴ امتیاز است، یعنی ۳۰٬۰۰۰ ریال در ماه؛ در حد خطای تایپ سند.
		tol float64
	}{
		// جدول ۱ — حق گرید شغلی (رشد ۱٫۰۵)
		{"job", 1, 3100, 0},
		{"job", 2, 3255, 0},
		{"job", 8, 4362, 0},
		{"job", 13, 5567, 0},
		{"job", 14, 5846, 0},
		{"job", 21, 8225, 0},
		{"job", 22, 8636, 0},
		{"job", 23, 9068, 0},
		{"job", 24, 9522, 0},
		{"job", 25, 10000, 2},
		// جدول ۲ — حداکثر فوق‌العادهٔ پست استراتژیک (رشد ۱٫۱۵)
		{"strategic", 1, 350, 1},
		{"strategic", 8, 929, 1},
		{"strategic", 9, 1069, 1},
		{"strategic", 14, 2149, 1},
		{"strategic", 19, 4323, 1},
		{"strategic", 21, 5718, 1},
		{"strategic", 22, 6575, 1},
		{"strategic", 25, 10000, 0},
		// جدول ۳ — فوق‌العادهٔ گرید شخصی (رشد ۱٫۱۵)
		{"personal", 1, 350, 1},
		{"personal", 14, 2149, 1},
		{"personal", 21, 5718, 1},
		{"personal", 24, 8696, 1},
		{"personal", 25, 10000, 0},
		// جدول ۴ — فوق‌العادهٔ نقش (رشد ۱٫۱۵)
		{"role", 1, 140, 1},
		{"role", 14, 860, 1},
		{"role", 16, 1137, 1},
		{"role", 20, 1989, 1},
		{"role", 24, 3474, 4},
		{"role", 25, 4000, 0},
		// جدول ۵ — منطقه (رشد ۱٫۰۵)
		{"regionClimate", 1, 775, 1},
		{"regionClimate", 3, 855, 1},
		{"regionClimate", 8, 1091, 1},
		{"regionClimate", 21, 2056, 1},
		{"regionClimate", 25, 2500, 0},
		{"regionDeprivation", 1, 775, 1},
		{"regionDeprivation", 25, 2500, 0},
		{"regionHardship", 1, 930, 1},
		{"regionHardship", 4, 1077, 1},
		{"regionHardship", 21, 2468, 1},
		{"regionHardship", 25, 3000, 0},
	}
	for _, c := range cases {
		got := Comp34006Points(c.ladder, c.grade, p)
		if math.Abs(got-c.want) > c.tol {
			t.Errorf("امتیاز نردبان %s گرید %d: شد %v، سند %v (تلورانس %v)",
				c.ladder, c.grade, got, c.want, c.tol)
		}
	}
}

// TestComp34006LaddersAreIncreasing نردبان‌ها اکیداً صعودی‌اند (پایهٔ هر گرید از گرید
// پیشین بیشتر است) تا نگاشت گرید به امتیاز بی‌ابهام بماند.
func TestComp34006LaddersAreIncreasing(t *testing.T) {
	p := DefaultComp34006Params()
	for _, l := range Comp34006Ladders(p) {
		previous := 0.0
		for grade := 1; grade <= p.MaxJobGrade; grade++ {
			points := Comp34006Points(l.Key, grade, p)
			if points <= previous {
				t.Fatalf("نردبان %s در گرید %d صعودی نیست: %v پس از %v", l.Key, grade, points, previous)
			}
			previous = points
		}
	}
}

// TestComp34006DefaultIsOff مدل ۳۴۰۰۶ باید مثل مدل صندوق نفت مستقل و پیش‌فرض خاموش باشد.
func TestComp34006DefaultIsOff(t *testing.T) {
	p := DefaultComp34006Params()
	if p.PayrollApply {
		t.Fatal("پیش‌فرض «اعمال در پرداخت» مدل ۳۴۰۰۶ باید خاموش باشد.")
	}
	if text := Comp34006IndependenceFa(false); !strings.Contains(text, "خاموش") {
		t.Fatalf("متن استقلال مدل باید خاموش‌بودن را بگوید: %q", text)
	}
	if text := Comp34006IndependenceFa(true); !strings.Contains(text, "بازنویسی نمی‌شوند") {
		t.Fatalf("متن روشن‌بودن پارامتر باید سرنوشت احکام پیشین را بگوید: %q", text)
	}
	if len(Comp34006InstructionFa()) < 5 {
		t.Fatal("راهنمای شروع این مدل ناقص است.")
	}
}

// TestComp34006JobGradeAllowance حق گرید شغلی = امتیاز گرید × ضریب ریالی (بند ۶-۱).
func TestComp34006JobGradeAllowance(t *testing.T) {
	p := DefaultComp34006Params()
	out := Comp34006Calc(Comp34006Input{EmployeeID: "EMP-1", JobGrade: 25}, p)
	if !out.Valid {
		t.Fatalf("محاسبهٔ معتبر نشد: %s", out.MessageFa)
	}
	if out.JobGradePoints != 9998 {
		// نردبان جدول ۱ از گرید ۱ بسته شده است (۳٬۱۰۰ امتیاز × ۱٫۰۵)، پس گرید ۲۵ روی
		// ۹٬۹۹۸ می‌افتد؛ سند در همان خانه ۱۰٬۰۰۰ چاپ کرده که ۱۵٬۰۰۰ ریال اختلاف می‌سازد.
		t.Fatalf("امتیاز گرید ۲۵ جدول ۱ باید ۹٬۹۹۸ باشد، شد %v", out.JobGradePoints)
	}
	// ۹٬۹۹۸ امتیاز × ۷٬۵۰۰ ریال
	if want := int64(9998) * 7500; out.JobGradeAllow != want {
		t.Fatalf("حق گرید شغلی گرید ۲۵ باید %d ریال باشد، شد %d", want, out.JobGradeAllow)
	}
	// پایهٔ سنواتی صفر است، پس حقوق پایه همان حق گرید شغلی است.
	if out.BaseSalary != out.JobGradeAllow {
		t.Fatalf("حقوق پایه باید برابر حق گرید شغلی + پایهٔ سنواتی باشد: %d", out.BaseSalary)
	}
	if out.Lines[0].Code != "34601" {
		t.Fatalf("نخستین سطر باید کد ۳۴۶۰۱ باشد: %+v", out.Lines[0])
	}
}

// TestComp34006QualificationDeduction تبصرهٔ ۲ بند ۶-۱: ۵٪ کاهش بابت هر گرید شخصی پایین‌تر.
func TestComp34006QualificationDeduction(t *testing.T) {
	p := DefaultComp34006Params()
	full := Comp34006Calc(Comp34006Input{EmployeeID: "EMP-2", JobGrade: 25}, p)
	out := Comp34006Calc(Comp34006Input{EmployeeID: "EMP-2", JobGrade: 25, QualificationGapGrades: 2}, p)
	want := RoundHalfUp(float64(full.JobGradeAllow) * 0.90)
	if out.JobGradeAllow != want {
		t.Fatalf("با دو گرید کسر صلاحیت باید %d باشد، شد %d", want, out.JobGradeAllow)
	}
}

// TestComp34006StrategicPost ضریب چهار دستهٔ پست (حیاتی ۱، کلیدی ۰٫۵، محوری ۰٫۲۵، ضروری ۰).
func TestComp34006StrategicPost(t *testing.T) {
	p := DefaultComp34006Params()
	points := Comp34006Points("strategic", 21, p) // ۵٬۷۱۸ امتیاز در گرید ۲۱
	cases := []struct {
		post string
		want int64
	}{
		{"critical", RoundHalfUp(7500 * points)},
		{"key", RoundHalfUp(7500 * 0.5 * points)},
		{"pivot", RoundHalfUp(7500 * 0.25 * points)},
		{"essential", 0},
	}
	for _, c := range cases {
		out := Comp34006Calc(Comp34006Input{EmployeeID: "EMP-3", JobGrade: 21, PostClass: c.post}, p)
		if out.StrategicAllow != c.want {
			t.Errorf("پست %s: فوق‌العادهٔ استراتژیک %d شد، انتظار %d", c.post, out.StrategicAllow, c.want)
		}
	}
	// پست ضروری صفر است، پس سطرش در سند تصمیم ظاهر نمی‌شود.
	essential := Comp34006Calc(Comp34006Input{EmployeeID: "EMP-3", JobGrade: 21, PostClass: "essential"}, p)
	for _, line := range essential.Lines {
		if line.Code == "34603" {
			t.Fatal("پست ضروری نباید سطر فوق‌العادهٔ پست استراتژیک بسازد.")
		}
	}
}

// TestComp34006RoleAllowance فوق‌العادهٔ نقش = امتیاز نقش × ضریب نوع نقش × ضریب ریالی،
// حداکثر برای سه نقش (بند ۶-۵).
func TestComp34006RoleAllowance(t *testing.T) {
	p := DefaultComp34006Params()
	points := Comp34006Points("role", 20, p) // ۱٬۹۸۹ امتیاز
	one := RoundHalfUp(7500 * 0.5 * points)
	out := Comp34006Calc(Comp34006Input{EmployeeID: "EMP-4", JobGrade: 20, RoleType: "type1", RoleCount: 1}, p)
	if out.RoleAllow != one {
		t.Fatalf("نقش نوع اول در یک نقش باید %d باشد، شد %d", one, out.RoleAllow)
	}
	three := Comp34006Calc(Comp34006Input{EmployeeID: "EMP-4", JobGrade: 20, RoleType: "type1", RoleCount: 3}, p)
	if three.RoleAllow != one*3 {
		t.Fatalf("سه نقش باید سه برابر یک نقش باشد: %d در برابر %d", three.RoleAllow, one*3)
	}
	// سند بیش از سه نقش را نمی‌پذیرد؛ سقف پارامتر باید اثر کند.
	five := Comp34006Calc(Comp34006Input{EmployeeID: "EMP-4", JobGrade: 20, RoleType: "type1", RoleCount: 5}, p)
	if five.RoleAllow != one*3 {
		t.Fatalf("شمار نقش‌ها باید به سقف پارامتر (۳) محدود شود: %d", five.RoleAllow)
	}
}

// TestComp34006MarketAttraction سقف فوق‌العادهٔ جذب بازار کار = معادل فوق‌العادهٔ گرید شخصی.
func TestComp34006MarketAttraction(t *testing.T) {
	p := DefaultComp34006Params()
	base := Comp34006Calc(Comp34006Input{EmployeeID: "EMP-5", JobGrade: 25, PersonalGrade: 25}, p)
	if base.PersonalAllow != 75_000_000 {
		t.Fatalf("فوق‌العادهٔ گرید شخصی گرید ۲۵ باید ۷۵٬۰۰۰٬۰۰۰ باشد، شد %d", base.PersonalAllow)
	}
	full := Comp34006Calc(Comp34006Input{
		EmployeeID: "EMP-5", JobGrade: 25, PersonalGrade: 25, MarketAttractionApproved: true,
	}, p)
	if full.MarketAllow != base.PersonalAllow {
		t.Fatalf("جذب بازار کار با تصویب کامل باید معادل گرید شخصی باشد: %d", full.MarketAllow)
	}
	// درخواست بیش از سقف پارامتر نباید از سقف بگذرد.
	over := Comp34006Calc(Comp34006Input{
		EmployeeID: "EMP-5", JobGrade: 25, PersonalGrade: 25,
		MarketAttractionApproved: true, MarketAttractionPercent: 400,
	}, p)
	if over.MarketAllow != base.PersonalAllow {
		t.Fatalf("درصد بیش از سقف باید به سقف پارامتر محدود شود: %d", over.MarketAllow)
	}
}

// TestComp34006RegionAndOffshore فوق‌العادهٔ منطقه و اقماری (بندهای ۶-۷ و ۶-۱۷).
func TestComp34006RegionAndOffshore(t *testing.T) {
	p := DefaultComp34006Params()
	// گرید ۲۵: حاصل جمع امتیاز سه ستون جدول ۵ = ۲٬۵۰۰ + ۲٬۵۰۰ + ۳٬۰۰۰ = ۸٬۰۰۰ امتیاز.
	plain := Comp34006Calc(Comp34006Input{EmployeeID: "EMP-6", JobGrade: 25}, p)
	tablePart := float64(7500) * 8000 // ۶۰٬۰۰۰٬۰۰۰ ریال
	jobPart := float64(plain.JobGradeAllow) * 0.80
	both := Comp34006Calc(Comp34006Input{
		EmployeeID: "EMP-6", JobGrade: 25, Region: "mahshahr", OffshoreDays: 7,
	}, p)
	if want := RoundHalfUp(tablePart + jobPart); both.RegionAllow != want {
		t.Fatalf("منطقهٔ ماهشهر با حالت «both» باید %d باشد، شد %d", want, both.RegionAllow)
	}
	if both.OffshoreAllow != RoundHalfUp(float64(both.RegionAllow)/14*7) {
		t.Fatalf("اقماری هفت روز باید نصف فوق‌العادهٔ منطقه باشد: %d", both.OffshoreAllow)
	}
	table := Comp34006Calc(Comp34006Input{EmployeeID: "EMP-6", JobGrade: 25, Region: "mahshahr"}, p)
	// ماهشهر: ۱۰۰٪ جدول ۵ و ۸۰٪ حق گرید شغلی؛ پس حالت «both» همان قاعدهٔ سند است.
	if table.RegionAllow != both.RegionAllow {
		t.Fatalf("حالت پیش‌فرض «both» باید همان جمع دو بخش باشد: %d در برابر %d", table.RegionAllow, both.RegionAllow)
	}
	tehran := Comp34006Calc(Comp34006Input{EmployeeID: "EMP-6", JobGrade: 25, Region: "tehran"}, p)
	wantTehran := RoundHalfUp(7500*0.6875*8000 + float64(plain.JobGradeAllow)*0.55)
	if tehran.RegionAllow != wantTehran {
		t.Fatalf("منطقهٔ تهران باید %d باشد، شد %d", wantTehran, tehran.RegionAllow)
	}
	// بدون منطقه، اقماری هم نباید ساخته شود و باید هشدار بدهد.
	none := Comp34006Calc(Comp34006Input{EmployeeID: "EMP-6", JobGrade: 25, OffshoreDays: 7}, p)
	if none.RegionAllow != 0 || none.OffshoreAllow != 0 {
		t.Fatalf("بدون منطقه نباید فوق‌العادهٔ منطقه یا اقماری بسازد: %d / %d", none.RegionAllow, none.OffshoreAllow)
	}
	if len(none.WarningsFa) == 0 {
		t.Fatal("نبود منطقه با درخواست اقماری باید هشدار بدهد.")
	}
}

// TestComp34006BaseDrivenItems اقلامی که بر حقوق پایه سوارند: اضافه‌کاری، نوبت‌کاری،
// مأموریت و تعطیل‌کاری (بندهای ۶-۱۴ تا ۶-۱۸).
func TestComp34006BaseDrivenItems(t *testing.T) {
	p := DefaultComp34006Params()
	in := Comp34006Input{
		EmployeeID: "EMP-7", JobGrade: 25,
		MonthlyWorkDays: 30, OvertimeHours: 10, ShiftPattern: "company", ShiftDays: 30,
		MissionDays: 5, HolidayHours: 10, FridayHours: 10,
	}
	out := Comp34006Calc(in, p)
	if out.BaseSalary != out.JobGradeAllow {
		t.Fatalf("حقوق پایه باید برابر حق گرید شغلی + پایهٔ سنواتی باشد: %d", out.BaseSalary)
	}
	base := float64(out.BaseSalary)
	daily := base / 30
	hourly := daily / 7.33
	// ۳۵٪ توافقی شرکت برای ۳۰ روز = ۳۵٪ حقوق پایه.
	if want := RoundHalfUp(base * 0.35); out.ShiftAllow != want {
		t.Fatalf("نوبت‌کاری ۳۵٪ تمام ماه باید %d باشد، شد %d", want, out.ShiftAllow)
	}
	// مأموریت: معادل یک روز حقوق پایه برای هر روز.
	if want := RoundHalfUp(daily * 5); out.MissionAllow != want {
		t.Fatalf("مأموریت ۵ روز باید %d باشد، شد %d", want, out.MissionAllow)
	}
	// اضافه‌کاری: (حقوق پایه ÷ ۳۰ ÷ ۷٫۳۳) × ۱٫۴ × ۱۰ ساعت.
	if got, want := out.Overtime, RoundHalfUp(hourly*1.4*10); got != want {
		t.Fatalf("اضافه‌کاری ۱۰ ساعت باید %d باشد، شد %d", want, got)
	}
	if out.Overtime == 0 {
		t.Fatal("اضافه‌کاری محاسبه نشد.")
	}
	// تعطیل‌کاری: ۴۰٪ روزهای هفته و ۸۰٪ جمعه، علاوه بر حقوق آن روز.
	weekday := RoundHalfUp(hourly * 10 * 1.4)
	friday := RoundHalfUp(hourly * 10 * 1.8)
	if out.HolidayAllow != weekday+friday {
		t.Fatalf("تعطیل‌کاری باید %d باشد، شد %d", weekday+friday, out.HolidayAllow)
	}
	// جمع اقلام باید برابر جمع سطرها باشد.
	sum := int64(0)
	for _, line := range out.Lines {
		sum += line.Amount
	}
	if sum != out.Gross {
		t.Fatalf("جمع سطرها (%d) با جمع اعلام‌شده (%d) نمی‌خواند.", sum, out.Gross)
	}
}

// TestComp34006PersianParameterText پارامترهای نوشته‌شده با ارقام فارسی هم خوانده شوند.
func TestComp34006PersianParameterText(t *testing.T) {
	p := Comp34006ParamsFromText(map[string]string{
		"comp34006.rialFactor":        "۸٬۵۰۰",
		"comp34006.role.type1Percent": "۵۵٪",
		"comp34006.payroll.apply":     "۱",
		"comp34006.region.mode":       "2",
		"comp34006.base.includeFixed": "1",
		"comp34006.job.growth":        "1.06",
		"comp34006.other.unknown.key": "999",
		"payroll.overtimeFactor":      "1.4",
	})
	if p.RialFactor != 8500 {
		t.Fatalf("ضریب ریالی از متن فارسی خوانده نشد: %d", p.RialFactor)
	}
	if p.RoleType1Percent != 55 {
		t.Fatalf("درصد نقش نوع اول: %v", p.RoleType1Percent)
	}
	if !p.PayrollApply {
		t.Fatal("پارامتر اعمال در پرداخت باید روشن شود.")
	}
	if p.RegionMode != "jobGrade" {
		t.Fatalf("حالت منطقه باید jobGrade باشد: %q", p.RegionMode)
	}
	if !p.BaseIncludeFixed {
		t.Fatal("پارامتر جمع عدد ثابت در حقوق پایه باید روشن شود.")
	}
	if p.JobGrowth != 1.06 {
		t.Fatalf("نرخ رشد جدول ۱ باید ۱٫۰۶ باشد: %v", p.JobGrowth)
	}
	// کلید نامربوط (نه comp34006.*) نباید اثری بگذارد.
	if p.OvertimeFactor != 1.4 {
		t.Fatalf("پارامتر پیش‌فرض اضافه‌کاری تغییر کرد: %v", p.OvertimeFactor)
	}
}

// TestComp34006InvalidGrade گرید بیرون از نردبان باید پیام فارسی بدهد، نه عدد بی‌معنا.
func TestComp34006InvalidGrade(t *testing.T) {
	p := DefaultComp34006Params()
	for _, grade := range []int{0, -3, 26, 99} {
		out := Comp34006Calc(Comp34006Input{EmployeeID: "EMP-8", JobGrade: grade}, p)
		if out.Valid {
			t.Fatalf("گرید %d نباید معتبر باشد.", grade)
		}
		if out.MessageFa == "" {
			t.Fatalf("گرید %d باید پیام فارسی داشته باشد.", grade)
		}
	}
}

// TestComp34006CodesCoverDocument هر ۱۸ کد سند در موتور شناخته شده است.
func TestComp34006CodesCoverDocument(t *testing.T) {
	out := Comp34006Calc(Comp34006Input{
		EmployeeID: "EMP-9", JobGrade: 25, PersonalGrade: 25, PostClass: "critical",
		RoleType: "type1", RoleCount: 1, Region: "mahshahr", MarketAttractionApproved: true,
		PerformanceScore: 100,
		ChildrenCount:    2, ShiftPattern: "company", ShiftDays: 30, OvertimeHours: 10,
		MissionDays: 5, OffshoreDays: 7, HolidayHours: 10, FridayHours: 10,
		MonthlyWorkDays: 30,
	}, Comp34006ParamsFromValues(map[string]float64{
		"comp34006.seniority.amount":    3_000_000,
		"comp34006.fixedMonthly.amount": 5_000_000,
		"comp34006.child.amount":        2_000_000,
		"comp34006.housing.amount":      4_000_000,
		"comp34006.grocery.amount":      6_000_000,
		"comp34006.performance.percent": 20,
	}))
	seen := map[string]bool{}
	for _, line := range out.Lines {
		seen[line.Code] = true
	}
	// کد ۳۴۶۱۳ (تفاوت تطبیق) در سند صفر است، پس در فهرست نمی‌آید؛ بقیه باید باشند.
	for code := 34601; code <= 34618; code++ {
		key := strconv.Itoa(code)
		if code == 34613 {
			continue
		}
		if !seen[key] {
			t.Errorf("کد %s در سطرهای محاسبه نیامد.", key)
		}
	}
	if seen["34613"] {
		t.Error("تفاوت تطبیق در طرح G2025 نباید پرداخت شود.")
	}
	if out.Gross <= 0 {
		t.Fatal("جمع اقلام باید مثبت باشد.")
	}
}

// c34006PayslipInput ورودی نمونهٔ فیش ماهانه برای آزمون‌های این بخش.
func c34006PayslipInput() Comp34006PayslipInput {
	return Comp34006PayslipInput{
		Comp34006Input: Comp34006Input{
			EmployeeID: "EMP-10", EmployeeName: "کارمند نمونه",
			JobGrade: 25, PersonalGrade: 22, PostClass: "key", Region: "mahshahr",
			ShiftPattern: "company", ShiftDays: 30, OvertimeHours: 10, MissionDays: 2,
			ChildrenCount: 2, MonthlyWorkDays: 30,
		},
		PeriodFa: "1405/08",
	}
}

// TestComp34006PayslipBalances فیش ماهانه باید بسته باشد: جمع اقلام = ناخالص، کسور =
// بیمه + مالیات + سایر، و خالص = ناخالص − کسور.
func TestComp34006PayslipBalances(t *testing.T) {
	slip := Comp34006PayslipFor(c34006PayslipInput(), DefaultComp34006Params(), DefaultPayrollParams())
	if !slip.Valid {
		t.Fatalf("فیش معتبر نشد: %s", slip.MessageFa)
	}

	sum := int64(0)
	for _, line := range slip.Earnings {
		if line.Amount < 0 {
			t.Fatalf("قلم بستانکار منفی: %+v", line)
		}
		sum += line.Amount
	}
	if sum != slip.Gross {
		t.Fatalf("جمع اقلام (%d) با ناخالص فیش (%d) نمی‌خواند.", sum, slip.Gross)
	}
	if slip.TotalDeductions != slip.InsuranceEmployee+slip.Tax+slip.OtherDeductions {
		t.Fatalf("جمع کسور نادرست است: %d", slip.TotalDeductions)
	}
	if slip.Net != slip.Gross-slip.TotalDeductions {
		t.Fatalf("خالص پرداختی نادرست است: %d", slip.Net)
	}
	if slip.EmployerCost != slip.Gross+slip.InsuranceEmployer {
		t.Fatalf("هزینهٔ کارفرما باید ناخالص + سهم کارفرما باشد: %d", slip.EmployerCost)
	}
	if slip.Net <= 0 {
		t.Fatalf("خالص پرداختی باید مثبت باشد: %d", slip.Net)
	}

	seen := map[string]bool{}
	for _, d := range slip.Deductions {
		seen[d.Code] = true
	}
	if !seen[Comp34006DeductionInsurance] || !seen[Comp34006DeductionTax] {
		t.Fatalf("سطرهای بیمه و مالیات باید در فیش باشند: %+v", slip.Deductions)
	}
	if slip.PeriodFa != "1405/08" {
		t.Fatalf("دورهٔ فیش درست نیست: %q", slip.PeriodFa)
	}
	if slip.SummaryFa == "" {
		t.Fatal("جمع‌بندی فارسی فیش لازم است.")
	}
}

// TestComp34006PayslipStatutoryParity کسور فیش ۳۴۰۰۶ باید از همان قاعدهٔ مشترک سه نظام
// پرداخت بیاید، نه قاعدهٔ جداگانه.
func TestComp34006PayslipStatutoryParity(t *testing.T) {
	pp := DefaultPayrollParams()
	slip := Comp34006PayslipFor(c34006PayslipInput(), DefaultComp34006Params(), pp)
	ref := StatutoryDeductions(slip.Gross, pp, nil)

	if slip.InsuranceBase != ref.InsuranceBase || slip.InsuranceEmployee != ref.InsuranceEmployee ||
		slip.InsuranceEmployer != ref.InsuranceEmployer || slip.TaxableIncome != ref.TaxableIncome ||
		slip.Tax != ref.Tax {
		t.Fatalf("کسور فیش با قاعدهٔ مشترک نمی‌خواند: فیش %+v در برابر %+v", slip, ref)
	}
}

// TestComp34006PayslipInsuranceCeiling سقف بیمه باید مثل فیش نظام جاری اعمال شود.
func TestComp34006PayslipInsuranceCeiling(t *testing.T) {
	pp := DefaultPayrollParams()
	pp.InsuranceCeiling = 50_000_000
	slip := Comp34006PayslipFor(c34006PayslipInput(), DefaultComp34006Params(), pp)
	if slip.Gross <= pp.InsuranceCeiling {
		t.Fatalf("برای این آزمون ناخالص باید از سقف بیمه بیشتر باشد: %d", slip.Gross)
	}
	if slip.InsuranceBase != pp.InsuranceCeiling {
		t.Fatalf("مبنای بیمه باید به سقف محدود شود: %d", slip.InsuranceBase)
	}
	if want := RoundHalfUp(float64(pp.InsuranceCeiling) * pp.InsuranceEmployeeRate); slip.InsuranceEmployee != want {
		t.Fatalf("سهم کارمند از بیمه باید %d باشد، شد %d", want, slip.InsuranceEmployee)
	}
}

// TestComp34006PayslipOtherDeductions سایر کسور باید دقیقاً از خالص کم شود.
func TestComp34006PayslipOtherDeductions(t *testing.T) {
	cp, pp := DefaultComp34006Params(), DefaultPayrollParams()
	base := Comp34006PayslipFor(c34006PayslipInput(), cp, pp)

	in := c34006PayslipInput()
	in.OtherDeductions = 7_000_000
	withOther := Comp34006PayslipFor(in, cp, pp)

	if diff := base.Net - withOther.Net; diff != 7_000_000 {
		t.Fatalf("سایر کسور باید خالص را دقیقاً ۷٬۰۰۰٬۰۰۰ کم کند؛ اختلاف: %d", diff)
	}
	if withOther.TotalDeductions != base.TotalDeductions+7_000_000 {
		t.Fatalf("جمع کسور با سایر کسور جمع نشد: %d", withOther.TotalDeductions)
	}
}

// TestComp34006PayslipVeteranRelief امتیاز ایثارگری باید مالیات را کم کند و در فیش بیاید.
func TestComp34006PayslipVeteranRelief(t *testing.T) {
	cp, pp := DefaultComp34006Params(), DefaultPayrollParams()
	plain := Comp34006PayslipFor(c34006PayslipInput(), cp, pp)

	in := c34006PayslipInput()
	in.Veteran = &VeteranEntitlement{Group: "disabled", GroupFa: "جانباز", Approved: true,
		IsVeteran: true, TaxExempt: 50, PensionExempt: 50}
	relieved := Comp34006PayslipFor(in, cp, pp)

	if !relieved.VeteranApplied {
		t.Fatal("امتیاز ایثارگری روی فیش اعمال نشد.")
	}
	if relieved.Tax >= plain.Tax {
		t.Fatalf("معافیت ایثارگری باید مالیات را کم کند: %d در برابر %d", relieved.Tax, plain.Tax)
	}
	if relieved.Net <= plain.Net {
		t.Fatalf("با معافیت، خالص پرداختی باید بیشتر شود: %d در برابر %d", relieved.Net, plain.Net)
	}
}

// TestComp34006PayslipInvalidGrade فیش با گرید نامعتبر نباید ساخته شود.
func TestComp34006PayslipInvalidGrade(t *testing.T) {
	in := c34006PayslipInput()
	in.JobGrade = 99
	slip := Comp34006PayslipFor(in, DefaultComp34006Params(), DefaultPayrollParams())
	if slip.Valid {
		t.Fatal("گرید ۹۹ نباید فیش معتبر بسازد.")
	}
	if slip.MessageFa == "" {
		t.Fatal("پیام فارسی لازم است.")
	}
	if len(slip.Earnings) != 0 || len(slip.Deductions) != 0 {
		t.Fatal("فیش نامعتبر نباید سطر بسازد.")
	}
}

// TestComp34006AllParamsDriveCalculation هیچ عدد سیاستی داخل کد نمانده باشد: پارامترهای
// کامل‌کنندهٔ مدل (مبنای روزهای ماه، سقف گرید شخصی، درصد پیش‌فرض جذب بازار، نرخ رشد
// جدول‌ها و درصد نوبت‌کاری هر ترکیب شیفت) باید روی خروجی اثر بگذارند.
func TestComp34006AllParamsDriveCalculation(t *testing.T) {
	base := DefaultComp34006Params()

	// ۱) مبنای روزهای ماه: با ۲۶ روز، مزد روزانه بزرگ‌تر می‌شود و نوبت‌کاری تمام ماه برابر
	//    درصد نوبت‌کاری × حقوق پایه می‌شود.
	in := Comp34006Input{EmployeeID: "EMP-11", JobGrade: 25, MissionDays: 1,
		ShiftPattern: "company", ShiftDays: 26}
	def := Comp34006Calc(in, base)
	twentySix := Comp34006ParamsFromValues(map[string]float64{"comp34006.monthlyDays": 26})
	alt := Comp34006Calc(in, twentySix)
	if alt.MissionAllow <= def.MissionAllow {
		t.Fatalf("با مبنای ۲۶ روز، مأموریت باید بیشتر شود: %d در برابر %d", alt.MissionAllow, def.MissionAllow)
	}
	if want := RoundHalfUp(float64(alt.BaseSalary) * 0.35); alt.ShiftAllow != want {
		t.Fatalf("نوبت‌کاری تمام ماه باید ۳۵٪ حقوق پایه باشد (%d)، شد %d", want, alt.ShiftAllow)
	}

	// ۲) سقف گرید شخصی: گرید بالاتر از سقف به سقف محدود می‌شود.
	capped := Comp34006ParamsFromValues(map[string]float64{"comp34006.maxPersonalGrade": 10})
	high := Comp34006Calc(Comp34006Input{EmployeeID: "EMP-11", JobGrade: 25, PersonalGrade: 25}, capped)
	atCap := Comp34006Calc(Comp34006Input{EmployeeID: "EMP-11", JobGrade: 25, PersonalGrade: 10}, capped)
	if high.PersonalGrade != 10 || high.PersonalAllow != atCap.PersonalAllow {
		t.Fatalf("سقف گرید شخصی اعمال نشد: گرید %d و مبلغ %d", high.PersonalGrade, high.PersonalAllow)
	}

	// ۳) درصد پیش‌فرض جذب بازار کار.
	half := Comp34006ParamsFromValues(map[string]float64{"comp34006.market.defaultPercent": 50})
	market := Comp34006Calc(Comp34006Input{EmployeeID: "EMP-11", JobGrade: 25,
		PersonalGrade: 20, MarketAttractionApproved: true}, half)
	if want := RoundHalfUp(float64(market.PersonalAllow) * 0.5); market.MarketAllow != want {
		t.Fatalf("جذب بازار با پیش‌فرض ۵۰٪ باید %d باشد، شد %d", want, market.MarketAllow)
	}

	// ۴) نرخ رشد جدول‌ها: لنگر گرید آخر ثابت می‌ماند ولی گریدهای پایین‌تر کم‌امتیازتر می‌شوند.
	steeper := Comp34006ParamsFromValues(map[string]float64{"comp34006.strategic.growth": 1.20})
	if got := Comp34006Points("strategic", 25, steeper); got != 10000 {
		t.Fatalf("لنگر گرید آخر جدول ۲ باید ثابت بماند: %v", got)
	}
	if Comp34006Points("strategic", 20, steeper) >= Comp34006Points("strategic", 20, base) {
		t.Fatal("با نرخ رشد بیشتر، امتیاز گرید پایین‌تر باید کمتر شود.")
	}

	// ۵) درصد نوبت‌کاری هر ترکیب شیفت از پارامتر شرکت می‌آید.
	shift := Comp34006ParamsFromValues(map[string]float64{"comp34006.shift.morningEveningPercent": 12})
	if got := Comp34006ShiftPercent("morning_evening", shift); got != 12 {
		t.Fatalf("درصد شیفت صبح و عصر باید ۱۲ باشد: %v", got)
	}
	if got := Comp34006ShiftPercent("morning_evening", base); got != 10 {
		t.Fatalf("پیش‌فرض شیفت صبح و عصر باید ۱۰ باشد: %v", got)
	}
}
