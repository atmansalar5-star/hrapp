// آزمون «تفاوت تطبیق» در فیش حقوقی و فرمت حقوق.
//
// چرا این آزمون: تفاوت تطبیق باید در دو جای چاپی دیده شود — سطر جداگانه در تفکیک اقلام
// فیش حقوقی و سطر مستقل در «اجزای مزد مبنا و فوق‌العاده‌ها». اگر این مسیر شکسته شود،
// کارمند مبلغی را در حکم می‌بیند که در فیش نیست.

package documents

import (
	"strings"
	"testing"
)

// payslipWithGap دادهٔ نمونهٔ فیش با تفاوت تطبیق.
func payslipWithGap() Data {
	layout := DefaultLayout(KindPayslip)
	return Data{
		Kind:   KindPayslip,
		Layout: layout,
		Payslip: &PayslipInfo{
			Period:            "1405/08",
			Gross:             120_000_000,
			InsuranceEmployee: 8_400_000,
			InsuranceEmployer: 27_600_000,
			Tax:               6_000_000,
			Net:               105_600_000,
			MazdBase:          90_000_000,
			PayGap:            23_460_000,
			PayGapFa:          "23,460,000",
			PayGapSince:       "1405/07/01",
			Elements: []PayslipElement{
				{TitleFa: "حق مسکن", CategoryFa: "مزایا", AmountFa: "۹٬۰۰۰٬۰۰۰", DetailFa: "مبلغ ثابت"},
			},
		},
	}
}

// TestPayslipShowsPayGapLine تفاوت تطبیق باید در فیش با مبلغ و تاریخ شروع بیاید.
func TestPayslipShowsPayGapLine(t *testing.T) {
	data := payslipWithGap()
	html, err := Render(data)
	if err != nil {
		t.Fatalf("رندر فیش ناموفق: %v", err)
	}
	if !strings.Contains(html, "تفاوت تطبیق") {
		t.Fatal("سطر «تفاوت تطبیق» در فیش حقوقی نیامد")
	}
	if !strings.Contains(html, "جذب‌شدنی") {
		t.Error("فیش باید توضیح دهد که تفاوت تطبیق جذب‌شدنی است")
	}
	// رقم ۲۳٬۴۶۰٬۰۰۰ باید با ارقام فارسی روی فیش باشد (جداکنندهٔ هزارگان از money می‌آید).
	if !strings.Contains(html, "۲۳,۴۶۰,۰۰۰") {
		t.Errorf("مبلغ تفاوت تطبیق با ارقام فارسی در فیش نیامد")
	}
	if !strings.Contains(html, "۱۴۰۵/۰۷/۰۱") {
		t.Errorf("تاریخ شروع تفاوت تطبیق در فیش نیامد")
	}
}

// TestPayslipWithoutPayGapHidesLine فیش بدون تفاوت تطبیق نباید سطر خالی نشان دهد.
func TestPayslipWithoutPayGapHidesLine(t *testing.T) {
	data := payslipWithGap()
	data.Payslip.PayGap = 0
	data.Payslip.PayGapSince = ""
	html, err := Render(data)
	if err != nil {
		t.Fatalf("رندر فیش ناموفق: %v", err)
	}
	if strings.Contains(html, "تفاوت تطبیق") {
		t.Error("فیش کارمند بدون تفاوت تطبیق نباید سطر تفاوت تطبیق داشته باشد")
	}
}

// TestPayGapPlaceholderVars جای‌نگهدارهای تفاوت تطبیق برای قالب‌های نوشتنی.
func TestPayGapPlaceholderVars(t *testing.T) {
	vars := PlaceholderVars(payslipWithGap())
	if vars["payslip.payGap"] != "23460000" {
		t.Errorf("جای‌نگهدار عددی تفاوت تطبیق نادرست است: %q", vars["payslip.payGap"])
	}
	if vars["payslip.payGapSince"] != "1405/07/01" {
		t.Errorf("جای‌نگهدار تاریخ تفاوت تطبیق نادرست است: %q", vars["payslip.payGapSince"])
	}
	if strings.TrimSpace(vars["payslip.payGapFa"]) == "" {
		t.Error("جای‌نگهدار خوانای تفاوت تطبیق خالی است")
	}
}

// TestWageStructureRowCarriesPayGapDetail سطر تفاوت تطبیق در فرمت حقوق باید بگوید جزو
// مزد مبنا نیست (مبنای اضافه‌کاری را بالا نمی‌برد).
func TestPayslipPayGapIsOutsideMazdBase(t *testing.T) {
	data := payslipWithGap()
	html, err := Render(data)
	if err != nil {
		t.Fatalf("رندر فیش ناموفق: %v", err)
	}
	if !strings.Contains(html, "خارج از مزد مبنا") {
		t.Error("فیش باید جایگاه تفاوت تطبیق را نسبت به مزد مبنا روشن کند")
	}
}
