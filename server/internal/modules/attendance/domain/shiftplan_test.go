// آزمون‌های جدولی «برنامهٔ نوبت و پوشش شیفت»: حل چرخه روی بازهٔ تاریخ، کوتاه‌شدن با
// تاریخ پایان تخصیص، رد تخصیص تأییدنشده و سنجش کمبود پوشش.

package domain

import "testing"

// planInput نمونهٔ یک تخصیص مصوب چهارشیفتی از ۱۴۰۴/۰۷/۰۱.
func planInput(status string) PlanInput {
	return PlanInput{
		AssignmentID: "SHA-1", EmployeeID: "EMP-008", EmployeeName: "مریم نوری",
		PatternTitle: "الگوی چهارشیفتی اورژانس",
		Cycle:        []string{ShiftMorning, ShiftEvening, ShiftNight, ShiftRest},
		CycleStart:   "1404/07/01", CycleOffset: 0, Status: status,
	}
}

func TestBuildPlanFollowsCycle(t *testing.T) {
	days, err := BuildPlan(planInput("approved"), "1404/07/01", "1404/07/04")
	if err != nil {
		t.Fatalf("ساخت برنامه نباید خطا بدهد: %v", err)
	}
	if len(days) != 4 {
		t.Fatalf("بازهٔ چهارروزه باید ۴ روز بدهد، %d شد", len(days))
	}
	want := []string{ShiftMorning, ShiftEvening, ShiftNight, ShiftRest}
	for i, d := range days {
		if d.Shift != want[i] {
			t.Errorf("روز %d باید شیفت «%s» باشد، «%s» شد", i+1, want[i], d.Shift)
		}
		if d.Date == "" || d.WeekdayFa == "" {
			t.Errorf("روز %d باید تاریخ و نام روز داشته باشد: %+v", i+1, d)
		}
	}
	if days[2].Night != true || days[2].Hours != 8 {
		t.Errorf("شیفت شب باید شبانه و ۸ ساعته باشد: %+v", days[2])
	}
	if !days[3].Rest || days[3].Hours != 0 {
		t.Errorf("روز چهارم باید استراحت و بدون ساعت موظفی باشد: %+v", days[3])
	}
	// چرخه جلو می‌رود: روز پنجم دوباره صبح.
	more, err := BuildPlan(planInput("approved"), "1404/07/05", "1404/07/05")
	if err != nil {
		t.Fatalf("ادامهٔ چرخه نباید خطا بدهد: %v", err)
	}
	if more[0].Shift != ShiftMorning {
		t.Errorf("روز پنجم باید صبح باشد، «%s» شد", more[0].Shift)
	}
}

func TestBuildPlanRules(t *testing.T) {
	// تخصیص تأییدنشده نوبت نمی‌سازد.
	if _, err := BuildPlan(planInput("draft"), "1404/07/01", "1404/07/03"); err == nil {
		t.Errorf("تخصیص پیش‌نویس نباید برنامه بسازد")
	}
	// بازهٔ معکوس خطای قاعده می‌دهد.
	if _, err := BuildPlan(planInput("approved"), "1404/07/05", "1404/07/01"); err == nil {
		t.Errorf("بازهٔ معکوس نباید برنامه بسازد")
	}
	// پیش از مبنای چرخه نوبتی معلق نمی‌شود: بازهٔ کاملاً پیش از آغاز تخصیص، برنامهٔ
	// خالی می‌دهد (نه خطا) تا پرس‌وجوی یک ماه کامل کارگزینی خطا نشود.
	before, err := BuildPlan(planInput("approved"), "1404/06/25", "1404/06/26")
	if err != nil {
		t.Fatalf("بازهٔ پیش از آغاز تخصیص نباید خطا بدهد: %v", err)
	}
	if len(before) != 0 {
		t.Errorf("بازهٔ پیش از آغاز تخصیص باید برنامهٔ خالی بدهد، %d روز آمد", len(before))
	}
	// بازه‌ای که از چند روز پیش از آغاز تخصیص شروع می‌شود، از روز شروع بریده می‌شود.
	clipped, err := BuildPlan(planInput("approved"), "1404/06/28", "1404/07/03")
	if err != nil {
		t.Fatalf("بازهٔ نیمه‌پیش از تخصیص نباید خطا بدهد: %v", err)
	}
	if len(clipped) != 3 || clipped[0].Date != "1404/07/01" {
		t.Errorf("برنامه باید از ۱۴۰۴/۰۷/۰۱ و سه‌روزه باشد: %+v", clipped)
	}
	// تاریخ پایان تخصیص بازه را کوتاه می‌کند.
	in := planInput("approved")
	in.EndDate = "1404/07/02"
	days, err := BuildPlan(in, "1404/07/01", "1404/07/10")
	if err != nil {
		t.Fatalf("کوتاه‌سازی بازه نباید خطا بدهد: %v", err)
	}
	if len(days) != 2 {
		t.Errorf("با تاریخ پایان ۱۴۰۴/۰۷/۰۲ باید ۲ روز بماند، %d شد", len(days))
	}
}

func TestCoverageShortfall(t *testing.T) {
	a, err := BuildPlan(planInput("approved"), "1404/07/01", "1404/07/02")
	if err != nil {
		t.Fatalf("برنامهٔ کارمند اول خطا داد: %v", err)
	}
	b, err := BuildPlan(PlanInput{
		AssignmentID: "SHA-2", EmployeeID: "EMP-005", EmployeeName: "محمد احمدی",
		Cycle: []string{ShiftMorning, ShiftRest}, CycleStart: "1404/07/01", Status: "approved",
	}, "1404/07/01", "1404/07/02")
	if err != nil {
		t.Fatalf("برنامهٔ کارمند دوم خطا داد: %v", err)
	}
	rows := Coverage(append(a, b...), 3)
	// کارمند دوم چرخهٔ دوردنی [صبح، استراحت] دارد؛ پس روز دوم شیفت عصر فقط یک نفر
	// دارد و روز استراحت هیچ ردیفی نمی‌سازد → دو ردیف پوشش.
	if len(rows) != 2 {
		t.Fatalf("دو روز × دو شیفت کاری باید ۲ ردیف پوشش بدهد، %d شد: %+v", len(rows), rows)
	}
	// روز اول: صبح دو نفر (کمبود ۱).
	first := rows[0]
	if first.Date != "1404/07/01" || first.Shift != ShiftMorning || first.Assigned != 2 {
		t.Fatalf("ردیف نخست پوشش درست نیست: %+v", first)
	}
	if first.Shortfall != 1 || first.NoteFa == "" {
		t.Errorf("کمبود پوشش روز اول باید ۱ نفر با یادداشت فارسی باشد: %+v", first)
	}
	// روز دوم: عصر یک نفر (کمبود ۲).
	if second := rows[1]; second.Shift != ShiftEvening || second.Assigned != 1 || second.Shortfall != 2 {
		t.Errorf("ردیف دوم پوشش درست نیست: %+v", second)
	}
	short := CoverageShortfalls(rows)
	if len(short) != len(rows) {
		t.Errorf("همهٔ ردیف‌ها کمبود دارند، %d از %d گزارش شد", len(short), len(rows))
	}
	// روز استراحت پوشش نمی‌سازد.
	for _, r := range rows {
		if r.Shift == ShiftRest {
			t.Errorf("روز استراحت نباید ردیف پوشش داشته باشد: %+v", r)
		}
	}
	// با حداقل پوشش صفر، کمبودی گزارش نمی‌شود.
	for _, r := range Coverage(rowsToDays(rows), 0) {
		if r.Shortfall != 0 {
			t.Errorf("با حداقل پوشش صفر نباید کمبودی باشد: %+v", r)
		}
	}
}

// rowsToDays ردیف‌های پوشش را برای سنجش دوباره به روزهای برنامه تبدیل می‌کند.
func rowsToDays(rows []CoverageRow) []PlanDay {
	out := make([]PlanDay, 0, len(rows))
	for _, r := range rows {
		for i := 0; i < r.Assigned; i++ {
			employee := ""
			if i < len(r.Employees) {
				employee = r.Employees[i]
			}
			out = append(out, PlanDay{Date: r.Date, Shift: r.Shift, EmployeeID: employee})
		}
	}
	return out
}
