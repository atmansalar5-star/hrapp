// پیاده‌سازی پورت «شناسنامهٔ شغل» برای مصرف ماژول‌های دیگر.
//
// چرا پورت و نه import مستقیم: تصویب‌خواهی انتصاب و ارتقا در ماژول career زندگی می‌کند و
// مرز ماژول اجازهٔ import نمی‌دهد. این آداپتور همان دادهٔ شناسنامه را از راه قرارداد
// مشترک `contracts.JobRegistryReader` در اختیار می‌گذارد؛ پس شرط شغل یک منبع دارد
// (شرح شغل تأییدشده) و مصرف‌کننده هیچ کلید فیلدی را از خودش نمی‌سازد.
package jobclass

import (
	"context"
	"strings"

	"hrms/internal/contracts"
	"hrms/internal/modules/jobclass/domain"
	"hrms/internal/platform/modulekit"
	"hrms/internal/platform/recordkit"
	"hrms/internal/rules"
)

// registry آداپتور پورت شناسنامهٔ شغل روی سرویس رکوردهای همین ماژول.
type registry struct{ svc *recordkit.Service }

// registerJobRegistry پورت شناسنامهٔ شغل را در مجموعهٔ پورت‌ها ثبت می‌کند.
func registerJobRegistry(deps modulekit.Deps, svc *recordkit.Service) {
	deps.Ports.SetJobRegistry(registry{svc: svc})
}

// Jobs فهرست شناسنامه‌های شغل سازمان.
func (r registry) Jobs(ctx context.Context) ([]contracts.JobRequirementView, error) {
	page, err := r.svc.List(ctx, contracts.RecordFilter{
		Module: "jobclass", Kind: domain.KindDescriptions, Scope: "all", Size: 300,
	})
	if err != nil {
		return nil, err
	}
	out := make([]contracts.JobRequirementView, 0, len(page.Items))
	for _, rec := range page.Items {
		out = append(out, requirementView(rec))
	}
	return out, nil
}

// Requirement شرایط احراز یک شغل با کد یا عنوان شغل.
func (r registry) Requirement(ctx context.Context, codeOrTitle string) (contracts.JobRequirementView, bool, error) {
	page, err := r.svc.List(ctx, contracts.RecordFilter{
		Module: "jobclass", Kind: domain.KindDescriptions, Scope: "all", Size: 300,
	})
	if err != nil {
		return contracts.JobRequirementView{}, false, err
	}
	rec, found := domain.FindDescription(page.Items, codeOrTitle)
	if !found {
		return contracts.JobRequirementView{}, false, nil
	}
	return requirementView(rec), true, nil
}

// requirementView یک رکورد شرح شغل را به شکل قراردادی شرایط احراز تبدیل می‌کند.
func requirementView(rec contracts.Record) contracts.JobRequirementView {
	req := domain.Requirement(rec.Fields)
	groupFa := strings.TrimSpace(rec.Fields[domain.FieldJobGroup])
	if rank := rules.GradeRankOf(groupFa); rank > 0 {
		groupFa = rules.GradeTitleFa(rank)
	}
	return contracts.JobRequirementView{
		RecordID:    rec.ID,
		JobCode:     strings.TrimSpace(rec.Fields[domain.FieldJobCode]),
		TitleFa:     rec.Title,
		UnitFa:      strings.TrimSpace(rec.Fields[domain.FieldUnit]),
		GroupFa:     groupFa,
		Approved:    rec.Status == domain.StatusApproved || rec.Status == domain.StatusRevised,
		StatusFa:    rec.StatusFa,
		Requirement: req,
		DegreeFa:    rules.DegreeTitleFa(req.DegreeLevel),
	}
}
