// پیاده‌سازی ماژول «settings» — «catalogs».

package settings

import (
	"context"
	"strings"

	"hrms/internal/contracts"
	"hrms/internal/platform/recordkit"
)

// catalogProvider پورت «کاتالوگ‌های مرجع» را روی رکوردهای ماژول تنظیمات پیاده می‌کند.
//
// چرا اینجا: مرجع مدیریت‌ها، بانک‌ها و واحدها یک‌جا در تنظیمات نگه‌داری می‌شود و ماژول‌های
// وام و پاداش آن را از راه همین پورت می‌خوانند. پس هیچ ماژولی داخلی‌های ماژول دیگر را
// import نمی‌کند و افزودن یک مدیرت یا بانک، همه‌جا اثر می‌گذارد.
type catalogProvider struct{ svc *recordkit.Service }

// Catalog فهرست اقلام یک کاتالوگ را برمی‌گرداند.
//
// قاعدهٔ تبدیل: کلید ماشین‌خوان از فیلد `key`، عنوان فارسی از عنوان رکورد، و «فعال بودن»
// از وضعیت رکورد. وضعیت `inactive` یعنی قلم از فرم‌ها و اعتبارسنجی بیرون می‌رود؛ رکورد
// حذف نمی‌شود تا دادهٔ قدیمی که به آن اشاره دارد از بین نرود.
func (c catalogProvider) Catalog(ctx context.Context, kind string) ([]contracts.CatalogItem, error) {
	if !validKind(kind) {
		return []contracts.CatalogItem{}, nil
	}
	page, err := c.svc.List(ctx, contracts.RecordFilter{
		Module: "settings", Kind: kind, Scope: "all", Page: 1, Size: 1000,
	})
	if err != nil {
		return nil, err
	}
	out := make([]contracts.CatalogItem, 0, len(page.Items))
	for _, rec := range page.Items {
		key := strings.TrimSpace(rec.Fields["key"])
		if key == "" {
			continue
		}
		out = append(out, contracts.CatalogItem{
			Key:       key,
			TitleFa:   strings.TrimSpace(rec.Title),
			NoteFa:    strings.TrimSpace(rec.NoteFa),
			ParentKey: strings.TrimSpace(rec.Fields["management"]),
			Active:    rec.Status != "inactive",
			RecordID:  rec.ID,
		})
	}
	return out, nil
}

// validKind کاتالوگ‌های شناخته‌شده. نگه‌داشتن این فهرست به‌جای خواندن از دامنه، از
// حلقهٔ import بین بستهٔ settings و domain جلوگیری می‌کند.
func validKind(kind string) bool {
	switch kind {
	case contracts.CatalogManagements, contracts.CatalogBanks, contracts.CatalogUnits:
		return true
	}
	return false
}
