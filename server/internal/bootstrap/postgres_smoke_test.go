// آزمون جریان سرتاسری «postgres_smoke_test»: سیم‌کشی ماژول‌ها، سیاست‌ها و گردش‌کار.

package bootstrap

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"hrms/internal/contracts"
	employeedomain "hrms/internal/modules/employee/domain"
	platformmodules "hrms/internal/platform/modules"
)

// آزمون دودی Postgres.
//
// چرا این آزمون جدا از بقیهٔ آزمون‌هاست: همهٔ آزمون‌های دیگر روی انبار درون‌حافظه‌ای
// اجرا می‌شوند، پس «بذرکاری ماژول‌ها روی پایگاه‌دادهٔ واقعی» و «دسترسی هم‌زمان دو نمونهٔ
// سرور به یک داده» را نمی‌سنجند. اینجا سرور واقعی (همان NewDeps و NewHandler با DSN
// Postgres) بالا می‌آید، مهاجرت‌ها اعمال می‌شوند، دادهٔ نمونه بذرکاری می‌شود و سپس:
//
//	۱. هر ماژول فعال باید پُر باشد — هر نوع رکورد دست‌کم یک رکورد، و ماژول‌های
//	   غیررکوردی از مسیر فهرست خودشان.
//	۲. چرخهٔ ایجاد ← جستجو ← ویرایش ← حذف برای هر نوع رکورد در همهٔ ماژول‌ها.
//	۳. رکورد ساخته‌شده در یک نمونه باید از نمونهٔ دوم همان سرور روی همان DSN خوانده
//	   شود؛ اگر انبار حافظه‌ای بود، نمونهٔ دوم چیزی نمی‌دید.
//
// اجرا: متغیر محیطی HRMS_TEST_DATABASE_URL را به یک پایگاه‌دادهٔ خالی/آزمایشی بدهید و
// «go test -run PostgresSmoke -v ./internal/bootstrap/» را اجرا کنید. بدون آن متغیر،
// آزمون رد می‌شود (skip) تا «go test ./...» محلی به پایگاه‌دادهٔ توسعۀ کسی دست نزند.
// همین آزمون در CI روی سرویس Postgres اجرا می‌شود (گردش‌کار .github/workflows/ci.yml).

// smokeDatabaseEnv نشانی پایگاه‌دادهٔ آزمون دودی.
const smokeDatabaseEnv = "HRMS_TEST_DATABASE_URL"

// smokeNonRecordProbes مسیر «پُری» ماژول‌هایی که موتور مشترک رکورد ندارند: شناسهٔ ماژول
// به مسیری که باید فهرستی ناتهی برگرداند.
//
// ماژول رکوردی خودش را با /schema و /records می‌سنجیم؛ این پنج ماژول انبار و API
// اختصاصی دارند، پس یک مسیر فهرست معتبر برای هرکدام اعلام می‌شود. افزودن ماژول تازه
// بدون این اعلام، آزمون را قرمز می‌کند تا پُری آن فراموش نشود.
var smokeNonRecordProbes = map[string]string{
	"dashboard":      "/api/v1/dashboard/widgets",
	"employee":       "/api/v1/employee/employees?size=1",
	"contracts":      "/api/v1/contracts/contracts?size=1",
	"staff_services": "/api/v1/personnel/dependents?size=1",
	"salary_admin":   "/api/v1/salary_admin/decrees?size=1",
}

// smokeHandler سرور کامل را روی پایگاه‌دادهٔ آزمون بالا می‌آورد.
//
// ServerRoot نسبت به پوشهٔ همین بسته («server/internal/bootstrap») داده می‌شود؛ پس
// مهاجرت‌ها از «server/migrations» و «server/internal/modules/*/migrations» خوانده
// می‌شوند — همان مسیری که اجرای واقعی سرور می‌بیند.
func smokeHandler(t *testing.T) (http.Handler, []string) {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv(smokeDatabaseEnv))
	if dsn == "" {
		t.Skipf("آزمون دودی Postgres اجرا نشد: %s تنظیم نشده است", smokeDatabaseEnv)
	}
	return smokeNewHandler(t, dsn)
}

func smokeNewHandler(t *testing.T, dsn string) (http.Handler, []string) {
	t.Helper()
	deps, err := NewDeps(Config{
		JWTSecret:   []byte("postgres-smoke-test-secret-key"),
		DatabaseURL: dsn,
		ServerRoot:  "../..",
		Migrate:     true,
		SeedDemo:    true,
		// آزمون دودی با حساب نمایشی وارد می‌شود؛ در استقرار واقعی همین پرچم خاموش است
		// و حساب‌ها از فایل کاربران خوانده می‌شوند (بند OPS-03).
		AllowDemoAccounts: true,
		Logger:            slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	if err != nil {
		t.Fatalf("بالا آوردن سرور روی پایگاه‌دادهٔ آزمون ناموفق بود: %v", err)
	}
	if deps.DB == nil {
		t.Fatalf("انتظار می‌رفت سرور روی Postgres بالا بیاید، ولی انبار درون‌حافظه‌ای شد (DSN=%q)", dsn)
	}
	handler, registered, err := NewHandler(deps)
	if err != nil {
		t.Fatalf("ساخت هندلر ناموفق بود: %v", err)
	}
	if want := len(Modules()); len(registered) != want {
		t.Fatalf("همهٔ %d ماژول باید ثبت شوند، %d ثبت شد", want, len(registered))
	}
	return handler, registered
}

// smokeReady آمادگی سرور را می‌سنجد: پایگاه‌داده پاسخ می‌دهد و مهاجرت‌ها اعمال شده‌اند.
func smokeReady(t *testing.T, h http.Handler) {
	t.Helper()
	rec := call(t, h, http.MethodGet, "/readyz", "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("بررسی آمادگی باید ۲۰۰ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Status            string `json:"status"`
		Database          string `json:"database"`
		MigrationsApplied int    `json:"migrationsApplied"`
		Modules           int    `json:"modules"`
		AuditChain        string `json:"auditChain"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("پاسخ /readyz قابل تجزیه نیست: %v", err)
	}
	if body.Status != "ok" || body.Database != "connected" {
		t.Fatalf("سرور روی Postgres آماده نیست: %s", rec.Body.String())
	}
	if body.MigrationsApplied == 0 {
		t.Fatalf("هیچ مهاجرتی اعمال نشده است: %s", rec.Body.String())
	}
	if body.Modules != len(Modules()) {
		t.Fatalf("شمار ماژول‌های فعال با فهرست ماژول‌ها نمی‌خواند: %d در برابر %d", body.Modules, len(Modules()))
	}
	if body.AuditChain != "valid" {
		t.Fatalf("زنجیرهٔ لاگ حسابرسی معتبر نیست: %s", rec.Body.String())
	}
}

// TestPostgresSmokeEveryModuleIsPopulated پس از بالا آمدن سرور روی Postgres، پُر بودن
// همهٔ ماژول‌ها را می‌سنجد: هر ماژول رکوردی باید در «هر نوع رکورد» دست‌کم یک رکورد
// داشته باشد و هر ماژول غیررکوردی در مسیر فهرست خودش ناتهی باشد.
//
// چرا «هر نوع رکورد» و نه فقط ماژول: ماژول‌هایی مثل مطالعات منابع انسانی چهار نوع
// دارند و ممکن است یک نوع بی‌داده بماند در حالی که ماژول «پُر» به نظر می‌رسد؛ همان
// اتفاقی که تب شاخص‌ها و نظرسنجی را خالی نشان می‌داد.
func TestPostgresSmokeEveryModuleIsPopulated(t *testing.T) {
	h, registered := smokeHandler(t)
	smokeReady(t, h)
	admin := login(t, h, "admin", "demo")

	recordModules, kinds, probed := 0, 0, 0
	defects := []string{}
	for _, m := range platformmodules.All {
		if !smokeRegistered(registered, m.ID) {
			continue
		}
		rec := call(t, h, http.MethodGet, m.APIPrefix+"/schema", admin.AccessToken, nil)
		switch {
		case rec.Code == http.StatusOK:
			recordModules++
			var schema contracts.RecordSchema
			decodeInto(t, rec, &schema)
			if len(schema.Kinds) == 0 {
				defects = append(defects, fmt.Sprintf("%s: شناسهٔ ماژول هیچ نوع رکوردی ندارد", m.ID))
			}
			declared := map[string]bool{}
			for _, kind := range schema.Kinds {
				declared[kind.Key] = true
			}
			// سطرهای بی‌صاحب: رکوردی که نوعش در شناسهٔ همین ماژول اعلام نشده است. این
			// نشانهٔ پیشوند شناسهٔ تکراری با ماژول دیگر است؛ پیش از این ایراد، رکورد
			// «آگهی شغلی» استخدام با ماژول طبقه‌بندی مشاغل ذخیره شده بود.
			if orphans := smokeOrphanKinds(t, h, admin.AccessToken, m.APIPrefix, declared); len(orphans) > 0 {
				shown := orphans
				if len(shown) > 5 {
					shown = shown[:5]
				}
				defects = append(defects, fmt.Sprintf(
					"%s: %d رکورد با نوعی که در شناسهٔ این ماژول اعلام نشده است (نمونه: %s)",
					m.ID, len(orphans), strings.Join(shown, "، ")))
			}
			for _, kind := range schema.Kinds {
				kinds++
				list := call(t, h, http.MethodGet,
					m.APIPrefix+"/records?kind="+kind.Key+"&size=1&page=1", admin.AccessToken, nil)
				if list.Code != http.StatusOK {
					defects = append(defects, fmt.Sprintf("%s/%s: فهرست رکوردها کد %d داد",
						m.ID, kind.Key, list.Code))
					continue
				}
				var page contracts.RecordPage
				decodeInto(t, list, &page)
				if page.Total < 1 {
					defects = append(defects, fmt.Sprintf("%s/%s: نوع رکورد «%s» بی‌داده است",
						m.ID, kind.Key, kind.TitleFa))
				}
			}
		default:
			path, ok := smokeNonRecordProbes[m.ID]
			if !ok {
				defects = append(defects, fmt.Sprintf(
					"%s: نه موتور رکورد دارد (/schema کد %d) و نه مسیر سنجش پُری برایش اعلام شده است",
					m.ID, rec.Code))
				continue
			}
			probed++
			body := call(t, h, http.MethodGet, path, admin.AccessToken, nil)
			if body.Code != http.StatusOK {
				defects = append(defects, fmt.Sprintf("%s: مسیر %s کد %d داد", m.ID, path, body.Code))
				continue
			}
			if !smokeNonEmpty(body.Body.Bytes()) {
				defects = append(defects,
					fmt.Sprintf("%s: مسیر %s پاسخ داد ولی داده‌ای برنگرداند (%s)",
						m.ID, path, smokeClip(body.Body.String())))
			}
		}
	}

	if len(defects) > 0 {
		t.Fatalf("پُر نبودن ماژول‌ها (%d مورد):\n  - %s", len(defects), strings.Join(defects, "\n  - "))
	}
	t.Logf("همهٔ %d ماژول فعال پُر است: %d ماژول رکوردی با %d نوع رکورد و %d ماژول با مسیر اختصاصی",
		len(registered), recordModules, kinds, probed)
}

// TestPostgresSmokeRecordLifecycle چرخهٔ ایجاد ← جستجو ← ویرایش ← حذف را برای «هر نوع
// رکورد در هر ماژول» اجرا می‌کند.
//
// چرا از خودِ دادهٔ نمونهٔ همان نوع رکورد به‌عنوان الگوی مقادیر استفاده می‌شود: مقادیر
// فیلدهای اجباری (نوع پاداش، مقطع، کارت و بخش، کلید پارامتر…) قاعدهٔ دامنهٔ همان ماژول
// را باید برآورده کنند و فهرست مجازشان در سرور است، نه در آزمون. رکورد نمونهٔ همان نوع
// از همان مسیر اعتبارسنجی گذشته است؛ پس تکرار مقادیرش معتبر است و آزمون هیچ فهرست
// مجاز یا ثابتی را در خود تکرار نمی‌کند.
func TestPostgresSmokeRecordLifecycle(t *testing.T) {
	h, registered := smokeHandler(t)
	smokeReady(t, h)
	admin := login(t, h, "admin", "demo")
	// نشانهٔ یکتا برای جستجو: هر اجرا رکوردهای خودش را می‌سازد و پاک می‌کند.
	stamp := fmt.Sprintf("SMOKE-%d", time.Now().UnixNano())

	cycles := 0
	for _, m := range platformmodules.All {
		if !smokeRegistered(registered, m.ID) {
			continue
		}
		rec := call(t, h, http.MethodGet, m.APIPrefix+"/schema", admin.AccessToken, nil)
		if rec.Code != http.StatusOK {
			continue // ماژول غیررکوردی؛ پُری‌اش در آزمون بالا سنجیده می‌شود.
		}
		var schema contracts.RecordSchema
		decodeInto(t, rec, &schema)
		for _, kind := range schema.Kinds {
			smokeKindCycle(t, h, admin.AccessToken, m.ID, m.APIPrefix, kind, stamp)
			cycles++
		}
	}
	if cycles == 0 {
		t.Fatal("هیچ نوع رکوردی برای آزمون چرخه یافت نشد")
	}
	t.Logf("چرخهٔ ایجاد/جستجو/ویرایش/حذف روی %d نوع رکورد در %d ماژول کامل شد",
		cycles, len(registered))
}

// smokeKindCycle یک چرخهٔ کامل روی یک نوع رکورد اجرا می‌کند.
func smokeKindCycle(t *testing.T, h http.Handler, token, moduleID, prefix string, kind contracts.RecordKindSchema, stamp string) {
	t.Helper()
	label := moduleID + "/" + kind.Key

	// الگو: نخستین رکورد همین نوع (بذرکاری‌شده روی پایگاه‌داده).
	list := call(t, h, http.MethodGet, prefix+"/records?kind="+kind.Key+"&size=1", token, nil)
	if list.Code != http.StatusOK {
		t.Errorf("%s: فهرست رکوردها کد %d داد", label, list.Code)
		return
	}
	var page contracts.RecordPage
	decodeInto(t, list, &page)
	if len(page.Items) == 0 {
		t.Errorf("%s: برای آزمون چرخه رکوردی برای الگوگیری نیست", label)
		return
	}
	template := page.Items[0]

	input := contracts.RecordInput{
		Kind:       kind.Key,
		Title:      stamp + " — " + kind.TitleFa,
		EmployeeID: template.EmployeeID,
		Fields:     template.Fields,
		NoteFa:     stamp,
	}
	created := call(t, h, http.MethodPost, prefix+"/records", token, input)
	if created.Code != http.StatusCreated {
		t.Errorf("%s: ایجاد رکورد کد %d داد: %s", label, created.Code, smokeClip(created.Body.String()))
		return
	}
	var record contracts.Record
	decodeInto(t, created, &record)
	if record.Status != kind.InitialStatus {
		t.Errorf("%s: رکورد تازه باید در وضعیت آغازین «%s» باشد، «%s» بود",
			label, kind.InitialStatus, record.Status)
	}

	// جستجو: رکورد تازه باید با عبارت جستجو در سرور پیدا شود.
	found := call(t, h, http.MethodGet, prefix+"/records?kind="+kind.Key+"&query="+stamp+"&size=200", token, nil)
	if found.Code != http.StatusOK {
		t.Errorf("%s: جستجو کد %d داد", label, found.Code)
	} else {
		var hits contracts.RecordPage
		decodeInto(t, found, &hits)
		if !smokeHasID(hits, record.ID) {
			t.Errorf("%s: جستجوی «%s» رکورد تازه (%s) را نیاورد", label, stamp, record.ID)
		}
	}

	// ویرایش: همان مقادیر با یادداشت تازه باید نسخه را بالا ببرد.
	edited := input
	edited.NoteFa = stamp + " ویرایش‌شده"
	updated := call(t, h, http.MethodPatch, prefix+"/records/"+record.ID, token, edited)
	if updated.Code != http.StatusOK {
		t.Errorf("%s: ویرایش رکورد کد %d داد: %s", label, updated.Code, smokeClip(updated.Body.String()))
	} else {
		var after contracts.Record
		decodeInto(t, updated, &after)
		if !strings.Contains(after.NoteFa, "ویرایش‌شده") || after.Version <= record.Version {
			t.Errorf("%s: ویرایش اعمال نشد یا نسخه بالا نرفت: %+v", label, after)
		}
	}

	// حذف: رکورد باید از فهرست و از خواندن مستقیم بیرون برود.
	deleted := call(t, h, http.MethodDelete, prefix+"/records/"+record.ID, token, nil)
	if deleted.Code != http.StatusNoContent {
		t.Errorf("%s: حذف رکورد کد %d داد: %s", label, deleted.Code, smokeClip(deleted.Body.String()))
		return
	}
	gone := call(t, h, http.MethodGet, prefix+"/records/"+record.ID, token, nil)
	if gone.Code != http.StatusNotFound {
		t.Errorf("%s: رکورد حذف‌شده باید ۴۰۴ بدهد، کد %d داد", label, gone.Code)
	}
	again := call(t, h, http.MethodGet, prefix+"/records?kind="+kind.Key+"&query="+stamp+"&size=200", token, nil)
	if again.Code == http.StatusOK {
		var hits contracts.RecordPage
		decodeInto(t, again, &hits)
		if smokeHasID(hits, record.ID) {
			t.Errorf("%s: رکورد حذف‌شده هنوز در جستجو دیده می‌شود", label)
		}
	}
}

// TestPostgresSmokeEmployeeUpdatePersists ویرایش پروندهٔ کارمند را روی Postgres می‌سنجد.
//
// چرا این آزمون: درج/به‌روزرسانی کارمند یک دستور upsert است و یک بار شمار پارامترهایش با
// ستون‌ها نمی‌خواند («got 21 parameters but the statement requires 22»)، پس هر ویرایش
// پرونده با خطای ۵۰۰ برمی‌گشت و تغییرهای پرسنلی (حقوق پایه، محل سکونت) ذخیره نمی‌شد.
// انبار درون‌حافظه‌ای این ایراد را نشان نمی‌داد و فقط آزمون روی Postgres می‌گیردش.
//
// نکتهٔ بهداشت دادهٔ آزمون: PATCH مقدار خالی را «بدون تغییر» می‌خواند، پس تنها فیلدی که
// می‌شود امن عوض و بازگرداند «حقوق پایه» است؛ محل سکونت فقط وقتی عوض می‌شود که از پیش
// مقدار داشته باشد تا بازگردانی‌اش ممکن باشد. گروه شغلی دست‌نخورده می‌ماند و تنها
// حفظ‌شدنش سنجیده می‌شود.
func TestPostgresSmokeEmployeeUpdatePersists(t *testing.T) {
	h, _ := smokeHandler(t)
	smokeReady(t, h)
	admin := login(t, h, "admin", "demo")

	list := call(t, h, http.MethodGet, "/api/v1/employee/employees?size=50", admin.AccessToken, nil)
	if list.Code != http.StatusOK {
		t.Fatalf("فهرست کارکنان کد %d داد: %s", list.Code, smokeClip(list.Body.String()))
	}
	var page contracts.EmployeePage
	decodeInto(t, list, &page)
	if len(page.Items) == 0 {
		t.Fatal("کارمندی برای این آزمون نیست؛ دادهٔ نمونهٔ کارکنان خالی است")
	}
	original := page.Items[0]
	for _, item := range page.Items {
		if item.ResidenceSite != "" {
			original = item
			break
		}
	}

	// بدنهٔ ویرایش همان چیزی است که رابط کاربری می‌فرستد: کل پرونده با مقدار تازه.
	// گروه شغلی: کلید معتبرِ نردبان گروه‌ها. مقدار خالی یعنی «تغییر نده»، پس برای
	// بازگرداندن پروندهٔ بدون گروه، نشانهٔ صریح پاک‌کردن فرستاده می‌شود.
	alternateGrade := "expert"
	if original.JobGrade == alternateGrade {
		alternateGrade = "senior_expert"
	}
	edited := contracts.EmployeeInput{
		FullName: original.FullName, NationalID: original.NationalID,
		DepartmentID: original.DepartmentID, Position: original.Position,
		ContractType: original.ContractType, EmploymentType: original.EmploymentType,
		Gender: original.Gender, BirthDate: original.BirthDate, Phone: original.Phone,
		Mobile: original.Mobile, HomePhone: original.HomePhone,
		PostalCode: original.PostalCode, Address: original.Address, Email: original.Email,
		HireDate: original.HireDate, Status: original.Status,
		BaseSalary:    original.BaseSalary + 1000000,
		ResidenceSite: original.ResidenceSite,
		JobGrade:      alternateGrade,
	}
	alteredSite := ""
	if original.ResidenceSite != "" {
		alteredSite = "abadan"
		if original.ResidenceSite == alteredSite {
			alteredSite = "mahshahr"
		}
		edited.ResidenceSite = alteredSite
	}

	path := "/api/v1/employee/employees/" + original.ID
	patched := call(t, h, http.MethodPatch, path, admin.AccessToken, edited)
	if patched.Code != http.StatusOK {
		t.Fatalf("ویرایش پروندهٔ کارمند ناموفق بود (کد %d): %s — دستور درج/به‌روزرسانی کارمند باید همهٔ ستون‌ها را با پارامترهایش هم‌خوان نگه دارد.", patched.Code, smokeClip(patched.Body.String()))
	}

	read := call(t, h, http.MethodGet, path, admin.AccessToken, nil)
	if read.Code != http.StatusOK {
		t.Fatalf("بازخوانی پروندهٔ ویرایش‌شده کد %d داد: %s", read.Code, smokeClip(read.Body.String()))
	}
	var saved contracts.Employee
	decodeInto(t, read, &saved)
	if saved.BaseSalary != edited.BaseSalary {
		t.Errorf("حقوق پایه پس از ویرایش %d است، انتظار %d بود", saved.BaseSalary, edited.BaseSalary)
	}
	if alteredSite != "" && saved.ResidenceSite != alteredSite {
		t.Errorf("محل سکونت پس از ویرایش %q است، انتظار %q بود", saved.ResidenceSite, alteredSite)
	}
	if saved.JobGrade != alternateGrade {
		t.Errorf("گروه شغلی پس از ویرایش %q است، انتظار %q بود", saved.JobGrade, alternateGrade)
	}

	// بازگرداندن مقدارهای اولیه تا دادهٔ نمونه برای اجراهای بعدی دست‌نخورده بماند.
	restore := edited
	restore.BaseSalary = original.BaseSalary
	restore.ResidenceSite = original.ResidenceSite
	restore.JobGrade = original.JobGrade
	if restore.JobGrade == "" {
		restore.JobGrade = employeedomain.ClearJobGrade
	}
	if back := call(t, h, http.MethodPatch, path, admin.AccessToken, restore); back.Code != http.StatusOK {
		t.Fatalf("بازگرداندن مقدارهای اولیه ناموفق بود (کد %d): %s", back.Code, smokeClip(back.Body.String()))
	}
}

// TestPostgresSmokeRecordsSurviveSecondInstance دو نمونهٔ سرور روی همان پایگاه‌داده
// بالا می‌آورد: رکوردی که با نمونهٔ اول ساخته می‌شود باید با نمونهٔ دوم خوانده شود.
//
// چرا لازم است: پُر بودن و چرخهٔ کامل را انبار درون‌حافظه‌ای هم می‌گذراند. تنها چیزی
// که ثابت می‌کند داده به‌راستی روی Postgres نشسته، خواندن همان رکورد از فرآیند/نمونهٔ
// دیگر روی همان DSN است.
func TestPostgresSmokeRecordsSurviveSecondInstance(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv(smokeDatabaseEnv))
	if dsn == "" {
		t.Skipf("آزمون دودی Postgres اجرا نشد: %s تنظیم نشده است", smokeDatabaseEnv)
	}
	first, _ := smokeNewHandler(t, dsn)
	smokeReady(t, first)
	firstAdmin := login(t, first, "admin", "demo")

	// یک نوع رکورد قطعی: «پروژهٔ مطالعاتی» در ماژول مطالعات منابع انسانی.
	const prefix = "/api/v1/hrstudies"
	list := call(t, first, http.MethodGet, prefix+"/records?kind=studies&size=1", firstAdmin.AccessToken, nil)
	if list.Code != http.StatusOK {
		t.Fatalf("فهرست پروژه‌های مطالعاتی کد %d داد: %s", list.Code, smokeClip(list.Body.String()))
	}
	var page contracts.RecordPage
	decodeInto(t, list, &page)
	if len(page.Items) == 0 {
		t.Fatal("پروژهٔ مطالعاتی نمونه‌ای برای الگوگیری نیست")
	}
	template := page.Items[0]

	stamp := fmt.Sprintf("SMOKE-INSTANCE-%d", time.Now().UnixNano())
	created := call(t, first, http.MethodPost, prefix+"/records", firstAdmin.AccessToken, contracts.RecordInput{
		Kind: "studies", Title: stamp, Fields: template.Fields, NoteFa: stamp,
	})
	if created.Code != http.StatusCreated {
		t.Fatalf("ایجاد رکورد با نمونهٔ اول ناموفق بود: %d %s", created.Code, smokeClip(created.Body.String()))
	}
	var record contracts.Record
	decodeInto(t, created, &record)

	// نمونهٔ دوم سرور روی همان پایگاه‌داده.
	second, _ := smokeNewHandler(t, dsn)
	secondAdmin := login(t, second, "admin", "demo")
	read := call(t, second, http.MethodGet, prefix+"/records/"+record.ID, secondAdmin.AccessToken, nil)
	if read.Code != http.StatusOK {
		t.Fatalf("نمونهٔ دوم سرور رکورد ساخته‌شدهٔ نمونهٔ اول را ندید (کد %d) — داده روی Postgres ننشسته است",
			read.Code)
	}
	// پاک‌سازی با همان نمونهٔ دوم (تا معلوم شود حذف هم روی پایگاه‌دادهٔ مشترک اثر دارد).
	del := call(t, second, http.MethodDelete, prefix+"/records/"+record.ID, secondAdmin.AccessToken, nil)
	if del.Code != http.StatusNoContent {
		t.Fatalf("نمونهٔ دوم نتوانست رکورد را حذف کند: کد %d", del.Code)
	}
	after := call(t, first, http.MethodGet, prefix+"/records/"+record.ID, firstAdmin.AccessToken, nil)
	if after.Code != http.StatusNotFound {
		t.Fatalf("رکورد پس از حذف از نمونهٔ دوم باید ۴۰۴ بدهد، کد %d داد", after.Code)
	}
	t.Logf("رکورد %s از نمونهٔ دوم سرور روی همان Postgres خوانده و پس از حذف، در نمونهٔ اول هم ۴۰۴ شد", record.ID)
}

/* --------------------------------- کمکی‌ها --------------------------------- */

// smokeOrphanKinds شناسهٔ رکوردهایی را برمی‌گرداند که نوعشان در شناسهٔ همین ماژول اعلام
// نشده است.
//
// چرا لازم است: جدول رکوردها مشترک است و کلید اصلی‌اش خودِ شناسه؛ اگر دو ماژول پیشوند
// شناسهٔ یکسان داشته باشند، سطر یکی با نام نوع ماژول دیگر ذخیره می‌شود. آن سطر در
// فهرست ماژول مالک دیده نمی‌شود ولی در انبار می‌ماند، پس فقط با یک پویش کامل پیدا
// می‌شود. این بند، همان دستهٔ ایراد پیشوند JOB را قابل سنجش می‌کند.
func smokeOrphanKinds(t *testing.T, h http.Handler, token, prefix string, declared map[string]bool) []string {
	t.Helper()
	orphans := []string{}
	// سقف بیست صفحه فقط برای بستن حلقه است؛ انبار نمونه بسیار کوچک‌تر از این است.
	for pageNo := 1; pageNo <= 20; pageNo++ {
		rec := call(t, h, http.MethodGet,
			fmt.Sprintf("%s/records?size=500&page=%d", prefix, pageNo), token, nil)
		if rec.Code != http.StatusOK {
			return orphans
		}
		var page contracts.RecordPage
		decodeInto(t, rec, &page)
		for _, item := range page.Items {
			if !declared[item.Kind] {
				orphans = append(orphans, item.ID+" («"+item.Kind+"»)")
			}
		}
		if len(page.Items) < 500 {
			break
		}
	}
	return orphans
}

// smokeRegistered آیا ماژول در این استقرار ثبت شده است.
func smokeRegistered(registered []string, id string) bool {
	for _, r := range registered {
		if r == id {
			return true
		}
	}
	return false
}

// smokeHasID آیا صفحهٔ رکوردها شناسهٔ خواسته‌شده را دارد.
func smokeHasID(page contracts.RecordPage, id string) bool {
	for _, item := range page.Items {
		if item.ID == id {
			return true
		}
	}
	return false
}

// smokeNonEmpty پاسخ یک مسیر فهرست را می‌سنجد: آرایهٔ ناتهی یا جمع بزرگ‌تر از صفر.
//
// معیار عمداً ساده است تا برای هر سه شکل پاسخ کار کند: آرایهٔ خام (کارت‌های داشبورد)،
// صفحهٔ رکورد ({items,total}) و پاسخ اختصاصی ({templates,total}).
func smokeNonEmpty(raw []byte) bool {
	var payload any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return false
	}
	return smokeHasContent(payload)
}

func smokeHasContent(v any) bool {
	switch value := v.(type) {
	case []any:
		return len(value) > 0
	case map[string]any:
		for key, item := range value {
			switch key {
			case "items", "rows", "widgets", "events", "templates", "sources", "records", "kinds":
				if arr, ok := item.([]any); ok && len(arr) > 0 {
					return true
				}
			case "total":
				if n, ok := item.(float64); ok && n > 0 {
					return true
				}
			}
		}
	}
	return false
}

// smokeClip پیام خطا را برای گزارش آزمون کوتاه می‌کند.
func smokeClip(s string) string {
	s = strings.TrimSpace(s)
	if len(s) <= 300 {
		return s
	}
	return s[:300] + "…"
}
