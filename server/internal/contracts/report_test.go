package contracts

import "testing"

// TestSplitValues تقسیم مقدار فیلترهای چندانتخابی را می‌گیرد.
//
// قرارداد این است که رابط مقدارهای تیک‌خورده را با کاما بفرستد؛ پس هم کامای لاتین و
// هم کامای فارسی/عربی و نقطه‌ویرگول باید پذیرفته شوند و فاصله‌های اضافی دور مقدارها
// نباید در تطبیق اثر بگذارد.
func TestSplitValues(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want []string
	}{
		{"خالی", "", nil},
		{"یک مقدار", "official_worker", []string{"official_worker"}},
		{"دو مقدار با کاما", "official_worker,fixed_1662", []string{"official_worker", "fixed_1662"}},
		{"فاصله و کامای فارسی", " official_worker ، fixed_1662 ", []string{"official_worker", "fixed_1662"}},
		{"نقطه‌ویرگول و مقدار خالی", "official_oil;;long_term", []string{"official_oil", "long_term"}},
		{"فقط جداکننده‌ها", " , ", nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := SplitValues(c.in)
			if len(got) != len(c.want) {
				t.Fatalf("SplitValues(%q) = %#v، انتظار %#v", c.in, got, c.want)
			}
			for i := range c.want {
				if got[i] != c.want[i] {
					t.Fatalf("SplitValues(%q) = %#v، انتظار %#v", c.in, got, c.want)
				}
			}
		})
	}
}

// TestMatchValue قاعدهٔ مشترک فیلتر تک‌مقداری/چندانتخابی را می‌گیرد: فهرست تیک‌خورده بر
// فیلتر تک‌مقداری مقدم است، مقدار باید در همان فهرست باشد و خالی بودن هر دو یعنی «همه».
func TestMatchValue(t *testing.T) {
	t.Run("بدون فیلتر همه می‌گذرند", func(t *testing.T) {
		if !MatchValue("", nil, "official_oil") {
			t.Fatal("خالی بودن فیلتر باید همهٔ مقدارها را بپذیرد")
		}
	})
	t.Run("فیلتر تک‌مقداری فقط همان را می‌پذیرد", func(t *testing.T) {
		if !MatchValue("official_oil", nil, "official_oil") {
			t.Fatal("مقدار برابر باید پذیرفته شود")
		}
		if MatchValue("official_oil", nil, "fixed_1662") {
			t.Fatal("مقدار نابرابر نباید پذیرفته شود")
		}
	})
	t.Run("فهرست تیک‌خورده هر یک از مقدارها را می‌پذیرد", func(t *testing.T) {
		selected := []string{"official_worker", "fixed_1662"}
		if !MatchValue("", selected, "official_worker") || !MatchValue("", selected, "fixed_1662") {
			t.Fatal("هر یک از مقدارهای تیک‌خورده باید پذیرفته شود")
		}
		if MatchValue("", selected, "official_oil") {
			t.Fatal("مقدار بیرون از فهرست تیک‌خورده نباید پذیرفته شود")
		}
	})
	t.Run("فهرست تیک‌خورده بر فیلتر تک‌مقداری مقدم است", func(t *testing.T) {
		if MatchValue("official_oil", []string{"official_worker"}, "official_oil") {
			t.Fatal("وقتی فهرست تیک‌خورده پر است، فیلتر تک‌مقداری نباید اثر کند")
		}
	})
	t.Run("فاصلهٔ اضافی در مقدار فیلتر اثر ندارد", func(t *testing.T) {
		if !MatchValue("", []string{" official_worker "}, "official_worker") {
			t.Fatal("فاصلهٔ اضافی در مقدار فیلتر نباید تطبیق را بشکند")
		}
	})
}

// TestEmployeeStatusCatalog یکپارچگی کاتالوگ وضعیت کارمند را می‌سنجد؛ این کاتالوگ
// مرجع مشترک فهرست API، اعتبارسنجی دامنه و گزینه‌های فیلتر گزارش است.
func TestEmployeeStatusCatalog(t *testing.T) {
	keys := EmployeeStatusKeys()
	if len(keys) != len(EmployeeStatusCatalog) {
		t.Fatalf("تعداد کلیدها با کاتالوگ نمی‌خواند: %d != %d", len(keys), len(EmployeeStatusCatalog))
	}
	seen := map[string]bool{}
	for _, s := range EmployeeStatusCatalog {
		if s.Key == "" || s.TitleFa == "" {
			t.Fatalf("وضعیت بی‌کلید یا بی‌عنوان در کاتالوگ: %+v", s)
		}
		if seen[s.Key] {
			t.Fatalf("کلید تکراری در کاتالوگ وضعیت: %s", s.Key)
		}
		seen[s.Key] = true
		if got := EmployeeStatusTitleFa(s.Key); got != s.TitleFa {
			t.Fatalf("عنوان وضعیت «%s» نادرست برگشت: %q", s.Key, got)
		}
	}
	if got := EmployeeStatusTitleFa("نامعلوم"); got != "نامعلوم" {
		t.Fatalf("وضعیت ناشناخته باید خودِ ورودی برگردد، %q برگشت", got)
	}
}
