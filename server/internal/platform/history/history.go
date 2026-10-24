// Package history چیدن و پالایش سطرهای «سابقه»ٔ کارمند.
//
// چرا این بسته جدا است: سابقه از چند ماژول می‌آید (احکام، قراردادها، ترفیعات، مدارک،
// وام‌ها و…)، ولی هیچ‌کدام از آن ماژول‌ها نباید به داخلی‌های دیگری دست بزند. هر ماژول
// فقط سابقهٔ خودش را به‌شکل `contracts.HistoryEntry` اعلام می‌کند و این بسته — که هیچ
// شناختی از داده و قواعد ماژول‌ها ندارد — سطرها را می‌چیند، آخرین سند هر نوع را نشان
// می‌کند و بر پایهٔ ماژول/نوع پالایش می‌کند. پس هیچ قاعدهٔ کسب‌وکاری اینجا نیست.
package history

import (
	"sort"
	"strings"

	"hrms/internal/contracts"
	"hrms/internal/platform/audit"
	"hrms/internal/platform/calendar"
)

// Sort سطرها را به ترتیب نزولی تاریخ (تازه‌ترین اول) می‌چیند.
//
// تاریخ‌ها به قالب شمسی YYYY/MM/DD هستند و همین قالب با مقایسهٔ رشته‌ای درست مرتب
// می‌شود؛ سطرهای بی‌تاریخ (سندی که هیچ فیلد تاریخی ندارد) به پایان می‌روند تا سابقهٔ
// تاریخ‌دار از دست نرود. برای پایداری چیدن، ترتیب دوم و سوم ماژول و شناسهٔ سند است.
func Sort(items []contracts.HistoryEntry) {
	sort.SliceStable(items, func(i, j int) bool {
		a, b := items[i], items[j]
		ad, bd := strings.TrimSpace(a.Date), strings.TrimSpace(b.Date)
		if ad != bd {
			if ad == "" {
				return false
			}
			if bd == "" {
				return true
			}
			return ad > bd
		}
		if a.Module != b.Module {
			return a.Module < b.Module
		}
		return a.RecordID < b.RecordID
	})
}

// MarkLatest روی تازه‌ترین سند هر (ماژول، نوع) برای این کارمند پرچم Latest را می‌گذارد.
//
// این کار قاعدهٔ نمایشی است، نه حکم دامنه: «آخرین سند این نوع» چیزی است که رابط کاربری
// باید به کاربر نشان دهد (حکم جاری، قرارداد جاری، سمت کنونی) و تفاوتش با اسناد پیشین
// روشن باشد. حذف‌شدن یا تأییدنشدن سند در دامنه تصمیم‌گیری نمی‌شود؛ تنها نشانهٔ زمانی است.
func MarkLatest(items []contracts.HistoryEntry) {
	best := map[string]int{}
	for i, e := range items {
		key := e.Module + "|" + e.Kind
		j, ok := best[key]
		if !ok {
			best[key] = i
			continue
		}
		if newer(e, items[j]) {
			best[key] = i
		}
	}
	for _, i := range best {
		items[i].Latest = true
	}
}

// newer آیا a سند تازه‌تری از b است (تاریخ بزرگ‌تر، و در تساوی تاریخ شناسهٔ بزرگ‌تر).
func newer(a, b contracts.HistoryEntry) bool {
	ad, bd := strings.TrimSpace(a.Date), strings.TrimSpace(b.Date)
	if ad != bd {
		return ad > bd
	}
	return a.RecordID > b.RecordID
}

// Filter پالایش بر پایهٔ ماژول و نوع سند؛ مقدار خالی یعنی «همه».
//
// چرا اینجا و نه در ماژول‌ها: فیلتر نمایشی است و کاربر می‌خواهد مثلاً فقط «احکام» یا
// فقط «وام‌ها» را ببیند، بی‌آنکه هر ماژول فیلتر تازه‌ای یاد بگیرد.
func Filter(items []contracts.HistoryEntry, module, kind string) []contracts.HistoryEntry {
	module = strings.TrimSpace(module)
	kind = strings.TrimSpace(kind)
	if module == "" && kind == "" {
		return items
	}
	out := make([]contracts.HistoryEntry, 0, len(items))
	for _, e := range items {
		if module != "" && e.Module != module {
			continue
		}
		if kind != "" && e.Kind != kind {
			continue
		}
		out = append(out, e)
	}
	return out
}

// FromAudit رخدادهای حسابرسی یک سند را به «سابقهٔ تغییرات» همان سند تبدیل می‌کند.
//
// چرا اینجا و نه در ماژول‌ها: تبدیل زمان و شکل سطر در همهٔ ماژول‌ها یکسان است؛ آنچه
// هر ماژول مالک آن است، **برچسب فارسی اقدام‌های خودش** است که با `labelFa` تزریق
// می‌شود (پس هیچ برچسبی این‌جا ساخته نمی‌شود). شرح رخداد (`Detail`) از خود لاگ می‌آید و
// دست‌نخورده می‌ماند؛ هیچ رخدادی ساخته یا حذف نمی‌شود.
func FromAudit(events []audit.Event, labelFa func(action string) string) []contracts.HistoryChange {
	if len(events) == 0 {
		return nil
	}
	out := make([]contracts.HistoryChange, 0, len(events))
	for _, e := range events {
		change := contracts.HistoryChange{
			At:     calendar.FormatStamp(e.At),
			Actor:  e.Actor,
			Action: e.Action,
			NoteFa: strings.TrimSpace(e.Detail),
		}
		if labelFa != nil {
			change.ActionFa = labelFa(e.Action)
		}
		out = append(out, change)
	}
	return out
}

// Counts شمارش سطرها به تفکیک ماژول (برای نوار خلاصهٔ سوابق).
func Counts(items []contracts.HistoryEntry) map[string]int {
	out := map[string]int{}
	for _, e := range items {
		out[e.Module]++
	}
	return out
}
