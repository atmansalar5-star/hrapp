// آزمون گرد کردن جمعهای آماری: ساعت/روز اعشاری در رابط نباید دنبالهٔ دودویی نشان بدهد.
//
// چرا این آزمون: کاربر روی کارت «مجموع کارکرد (ساعت)» عددی مثل
// ۱۲۱٫۲۸۰۰۰۰۰۰۰۰۰۰۰۲ میدید، در حالی که جمع واقعی ۱۲۱٫۲۸ ساعت بود. علت، جمع شناورِ
// فیلدهای اعشاری (۷٫۱۷ و ۷٫۹۲) است؛ پس عدد آماری باید پیش از انتشار گرد شود و این
// آزمون همان مرز را میسنجد.

package statistics

import (
	"context"
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"hrms/internal/contracts"
	"hrms/internal/platform/modulekit"
)

// attendanceFake پورت فقطخواندنی ساختگی ماژول حضور؛ فیلتر «نوع رکورد» را همانجا
// اعمال میکند تا مسیر واقعی خواندن رکوردها آزموده شود.
type attendanceFake struct{ items []contracts.Record }

func (f attendanceFake) List(_ context.Context, filter contracts.RecordFilter) (contracts.RecordPage, error) {
	items := []contracts.Record{}
	for _, rec := range f.items {
		if filter.Kind == "" || rec.Kind == filter.Kind {
			items = append(items, rec)
		}
	}
	return contracts.RecordPage{Items: items, Total: len(items), Page: 1, Size: len(items)}, nil
}

func (f attendanceFake) Schema() contracts.RecordSchema {
	return contracts.RecordSchema{Module: "attendance", ModuleTitleFa: "حضور و غیاب"}
}

// cleanDecimals میسنجد که نمایش قطعی یک عدد شناور حداکثر دو رقم اعشار داشته باشد.
func cleanDecimals(v float64) bool {
	s := strconv.FormatFloat(v, 'g', -1, 64)
	if strings.ContainsAny(s, "eE") {
		return false // نمای علمی یعنی عدد پاک نشده است.
	}
	if i := strings.IndexByte(s, '.'); i >= 0 {
		return len(s)-i-1 <= 2
	}
	return true
}

// TestAttendanceHoursRounded مقادیر واقعی نمونهٔ نمایشی را جمع میزند: جمع اعشاری
// «۱۲۱٫۲۸» است، ولی جمع شناور دنبالهدار میماند؛ پاسخ ماژول باید ۱۲۱٫۲۸ باشد.
func TestAttendanceHoursRounded(t *testing.T) {
	hours := []string{
		"7.5", "8.4", "8.0", "8.3", "9.6", "7.6", "8.0", "9.1", "7.9", "8.2",
		"7.17", "7.92", "7.92", "7.17", "8.5",
	}
	items := make([]contracts.Record, 0, len(hours))
	var rawSum float64
	for i, h := range hours {
		items = append(items, contracts.Record{
			ID: "att-" + strconv.Itoa(i+1), Kind: "attendanceRecords", Title: "تردد روزانه",
			Status: "approved", Fields: map[string]string{"workedHours": h},
		})
		rawSum += num(h)
	}
	// اگر مقادیر نمونه دنبالهٔ شناور نسازند، این آزمون دیگر مرزِ گزارششده را نمیسنجد.
	if cleanDecimals(rawSum) {
		t.Fatalf("مقادیر آزمون دنبالهٔ شناور نمیسازد (جمع خام %s)؛ نمونهٔ آزمون بیاثر است", strconv.FormatFloat(rawSum, 'g', -1, 64))
	}

	ports := modulekit.NewPorts()
	ports.SetRecordList("attendance", attendanceFake{items: items})
	out := attendanceStats(context.Background(), modulekit.Deps{Ports: ports})

	if out.WorkedHours != 121.28 {
		t.Fatalf("مجموع کارکرد باید ۱۲۱٫۲۸ باشد: %s", strconv.FormatFloat(out.WorkedHours, 'g', -1, 64))
	}
	if !cleanDecimals(out.WorkedHours) {
		t.Fatalf("مجموع کارکرد دنبالهٔ اعشاری دارد: %s", strconv.FormatFloat(out.WorkedHours, 'g', -1, 64))
	}
	// جمع چهار سنجهٔ کارتهای آماری و سطرهای برش واحد هم باید پاک باشد.
	for name, v := range map[string]float64{
		"اضافهکاری": out.OvertimeHours, "مرخصی": out.LeaveDays, "مأموریت": out.MissionDays,
	} {
		if !cleanDecimals(v) {
			t.Errorf("%s دنبالهٔ اعشاری دارد: %s", name, strconv.FormatFloat(v, 'g', -1, 64))
		}
	}
	var deptSum float64
	for _, row := range out.DeptRows {
		if !cleanDecimals(row.Hours) {
			t.Errorf("ساعت برش واحد «%s» دنبالهٔ اعشاری دارد: %s", row.TitleFa, strconv.FormatFloat(row.Hours, 'g', -1, 64))
		}
		deptSum += row.Hours
	}
	if round2(deptSum) != out.WorkedHours {
		t.Errorf("جمع ساعت برش واحد‌ها (%s) با مجموع کارکرد (%s) نمی‌خواند",
			strconv.FormatFloat(deptSum, 'g', -1, 64), strconv.FormatFloat(out.WorkedHours, 'g', -1, 64))
	}

	// مرز نهایی: همان چیزی که کلاینت می‌گیرد — بدنهٔ JSON.
	body, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("ترجمان JSON پاسخ ناموفق بود: %v", err)
	}
	if !strings.Contains(string(body), `"workedHours":121.28`) {
		t.Errorf("بدنهٔ JSON باید کارکرد گردشده را بفرستد؛ یافت‌نشده در: %s",
			regexp.MustCompile(`"workedHours":[^,}]*`).FindString(string(body)))
	}
}
