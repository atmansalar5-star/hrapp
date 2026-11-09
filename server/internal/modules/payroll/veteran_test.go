package payroll

import (
	"context"
	"testing"

	"hrms/internal/contracts"
	"hrms/internal/platform/modulekit"
	"hrms/internal/rules"
)

// veteranPortFake پورت فقط‌خواندنی ساختگی ماژول امور ایثارگران برای آزمون میان‌بر.
type veteranPortFake struct{ items []contracts.Record }

func (f veteranPortFake) List(_ context.Context, filter contracts.RecordFilter) (contracts.RecordPage, error) {
	items := []contracts.Record{}
	for _, rec := range f.items {
		if filter.Kind == "" || rec.Kind == filter.Kind {
			items = append(items, rec)
		}
	}
	return contracts.RecordPage{Items: items, Total: len(items), Page: 1, Size: len(items)}, nil
}

func (f veteranPortFake) Schema() contracts.RecordSchema {
	return contracts.RecordSchema{Module: "grievances", ModuleTitleFa: "امور رفاهی و ایثارگران"}
}

// TestVeteranEntitlementAppliesApprovedFile میان‌بر این ماژول: پروندهٔ تأییدشده به فیش
// می‌رسد و پروندهٔ تأییدنشده هیچ اثری ندارد (آزمون کامل قاعده در بستهٔ modulekit است).
func TestVeteranEntitlementAppliesApprovedFile(t *testing.T) {
	rec := contracts.Record{
		ID: "VET-0001", Kind: "veteranRecords", EmployeeID: "EMP-001", Status: "approved",
		Fields: map[string]string{"veteranType": "جانباز", "percent": "25"},
	}
	ports := modulekit.NewPorts()
	ports.SetRecordList("grievances", veteranPortFake{items: []contracts.Record{rec}})
	deps := modulekit.Deps{Ports: ports}

	ent := veteranEntitlement(context.Background(), deps, "EMP-001")
	if ent == nil || !ent.Approved || ent.TaxExempt != 100 {
		t.Fatalf("امتیاز پروندهٔ تأییدشده باید به فیش برسد: %+v", ent)
	}
	withVeteran := rules.ComputePayroll(
		rules.PayrollInput{BaseSalary: 120_000_000, Veteran: ent}, rules.DefaultPayrollParams())
	if !withVeteran.VeteranApplied {
		t.Fatal("فیش باید امتیاز اعمال‌شده را نشان دهد")
	}

	rec.Status = "draft"
	ports = modulekit.NewPorts()
	ports.SetRecordList("grievances", veteranPortFake{items: []contracts.Record{rec}})
	if ent := veteranEntitlement(context.Background(), modulekit.Deps{Ports: ports}, "EMP-001"); ent != nil {
		t.Fatalf("پروندهٔ پیش‌نویس نباید اثر مالی داشته باشد: %+v", ent)
	}
}

// TestVeteranEntitlementsMapForWorkTime نقشهٔ کارکرد دوره از همان میان‌بر ساخته می‌شود.
func TestVeteranEntitlementsMapForWorkTime(t *testing.T) {
	ports := modulekit.NewPorts()
	ports.SetRecordList("grievances", veteranPortFake{items: []contracts.Record{{
		ID: "VET-0001", Kind: "veteranRecords", EmployeeID: "EMP-001", Status: "approved",
		Fields: map[string]string{"veteranType": "آزاده", "captivityMonths": "24"},
	}}})
	m := veteranEntitlements(context.Background(), modulekit.Deps{Ports: ports})
	if m["EMP-001"] == nil {
		t.Fatalf("پروندهٔ تأییدشدهٔ کارمند باید در نقشهٔ کارکرد باشد: %+v", m)
	}
}