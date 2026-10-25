// پیاده‌سازی ماژول «attendance» — «deviceimport»: ورود فایل دستگاه تردد، برنامهٔ نوبت
// و پوشش شیفت (زیرکارهای ۱.۲ تا ۱.۳ فاز ۱ نقشهٔ راه).
//
// قاعدهٔ معماری: تحلیل فایل، قاعدهٔ تأخیر/تعجیل/شب‌کاری و حل نوبت همه در بستهٔ
// `domain` هستند؛ این لایه فقط متن فایل را می‌خواند، نگاشت کد دستگاه را از پروندهٔ
// کارکنان می‌سازد، نوبت مصوب را از `roster` می‌گیرد و رکورد تردد را می‌نویسد.

package attendance

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"hrms/internal/contracts"
	"hrms/internal/modules/attendance/domain"
	"hrms/internal/platform/auth"
	"hrms/internal/platform/calendar"
	"hrms/internal/platform/httpx"
	"hrms/internal/platform/modulekit"
	"hrms/internal/platform/recordkit"
)

// maxDeviceFileBytes سقف حجم فایل تردد (۸ مگابایت)؛ فایل بزرگ‌تر باید تکه‌تکه فرستاده شود.
const maxDeviceFileBytes = 8 << 20

// registerDeviceEndpoints مسیرهای «فایل دستگاه تردد» و «برنامهٔ نوبت» را ثبت می‌کند.
func registerDeviceEndpoints(mux *http.ServeMux, deps modulekit.Deps, svc *recordkit.Service) {
	read := func(fn http.HandlerFunc) http.HandlerFunc { return deps.Require("attendance", "read", fn) }
	write := func(fn http.HandlerFunc) http.HandlerFunc { return deps.Require("attendance", "write", fn) }

	// قالب نمونهٔ فایل تردد — همان چیزی که کارگزینی به پیمانکار دستگاه می‌دهد.
	mux.HandleFunc("GET /api/v1/attendance/device-import/template", read(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, domain.DeviceFileTemplate)
	}))

	// ورود فایل تردد: تحلیل، نگاشت کد دستگاه، سنجش با نوبت مصوب و ثبت رکورد تردد.
	// این کار کارگزینی است، نه خودخدمتی کارمند؛ پس علاوه بر مجوز «نوشتن» ماژول، نقش
	// هم سنجیده می‌شود (کارمند می‌تواند درخواست مرخصی بدهد، ولی فایل دستگاه را نه).
	mux.HandleFunc("POST /api/v1/attendance/device-import", write(requireHRRole(func(w http.ResponseWriter, r *http.Request) {
		req, err := readDeviceImport(r)
		if err != nil {
			httpx.WriteErrorMsg(w, http.StatusUnprocessableEntity, contracts.CodeConflict, err.Error(), nil)
			return
		}
		mapping, err := domain.ParseDeviceMap(req.DeviceMap)
		if err != nil {
			httpx.WriteErrorMsg(w, http.StatusUnprocessableEntity, contracts.CodeConflict, err.Error(), nil)
			return
		}
		// کد پرسنلی و کد ملی هر کارمند به‌عنوان نگاشت پیش‌فرض؛ دستگاه‌ها اغلب یکی از
		// این دو را چاپ می‌کنند و کارگزینی لازم نیست نگاشت را دستی بنویسد.
		employees, names := employeeDirectory(r.Context(), deps)
		for _, id := range employees {
			mapping[id] = id
		}
		for code, id := range nationalIDMap(r.Context(), deps) {
			if _, ok := mapping[code]; !ok {
				mapping[code] = id
			}
		}

		file := domain.MapDeviceCodes(domain.ParseDeviceFile(req.Content), mapping)
		report := applyDeviceImport(r, deps, svc, file, req.FileName, strings.EqualFold(req.Mode, "validate"))

		httpx.WriteJSON(w, http.StatusOK, map[string]any{
			"fileName":      req.FileName,
			"importedAtFa":  nowFa(deps),
			"mode":          modeFa(req.Mode),
			"summary":       report.summary,
			"created":       report.created,
			"updated":       report.updated,
			"skipped":       report.skipped,
			"skippedRows":   report.skippedRows,
			"unmappedCodes": domain.UnmappedDeviceCodes(file),
			"unknownCodes":  domain.DeviceCodeList(file),
			"punches":       file.Punches,
			"rejected":      file.Rejected,
			"batchId":       report.batchID,
			"graceMinutes":  graceMinutes(deps),
			"noteFa":        report.noteFa,
			"employeesFa":   names,
		})
	})))

	// برنامهٔ نوبت و پوشش شیفت: نوبت هر روز هر کارمند و کمبود پوشش نسبت به الگو.
	mux.HandleFunc("GET /api/v1/attendance/shift-plan", read(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		from, to, errMsg := planRange(q.Get("from"), q.Get("to"), deps)
		if errMsg != "" {
			httpx.WriteErrorMsg(w, http.StatusUnprocessableEntity, contracts.CodeConflict, errMsg, nil)
			return
		}
		rost := loadRoster(r.Context(), svc)
		wanted := strings.TrimSpace(q.Get("employeeId"))
		ids := rost.Employees()
		if wanted != "" {
			ids = []string{wanted}
		}
		days := []domain.PlanDay{}
		for _, id := range ids {
			for _, in := range rost.byEmployee[id] {
				planned, err := domain.BuildPlan(in, from, to)
				if err != nil {
					continue
				}
				days = append(days, planned...)
			}
		}
		sort.SliceStable(days, func(i, j int) bool {
			if days[i].Date != days[j].Date {
				return days[i].Date < days[j].Date
			}
			return days[i].EmployeeID < days[j].EmployeeID
		})
		coverage := domain.Coverage(days, rost.Coverage())
		httpx.WriteJSON(w, http.StatusOK, map[string]any{
			"from":        from,
			"to":          to,
			"days":        days,
			"total":       len(days),
			"coverage":    coverage,
			"shortfalls":  domain.CoverageShortfalls(coverage),
			"shifts":      domain.KnownShifts(),
			"minCoverage": rost.Coverage(),
			"noteFa": "نوبت هر روز از چرخهٔ الگوی تخصیص مصوب حل می‌شود؛ این محاسبه در " +
				"دامنه انجام می‌گیرد و رابط کاربری آن را تکرار نمی‌کند.",
		})
	}))
}

// hrRoles نقش‌هایی که کار کارگزینی (ورود فایل تردد) را انجام می‌دهند.
var hrRoles = map[contracts.Role]bool{"admin": true, "hr_manager": true, "hr_spec": true}

// requireHRRole ورود فایل دستگاه تردد را به کارگزینی و مدیر منابع انسانی محدود می‌کند.
//
// قاعده: خودکارسازی ثبت کارکرد کار کارگزینی است؛ نقش کارمند در همین ماژول مجوز
// «نوشتن» دارد (برای درخواست مرخصی)، پس محدودیت نقش باید صریح باشد.
func requireHRRole(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := auth.UserFrom(r.Context())
		if !ok || !hrRoles[user.Role] {
			httpx.WriteErrorMsg(w, http.StatusForbidden, contracts.CodeForbidden,
				"ورود فایل دستگاه تردد فقط برای کارگزینی و مدیر منابع انسانی مجاز است.", nil)
			return
		}
		next(w, r)
	}
}

// deviceImportRequest ورودی مسیر ورود فایل تردد.
type deviceImportRequest struct {
	FileName  string `json:"fileName"`
	Content   string `json:"content"`
	DeviceMap string `json:"deviceMap"`
	// Mode خالی یا apply یعنی ثبت؛ validate یعنی فقط اعتبارسنجی بدون نوشتن رکورد.
	Mode string `json:"mode"`
}

// readDeviceImport بدنهٔ درخواست را می‌خواند؛ هم قالب JSON و هم بارگذاری فایل
// (multipart/form-data با فیلد «file») پذیرفته می‌شود تا هم رابط وب و هم ابزار
// خط فرمان بتوانند فایل را بفرستند.
func readDeviceImport(r *http.Request) (deviceImportRequest, error) {
	var req deviceImportRequest
	ctype := strings.ToLower(r.Header.Get("Content-Type"))
	if strings.HasPrefix(ctype, "multipart/form-data") {
		if err := r.ParseMultipartForm(maxDeviceFileBytes); err != nil {
			return req, fmt.Errorf("خواندن فایل بارگذاری‌شده ممکن نشد: %v", err)
		}
		req.FileName = strings.TrimSpace(r.FormValue("fileName"))
		req.DeviceMap = r.FormValue("deviceMap")
		req.Mode = strings.TrimSpace(r.FormValue("mode"))
		file, header, err := r.FormFile("file")
		if err != nil {
			return req, fmt.Errorf("فیلد فایل («file») در درخواست نبود یا خوانده نشد.")
		}
		defer file.Close()
		if req.FileName == "" && header != nil {
			req.FileName = header.Filename
		}
		body, err := io.ReadAll(io.LimitReader(file, maxDeviceFileBytes))
		if err != nil {
			return req, fmt.Errorf("خواندن محتوای فایل ممکن نشد: %v", err)
		}
		req.Content = string(body)
		return req, nil
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, maxDeviceFileBytes))
	if err != nil {
		return req, fmt.Errorf("خواندن بدنهٔ درخواست ممکن نشد: %v", err)
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return req, fmt.Errorf("بدنهٔ درخواست JSON معتبر نیست: %v", err)
	}
	if strings.TrimSpace(req.Content) == "" {
		return req, fmt.Errorf("محتوای فایل تردد («content») خالی است.")
	}
	return req, nil
}

// deviceImportReport نتیجهٔ ثبت فایل تردد.
type deviceImportReport struct {
	summary     domain.DeviceFileSummary
	created     int
	updated     int
	skipped     int
	skippedRows []string
	batchID     string
	noteFa      string
}

// applyDeviceImport رکوردهای تردد را از زوج‌های ورود/خروج می‌نویسد.
//
// قاعده‌ها: تردد بی‌نگاشت، تردد نامعتبر و تردد تکرارشده ثبت نمی‌شود؛ تردد ناقص ثبت
// می‌شود ولی با یادداشت مغایرت. تأخیر/تعجیل/شب‌کاری/جمعه‌کاری از نوبت مصوب همان روز
// و پارامتر «مهلت تأخیر مجاز» شرکت محاسبه و روی رکورد نوشته می‌شود.
func applyDeviceImport(r *http.Request, deps modulekit.Deps, svc *recordkit.Service, file domain.DeviceFile, fileName string, validateOnly bool) deviceImportReport {
	rep := deviceImportReport{summary: file.Summary}
	grace := graceMinutes(deps)
	rost := loadRoster(r.Context(), svc)
	names := employeeNames(r.Context(), deps)
	actor := actorName(r)

	if validateOnly {
		rep.noteFa = "حالت اعتبارسنجی: فایل تحلیل شد و هیچ رکوردی نوشته نشد."
		return rep
	}

	existing := attendanceIndex(r.Context(), svc)
	for _, p := range file.Punches {
		if p.EmployeeID == "" {
			rep.skipped++
			rep.skippedRows = append(rep.skippedRows, fmt.Sprintf("تردد %s در %s بی‌نگاشت است.", p.DeviceCode, p.Date))
			continue
		}
		if p.Status == "invalid" {
			rep.skipped++
			rep.skippedRows = append(rep.skippedRows, fmt.Sprintf("تردد %s در %s نامعتبر است.", p.DeviceCode, p.Date))
			continue
		}
		shift, hasShift := rost.ShiftFor(p.EmployeeID, p.Date)
		holiday := false
		if hasShift {
			holiday = shift.Holiday
		} else if day, err := calendar.Parse(p.Date); err == nil {
			cal := calendar.DefaultWorkCalendar()
			weekend, _ := cal.IsWeekend(day)
			holiday = weekend || cal.IsHoliday(day)
		}
		metrics := domain.PunchMetricsFor(p, shift, grace, holiday)

		fields := map[string]string{
			"date":         p.Date,
			"checkIn":      p.CheckIn,
			"checkOut":     p.CheckOut,
			"workedHours":  trimFloat(p.WorkedHours),
			"lateMinutes":  strconv.Itoa(metrics.LateMinutes),
			"earlyMinutes": strconv.Itoa(metrics.EarlyLeaveMinutes),
			"nightHours":   trimFloat(metrics.NightHours),
			"fridayHours":  trimFloat(metrics.FridayHours),
			"deviceCode":   p.DeviceCode,
			"sourceFile":   fileName,
		}
		shiftField := ""
		if hasShift {
			shiftField = shift.Shift
			fields["shift"] = shift.Shift
		}
		note := notesFa(p, metrics, shiftField, grace)

		name := names[p.EmployeeID]
		if name == "" {
			name = p.EmployeeID
		}
		title := fmt.Sprintf("تردد %s — %s", p.Date, name)
		in := contracts.RecordInput{Kind: "attendanceRecords", Title: title, EmployeeID: p.EmployeeID, Fields: fields, NoteFa: note}

		if id, ok := existing[p.EmployeeID+"|"+p.Date]; ok {
			if _, err := svc.Update(r.Context(), id, in, actor); err != nil {
				rep.skipped++
				rep.skippedRows = append(rep.skippedRows,
					fmt.Sprintf("ویرایش تردد %s در %s ممکن نشد: %s", p.EmployeeID, p.Date, err.Error()))
				continue
			}
			rep.updated++
			continue
		}
		rec, err := svc.Create(r.Context(), in, actor)
		if err != nil {
			rep.skipped++
			rep.skippedRows = append(rep.skippedRows, fmt.Sprintf("ثبت تردد %s در %s ممکن نشد: %s", p.EmployeeID, p.Date, err.Error()))
			continue
		}
		existing[p.EmployeeID+"|"+p.Date] = rec.ID
		rep.created++
	}

	// سند گردش ورود فایل: چه فایلی، چه شماره‌ای و با چه مغایرتی — برای پیگیری بعدی.
	batchTitle := fmt.Sprintf("ورود فایل تردد %s (%s)", fileName, nowFa(deps))
	if strings.TrimSpace(fileName) == "" {
		batchTitle = fmt.Sprintf("ورود فایل تردد (%s)", nowFa(deps))
	}
	batch, err := svc.Create(r.Context(), contracts.RecordInput{
		Kind: "deviceImports", Title: batchTitle,
		Fields: map[string]string{
			"fileName":   fileName,
			"importedAt": nowFa(deps),
			"lines":      strconv.Itoa(file.Summary.Lines),
			"accepted":   strconv.Itoa(file.Summary.Accepted),
			"rejected":   strconv.Itoa(file.Summary.Rejected),
			"duplicate":  strconv.Itoa(file.Summary.Duplicate),
			"created":    strconv.Itoa(rep.created),
			"updated":    strconv.Itoa(rep.updated),
			"skipped":    strconv.Itoa(rep.skipped),
		},
		NoteFa: fmt.Sprintf("فایل تردد با %d خط خوانده شد؛ %d رکورد تازه، %d ویرایش و %d ردیف ثبت‌نشده.",
			file.Summary.Lines, rep.created, rep.updated, rep.skipped),
	}, actor)
	if err == nil {
		rep.batchID = batch.ID
	}

	rep.noteFa = fmt.Sprintf("فایل با %d خط خوانده شد (%d ردیف پذیرفته، %d تکراری، %d رد). "+
		"%d رکورد تازه ثبت و %d رکورد به‌روزرسانی شد؛ مهلت تأخیر مجاز %d دقیقه از پارامتر شرکت فعال خوانده شد.",
		file.Summary.Lines, file.Summary.Accepted, file.Summary.Duplicate, file.Summary.Rejected,
		rep.created, rep.updated, grace)
	if rep.skipped > 0 {
		rep.noteFa += fmt.Sprintf(" %d ردیف ثبت نشد که در فهرست «ثبت‌نشده‌ها» آمده است.", rep.skipped)
	}
	return rep
}

// attendanceIndex نقشهٔ «کارمند|تاریخ ← شناسهٔ رکورد تردد» را از رکوردهای موجود می‌سازد
// تا ورود دوبارهٔ فایل، تردد روز را ویرایش کند نه اینکه رکورد تکراری بسازد.
func attendanceIndex(ctx context.Context, svc *recordkit.Service) map[string]string {
	out := map[string]string{}
	if svc == nil {
		return out
	}
	page, err := svc.List(ctx, contracts.RecordFilter{
		Module: "attendance", Kind: "attendanceRecords", Size: 500, Scope: "all"})
	if err != nil {
		return out
	}
	for _, rec := range page.Items {
		if date := strings.TrimSpace(rec.Fields["date"]); date != "" && rec.EmployeeID != "" {
			out[rec.EmployeeID+"|"+date] = rec.ID
		}
	}
	return out
}

// notesFa یادداشت مغایرت یک رکورد تردد را به فارسی می‌سازد.
func notesFa(p domain.DevicePunch, m domain.PunchMetrics, shift string, grace int) string {
	parts := []string{"ثبت‌شده از فایل دستگاه تردد."}
	if shift == "" {
		parts = append(parts, "برای این روز نوبت مصوبی ثبت نشده است؛ تأخیر/تعجیل محاسبه نشد.")
	} else {
		parts = append(parts, "نوبت روز: "+shift+".")
	}
	if p.Status == "incomplete" {
		parts = append(parts, "تردد ناقص؛ رکورد برای پیگیری اصلاح ثبت شد.")
	}
	if m.NoteFa != "" {
		parts = append(parts, m.NoteFa)
	}
	if m.NightHours > 0 {
		parts = append(parts, fmt.Sprintf("%s ساعت شب‌کاری.", trimFloat(m.NightHours)))
	}
	if m.FridayHours > 0 {
		parts = append(parts, fmt.Sprintf("%s ساعت کار در روز تعطیل/پایان هفته.", trimFloat(m.FridayHours)))
	}
	if grace > 0 {
		parts = append(parts, fmt.Sprintf("مهلت تأخیر مجاز %d دقیقه.", grace))
	}
	return strings.Join(parts, " ")
}

// graceMinutes مهلت تأخیر مجاز روزانه از پارامتر شرکت فعال (پیش‌فرض ۱۵ دقیقه).
func graceMinutes(deps modulekit.Deps) int {
	for _, p := range deps.Org.Params {
		if p.Key != "attendance.graceMinutes" {
			continue
		}
		if n, err := strconv.Atoi(strings.TrimSpace(calendar.NormalizeDigits(p.Value))); err == nil && n >= 0 {
			return n
		}
	}
	return 15
}

// employeeDirectory شناسه‌های کارکنان و نقشهٔ «شناسه ← نام» را برمی‌گرداند.
func employeeDirectory(ctx context.Context, deps modulekit.Deps) ([]string, map[string]string) {
	names := employeeNames(ctx, deps)
	ids := make([]string, 0, len(names))
	for id := range names {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids, names
}

// employeeNames نقشهٔ «شناسهٔ کارمند ← نام کامل».
func employeeNames(ctx context.Context, deps modulekit.Deps) map[string]string {
	out := map[string]string{}
	reader, ok := deps.Ports.Employees()
	if !ok {
		return out
	}
	page, err := reader.List(ctx, contracts.EmployeeFilter{Scope: "all", Size: 500})
	if err != nil {
		return out
	}
	for _, emp := range page.Items {
		out[emp.ID] = emp.FullName
	}
	return out
}

// nationalIDMap نگاشت «کد ملی ← شناسهٔ کارمند»؛ دستگاه‌های تردد اغلب کد ملی را
// به‌جای کد پرسنلی چاپ می‌کنند.
func nationalIDMap(ctx context.Context, deps modulekit.Deps) map[string]string {
	out := map[string]string{}
	reader, ok := deps.Ports.Employees()
	if !ok {
		return out
	}
	page, err := reader.List(ctx, contracts.EmployeeFilter{Scope: "all", Size: 500})
	if err != nil {
		return out
	}
	for _, emp := range page.Items {
		if code := strings.TrimSpace(emp.NationalID); code != "" {
			out[code] = emp.ID
		}
	}
	return out
}

// planRange بازهٔ برنامهٔ نوبت را می‌سازد؛ پیش‌فرض، ماه جاری سرور است.
func planRange(fromRaw, toRaw string, deps modulekit.Deps) (string, string, string) {
	from := strings.TrimSpace(calendar.NormalizeDigits(fromRaw))
	to := strings.TrimSpace(calendar.NormalizeDigits(toRaw))
	if from == "" || to == "" {
		today, err := calendar.FromGregorian(deps.Now())
		if err != nil {
			return from, to, "تاریخ جاری سرور قابل تبدیل نیست."
		}
		length, err := calendar.MonthLength(today.Year, today.Month)
		if err != nil {
			length = 30
		}
		from = fmt.Sprintf("%04d/%02d/01", today.Year, today.Month)
		to = fmt.Sprintf("%04d/%02d/%02d", today.Year, today.Month, length)
	}
	if _, err := calendar.Parse(from); err != nil {
		return from, to, "تاریخ شروع بازه معتبر نیست؛ قالب YYYY/MM/DD لازم است."
	}
	if _, err := calendar.Parse(to); err != nil {
		return from, to, "تاریخ پایان بازه معتبر است؛ قالب YYYY/MM/DD لازم است."
	}
	return from, to, ""
}

// nowFa تاریخ جاری سرور به تقویم شمسی.
func nowFa(deps modulekit.Deps) string {
	today, err := calendar.FromGregorian(deps.Now())
	if err != nil {
		return ""
	}
	return today.String()
}

// modeFa نام حالت ورود فایل برای نمایش.
func modeFa(mode string) string {
	if strings.EqualFold(strings.TrimSpace(mode), "validate") {
		return "اعتبارسنجی بدون ثبت"
	}
	return "ثبت"
}

// actorName نام کاربر جاری برای ثبت رخداد.
func actorName(r *http.Request) string {
	if user, ok := auth.UserFrom(r.Context()); ok && user.Username != "" {
		return user.Username
	}
	return "attendance"
}

// trimFloat عدد اعشاری را بدون صفرهای اضافی می‌نویسد.
func trimFloat(x float64) string {
	return strconv.FormatFloat(x, 'f', -1, 64)
}
