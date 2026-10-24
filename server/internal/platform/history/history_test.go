// آزمون زیرساخت «history» در پلتفرم سرور — «history_test».

package history

import (
	"testing"

	"hrms/internal/contracts"
	"hrms/internal/platform/audit"
)

// entry کمکی ساخت یک سطر سابقه در آزمون.
func entry(module, kind, id, date string) contracts.HistoryEntry {
	return contracts.HistoryEntry{Module: module, Kind: kind, RecordID: id, Date: date}
}

func TestSortDescendingWithEmptyDatesLast(t *testing.T) {
	items := []contracts.HistoryEntry{
		entry("contracts", "permanent", "CT-1", "1404/01/10"),
		entry("salary_admin", "decrees", "DEC-2", ""),
		entry("salary_admin", "decrees", "DEC-3", "1404/06/01"),
		entry("archive", "personnelFiles", "FILE-1", "1403/12/29"),
	}
	Sort(items)
	want := []string{"DEC-3", "CT-1", "FILE-1", "DEC-2"}
	for i, id := range want {
		if items[i].RecordID != id {
			t.Fatalf("ترتیب سوابق نادرست است: جای %d باید %s باشد، %s آمد (%v)", i, id, items[i].RecordID, ids(items))
		}
	}
}

func TestMarkLatestOnlyNewestPerKind(t *testing.T) {
	items := []contracts.HistoryEntry{
		entry("salary_admin", "decrees", "DEC-OLD", "1403/01/01"),
		entry("salary_admin", "decrees", "DEC-NEW", "1404/06/01"),
		entry("contracts", "temporary", "CT-1", "1404/02/01"),
	}
	Sort(items)
	MarkLatest(items)
	got := map[string]bool{}
	for _, e := range items {
		if e.Latest {
			got[e.RecordID] = true
		}
	}
	if len(got) != 2 || !got["DEC-NEW"] || !got["CT-1"] {
		t.Fatalf("نشانهٔ «آخرین سند» نادرست است: %v", got)
	}
	if got["DEC-OLD"] {
		t.Fatal("سند قدیمی نباید نشانهٔ آخرین سند بگیرد")
	}
}

func TestFilterByModuleAndKind(t *testing.T) {
	items := []contracts.HistoryEntry{
		entry("salary_admin", "decrees", "DEC-1", "1404/01/01"),
		entry("contracts", "permanent", "CT-1", "1404/01/02"),
		entry("contracts", "temporary", "CT-2", "1404/01/03"),
	}
	if out := Filter(items, "contracts", ""); len(out) != 2 {
		t.Fatalf("پالایش بر پایهٔ ماژول باید ۲ سطر بدهد، %d داد", len(out))
	}
	if out := Filter(items, "contracts", "permanent"); len(out) != 1 || out[0].RecordID != "CT-1" {
		t.Fatalf("پالایش بر پایهٔ نوع نادرست است: %v", out)
	}
	if out := Filter(items, "", ""); len(out) != len(items) {
		t.Fatalf("فیلتر خالی نباید چیزی حذف کند: %d", len(out))
	}
	if counts := Counts(items); counts["contracts"] != 2 || counts["salary_admin"] != 1 {
		t.Fatalf("شمارش به تفکیک ماژول نادرست است: %v", counts)
	}
}

// TestFromAuditBuildsChangeTrail می‌سنجد که رخدادهای حسابرسی یک سند به سطر «سابقهٔ
// تغییرات» تبدیل می‌شوند: زمان شمسی، کاربر، برچسب فارسی از ماژول مالک و شرح دست‌نخورده.
func TestFromAuditBuildsChangeTrail(t *testing.T) {
	events := []audit.Event{
		{At: "2026-04-22T09:00:00Z", Actor: "hr.manager", Action: "notify", Detail: "ابلاغ حکم DEC-1404-0213 — محمد احمدی"},
		{At: "2026-05-02T09:00:00Z", Actor: "system", Action: "نامعلوم", Detail: "رخداد بدون برچسب فارسی"},
	}
	labels := map[string]string{"notify": "ابلاغ حکم"}
	changes := FromAudit(events, func(action string) string { return labels[action] })
	if len(changes) != 2 {
		t.Fatalf("باید ۲ سطر سابقهٔ تغییرات بسازد، %d ساخت", len(changes))
	}
	if changes[0].ActionFa != "ابلاغ حکم" {
		t.Errorf("برچسب فارسی اقدام از ماژول مالک نیامد: %q", changes[0].ActionFa)
	}
	if changes[0].Actor != "hr.manager" || changes[0].Action != "notify" {
		t.Errorf("کاربر یا کلید اقدام رخداد نادرست است: %+v", changes[0])
	}
	if changes[0].At == "" {
		t.Errorf("زمان رخداد باید به تاریخ شمسی تبدیل شود: %+v", changes[0])
	}
	if changes[0].NoteFa != "ابلاغ حکم DEC-1404-0213 — محمد احمدی" {
		t.Errorf("شرح رخداد باید دست‌نخورده بماند: %q", changes[0].NoteFa)
	}
	if changes[1].ActionFa != "" {
		t.Errorf("کلید ناشناخته نباید برچسب بسازد: %q", changes[1].ActionFa)
	}
	if out := FromAudit(nil, func(string) string { return "x" }); out != nil {
		t.Fatalf("سند بدون رخداد باید سابقهٔ تغییرات نداشته باشد: %+v", out)
	}
}

func ids(items []contracts.HistoryEntry) []string {
	out := make([]string, 0, len(items))
	for _, e := range items {
		out = append(out, e.RecordID+"@"+e.Date)
	}
	return out
}
