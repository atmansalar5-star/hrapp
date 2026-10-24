// Package access انبار «سطوح دسترسی» است که مدیر سیستم آن را در زمان اجرا تعریف می‌کند.
//
// ماتریس پایه از مانیفست ماژول‌ها می‌آید (و همان مانیفست، سیاست هستهٔ Rust را هم
// می‌سازد)؛ این پکیج *فقط افزودنی* است: مدیر سیستم می‌تواند به یک نقش، برای یک اداره و
// یک ماژول، اقدام‌های بیشتر بدهد. هیچ سطحی نمی‌تواند دسترسی‌ای را که مانیفست داده، بگیرد
// و ماژولی که در این استقرار خاموش است با grant روشن نمی‌شود — این محدودیت در rbac اعمال
// می‌شود. جدول ماندگار نیست و با راه‌اندازی دوباره از نو ساخته می‌شود، چون مرجع نهایی
// همان مانیفست نسخه‌دار است.
package access

import (
	"sort"
	"strings"
	"sync"
)

// همهٔ ماژول‌ها و همهٔ اقدام‌ها: برای دادن دسترسی گسترده («*») به یک نقش.
const (
	AnyModule = "*"
	AnyAction = "*"
)

// Grant یک سطح دسترسی افزوده.
//
// دامنهٔ سطح سه‌گانه است و به همان ترتیب محدود می‌شود:
//   - اداره (DepartmentID): خالی یعنی همهٔ اداره‌ها، وگرنه فقط همان اداره.
//   - شرکت (OrganizationID): خالی یعنی هیچ شرکت خاصی هدف نیست.
//   - هلدینگ (HoldingFa): خالی یعنی هیچ هلدینگ خاصی هدف نیست.
//
// اگر هر دو شناسهٔ شرکت و هلدینگ خالی باشند، سطح «جهانی» است و همه‌جا اعمال می‌شود
// (کار مدیر سیستم). اگر OrganizationID پر باشد فقط در همان شرکت و اگر HoldingFa پر
// باشد فقط در شرکت‌های همان هلدینگ معتبر است.
type Grant struct {
	ID             string   `json:"id"`
	Role           string   `json:"role"`
	ModuleID       string   `json:"moduleId"`
	DepartmentID   string   `json:"departmentId,omitempty"`
	OrganizationID string   `json:"organizationId,omitempty"`
	HoldingFa      string   `json:"holdingFa,omitempty"`
	Actions        []string `json:"actions"`
	NoteFa         string   `json:"noteFa,omitempty"`
	CreatedBy      string   `json:"createdBy,omitempty"`
}

// ScopeMatches آیا سطح با دامنهٔ فعال (شرکت/هلدینگ در حال باز بودن) هم‌خوان است.
//
// قاعدهٔ مرجع: سطح جهانی (بی شرکت و بی هلدینگ) همه‌جا اعمال می‌شود؛ سطح شرکت‌محور
// فقط در همان شرکت؛ سطح هلدینگ‌محور فقط در شرکت‌های همان هلدینگ.
func ScopeMatches(grantOrg, grantHolding, activeOrg, activeHolding string) bool {
	if grantOrg == "" && grantHolding == "" {
		return true
	}
	if grantOrg != "" {
		return grantOrg == activeOrg
	}
	return activeHolding != "" && grantHolding == activeHolding
}

// Matches آیا این سطح برای ترکیب نقش/اداره/ماژول/اقدام و دامنهٔ فعال معتبر است.
func (g Grant) Matches(role, departmentID, moduleID, action, organizationID, holdingFa string) bool {
	if g.Role != role {
		return false
	}
	if g.ModuleID != AnyModule && g.ModuleID != moduleID {
		return false
	}
	if g.DepartmentID != "" && g.DepartmentID != departmentID {
		return false
	}
	if !ScopeMatches(g.OrganizationID, g.HoldingFa, organizationID, holdingFa) {
		return false
	}
	for _, a := range g.Actions {
		if a == AnyAction || a == action {
			return true
		}
	}
	return false
}

// Store انبار درون‌حافظه‌ای سطوح دسترسی.
type Store struct {
	mu     sync.RWMutex
	grants map[string]Grant
	seq    int
}

// NewStore انبار خالی می‌سازد.
func NewStore() *Store {
	return &Store{grants: map[string]Grant{}}
}

// Grants فهرست سطوح به ترتیب پایدار (نقش، ماژول، اداره).
func (s *Store) Grants() []Grant {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Grant, 0, len(s.grants))
	for _, g := range s.grants {
		out = append(out, g)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Role != out[j].Role {
			return out[i].Role < out[j].Role
		}
		if out[i].ModuleID != out[j].ModuleID {
			return out[i].ModuleID < out[j].ModuleID
		}
		return out[i].DepartmentID < out[j].DepartmentID
	})
	return out
}

// Upsert سطح دسترسی را می‌سازد یا (بر پایهٔ نقش+ماژول+اداره) به‌روز می‌کند.
func (s *Store) Upsert(g Grant) Grant {
	s.mu.Lock()
	defer s.mu.Unlock()
	g.Role = strings.TrimSpace(g.Role)
	g.ModuleID = strings.TrimSpace(g.ModuleID)
	g.DepartmentID = strings.TrimSpace(g.DepartmentID)
	g.OrganizationID = strings.TrimSpace(g.OrganizationID)
	g.HoldingFa = strings.TrimSpace(g.HoldingFa)
	g.Actions = normalizeActions(g.Actions)

	for id, existing := range s.grants {
		if existing.Role == g.Role && existing.ModuleID == g.ModuleID && existing.DepartmentID == g.DepartmentID &&
			existing.OrganizationID == g.OrganizationID && existing.HoldingFa == g.HoldingFa {
			g.ID = id
			if g.CreatedBy == "" {
				g.CreatedBy = existing.CreatedBy
			}
			s.grants[id] = g
			return g
		}
	}
	s.seq++
	g.ID = "GRANT-" + itoa(s.seq)
	s.grants[g.ID] = g
	return g
}

// Delete یک سطح را حذف می‌کند و می‌گوید پیدا شد یا نه.
func (s *Store) Delete(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.grants[id]; !ok {
		return false
	}
	delete(s.grants, id)
	return true
}

// Allows آیا سطوح افزوده اجازهٔ این اقدام را در دامنهٔ فعال می‌دهند.
func (s *Store) Allows(role, departmentID, moduleID, action, organizationID, holdingFa string) bool {
	if s == nil {
		return false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, g := range s.grants {
		if g.Matches(role, departmentID, moduleID, action, organizationID, holdingFa) {
			return true
		}
	}
	return false
}

// GrantedModules ماژول‌هایی که این نقش در این اداره و دامنهٔ فعال دسترسی افزوده دارد.
func (s *Store) GrantedModules(role, departmentID, organizationID, holdingFa string) []string {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	seen := map[string]bool{}
	for _, g := range s.grants {
		if g.Role != role || g.ModuleID == AnyModule || len(g.Actions) == 0 {
			continue
		}
		if g.DepartmentID != "" && g.DepartmentID != departmentID {
			continue
		}
		if !ScopeMatches(g.OrganizationID, g.HoldingFa, organizationID, holdingFa) {
			continue
		}
		seen[g.ModuleID] = true
	}
	out := make([]string, 0, len(seen))
	for id := range seen {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

// ActionsFor اقدام‌هایی که سطوح افزوده برای این ترکیب و دامنهٔ فعال می‌دهند.
func (s *Store) ActionsFor(role, departmentID, moduleID, organizationID, holdingFa string) []string {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	seen := map[string]bool{}
	for _, g := range s.grants {
		if g.Role != role || (g.ModuleID != AnyModule && g.ModuleID != moduleID) {
			continue
		}
		if g.DepartmentID != "" && g.DepartmentID != departmentID {
			continue
		}
		if !ScopeMatches(g.OrganizationID, g.HoldingFa, organizationID, holdingFa) {
			continue
		}
		for _, a := range g.Actions {
			seen[a] = true
		}
	}
	out := make([]string, 0, len(seen))
	for a := range seen {
		out = append(out, a)
	}
	sort.Strings(out)
	return out
}

// normalizeActions اقدام‌ها را یکتا و مرتب می‌کند.
func normalizeActions(actions []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, a := range actions {
		a = strings.TrimSpace(a)
		if a == "" || seen[a] {
			continue
		}
		seen[a] = true
		out = append(out, a)
	}
	sort.Strings(out)
	return out
}

// itoa تبدیل ساده و بدون وابستگی عدد به رشته.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
