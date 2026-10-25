// ورود فایل دستگاه تردد — زیرکار ۱.۲ فاز ۱ نقشهٔ راه.
//
// این فایل «قالب فایل تردد»، اعتبارسنجی سطربه‌سطر، تشخیص تکراری/ناقص، نگاشت
// «کد دستگاه ← کد پرسنلی» و قاعدهٔ تأخیر/تعجیل/شبکاری/جمعه‌کاری را در دامنه نگه
// می‌دارد. هیچ‌کدام از این قاعده‌ها در رابط کاربری تکرار نمی‌شود.
//
// قالب پیش‌فرض فایل (قالب مشترک خروجی دستگاه‌های حضور):
//
//	# هر خط: کد دستگاه، تاریخ، ساعت، جهت
//	1001,1404/07/05,07:52,ورود
//	1001,1404/07/05,16:35,خروج
//
// جداکننده می‌تواند ویرگول، نقطه‌ویرگول، خط تیرهٔ عمودی یا تب باشد؛ خط‌های آغازشده با
// «#» توضیح‌اند و نادیده گرفته می‌شوند. ارقام فارسی/عربی هم پذیرفته می‌شود.
package domain

import (
	"sort"
	"strconv"
	"strings"

	"hrms/internal/platform/calendar"
)

// DeviceFileTemplate قالب نمونهٔ فایل تردد که مسیر `/device-import/template` سرو
// می‌کند؛ همان چیزی که کارگزینی برای گرفتن خروجی از دستگاه به پیمانکار می‌دهد.
const DeviceFileTemplate = `# فایل تردد دستگاه — ستون‌ها: کد دستگاه، تاریخ، ساعت، جهت
# تاریخ شمسی به قالب YYYY/MM/DD و ساعت به قالب HH:MM
# جهت: ورود یا خروج (in/out هم پذیرفته می‌شود)
1001,1404/07/05,07:52,ورود
1001,1404/07/05,16:35,خروج
1002,1404/07/05,13:48,ورود
1002,1404/07/05,22:05,خروج
1003,1404/07/05,06:05,ورود
1003,1404/07/05,14:02,خروج
`

// DeviceRowError خطای یک خط فایل با شمارهٔ خط و پیام فارسی.
type DeviceRowError struct {
	Line      int    `json:"line"`
	Raw       string `json:"raw,omitempty"`
	MessageFa string `json:"messageFa"`
}

// DeviceRow یک تردد خام از فایل دستگاه.
type DeviceRow struct {
	Line       int    `json:"line"`
	DeviceCode string `json:"deviceCode"`
	Date       string `json:"date"`
	Time       string `json:"time"`
	Direction  string `json:"direction"`
	// EmployeeID پس از نگاشت «کد دستگاه ← کد پرسنلی» پر می‌شود.
	EmployeeID string `json:"employeeId,omitempty"`
	// Status وضعیت خط: ok یا duplicate.
	Status string `json:"status"`
}

// DevicePunch تردد یک کارمند در یک روز (زوج ورود/خروج ساخته‌شده از خط‌های خام).
type DevicePunch struct {
	DeviceCode  string  `json:"deviceCode"`
	EmployeeID  string  `json:"employeeId,omitempty"`
	Date        string  `json:"date"`
	CheckIn     string  `json:"checkIn,omitempty"`
	CheckOut    string  `json:"checkOut,omitempty"`
	WorkedHours float64 `json:"workedHours"`
	// Status وضعیت زوج تردد: ok، incomplete (ناقص)، unknownDevice (بی‌نگاشت)، invalid.
	Status string `json:"status"`
	// PunchCount شمار ترددهای خام این روز و IssuesFa فهرست مغایرت‌های همان روز.
	PunchCount int      `json:"punchCount"`
	IssuesFa   []string `json:"issuesFa,omitempty"`
}

// DeviceFileSummary جمع فایل: همان چیزی که گزارش مغایرت نشان می‌دهد.
type DeviceFileSummary struct {
	Lines         int `json:"lines"`
	Accepted      int `json:"accepted"`
	Duplicate     int `json:"duplicate"`
	Rejected      int `json:"rejected"`
	UnknownDevice int `json:"unknownDevice"`
	Punches       int `json:"punches"`
	OkPunches     int `json:"okPunches"`
	Incomplete    int `json:"incomplete"`
	Invalid       int `json:"invalid"`
	Employees     int `json:"employees"`
}

// DeviceFile نتیجهٔ تحلیل یک فایل تردد.
type DeviceFile struct {
	Rows     []DeviceRow       `json:"rows"`
	Rejected []DeviceRowError  `json:"rejected"`
	Punches  []DevicePunch     `json:"punches"`
	Summary  DeviceFileSummary `json:"summary"`
}

// ParseDeviceFile محتوای فایل تردد را سطربه‌سطر می‌خواند و اعتبارسنجی می‌کند.
//
// قاعده‌ها: خط ناقص، تاریخ نامعتبر، ساعت نامعتبر و جهت ناشناخته رد می‌شوند و خط
// تکرارشده (همان کد، تاریخ، ساعت و جهت) به‌جای ساخت تردد تازه، «تکراری» علامت
// می‌خورد تا دوباره‌گرفتن فایل، تردد را دو بار ثبت نکند.
func ParseDeviceFile(content string) DeviceFile {
	out := DeviceFile{Rows: []DeviceRow{}, Rejected: []DeviceRowError{}, Punches: []DevicePunch{}}
	seen := map[string]bool{}
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	for i, raw := range lines {
		out.Summary.Lines++
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		lineNo := i + 1
		cols := strings.FieldsFunc(line, func(r rune) bool {
			return r == ',' || r == ';' || r == '|' || r == '\t'
		})
		clean := make([]string, 0, len(cols))
		for _, c := range cols {
			if v := strings.TrimSpace(c); v != "" {
				clean = append(clean, v)
			}
		}
		if len(clean) < 4 {
			out.Rejected = append(out.Rejected, DeviceRowError{Line: lineNo, Raw: line,
				MessageFa: "خط باید چهار ستون داشته باشد: کد دستگاه، تاریخ، ساعت، جهت."})
			continue
		}
		code := calendar.NormalizeDigits(clean[0])
		dateRaw := strings.ReplaceAll(calendar.NormalizeDigits(clean[1]), "-", "/")
		date, err := calendar.Parse(dateRaw)
		if err != nil {
			out.Rejected = append(out.Rejected, DeviceRowError{Line: lineNo, Raw: line,
				MessageFa: "تاریخ «" + clean[1] + "» معتبر نیست؛ قالب YYYY/MM/DD لازم است."})
			continue
		}
		clock := calendar.NormalizeDigits(clean[2])
		if total, ok := clockMinutes(clock); !ok || total < 0 || total >= 24*60 {
			out.Rejected = append(out.Rejected, DeviceRowError{Line: lineNo, Raw: line,
				MessageFa: "ساعت «" + clean[2] + "» معتبر نیست؛ قالب HH:MM لازم است."})
			continue
		}
		dir, ok := deviceDirection(clean[3])
		if !ok {
			out.Rejected = append(out.Rejected, DeviceRowError{Line: lineNo, Raw: line,
				MessageFa: "جهت «" + clean[3] + "» شناخته نشد؛ «ورود» یا «خروج» لازم است."})
			continue
		}
		key := code + "|" + date.String() + "|" + clock + "|" + dir
		status := "ok"
		if seen[key] {
			status = "duplicate"
		}
		seen[key] = true
		out.Rows = append(out.Rows, DeviceRow{
			Line: lineNo, DeviceCode: code, Date: date.String(),
			Time: normalizeClock(clock), Direction: dir, Status: status,
		})
	}
	out.Punches = mergeCrossMidnight(buildPunches(out.Rows))
	out.recount()
	return out
}

// ParseDeviceMap نگاشت «کد دستگاه ← کد پرسنلی» را از متن ساده می‌خواند.
//
// هر خط یک نگاشت است با جداکنندهٔ «=»، «:» یا ویرگول؛ خط خالی و خط توضیح («#») مجاز
// است. نبود متن یعنی نگاشت خالی (در آن حالت فقط کدهای هم‌نام کد پرسنلی می‌مانند).
func ParseDeviceMap(raw string) (map[string]string, error) {
	out := map[string]string{}
	if strings.TrimSpace(raw) == "" {
		return out, nil
	}
	for i, line := range strings.Split(strings.ReplaceAll(raw, "\r\n", "\n"), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.FieldsFunc(line, func(r rune) bool { return r == '=' || r == ':' || r == ',' || r == '\t' })
		if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
			return nil, ruleErr("نگاشت «کد دستگاه ← کد پرسنلی» در خط " + itoaFa(i+1) +
				" معتبر نیست؛ قالب درست: 1001=EMP-001")
		}
		key := calendar.NormalizeDigits(strings.TrimSpace(parts[0]))
		id := strings.TrimSpace(parts[1])
		out[key] = id
		// دستگاه‌ها یک کد را گاهی با صفر پیشوند و گاهی بدون آن چاپ می‌کنند؛ هر دو شکل
		// نگاشت می‌شود تا کارگزینی مجبور نباشد صفرها را دستی اصلاح کند.
		if trimmed := strings.TrimLeft(key, "0"); trimmed != key && trimmed != "" {
			if _, exists := out[trimmed]; !exists {
				out[trimmed] = id
			}
		}
	}
	if len(out) == 0 {
		return out, nil
	}
	return out, nil
}

// MapDeviceCodes کد هر خط تردد را به کد پرسنلی نگاشت می‌کند.
//
// جست‌وجو اول با خودِ کد انجام می‌شود و اگر نیافت، با کد بدون صفرهای آغازین؛ چون
// دستگاه‌ها اغلب کد پرسنلی را با صفر پیشوند («001001») چاپ می‌کنند.
func MapDeviceCodes(f DeviceFile, mapping map[string]string) DeviceFile {
	for i := range f.Rows {
		f.Rows[i].EmployeeID = lookupDevice(mapping, f.Rows[i].DeviceCode)
	}
	f.Punches = mergeCrossMidnight(buildPunches(f.Rows))
	f.recount()
	return f
}

// lookupDevice نگاشت یک کد دستگاه.
func lookupDevice(mapping map[string]string, code string) string {
	if len(mapping) == 0 {
		return ""
	}
	if id, ok := mapping[code]; ok {
		return id
	}
	if trimmed := strings.TrimLeft(code, "0"); trimmed != code && trimmed != "" {
		if id, ok := mapping[trimmed]; ok {
			return id
		}
	}
	return ""
}

// DeviceCodeList فهرست یکتای کدهای دستگاه در فایل (مرتب‌شده) — برای گزارش بی‌نگاشت‌ها.
func DeviceCodeList(f DeviceFile) []string {
	set := map[string]bool{}
	for _, row := range f.Rows {
		set[row.DeviceCode] = true
	}
	out := make([]string, 0, len(set))
	for code := range set {
		out = append(out, code)
	}
	sort.Strings(out)
	return out
}

// UnmappedDeviceCodes کدهای دستگاهی که به هیچ کارمندی نگاشت نشده‌اند.
func UnmappedDeviceCodes(f DeviceFile) []string {
	set := map[string]bool{}
	for _, p := range f.Punches {
		if p.EmployeeID == "" {
			set[p.DeviceCode] = true
		}
	}
	out := make([]string, 0, len(set))
	for code := range set {
		out = append(out, code)
	}
	sort.Strings(out)
	return out
}

// buildPunches خط‌های خام را به زوج ورود/خروج هر کارمند در هر روز تبدیل می‌کند.
//
// قاعده‌ها: خط تکراری کنار می‌رود؛ اولین «ورود» و آخرین «خروج» روز ملاک است؛ روزی که
// فقط یک تردد دارد «ناقص» علامت می‌خورد و روزی که خروجش پیش از ورود ثبت شده، شیفت
// شبانه (خروج در روز بعد) در نظر گرفته می‌شود.
func buildPunches(rows []DeviceRow) []DevicePunch {
	type key struct{ code, date string }
	order := []key{}
	grouped := map[key][]DeviceRow{}
	for _, row := range rows {
		if row.Status == "duplicate" {
			continue
		}
		k := key{row.DeviceCode, row.Date}
		if _, ok := grouped[k]; !ok {
			order = append(order, k)
		}
		grouped[k] = append(grouped[k], row)
	}

	out := make([]DevicePunch, 0, len(order))
	for _, k := range order {
		items := grouped[k]
		sort.SliceStable(items, func(i, j int) bool { return items[i].Time < items[j].Time })
		p := DevicePunch{DeviceCode: k.code, Date: k.date, Status: "ok", PunchCount: len(items)}
		for _, it := range items {
			p.EmployeeID = firstNonEmpty(p.EmployeeID, it.EmployeeID)
		}
		var ins, outs []string
		for _, it := range items {
			if it.Direction == "ورود" {
				ins = append(ins, it.Time)
			} else {
				outs = append(outs, it.Time)
			}
		}
		switch {
		case len(ins) > 0:
			p.CheckIn = ins[0]
		case len(outs) > 0:
			p.CheckOut = outs[len(outs)-1]
		}
		if len(ins) > 0 && len(outs) > 0 {
			p.CheckOut = outs[len(outs)-1]
		}
		if p.CheckIn != "" && p.CheckOut != "" {
			inMin, _ := clockMinutes(p.CheckIn)
			outMin, _ := clockMinutes(p.CheckOut)
			if outMin <= inMin {
				outMin += 24 * 60
				p.IssuesFa = append(p.IssuesFa,
					"خروج پیش از ورود ثبت شده است؛ خروج در روز بعد (شیفت شبانه) در نظر گرفته شد.")
			}
			p.WorkedHours = round2f(float64(outMin-inMin) / 60)
			if p.WorkedHours > 24 {
				p.Status = "invalid"
				p.IssuesFa = append(p.IssuesFa, "بازهٔ کارکرد بیش از ۲۴ ساعت است؛ تردد نامعتبر.")
			}
		} else {
			p.Status = "incomplete"
			switch {
			case len(ins) > 1 && len(outs) == 0:
				p.IssuesFa = append(p.IssuesFa, "چند ورود بدون خروج ثبت شده است.")
			case len(outs) > 1 && len(ins) == 0:
				p.IssuesFa = append(p.IssuesFa, "چند خروج بدون ورود ثبت شده است.")
			default:
				p.IssuesFa = append(p.IssuesFa, "تردد ناقص: ورود و خروج کامل ثبت نشده است.")
			}
		}
		if p.EmployeeID == "" {
			p.Status = "unknownDevice"
			p.IssuesFa = append(p.IssuesFa, "کد دستگاه به هیچ کارمندی نگاشت نشده است.")
		}
		out = append(out, p)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Date != out[j].Date {
			return out[i].Date < out[j].Date
		}
		return out[i].DeviceCode < out[j].DeviceCode
	})
	return out
}

// mergeCrossMidnight تردد شیفت شبانه‌ای که خروجش در بامداد روز بعد در فایل افتاده را
// به یک زوج کامل تبدیل می‌کند.
//
// قاعده: اگر روزی فقط «ورود» داشته باشد و روز بعدش فقط «خروج» و آن خروج پیش از ساعت
// ۱۲ روز بعد باشد، خروج به همان شیفت شبانه نسبت داده می‌شود؛ وگرنه دو تردد ناقص
// می‌مانند. بدون این قاعده، نوبتکارِ شب هر شب دو مغایرت «ناقص» تولید می‌کند.
func mergeCrossMidnight(punches []DevicePunch) []DevicePunch {
	dropped := map[int]bool{}
	for i := range punches {
		if dropped[i] || punches[i].Status != "incomplete" || punches[i].CheckIn == "" || punches[i].CheckOut != "" {
			continue
		}
		for j := i + 1; j < len(punches); j++ {
			if dropped[j] || punches[j].DeviceCode != punches[i].DeviceCode ||
				punches[j].EmployeeID != punches[i].EmployeeID {
				continue
			}
			if punches[j].Status != "incomplete" || punches[j].CheckOut == "" || punches[j].CheckIn != "" {
				continue
			}
			if !isNextDay(punches[i].Date, punches[j].Date) {
				continue
			}
			if minutes, ok := clockMinutes(punches[j].CheckOut); !ok || minutes >= 12*60 {
				continue
			}
			punches[i].CheckOut = punches[j].CheckOut
			punches[i].PunchCount += punches[j].PunchCount
			punches[i].Status = "ok"
			punches[i].IssuesFa = append(punches[i].IssuesFa,
				"خروج در بامداد روز بعد ثبت شده است؛ به شیفت شبانهٔ همان روز نسبت داده شد.")
			if inMin, ok := clockMinutes(punches[i].CheckIn); ok {
				if outMin, ok := clockMinutes(punches[i].CheckOut); ok {
					punches[i].WorkedHours = round2f(float64(outMin+24*60-inMin) / 60)
				}
			}
			dropped[j] = true
			break
		}
	}
	out := make([]DevicePunch, 0, len(punches))
	for i, p := range punches {
		if !dropped[i] {
			out = append(out, p)
		}
	}
	return out
}

// isNextDay آیا b دقیقاً فردای a است.
func isNextDay(a, b string) bool {
	first, errA := calendar.Parse(a)
	second, errB := calendar.Parse(b)
	if errA != nil || errB != nil {
		return false
	}
	gap, err := calendar.DaysBetween(first, second)
	return err == nil && gap == 1
}

// recount جمع فایل را از خط‌ها و زوج‌ها بازمی‌سازد.
func (f *DeviceFile) recount() {
	s := DeviceFileSummary{Lines: f.Summary.Lines, Rejected: len(f.Rejected)}
	emps := map[string]bool{}
	for _, row := range f.Rows {
		switch row.Status {
		case "duplicate":
			s.Duplicate++
		default:
			s.Accepted++
		}
	}
	for _, p := range f.Punches {
		s.Punches++
		switch p.Status {
		case "ok":
			s.OkPunches++
		case "incomplete":
			s.Incomplete++
		case "invalid":
			s.Invalid++
		case "unknownDevice":
			s.UnknownDevice++
		}
		if p.EmployeeID != "" {
			emps[p.EmployeeID] = true
		}
	}
	s.Employees = len(emps)
	f.Summary = s
}

// PunchMetrics نتیجهٔ سنجش یک تردد روزانه با نوبت همان روز.
type PunchMetrics struct {
	WorkedHours       float64 `json:"workedHours"`
	LateMinutes       int     `json:"lateMinutes"`
	EarlyLeaveMinutes int     `json:"earlyLeaveMinutes"`
	NightHours        float64 `json:"nightHours"`
	FridayHours       float64 `json:"fridayHours"`
	NoteFa            string  `json:"noteFa,omitempty"`
}

// PunchMetricsFor قاعدهٔ تأخیر/تعجیل/شب‌کاری/جمعه‌کاری یک تردد را می‌سنجد.
//
// قاعده‌ها (همه از دامنه، نه از رابط):
//   - تأخیر = ورود پس از «شروع شیفت + مهلت تأخیر مجاز»؛ مهلت از پارامتر شرکت می‌آید.
//   - تعجیل = خروج پیش از «پایان شیفت − مهلت تأخیر مجاز».
//   - شب‌کاری = هم‌پوشانی بازهٔ کارکرد با بازهٔ ۲۲:۰۰ تا ۰۶:۰۰.
//   - جمعه‌کاری = تمام ساعت کارکرد روز تعطیل/پایان هفته.
func PunchMetricsFor(p DevicePunch, shift *PlanDay, graceMinutes int, holiday bool) PunchMetrics {
	out := PunchMetrics{WorkedHours: p.WorkedHours}
	inMin, okIn := clockMinutes(p.CheckIn)
	outMin, okOut := clockMinutes(p.CheckOut)
	if !okIn || !okOut || p.WorkedHours <= 0 {
		return out
	}
	if outMin <= inMin {
		outMin += 24 * 60
	}
	if graceMinutes < 0 {
		graceMinutes = 0
	}

	// نگاشت ورود/خروج روی خط زمانیِ شروع شیفت (شیفت شبانه از ۲۲:۰۰ آغاز می‌شود و
	// خروجش در روز بعد است).
	if shift != nil {
		if sMin, ok := clockMinutes(shift.Start); ok {
			eMin, okE := clockMinutes(shift.End)
			if !okE {
				eMin = sMin
			}
			if eMin <= sMin {
				eMin += 24 * 60
			}
			// ورود را به نزدیک‌ترین روز نسبت به شروع شیفت می‌بریم.
			relIn := inMin
			for relIn < sMin-720 {
				relIn += 24 * 60
			}
			for relIn > sMin+720 {
				relIn -= 24 * 60
			}
			relOut := outMin
			for relOut < relIn {
				relOut += 24 * 60
			}
			if relIn > sMin+graceMinutes {
				out.LateMinutes = relIn - sMin - graceMinutes
			}
			if relOut < eMin-graceMinutes {
				out.EarlyLeaveMinutes = eMin - graceMinutes - relOut
			}
			if out.LateMinutes > 0 || out.EarlyLeaveMinutes > 0 {
				out.NoteFa = "تأخیر " + itoaFa(out.LateMinutes) + " دقیقه و تعجیل " +
					itoaFa(out.EarlyLeaveMinutes) + " دقیقه پس از کسر مهلت مجاز " +
					itoaFa(graceMinutes) + " دقیقه."
			}
		}
	}

	// شب‌کاری: هم‌پوشانی با بازهٔ ۲۲:۰۰–۰۶:۰۰ (سه پنجرهٔ ممکن روی خط زمانی روز).
	nightWindowStart := 22 * 60
	nightWindowEnd := 6*60 + 24*60
	covered := 0
	for _, shiftBy := range []int{-24 * 60, 0, 24 * 60} {
		covered += overlapMinutes(inMin, outMin, nightWindowStart+shiftBy, nightWindowEnd+shiftBy)
	}
	out.NightHours = round2f(float64(covered) / 60)

	if holiday {
		out.FridayHours = p.WorkedHours
	}
	return out
}

// overlapMinutes هم‌پوشانی دو بازهٔ دقیقه‌ای را برمی‌گرداند (بازهٔ خالی = صفر).
func overlapMinutes(aStart, aEnd, bStart, bEnd int) int {
	start := aStart
	if bStart > start {
		start = bStart
	}
	end := aEnd
	if bEnd < end {
		end = bEnd
	}
	if end <= start {
		return 0
	}
	return end - start
}

// clockMinutes «HH:MM» یا «HH:MM:SS» را به دقیقهٔ روز تبدیل می‌کند.
func clockMinutes(raw string) (int, bool) {
	raw = strings.TrimSpace(calendar.NormalizeDigits(raw))
	if raw == "" {
		return 0, false
	}
	parts := strings.Split(raw, ":")
	if len(parts) < 2 || len(parts) > 3 {
		return 0, false
	}
	hour, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil || hour < 0 || hour > 24 {
		return 0, false
	}
	minute, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil || minute < 0 || minute > 59 {
		return 0, false
	}
	return hour*60 + minute, true
}

// normalizeClock ساعت پذیرفته‌شده را به قالب HH:MM برمی‌گرداند.
func normalizeClock(raw string) string {
	total, ok := clockMinutes(raw)
	if !ok {
		return raw
	}
	hour := total / 60
	minute := total % 60
	return pad2(hour) + ":" + pad2(minute)
}

// deviceDirection جهت تردد را به واژگان دامنه نگاشت می‌کند.
func deviceDirection(raw string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(calendar.NormalizeDigits(raw))) {
	case "ورود", "in", "i", "1", "enter", "checkin", "check-in":
		return "ورود", true
	case "خروج", "out", "o", "0", "exit", "checkout", "check-out":
		return "خروج", true
	}
	return "", false
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func pad2(n int) string {
	if n < 10 {
		return "0" + strconv.Itoa(n)
	}
	return strconv.Itoa(n)
}

func round2f(x float64) float64 {
	if x < 0 {
		return float64(int64(x*100-0.5)) / 100
	}
	return float64(int64(x*100+0.5)) / 100
}
