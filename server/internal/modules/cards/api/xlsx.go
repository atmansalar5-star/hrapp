// خروجی اکسل (xlsx) فایل شارژ کارت‌های رفاهی.
//
// چرا اکسل و نه فقط CSV: فایل CSV فارسی در Excel وابسته به تنظیم ناحیه‌ای هر رایانه است
// (جداکنندهٔ فهرست و کدگذاری) و کاربر همان چیزی را می‌بیند که «فارسی نیست و فونت به‌هم
// ریخته». فایل xlsx خودش کدگذاری، قلم و راست‌به‌چپ را حمل می‌کند و روی هر رایانه‌ای یک
// شکل باز می‌شود. CSV هم می‌ماند (برای سامانهٔ مالی) ولی با پایان‌خط CRLF.
package api

import (
	"fmt"
	"strconv"
	"strings"

	"hrms/internal/contracts"
	"hrms/internal/platform/calendar"

	"github.com/xuri/excelize/v2"
)

// chargeSheetName نام برگهٔ فایل شارژ (فارسی، بدون نویسهٔ ممنوع در نام برگه).
const chargeSheetName = "فایل شارژ"

// excelFont قلم خروجی: قلمی که فارسی را درست نشان می‌دهد و روی ویندوز هست.
const excelFont = "Tahoma"

// chargeWorkbook فایل شارژ را به شکل کتاب کار اکسل می‌سازد.
//
// ساختار: سرصفحه (دوره، کارت، دامنه)، جدول ردیف‌ها با سرستون‌های فارسی، جمع هر بخش و
// جمع کل. برگه راست‌به‌چپ است و ستون مبلغ قالب عددی هزارجداکننده دارد.
func chargeWorkbook(result contracts.CardChargeResult) ([]byte, error) {
	book := excelize.NewFile()
	defer func() { _ = book.Close() }()
	// قلم پیشفرض کتاب کار هم فارسیپسند میشود؛ وگرنه ستونها و سلولهای تازه که
	// واحد مالی خودش اضافه میکند با قلم لاتین باز میشوند.
	if err := book.SetDefaultFont(excelFont); err != nil {
		return nil, err
	}

	index, err := book.NewSheet(chargeSheetName)
	if err != nil {
		return nil, err
	}
	book.SetActiveSheet(index)
	_ = book.DeleteSheet("Sheet1")
	// برگه راست‌به‌چپ باز می‌شود تا ستون‌ها و متن فارسی درست بچینند.
	if err := book.SetSheetView(chargeSheetName, 0, &excelize.ViewOptions{RightToLeft: boolPtr(true)}); err != nil {
		return nil, err
	}

	headerStyle, err := book.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF", Size: 11, Family: excelFont},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"0E7490"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true, ReadingOrder: 2},
		Border: []excelize.Border{
			{Type: "left", Color: "94A3B8", Style: 1}, {Type: "right", Color: "94A3B8", Style: 1},
			{Type: "top", Color: "94A3B8", Style: 1}, {Type: "bottom", Color: "94A3B8", Style: 1},
		},
	})
	if err != nil {
		return nil, err
	}
	titleStyle, err := book.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 13, Family: excelFont},
		Alignment: &excelize.Alignment{Horizontal: "right", ReadingOrder: 2},
	})
	if err != nil {
		return nil, err
	}
	cellStyle, err := book.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 11, Family: excelFont},
		Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "center", ReadingOrder: 2},
		Border: []excelize.Border{
			{Type: "left", Color: "E2E8F0", Style: 1}, {Type: "right", Color: "E2E8F0", Style: 1},
			{Type: "top", Color: "E2E8F0", Style: 1}, {Type: "bottom", Color: "E2E8F0", Style: 1},
		},
	})
	if err != nil {
		return nil, err
	}
	moneyStyle, err := book.NewStyle(&excelize.Style{
		Font:         &excelize.Font{Size: 11, Family: excelFont},
		Alignment:    &excelize.Alignment{Horizontal: "left", ReadingOrder: 2},
		NumFmt:       3, // #,##0
		CustomNumFmt: strPtr("#,##0"),
		Border: []excelize.Border{
			{Type: "left", Color: "E2E8F0", Style: 1}, {Type: "right", Color: "E2E8F0", Style: 1},
			{Type: "top", Color: "E2E8F0", Style: 1}, {Type: "bottom", Color: "E2E8F0", Style: 1},
		},
	})
	if err != nil {
		return nil, err
	}
	totalStyle, err := book.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 11, Family: excelFont},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"F1F5F9"}},
		Alignment: &excelize.Alignment{Horizontal: "right", ReadingOrder: 2},
	})
	if err != nil {
		return nil, err
	}
	// جمعها هم مثل ردیفهای ریز، هزارجداکننده دارند تا مبلغ مالی خوانا باشد.
	totalMoneyStyle, err := book.NewStyle(&excelize.Style{
		Font:         &excelize.Font{Bold: true, Size: 11, Family: excelFont},
		Fill:         excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"F1F5F9"}},
		Alignment:    &excelize.Alignment{Horizontal: "left", ReadingOrder: 2},
		NumFmt:       3,
		CustomNumFmt: strPtr("#,##0"),
	})
	if err != nil {
		return nil, err
	}

	// سرصفحهٔ سند.
	_ = book.SetCellValue(chargeSheetName, "A1", "فایل شارژ "+result.CardFa+" — دورهٔ "+result.Period)
	_ = book.SetCellStyle(chargeSheetName, "A1", "A1", titleStyle)
	_ = book.SetCellValue(chargeSheetName, "A2", "دامنه: "+result.ScopeFa+" · تعداد ردیف: "+persianInt(int64(result.Count)))
	_ = book.SetCellStyle(chargeSheetName, "A2", "A2", titleStyle)
	if result.FileNoFa != "" {
		_ = book.SetCellValue(chargeSheetName, "A3", "شمارهٔ فایل شارژ: "+result.FileNoFa)
		_ = book.SetCellStyle(chargeSheetName, "A3", "A3", titleStyle)
	}

	const firstRow = 4
	headers := []string{"کد پرسنلی", "نام کارمند", "ذینفع", "نسبت", "بخش / قلم", "مناسبت", "مبنای محاسبه", "مبلغ (ریال)"}
	for column, title := range headers {
		cell, err := excelize.CoordinatesToCellName(column+1, firstRow)
		if err != nil {
			return nil, err
		}
		_ = book.SetCellValue(chargeSheetName, cell, title)
		_ = book.SetCellStyle(chargeSheetName, cell, cell, headerStyle)
	}

	row := firstRow + 1
	for _, item := range result.Rows {
		values := []any{
			item.EmployeeID, item.EmployeeName, item.Beneficiary, item.Relation,
			item.SectionFa, item.OccasionFa, item.BasisFa, item.Amount,
		}
		for column, value := range values {
			cell, err := excelize.CoordinatesToCellName(column+1, row)
			if err != nil {
				return nil, err
			}
			_ = book.SetCellValue(chargeSheetName, cell, value)
			if column == len(values)-1 {
				_ = book.SetCellStyle(chargeSheetName, cell, cell, moneyStyle)
				continue
			}
			_ = book.SetCellStyle(chargeSheetName, cell, cell, cellStyle)
		}
		row++
	}

	// جمع‌ها: یک ردیف خالی، سپس جمع هر بخش و جمع کل.
	row++
	for _, total := range result.BySection {
		_ = book.SetCellValue(chargeSheetName, fmt.Sprintf("E%d", row), "جمع "+total.SectionFa)
		_ = book.SetCellValue(chargeSheetName, fmt.Sprintf("G%d", row), persianInt(int64(total.Count))+" ردیف")
		_ = book.SetCellValue(chargeSheetName, fmt.Sprintf("H%d", row), total.Total)
		_ = book.SetCellStyle(chargeSheetName, fmt.Sprintf("E%d", row), fmt.Sprintf("G%d", row), totalStyle)
		_ = book.SetCellStyle(chargeSheetName, fmt.Sprintf("H%d", row), fmt.Sprintf("H%d", row), totalMoneyStyle)
		row++
	}
	_ = book.SetCellValue(chargeSheetName, fmt.Sprintf("E%d", row), "جمع کل")
	_ = book.SetCellValue(chargeSheetName, fmt.Sprintf("G%d", row), persianInt(int64(result.Count))+" ردیف")
	_ = book.SetCellValue(chargeSheetName, fmt.Sprintf("H%d", row), result.Total)
	_ = book.SetCellStyle(chargeSheetName, fmt.Sprintf("E%d", row), fmt.Sprintf("G%d", row), totalStyle)
	_ = book.SetCellStyle(chargeSheetName, fmt.Sprintf("H%d", row), fmt.Sprintf("H%d", row), totalMoneyStyle)

	// هشدارها (مثل «تعلق ندارد» یا «بیش از سقف») در پایان همان برگه می‌آید تا مالی بداند
	// چرا بعضی افراد در فایل نیستند.
	if len(result.Warnings) > 0 {
		row += 2
		_ = book.SetCellValue(chargeSheetName, fmt.Sprintf("A%d", row), "هشدارهای شارژ")
		_ = book.SetCellStyle(chargeSheetName, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), titleStyle)
		row++
		for _, warning := range result.Warnings {
			_ = book.SetCellValue(chargeSheetName, fmt.Sprintf("A%d", row), "• "+warning)
			_ = book.SetCellStyle(chargeSheetName, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), cellStyle)
			row++
		}
	}

	widths := map[string]float64{"A": 14, "B": 20, "C": 20, "D": 12, "E": 26, "F": 18, "G": 30, "H": 18}
	for column, width := range widths {
		if err := book.SetColWidth(chargeSheetName, column, column, width); err != nil {
			return nil, err
		}
	}
	_ = book.SetPanes(chargeSheetName, &excelize.Panes{Freeze: true, YSplit: firstRow, TopLeftCell: fmt.Sprintf("A%d", firstRow+1), ActivePane: "bottomLeft"})

	buf, err := book.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// strPtr اشاره‌گر رشتهٔ قالب عددی (excelize این شکل را می‌خواهد).
func strPtr(v string) *string { return &v }

// persianInt عدد را برای متنهای فارسی برگه با ارقام فارسی برمی‌گرداند (مبلغها عددی
// و با قالب #,##0 میمانند تا مالی بتواند جمع بزند؛ فقط برچسبها فارسی میشوند).
func persianInt(n int64) string {
	return calendar.ToPersianDigits(strconv.FormatInt(n, 10))
}

// boolPtr اشاره‌گر بولین (excelize این شکل را می‌خواهد).
func boolPtr(v bool) *bool { return &v }

// csvEscapeLineEndings پایان‌خط‌ها را به CRLF برمی‌گرداند.
//
// چرا: Excel روی ویندوز فایل با پایان‌خط LF را در برخی نسخه‌ها یک‌خطی نشان می‌دهد.
func csvEscapeLineEndings(body string) string {
	return strings.ReplaceAll(strings.ReplaceAll(body, "\r\n", "\n"), "\n", "\r\n")
}
