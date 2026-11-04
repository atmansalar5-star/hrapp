// آزمون «پاراگراف متن سند» و «یادداشت پای سند» به‌عنوان متن قابل ویرایش.
//
// چرا این آزمون‌ها: خواستهٔ کارفرما این است که علاوه بر بلوک‌ها، خودِ متن گواهی/حکم در
// پنل پارامتریک نوشتنی باشد. سه چیز باید تضمین شود: متن پیش‌فرض از قالب جای‌نگهداردار
// ساخته شود (تا پنل همان را نشان دهد)، متن پنل جای پیش‌فرض را بگیرد، و جای‌نگهدارهای
// شرطی هرگز خام روی برگه چاپ نشوند.

package documents

import (
	"strings"
	"testing"
)

// TestDefaultTextIsPlaceholderTemplate بررسی می‌کند متن پیش‌فرض «قالب» باشد نه متن آماده.
func TestDefaultTextIsPlaceholderTemplate(t *testing.T) {
	cert := DefaultText(KindEmploymentCertificate)["body"]
	for _, want := range []string{"{{employee.fullName}}", "{{employee.hireDate}}", "{{employee.position}}", "{{contract.kindClause}}"} {
		if !strings.Contains(cert, want) {
			t.Errorf("متن پیش‌فرض گواهی جای‌نگهدار %s را ندارد: %q", want, cert)
		}
	}
	payslip := DefaultText(KindPayslip)["body"]
	if !strings.Contains(payslip, "{{payslip.grossFa}}") {
		t.Errorf("متن پیش‌فرض فیش مبلغ خوانا را ندارد: %q", payslip)
	}
	if note := DefaultText(KindSalaryDecree)["note"]; !strings.Contains(note, "کارانه") {
		t.Errorf("یادداشت پیش‌فرض حکم درست نیست: %q", note)
	}
	// سند فرم‌محور (مدرک ملاک حقوق) متن پاراگراف پیش‌فرض ندارد.
	if len(DefaultText(KindDegreeBasis)) != 0 {
		t.Error("سند مدرک ملاک حقوق نباید متن پاراگراف پیش‌فرض داشته باشد")
	}
}

// TestApplyDefaultTextFillsRealValues بررسی می‌کند قالب پیش‌فرض با دادهٔ واقعی پر شود.
func TestApplyDefaultTextFillsRealValues(t *testing.T) {
	data := certificateData()
	data.BodyFa = nil
	data.Contract = &ContractInfo{ID: "CT-1", KindFa: "تمام‌وقت"}
	applyDefaultText(&data)
	if len(data.BodyFa) != 3 {
		t.Fatalf("گواهی باید سه پاراگراف پیش‌فرض داشته باشد، %d آمد: %#v", len(data.BodyFa), data.BodyFa)
	}
	first := data.BodyFa[0]
	for _, want := range []string{"علی مرادی", "1399/06/15", "مالی و حسابداری", "کارشناس ارشد مالی", "پتروشیمی بندر امام", "تمام‌وقت"} {
		if !strings.Contains(first, want) {
			t.Errorf("پاراگراف گواهی «%s» را ندارد: %q", want, first)
		}
	}
	if strings.Contains(strings.Join(data.BodyFa, " "), "{{") {
		t.Errorf("جای‌نگهدار خام روی سند ماند: %#v", data.BodyFa)
	}
}

// TestPanelTextReplacesBodyAndNote متن پنل پارامتریک را بر پیش‌فرض مقدم می‌داند.
func TestContractClausesComeFromTemplate(t *testing.T) {
	data := Data{
		Kind:     KindContract,
		Company:  Company{TitleFa: "پتروشیمی بندر امام", WorkWeekFa: "شنبه تا چهارشنبه"},
		Employee: baseInfoEmployee(),
		// پارامترهای سازمانی خارج از جدول سند: باید در متن قرارداد بیایند.
		OrgParams: []Param{
			{Key: "leave.annualDays", Value: "26"},
			{Key: "payroll.monthlyHours", Value: "192"},
		},
	}
	applyDefaultText(&data)
	if len(data.BodyFa) != 6 {
		t.Fatalf("قرارداد باید شش بند پیش‌فرض داشته باشد، %d آمد: %#v", len(data.BodyFa), data.BodyFa)
	}
	joined := strings.Join(data.BodyFa, " ")
	for _, want := range []string{"مالی و حسابداری", "کارشناس ارشد مالی", "شنبه تا چهارشنبه", "192", "26"} {
		if !strings.Contains(joined, want) {
			t.Errorf("بندهای قرارداد «%s» را ندارد: %q", want, joined)
		}
	}
	if strings.Contains(joined, "{{") {
		t.Errorf("جای‌نگهدار خام در بندهای قرارداد ماند: %#v", data.BodyFa)
	}
}

func TestPanelTextReplacesBodyAndNote(t *testing.T) {
	data := certificateData()
	data.BodyFa = nil
	layout := DefaultLayout(KindEmploymentCertificate)
	layout.Texts = map[string]string{
		"body": "گواهی می‌شود {{employee.fullName}} با کد پرسنلی {{employee.id}} در «{{company.title}}» شاغل است.\nاین گواهی برای ارائه به مراجع ذی‌ربط صادر شده است.",
		"note": "صدور این سند از سامانهٔ منابع انسانی قابل بررسی است.",
	}
	data.Layout = layout

	applyDefaultText(&data)
	applyPanelTexts(&data)

	if len(data.BodyFa) != 2 {
		t.Fatalf("متن پنل باید دو پاراگراف بدهد، %d آمد: %#v", len(data.BodyFa), data.BodyFa)
	}
	if !data.CustomBody {
		t.Error("متن نوشته‌شده در پنل باید به شکل پاراگراف چاپ شود")
	}
	if !strings.Contains(data.BodyFa[0], "علی مرادی") || !strings.Contains(data.BodyFa[0], "EMP-004") {
		t.Errorf("جای‌نگهدار متن پنل پر نشد: %q", data.BodyFa[0])
	}
	joined := strings.Join(data.BodyFa, " ")
	if strings.Contains(joined, "بنا به درخواست نامبرده") {
		t.Error("پاراگراف پیش‌فرض به‌جای متن پنل چاپ شد")
	}
	if data.NoteFa != "صدور این سند از سامانهٔ منابع انسانی قابل بررسی است." {
		t.Errorf("یادداشت پنل اعمال نشد: %q", data.NoteFa)
	}

	html, err := Render(data)
	if err != nil {
		t.Fatalf("رندر سند ناموفق: %v", err)
	}
	if !strings.Contains(html, "doc-body") || !strings.Contains(html, "مراجع ذی‌ربط") {
		t.Error("متن پنل در برگهٔ چاپی نیامد")
	}
}

// TestKindTextSlots هر سند باید جای متن «پاراگراف» و «یادداشت» داشته باشد.
func TestKindTextSlots(t *testing.T) {
	for _, doc := range Catalog() {
		keys := map[string]bool{}
		for _, slot := range doc.Texts {
			if strings.TrimSpace(slot.TitleFa) == "" || strings.TrimSpace(slot.HintFa) == "" {
				t.Errorf("سند %s جای متن %s عنوان/راهنمای فارسی ندارد", doc.Kind, slot.Key)
			}
			keys[slot.Key] = true
		}
		for _, need := range []string{"body", "note"} {
			if !keys[need] {
				t.Errorf("سند %s جای متن %s را ندارد", doc.Kind, need)
			}
		}
	}
	if len(TextSlots()) != 2 {
		t.Errorf("تعداد جای‌های متن %d است، انتظار ۲", len(TextSlots()))
	}
}
