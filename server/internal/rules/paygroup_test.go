// آزمون گروه‌های نظام پرداخت نفت (جدول ۳) و روش‌های پرداخت مصوب.
// ارجاع: docs/pay-classification-spec-fa.md (بندهای PAY-01 … PAY-03).

package rules

import (
	"strings"
	"testing"
)

// TestPAY01OilPayGroupsRenumbered می‌سنجد «کارمند رسمی نفت» گروه مستقل یکم شده و گروه یکم
// پیشین (پایهٔ سمت ۶ تا ۱۰) به گروه دوم رفته و بقیه یک پله جابه‌جا شده‌اند.
func TestPAY01OilPayGroupsRenumbered(t *testing.T) {
	groups := OilPayGroups()
	if len(groups) != 10 {
		t.Fatalf("ده گروه انتظار می‌رفت، %d آمد", len(groups))
	}
	first, ok := OilPayGroupByNumber(1)
	if !ok || !first.IsOfficialOil() {
		t.Fatalf("گروه یکم باید گروه مستقل کارمند رسمی نفت باشد: %+v", first)
	}
	if len(first.Bases) != 0 {
		t.Fatal("گروه کارمند رسمی نفت نباید پایهٔ سمت داشته باشد (گروه مستقل است)")
	}
	if first.PublicAttract != 4 || first.PositionRank != 8 {
		t.Fatalf("درصد پیش‌فرض گروه یکم باید ۴٪ + ۸٪ باشد: %+v", first)
	}

	second, ok := OilPayGroupByNumber(2)
	if !ok {
		t.Fatal("گروه دوم پیدا نشد")
	}
	if strings.Join(second.Bases, ",") != "6-10" {
		t.Fatalf("گروه دوم باید پایهٔ سمت ۶ تا ۱۰ باشد (گروه یکم پیشین): %+v", second)
	}
	if !strings.Contains(second.NoteFa, "پیش از این") {
		t.Fatal("یادداشت گروه دوم باید جابه‌جایی شماره را توضیح دهد")
	}

	// ترتیب پایه‌های سمت از سوم به بعد همان نردبان جدول ۳ است.
	want := []string{"11-15", "16", "17", "A", "B", "C", "D", "OD"}
	for i, base := range want {
		group, ok := OilPayGroupByNumber(i + 3)
		if !ok || strings.Join(group.Bases, ",") != base {
			t.Fatalf("گروه %d باید پایهٔ سمت %s باشد: %+v", i+3, base, group)
		}
	}

	// پایهٔ سمت به گروه درست نگاشت می‌شود و رسمی نفت گروه یکم می‌گیرد.
	if group, ok := OilPayGroupOfBase("6-10"); !ok || group.Number != 2 {
		t.Fatalf("پایهٔ سمت ۶ تا ۱۰ باید گروه دوم باشد: %+v", group)
	}
	if group, ok := OilPayGroupOfBase("OD"); !ok || group.Number != 10 {
		t.Fatalf("پایهٔ سمت OD باید گروه دهم باشد: %+v", group)
	}
	if group, ok := OilPayGroupOfEmployment("official_oil"); !ok || group.Number != 1 {
		t.Fatalf("کارمند رسمی نفت باید گروه یکم باشد: %+v", group)
	}
	if group, ok := OilPayGroupOfEmployment("کارمند رسمی نفت"); !ok || group.Number != 1 {
		t.Fatal("عنوان فارسی کارمند رسمی نفت هم باید گروه یکم بدهد")
	}
	if _, ok := OilPayGroupOfEmployment("contractor_staff"); ok {
		t.Fatal("نوع استخدام غیررسمی نباید گروه رسمی نفت بگیرد")
	}
}

// TestPAY02GroupResolution می‌سنجد گروه نهایی از نوع استخدام، پایهٔ سمت و گروه شغلی درست
// تعیین می‌شود و پارامتر شرکت می‌تواند درصدهای هر گروه را جدا بازنویسی کند.
func TestPAY02GroupResolution(t *testing.T) {
	params := DefaultOilPayParams()

	// نوع استخدام رسمی نفت بر پایهٔ سمت مقدم است: رسمی نفت با پایهٔ سمت ۱۱ تا ۱۵ هم
	// گروه یکم می‌ماند، چون گروه مستقل خودش را دارد.
	if group, ok := OilPayGroupFor("official_oil", "11-15"); !ok || group.Number != 1 {
		t.Fatalf("رسمی نفت باید گروه یکم بماند: %+v", group)
	}
	if group, ok := OilPayGroupFor("fixed_1662", "11-15"); !ok || group.Number != 3 {
		t.Fatalf("قرارداد مدت معین با پایهٔ سمت ۱۱ تا ۱۵ باید گروه سوم باشد: %+v", group)
	}
	// از راه گروه شغلی سازمان (نگاشت GradeSiteMap): کارشناس ارشد ⇒ پایهٔ سمت ۱۶ ⇒ گروه چهارم.
	if n := OilPayGroupForJobGradeNumber(params, "fixed_1662", "senior_expert"); n != 4 {
		t.Fatalf("کارشناسی ارشد باید گروه چهارم باشد، %d آمد", n)
	}
	if n := OilPayGroupForJobGradeNumber(params, "official_oil", "employee"); n != 1 {
		t.Fatalf("کارمند رسمی نفت باید گروه یکم باشد، %d آمد", n)
	}

	// پارامتر شرکت: درصدهای گروه یکم (رسمی نفت) جدا از گروه دوم بازنویسی می‌شود.
	override := DefaultOilPayParams()
	for i := range override.Groups {
		if override.Groups[i].Number == 1 {
			override.Groups[i].PublicAttract = 6
			override.Groups[i].PositionRank = 12
		}
	}
	group, ok := OilPayGroupFor("official_oil", "")
	if !ok || group.Number != 1 {
		t.Fatalf("گروه رسمی نفت پیدا نشد: %+v", group)
	}
	oilParams, _, _ := oilEffectiveGroup(OilPayInput{EmploymentType: "official_oil"}, override)
	if oilParams.PublicAttract != 6 || oilParams.PositionRank != 12 {
		t.Fatalf("درصدهای بازنویسی‌شدهٔ گروه یکم اعمال نشد: %+v", oilParams)
	}
	twoParams, _, _ := oilEffectiveGroup(OilPayInput{JobGrade: "6-10"}, override)
	if twoParams.PublicAttract != 4 {
		t.Fatalf("بازنویسی گروه یکم نباید گروه دوم را عوض کند: %+v", twoParams)
	}
}

// TestPAY03PayModelsCatalogAndRules می‌سنجد چهار روش پرداخت مصوب سرجایشان‌اند و سازگاری
// نوع استخدام با روش پرداخت درست سنجیده می‌شود.
func TestPAY03PayModelsCatalogAndRules(t *testing.T) {
	keys := PayModelKeys()
	if len(keys) != 4 {
		t.Fatalf("چهار روش پرداخت انتظار می‌رفت، %d آمد: %v", len(keys), keys)
	}
	for _, want := range []string{"payroll", "oil", "comp34006", "fourp"} {
		if !ValidPayModel(want) {
			t.Fatalf("روش پرداخت «%s» باید معتبر باشد", want)
		}
		if PayModelTitleFa(want) == "" {
			t.Fatalf("روش پرداخت «%s» باید عنوان فارسی داشته باشد", want)
		}
	}
	if PayModelOf("چیز-نامعتبر") != PayModelPayroll {
		t.Fatal("روش ناشناخته باید به روش پیش‌فرض سازمان برگردد")
	}
	if PayModelDefaultFor("official_oil") != PayModelOil {
		t.Fatal("پیش‌فرض کارمند رسمی نفت باید نظام صندوق نفت باشد")
	}
	if PayModelDefaultFor("fixed_1662") != PayModelPayroll {
		t.Fatal("پیش‌فرض بقیه انواع استخدام باید حقوق و دستمزد جاری باشد")
	}

	// سازگاری: رسمی نفت فقط با صندوق نفت، و صندوق نفت فقط برای رسمی نفت.
	if ok, _ := PayModelAllowed("official_oil", "payroll"); ok {
		t.Fatal("رسمی نفت با حقوق و دستمزد جاری نباید مجاز باشد")
	}
	if ok, reason := PayModelAllowed("official_oil", "oil"); !ok || reason != "" {
		t.Fatalf("رسمی نفت با صندوق نفت باید مجاز باشد: %s", reason)
	}
	if ok, _ := PayModelAllowed("contractor_staff", "oil"); ok {
		t.Fatal("صندوق نفت برای پیمانکار نباید مجاز باشد")
	}
	// دو مدل توسعه‌ای اختیاری‌اند و با انتخاب صریح پذیرفته می‌شوند.
	if ok, reason := PayModelAllowed("fixed_1662", "comp34006"); !ok {
		t.Fatalf("مدل ۳۴۰۰۶ باید برای مدت معین مجاز باشد: %s", reason)
	}
	if ok, reason := PayModelAllowed("long_term", "fourp"); !ok {
		t.Fatalf("مدل ۴پی باید برای قراردادی طویل‌المدت مجاز باشد: %s", reason)
	}
}
