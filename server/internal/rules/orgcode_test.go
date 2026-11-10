package rules

import "testing"

// TestOrgCodeDefaultsAreSixPrefixes شش پیش‌شمارهٔ پیش‌فرض ۰۸۰ تا ۰۸۵.
func TestOrgCodeDefaultsAreSixPrefixes(t *testing.T) {
	scheme := DefaultOrgCodeScheme()
	want := map[string]string{
		OrgCodeKindPost:       "080",
		OrgCodeKindOrgGrade:   "081",
		OrgCodeKindPersonal:   "082",
		OrgCodeKindUnit:       "083",
		OrgCodeKindAssignment: "084",
		OrgCodeKindJobFamily:  "085",
	}
	view := scheme.View()
	if len(view) != 6 {
		t.Fatalf("شش نوع کد انتظار می‌رفت: %d", len(view))
	}
	for _, kind := range view {
		if want[kind.Key] != kind.Prefix || kind.TitleFa == "" {
			t.Errorf("تعریف نوع کد نادرست است: %+v", kind)
		}
	}
}

// TestOrgCodeFormatFillsTenDigits قالب کد: سه رقم پیش‌شماره، نقطه و ده رقم با صفر پیشرو.
func TestOrgCodeFormatFillsTenDigits(t *testing.T) {
	scheme := DefaultOrgCodeScheme()
	cases := []struct {
		kind   string
		serial int64
		want   string
	}{
		{OrgCodeKindPost, 1, "080.0000000001"},
		{OrgCodeKindOrgGrade, 42, "081.0000000042"},
		{OrgCodeKindPersonal, 4_294_967_295, "082.4294967295"},
		{OrgCodeKindAssignment, OrgCodeSerialMax, "084.9999999999"},
	}
	for _, c := range cases {
		got, ok := scheme.Format(c.kind, c.serial)
		if !ok || got != c.want {
			t.Errorf("کد %s/%d باید %q باشد، %q شد (ok=%v)", c.kind, c.serial, c.want, got, ok)
		}
		if len(got) != OrgCodeLength {
			t.Errorf("طول کد باید %d باشد: %q", OrgCodeLength, got)
		}
	}
	if code, ok := scheme.Format("kind-unknown", 1); ok {
		t.Errorf("نوع ناشناخته نباید کد بسازد: %q", code)
	}
	for _, serial := range []int64{0, -5, OrgCodeSerialMax + 1} {
		if code, ok := scheme.Format(OrgCodeKindPost, serial); ok {
			t.Errorf("سریال %d نباید پذیرفته شود: %q", serial, code)
		}
	}
}

// TestOrgCodeSchemeFromValues پیش‌شماره‌ها متغیرند و مقدار نامعتبر/تکراری نادیده می‌رود.
func TestOrgCodeSchemeFromValues(t *testing.T) {
	scheme := OrgCodeSchemeFromValues(map[string]string{
		"orgcode.prefix.post":      "201",
		"orgcode.prefix.personal":  "201", // تکراری با شمارهٔ سمت ⇒ نادیده
		"orgcode.prefix.unit":      "۸۳",  // دورقمی ⇒ نادیده
		"orgcode.prefix.jobFamily": "abc", // غیررقمی ⇒ نادیده
	})
	if got := scheme.Prefix(OrgCodeKindPost); got != "201" {
		t.Fatalf("پیش‌شمارهٔ شمارهٔ سمت باید ۲۰۱ شود: %q", got)
	}
	if got := scheme.Prefix(OrgCodeKindPersonal); got != "082" {
		t.Fatalf("پیش‌شمارهٔ تکراری باید نادیده گرفته شود: %q", got)
	}
	if got := scheme.Prefix(OrgCodeKindUnit); got != "083" {
		t.Fatalf("پیش‌شمارهٔ نامعتبر باید پیش‌فرض بماند: %q", got)
	}
	if got := scheme.Prefix(OrgCodeKindJobFamily); got != "085" {
		t.Fatalf("پیش‌شمارهٔ غیررقمی باید پیش‌فرض بماند: %q", got)
	}
	// کد ساخته‌شده با پیش‌شمارهٔ تازه و همان قالب خوانده می‌شود.
	code, ok := scheme.Format(OrgCodeKindPost, 7)
	if !ok || code != "201.0000000007" {
		t.Fatalf("کد با پیش‌شمارهٔ تازه ساخته نشد: %q", code)
	}
	if !scheme.ValidFor(code, OrgCodeKindPost) || scheme.ValidFor(code, OrgCodeKindUnit) {
		t.Fatal("تشخیص نوع کد با پیش‌شمارهٔ تازه نادرست است")
	}
}

// TestOrgCodeParseAndValidate تجزیهٔ کد، تشخیص نوع و رد قالب‌های نادرست.
func TestOrgCodeParseAndValidate(t *testing.T) {
	scheme := DefaultOrgCodeScheme()
	parsed, ok := scheme.Parse("081.0000000007")
	if !ok || parsed.Kind != OrgCodeKindOrgGrade || parsed.Serial != 7 {
		t.Fatalf("کد درست تجزیه نشد: %+v ok=%v", parsed, ok)
	}
	if parsed.TitleFa != "پایهٔ سمت سازمانی" {
		t.Fatalf("عنوان نوع کد نادرست است: %q", parsed.TitleFa)
	}
	for _, code := range []string{
		"", "080", "080.000000000", "080.00000000011", "080/0000000001",
		"999.0000000001", "080.0000000000", "abc.defghijklm", "080.000000000a",
	} {
		if scheme.Valid(code) {
			t.Errorf("کد نامعتبر پذیرفته شد: %q", code)
		}
	}
}

// TestOrgCodeAcceptsPersianDigits کد با ارقام فارسی هم خوانده می‌شود (ورودی کاربر).
func TestOrgCodeAcceptsPersianDigits(t *testing.T) {
	scheme := DefaultOrgCodeScheme()
	parsed, ok := scheme.Parse("۰۸۰.۰۰۰۰۰۰۰۰۰۳")
	if !ok || parsed.Serial != 3 || parsed.Kind != OrgCodeKindPost {
		t.Fatalf("کد با ارقام فارسی باید خوانده شود: %+v ok=%v", parsed, ok)
	}
	if !scheme.ValidFor("۰۸۲.۰۰۰۰۰۰۰۰۰۹", OrgCodeKindPersonal) {
		t.Fatal("کد پایهٔ شخصی با ارقام فارسی باید معتبر باشد")
	}
	if scheme.ValidFor("082.0000000009", OrgCodeKindPost) {
		t.Fatal("کد پایهٔ شخصی نباید به‌عنوان شمارهٔ سمت پذیرفته شود")
	}
}

// TestOrgCodeNext سریال بعدی بر پایهٔ کدهای موجود.
func TestOrgCodeNext(t *testing.T) {
	scheme := DefaultOrgCodeScheme()
	existing := []string{"080.0000000001", "080.0000000004", "081.0000000002", "نامعتبر"}
	code, ok := scheme.Next(OrgCodeKindPost, existing)
	if !ok || code != "080.0000000005" {
		t.Fatalf("شمارهٔ سمت بعدی باید ۰۸۰.۰۰۰۰۰۰۰۰۰۵ باشد: %q ok=%v", code, ok)
	}
	first, ok := scheme.Next(OrgCodeKindPersonal, nil)
	if !ok || first != "082.0000000001" {
		t.Fatalf("اولین کد پایهٔ شخصی باید ۰۸۲.۰۰۰۰۰۰۰۰۰۱ باشد: %q", first)
	}
}

// TestOrgCodeDerivedCodes کدهای ساخته‌شده از داده: پایهٔ شخصی، سمت، انتصاب، واحد و خانواده.
func TestOrgCodeDerivedCodes(t *testing.T) {
	scheme := DefaultOrgCodeScheme()
	if got := scheme.PersonalForEmployee("EMP-004"); got != "082.0000000004" {
		t.Fatalf("کد پایهٔ شخصی EMP-004 نادرست است: %q", got)
	}
	if got := scheme.ForRecord(OrgCodeKindPost, "POS-201"); got != "080.0000000201" {
		t.Fatalf("شمارهٔ سمت POS-201 نادرست است: %q", got)
	}
	if got := scheme.ForRecord(OrgCodeKindAssignment, "ASG-0007"); got != "084.0000000007" {
		t.Fatalf("شمارهٔ انتصاب ASG-0007 نادرست است: %q", got)
	}
	if got := scheme.OrgGradeFor("کارشناسی ارشد"); got != "081.0000000004" {
		t.Fatalf("کد پایهٔ سازمانی کارشناسی ارشد نادرست است: %q", got)
	}
	if got := scheme.OrgGradeFor(""); got != "" {
		t.Fatalf("گروه خالی نباید کد بسازد: %q", got)
	}
	for _, got := range []string{
		scheme.UnitFor("امور مالی"),
		scheme.JobFamilyFor("مالی و حسابداری"),
		scheme.PersonalForEmployee("کارمند-بدون-شماره"),
	} {
		if !scheme.Valid(got) {
			t.Fatalf("کد ساخته‌شده باید معتبر باشد: %q", got)
		}
	}
	if scheme.UnitFor("امور مالی") != scheme.UnitFor("امور مالی") {
		t.Fatal("کد واحد باید پایدار باشد")
	}
}

// TestOrgCodeFaAndParamKeys نمایش فارسی کد و کلیدهای پارامتر پیش‌شماره.
func TestOrgCodeFaAndParamKeys(t *testing.T) {
	if got := OrgCodeFa("080.0000000012"); got != "۰۸۰.۰۰۰۰۰۰۰۰۱۲" {
		t.Fatalf("نمایش فارسی کد نادرست است: %q", got)
	}
	keys := OrgCodePrefixParamKeys()
	if len(keys) != 6 || keys[0] != "orgcode.prefix.assignment" {
		t.Fatalf("کلیدهای پارامتر پیش‌شماره نادرست است: %+v", keys)
	}
	values := DefaultOrgCodeScheme().ParamValues()
	if len(values) != 6 || values["orgcode.prefix.post"] != "080" {
		t.Fatalf("مقادیر پارامتر پیش‌شماره نادرست است: %+v", values)
	}
}

// TestOrgCodeKindTitles عنوان‌های فارسی شش نوع کد.
func TestOrgCodeKindTitles(t *testing.T) {
	want := map[string]string{
		OrgCodeKindPost:       "شمارهٔ سمت",
		OrgCodeKindOrgGrade:   "پایهٔ سمت سازمانی",
		OrgCodeKindPersonal:   "پایهٔ شخصی",
		OrgCodeKindUnit:       "شمارهٔ واحد سازمانی",
		OrgCodeKindAssignment: "شمارهٔ انتصاب",
		OrgCodeKindJobFamily:  "کد خانوادهٔ شغلی",
	}
	for key, title := range want {
		if got := OrgCodeKindTitleFa(key); got != title {
			t.Errorf("عنوان %s باید %q باشد: %q", key, title, got)
		}
		if OrgCodeKindNoteFa(key) == "" {
			t.Errorf("توضیح %s نباید خالی باشد", key)
		}
	}
}
