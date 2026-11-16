// آزمون قاعدهٔ «compensation_independence» در موتور قواعد. آزمون خودکار؛ هر تغییر رفتار پیاده‌سازی
// باید آزمون همین پرونده را به‌روز کند.

package rules

import (
	"strings"
	"testing"
)

// TestCompensationModelIsOptionalByDefault مدل ۴پی باید مستقل و اختیاری باشد: پیش‌فرض
// «اعمال در پرداخت» خاموش است، پس هیچ عددی از این مدل خودبه‌خود به حکم و فیش نمی‌رسد.
//
// این آزمون عمدی است: اگر روزی کسی پیش‌فرض را روشن کند، پرداخت سازمان بی‌اعلام عوض
// می‌شود و احکام مبتنی بر مدل‌های پیشین (حقوق و دستمزد جاری و صندوق نفت) باید بی‌اثر
// می‌ماندند. قاعدهٔ «اختیاری بودن» با همین آزمون قفل می‌شود.
func TestCompensationModelIsOptionalByDefault(t *testing.T) {
	if DefaultCompensationParams().PayrollApply {
		t.Fatal("مدل ۴پی باید اختیاری باشد: پیش‌فرض «اعمال در پرداخت» باید خاموش باشد")
	}
	if CompensationParamsFromText(map[string]string{}).PayrollApply {
		t.Fatal("نبود پارامتر شرکت یعنی خاموش؛ نه روشن")
	}
	on := CompensationParamsFromText(map[string]string{"compensation.payroll.apply": "1"})
	if !on.PayrollApply {
		t.Fatal("با روشن کردن صریح پارامتر شرکت، مدل باید قابل اعمال شود")
	}
	off := CompensationParamsFromText(map[string]string{"compensation.payroll.apply": "0"})
	if off.PayrollApply {
		t.Fatal("پارامتر صفر باید مدل را خاموش نگه دارد")
	}
	// پارامترهای دیگر با روشن/خاموش شدن این کلید عوض نمی‌شوند.
	if on.BandBaseMidpoint != off.BandBaseMidpoint || on.JobMaxGrade != off.JobMaxGrade {
		t.Fatal("کلید «اعمال در پرداخت» نباید پارامترهای دیگر را تغییر دهد")
	}
}

// TestCompensationIndependenceNote متن رسمی استقلال مدل، حالت خاموش و روشن را درست بگوید.
func TestCompensationIndependenceNote(t *testing.T) {
	off := CompensationIndependenceFa(false)
	for _, want := range []string{"مستقل از مدل حقوق و دستمزد جاری", "صندوق نفت", "خاموش", "اختیاری"} {
		if !strings.Contains(off, want) {
			t.Fatalf("متن حالت خاموش باید شامل «%s» باشد: %s", want, off)
		}
	}
	on := CompensationIndependenceFa(true)
	for _, want := range []string{"روشن است", "احکام", "معتبر می‌مانند", "بازنویسی نمی‌شوند"} {
		if !strings.Contains(on, want) {
			t.Fatalf("متن حالت روشن باید شامل «%s» باشد: %s", want, on)
		}
	}
}
