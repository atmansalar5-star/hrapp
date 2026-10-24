// آزمون دامنهٔ ماژول «settings» (catalog_test).

package domain

import (
	"errors"
	"testing"

	"hrms/internal/contracts"
	"hrms/internal/platform/recordkit"
)

// TestValidateCatalogItem قاعدهٔ قلم کاتالوگ مرجع را می‌گیرد: کلید باید ماشین‌خوان
// (لاتین و بی‌فاصله) باشد چون در داده و API ذخیره می‌شود، و عنوان فارسی داشته باشد چون
// در فرم‌های وام و پاداش نمایش داده می‌شود.
func TestValidateCatalogItem(t *testing.T) {
	cases := []struct {
		name    string
		in      contracts.RecordInput
		wantErr bool
	}{
		{
			name: "قلم درست",
			in:   contracts.RecordInput{Title: "بانک تازه", Fields: map[string]string{"key": "new_bank"}},
		},
		{
			name:    "کلید خالی",
			in:      contracts.RecordInput{Title: "بانک تازه", Fields: map[string]string{}},
			wantErr: true,
		},
		{
			name:    "کلید فارسی یا با فاصله",
			in:      contracts.RecordInput{Title: "بانک تازه", Fields: map[string]string{"key": "بانک تازه"}},
			wantErr: true,
		},
		{
			name:    "عنوان فارسی خالی",
			in:      contracts.RecordInput{Title: "   ", Fields: map[string]string{"key": "new_bank"}},
			wantErr: true,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := validateCatalogItem(c.in)
			if c.wantErr {
				var rule *recordkit.RuleError
				if !errors.As(err, &rule) {
					t.Fatalf("انتظار خطای قاعده بود، اما %v برگشت", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("قلم درست رد شد: %v", err)
			}
		})
	}
}

// TestCatalogKindsHelpers کمکی‌های شناسایی کاتالوگ‌ها را می‌گیرد؛ ماژول وام و پاداش بر
// پایهٔ همین‌ها فهرست‌هایشان را می‌خوانند.
func TestCatalogKindsHelpers(t *testing.T) {
	for _, k := range []string{contracts.CatalogManagements, contracts.CatalogBanks, contracts.CatalogUnits} {
		if !ValidCatalogKind(k) {
			t.Fatalf("کاتالوگ شناخته‌شده «%s» نامعتبر تشخیص داده شد", k)
		}
		if CatalogKindFa(k) == k || CatalogKindFa(k) == "" {
			t.Fatalf("عنوان فارسی کاتالوگ «%s» ساخته نشد", k)
		}
	}
	if ValidCatalogKind("unknownKind") {
		t.Fatal("کاتالوگ ناشناخته نباید معتبر باشد")
	}
}

// TestCatalogKindsEditableInEveryStatus کاتالوگ مرجع «دادهٔ پایه» است، نه سند گردش‌کاری؛
// پس ویرایش قلم در وضعیت فعال هم باید مجاز باشد. این آزمون جلوی برگشت آن ایراد را می‌گیرد.
func TestCatalogKindsEditableInEveryStatus(t *testing.T) {
	desc := Descriptor()
	for _, key := range CatalogKinds {
		kind, ok := desc.KindByKey(key)
		if !ok {
			t.Fatalf("نوع رکورد کاتالوگ «%s» در شناسهٔ ماژول تنظیمات نیست", key)
		}
		for _, st := range []string{"draft", "active", "inactive"} {
			if !kind.Editable(st) {
				t.Fatalf("ویرایش قلم کاتالوگ «%s» در وضعیت «%s» مجاز نیست", key, st)
			}
		}
		if kind.Validate == nil {
			t.Fatalf("نوع رکورد «%s» قاعدهٔ اعتبارسنجی ندارد", key)
		}
	}
}

// TestCatalogSeedIntegrity دادهٔ نمونهٔ کاتالوگ‌ها را می‌گیرد: همهٔ اقلام فعال‌اند (تا از
// همان ابتدا در فرم‌ها دیده شوند)، کلیدها یگانه‌اند، و مدیریت بالادست هر واحد واقعاً در
// کاتالوگ مدیریت‌ها وجود دارد — وگرنه واحد بی‌والد در فرم‌ها گم می‌شود.
func TestCatalogSeedIntegrity(t *testing.T) {
	counts := map[string]int{}
	keys := map[string]map[string]bool{}
	mgmtKeys := map[string]bool{}
	for _, in := range Seed() {
		if !ValidCatalogKind(in.Kind) {
			continue
		}
		if in.SeedStatus != "active" {
			t.Fatalf("قلم کاتالوگ «%s» با وضعیت «%s» کاشته شد؛ باید فعال باشد", in.Kind, in.SeedStatus)
		}
		key := in.Fields["key"]
		if key == "" {
			t.Fatalf("قلم کاتالوگ «%s» کلید ندارد", in.Kind)
		}
		if keys[in.Kind] == nil {
			keys[in.Kind] = map[string]bool{}
		}
		if keys[in.Kind][key] {
			t.Fatalf("کلید تکراری «%s» در کاتالوگ «%s»", key, in.Kind)
		}
		keys[in.Kind][key] = true
		counts[in.Kind]++
		if in.Kind == contracts.CatalogManagements {
			mgmtKeys[key] = true
		}
	}
	for _, k := range CatalogKinds {
		if counts[k] == 0 {
			t.Fatalf("کاتالوگ «%s» هیچ قلم نمونه‌ای ندارد", k)
		}
	}
	for _, in := range Seed() {
		if in.Kind != contracts.CatalogUnits {
			continue
		}
		if parent := in.Fields["management"]; !mgmtKeys[parent] {
			t.Fatalf("مدیریت بالادست «%s» واحد «%s» در کاتالوگ مدیریت‌ها نیست", parent, in.Fields["key"])
		}
	}
}
