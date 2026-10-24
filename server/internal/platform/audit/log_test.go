// آزمون زیرساخت «audit» در پلتفرم سرور — «log_test».

package audit

import (
	"strings"
	"testing"
	"time"
)

func TestAppendChainsHashesAndVerifies(t *testing.T) {
	log := New()
	log.clock = func() time.Time { return time.Date(2026, 9, 28, 10, 0, 0, 0, time.UTC) }

	e1 := log.Append("hr.manager", "employee", "create", "EMP-013", "ایجاد پرونده")
	e2 := log.Append("hr.manager", "employee", "update", "EMP-013", "ویرایش پست")

	if e1.Seq != 1 || e2.Seq != 2 {
		t.Fatalf("شمارهٔ ترتیب رخدادها نادرست است: %d و %d", e1.Seq, e2.Seq)
	}
	if e1.PrevHash != genesis {
		t.Errorf("رخداد نخست باید به هش آغازین زنجیر شود")
	}
	if e2.PrevHash != e1.Hash {
		t.Errorf("هر رخداد باید هش رخداد پیشین را در بر بگیرد")
	}
	if e1.Hash == "" || e2.Hash == "" {
		t.Fatal("هر رخداد باید هش داشته باشد")
	}
	if err := log.Verify(); err != nil {
		t.Fatalf("زنجیرهٔ سالم باید تأیید شود: %v", err)
	}
	if log.Len() != 2 {
		t.Errorf("تعداد رخدادها باید ۲ باشد، %d بود", log.Len())
	}
}

func TestTamperedEventBreaksVerification(t *testing.T) {
	log := New()
	log.Append("admin", "settings", "update", "sysParams", "تغییر نرخ بیمه")
	log.Append("admin", "settings", "update", "sysParams", "تغییر جدول مالیات")

	// دست‌کاری تاریخی یک رخداد باید در بازبینی گرفته شود.
	log.events[0].Detail = "تغییر پنهانی نرخ بیمه به ۷٪"
	if err := log.Verify(); err == nil {
		t.Fatal("دست‌کاری رخداد باید زنجیره را نامعتبر کند")
	} else if !strings.Contains(err.Error(), "دست‌کاری") {
		t.Errorf("پیام خطا باید به دست‌کاری اشاره کند، دریافت شد: %v", err)
	}
}

func TestAppendOnlyNeverRewritesHistory(t *testing.T) {
	log := New()
	first := log.Append("hr.spec", "training", "create", "TR-01", "ثبت دوره")
	second := log.Append("hr.spec", "training", "update", "TR-01", "افزودن ظرفیت")

	if first == second {
		t.Fatal("رخداد تازه نباید رخداد قبلی را بازنویسی کند")
	}
	all := log.All()
	if len(all) != 2 || all[0].Detail != "ثبت دوره" {
		t.Errorf("تاریخ رخدادها باید دست‌نخورده بماند: %+v", all)
	}
}

// TestByTargetReturnsOnlyThatDocumentInOrder می‌سنجد که سابقهٔ تغییرات یک سند فقط از
// رخدادهای همان سند (و همان ماژول) و به ترتیب ثبت ساخته می‌شود.
func TestByTargetReturnsOnlyThatDocumentInOrder(t *testing.T) {
	log := New()
	log.Append("hr.manager", "salary_admin", "issue", "DEC-1", "صدور حکم")
	log.Append("hr.manager", "staff_services", "dependent-decide", "DEC-1", "سند هم‌نام در ماژول دیگر")
	log.Append("hr.manager", "salary_admin", "notify", "DEC-1", "ابلاغ حکم")
	log.Append("hr.manager", "salary_admin", "issue", "DEC-2", "صدور حکم دیگر")

	got := log.ByTarget("salary_admin", "DEC-1")
	if len(got) != 2 {
		t.Fatalf("باید ۲ رخداد همان سند بیاید، %d آمد: %+v", len(got), got)
	}
	if got[0].Action != "issue" || got[1].Action != "notify" {
		t.Fatalf("ترتیب ثبت رخدادها حفظ نشد: %+v", got)
	}
	for _, e := range got {
		if e.Module != "salary_admin" || e.Target != "DEC-1" {
			t.Fatalf("رخداد ماژول یا سند دیگری آمد: %+v", e)
		}
	}
	if got := log.ByTarget("salary_admin", ""); got != nil {
		t.Fatalf("سند نامعلوم باید سابقهٔ خالی بدهد: %+v", got)
	}
	if got := log.ByTarget("salary_admin", "DEC-404"); len(got) != 0 {
		t.Fatalf("سند بدون رخداد باید فهرست خالی بدهد: %+v", got)
	}
}

func TestActorAndModuleAreRecorded(t *testing.T) {
	log := New()
	log.Append("line.mgr", "attendance", "approve", "LV-201", "تأیید مرخصی")
	e := log.All()[0]
	if e.Actor != "line.mgr" || e.Module != "attendance" || e.Action != "approve" {
		t.Errorf("رخداد باید کاربر، ماژول و اقدام را ثبت کند: %+v", e)
	}
	if e.At == "" {
		t.Error("زمان رخداد باید ثبت شود")
	}
}
