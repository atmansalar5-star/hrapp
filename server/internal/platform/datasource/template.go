// ساخت قالب آمادهٔ بارگذاری از ساختار داده: یک کارپوشهٔ اکسل با یک برگه برای هر جدول
// (به‌همراه برگهٔ راهنما) و قالب CSV برای هر جدول به‌صورت جداگانه.
//
// چرا قالب ساختهٔ سرور است: سرصفحهٔ ستون‌ها همان کلیدهای فنی سامانه است و از ساختار داده
// می‌آید؛ پس قالب هرگز با سرور ناهمخوان نمی‌شود و کاربر فقط ردیف‌ها را پر می‌کند.

package datasource

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"sort"
	"strings"

	"github.com/xuri/excelize/v2"
)

// sheetName درست‌کردن نام برگه برای اکسل: بی‌نویسه‌های ممنوع و کوتاه‌تر از ۳۱ نویسه.
func sheetName(title string, index int) string {
	replacer := strings.NewReplacer("\\", "-", "/", "-", "?", "-", "*", "-", "[", "(", "]", ")", ":", "-")
	name := strings.TrimSpace(replacer.Replace(title))
	if name == "" {
		name = fmt.Sprintf("جدول %d", index)
	}
	runes := []rune(name)
	if len(runes) > 31 {
		name = string(runes[:31])
	}
	return name
}

// TemplateXLSX کارپوشهٔ قالب را می‌سازد: برگهٔ «راهنما» + یک برگه برای هر جدول.
//
// در هر برگه، ردیف نخست سرصفحهٔ فنی (کلید ستون) و ردیف دوم نمونهٔ فارسی است تا کاربر
// نداند کدام ستون را چه بنویسد؛ ردیف نمونه با رنگ کم‌رنگ و ایتالیک نوشته می‌شود.
func TemplateXLSX(entities []Entity, orgTitle string) ([]byte, error) {
	book := excelize.NewFile()
	defer book.Close()

	headerStyle, err := book.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF", Size: 11},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"0E7490"}},
		Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "center", WrapText: true},
		Border: []excelize.Border{
			{Type: "bottom", Color: "0B5566", Style: 1},
		},
	})
	if err != nil {
		return nil, err
	}
	sampleStyle, err := book.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Italic: true, Color: "64748B", Size: 10},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"F1F5F9"}},
		Alignment: &excelize.Alignment{Horizontal: "right"},
	})
	if err != nil {
		return nil, err
	}
	requiredStyle, err := book.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Color: "B91C1C", Size: 10},
		Alignment: &excelize.Alignment{Horizontal: "right"},
	})
	if err != nil {
		return nil, err
	}

	// برگهٔ راهنما: قاعده‌های عمومی + فهرست جدول‌ها و برگه‌هایشان.
	guide := "راهنما"
	if _, err := book.NewSheet(guide); err != nil {
		return nil, err
	}
	book.SetActiveSheet(0)
	_ = book.DeleteSheet("Sheet1")
	_ = book.SetCellValue(guide, "A1", "راهنمای تکمیل قالب بارگذاری — "+orgTitle)
	_ = book.SetCellStyle(guide, "A1", "A1", headerStyle)
	guideLines := []string{
		"۱) هر برگهٔ این فایل یک جدول هدف است؛ سرصفحهٔ ستون‌ها را عوض نکنید (کلیدهای فنی سامانه‌اند).",
		"۲) ردیف دوم هر برگه نمونهٔ فارسی است؛ آن را پاک کنید و دادهٔ خودتان را از همان ردیف دوم بنویسید.",
		"۳) اعداد را بدون جداکنندهٔ هزارگان و تاریخ‌ها را شمسی به قالب YYYY/MM/DD بنویسید.",
		"۴) ستون‌هایی که با «الزامی» نشان شده‌اند نباید خالی بمانند.",
		"۵) ترتیب بارگذاری: ابتدا برگهٔ «شرکت‌ها»، سپس «کارکنان»، سپس برگه‌های وابسته.",
		"۶) پس از تکمیل، فایل را در صفحهٔ «تبادل داده» بارگذاری و گزارش اعتبارسنجی را ببینید.",
	}
	for index, line := range guideLines {
		cell := fmt.Sprintf("A%d", index+3)
		_ = book.SetCellValue(guide, cell, line)
	}
	_ = book.SetCellValue(guide, "A11", "فهرست جدول‌ها")
	_ = book.SetCellStyle(guide, "A11", "A11", headerStyle)
	_ = book.SetCellValue(guide, "A12", "برگه")
	_ = book.SetCellValue(guide, "B12", "جدول هدف")
	_ = book.SetCellValue(guide, "C12", "جای بارگذاری")
	_ = book.SetCellValue(guide, "D12", "شمار ستون")
	for index, entity := range entities {
		row := 13 + index
		_ = book.SetCellValue(guide, fmt.Sprintf("A%d", row), sheetName(entity.SheetFa, index+1))
		_ = book.SetCellValue(guide, fmt.Sprintf("B%d", row), entity.TitleFa)
		_ = book.SetCellValue(guide, fmt.Sprintf("C%d", row), entity.Module+"."+entity.Kind)
		_ = book.SetCellValue(guide, fmt.Sprintf("D%d", row), len(entity.Fields))
	}
	_ = book.SetColWidth(guide, "A", "A", 34)
	_ = book.SetColWidth(guide, "B", "B", 30)
	_ = book.SetColWidth(guide, "C", "C", 28)
	_ = book.SetColWidth(guide, "D", "D", 12)

	for index, entity := range entities {
		name := sheetName(entity.SheetFa, index+1)
		if _, err := book.NewSheet(name); err != nil {
			// نام برگهٔ تکراری: پسوند شماره‌دار.
			name = sheetName(fmt.Sprintf("%s %d", entity.SheetFa, index+1), index+1)
			if _, err := book.NewSheet(name); err != nil {
				return nil, err
			}
		}
		for column, field := range entity.Fields {
			cell, err := excelize.CoordinatesToCellName(column+1, 1)
			if err != nil {
				return nil, err
			}
			_ = book.SetCellValue(name, cell, field.Key)
			_ = book.SetCellStyle(name, cell, cell, headerStyle)
			width := float64(len([]rune(field.TitleFa))) + 6
			if width < 14 {
				width = 14
			}
			columnName, err := excelize.ColumnNumberToName(column + 1)
			if err == nil {
				_ = book.SetColWidth(name, columnName, columnName, width)
			}
			// ردیف نمونه: مقدار نمونه یا راهنمای کوتاه همان ستون.
			sampleCell, err := excelize.CoordinatesToCellName(column+1, 2)
			if err != nil {
				return nil, err
			}
			sample := strings.TrimSpace(field.Example)
			if sample == "" {
				sample = strings.TrimSpace(field.FormatFa)
			}
			if sample == "" && len(field.ValuesFa) > 0 {
				sample = strings.Join(field.ValuesFa, " | ")
			}
			if sample == "" {
				sample = field.TitleFa
			}
			_ = book.SetCellValue(name, sampleCell, sample)
			style := sampleStyle
			if field.Required {
				style = requiredStyle
			}
			_ = book.SetCellStyle(name, sampleCell, sampleCell, style)
		}
		_ = book.SetPanes(name, &excelize.Panes{Freeze: true, Split: false, YSplit: 2, TopLeftCell: "A3", ActivePane: "bottomLeft"})
	}

	buffer := &bytes.Buffer{}
	if err := book.Write(buffer); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

// TemplateCSV قالب CSV یک جدول را می‌سازد (سرصفحه + ردیف نمونه).
func TemplateCSV(entity Entity) ([]byte, error) {
	buffer := &bytes.Buffer{}
	buffer.WriteString("\ufeff") // BOM تا Excel فارسی درست باز کند
	writer := csv.NewWriter(buffer)
	header := make([]string, 0, len(entity.Fields))
	sample := make([]string, 0, len(entity.Fields))
	for _, field := range entity.Fields {
		header = append(header, field.Key)
		value := strings.TrimSpace(field.Example)
		if value == "" {
			value = strings.TrimSpace(field.FormatFa)
		}
		if value == "" && len(field.ValuesFa) > 0 {
			value = strings.Join(field.ValuesFa, " | ")
		}
		sample = append(sample, value)
	}
	if err := writer.Write(header); err != nil {
		return nil, err
	}
	if err := writer.Write(sample); err != nil {
		return nil, err
	}
	writer.Flush()
	return buffer.Bytes(), writer.Error()
}

// FindEntity موجودیت را با کلید، نام برگه یا «ماژول.نوع» پیدا می‌کند.
func FindEntity(entities []Entity, key string) (Entity, bool) {
	value := strings.TrimSpace(key)
	if value == "" {
		return Entity{}, false
	}
	lower := strings.ToLower(value)
	for _, entity := range entities {
		if strings.ToLower(entity.Key) == lower || entity.SheetFa == value ||
			strings.ToLower(entity.Module+"."+entity.Kind) == lower {
			return entity, true
		}
	}
	return Entity{}, false
}

// SelectEntities برگه‌های خواسته‌شده را به ترتیب ساختار داده برمی‌گرداند.
//
// ورودی خالی یعنی همهٔ جدول‌ها؛ برای قالب سبک‌تر می‌توان مثلاً employees,bankAccounts داد.
func SelectEntities(entities []Entity, keys []string) []Entity {
	if len(keys) == 0 {
		return entities
	}
	picked := map[string]bool{}
	for _, key := range keys {
		if entity, ok := FindEntity(entities, key); ok {
			picked[entity.Key] = true
		}
	}
	out := make([]Entity, 0, len(picked))
	for _, entity := range entities {
		if picked[entity.Key] {
			out = append(out, entity)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Order < out[j].Order })
	return out
}
