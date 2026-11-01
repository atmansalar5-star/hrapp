// آزمون جریان سرتاسری «عکس کارگزینی، متصدیان پست سازمانی و سابقهٔ انتصاب».
//
// چه چیزی سنجیده می‌شود:
//   - بارگذاری عکس کارگزینی از مسیر کارمند، سرو امضاشدهٔ فایل و نمایش نشانی عکس در
//     پروندهٔ کارمند (DTO) و در فهرست کارکنان؛
//   - ساختار سازمانی: متصدی هر پست با عکس و نتیجهٔ احراز شرایط، و هشدارهای صادقانه؛
//   - قاعدهٔ دامنه: انتصاب بدون «احرازشده» و بدون شواهد آغاز نمی‌شود (۴۲۲)؛
//   - سابقهٔ انتصاب‌ها و تاریخچهٔ تغییرات یک پست (انتصاب پیشین پایان‌یافته هم می‌ماند).
package bootstrap

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"hrms/internal/contracts"
)

// uploadPhoto عکس کارگزینی یک کارمند را از مسیر multipart می‌فرستد.
func uploadPhoto(t *testing.T, h http.Handler, token, employeeID string) *httptest.ResponseRecorder {
	t.Helper()
	body, contentType := multipartPNG(t)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/employee/employees/"+employeeID+"/photo", bytes.NewReader(body))
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// rosterResponse شکل پاسخ مسیر ساختار سازمانی برای خواندن در آزمون.
type rosterResponse struct {
	Positions []struct {
		ID              string `json:"id"`
		Code            string `json:"code"`
		Title           string `json:"title"`
		Unit            string `json:"unit"`
		Requirements    string `json:"requirements"`
		HolderCount     int    `json:"holderCount"`
		AssignmentCount int    `json:"assignmentCount"`
		HasPhoto        bool   `json:"hasPhoto"`
		RequirementsMet bool   `json:"requirementsMet"`
		Holder          *struct {
			AssignmentID    string `json:"assignmentId"`
			EmployeeID      string `json:"employeeId"`
			FullName        string `json:"fullName"`
			PhotoURL        string `json:"photoUrl"`
			HasPhoto        bool   `json:"hasPhoto"`
			Qualification   string `json:"qualification"`
			QualificationFa string `json:"qualificationFa"`
			Evidence        string `json:"evidence"`
			StartDate       string `json:"startDate"`
		} `json:"holder"`
	} `json:"positions"`
	Units   []map[string]any `json:"units"`
	Summary struct {
		Positions   int `json:"positions"`
		Filled      int `json:"filled"`
		Vacant      int `json:"vacant"`
		Unverified  int `json:"unverified"`
		WithoutFoto int `json:"withoutPhoto"`
		Assignments int `json:"assignments"`
		Active      int `json:"active"`
		Historical  int `json:"historical"`
	} `json:"summary"`
	QualificationOptions []struct {
		Key     string `json:"key"`
		TitleFa string `json:"titleFa"`
	} `json:"qualificationOptions"`
	Warnings []string `json:"warnings"`
	NoteFa   string   `json:"noteFa"`
}

// TestEmployeePhotoAndRosterFlow بارگذاری عکس، نمایش آن در پرونده و ساختار سازمانی را
// سرتاسری می‌سنجد.
func TestEmployeePhotoAndRosterFlow(t *testing.T) {
	h := newTestServer(t)
	admin := login(t, h, "admin", "demo").AccessToken

	// ---------- کارمند در آغاز عکس ندارد ----------
	before := call(t, h, http.MethodGet, "/api/v1/employee/employees/EMP-004", admin, nil)
	if before.Code != http.StatusOK {
		t.Fatalf("خواندن پروندهٔ کارمند باید ۲۰۰ بدهد، %d داد", before.Code)
	}
	var employee contracts.Employee
	decodeInto(t, before, &employee)
	if employee.PhotoURL != "" {
		t.Fatalf("کارمند تازه نباید عکس داشته باشد: %q", employee.PhotoURL)
	}

	// ---------- بارگذاری عکس بدون توکن رد می‌شود ----------
	if got := call(t, h, http.MethodPost, "/api/v1/employee/employees/EMP-004/photo", "", nil); got.Code != http.StatusUnauthorized {
		t.Errorf("بارگذاری عکس بدون توکن باید ۴۰۱ بدهد، %d داد", got.Code)
	}

	// ---------- بارگذاری عکس کارگزینی ----------
	uploaded := uploadPhoto(t, h, admin, "EMP-004")
	if uploaded.Code != http.StatusOK {
		t.Fatalf("بارگذاری عکس باید ۲۰۰ بدهد، %d داد: %s", uploaded.Code, uploaded.Body.String())
	}
	var photo contracts.EmployeePhoto
	decodeInto(t, uploaded, &photo)
	if !photo.HasPhoto || !strings.HasPrefix(photo.URL, "/api/v1/photos/") {
		t.Fatalf("نشانی عکس درست نیست: %+v", photo)
	}
	if !strings.Contains(photo.URL, "t=") || !strings.Contains(photo.URL, "e=EMP-004") {
		t.Fatalf("نشانی عکس باید امضاشده و گره‌خورده به کد پرسنلی باشد: %s", photo.URL)
	}
	if photo.UpdatedAt == "" || photo.SizeKb < 1 || photo.ContentType != "image/png" {
		t.Errorf("اطلاعات عکس ناقص است: %+v", photo)
	}
	if photo.NoteFa == "" {
		t.Errorf("پاسخ بارگذاری عکس باید یادداشت فارسی داشته باشد")
	}

	// ---------- نشانی عکس روی پرونده و فهرست کارکنان می‌نشیند ----------
	after := call(t, h, http.MethodGet, "/api/v1/employee/employees/EMP-004", admin, nil)
	decodeInto(t, after, &employee)
	if employee.PhotoURL != photo.URL {
		t.Fatalf("نشانی عکس باید در پروندهٔ کارمند باشد؛ %q در برابر %q", employee.PhotoURL, photo.URL)
	}
	if employee.PhotoUpdatedAt == "" {
		t.Errorf("تاریخ بارگذاری عکس باید در پرونده بیاید")
	}
	list := call(t, h, http.MethodGet, "/api/v1/employee/employees?size=50", admin, nil)
	var page contracts.EmployeePage
	decodeInto(t, list, &page)
	found := false
	for _, item := range page.Items {
		if item.ID == "EMP-004" && item.PhotoURL == photo.URL {
			found = true
		}
	}
	if !found {
		t.Errorf("فهرست کارکنان باید نشانی عکس را هم بدهد")
	}

	// ---------- سرو فایل عکس: با امضا ۲۰۰، بدون امضا ۴۰۳ ----------
	served := call(t, h, http.MethodGet, photo.URL, "", nil)
	if served.Code != http.StatusOK {
		t.Fatalf("سرو عکس با امضا باید ۲۰۰ بدهد، %d داد", served.Code)
	}
	if ct := served.Header().Get("Content-Type"); !strings.Contains(ct, "image/png") {
		t.Errorf("نوع محتوای عکس باید PNG باشد، «%s» شد", ct)
	}
	if served.Body.Len() == 0 {
		t.Errorf("بدنهٔ فایل عکس خالی است")
	}
	bare := strings.Split(photo.URL, "?")[0]
	if got := call(t, h, http.MethodGet, bare, "", nil); got.Code != http.StatusForbidden {
		t.Errorf("سرو عکس بدون امضا باید ۴۰۳ بدهد، %d داد", got.Code)
	}
	// امضای کارمند دیگر نباید همین فایل را باز کند.
	if got := call(t, h, http.MethodGet, strings.Replace(photo.URL, "e=EMP-004", "e=EMP-005", 1), "", nil); got.Code != http.StatusForbidden {
		t.Errorf("امضای عکس کارمند دیگر باید رد شود، %d داد", got.Code)
	}

	// ---------- ساختار سازمانی: متصدی، عکس و نتیجهٔ احراز ----------
	roster := call(t, h, http.MethodGet, "/api/v1/orgstructure/roster", admin, nil)
	if roster.Code != http.StatusOK {
		t.Fatalf("ساختار سازمانی باید ۲۰۰ بدهد، %d داد: %s", roster.Code, roster.Body.String())
	}
	var view rosterResponse
	decodeInto(t, roster, &view)
	if len(view.Positions) == 0 {
		t.Fatal("ساختار سازمانی باید پست داشته باشد")
	}
	if len(view.QualificationOptions) != 3 {
		t.Errorf("سه نتیجهٔ احراز انتظار می‌رفت، %d آمد", len(view.QualificationOptions))
	}
	if view.Summary.Assignments < 5 || view.Summary.Active < 3 {
		t.Errorf("جمع‌بندی انتصاب‌ها درست نیست: %+v", view.Summary)
	}

	var finance struct {
		found      bool
		hasHolder  bool
		hasPhoto   bool
		metChecked bool
	}
	for _, p := range view.Positions {
		if p.Code != "POS-201" {
			continue
		}
		finance.found = true
		if p.Holder == nil {
			t.Fatal("پست کارشناس ارشد مالی باید متصدی داشته باشد")
		}
		finance.hasHolder = true
		if p.Holder.EmployeeID != "EMP-004" {
			t.Errorf("متصدی پست POS-201 باید EMP-004 باشد، %q آمد", p.Holder.EmployeeID)
		}
		if p.Holder.FullName == "" {
			t.Errorf("نام متصدی باید از پروندهٔ کارمند بیاید")
		}
		finance.hasPhoto = p.Holder.HasPhoto && p.Holder.PhotoURL == photo.URL
		if p.Holder.Qualification != "met" || p.Holder.QualificationFa != "احرازشده" {
			t.Errorf("نتیجهٔ احراز متصدی درست نیست: %q/%q", p.Holder.Qualification, p.Holder.QualificationFa)
		}
		if p.Holder.Evidence == "" {
			t.Errorf("شواهد احراز متصدی باید نمایش داده شود")
		}
		finance.metChecked = p.RequirementsMet
		if p.Requirements == "" {
			t.Errorf("شرط‌های احراز پست باید در پاسخ باشد")
		}
	}
	if !finance.found || !finance.hasHolder || !finance.hasPhoto || !finance.metChecked {
		t.Fatalf("سطر پست مالی کامل نیست: %+v", finance)
	}
	if view.NoteFa == "" {
		t.Errorf("پاسخ ساختار سازمانی باید یادداشت فارسی داشته باشد")
	}

	// ---------- مسیر سابقه و تاریخچهٔ یک پست ----------
	var positionID string
	for _, p := range view.Positions {
		if p.Code == "POS-405" { // سرپرست فروش: یک انتصاب جاری و یک انتصاب پیشین دارد
			positionID = p.ID
		}
	}
	if positionID == "" {
		t.Fatal("پست سرپرست فروش پیدا نشد")
	}
	history := call(t, h, http.MethodGet, "/api/v1/orgstructure/positions/"+positionID+"/history", admin, nil)
	if history.Code != http.StatusOK {
		t.Fatalf("تاریخچهٔ پست باید ۲۰۰ بدهد، %d داد: %s", history.Code, history.Body.String())
	}
	var hist struct {
		Position struct {
			Title           string `json:"title"`
			AssignmentCount int    `json:"assignmentCount"`
			HolderCount     int    `json:"holderCount"`
		} `json:"position"`
		Assignments []contracts.Record `json:"assignments"`
		Timeline    []struct {
			Date     string `json:"date"`
			Source   string `json:"source"`
			SourceFa string `json:"sourceFa"`
			ActionFa string `json:"actionFa"`
			Actor    string `json:"actor"`
			NoteFa   string `json:"noteFa"`
		} `json:"timeline"`
		NoteFa string `json:"noteFa"`
	}
	decodeInto(t, history, &hist)
	if hist.Position.Title == "" || hist.Position.AssignmentCount < 2 {
		t.Fatalf("سابقهٔ انتصاب‌های پست ناقص است: %+v", hist.Position)
	}
	if len(hist.Timeline) < 3 {
		t.Fatalf("تاریخچهٔ تغییرات باید سطر داشته باشد، %d سطر آمد", len(hist.Timeline))
	}
	sawPosition, sawAssignment, sawEnd := false, false, false
	for _, entry := range hist.Timeline {
		switch entry.Source {
		case "position":
			sawPosition = true
		case "assignment":
			sawAssignment = true
		}
		if entry.Date == "" || entry.ActionFa == "" {
			t.Errorf("هر سطر تاریخچه باید تاریخ شمسی و عنوان اقدام داشته باشد: %+v", entry)
		}
		if strings.Contains(entry.NoteFa, "پایان") {
			sawEnd = true
		}
	}
	if !sawPosition || !sawAssignment {
		t.Errorf("تاریخچه باید هم رخداد شناسنامهٔ پست و هم رخداد انتصاب را داشته باشد")
	}
	if !sawEnd {
		t.Errorf("تاریخچه باید رخداد «پایان تصدی» انتصاب پیشین را هم نشان بدهد")
	}
	// انتصاب پایان‌یافته باید در سابقه بماند.
	ended := false
	for _, a := range hist.Assignments {
		if a.Status == "ended" {
			ended = true
		}
	}
	if !ended {
		t.Errorf("انتصاب پایان‌یافته باید در سابقهٔ پست بماند")
	}
	if hist.NoteFa == "" {
		t.Errorf("پاسخ تاریخچه باید یادداشت فارسی داشته باشد")
	}

	// ---------- قاعدهٔ دامنه: انتصاب بدون احراز شرایط مجاز نیست ----------
	createdID := createAssignment(t, h, admin, map[string]any{
		"kind": "assignments", "title": "انتصاب آزمایشی کارشناس منابع انسانی", "employeeId": "EMP-007",
		"fields": map[string]string{
			"position": "POS-120", "positionTitle": "کارشناس منابع انسانی", "unit": "منابع انسانی",
			"startDate": "1405/09/01", "qualification": "not_met", "evidence": "پروندهٔ احراز ناقص است",
		},
	})
	act := call(t, h, http.MethodPost, "/api/v1/orgstructure/records/"+createdID+"/action", admin,
		map[string]any{"action": "appoint"})
	if act.Code != http.StatusUnprocessableEntity {
		t.Fatalf("انتصاب با احرازنشده باید ۴۲۲ بدهد، %d داد: %s", act.Code, act.Body.String())
	}
	var ruleErr contracts.Error
	decodeInto(t, act, &ruleErr)
	if !strings.Contains(ruleErr.Message, "احرازشده") {
		t.Errorf("پیام قاعده باید به «احرازشده» اشاره کند: %q", ruleErr.Message)
	}

	// ---------- انتصاب کامل: احرازشده + شواهد، سپس آغاز تصدی ----------
	validID := createAssignment(t, h, admin, map[string]any{
		"kind": "assignments", "title": "انتصاب کارشناس منابع انسانی", "employeeId": "EMP-007",
		"fields": map[string]string{
			"position": "POS-120", "positionTitle": "کارشناس منابع انسانی", "unit": "منابع انسانی",
			"startDate": "1405/09/01", "decreeNo": "ABL-1405-002",
			"qualification": "met",
			"evidence":      "کارشناسی مدیریت منابع انسانی، ۵ سال سابقهٔ سرپرستی، صورت‌جلسهٔ کمیتهٔ انتصابات",
		},
	})
	passed := call(t, h, http.MethodPost, "/api/v1/orgstructure/records/"+validID+"/action", admin,
		map[string]any{"action": "appoint"})
	if passed.Code != http.StatusOK {
		t.Fatalf("انتصاب کامل باید ۲۰۰ بدهد، %d داد: %s", passed.Code, passed.Body.String())
	}
	var appointed contracts.Record
	decodeInto(t, passed, &appointed)
	if appointed.Status != "active" {
		t.Fatalf("وضعیت انتصاب پس از اقدام باید active باشد، %q آمد", appointed.Status)
	}

	// پست بلا تصدی باید الان متصدی داشته باشد.
	again := call(t, h, http.MethodGet, "/api/v1/orgstructure/roster", admin, nil)
	var refreshed rosterResponse
	decodeInto(t, again, &refreshed)
	filled := false
	for _, p := range refreshed.Positions {
		if p.Code == "POS-120" && p.Holder != nil && p.Holder.EmployeeID == "EMP-007" {
			filled = true
		}
	}
	if !filled {
		t.Error("پس از آغاز تصدی، پست POS-120 باید متصدی EMP-007 را نشان بدهد")
	}

	// ---------- گزینه‌های فرم انتصاب ----------
	options := call(t, h, http.MethodGet, "/api/v1/orgstructure/options", admin, nil)
	if options.Code != http.StatusOK {
		t.Fatalf("گزینه‌های انتصاب باید ۲۰۰ بدهد، %d داد", options.Code)
	}
	var opts struct {
		Qualifications []map[string]string `json:"qualifications"`
		Positions      []map[string]string `json:"positions"`
		NoteFa         string              `json:"noteFa"`
	}
	decodeInto(t, options, &opts)
	if len(opts.Qualifications) != 3 || len(opts.Positions) == 0 || opts.NoteFa == "" {
		t.Errorf("پاسخ گزینه‌ها کامل نیست: %+v", opts)
	}

	// ---------- حذف عکس ----------
	if del := call(t, h, http.MethodDelete, "/api/v1/employee/employees/EMP-004/photo", admin, nil); del.Code != http.StatusOK {
		t.Fatalf("حذف عکس باید ۲۰۰ بدهد، %d داد: %s", del.Code, del.Body.String())
	}
	cleared := call(t, h, http.MethodGet, "/api/v1/employee/employees/EMP-004", admin, nil)
	var afterDelete contracts.Employee
	decodeInto(t, cleared, &afterDelete)
	if afterDelete.PhotoURL != "" {
		t.Errorf("پس از حذف، پرونده نباید نشانی عکس داشته باشد: %q", afterDelete.PhotoURL)
	}
	if got := call(t, h, http.MethodGet, photo.URL, "", nil); got.Code != http.StatusNotFound {
		t.Errorf("فایل عکس حذف‌شده باید ۴۰۴ بدهد، %d داد", got.Code)
	}
}

// createAssignment یک رکورد انتصاب می‌سازد و شناسهٔ آن را برمی‌گرداند.
func createAssignment(t *testing.T, h http.Handler, token string, body map[string]any) string {
	t.Helper()
	created := call(t, h, http.MethodPost, "/api/v1/orgstructure/records", token, body)
	if created.Code != http.StatusCreated {
		t.Fatalf("ساخت انتصاب باید ۲۰۱ بدهد، %d داد: %s", created.Code, created.Body.String())
	}
	var rec contracts.Record
	decodeInto(t, created, &rec)
	if rec.ID == "" {
		t.Fatal("شناسهٔ انتصاب ساخته‌شده خالی است")
	}
	return rec.ID
}
