// آزمون جریان سرتاسری «کارکرد واقعی»: نوبت‌کاری مصوب → ورود فایل دستگاه تردد →
// کارکرد دوره → تطبیق کارکرد با حقوق (زیرکارهای ۱.۱ تا ۱.۳ فاز ۱ نقشهٔ راه).
//
// این آزمون همان زنجیره‌ای را می‌سنجد که در واحد حقوق استفاده می‌شود: نوبت مصوب روز را
// تعیین می‌کند، فایل دستگاه تأخیر/تعجیل و شب‌کاری را از همان نوبت می‌سازد، کارکرد دوره
// آن را به پورت قراردادی می‌دهد و گزارش تطبیق مبلغ موتور قواعد را کنار کارکرد می‌گذارد.

package bootstrap

import (
	"net/http"
	"strings"
	"testing"

	"hrms/internal/contracts"
)

func TestDeviceImportAndReconciliationFlow(t *testing.T) {
	h := newTestServer(t)
	hr := login(t, h, "hr.manager", "demo").AccessToken
	emp := login(t, h, "employee", "demo").AccessToken
	today := jdate(t, 0)
	period := today[:7] // YYYY/MM

	// ---------- قالب فایل تردد در دسترس کارگزینی است ----------
	rec := call(t, h, http.MethodGet, "/api/v1/attendance/device-import/template", hr, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("قالب فایل تردد باید ۲۰۰ بدهد، %d داد", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "کد دستگاه") {
		t.Errorf("قالب فایل باید ستون‌های فایل تردد را توضیح دهد: %s", rec.Body.String()[:120])
	}

	// ---------- الگوی چرخهٔ نوبت و تخصیص مصوب امروز (مبنای سنجش تأخیر) ----------
	rec = call(t, h, http.MethodPost, "/api/v1/attendance/records", hr, map[string]any{
		"kind": "shiftPatterns", "title": "الگوی آزمون کارکرد",
		"fields": map[string]string{"rotation": "صبح,عصر,شب,استراحت", "minCoverage": "2"},
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("ایجاد الگوی نوبت باید ۲۰۱ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	var pattern contracts.Record
	decodeInto(t, rec, &pattern)
	act := func(token, id, action string) int {
		return call(t, h, http.MethodPost, "/api/v1/attendance/records/"+id+"/action", token,
			map[string]any{"action": action}).Code
	}
	if code := act(hr, pattern.ID, "activate"); code != http.StatusOK {
		t.Fatalf("فعال‌سازی الگو باید ۲۰۰ بدهد، %d داد", code)
	}

	rec = call(t, h, http.MethodPost, "/api/v1/attendance/records", hr, map[string]any{
		"kind": "shiftAssignments", "title": "تخصیص نوبت آزمون کارکرد", "employeeId": "EMP-005",
		"fields": map[string]string{"pattern": "الگوی آزمون کارکرد", "startDate": today, "cycleOffset": "0"},
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("ایجاد تخصیص نوبت باید ۲۰۱ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	var assignment contracts.Record
	decodeInto(t, rec, &assignment)
	if code := act(hr, assignment.ID, "submit"); code != http.StatusOK {
		t.Fatalf("ارسال تخصیص باید ۲۰۰ بدهد، %d داد", code)
	}
	if code := act(hr, assignment.ID, "approve"); code != http.StatusOK {
		t.Fatalf("تأیید تخصیص باید ۲۰۰ بدهد، %d داد", code)
	}

	// ---------- دسترسی: کارمند نباید فایل دستگاه را وارد کند ----------
	body := map[string]any{"fileName": "denied.txt", "content": "EMP-005," + today + ",07:52,ورود\n"}
	if got := call(t, h, http.MethodPost, "/api/v1/attendance/device-import", emp, body); got.Code != http.StatusForbidden {
		t.Fatalf("ورود فایل توسط کارمند باید ۴۰۳ بدهد، %d داد", got.Code)
	}

	// ---------- ورود فایل: یک روز برای دو کارمند + یک خط ناقص ----------
	content := strings.Join([]string{
		"# فایل آزمون دستگاه تردد",
		"EMP-005," + today + ",06:20,ورود",
		"EMP-005," + today + ",13:30,خروج",
		"EMP-008," + today + ",13:48,ورود",
		"EMP-008," + today + ",22:05,خروج",
		"EMP-009," + today + ",07:00",
	}, "\n")

	rec = call(t, h, http.MethodPost, "/api/v1/attendance/device-import", hr, map[string]any{
		"fileName": "device.txt", "content": content,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("ورود فایل تردد باید ۲۰۰ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	var report struct {
		Summary struct {
			Lines, Accepted, Duplicate, Rejected, Punches, OkPunches, Incomplete, Employees int
		} `json:"summary"`
		Created int    `json:"created"`
		Updated int    `json:"updated"`
		Skipped int    `json:"skipped"`
		BatchID string `json:"batchId"`
		Punches []struct {
			EmployeeID, Date, Status, CheckIn, CheckOut string
		} `json:"punches"`
		Rejected []struct {
			Line      int
			MessageFa string
		} `json:"rejected"`
		GraceMinutes int `json:"graceMinutes"`
	}
	decodeInto(t, rec, &report)
	if report.Summary.Lines != 6 || report.Summary.Accepted != 4 || report.Summary.Rejected != 1 {
		t.Errorf("جمع فایل درست نیست: %+v", report.Summary)
	}
	if report.Created != 2 || report.Updated != 0 {
		t.Errorf("باید ۲ رکورد تازه ساخته شود، %d ساخته و %d ویرایش شد", report.Created, report.Updated)
	}
	if report.GraceMinutes != 15 {
		t.Errorf("مهلت تأخیر مجاز باید از پارامتر شرکت (۱۵ دقیقه) بیاید، %d بود", report.GraceMinutes)
	}
	if len(report.Rejected) != 1 || report.Rejected[0].MessageFa == "" {
		t.Errorf("خط ناقص باید با پیام فارسی گزارش شود: %+v", report.Rejected)
	}
	if report.BatchID == "" || !strings.HasPrefix(report.BatchID, "DVI-") {
		t.Errorf("سند گردش ورود فایل باید با پیشوند DVI- ساخته شود، «%s» شد", report.BatchID)
	}

	// ---------- قاعدهٔ دامنه روی رکورد نوشته‌شده: تأخیر و تعجیل از نوبت مصوب ----------
	page := call(t, h, http.MethodGet, "/api/v1/attendance/records?kind=attendanceRecords&size=100", hr, nil)
	if page.Code != http.StatusOK {
		t.Fatalf("فهرست ترددها باید ۲۰۰ بدهد، %d داد", page.Code)
	}
	var records contracts.RecordPage
	decodeInto(t, page, &records)
	var imported *contracts.Record
	for i := range records.Items {
		item := records.Items[i]
		if item.EmployeeID == "EMP-005" && item.Fields["date"] == today {
			imported = &records.Items[i]
		}
	}
	if imported == nil {
		t.Fatalf("رکورد تردد کارمند EMP-005 برای %s ساخته نشد", today)
	}
	if got := imported.Fields["shift"]; got != "صبح" {
		t.Errorf("نوبت روز باید از چرخهٔ مصوب «صبح» باشد، «%s» شد", got)
	}
	if got := imported.Fields["lateMinutes"]; got != "5" {
		t.Errorf("تأخیر باید ۵ دقیقه باشد (۰۶:۲۰ با مهلت ۱۵ دقیقه)، «%s» شد", got)
	}
	if got := imported.Fields["earlyMinutes"]; got != "15" {
		t.Errorf("تعجیل باید ۱۵ دقیقه باشد (۱۳:۳۰ در برابر پایان ۱۴:۰۰)، «%s» شد", got)
	}
	if got := imported.Fields["workedHours"]; got != "7.17" {
		t.Errorf("ساعت کارکرد باید ۷٫۱۷ باشد (۰۶:۲۰ تا ۱۳:۳۰)، «%s» شد", got)
	}
	if imported.NoteFa == "" {
		t.Errorf("رکورد تردد باید یادداشت مغایرت فارسی داشته باشد")
	}

	// ---------- ورود دوبارهٔ همان فایل: ویرایش، نه رکورد تکراری ----------
	rec = call(t, h, http.MethodPost, "/api/v1/attendance/device-import", hr, map[string]any{
		"fileName": "device.txt", "content": content,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("ورود دوبارهٔ فایل باید ۲۰۰ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	decodeInto(t, rec, &report)
	if report.Created != 0 || report.Updated != 2 {
		t.Errorf("ورود دوباره نباید رکورد تازه بسازد؛ %d ساخته و %d ویرایش شد", report.Created, report.Updated)
	}

	// ---------- برنامهٔ نوبت و پوشش شیفت ----------
	plan := call(t, h, http.MethodGet,
		"/api/v1/attendance/shift-plan?from="+today+"&to="+today+"&employeeId=EMP-005", hr, nil)
	if plan.Code != http.StatusOK {
		t.Fatalf("برنامهٔ نوبت باید ۲۰۰ بدهد، %d داد: %s", plan.Code, plan.Body.String())
	}
	var planBody struct {
		Days     []map[string]any `json:"days"`
		Coverage []struct {
			Date, Shift string
			Assigned    int
			MinCoverage int
			Shortfall   int
		} `json:"coverage"`
		MinCoverage int `json:"minCoverage"`
	}
	decodeInto(t, plan, &planBody)
	if len(planBody.Days) != 1 {
		t.Fatalf("برای بازهٔ یک‌روزه باید یک روز برنامه بیاید، %d آمد", len(planBody.Days))
	}
	if len(planBody.Coverage) != 1 || planBody.Coverage[0].Shift != "صبح" {
		t.Fatalf("پوشش شیفت صبح باید یک ردیف باشد: %+v", planBody.Coverage)
	}
	if planBody.MinCoverage != 2 || planBody.Coverage[0].Shortfall != 1 {
		t.Errorf("حداقل پوشش الگو ۲ و کمبود یک نفر است: %+v", planBody.Coverage[0])
	}

	// ---------- تطبیق کارکرد با حقوق ----------
	rec = call(t, h, http.MethodGet, "/api/v1/payroll/worktime-reconciliation?period="+period, hr, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("گزارش تطبیق باید ۲۰۰ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	var recon struct {
		Period  string `json:"period"`
		Total   int    `json:"total"`
		Factors []struct {
			Key, SourceFa string
			Value         float64
		} `json:"factors"`
		Rows []struct {
			EmployeeID        string   `json:"employeeId"`
			WorkedHours       float64  `json:"workedHours"`
			LateMinutes       int      `json:"lateMinutes"`
			EarlyLeaveMinutes int      `json:"earlyLeaveMinutes"`
			ExpectedDays      float64  `json:"expectedDays"`
			MissingDays       float64  `json:"missingDays"`
			Gross             int64    `json:"gross"`
			Net               int64    `json:"net"`
			Mismatch          bool     `json:"mismatch"`
			IssuesFa          []string `json:"issuesFa"`
		} `json:"rows"`
		Run struct {
			Found     bool  `json:"found"`
			Matched   bool  `json:"matched"`
			StoredNet int64 `json:"storedNet"`
		} `json:"run"`
		MismatchCount int   `json:"mismatchCount"`
		AdvisoryTotal int64 `json:"advisoryPremiumTotal"`
	}
	decodeInto(t, rec, &recon)
	if recon.Period != period {
		t.Errorf("دورهٔ گزارش باید %s باشد، «%s» شد", period, recon.Period)
	}
	if len(recon.Factors) != 2 {
		t.Errorf("ضرایب نوبت‌کاری و جمعه‌کاری باید در گزارش بیایند: %+v", recon.Factors)
	}
	var row *struct {
		EmployeeID        string   `json:"employeeId"`
		WorkedHours       float64  `json:"workedHours"`
		LateMinutes       int      `json:"lateMinutes"`
		EarlyLeaveMinutes int      `json:"earlyLeaveMinutes"`
		ExpectedDays      float64  `json:"expectedDays"`
		MissingDays       float64  `json:"missingDays"`
		Gross             int64    `json:"gross"`
		Net               int64    `json:"net"`
		Mismatch          bool     `json:"mismatch"`
		IssuesFa          []string `json:"issuesFa"`
	}
	for i := range recon.Rows {
		if recon.Rows[i].EmployeeID == "EMP-005" {
			row = &recon.Rows[i]
		}
	}
	if row == nil {
		t.Fatalf("ردیف کارمند EMP-005 در گزارش تطبیق نبود (%d ردیف)", recon.Total)
	}
	if row.LateMinutes != 5 || row.EarlyLeaveMinutes != 15 {
		t.Errorf("تأخیر/تعجیل باید در کارکرد دوره دیده شود: %+v", row)
	}
	if row.ExpectedDays != 1 {
		t.Errorf("روزهای موظفی از نوبت مصوب باید ۱ باشد، %v شد", row.ExpectedDays)
	}
	if row.MissingDays != 0 {
		t.Errorf("روز موظفی این کارمند تردد دارد، پس غیبت نباید ثبت شود: %v", row.MissingDays)
	}
	if row.Gross <= 0 || row.Net <= 0 {
		t.Errorf("مبلغ محاسبه‌شده باید مثبت باشد: %+v", row)
	}
	if row.Mismatch && len(row.IssuesFa) == 0 {
		t.Errorf("ردیف مغایرت‌دار باید دلیل فارسی داشته باشد")
	}
}
