// آزمون دامنهٔ ماژول «orgstructure» (codes_test). آزمون خودکار؛ هر تغییر رفتار پیاده‌سازی باید آزمون
// همین پرونده را به‌روز کند.

package domain

import (
	"strings"
	"testing"

	"hrms/internal/contracts"
	"hrms/internal/rules"
)

// testScheme طرح پیش‌شماره‌های پیش‌فرض آزمون.
func testScheme() rules.OrgCodeScheme { return rules.DefaultOrgCodeScheme() }

// TestPositionValidateEnforcesOrgCodeFormat اعتبارسنجی کدهای پست هنگام ثبت/ویرایش.
func TestPositionValidateEnforcesOrgCodeFormat(t *testing.T) {
	validate := positionValidateWith(testScheme())

	ok := contracts.RecordInput{Kind: KindPositions, Title: "پست آزمایشی", Fields: map[string]string{
		FieldPostCode:      "080.0000000001",
		FieldOrgGradeCode:  "081.0000000003",
		FieldUnitCode:      "083.0000000007",
		FieldJobFamilyCode: "085.0000000012",
	}}
	if err := validate(ok); err != nil {
		t.Fatalf("کدهای معتبر نباید خطا بدهند: %v", err)
	}

	if err := validate(contracts.RecordInput{Kind: KindPositions, Fields: map[string]string{}}); err != nil {
		t.Fatalf("نبود کد نباید خطا بدهد: %v", err)
	}

	badPost := contracts.RecordInput{Kind: KindPositions, Fields: map[string]string{
		FieldPostCode: "POS-201",
	}}
	if err := validate(badPost); err == nil {
		t.Fatal("شمارهٔ سمت با قالب دیگر باید رد شود")
	}

	// جابه‌جایی پیش‌شماره‌ها هم خطاست: پایهٔ سازمانی با پیش‌شمارهٔ شمارهٔ سمت.
	swapped := contracts.RecordInput{Kind: KindPositions, Fields: map[string]string{
		FieldOrgGradeCode: "080.0000000003",
	}}
	if err := validate(swapped); err == nil {
		t.Fatal("پایهٔ سازمانی با پیش‌شمارهٔ اشتباه باید رد شود")
	}
}

// TestPositionValidateFollowsScheme پیش‌شمارهٔ متغیر: با عوض‌کردن پارامتر شرکت، اعتبارسنجی
// کد ساخته‌شده با پیش‌شمارهٔ تازه را می‌پذیرد و کد پیش‌فرض را رد می‌کند.
func TestPositionValidateFollowsScheme(t *testing.T) {
	custom := rules.OrgCodeSchemeFromValues(map[string]string{"orgcode.prefix.post": "201"})
	validate := positionValidateWith(custom)

	fresh := contracts.RecordInput{Kind: KindPositions, Fields: map[string]string{
		FieldPostCode: "201.0000000005",
	}}
	if err := validate(fresh); err != nil {
		t.Fatalf("کد با پیش‌شمارهٔ تازه باید پذیرفته شود: %v", err)
	}
	old := contracts.RecordInput{Kind: KindPositions, Fields: map[string]string{
		FieldPostCode: "080.0000000005",
	}}
	if err := validate(old); err == nil {
		t.Fatal("کد با پیش‌شمارهٔ پیش‌فرض در طرح تازه باید رد شود")
	}
}

// TestSeedAssignsOrgCodes دادهٔ نمونهٔ پست‌ها همهٔ کدها را با پیش‌شمارهٔ درست می‌گیرد.
func TestSeedAssignsOrgCodes(t *testing.T) {
	s := testScheme()
	seeds := SeedWith(s)
	posts := 0
	seen := map[string]bool{}
	gradeCodes := map[string]string{}
	for _, item := range seeds {
		if item.Kind != KindPositions {
			continue
		}
		posts++
		for field, kind := range positionCodeFieldKinds() {
			code := strings.TrimSpace(item.Fields[field])
			if !s.ValidFor(code, kind) {
				t.Fatalf("کد %s نمونه معتبر نیست: %q", field, code)
			}
			if seen[field+"="+code] {
				t.Fatalf("کد تکراری در دادهٔ نمونه: %s=%s", field, code)
			}
			seen[field+"="+code] = true
		}
		grade := strings.TrimSpace(item.Fields["grade"])
		orgCode := strings.TrimSpace(item.Fields[FieldOrgGradeCode])
		if prev, ok := gradeCodes[grade]; ok && prev != orgCode {
			t.Fatalf("دو پست هم‌گروه باید یک پایهٔ سازمانی داشته باشند: %s / %s", prev, orgCode)
		}
		gradeCodes[grade] = orgCode
	}
	if posts < 4 {
		t.Fatalf("دادهٔ نمونه باید دست‌کم چهار پست داشته باشد: %d", posts)
	}
}

// TestCodeHelpersDeriveFromData کدهای ساخته‌شده از دادهٔ پرونده (پست، واحد، خانواده، شخصی).
func TestCodeHelpersDeriveFromData(t *testing.T) {
	s := testScheme()
	fields := map[string]string{"unit": "امور مالی", "jobFamily": "مالی و حسابداری", "grade": "کارشناسی"}
	if code := PostCodeOf(s, fields, "POS-201"); code != "080.0000000201" {
		t.Fatalf("شمارهٔ سمت ساخته‌شده نادرست است: %q", code)
	}
	if code := OrgGradeCodeOf(s, fields, "کارشناسی"); code != "081.0000000005" {
		t.Fatalf("پایهٔ سازمانی ساخته‌شده نادرست است: %q", code)
	}
	if code := UnitCodeOf(s, fields, "POS-201"); !s.ValidFor(code, rules.OrgCodeKindUnit) {
		t.Fatalf("کد واحد ساخته‌شده نادرست است: %q", code)
	}
	if code := JobFamilyCodeOf(s, fields); !s.ValidFor(code, rules.OrgCodeKindJobFamily) {
		t.Fatalf("کد خانوادهٔ شغلی ساخته‌شده نادرست است: %q", code)
	}
	if code := PersonalGradeCodeOf(s, "EMP-004"); code != "082.0000000004" {
		t.Fatalf("کد پایهٔ شخصی نادرست است: %q", code)
	}
	if code := AssignmentCodeOf(s, map[string]string{}, "ASG-0009"); code != "084.0000000009" {
		t.Fatalf("شمارهٔ انتصاب نادرست است: %q", code)
	}
	// کد ذخیره‌شده بر کد ساخته‌شده مقدم است.
	stored := map[string]string{FieldPostCode: "080.0000000042"}
	if code := PostCodeOf(s, stored, "POS-201"); code != "080.0000000042" {
		t.Fatalf("کد ذخیره‌شده باید مقدم باشد: %q", code)
	}
}

// TestDescriptorExposesCodeFields شناسهٔ پست سازمانی چهار فیلد کد و قاعدهٔ اعتبارسنجی دارد.
func TestDescriptorExposesCodeFields(t *testing.T) {
	d := DescriptorWith(testScheme())
	for _, kind := range d.Kinds {
		if kind.Key != KindPositions {
			continue
		}
		found := map[string]bool{}
		for _, f := range kind.Fields {
			found[f.Key] = true
			if strings.Contains(f.Key, "Code") && strings.TrimSpace(f.LabelFa) == "" {
				t.Errorf("برچسب فارسی فیلد کد خالی است: %+v", f)
			}
		}
		for _, key := range []string{FieldPostCode, FieldOrgGradeCode, FieldUnitCode, FieldJobFamilyCode} {
			if !found[key] {
				t.Fatalf("فیلد کد %s در شناسنامهٔ پست نیست", key)
			}
		}
		if kind.Validate == nil {
			t.Fatal("اعتبارسنجی کدهای پست وصل نشده است")
		}
		return
	}
	t.Fatal("نوع رکورد پست سازمانی پیدا نشد")
}
