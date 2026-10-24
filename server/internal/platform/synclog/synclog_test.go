// آزمون زیرساخت «synclog» در پلتفرم سرور — «synclog_test».

package synclog

import (
	"testing"

	"hrms/internal/contracts"
)

func mutation(opID, op, id string, baseVersion int) contracts.Mutation {
	return contracts.Mutation{
		OpID:        opID,
		Entity:      "Employee",
		Op:          op,
		ID:          id,
		BaseVersion: baseVersion,
		Payload:     map[string]any{"fullName": "سارا محمدی"},
	}
}

func TestRecordAndDeltaSinceCursor(t *testing.T) {
	log := New()
	log.Record("Employee", OpUpsert, "EMP-001", 1, map[string]any{"fullName": "سارا محمدی"})
	log.Record("Employee", OpUpsert, "EMP-002", 1, map[string]any{"fullName": "آرش رضایی"})
	log.Record("Employee", OpUpsert, "EMP-001", 2, map[string]any{"position": "مدیر"})

	all := log.Since(0)
	if len(all.Changes) != 3 || all.NextSeq != 3 {
		t.Fatalf("دلتای کامل باید ۳ تغییر باشد، دریافت شد %d (nextSeq=%d)", len(all.Changes), all.NextSeq)
	}

	delta := log.Since(1)
	if len(delta.Changes) != 2 {
		t.Fatalf("دلتا از نشانگر ۱ باید ۲ تغییر باشد، %d بود", len(delta.Changes))
	}
	if delta.Changes[0].ID != "EMP-002" {
		t.Errorf("نخستین تغییر دلتا باید EMP-002 باشد، %s بود", delta.Changes[0].ID)
	}

	if version, ok := log.Version("Employee", "EMP-001"); !ok || version != 2 {
		t.Errorf("نسخهٔ فعلی EMP-001 باید ۲ باشد، %d بود", version)
	}
	if empty := log.Since(99); len(empty.Changes) != 0 {
		t.Errorf("دلتا از نشانگر جلوتر باید خالی باشد، %d تغییر داشت", len(empty.Changes))
	}
}

func TestApplyIsIdempotentForRepeatedOpID(t *testing.T) {
	log := New()
	first := log.Apply([]contracts.Mutation{mutation("op-1", "create", "EMP-050", 0)})
	if len(first.Accepted) != 1 || len(first.Rejected) != 0 {
		t.Fatalf("نخستین اعمال باید پذیرفته شود: %+v", first)
	}
	second := log.Apply([]contracts.Mutation{mutation("op-1", "create", "EMP-050", 0)})
	if len(second.Accepted) != 1 || len(second.Rejected) != 0 {
		t.Fatalf("تکرار همان opId باید بی‌اثر و پذیرفته باشد: %+v", second)
	}
	if log.Count() != 1 {
		t.Errorf("تکرار عملیات نباید رخداد تازه بسازد؛ تعداد=%d", log.Count())
	}
}

func TestCreateOnExistingRecordConflicts(t *testing.T) {
	log := New()
	log.Record("Employee", OpUpsert, "EMP-001", 3, nil)

	result := log.Apply([]contracts.Mutation{mutation("op-2", "create", "EMP-001", 0)})
	if len(result.Rejected) != 1 {
		t.Fatalf("ساخت رکورد موجود باید رد شود: %+v", result)
	}
	if result.Rejected[0].Reason != contracts.CodeConflict || result.Rejected[0].ServerVersion != 3 {
		t.Errorf("دلیل و نسخهٔ سرور نادرست است: %+v", result.Rejected[0])
	}
}

func TestStaleBaseVersionIsRejectedSoLocalEditsDoNotClobber(t *testing.T) {
	log := New()
	log.Record("Employee", OpUpsert, "EMP-001", 1, nil)
	log.Record("Employee", OpUpsert, "EMP-001", 2, nil) // سرور جلوتر رفته است

	result := log.Apply([]contracts.Mutation{mutation("op-3", "update", "EMP-001", 1)})
	if len(result.Rejected) != 1 {
		t.Fatalf("ویرایش با نسخهٔ کهنه باید رد شود: %+v", result)
	}
	if result.Rejected[0].Reason != contracts.CodeConflict || result.Rejected[0].ServerVersion != 2 {
		t.Errorf("پاسخ تعارض باید نسخهٔ سرور را برگرداند: %+v", result.Rejected[0])
	}

	// ویرایش با نسخهٔ درست پذیرفته می‌شود و نسخه یک پله جلو می‌رود.
	ok := log.Apply([]contracts.Mutation{mutation("op-4", "update", "EMP-001", 2)})
	if len(ok.Accepted) != 1 {
		t.Fatalf("ویرایش با نسخهٔ درست باید پذیرفته شود: %+v", ok)
	}
	if version, _ := log.Version("Employee", "EMP-001"); version != 3 {
		t.Errorf("نسخه پس از ویرایش باید ۳ باشد، %d بود", version)
	}
}

func TestUpdateAndDeleteOfMissingRecordAreRejected(t *testing.T) {
	log := New()
	upd := log.Apply([]contracts.Mutation{mutation("op-5", "update", "EMP-999", 1)})
	if len(upd.Rejected) != 1 || upd.Rejected[0].Reason != contracts.CodeNotFound {
		t.Errorf("ویرایش رکورد ناموجود باید ۴۰۴ منطقی بدهد: %+v", upd)
	}

	del := log.Apply([]contracts.Mutation{mutation("op-6", "delete", "EMP-999", 0)})
	if len(del.Rejected) != 1 || del.Rejected[0].Reason != contracts.CodeNotFound {
		t.Errorf("حذف رکورد ناموجود باید رد شود: %+v", del)
	}
}

func TestDeletePublishesTombstone(t *testing.T) {
	log := New()
	log.Record("Employee", OpUpsert, "EMP-002", 1, map[string]any{"fullName": "آرش رضایی"})

	result := log.Apply([]contracts.Mutation{mutation("op-7", "delete", "EMP-002", 1)})
	if len(result.Accepted) != 1 {
		t.Fatalf("حذف باید پذیرفته شود: %+v", result)
	}
	delta := log.Since(1)
	if len(delta.Changes) != 1 || delta.Changes[0].Op != OpDelete || delta.Changes[0].Version != 2 {
		t.Fatalf("حذف باید به‌صورت tombstone با نسخهٔ ۲ منتشر شود: %+v", delta.Changes)
	}
}

func TestStructuralAndUnknownOperationsAreRejected(t *testing.T) {
	log := New()
	result := log.Apply([]contracts.Mutation{
		{OpID: "", Entity: "Employee", Op: "update", ID: "EMP-001"},
		{OpID: "op-8", Entity: "Employee", Op: "archive", ID: "EMP-001"},
	})
	if len(result.Rejected) != 2 {
		t.Fatalf("هر دو عملیات نامعتبر باید رد شوند: %+v", result)
	}
	for _, r := range result.Rejected {
		if r.Message == "" {
			t.Error("هر رد باید پیام فارسی داشته باشد تا در UI نمایش داده شود")
		}
	}
	if log.Count() != 0 {
		t.Errorf("عملیات نامعتبر نباید تغییر ثبت کند؛ تعداد=%d", log.Count())
	}
}

func TestMixedBatchReportsBothOutcomes(t *testing.T) {
	log := New()
	log.Record("Employee", OpUpsert, "EMP-001", 1, nil)

	result := log.Apply([]contracts.Mutation{
		mutation("ok-1", "update", "EMP-001", 1),
		mutation("bad-1", "update", "EMP-404", 1),
	})
	if len(result.Accepted) != 1 || result.Accepted[0] != "ok-1" {
		t.Errorf("عملیات معتبر باید پذیرفته شود: %+v", result.Accepted)
	}
	if len(result.Rejected) != 1 || result.Rejected[0].OpID != "bad-1" {
		t.Errorf("عملیات ناموجود باید رد شود: %+v", result.Rejected)
	}
	if result.NextSeq != log.Seq() {
		t.Errorf("nextSeq باید با نشانگر دفتر هم‌خوان باشد: %d != %d", result.NextSeq, log.Seq())
	}
}
