// آزمون جریان سرتاسری «record_prefix_gate_test»: سیم‌کشی ماژول‌ها، سیاست‌ها و گردش‌کار.

package bootstrap

import (
	"net/http"
	"strings"
	"testing"

	"hrms/internal/contracts"
	platformmodules "hrms/internal/platform/modules"
)

// TestRecordKindIDPrefixesAreUniqueAcrossModules دروازه‌ای برای یک‌تایی پیشوند شناسهٔ
// نوع رکوردها در کل سامانه است — بدون نیاز به پایگاه‌داده، پس در آزمون‌های همیشگی CI
// هم اجرا می‌شود.
//
// چرا حیاتی است: همهٔ ماژول‌های رکوردی جدول مشترک `hr_records.record` را به کار می‌برند
// و کلید اصلی آن خودِ شناسه است. اگر دو ماژول یک پیشوند شناسه داشته باشند، دو نوع رکورد
// با شناسهٔ یکسان ساخته می‌شود و ذخیره‌سازی روی همان سطر می‌نویسد؛ یعنی رکورد یک ماژول
// بی‌صدا رکورد ماژول دیگر را بازنویسی می‌کند. این ایراد فقط روی Postgres دیده می‌شود
// (در حافظه هر ماژول انبار جدا دارد) و همین باعث شد آزمون دودی Postgres آن را پیدا کند:
// «آگهی شغلی» استخدام و «گروه شغلی» طبقه‌بندی مشاغل هر دو پیشوند JOB داشتند.
func TestRecordKindIDPrefixesAreUniqueAcrossModules(t *testing.T) {
	h := newTestServer(t)
	admin := login(t, h, "admin", "demo")

	owner := map[string]string{}
	kinds := 0
	for _, m := range platformmodules.All {
		rec := call(t, h, http.MethodGet, m.APIPrefix+"/schema", admin.AccessToken, nil)
		if rec.Code != http.StatusOK {
			continue // ماژول غیررکوردی؛ پیشوند شناسه ندارد.
		}
		var schema contracts.RecordSchema
		decodeInto(t, rec, &schema)
		for _, kind := range schema.Kinds {
			kinds++
			here := m.ID + "/" + kind.Key
			prefix := strings.TrimSpace(kind.IDPrefix)
			if prefix == "" {
				t.Errorf("%s: پیشوند شناسه خالی است؛ رکوردهای این نوع شناسهٔ بی‌پیشوند می‌گیرند", here)
				continue
			}
			if prev, ok := owner[prefix]; ok {
				t.Errorf("پیشوند شناسهٔ «%s» هم به %s و هم به %s می‌رسد؛ "+
					"چون جدول رکوردها مشترک است، این دو نوع رکورد روی هم می‌نویسند", prefix, prev, here)
				continue
			}
			owner[prefix] = here
		}
	}
	if kinds == 0 {
		t.Fatal("هیچ نوع رکوردی برای بررسی پیشوند شناسه یافت نشد")
	}
	t.Logf("یک‌تایی پیشوند شناسه در %d نوع رکورد تأیید شد (%d پیشوند متمایز)", kinds, len(owner))
}

// TestRecordKindKeysMatchTheirModuleIDPrefixes یک بند دیگر همان قرارداد را می‌سنجد:
// شناسهٔ رکورد تازه باید با پیشوند اعلامی همان نوع ساخته شود. اگر این هم‌خوانی بشکند،
// جستجو و ارجاع بین ماژول‌ها بر پایهٔ شناسه ناپایدار می‌شود. اینجا فقط شکل قاعده با
// مقایسهٔ دادهٔ نمونه (که از همان مسیر Create گذشته) بررسی می‌شود.
func TestSeedRecordsCarryTheirKindIDPrefix(t *testing.T) {
	h := newTestServer(t)
	admin := login(t, h, "admin", "demo")

	checked := 0
	for _, m := range platformmodules.All {
		rec := call(t, h, http.MethodGet, m.APIPrefix+"/schema", admin.AccessToken, nil)
		if rec.Code != http.StatusOK {
			continue
		}
		var schema contracts.RecordSchema
		decodeInto(t, rec, &schema)
		for _, kind := range schema.Kinds {
			list := call(t, h, http.MethodGet,
				m.APIPrefix+"/records?kind="+kind.Key+"&size=5", admin.AccessToken, nil)
			if list.Code != http.StatusOK {
				continue
			}
			var page contracts.RecordPage
			decodeInto(t, list, &page)
			for _, item := range page.Items {
				checked++
				if !strings.HasPrefix(item.ID, kind.IDPrefix+"-") {
					t.Errorf("%s/%s: شناسهٔ «%s» با پیشوند اعلامی «%s» هم‌خوان نیست",
						m.ID, kind.Key, item.ID, kind.IDPrefix)
				}
			}
		}
	}
	if checked == 0 {
		t.Fatal("رکوردی برای بررسی هم‌خوانی شناسه یافت نشد")
	}
	t.Logf("هم‌خوانی شناسه با پیشوند نوع در %d رکورد نمونه تأیید شد", checked)
}
