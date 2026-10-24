// آزمون جریان سرتاسری «standalone»: سیم‌کشی ماژول‌ها، سیاست‌ها و گردش‌کار.

package bootstrap

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"hrms/internal/contracts"
	"hrms/internal/platform/localstore"
)

// newStandaloneServer «سیستم پاداش» را روی یک پوشهٔ دادهٔ موقت بالا می‌آورد.
// مسیر بازگشتی همان پوشهٔ داده است تا آزمون بتواند ماندگاری را پس از راه‌اندازی دوباره بسنجد.
func newStandaloneServer(t *testing.T, dataDir string) (http.Handler, StandaloneInfo) {
	t.Helper()
	if dataDir == "" {
		dataDir = t.TempDir()
	}
	handler, info, err := NewStandalone(StandaloneConfig{
		DataDir: dataDir,
		// کلید آزمون ثابت تا توکن‌ها بین دو راه‌اندازی همان پوشه قابل استفاده بمانند.
		JWTSecret: []byte("standalone-test-secret-key-1234567890"),
	})
	if err != nil {
		t.Fatalf("ساخت اجرای مستقل ناموفق: %v", err)
	}
	return handler, info
}

// TestStandaloneBonusFlow جریان کامل «سیستم پاداش» را می‌سنجد: ورود از فایل
// کاربران، تخصیص پاداش از فهرست CSV، تأیید و پرداخت، و ماندگاری پس از راه‌اندازی دوباره.
//
// چرا این آزمون مهم است: وعدهٔ این زیرسامانه «اجرا روی یک رایانه با یک فایل اجرایی» است؛
// اگر ماندگاری یا مسیرهای پایه بشکند، همین‌جا دیده می‌شود، نه وقتی کاربر دادهٔ واقعی
// وارد کرده است.
func TestStandaloneBonusFlow(t *testing.T) {
	dataDir := t.TempDir()
	h, info := newStandaloneServer(t, dataDir)

	if len(info.Modules) != 1 || info.Modules[0] != "bonus" {
		t.Fatalf("اجرای مستقل باید تنها ماژول پاداش را داشته باشد، %v داشت", info.Modules)
	}
	if info.Store != "json-file" {
		t.Fatalf("انبار اجرای مستقل باید فایل JSON باشد، «%s» بود", info.Store)
	}
	if info.StaffCount == 0 {
		t.Fatal("اجرای نخست باید فایل نمونهٔ کارکنان را بکارد")
	}
	if info.RecordCount == 0 {
		t.Fatal("اجرای نخست باید نمونهٔ پرداخت‌های پاداش را بکارد")
	}

	// ورود با کاربر پیش‌فرضِ فایل کاربران.
	session := login(t, h, localstore.DefaultAdminUser, localstore.DefaultAdminPassword)
	token := session.AccessToken

	// وضعیت انبار و فهرست کارکنان: دو مسیر ویژهٔ همین اجرا.
	rec := call(t, h, http.MethodGet, "/api/v1/standalone/info", token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("وضعیت اجرای مستقل باید ۲۰۰ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	var status StandaloneInfo
	decodeInto(t, rec, &status)
	if !strings.HasSuffix(status.RecordsPath, "records.json") {
		t.Fatalf("مسیر داده باید فایل records.json در پوشهٔ داده باشد: %s", status.RecordsPath)
	}

	rec = call(t, h, http.MethodGet, "/api/v1/standalone/staff?size=100", token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("فهرست کارکنان باید ۲۰۰ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	var staff contracts.EmployeePage
	decodeInto(t, rec, &staff)
	if staff.Total < 10 {
		t.Fatalf("فهرست نمونهٔ کارکنان بارگذاری نشده (total=%d)", staff.Total)
	}

	// گزینه‌های فرم تخصیص: نوع پاداش، مدیریت و کارکنان.
	rec = call(t, h, http.MethodGet, "/api/v1/bonus/options", token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("گزینه‌های پاداش باید ۲۰۰ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	var options contracts.BonusChoices
	decodeInto(t, rec, &options)
	if len(options.Types) == 0 || len(options.Managements) == 0 || len(options.PaymentMethods) == 0 {
		t.Fatalf("گزینه‌های فرم ناقص است: %+v", options)
	}
	if options.TodayFa == "" {
		t.Fatal("تاریخ امروز برای فرم تخصیص برنگشت")
	}

	// طرح‌وارهٔ رکورد: فرم وب از همین وضعیت‌ها، اکشن‌ها و وضعیت‌های مجاز ویرایش را
	// می‌سازد؛ پس اگر روزی دامنه عوض شود، آزمون UI را هم می‌گیرد.
	rec = call(t, h, http.MethodGet, "/api/v1/bonus/schema", token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("طرح‌وارهٔ پاداش باید ۲۰۰ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	var schema contracts.RecordSchema
	decodeInto(t, rec, &schema)
	if len(schema.Kinds) != 1 {
		t.Fatalf("طرح‌وارهٔ پاداش باید یک نوع رکورد داشته باشد، %d داشت", len(schema.Kinds))
	}
	kind := schema.Kinds[0]
	if kind.Key != "bonusPayments" || kind.InitialStatus == "" || len(kind.Statuses) == 0 || len(kind.Actions) == 0 {
		t.Fatalf("طرح‌وارهٔ نوع رکورد ناقص است: %+v", kind)
	}
	if len(kind.EditableStatuses) == 0 {
		t.Fatal("وضعیت‌های مجاز ویرایش باید از سرور به کلاینت بیاید")
	}

	// تخصیص پاداش: پیش‌نمایش سپس ثبت.
	request := map[string]any{
		"bonusType": "retention", "management": "kimia", "amount": 15000000,
		"paymentDate": jdate(t, 0), "paymentMethod": "payroll",
		"decreeNo": "1404/م/0700", "employeeIds": []string{"EMP-009"},
	}
	rec = call(t, h, http.MethodPost, "/api/v1/bonus/allocate/preview", token, request)
	if rec.Code != http.StatusOK {
		t.Fatalf("پیش‌نمایش تخصیص باید ۲۰۰ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	var preview contracts.BonusAllocationResult
	decodeInto(t, rec, &preview)
	if preview.Count != 1 || preview.TotalAmount != 15000000 {
		t.Fatalf("پیش‌نمایش نادرست: %+v", preview)
	}

	rec = call(t, h, http.MethodPost, "/api/v1/bonus/allocate", token, request)
	if rec.Code != http.StatusCreated {
		t.Fatalf("ثبت تخصیص باید ۲۰۱ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	var allocated contracts.BonusAllocationResult
	decodeInto(t, rec, &allocated)
	if allocated.Created != 1 || len(allocated.Rows) != 1 || allocated.Rows[0].RecordID == "" {
		t.Fatalf("ثبت تخصیص رکورد نساخت: %+v", allocated)
	}
	recordID := allocated.Rows[0].RecordID

	// رد قاعده در دامنه: مبلغ غیرمجاز باید ۴۲۲ بدهد، نه اینکه بی‌صدا ثبت شود.
	bad := map[string]any{
		"bonusType": "retention", "management": "kimia", "amount": -5,
		"paymentDate": jdate(t, 0), "paymentMethod": "payroll", "employeeIds": []string{"EMP-009"},
	}
	if rec := call(t, h, http.MethodPost, "/api/v1/bonus/allocate", token, bad); rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("مبلغ نامعتبر باید ۴۲۲ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}

	// گردش‌کار: تأیید پرداخت سپس ثبت پرداخت.
	for _, action := range []string{"approve", "pay"} {
		rec = call(t, h, http.MethodPost, "/api/v1/bonus/records/"+recordID+"/action", token,
			map[string]string{"action": action})
		if rec.Code != http.StatusOK {
			t.Fatalf("اکشن %s باید ۲۰۰ بدهد، %d داد: %s", action, rec.Code, rec.Body.String())
		}
	}
	rec = call(t, h, http.MethodGet, "/api/v1/bonus/records/"+recordID, token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("خواندن رکورد باید ۲۰۰ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	var record contracts.Record
	decodeInto(t, rec, &record)
	if record.Status != "paid" {
		t.Fatalf("وضعیت رکورد پس از پرداخت باید paid باشد، «%s» بود", record.Status)
	}
	if record.EmployeeID != "EMP-009" {
		t.Fatalf("رکورد باید به کارمند تخصیص‌یافته بسته باشد، «%s» بود", record.EmployeeID)
	}

	// ماندگاری: همان پوشهٔ داده، اجرای تازه.
	reopened, reopenedInfo := newStandaloneServer(t, dataDir)
	if reopenedInfo.RecordCount <= info.RecordCount {
		t.Fatalf("شمار رکوردها پس از راه‌اندازی دوباره باید افزایش یافته باشد (%d → %d)",
			info.RecordCount, reopenedInfo.RecordCount)
	}
	session2 := login(t, reopened, localstore.DefaultAdminUser, localstore.DefaultAdminPassword)
	rec = call(t, reopened, http.MethodGet, "/api/v1/bonus/records/"+recordID, session2.AccessToken, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("رکورد باید پس از راه‌اندازی دوباره خوانده شود، %d داد: %s", rec.Code, rec.Body.String())
	}
	decodeInto(t, rec, &record)
	if record.Status != "paid" {
		t.Fatalf("وضعیت رکورد پس از راه‌اندازی دوباره باید paid بماند، «%s» بود", record.Status)
	}
}

// TestStandaloneKeepsSessionsAcrossRestart می‌سنجد کاربر پس از بسته شدن و بالا آمدن دوبارهٔ
// سامانه از ورود بیرون نمی‌افتد: کلید امضا در پوشهٔ داده می‌ماند، نه در حافظه.
//
// چرا مهم است: اگر کلید هر بار تصادفی ساخته شود، همهٔ نشست‌های فعال با هر راه‌اندازی باطل
// می‌شوند — رفتاری که روی یک رایانهٔ کاری آزاردهنده و بی‌دلیل است.
func TestStandaloneKeepsSessionsAcrossRestart(t *testing.T) {
	dataDir := t.TempDir()
	first, _, err := NewStandalone(StandaloneConfig{DataDir: dataDir})
	if err != nil {
		t.Fatalf("ساخت اجرای مستقل ناموفق: %v", err)
	}
	token := login(t, first, localstore.DefaultAdminUser, localstore.DefaultAdminPassword).AccessToken
	if _, err := os.Stat(filepath.Join(dataDir, "secret.key")); err != nil {
		t.Fatalf("کلید امضا باید در پوشهٔ داده ساخته شود: %v", err)
	}

	second, _, err := NewStandalone(StandaloneConfig{DataDir: dataDir})
	if err != nil {
		t.Fatalf("راه‌اندازی دوباره ناموفق: %v", err)
	}
	// همان توکن پیش از راه‌اندازی دوباره باید هنوز معتبر باشد.
	if rec := call(t, second, http.MethodGet, "/api/v1/auth/me", token, nil); rec.Code != http.StatusOK {
		t.Fatalf("توکن پیش از راه‌اندازی دوباره باید معتبر بماند، %d داد: %s", rec.Code, rec.Body.String())
	}
}

// TestStandaloneServesOnlyBonusModule می‌سنجد اجرای مستقل تفکیکی است: مسیرهای ماژول‌های
// دیگر نه ثبت می‌شوند و نه به‌جای ۴۰۴ صفحهٔ HTML می‌دهند.
func TestStandaloneServesOnlyBonusModule(t *testing.T) {
	h, _ := newStandaloneServer(t, "")
	token := login(t, h, localstore.DefaultAdminUser, localstore.DefaultAdminPassword).AccessToken

	// مسیر درست ماژول پاداش کار می‌کند.
	if rec := call(t, h, http.MethodGet, "/api/v1/bonus/options", token, nil); rec.Code != http.StatusOK {
		t.Fatalf("مسیر پاداش باید ۲۰۰ بدهد، %d داد", rec.Code)
	}

	// ماژول دیگری در این اجرا وجود ندارد: باید ۴۰۴ JSON بدهد.
	for _, path := range []string{"/api/v1/loans/options", "/api/v1/settings/records", "/api/v1/empl/unknown"} {
		rec := call(t, h, http.MethodGet, path, token, nil)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("مسیر ناشناس %s باید ۴۰۴ بدهد، %d داد: %s", path, rec.Code, rec.Body.String())
		}
		if got := rec.Header().Get("Content-Type"); !strings.Contains(got, "application/json") {
			t.Fatalf("پاسخ ۴۰۴ مسیر %s باید JSON باشد، «%s» بود: %s", path, got, rec.Body.String())
		}
	}

	// بدون توکن، مسیرهای داده باید ۴۰۱ بدهند (نه HTML رابط).
	for _, path := range []string{"/api/v1/bonus/records", "/api/v1/standalone/info"} {
		if rec := call(t, h, http.MethodGet, path, "", nil); rec.Code != http.StatusUnauthorized {
			t.Fatalf("مسیر %s بدون توکن باید ۴۰۱ بدهد، %d داد", path, rec.Code)
		}
	}
}

// TestStandaloneServesEmbeddedUI می‌سنجد رابط وب در خود باینری جاسازی شده و روی همان
// مبدأ بالا می‌آید: RTL، فارسی، و مسیر تازه در آدرس کاربر را به ۴۰۴ نمی‌برد.
func TestStandaloneServesEmbeddedUI(t *testing.T) {
	h, _ := newStandaloneServer(t, "")

	rec := call(t, h, http.MethodGet, "/", "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("ریشه باید رابط را بدهد، %d داد", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `dir="rtl"`) || !strings.Contains(body, `lang="fa"`) {
		t.Fatal("رابط باید راست‌به‌چپ و فارسی باشد")
	}
	if !strings.Contains(body, "پاداش") {
		t.Fatal("رابط باید عنوان فارسی پاداش را داشته باشد")
	}

	// آرم سازمان هم داخل همین باینری جاسازی شده است. سرصفحهٔ برنامه از همین مسیر
	// آرم را می‌گیرد، پس نباید به هیچ فایل کنارِ باینری وابسته باشد.
	if !strings.Contains(body, "/assets/brand/logo-pgpic.jpg") {
		t.Fatal("سرصفحهٔ رابط باید آرم سازمان را ارجاع دهد")
	}
	logo := call(t, h, http.MethodGet, "/assets/brand/logo-pgpic.jpg", "", nil)
	if logo.Code != http.StatusOK {
		t.Fatalf("آرم سازمان باید از خود باینری سرو شود، %d داد", logo.Code)
	}
	if ct := logo.Header().Get("Content-Type"); !strings.HasPrefix(ct, "image/") {
		t.Fatalf("نوع محتوای آرم باید تصویر باشد، «%s» بود", ct)
	}

	// مسیر ناشناس غیر-API: همان صفحهٔ تک‌فایلی.
	deep := call(t, h, http.MethodGet, "/bonus/payments/42", "", nil)
	if deep.Code != http.StatusOK || deep.Body.String() != body {
		t.Fatalf("مسیر تازهٔ غیر-API باید همان رابط را بدهد، کد %d", deep.Code)
	}
}

// TestStandaloneUsesLocalJSONStore می‌سنجد داده در فایل JSON پوشهٔ داده می‌نشیند، نه در
// حافظه: این همان چیزی است که پشتیبان‌گیری «کپی یک پوشه» را ممکن می‌کند.
func TestStandaloneUsesLocalJSONStore(t *testing.T) {
	dataDir := t.TempDir()
	h, _ := newStandaloneServer(t, dataDir)
	token := login(t, h, localstore.DefaultAdminUser, localstore.DefaultAdminPassword).AccessToken

	rec := call(t, h, http.MethodGet, "/api/v1/bonus/records?size=5", token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("فهرست رکوردها باید ۲۰۰ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	var page contracts.RecordPage
	decodeInto(t, rec, &page)
	if page.Total == 0 {
		t.Fatal("نمونهٔ پرداخت‌ها بارگذاری نشده است")
	}

	recordsFile := filepath.Join(dataDir, "records.json")
	raw := readFile(t, recordsFile)
	var snap struct {
		Version int                `json:"version"`
		Records []contracts.Record `json:"records"`
	}
	if err := json.Unmarshal(raw, &snap); err != nil {
		t.Fatalf("فایل داده باید JSON خوانا باشد: %v", err)
	}
	if len(snap.Records) == 0 {
		t.Fatal("فایل داده هیچ رکوردی ندارد")
	}
	if snap.Records[0].ID == "" || snap.Records[0].Kind == "" {
		t.Fatalf("رکورد ذخیره‌شده ناقص است: %+v", snap.Records[0])
	}

	// پوشهٔ داده خودکفاست: پشتیبان‌گیری یعنی کپی همین سه فایل.
	// (secret.key فقط وقتی ساخته می‌شود که کلید امضا تزریق نشده باشد؛ در آزمون کلید
	// ثابت تزریق شده است، پس این فایل انتظار نمی‌رود.)
	for _, name := range []string{"records.json", "users.json", "staff.csv"} {
		if len(readFile(t, filepath.Join(dataDir, name))) == 0 {
			t.Fatalf("فایل %s باید در پوشهٔ داده ساخته شود", name)
		}
	}
}

// readFile فایل آزمون را می‌خواند و در نبود آن، آزمون را شکست می‌دهد.
func readFile(t *testing.T, path string) []byte {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("خواندن فایل «%s» ناموفق: %v", path, err)
	}
	return raw
}
