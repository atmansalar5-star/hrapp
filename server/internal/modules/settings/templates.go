// پیاده‌سازی ماژول «settings» — «templates».

package settings

import (
	"context"

	"hrms/internal/contracts"
	"hrms/internal/modules/settings/domain"
	"hrms/internal/platform/recordkit"
)

// templateProvider قالب‌های متن اسناد را از رکوردهای «قالب متن اسناد» می‌خواند و به
// موتور اسناد (و ماژول قراردادها) می‌دهد. این‌گونه موتور اسناد به داخلی‌های ماژول
// تنظیمات وابسته نمی‌شود و فقط پورت قراردادی را می‌شناسد.
type templateProvider struct {
	svc *recordkit.Service
}

// Templates همهٔ قالب‌ها را برمی‌گرداند (فعال و غیرفعال).
func (p templateProvider) Templates(ctx context.Context) ([]contracts.DocumentTemplate, error) {
	if p.svc == nil {
		return nil, nil
	}
	page, err := p.svc.List(ctx, contracts.RecordFilter{Module: "settings", Kind: "docTemplates", Size: 500})
	if err != nil {
		return nil, err
	}
	out := make([]contracts.DocumentTemplate, 0, len(page.Items))
	for _, rec := range page.Items {
		docType := rec.Fields["docType"]
		out = append(out, contracts.DocumentTemplate{
			ID:        rec.ID,
			DocType:   docType,
			DocTypeFa: domain.DocTypeFa(docType),
			Title:     rec.Title,
			Header:    rec.Fields["header"],
			Body:      rec.Fields["body"],
			Footer:    rec.Fields["footer"],
			Status:    rec.Status,
			Version:   rec.Version,
		})
	}
	return out, nil
}

// ActiveTemplate قالب فعال یک نوع سند را برمی‌گرداند؛ نبود قالب فعال یعنی متن پیش‌فرض.
func (p templateProvider) ActiveTemplate(ctx context.Context, docType string) (contracts.DocumentTemplate, bool, error) {
	templates, err := p.Templates(ctx)
	if err != nil {
		return contracts.DocumentTemplate{}, false, err
	}
	for _, t := range templates {
		if t.DocType == docType && t.Status == "active" {
			return t, true, nil
		}
	}
	return contracts.DocumentTemplate{}, false, nil
}
