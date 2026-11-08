// آزمون هم‌خوانی فهرست مقاطع تحصیلی با موتور قواعد.
//
// چرا این آزمون: شرط مدرک در شناسنامهٔ شغل با کلیدهای موتور قواعد (`rules`) ذخیره می‌شود و
// پروندهٔ تحصیلی کارمند با کلیدهای همین بسته (`contracts`). اگر دو فهرست واگرا شوند، یک
// مقطع در یک جا شناخته و در جای دیگر ناشناخته می‌شود و سنجش شرایط احراز بی‌صدا خطا می‌دهد.
// تست در این بسته است چون `rules` نمی‌تواند `contracts` را وارد کند (وابستگی یک‌طرفه).

package contracts

import (
	"testing"

	"hrms/internal/rules"
)

// TestDegreeLadderMatchesContractsCatalog کلیدها، عنوان‌ها و ترتیب دو فهرست یکی باشد.
func TestDegreeLadderMatchesContractsCatalog(t *testing.T) {
	catalog := DegreeLevels()
	ladder := rules.DegreeLevels()

	if len(catalog) != len(ladder) {
		t.Fatalf("شمار مقاطع یکی نیست: contracts=%d rules=%d", len(catalog), len(ladder))
	}
	for i := range catalog {
		c, r := catalog[i], ladder[i]
		if c.Key != r.Key {
			t.Errorf("پلهٔ %d: کلید contracts=%q ولی rules=%q", i+1, c.Key, r.Key)
		}
		if c.TitleFa != r.TitleFa {
			t.Errorf("پلهٔ %d (%s): عنوان contracts=%q ولی rules=%q", i+1, c.Key, c.TitleFa, r.TitleFa)
		}
		if c.Order != r.Rank {
			t.Errorf("پلهٔ %d (%s): ترتیب contracts=%d ولی رتبهٔ rules=%d", i+1, c.Key, c.Order, r.Rank)
		}
		if got := rules.DegreeRankOf(c.Key); got != c.Order {
			t.Errorf("rules.DegreeRankOf(%q) = %d، انتظار %d", c.Key, got, c.Order)
		}
	}
}
