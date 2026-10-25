// آزمون‌های جدولی دامنهٔ «فایل دستگاه تردد»: خواندن فایل، تشخیص خطای سطربه‌سطر،
// تشخیص تکراری/ناقص، ادغام شیفت شبانه، نگاشت کد دستگاه و قاعدهٔ تأخیر/تعجیل/شب‌کاری.

package domain

import (
	"strings"
	"testing"
)

// deviceFileSample یک فایل نمونه با دو کارمند، یک خط تکراری و یک روز ناقص.
const deviceFileSample = `# فایل تردد آزمون
1001,1404/07/05,07:52,ورود
1001,1404/07/05,16:35,خروج
1001,1404/07/05,07:52,ورود
1002,1404/07/05,13:48,in
1002,1404/07/05,22:05,out
1003,1404/07/05,06:05,ورود
`

func TestDeviceFileParseAndSummary(t *testing.T) {
	f := ParseDeviceFile(deviceFileSample)
	if f.Summary.Accepted != 5 {
		t.Errorf("ردیف پذیرفته باید ۵ باشد، %d شد", f.Summary.Accepted)
	}
	if f.Summary.Duplicate != 1 {
		t.Errorf("ردیف تکراری باید ۱ باشد، %d شد", f.Summary.Duplicate)
	}
	if f.Summary.Rejected != 0 {
		t.Errorf("نباید ردیف ردشده داشته باشیم، %d شد", f.Summary.Rejected)
	}
	if len(f.Punches) != 3 {
		t.Fatalf("باید ۳ زوج تردد ساخته شود، %d شد", len(f.Punches))
	}
	// خط تکراری نباید تردد تازه بسازد.
	for _, row := range f.Rows {
		if row.Status == "duplicate" && row.DeviceCode != "1001" {
			t.Errorf("ردیف تکراری باید همان کد ۱۰۰۱ باشد، %s بود", row.DeviceCode)
		}
	}
	// پیش از نگاشت، تردد بی‌نگاشت علامت می‌خورد.
	if got := f.Punches[0].Status; got != "unknownDevice" {
		t.Errorf("پیش از نگاشت وضعیت باید unknownDevice باشد، «%s» شد", got)
	}
	// پس از نگاشت، کارمند ۱۰۰۳ فقط یک تردد دارد → ناقص.
	mapped := MapDeviceCodes(f, map[string]string{"1001": "EMP-005", "1002": "EMP-008", "1003": "EMP-009"})
	byCode := map[string]DevicePunch{}
	for _, p := range mapped.Punches {
		byCode[p.DeviceCode] = p
	}
	if got := byCode["1003"]; got.Status != "incomplete" || len(got.IssuesFa) == 0 {
		t.Errorf("تردد ناقص باید incomplete با یادداشت باشد: %+v", got)
	}
	if got := byCode["1001"]; got.CheckIn != "07:52" || got.CheckOut != "16:35" || got.WorkedHours != 8.72 {
		t.Errorf("زوج تردد ۱۰۰۱ درست ساخته نشد: %+v", got)
	}
	// جهت‌های لاتین هم پذیرفته می‌شوند.
	if got := byCode["1002"]; got.CheckIn != "13:48" || got.CheckOut != "22:05" {
		t.Errorf("جهت لاتین in/out درست خوانده نشد: %+v", got)
	}
}

func TestDeviceFileRejectsBadLines(t *testing.T) {
	content := `1001,1404/07/05,07:52
1001,1404/13/05,07:52,ورود
1001,1404/07/05,25:90,ورود
1001,1404/07/05,07:52,نامعلوم
1001,1404/07/05,07:52,ورود
`
	f := ParseDeviceFile(content)
	if f.Summary.Rejected != 4 {
		t.Fatalf("باید ۴ ردیف رد شود، %d شد (خط‌ها: %+v)", f.Summary.Rejected, f.Rejected)
	}
	if f.Summary.Accepted != 1 {
		t.Errorf("فقط یک ردیف باید پذیرفته شود، %d شد", f.Summary.Accepted)
	}
	lines := map[int]bool{}
	for _, e := range f.Rejected {
		if e.MessageFa == "" {
			t.Errorf("خطای خط %d باید پیام فارسی داشته باشد", e.Line)
		}
		lines[e.Line] = true
	}
	for _, want := range []int{1, 2, 3, 4} {
		if !lines[want] {
			t.Errorf("خط %d باید در فهرست خطاها باشد", want)
		}
	}
}

func TestDeviceFileMergesNightShiftAcrossMidnight(t *testing.T) {
	// شب‌کار ساعت ۲۲:۱۰ وارد و ۰۶:۰۵ روز بعد خارج می‌شود؛ خروج در تاریخ روز بعد در
	// فایل افتاده است و باید به شیفت شبانهٔ همان روز نسبت داده شود.
	f := MapDeviceCodes(ParseDeviceFile(strings.Join([]string{
		"1001,1404/07/05,22:10,ورود",
		"1001,1404/07/06,06:05,خروج",
	}, "\n")), map[string]string{"1001": "EMP-005"})

	if len(f.Punches) != 1 {
		t.Fatalf("دو خط باید یک زوج تردد بدهد، %d زوج ساخته شد: %+v", len(f.Punches), f.Punches)
	}
	p := f.Punches[0]
	if p.Date != "1404/07/05" || p.CheckIn != "22:10" || p.CheckOut != "06:05" {
		t.Errorf("زوج تردد شبانه درست ساخته نشد: %+v", p)
	}
	if p.Status != "ok" || p.WorkedHours != 7.92 {
		t.Errorf("تردد شبانه باید کامل و ۷٫۹۲ ساعت باشد: %+v", p)
	}
	// خروج پس از ظهر روز بعد، شیفت شبانه نیست و ادغام نمی‌شود.
	g := MapDeviceCodes(ParseDeviceFile(strings.Join([]string{
		"1001,1404/07/05,07:00,ورود",
		"1001,1404/07/06,18:00,خروج",
	}, "\n")), map[string]string{"1001": "EMP-005"})
	if len(g.Punches) != 2 || g.Summary.Incomplete != 2 {
		t.Errorf("ترددی که یک روز کامل شکاف دارد نباید ادغام شود: %+v", g.Punches)
	}
}

func TestDeviceMapCodes(t *testing.T) {
	mapping, err := ParseDeviceMap("# نگاشت\n01001=EMP-005\n1002=EMP-008\n")
	if err != nil {
		t.Fatalf("نگاشت معتبر نباید خطا بدهد: %v", err)
	}
	if _, err := ParseDeviceMap("1001\n"); err == nil {
		t.Errorf("نگاشت ناقص باید خطا بدهد")
	}
	f := MapDeviceCodes(ParseDeviceFile("1001,1404/07/05,07:52,ورود\n1002,1404/07/05,07:52,ورود\n"), mapping)
	got := map[string]string{}
	for _, p := range f.Punches {
		got[p.DeviceCode] = p.EmployeeID
	}
	// «1001» با حذف صفر پیشوند به نگاشت «01001» می‌رسد.
	if got["1001"] != "EMP-005" {
		t.Errorf("کد ۱۰۰۱ باید به EMP-005 نگاشت شود، «%s» شد", got["1001"])
	}
	if got["1002"] != "EMP-008" {
		t.Errorf("کد ۱۰۰۲ باید به EMP-008 نگاشت شود، «%s» شد", got["1002"])
	}
	unmapped := UnmappedDeviceCodes(MapDeviceCodes(ParseDeviceFile("9999,1404/07/05,07:52,ورود\n"), mapping))
	if len(unmapped) != 1 || unmapped[0] != "9999" {
		t.Errorf("کد بی‌نگاشت باید ۹۹۹۹ باشد، %v شد", unmapped)
	}
}

func TestPunchMetricsLateAndEarly(t *testing.T) {
	shift := &PlanDay{Date: "1404/07/05", Shift: ShiftMorning, Start: "06:00", End: "14:00", Hours: 8}
	m := PunchMetricsFor(DevicePunch{
		Date: "1404/07/05", CheckIn: "06:20", CheckOut: "13:30", WorkedHours: 7.17, Status: "ok",
	}, shift, 15, false)
	if m.LateMinutes != 5 {
		t.Errorf("تأخیر باید ۵ دقیقه باشد (۲۰ منهای مهلت ۱۵)، %d شد", m.LateMinutes)
	}
	if m.EarlyLeaveMinutes != 15 {
		t.Errorf("تعجیل باید ۱۵ دقیقه باشد (۱۴:۰۰ منهای مهلت ۱۵ منهای ۱۳:۳۰)، %d شد", m.EarlyLeaveMinutes)
	}
	if m.NightHours != 0 || m.FridayHours != 0 {
		t.Errorf("شیفت روز نباید شب‌کاری یا جمعه‌کاری بسازد: %+v", m)
	}

	// کسی که در مهلت مجاز می‌آید تأخیر ندارد.
	m = PunchMetricsFor(DevicePunch{CheckIn: "06:12", CheckOut: "14:05", WorkedHours: 7.88}, shift, 15, false)
	if m.LateMinutes != 0 || m.EarlyLeaveMinutes != 0 {
		t.Errorf("تردد داخل مهلت مجاز نباید تأخیر/تعجیل بدهد: %+v", m)
	}
}

func TestPunchMetricsNightAndHoliday(t *testing.T) {
	night := &PlanDay{Date: "1404/07/05", Shift: ShiftNight, Start: "22:00", End: "06:00", Hours: 8, Night: true}
	// ورود ۲۱:۵۰ و خروج ۰۶:۱۰ روز بعد → ۸ ساعت کامل در بازهٔ ۲۲:۰۰ تا ۰۶:۰۰.
	m := PunchMetricsFor(DevicePunch{CheckIn: "21:50", CheckOut: "06:10", WorkedHours: 8.33}, night, 15, false)
	if m.NightHours != 8 {
		t.Errorf("شب‌کاری باید ۸ ساعت باشد، %v شد", m.NightHours)
	}
	if m.LateMinutes != 0 {
		t.Errorf("ورود پیش از شروع شیفت نباید تأخیر بسازد، %d شد", m.LateMinutes)
	}
	// کار در روز تعطیل/پایان هفته تمام ساعت کارکرد را جمعه‌کاری می‌کند.
	day := &PlanDay{Start: "08:00", End: "16:00", Hours: 8}
	holiday := PunchMetricsFor(DevicePunch{CheckIn: "08:00", CheckOut: "16:00", WorkedHours: 8}, day, 15, true)
	if holiday.FridayHours != 8 {
		t.Errorf("کار در تعطیل باید ۸ ساعت جمعه‌کاری باشد، %v شد", holiday.FridayHours)
	}
	// بازه‌ای که به بازهٔ شبانه نمی‌رسد شب‌کاری ندارد.
	if holiday.NightHours != 0 {
		t.Errorf("شیفت ۰۸:۰۰ تا ۱۶:۰۰ شب‌کاری ندارد، %v شد", holiday.NightHours)
	}
}
