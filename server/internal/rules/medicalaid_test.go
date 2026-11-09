// آزمون قاعدهٔ «کمک هزینه‌های درمانی»: تعرفه، سقف قلم، سهم سازمان و بیمه، نسخه و سقف سالانه.

package rules

import "testing"

// aidTariff یک قلم تعرفهٔ نمونه: ۱۰٬۰۰۰٬۰۰۰ ریال، سهم سازمان ۷۰٪.
func aidTariff() MedicalTariffItem {
	return MedicalTariffItem{
		Code: "ORT-001", Category: MedicalAidOrtesisProsthesis, TitleFa: "کفش طبی",
		UnitFa: "هر عدد", Amount: 10_000_000, OrgSharePercent: 0.7,
		RequirePrescription: true, RefFa: "جدول اروتز و پروتز — بند ۱",
	}
}

// aidParams پارامترهای پیش‌فرض آزمون.
func aidParams() MedicalAidParams {
	return DefaultMedicalAidParams()
}

// TestMedicalAidCapsAtTariff سازمان بیش از تعرفه نمی‌پردازد.
func TestMedicalAidCapsAtTariff(t *testing.T) {
	res := ComputeMedicalAid(MedicalAidInput{
		Category: MedicalAidOrtesisProsthesis,
		Items: []MedicalAidClaimItem{{
			Code: "ORT-001", Quantity: 1, InvoiceAmount: 15_000_000,
			Tariff: aidTariff(), PrescriptionApproved: true,
		}},
	}, aidParams())

	line := res.Lines[0]
	if line.BaseAmount != 10_000_000 {
		t.Fatalf("مبنای محاسبه باید تعرفه باشد، شد %d", line.BaseAmount)
	}
	if line.PayableAmount != 7_000_000 {
		t.Fatalf("سهم سازمان باید ۷٬۰۰۰٬۰۰۰ باشد، شد %d", line.PayableAmount)
	}
	if line.EmployeeAmount != 8_000_000 {
		t.Fatalf("سهم کارمند باید مازاد تعرفه باشد (۸٬۰۰۰٬۰۰۰)، شد %d", line.EmployeeAmount)
	}
}

// TestMedicalAidUsesInvoiceWhenLower فاکتور کمتر از تعرفه، مبنای محاسبه است.
func TestMedicalAidUsesInvoiceWhenLower(t *testing.T) {
	res := ComputeMedicalAid(MedicalAidInput{
		Category: MedicalAidDrug,
		Items: []MedicalAidClaimItem{{
			Code: "DRG-001", Quantity: 1, InvoiceAmount: 4_000_000,
			Tariff:               MedicalTariffItem{Code: "DRG-001", Amount: 6_000_000, OrgSharePercent: 0.7},
			PrescriptionApproved: true,
		}},
	}, aidParams())

	if res.Lines[0].BaseAmount != 4_000_000 {
		t.Fatalf("مبنا باید مبلغ فاکتور باشد: %d", res.Lines[0].BaseAmount)
	}
	if res.Lines[0].PayableAmount != 2_800_000 {
		t.Fatalf("سهم سازمان باید ۷۰٪ فاکتور باشد: %d", res.Lines[0].PayableAmount)
	}
}

// TestMedicalAidQuantityMultiplies تعرفه در تعداد ضرب می‌شود.
func TestMedicalAidQuantityMultiplies(t *testing.T) {
	res := ComputeMedicalAid(MedicalAidInput{
		Category: MedicalAidDrug,
		Items: []MedicalAidClaimItem{{
			Code: "DRG-002", Quantity: 3, InvoiceAmount: 99_000_000,
			Tariff:               MedicalTariffItem{Code: "DRG-002", Amount: 5_000_000, OrgSharePercent: 0.7},
			PrescriptionApproved: true,
		}},
	}, aidParams())

	if res.Lines[0].TariffAmount != 15_000_000 {
		t.Fatalf("تعرفهٔ سه واحد باید ۱۵٬۰۰۰٬۰۰۰ باشد: %d", res.Lines[0].TariffAmount)
	}
	if res.Lines[0].BaseAmount != 15_000_000 || res.Lines[0].PayableAmount != 10_500_000 {
		t.Fatalf("محاسبهٔ تعداد نادرست است: base=%d payable=%d", res.Lines[0].BaseAmount, res.Lines[0].PayableAmount)
	}
}

// TestMedicalAidItemCeiling سقف قلم کسر می‌شود.
func TestMedicalAidItemCeiling(t *testing.T) {
	tariff := aidTariff()
	tariff.CeilingAmount = 6_000_000
	res := ComputeMedicalAid(MedicalAidInput{
		Category: MedicalAidOrtesisProsthesis,
		Items: []MedicalAidClaimItem{{
			Code: "ORT-001", Quantity: 1, InvoiceAmount: 10_000_000,
			Tariff: tariff, PrescriptionApproved: true,
		}},
	}, aidParams())

	line := res.Lines[0]
	if line.BaseAmount != 6_000_000 || line.CeilingApplied != 4_000_000 {
		t.Fatalf("سقف قلم اعمال نشد: base=%d ceiling=%d", line.BaseAmount, line.CeilingApplied)
	}
	if line.PayableAmount != 4_200_000 {
		t.Fatalf("سهم سازمان پس از سقف باید ۴٬۲۰۰٬۰۰۰ باشد: %d", line.PayableAmount)
	}
	if res.CeilingApplied != 4_000_000 {
		t.Fatalf("جمع کسر سقف نادرست است: %d", res.CeilingApplied)
	}
}

// TestMedicalAidPrescriptionRequired بدون نسخهٔ تأییدشده پرداخت نمی‌شود.
func TestMedicalAidPrescriptionRequired(t *testing.T) {
	res := ComputeMedicalAid(MedicalAidInput{
		Category: MedicalAidOrtesisProsthesis,
		Items: []MedicalAidClaimItem{{
			Code: "ORT-001", Quantity: 1, InvoiceAmount: 10_000_000, Tariff: aidTariff(),
		}},
	}, aidParams())

	if res.Lines[0].Payable || res.Lines[0].PayableAmount != 0 {
		t.Fatalf("قلم بدون نسخهٔ تأییدشده نباید پرداخت شود: %+v", res.Lines[0])
	}
	if len(res.BlockingFa) == 0 {
		t.Fatal("دلیل پرداخت‌نشدن باید در خروجی بیاید")
	}
	// با خاموش‌بودن پارامتر، همان قلم پرداخت می‌شود.
	params := aidParams()
	params.RequirePrescription = false
	ok := ComputeMedicalAid(MedicalAidInput{
		Category: MedicalAidOrtesisProsthesis,
		Items: []MedicalAidClaimItem{{
			Code: "ORT-001", Quantity: 1, InvoiceAmount: 10_000_000, Tariff: aidTariff(),
		}},
	}, params)
	if ok.Lines[0].PayableAmount != 7_000_000 {
		t.Fatalf("با خاموش‌بودن شرط نسخه باید پرداخت شود: %d", ok.Lines[0].PayableAmount)
	}
}

// TestMedicalAidUnknownTariff تعرفهٔ ثبت‌نشده پرداخت نمی‌شود.
func TestMedicalAidUnknownTariff(t *testing.T) {
	res := ComputeMedicalAid(MedicalAidInput{
		Category: MedicalAidOther,
		Items: []MedicalAidClaimItem{{
			Code: "XXX", Quantity: 1, InvoiceAmount: 5_000_000, PrescriptionApproved: true,
		}},
	}, aidParams())

	if res.Lines[0].PayableAmount != 0 || res.Lines[0].Payable {
		t.Fatalf("قلم بدون تعرفه نباید پرداخت شود: %+v", res.Lines[0])
	}
}

// TestMedicalAidAnnualCeiling سقف سالانه از سطرهای آخر کسر می‌شود.
func TestMedicalAidAnnualCeiling(t *testing.T) {
	params := aidParams()
	params.AnnualCeiling = 12_000_000
	res := ComputeMedicalAid(MedicalAidInput{
		Category:   MedicalAidDrug,
		AnnualPaid: 4_000_000,
		Items: []MedicalAidClaimItem{
			{Code: "A", Quantity: 1, InvoiceAmount: 10_000_000,
				Tariff: MedicalTariffItem{Code: "A", Amount: 10_000_000, OrgSharePercent: 0.7}, PrescriptionApproved: true},
			{Code: "B", Quantity: 1, InvoiceAmount: 10_000_000,
				Tariff: MedicalTariffItem{Code: "B", Amount: 10_000_000, OrgSharePercent: 0.7}, PrescriptionApproved: true},
		},
	}, params)

	// جمع سهم سازمان پیش از سقف ۱۴٬۰۰۰٬۰۰۰ است و سقف (۱۲ − ۴ = ۸ میلیون) فقط ۸ میلیون می‌دهد.
	if res.TotalPayable != 8_000_000 {
		t.Fatalf("سهم سازمان باید به سقف محدود شود (۸٬۰۰۰٬۰۰۰): %d", res.TotalPayable)
	}
	if !res.AnnualExceeded || res.CeilingApplied != 6_000_000 {
		t.Fatalf("کسر سقف سالانه نادرست است: exceeded=%v ceiling=%d", res.AnnualExceeded, res.CeilingApplied)
	}
	if res.AnnualPaidAfter != 12_000_000 || res.AnnualRemaining != 0 {
		t.Fatalf("وضعیت سالانه نادرست است: after=%d remaining=%d", res.AnnualPaidAfter, res.AnnualRemaining)
	}
	// کسر از سطر آخر آغاز می‌شود.
	if res.Lines[1].PayableAmount >= res.Lines[0].PayableAmount {
		t.Fatalf("کسر باید از سطرهای آخر باشد: %d / %d", res.Lines[0].PayableAmount, res.Lines[1].PayableAmount)
	}
}

// TestMedicalAidInsuranceShare سهم بیمه بخشی از سهم سازمان است و دوباره شمرده نمی‌شود.
func TestMedicalAidInsuranceShare(t *testing.T) {
	tariff := aidTariff()
	tariff.InsuranceSharePercent = 0.5
	res := ComputeMedicalAid(MedicalAidInput{
		Category: MedicalAidOrtesisProsthesis,
		Items: []MedicalAidClaimItem{{
			Code: "ORT-001", Quantity: 1, InvoiceAmount: 10_000_000, Tariff: tariff, PrescriptionApproved: true,
		}},
	}, aidParams())

	if res.TotalInsurance != 3_500_000 {
		t.Fatalf("سهم بیمه باید نصف سهم سازمان باشد (۳٬۵۰۰٬۰۰۰): %d", res.TotalInsurance)
	}
	if res.TotalPayable != 7_000_000 {
		t.Fatalf("سهم بیمه نباید به جمع پرداخت اضافه شود: %d", res.TotalPayable)
	}
}

// TestMedicalAidParamsFromValues پارامتر شرکت روی پیش‌فرض می‌نشیند (درصد و نسبت هر دو).
func TestMedicalAidParamsFromValues(t *testing.T) {
	p := MedicalAidParamsFromValues(map[string]float64{
		"medical.aid.orgSharePercent":     80, // درصدی نوشته شده
		"medical.aid.annualCeiling":       250_000_000,
		"medical.aid.perItemCeiling":      50_000_000,
		"medical.aid.requirePrescription": 0,
	})
	if p.OrgSharePercent != 0.8 {
		t.Fatalf("درصد ۸۰ باید نسبت ۰٫۸ شود: %v", p.OrgSharePercent)
	}
	if p.AnnualCeiling != 250_000_000 || p.PerItemCeiling != 50_000_000 {
		t.Fatalf("سقف‌ها نادرست نشستند: %+v", p)
	}
	if p.RequirePrescription {
		t.Fatal("مقدار صفر باید شرط نسخه را خاموش کند")
	}

	ratio := MedicalAidParamsFromValues(map[string]float64{"medical.aid.orgSharePercent": 0.65})
	if ratio.OrgSharePercent != 0.65 {
		t.Fatalf("نسبت ۰٫۶۵ باید همان بماند: %v", ratio.OrgSharePercent)
	}
	if d := MedicalAidParamsFromValues(nil); d.OrgSharePercent != DefaultMedicalAidParams().OrgSharePercent {
		t.Fatal("کلید غایب باید پیش‌فرض بماند")
	}
}

// TestMedicalAidPerItemCeilingParam سقف پیش‌فرض پارامتر روی اقلام بی‌سقف اثر می‌گذارد.
func TestMedicalAidPerItemCeilingParam(t *testing.T) {
	params := aidParams()
	params.PerItemCeiling = 3_000_000
	res := ComputeMedicalAid(MedicalAidInput{
		Category: MedicalAidDrug,
		Items: []MedicalAidClaimItem{{
			Code: "DRG-003", Quantity: 1, InvoiceAmount: 9_000_000,
			Tariff: MedicalTariffItem{Code: "DRG-003", Amount: 9_000_000, OrgSharePercent: 0.7}, PrescriptionApproved: true,
		}},
	}, params)

	if res.Lines[0].BaseAmount != 3_000_000 || res.Lines[0].PayableAmount != 2_100_000 {
		t.Fatalf("سقف پیش‌فرض قلم اعمال نشد: %+v", res.Lines[0])
	}
}

// TestMedicalAidCategories و دستورالعمل کامل‌اند.
func TestMedicalAidCategories(t *testing.T) {
	want := []string{
		MedicalAidOrtesisProsthesis, MedicalAidEquipment, MedicalAidDrug, MedicalAidGlasses,
		MedicalAidDental, MedicalAidParaclinic, MedicalAidVisit, MedicalAidOther,
	}
	cats := MedicalAidCategories()
	if len(cats) != len(want) {
		t.Fatalf("شمار دسته‌ها نادرست است: %d", len(cats))
	}
	for i, key := range want {
		if cats[i].Key != key {
			t.Errorf("دستهٔ %d باید %q باشد، شد %q", i, key, cats[i].Key)
		}
		if cats[i].TitleFa == "" {
			t.Errorf("دستهٔ %q بدون عنوان فارسی است", key)
		}
	}
	if MedicalAidCategoryTitleFa(MedicalAidDrug) != "دارو" {
		t.Fatalf("عنوان دستهٔ دارو نادرست است: %q", MedicalAidCategoryTitleFa(MedicalAidDrug))
	}
	if steps := MedicalAidInstructionFa(); len(steps) < 8 {
		t.Fatalf("دستورالعمل باید دست‌کم ۸ گام داشته باشد: %d", len(steps))
	}
}

// TestMedicalAidSummaryIncludesTotals جمع‌بندی باید مبلغ‌ها را بگوید.
func TestMedicalAidSummaryIncludesTotals(t *testing.T) {
	res := ComputeMedicalAid(MedicalAidInput{
		Category: MedicalAidGlasses,
		Items: []MedicalAidClaimItem{{
			Code: "GL-001", Quantity: 1, InvoiceAmount: 10_000_000,
			Tariff: MedicalTariffItem{Code: "GL-001", Amount: 10_000_000, OrgSharePercent: 0.7}, PrescriptionApproved: true,
		}},
	}, aidParams())

	if res.SummaryFa == "" || res.CategoryFa != "عینک" {
		t.Fatalf("جمع‌بندی یا عنوان دسته نیامد: %q / %q", res.SummaryFa, res.CategoryFa)
	}
	if res.ItemsCount != 1 || res.TotalPayable != 7_000_000 {
		t.Fatalf("جمع‌های خروجی نادرست است: %+v", res)
	}
}
