// آزمون جریان سرتاسری «workflow»: سیم‌کشی ماژول‌ها، سیاست‌ها و گردش‌کار.

package bootstrap

import (
	"net/http"
	"testing"

	"hrms/internal/contracts"
)

// TestWorkflowModulesGrantApprove زنجیرهٔ «اقدام گردش‌کار» را روی همهٔ ماژول‌های رکوردی
// می‌سنجد.
//
// چرا این آزمون لازم است: مسیر POST /records/{id}/action و کارتابل سراسری هر دو با
// مجوز «approve» در ماتریس مانیفست دروازه‌بندی می‌شوند. اگر مانیفست یک ماژول این مجوز
// را اعلام نکند، هیچ نقشی (حتی مدیر سیستم) نمی‌تواند وضعیت رکورد را عوض کند و کل
// برنامهٔ گردش‌کاری آن ماژول بی‌صدا از کار می‌افتد. این آزمون همان ناسازگاری را
// می‌گیرد: هر ماژولی که در شناسهٔ خود اقدام گردش‌کار اعلام کرده، باید در ماتریس
// مانیفست به نقش‌های راهبر «approve» داده باشد.
func TestWorkflowModulesGrantApprove(t *testing.T) {
	deps, err := NewDeps(Config{JWTSecret: []byte("test-secret-key-for-workflow-approve")})
	if err != nil {
		t.Fatalf("ساخت وابستگی‌ها ناموفق: %v", err)
	}
	handler, _, err := NewHandler(deps)
	if err != nil {
		t.Fatalf("ساخت هندلر ناموفق: %v", err)
	}
	admin := login(t, handler, "admin", "demo")

	checked := 0
	for _, m := range deps.Registry.All() {
		if m.APIPrefix == "" {
			continue
		}
		rec := call(t, handler, http.MethodGet, m.APIPrefix+"/schema", admin.AccessToken, nil)
		if rec.Code == http.StatusNotFound {
			// ماژول روی موتور رکوردهای گردش‌کاری سوار نیست؛ چیزی برای سنجش نیست.
			continue
		}
		if rec.Code != http.StatusOK {
			t.Fatalf("شناسهٔ ماژول %s باید ۲۰۰ بدهد، %d داد: %s", m.ID, rec.Code, rec.Body.String())
		}
		var schema contracts.RecordSchema
		decodeInto(t, rec, &schema)
		hasActions := false
		for _, k := range schema.Kinds {
			if len(k.Actions) > 0 {
				hasActions = true
				break
			}
		}
		if !hasActions {
			continue
		}
		checked++
		mod, ok := deps.Registry.ByID(m.ID)
		if !ok {
			t.Fatalf("ماژول %s در رجیستری یافت نشد", m.ID)
		}
		// مدیر سیستم و مدیر منابع انسانی در همهٔ ماژول‌های گردش‌کاری مجاز به اقدام‌اند؛
		// بدون این مجوز، دکمهٔ اقدام در رابط همیشه ۴۰۳ می‌شود.
		for _, role := range []string{"admin", "hr_manager"} {
			if !mod.Can(role, "approve") {
				t.Errorf("ماژول %s نقش %s را مجاز به اقدام گردش‌کار نکرده است؛ "+
					"بدون «approve» در مانیفست، دکمه‌های گردش‌کار ۴۰۳ می‌دهند", m.ID, role)
			}
		}
	}
	if checked == 0 {
		t.Fatal("هیچ ماژول گردش‌کاری با اقدام در شناسهٔ خود یافت نشد")
	}
}
