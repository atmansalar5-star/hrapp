// آزمون زیرساخت «recordkit» در پلتفرم سرور — «workflow_test».

package recordkit

import (
	"testing"

	"hrms/internal/contracts"
)

// leaveKind یک گردش‌کار دومرحله‌ای نمونه: مدیر واحد و سپس منابع انسانی.
func leaveKind() Kind {
	return Kind{
		Key: "leaveRequests", TitleFa: "درخواست مرخصی", IDPrefix: "LV", EmployeeScoped: true,
		Statuses: Statuses(
			"draft|پیش‌نویس", "pending|در انتظار مدیر واحد",
			"unitApproved|تأیید مدیر واحد", "approved|تأیید نهایی منابع انسانی", "rejected|ردشده",
		),
		Initial: "draft",
		Transitions: Steps(
			"draft|pending", "pending|unitApproved,rejected",
			"unitApproved|approved,rejected",
		),
		Actions: Actions(
			"submit|ارسال برای تأیید|primary|pending|0|employee,hr_spec",
			"endorse|تأیید مدیر واحد|primary|unitApproved|0|line_manager",
			"approve|تأیید نهایی|primary|approved|0|hr_manager",
			"reject|رد درخواست|danger|rejected|1|line_manager,hr_manager",
		),
	}
}

func TestWorkflowStepsChain(t *testing.T) {
	steps := leaveKind().WorkflowSteps()
	if len(steps) != 3 {
		t.Fatalf("باید ۳ مرحلهٔ تصمیم باشد، %d شد: %+v", len(steps), steps)
	}
	if steps[0].Status != "draft" || steps[1].Status != "pending" || steps[2].Status != "unitApproved" {
		t.Fatalf("ترتیب مرحله‌ها نادرست است: %+v", steps)
	}
	// مرحلهٔ «در انتظار مدیر واحد» نقش مدیر واحد و (برای رد) منابع انسانی را دارد.
	want := map[string]bool{"line_manager": true, "hr_manager": true}
	if len(steps[1].Roles) != 2 {
		t.Fatalf("نقش‌های مرحلهٔ pending نادرست است: %+v", steps[1].Roles)
	}
	for _, r := range steps[1].Roles {
		if !want[r] {
			t.Fatalf("نقش غیرمنتظره در مرحلهٔ pending: %s", r)
		}
	}
	// مرحلهٔ تأیید نهایی فقط منابع انسانی است.
	if len(steps[2].Roles) != 2 { // approve(hr_manager) + reject(hr_manager,line_manager)
		t.Fatalf("نقش‌های مرحلهٔ unitApproved نادرست است: %+v", steps[2].Roles)
	}
}

func TestCanActEnforcesStepRole(t *testing.T) {
	k := leaveKind()
	lm := Actor{Username: "line.mgr", Role: contracts.RoleLineManager}
	hr := Actor{Username: "hr.manager", Role: contracts.RoleHRManager}
	admin := Actor{Username: "admin", Role: contracts.RoleAdmin}

	endorse, _ := k.ActionByValue("endorse")
	approve, _ := k.ActionByValue("approve")

	// مدیر واحد در وضعیت pending می‌تواند تأیید کند، ولی منابع انسانی نمی‌تواند از این مرحله رد شود.
	if !k.CanAct("pending", endorse, lm, "", "") {
		t.Fatal("مدیر واحد باید بتواند در مرحلهٔ pending اقدام endorse را انجام دهد")
	}
	if k.CanAct("pending", approve, hr, "", "") {
		t.Fatal("منابع انسانی نباید بتواند مرحلهٔ مدیر واحد را دور بزند")
	}
	// پس از تأیید مدیر واحد، منابع انسانی نهایی می‌کند و مدیر واحد دیگر مجاز نیست.
	if !k.CanAct("unitApproved", approve, hr, "", "") {
		t.Fatal("منابع انسانی باید بتواند در مرحلهٔ unitApproved نهایی کند")
	}
	if k.CanAct("unitApproved", approve, lm, "", "") {
		t.Fatal("مدیر واحد نباید بتواند خودش تأیید نهایی کند")
	}
	// مدیر سیستم از هر مرحله مجاز است تا قفلِ گردش‌کار رخ ندهد.
	if !k.CanAct("pending", endorse, admin, "", "") || !k.CanAct("unitApproved", approve, admin, "", "") {
		t.Fatal("مدیر سیستم باید در همهٔ مرحله‌ها مجاز باشد")
	}
	// کنشگر ناشناخته (فراخوانی داخلی/بذرکاری) محدود نمی‌شود.
	if !k.CanAct("pending", endorse, Actor{}, "", "") {
		t.Fatal("کنشگر بدون هویت نباید محدود شود")
	}
}

func TestAvailableActions(t *testing.T) {
	k := leaveKind()
	lm := Actor{Username: "line.mgr", Role: contracts.RoleLineManager}
	hr := Actor{Username: "hr.manager", Role: contracts.RoleHRManager}

	pending := k.AvailableActions("pending", lm, "", "")
	if len(pending) != 2 { // endorse و reject
		t.Fatalf("مدیر واحد در pending باید ۲ اقدام مجاز داشته باشد: %+v", pending)
	}
	if got := k.AvailableActions("pending", hr, "", ""); len(got) != 1 || got[0].Value != "reject" {
		t.Fatalf("منابع انسانی در pending فقط مجاز به رد است: %+v", got)
	}
	if got := k.AvailableActions("approved", hr, "", ""); len(got) != 0 {
		t.Fatalf("در وضعیت نهایی نباید اقدامی مجاز باشد: %+v", got)
	}
}

func TestActionScopeOwn(t *testing.T) {
	k := Kind{
		Key: "selfService", Statuses: Statuses("draft|پیش‌نویس", "confirmed|تأییدشده"),
		Initial: "draft", Transitions: Steps("draft|confirmed"),
		Actions: Actions("confirm|تأیید|primary|confirmed|0|employee|own"),
	}
	confirm, _ := k.ActionByValue("confirm")
	owner := Actor{Username: "e1", Role: contracts.RoleEmployee, EmployeeID: "EMP-001"}
	other := Actor{Username: "e2", Role: contracts.RoleEmployee, EmployeeID: "EMP-002"}

	if !k.CanAct("draft", confirm, owner, "EMP-001", "") {
		t.Fatal("کارمند باید رکورد خودش را تأیید کند")
	}
	if k.CanAct("draft", confirm, other, "EMP-001", "") {
		t.Fatal("کارمند نباید رکورد کارمند دیگر را تأیید کند")
	}
}

func TestRolesParsedAndValidated(t *testing.T) {
	a := Actions("x|اقدام|primary|done|0|hr_manager,ghost_role,line_manager")[0]
	if len(a.Roles) != 2 {
		t.Fatalf("نقش نامعتبر باید حذف شود: %+v", a.Roles)
	}
	if a.Roles[0] != contracts.RoleHRManager || a.Roles[1] != contracts.RoleLineManager {
		t.Fatalf("نقش‌ها نادرست تجزیه شدند: %+v", a.Roles)
	}
}
