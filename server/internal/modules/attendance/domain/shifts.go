// نوبت‌کاری — الگوی چرخه، تخصیص نوبت و تعویض شیفت (زیرکار ۱.۱ فاز ۱ نقشهٔ راه).
//
// هر قاعدهٔ نوبت‌کاری همین‌جا و در آزمون‌های جدولی همین پکیج نوشته و سنجیده می‌شود؛
// UI هیچ بخشی از آن را تکرار نمی‌کند. موتور مشترک `recordkit` فقط قلاب Validate را
// پس از اعتبارسنجی عمومی و پیش از ذخیره صدا می‌زند و خطای قاعده را به ۴۲۲ تبدیل می‌کند.
// مبنای محاسبه:
//
//	جایگاه(تاریخ) = (cycleOffset + فاصلهٔ روز از تاریخ شروع) mod طول چرخه
//
// یعنی چرخه از «تاریخ مؤثر» تخصیص (startDate) و جایگاهش در چرخه آغاز می‌شود و تا
// بی‌نهایت جلو می‌رود؛ پیش از آن تاریخ نوبتی معلق نمی‌شود.
package domain

import (
	"strconv"
	"strings"

	"hrms/internal/contracts"
	"hrms/internal/platform/calendar"
	"hrms/internal/platform/recordkit"
)

// کلیدهای شیفت این استقرار — واژگان مشترک چرخه، تخصیص و تعویض؛ همه جا لاتین/فارسی
// یکسان خوانده می‌شود و فقط از همین جدول مجاز است.
const (
	ShiftMorning = "صبح"
	ShiftEvening = "عصر"
	ShiftNight   = "شب"
	ShiftRest    = "استراحت"
)

// ShiftDef تعریف یک شیفت شناخته‌شده. ساعت‌ها مبنای محاسبه‌های بعدی فاز ۱ (کارکرد شب،
// فوق‌العاده، پوشش حداقلی) است؛ خودِ محاسبه در زیرکارهای ۱.۲ و ۱.۳ می‌آید.
type ShiftDef struct {
	Key     string  `json:"key"`
	TitleFa string  `json:"titleFa"`
	Start   string  `json:"start"` // HH:MM
	End     string  `json:"end"`   // HH:MM
	Hours   float64 `json:"hours"`
	Night   bool    `json:"night"` // بازهٔ شبانه؛ مبنای فوق‌العادهٔ شب
	Rest    bool    `json:"rest"`  // روز استراحت؛ نوبت کاری محسوب نمی‌شود
}

// knownShifts جدول ثابت شیفت‌های این استقرار؛ افزودن شیفت تازه فقط همین‌جاست.
func knownShifts() []ShiftDef {
	return []ShiftDef{
		{Key: ShiftMorning, TitleFa: "شیفت صبح", Start: "06:00", End: "14:00", Hours: 8},
		{Key: ShiftEvening, TitleFa: "شیفت عصر", Start: "14:00", End: "22:00", Hours: 8},
		{Key: ShiftNight, TitleFa: "شیفت شب", Start: "22:00", End: "06:00", Hours: 8, Night: true},
		{Key: ShiftRest, TitleFa: "روز استراحت", Rest: true},
	}
}

// KnownShifts فهرست شیفت‌های شناخته‌شده برای نمایش و مصرف‌کننده‌های دیگر.
func KnownShifts() []ShiftDef { return knownShifts() }

// ruleErr خطای قاعدهٔ دامنه با پیام فارسی؛ در لایهٔ HTTP به ۴۲۲ تبدیل می‌شود.
func ruleErr(msg string) *recordkit.RuleError { return &recordkit.RuleError{Message: msg} }

// ParseCycle رشتهٔ «ترتیب چرخه» را به کلیدهای شیفت می‌شکند. جداکننده: ویرگول لاتین
// یا فارسی، فاصله یا «|». قاعدهٔ دامنه: دست‌کم دو جایگاه و همهٔ کلیدها از واژگان
// شناخته‌شده — چرخهٔ نامعتبر نباید بی‌صدا نوبت بسازد.
func ParseCycle(raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, ruleErr("ترتیب چرخه نوبت الزامی است.")
	}
	valid := map[string]bool{}
	for _, sh := range knownShifts() {
		valid[sh.Key] = true
	}
	chunks := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == '،' || r == '|' || r == ' ' || r == '/'
	})
	cycle := make([]string, 0, len(chunks))
	for _, chunk := range chunks {
		key := strings.TrimSpace(chunk)
		if key == "" {
			continue
		}
		if !valid[key] {
			return nil, ruleErr("کلید شیفت «" + key + "» در جدول شیفت‌های این استقرار نیست؛ " +
				"کلیدهای مجاز: صبح، عصر، شب، استراحت.")
		}
		cycle = append(cycle, key)
	}
	if len(cycle) < 2 {
		return nil, ruleErr("الگوی چرخه باید دست‌کم دو جایگاه داشته باشد.")
	}
	return cycle, nil
}

// ResolveShiftDay نوبتِ یک تاریخ را در چرخهٔ تخصیص پیدا می‌کند و (کلید شیفت، جایگاه
// در چرخه) را برمی‌گرداند. تاریخ پیش از مبنای تخصیص خطا می‌دهد چون تخصیص هنوز شروع
// نشده است. اگر offset خارج از بازهٔ چرخه باشد mod می‌شود (قلاب Validate آن را پیشتر
// رد می‌کند؛ اینجا تدافعی است تا محاسبه هرگز panic یا جایگاه منفی نسازد).
func ResolveShiftDay(cycle []string, cycleStart string, cycleOffset int, day string) (string, int, error) {
	if len(cycle) == 0 {
		return "", 0, ruleErr("الگوی چرخه خالی است.")
	}
	start, err := calendar.Parse(cycleStart)
	if err != nil {
		return "", 0, ruleErr("تاریخ شروع چرخه معتبر نیست؛ قالب YYYY/MM/DD لازم است.")
	}
	date, err := calendar.Parse(day)
	if err != nil {
		return "", 0, ruleErr("تاریخ نوبت معتبر نیست؛ قالب YYYY/MM/DD لازم است.")
	}
	delta, err := calendar.DaysBetween(start, date)
	if err != nil {
		return "", 0, ruleErr("فاصلهٔ تاریخ نوبت از مبنای چرخه قابل محاسبه نیست.")
	}
	if delta < 0 {
		return "", 0, ruleErr("تاریخ «" + day + "» پیش از آغاز تخصیص نوبت («" + cycleStart + "») است.")
	}
	offset := cycleOffset % len(cycle)
	if offset < 0 {
		offset += len(cycle)
	}
	pos := (offset + delta) % len(cycle)
	return cycle[pos], pos, nil
}

// validateShiftPattern قاعدهٔ الگوی چرخه: ترتیب باید از واژگان معتبر ساخته شده باشد و
// حداقل پوشش (اگر پر شده) عدد صحیح غیرمنفی باشد. اجباری‌ها و قالبِ عدد را عمومیِ
// recordkit می‌سنجد.
func validateShiftPattern(in contracts.RecordInput) error {
	if _, err := ParseCycle(in.Fields["rotation"]); err != nil {
		return err
	}
	if cov := strings.TrimSpace(in.Fields["minCoverage"]); cov != "" {
		f, err := strconv.ParseFloat(calendar.NormalizeDigits(cov), 64)
		if err != nil || f < 0 || f != float64(int64(f)) {
			return ruleErr("حداقل پوشش هر شیفت باید عدد صحیح غیرمنفی باشد.")
		}
	}
	return nil
}

// validateShiftAssignment قاعدهٔ تخصیص نوبت: بازهٔ تاریخ نباید معکوس باشد و جایگاه
// آغاز در چرخه باید عدد صحیح غیرمنفی باشد. وجود خودِ الگو را نمی‌توان اینجا سنجید
// (اعتبارسنجی به انبار دسترسی ندارد)؛ حل نوبت در ResolveShiftDay الگوی ناموجود را
// کنار می‌گذارد و ارجاع الگو در زیرکار ۱.۲ هنگام سرویس تأیید می‌شود.
func validateShiftAssignment(in contracts.RecordInput) error {
	if raw := strings.TrimSpace(in.Fields["cycleOffset"]); raw != "" {
		n, err := strconv.Atoi(calendar.NormalizeDigits(raw))
		if err != nil || n < 0 {
			return ruleErr("جایگاه روز شروع در چرخه باید عدد صحیح غیرمنفی باشد.")
		}
	}
	endRaw := strings.TrimSpace(in.Fields["endDate"])
	if endRaw == "" {
		return nil
	}
	start, err := calendar.Parse(strings.TrimSpace(in.Fields["startDate"]))
	if err != nil {
		return ruleErr("تاریخ شروع تخصیص معتبر نیست؛ قالب YYYY/MM/DD لازم است.")
	}
	end, err := calendar.Parse(endRaw)
	if err != nil {
		return ruleErr("تاریخ پایان تخصیص معتبر نیست؛ قالب YYYY/MM/DD لازم است.")
	}
	if gap, err := calendar.DaysBetween(start, end); err != nil || gap < 0 {
		return ruleErr("تاریخ پایان تخصیص نباید پیش از تاریخ شروع باشد.")
	}
	return nil
}

// validateShiftSwap قاعدهٔ تعویض نوبت: همتا نباید خودِ درخواست‌دهنده باشد و دو طرف
// نباید در آن روز یک شیفت داشته باشند (تعویض بی‌اثر است). شیفت‌ها اختیاری‌اند چون
// پرکردنِ خودکارشان نیاز به خواندن تخصیص‌ها دارد (سرورِ تعویض در زیرکار بعدی)؛ اما اگر
// پر شده باشند باید از واژگان معتبر باشند.
func validateShiftSwap(in contracts.RecordInput) error {
	if _, err := calendar.Parse(strings.TrimSpace(in.Fields["swapDate"])); err != nil {
		return ruleErr("تاریخ تعویض معتبر نیست؛ قالب YYYY/MM/DD لازم است.")
	}
	partner := strings.TrimSpace(in.Fields["partnerEmployee"])
	if self := strings.TrimSpace(in.EmployeeID); partner != "" && partner == self {
		return ruleErr("کارمند مقابل نمی‌تواند خودِ درخواست‌دهنده باشد.")
	}
	my := strings.TrimSpace(in.Fields["myShift"])
	theirs := strings.TrimSpace(in.Fields["partnerShift"])
	if my != "" && theirs != "" && my == theirs {
		return ruleErr("هر دو طرف در آن روز یک شیفت دارند؛ تعویض بی‌اثر است.")
	}
	valid := map[string]bool{}
	for _, sh := range knownShifts() {
		valid[sh.Key] = true
	}
	if my != "" && !valid[my] {
		return ruleErr("شیفت شما «" + my + "» در جدول شیفت‌ها نیست؛ کلیدهای مجاز: صبح، عصر، شب، استراحت.")
	}
	if theirs != "" && !valid[theirs] {
		return ruleErr("شیفت طرف مقابل «" + theirs + "» در جدول شیفت‌ها نیست؛ کلیدهای مجاز: صبح، عصر، شب، استراحت.")
	}
	return nil
}
