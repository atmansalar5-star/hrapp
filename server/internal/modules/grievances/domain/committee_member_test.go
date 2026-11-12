package domain

import (
	"strings"
	"testing"

	"hrms/internal/contracts"
	"hrms/internal/rules"
)

// memberRecord ورودی تعریف عضو کمیته می‌سازد.
func memberRecord(fields map[string]string) contracts.RecordInput {
	return contracts.RecordInput{Kind: KindVeteranCommitteeMember, Title: "عضو آزمایشی", Fields: fields}
}

// TestVeteranCommitteeMemberKindIsDefined نوع رکورد عضو در فهرست انواع ماژول است.
func TestVeteranCommitteeMemberKindIsDefined(t *testing.T) {
	schema := Descriptor()
	var found bool
	for _, kind := range schema.Kinds {
		if kind.Key != KindVeteranCommitteeMember {
			continue
		}
		found = true
		if kind.IDPrefix != "VCMEM" {
			t.Fatalf("پیش‌شمارهٔ شناسه باید VCMEM باشد، بود: %s", kind.IDPrefix)
		}
		if kind.EmployeeScoped {
			t.Fatal("عضو کمیته نباید کارمندمحور باشد؛ نمایندهٔ بنیاد شهید هم عضو است")
		}
		keys := map[string]bool{}
		for _, field := range kind.Fields {
			keys[field.Key] = true
		}
		for _, want := range []string{"fullName", "role", "positionTitle", "orderNo", "orderDate", "signatureTitle", "sortOrder"} {
			if !keys[want] {
				t.Fatalf("فیلد %s در فرم عضو کمیته نیست", want)
			}
		}
	}
	if !found {
		t.Fatal("نوع «عضو کمیتهٔ ایثارگری» در فهرست انواع ماژول نیست")
	}
}

// TestVeteranCommitteeMemberValid عضو کامل باید پذیرفته شود.
func TestVeteranCommitteeMemberValid(t *testing.T) {
	in := memberRecord(map[string]string{
		"fullName":      "سارا محمدی",
		"role":          rules.VeteranCommitteeChairman,
		"positionTitle": "مدیر منابع انسانی",
		"orderNo":       "۱۴۰۵/ح/۱۰۲",
		"orderDate":     "1405/01/05",
		"sortOrder":     "1",
	})
	if err := validateVeteranCommitteeMember(in); err != nil {
		t.Fatalf("عضو کامل نباید خطا بدهد: %v", err)
	}
}

// TestVeteranCommitteeMemberValidationRules عضو ناقص یا بیرون بازه باید رد شود.
func TestVeteranCommitteeMemberValidationRules(t *testing.T) {
	cases := []struct {
		name   string
		fields map[string]string
		want   string
	}{
		{"بدون نام", map[string]string{"role": rules.VeteranCommitteeMember}, "نام و نام خانوادگی"},
		{"نقش نامعلوم", map[string]string{"fullName": "الف", "role": "king"}, "نقش عضو"},
		{"نقش خالی", map[string]string{"fullName": "الف"}, "نقش عضو"},
		{"تاریخ ابلاغ نامعتبر", map[string]string{"fullName": "الف", "role": rules.VeteranCommitteeMember, "orderDate": "1405/13/40"}, "تاریخ ابلاغ"},
		{"ترتیب امضا منفی", map[string]string{"fullName": "الف", "role": rules.VeteranCommitteeMember, "sortOrder": "-2"}, "ترتیب امضا"},
		{"ترتیب امضا بزرگ", map[string]string{"fullName": "الف", "role": rules.VeteranCommitteeMember, "sortOrder": "150"}, "ترتیب امضا"},
		{"کد پرسنلی بلند", map[string]string{"fullName": "الف", "role": rules.VeteranCommitteeMember,
			"employeeId": "EMP-123456789012345678901234"}, "کد پرسنلی"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateVeteranCommitteeMember(memberRecord(tc.fields))
			if err == nil {
				t.Fatalf("انتظار خطا بود، ولی پذیرفته شد")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("پیام خطا باید شامل «%s» باشد، بود: %s", tc.want, err.Error())
			}
		})
	}
}

// TestCommitteeMemberSeedCoversKeyRoles بذر اعضا باید نقش‌های کلیدی کمیته را داشته باشد.
// TestCommitteeMemberDeactivateNeedsReason اقدام «غیرفعالکردن عضویت» ثبت دلیل را الزامی
// میداند؛ پس هر مصرفکنندهای (پنل اعضا یا تغییر وضعیت از فرم) باید دلیل بفرستد، وگرنه
// تغییر وضعیت بیصدا رد میشود و عضو فعال میماند.
func TestCommitteeMemberDeactivateNeedsReason(t *testing.T) {
	for _, kind := range Descriptor().Kinds {
		if kind.Key != KindVeteranCommitteeMember {
			continue
		}
		for _, action := range kind.Actions {
			if action.Value != "deactivate" {
				continue
			}
			if !action.NoteRequired {
				t.Fatal("غیرفعالکردن عضویت باید دلیل الزامی داشته باشد")
			}
			if !kind.CanTransition("active", action) {
				t.Fatal("غیرفعالکردن عضو از وضعیت «عضو فعال» باید مجاز باشد")
			}
			return
		}
		t.Fatal("اقدام «غیرفعالکردن عضویت» برای عضو کمیته تعریف نشده است")
	}
	t.Fatal("نوع «عضو کمیتهٔ ایثارگری» در فهرست انواع ماژول نیست")
}

func TestCommitteeMemberSeedCoversKeyRoles(t *testing.T) {
	roles := map[string]bool{}
	count := 0
	for _, item := range Seed() {
		if item.Kind != KindVeteranCommitteeMember {
			continue
		}
		count++
		roles[item.Fields["role"]] = true
	}
	if count < 4 {
		t.Fatalf("دست‌کم چهار عضو کمیته برای نمونه لازم است، بود: %d", count)
	}
	for _, want := range []string{
		rules.VeteranCommitteeChairman, rules.VeteranCommitteeSecretary, rules.VeteranCommitteeVeteranRep,
	} {
		if !roles[want] {
			t.Fatalf("نقش %s در بذر اعضای کمیته نیست", want)
		}
	}
}
