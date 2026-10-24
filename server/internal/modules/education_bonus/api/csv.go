// پایانهٔ HTTP ماژول «education_bonus»: پذیرش JSON، اعتبارسنجی ورودی و پاسخ استاندارد httpx.

package api

import (
	"bytes"
	"encoding/csv"
	"strconv"

	"hrms/internal/contracts"
)

// خروجی فایل پاداش تحصیلی با CSV ساخته می‌شود چون واحد مالی همان را در Excel باز
// می‌کند. جداکنندهٔ «,» و کدگذاری UTF-8 با BOM (در هندلر نوشته می‌شود) است تا فارسی
// در Excel درست دیده شود.

func csvHeader() string {
	return renderCSV([][]string{{"کد پرسنلی", "نام کارمند", "دانش‌آموز ممتاز", "نسبت", "مقطع", "معدل", "مبنا", "مبلغ (ریال)"}})
}

func csvRow(row contracts.BonusRow) string {
	return renderCSV([][]string{{
		row.EmployeeID, row.EmployeeName, row.Student, row.Relation,
		row.LevelFa, row.Grade, row.BasisFa, strconv.FormatInt(row.Amount, 10),
	}})
}

// csvFooter جمع هر مقطع و جمع کل را به‌عنوان ردیف‌های پایانی می‌نویسد تا واحد مالی
// مجبور نباشد جمع را خودش حساب کند.
func csvFooter(result contracts.BonusResult) string {
	rows := [][]string{{"", "", "", "", "", "", "", ""}}
	for _, t := range result.ByLevel {
		rows = append(rows, []string{"", "", "", "", "جمع " + t.LevelFa, "", strconv.Itoa(t.Count) + " ردیف", strconv.FormatInt(t.Total, 10)})
	}
	rows = append(rows, []string{"", "", "", "", "جمع کل", "", strconv.Itoa(result.Count) + " ردیف", strconv.FormatInt(result.Total, 10)})
	return renderCSV(rows)
}

func renderCSV(rows [][]string) string {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	for _, r := range rows {
		_ = w.Write(r)
	}
	w.Flush()
	return buf.String()
}
