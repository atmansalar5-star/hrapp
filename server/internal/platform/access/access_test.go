// آزمون زیرساخت «access» در پلتفرم سرور — «access_test».

package access

import (
	"testing"
)

func TestScopeMatchesCompanyAndHolding(t *testing.T) {
	cases := []struct {
		name                     string
		grantOrg, grantHolding   string
		activeOrg, activeHolding string
		want                     bool
	}{
		{"جهانی در هر دامنه", "", "", "bandar_imam", "Persian Gulf Holding", true},
		{"شرکت‌محور در همان شرکت", "arvand", "", "arvand", "Persian Gulf Holding", true},
		{"شرکت‌محور در شرکت دیگر", "arvand", "", "maroun", "Maroun Petrochemical Holding", false},
		{"هلدینگ‌محور در شرکت همان هلدینگ", "", "Persian Gulf Holding", "nouri", "Persian Gulf Holding", true},
		{"هلدینگ‌محور در هلدینگ دیگر", "", "Persian Gulf Holding", "maroun", "Maroun Petrochemical Holding", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := ScopeMatches(c.grantOrg, c.grantHolding, c.activeOrg, c.activeHolding); got != c.want {
				t.Fatalf("ScopeMatches=%v، انتظار %v", got, c.want)
			}
		})
	}
}

func TestCompanyGrantNotAppliedInOtherCompany(t *testing.T) {
	s := NewStore()
	s.Upsert(Grant{Role: "hr_spec", ModuleID: "reports", OrganizationID: "arvand", Actions: []string{"read"}})
	if !s.Allows("hr_spec", "", "reports", "read", "arvand", "Persian Gulf Holding") {
		t.Fatal("سطح شرکت‌محور باید در همان شرکت اعمال شود")
	}
	if s.Allows("hr_spec", "", "reports", "read", "maroun", "Maroun Petrochemical Holding") {
		t.Fatal("سطح شرکت‌محور نباید در شرکت دیگر اعمال شود")
	}
	s.Upsert(Grant{Role: "hr_manager", ModuleID: "reports", HoldingFa: "Persian Gulf Holding", Actions: []string{"read"}})
	if !s.Allows("hr_manager", "", "reports", "read", "nouri", "Persian Gulf Holding") {
		t.Fatal("سطح هلدینگ‌محور باید در شرکت‌های همان هلدینگ اعمال شود")
	}
	if s.Allows("hr_manager", "", "reports", "read", "maroun", "Maroun Petrochemical Holding") {
		t.Fatal("سطح هلدینگ‌محور نباید در هلدینگ دیگر اعمال شود")
	}
}

func TestGrantMatchesDepartment(t *testing.T) {
	s := NewStore()
	// دسترسی ویژهٔ ادارهٔ D2 برای کارشناس منابع انسانی.
	s.Upsert(Grant{Role: "hr_spec", ModuleID: "payroll", DepartmentID: "D2", Actions: []string{"read", "write"}})
	// دسترسی سراسری برای مدیر واحد.
	s.Upsert(Grant{Role: "line_manager", ModuleID: "attendance", Actions: []string{"read"}})

	cases := []struct {
		role, dept, module, action string
		want                       bool
	}{
		{"hr_spec", "D2", "payroll", "write", true},
		{"hr_spec", "D1", "payroll", "write", false},  // ادارهٔ دیگر
		{"hr_spec", "D2", "payroll", "delete", false}, // اقدام دیگر
		{"line_manager", "D1", "attendance", "read", true},
		{"line_manager", "D9", "attendance", "read", true}, // سراسری
		{"employee", "D2", "payroll", "read", false},       // نقش دیگر
	}
	for _, tc := range cases {
		if got := s.Allows(tc.role, tc.dept, tc.module, tc.action, "", ""); got != tc.want {
			t.Fatalf("Allows(%s,%s,%s,%s) = %v، انتظار %v", tc.role, tc.dept, tc.module, tc.action, got, tc.want)
		}
	}
}

func TestGrantUpsertAndDelete(t *testing.T) {
	s := NewStore()
	a := s.Upsert(Grant{Role: "employee", ModuleID: "reports", Actions: []string{"read"}})
	if a.ID == "" {
		t.Fatal("شناسهٔ سطح دسترسی ساخته نشد")
	}
	// همان ترکیب به‌روز می‌شود، نه اینکه سطح تازه بسازد.
	b := s.Upsert(Grant{Role: "employee", ModuleID: "reports", Actions: []string{"read", "write"}})
	if b.ID != a.ID {
		t.Fatalf("به‌روزرسانی سطح دسترسی شناسه را عوض کرد: %s → %s", a.ID, b.ID)
	}
	if len(s.Grants()) != 1 {
		t.Fatalf("تعداد سطوح باید ۱ باشد: %d", len(s.Grants()))
	}
	if !s.Delete(b.ID) {
		t.Fatal("حذف سطح دسترسی ناموفق بود")
	}
	if s.Delete(b.ID) {
		t.Fatal("حذف دوباره باید ناموفق باشد")
	}
	if len(s.Grants()) != 0 {
		t.Fatalf("انبار باید خالی شود: %d", len(s.Grants()))
	}
}

func TestGrantWildcardsAndModules(t *testing.T) {
	s := NewStore()
	s.Upsert(Grant{Role: "admin", ModuleID: AnyModule, Actions: []string{AnyAction}})
	if !s.Allows("admin", "D1", "anything", "whatever", "", "") {
		t.Fatal("دسترسی گسترده باید همه‌چیز را بدهد")
	}
	// ماژول گسترده در فهرست ماژول‌های منو نمی‌آید (چون ماژول مشخصی نیست).
	if mods := s.GrantedModules("admin", "D1", "", ""); len(mods) != 0 {
		t.Fatalf("ماژول گسترده نباید در منو بیاید: %v", mods)
	}

	s.Upsert(Grant{Role: "hr_spec", ModuleID: "reports", Actions: []string{"read"}})
	mods := s.GrantedModules("hr_spec", "D1", "", "")
	if len(mods) != 1 || mods[0] != "reports" {
		t.Fatalf("ماژول‌های افزوده نادرست است: %v", mods)
	}
	acts := s.ActionsFor("hr_spec", "D1", "reports", "", "")
	if len(acts) != 1 || acts[0] != "read" {
		t.Fatalf("اقدام‌های افزوده نادرست است: %v", acts)
	}
}
