// آزمون پیاده‌سازی ماژول «reports» — «holding_source_test».

package reports

import (
	"context"
	"testing"

	"hrms/internal/contracts"
	"hrms/internal/platform/auth"
	"hrms/internal/platform/modulekit"
	"hrms/internal/platform/orgs"
)

func holdingTestDeps(orgID string) modulekit.Deps {
	o, _ := orgs.Lookup(orgID)
	return modulekit.Deps{Org: o, Ports: modulekit.NewPorts()}
}

func ctxWithRole(role contracts.Role) context.Context {
	return auth.WithPrincipal(context.Background(), auth.Principal{User: contracts.User{Username: "tester", Role: role}})
}

func TestHoldingHrMetadata(t *testing.T) {
	src := holdingHrSource{deps: holdingTestDeps("bandar_imam")}
	meta := src.Metadata()
	if meta.Key != holdingHrKey {
		t.Fatalf("کلید منبع %q بود، انتظار %q", meta.Key, holdingHrKey)
	}
	if len(meta.Fields) < 10 {
		t.Fatalf("اقلام گزارش هلدینگ کم است: %d", len(meta.Fields))
	}
	var holdingFilter *contracts.ReportFilterDef
	for i := range meta.Filters {
		if meta.Filters[i].Key == "holding" {
			holdingFilter = &meta.Filters[i]
		}
	}
	if holdingFilter == nil {
		t.Fatal("فیلتر هلدینگ اعلام نشده است")
	}
	// گزینهٔ «همهٔ هلدینگ‌ها» + یک گزینه برای هر هلدینگ شناخته‌شده.
	if want := len(orgs.Holdings()) + 1; len(holdingFilter.Options) != want {
		t.Fatalf("تعداد گزینه‌های فیلتر هلدینگ %d بود، انتظار %d", len(holdingFilter.Options), want)
	}
}

// کاربر غیرمدیر نباید بتواند با تغییر فیلتر از هلدینگ خودش فراتر برود.
func TestHoldingHrRowsRestrictToOwnHoldingForNonAdmin(t *testing.T) {
	src := holdingHrSource{deps: holdingTestDeps("bandar_imam")}
	own := len(orgs.CompaniesOfHolding(orgs.HoldingKey(mustOrg(t, "bandar_imam"))))
	res, err := src.Rows(ctxWithRole(contracts.RoleHRManager), contracts.ReportRequest{})
	if err != nil {
		t.Fatalf("اجرای گزارش خطا داد: %v", err)
	}
	if res.Total != own {
		t.Fatalf("تعداد سطرها %d بود، انتظار %d شرکت هلدینگ خود", res.Total, own)
	}
	if res.Total >= len(orgs.All) {
		t.Fatalf("کاربر غیرمدیر نباید شرکت‌های هلدینگ‌های دیگر را ببیند")
	}
	if len(res.Summary) == 0 || res.Summary[0].Total <= 0 {
		t.Fatalf("جمع کل کارکنان باید مثبت باشد: %+v", res.Summary)
	}
	// درخواست فیلتر «همهٔ هلدینگ‌ها» هم باید برای غیرمدیر نادیده گرفته شود.
	res2, err := src.Rows(ctxWithRole(contracts.RoleHRManager), contracts.ReportRequest{Filters: map[string]string{"holding": holdingAllKey}})
	if err != nil {
		t.Fatal(err)
	}
	if res2.Total != own {
		t.Fatalf("غیرمدیر با فیلتر همهٔ هلدینگ‌ها %d سطر گرفت، انتظار %d", res2.Total, own)
	}
}

// مدیر می‌تواند کل هلدینگ‌ها را ببیند و روی شاخص‌ها شرط بگذارد.
func TestHoldingHrAdminAllHoldingsAndConditions(t *testing.T) {
	src := holdingHrSource{deps: holdingTestDeps("bandar_imam")}
	all, err := src.Rows(ctxWithRole(contracts.RoleAdmin), contracts.ReportRequest{Filters: map[string]string{"holding": holdingAllKey}})
	if err != nil {
		t.Fatal(err)
	}
	if all.Total != len(orgs.All) {
		t.Fatalf("مدیر باید %d شرکت را ببیند، %d دید", len(orgs.All), all.Total)
	}

	// انتخاب اقلام: فقط دو قلم باید دو ستون بدهد.
	compact, err := src.Rows(ctxWithRole(contracts.RoleAdmin), contracts.ReportRequest{
		Items: []string{"company", "headcount"}, Filters: map[string]string{"holding": holdingAllKey},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(compact.Columns) != 2 || compact.Columns[0].Key != "company" || compact.Columns[1].Key != "headcount" {
		t.Fatalf("انتخاب اقلام اعمال نشد: %+v", compact.Columns)
	}

	// شرط «تعداد کل کارکنان > 1000»: هر سطر باید همین را برآورده کند.
	filtered, err := src.Rows(ctxWithRole(contracts.RoleAdmin), contracts.ReportRequest{
		Filters:    map[string]string{"holding": holdingAllKey},
		Conditions: []contracts.ReportCondition{{Field: "headcount", Op: "gt", Value: "1000"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if filtered.Total == 0 || filtered.Total >= all.Total {
		t.Fatalf("شرط headcount>1000 باید زیرمجموعه‌ای غیرخالی بدهد، %d از %d", filtered.Total, all.Total)
	}
}

// سطر شرکت فعال باید شاخص‌های رکوردی را داشته باشد حتی وقتی پورت کارکنان نیست (خالی).
func TestHoldingHrActiveCompanyFieldsWithoutPort(t *testing.T) {
	src := holdingHrSource{deps: holdingTestDeps("bandar_imam")}
	res, err := src.Rows(ctxWithRole(contracts.RoleAdmin), contracts.ReportRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Columns) == 0 || len(res.Rows) == 0 {
		t.Fatal("گزارش باید ستون و سطر داشته باشد")
	}
	for _, r := range res.Rows {
		if len(r) != len(res.Columns) {
			t.Fatalf("طول سطر %d با تعداد ستون %d نمی‌خواند", len(r), len(res.Columns))
		}
	}
}

func mustOrg(t *testing.T, id string) orgs.Organization {
	t.Helper()
	o, ok := orgs.Lookup(id)
	if !ok {
		t.Fatalf("شرکت نمونه %q پیدا نشد", id)
	}
	return o
}
