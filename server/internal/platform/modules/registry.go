// Package modules رجیستری ماژول‌های سامانه است.
// داده‌ها از مانیفست‌ها تولید می‌شوند (registry.gen.go) و این فایل فقط رفتار می‌دهد.
package modules

import (
	"sort"
	"strings"
)

// Directorate حوزهٔ سازمانی — گروه سطح اول منو (مثلاً مدیریت سرمایه‌های انسانی).
// این نوع رفتاری ندارد؛ داده‌اش از contracts/departments.json تولید می‌شود.
type Directorate struct {
	ID       string
	Order    int
	TitleFa  string
	TitleEn  string
	DutiesFa []string
}

// SubUnit زیرواحد درون یک اداره — گروه سطح سوم منو.
type SubUnit struct {
	ID       string
	Order    int
	TitleFa  string
	DutiesFa []string
}

// Department ادارهٔ منابع انسانی — گروه سطح دوم منو.
// این نوع رفتاری ندارد؛ داده‌اش از contracts/departments.json تولید می‌شود (departments.gen.go).
type Department struct {
	ID          string
	Order       int
	Directorate string
	TitleFa     string
	TitleEn     string
	DutiesFa    []string
	SubUnits    []SubUnit
}

// departmentOrder رتبهٔ اداره در ترتیب منو؛ صفر یعنی ادارهٔ ناشناخته.
func departmentOrder(id string) int {
	for _, d := range Departments {
		if d.ID == id {
			return d.Order
		}
	}
	return 0
}

// DepartmentTitleFa عنوان فارسی اداره بر پایهٔ کلید آن.
func DepartmentTitleFa(id string) string {
	for _, d := range Departments {
		if d.ID == id {
			return d.TitleFa
		}
	}
	return ""
}

// DirectorateTitleFa عنوان فارسی حوزه بر پایهٔ کلید آن.
func DirectorateTitleFa(id string) string {
	for _, d := range Directorates {
		if d.ID == id {
			return d.TitleFa
		}
	}
	return ""
}

// DepartmentOf یافتن اداره با کلید آن.
func DepartmentOf(id string) (Department, bool) {
	for _, d := range Departments {
		if d.ID == id {
			return d, true
		}
	}
	return Department{}, false
}

// SubUnitTitleFa عنوان فارسی زیرواحد در یک اداره.
func SubUnitTitleFa(departmentID, subUnitID string) string {
	d, ok := DepartmentOf(departmentID)
	if !ok {
		return ""
	}
	for _, s := range d.SubUnits {
		if s.ID == subUnitID {
			return s.TitleFa
		}
	}
	return ""
}

// directorateOfDepartment حوزهٔ سازمانی یک اداره.
func directorateOfDepartment(id string) string {
	if d, ok := DepartmentOf(id); ok {
		return d.Directorate
	}
	return ""
}

// Screen یک صفحهٔ قابل نمایش در کلاینت.
type Screen struct {
	Key       string
	Order     int
	Icon      string
	TitleFa   string
	Component string
	Roles     []string
}

// Module یک ماژول مستقل با مرز مشخص: پیشوند API، نقش‌ها، دامنهٔ داده، صفحه‌ها و پرچم ویژگی.
type Module struct {
	ID          string
	Order       int
	Department  string
	TitleFa     string
	TitleEn     string
	APIPrefix   string
	Version     string
	FeatureFlag string
	DBSchema    string
	Owner       string
	Entities    []string
	Roles       map[string][]string
	Scopes      map[string]string
	Screens     []Screen
	// SubUnit کلید زیرواحد درون اداره؛ خالی یعنی اداره زیرواحد ندارد.
	SubUnit string
	// SharedWith اداره‌هایی که همین ماژول در منوی آن‌ها هم نمایش داده می‌شود.
	SharedWith []string
}

// Can بررسی می‌کند نقش اجازهٔ انجام اقدام را در این ماژول دارد.
func (m Module) Can(role, action string) bool {
	for _, a := range m.Roles[role] {
		if a == action {
			return true
		}
	}
	return false
}

// ActionsFor اقدام‌های مجاز یک نقش در این ماژول.
func (m Module) ActionsFor(role string) []string {
	return m.Roles[role]
}

// ScopeFor دامنهٔ دادهٔ نقش؛ پیش‌فرض سازمانی (all) است.
func (m Module) ScopeFor(role string) string {
	if s, ok := m.Scopes[role]; ok {
		return s
	}
	return "all"
}

// SubUnitTitleFa عنوان زیرواحد این ماژول در ادارهٔ خانهٔ خودش.
func (m Module) SubUnitTitleFa() string {
	return SubUnitTitleFa(m.Department, m.SubUnit)
}

// ScreensFor صفحه‌های قابل نمایش برای یک نقش (اقلام غیرمجاز حذف می‌شوند — دستور ۳-۱).
func (m Module) ScreensFor(role string) []Screen {
	out := make([]Screen, 0, len(m.Screens))
	for _, s := range m.Screens {
		for _, r := range s.Roles {
			if r == role {
				out = append(out, s)
				break
			}
		}
	}
	return out
}

// Registry رجیستری ماژول‌های فعال در این استقرار.
type Registry struct {
	byID    map[string]Module
	ordered []Module
	enabled map[string]bool // nil یعنی همه فعال
}

// NewRegistry رجیستری می‌سازد. اگر enabled خالی باشد همهٔ ماژول‌ها فعال‌اند.
func NewRegistry(enabled map[string]bool) *Registry {
	r := &Registry{byID: make(map[string]Module), enabled: enabled}
	for _, m := range All {
		if enabled != nil && !enabled[m.ID] {
			continue
		}
		r.byID[m.ID] = m
		r.ordered = append(r.ordered, m)
	}
	// ترتیب منو: نخست اداره، سپس ماژول — مستقل از ترتیب فایل مانیفست.
	sort.SliceStable(r.ordered, func(i, j int) bool {
		di, dj := departmentOrder(r.ordered[i].Department), departmentOrder(r.ordered[j].Department)
		if di != dj {
			return di < dj
		}
		return r.ordered[i].Order < r.ordered[j].Order
	})
	return r
}

// All ماژول‌های فعال به ترتیب منو.
func (r *Registry) All() []Module { return r.ordered }

// Count تعداد ماژول‌های فعال.
func (r *Registry) Count() int { return len(r.ordered) }

// ByID یافتن ماژول با شناسه.
func (r *Registry) ByID(id string) (Module, bool) {
	m, ok := r.byID[id]
	return m, ok
}

// IsEnabled بررسی فعال بودن ماژول در این استقرار (پایهٔ پرچم ویژگی).
func (r *Registry) IsEnabled(id string) bool {
	_, ok := r.byID[id]
	return ok
}

// Match پیدا کردن ماژول مالک یک مسیر بر اساس بلندترین پیشوند.
func (r *Registry) Match(path string) (Module, bool) {
	var best Module
	found := false
	for _, m := range r.ordered {
		if path == m.APIPrefix || strings.HasPrefix(path, m.APIPrefix+"/") {
			if !found || len(m.APIPrefix) > len(best.APIPrefix) {
				best, found = m, true
			}
		}
	}
	return best, found
}

// MenuFor درخت منوی نقش‌محور که کلاینت از سرور می‌گیرد.
//
// ترتیب و گروه‌بندی همان ترتیب مانیفست است: حوزه ← اداره ← زیرواحد ← ماژول ← صفحه.
// ماژول‌های مشترک (مانند طبقه‌بندی مشاغل که بین امور کارکنان و برون‌سپاری مشترک است)
// در منوی همهٔ اداره‌های مربوط تکرار می‌شوند و با پرچم Shared مشخص می‌شوند.
func (r *Registry) MenuFor(role string) []MenuEntry {
	out := make([]MenuEntry, 0, len(r.ordered))
	for _, m := range r.ordered {
		screens := m.ScreensFor(role)
		if len(screens) == 0 {
			continue
		}
		entries := make([]MenuScreen, 0, len(screens))
		for _, s := range screens {
			entries = append(entries, MenuScreen{Key: s.Key, TitleFa: s.TitleFa, Icon: s.Icon, Component: s.Component})
		}
		out = append(out, menuEntryFor(m, m.Department, false, entries))
		for _, dep := range m.SharedWith {
			out = append(out, menuEntryFor(m, dep, true, entries))
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		di, dj := departmentOrder(out[i].Department), departmentOrder(out[j].Department)
		if di != dj {
			return di < dj
		}
		if out[i].Shared != out[j].Shared {
			return !out[i].Shared
		}
		return out[i].Order < out[j].Order
	})
	return out
}

// MenuForWithExtra مانند MenuFor است، ولی ماژول‌های «افزوده» (که مانیفست برای این نقش
// صفحه‌ای تعریف نکرده اما مدیر سیستم دسترسی داده) هم با همهٔ صفحه‌هایشان به منو
// اضافه می‌شوند تا سطح دسترسی در ناوبری هم دیده شود.
func (r *Registry) MenuForWithExtra(role string, extraModules []string) []MenuEntry {
	out := r.MenuFor(role)
	seen := make(map[string]bool, len(out))
	for _, e := range out {
		seen[e.ModuleID] = true
	}
	for _, moduleID := range extraModules {
		if seen[moduleID] {
			continue
		}
		m, ok := r.byID[moduleID]
		if !ok || len(m.Screens) == 0 {
			continue
		}
		entries := make([]MenuScreen, 0, len(m.Screens))
		for _, s := range m.Screens {
			entries = append(entries, MenuScreen{Key: s.Key, TitleFa: s.TitleFa, Icon: s.Icon, Component: s.Component})
		}
		out = append(out, menuEntryFor(m, m.Department, false, entries))
		seen[moduleID] = true
	}
	sort.SliceStable(out, func(i, j int) bool {
		di, dj := departmentOrder(out[i].Department), departmentOrder(out[j].Department)
		if di != dj {
			return di < dj
		}
		if out[i].Shared != out[j].Shared {
			return !out[i].Shared
		}
		return out[i].Order < out[j].Order
	})
	return out
}

// menuEntryFor ساخت یک آیتم منو برای یک ادارهٔ مشخص (خانهٔ ماژول یا ادارهٔ مشترک).
func menuEntryFor(m Module, department string, shared bool, screens []MenuScreen) MenuEntry {
	return MenuEntry{
		ModuleID:          m.ID,
		TitleFa:           m.TitleFa,
		Order:             m.Order,
		Department:        department,
		DepartmentTitleFa: DepartmentTitleFa(department),
		Directorate:       directorateOfDepartment(department),
		SubUnit:           m.SubUnit,
		SubUnitTitleFa:    m.SubUnitTitleFa(),
		Shared:            shared,
		Screens:           screens,
	}
}

// MenuEntry یک آیتم منوی کلاینت.
type MenuEntry struct {
	ModuleID          string       `json:"moduleId"`
	TitleFa           string       `json:"titleFa"`
	Order             int          `json:"order"`
	Department        string       `json:"department"`
	DepartmentTitleFa string       `json:"departmentTitleFa"`
	Directorate       string       `json:"directorate"`
	SubUnit           string       `json:"subUnit"`
	SubUnitTitleFa    string       `json:"subUnitTitleFa"`
	Shared            bool         `json:"shared"`
	Screens           []MenuScreen `json:"screens"`
}

// MenuScreen یک زیرآیتم منو.
type MenuScreen struct {
	Key       string `json:"key"`
	TitleFa   string `json:"titleFa"`
	Icon      string `json:"icon"`
	Component string `json:"component"`
}
