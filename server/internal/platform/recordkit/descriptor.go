// Package recordkit موتور مشترک «رکورد گردش‌کاری» برای ماژول‌های اداری سامانه.
//
// چرا این بسته وجود دارد: ماژول‌هایی مثل انتصابات، ترفیعات، بایگانی، انضباط کار و
// مطالعات منابع انسانی همه یک الگو دارند — رکوردی با چند فیلد، یک وضعیت، گردش‌کار
// تأیید و سابقهٔ تغییرات. به‌جای تکرار یک برش عمودی برای هر کدام، **قواعد هر ماژول در
// بستهٔ domain خودش اعلام می‌شود** (فیلدها، وضعیت‌ها، انتقال‌های مجاز، اقدام‌ها) و این
// بسته همان قواعد را اجرا می‌کند: اعتبارسنجی فیلدها، کنترل انتقال وضعیت، ثبت رخداد،
// محدودهٔ دسترسی نقش و ذخیره‌سازی (حافظه یا Postgres).
//
// قاعدهٔ معماری حفظ می‌شود: هیچ قاعدهٔ کسب‌وکاری اینجا هاردکد نیست؛ این بسته فقط
// «موتور» است و ورودی‌اش شناسهٔ اعلامیِ هر ماژول. UI هم هیچ برچسب یا فیلدی را از خودش
// نمی‌سازد و شناسهٔ ماژول را از سرور می‌خواند.
package recordkit

import (
	"fmt"
	"strings"

	"hrms/internal/contracts"
	"hrms/internal/platform/calendar"
)

// Field یک فیلد رکورد؛ Type یکی از text|date|number|money|select.
type Field struct {
	Key      string
	LabelFa  string
	Type     string
	Required bool
}

// Status یک وضعیت گردش‌کار.
type Status struct {
	Value   string
	LabelFa string
}

// Action یک اقدام ممکن روی رکورد.
//
//   - با To مقصد، وضعیت را عوض می‌کند (تأیید، رد، بستن، اجرا).
//   - Style فقط ظاهر دکمه را تعیین می‌کند (primary|ghost|danger) و هیچ منطقی ندارد.
//   - NoteRequired یعنی انجام این اقدام بدون یادداشت مجاز نیست (مثل رد یا لغو).
//   - Roles مرحلهٔ گردش‌کار را می‌سازد: اگر غیرخالی باشد، تنها همین نقش‌ها (یا مدیر
//     سیستم) مجاز به انجام این اقدام‌اند — یعنی «چه کسی این مرحله را می‌بندد» در دامنه
//     اعلام می‌شود، نه در UI. خالی یعنی هر نقشی که در سطح ماژول اجازهٔ اقدام دارد.
//   - Scope حداقل دامنهٔ دادهٔ لازم برای این اقدام: all|own_unit|own. مثلاً تأیید مدیر
//     واحد فقط روی رکورد واحد خودش (own_unit) مجاز است.
type Action struct {
	Value        string
	LabelFa      string
	Style        string
	To           string
	NoteRequired bool
	Roles        []contracts.Role
	Scope        string
}

// Actor کنشگر گردش‌کار در لبهٔ موتور: نقش و دامنهٔ سازمانی تصمیم‌گیرنده. موتور تنها از
// همین ساختار برای سنجش «آیا این اقدام برای این نقش مجاز است؟» استفاده می‌کند.
type Actor struct {
	Username     string
	Role         contracts.Role
	DepartmentID string
	EmployeeID   string
}

// Kind یک نوع رکورد در ماژول (مثلاً «انتصاب»، «ترفیع»، «پروندهٔ بایگانی»).
//
// Fields فیلدهای اختصاصی همین نوع‌اند؛ فیلدهای مشترک (عنوان، کارمند، یادداشت) در
// لایهٔ بالاتر و به‌صورت ثابت مدیریت می‌شوند تا در هر ماژول تکرار نشوند.
type Kind struct {
	Key            string
	TitleFa        string
	IDPrefix       string
	Fields         []Field
	Statuses       []Status
	Initial        string
	Transitions    map[string][]Status // وضعیت فعلی → مقصدهای مجاز (با برچسب فارسی)
	Actions        []Action
	EmployeeScoped bool
	// Validate قاعدهٔ اختصاصی همین نوع رکورد (اختیاری). پس از اعتبارسنجی عمومی و پیش
	// از ذخیره صدا زده می‌شود؛ اگر خطا برگرداند، ذخیره نمی‌شود. قاعده در بستهٔ domain
	// خودِ ماژول نوشته می‌شود تا موتور recordkit هیچ منطق کسب‌وکاری نداشته باشد.
	Validate func(in contracts.RecordInput) error
	// EditableStatuses وضعیت‌هایی که ویرایش در آن‌ها مجاز است (پیش‌فرض: وضعیت آغازین).
	EditableStatuses []string
	// HistoryDate کلید فیلد «تاریخ مؤثر» همین نوع برای سابقهٔ کارمند. خالی یعنی
	// نخستین فیلد از نوع date (مثل تاریخ اجرا، تاریخ شروع، تاریخ مدرک) و اگر هیچ فیلد
	// تاریخی نبود، زمان ثبت/ویرایش رکورد مبنای چیدن زمانی می‌شود.
	HistoryDate string
}

// Descriptor شناسهٔ یک ماژول: انواع رکوردی که مدیریت می‌کند.
type Descriptor struct {
	Module    string
	TitleFa   string
	Kinds     []Kind
	Summaries []string
}

// ValidationError خطای اعتبارسنجی فیلدبه‌فیلد.
type ValidationError struct{ Details map[string]string }

func (e *ValidationError) Error() string { return "اطلاعات ارسالی معتبر نیست." }

// RuleError نقض یک قاعدهٔ کسب‌وکار.
type RuleError struct{ Message string }

func (e *RuleError) Error() string { return e.Message }

// KindByKey یافتن نوع رکورد با کلید (کلید ماشین‌خوان یا عنوان فارسی).
func (d Descriptor) KindByKey(key string) (Kind, bool) {
	for _, k := range d.Kinds {
		if k.Key == key {
			return k, true
		}
	}
	return Kind{}, false
}

// Schema شناسهٔ ماژول را برای کلاینت می‌سازد تا UI خودش فرم و جدول بسازد.
func (d Descriptor) Schema() contracts.RecordSchema {
	kinds := make([]contracts.RecordKindSchema, 0, len(d.Kinds))
	for _, k := range d.Kinds {
		fields := make([]contracts.RecordField, 0, len(k.Fields))
		for _, f := range k.Fields {
			fields = append(fields, contracts.RecordField{Key: f.Key, LabelFa: f.LabelFa, Type: f.Type, Required: f.Required})
		}
		statuses := make([]contracts.RecordStatus, 0, len(k.Statuses))
		for _, s := range k.Statuses {
			statuses = append(statuses, contracts.RecordStatus{Value: s.Value, LabelFa: s.LabelFa})
		}
		actions := make([]contracts.RecordActionDef, 0, len(k.Actions))
		for _, a := range k.Actions {
			actions = append(actions, contracts.RecordActionDef{
				Value: a.Value, LabelFa: a.LabelFa, Style: a.Style,
				To: a.To, NoteRequired: a.NoteRequired,
				Roles: roleKeys(a.Roles), Scope: a.Scope,
			})
		}
		transitions := make(map[string][]string, len(k.Transitions))
		for from, targets := range k.Transitions {
			values := make([]string, 0, len(targets))
			for _, t := range targets {
				values = append(values, t.Value)
			}
			transitions[from] = values
		}
		// وضعیت‌های مجاز ویرایش را هم به کلاینت می‌دهیم؛ خالی یعنی همان وضعیت آغازین.
		editable := k.EditableStatuses
		if len(editable) == 0 {
			editable = []string{k.Initial}
		}
		kinds = append(kinds, contracts.RecordKindSchema{
			Key: k.Key, TitleFa: k.TitleFa, IDPrefix: k.IDPrefix,
			Fields: fields, Statuses: statuses, Actions: actions, Transitions: transitions,
			Workflow: k.WorkflowSteps(), InitialStatus: k.Initial,
			EditableStatuses: editable, EmployeeScoped: k.EmployeeScoped,
		})
	}
	return contracts.RecordSchema{Module: d.Module, ModuleTitleFa: d.TitleFa, Kinds: kinds}
}

/* ------------------------------ اعتبارسنجی ------------------------------ */

// StatusFa عنوان فارسی یک وضعیت.
func (k Kind) StatusFa(value string) string {
	for _, s := range k.Statuses {
		if s.Value == value {
			return s.LabelFa
		}
	}
	return value
}

// ActionByValue یافتن اقدام با مقدار.
func (k Kind) ActionByValue(value string) (Action, bool) {
	for _, a := range k.Actions {
		if a.Value == value {
			return a, true
		}
	}
	return Action{}, false
}

// HistoryDateField کلید و برچسب فارسی فیلد تاریخ مؤثر این نوع رکورد برای سابقه.
//
// قاعدهٔ انتخاب: فیلد اعلامی HistoryDate، وگرنه نخستین فیلد از نوع date. این انتخاب در
// دامنه اعلام می‌شود (ترتیب فیلدها یا اعلام صریح) و در رابط کاربری بازنویسی نمی‌شود.
func (k Kind) HistoryDateField() (key, labelFa string) {
	if declared := strings.TrimSpace(k.HistoryDate); declared != "" {
		for _, f := range k.Fields {
			if f.Key == declared {
				return f.Key, f.LabelFa
			}
		}
	}
	for _, f := range k.Fields {
		if f.Type == "date" {
			return f.Key, f.LabelFa
		}
	}
	return "", ""
}

// CanTransition بررسی مجاز بودن انتقال وضعیت.
func (k Kind) CanTransition(from string, a Action) bool {
	if a.To == "" {
		return false
	}
	target, ok := k.Transitions[from]
	if !ok {
		return false
	}
	for _, t := range target {
		if t.Value == a.To {
			return true
		}
	}
	return false
}

// CanAct بررسی می‌کند آیا این کنشگر می‌تواند این اقدام را از این وضعیت انجام دهد.
// سه شرط: انتقال وضعیت مجاز باشد، نقش در مرحلهٔ اقدام مجاز باشد (اگر اعلام شده) و دامنهٔ
// سازمانی کنشگر با دامنهٔ لازم اقدام بخواند. مدیر سیستم همیشه مجاز است.
//
// دامنهٔ رکورد به‌صورت رشته‌های مقایسه‌شدنی به این تابع می‌آید تا این لایه هیچ شناختی از
// ساختار انبار نداشته باشد.
func (k Kind) CanAct(status string, a Action, actor Actor, recordEmployeeID, recordDepartment string) bool {
	if !k.CanTransition(status, a) {
		return false
	}
	if !roleAllowed(a, actor) {
		return false
	}
	return scopeAllows(a.Scope, actor, recordEmployeeID, recordDepartment)
}

// roleAllowed آیا نقش کنشگر در مرحلهٔ اقدام مجاز است. کنشگر ناشناخته (نقش خالی، مثل
// فراخوانی داخلی) محدود نمی‌شود تا بذرکاری و آزمون‌های بدون هویت نشکنند.
func roleAllowed(a Action, actor Actor) bool {
	if len(a.Roles) == 0 || actor.Role == "" || actor.Role == contracts.RoleAdmin {
		return true
	}
	for _, r := range a.Roles {
		if r == actor.Role {
			return true
		}
	}
	return false
}

// scopeAllows سنجش دامنهٔ داده: all آزاد، own_unit محدود به واحد کنشگر و own محدود به
// خودِ کارمند. دامنهٔ خالی یا کنشگر بی‌شناسه محدود نمی‌شود (سازگاری با نقش‌های سراسری).
func scopeAllows(scope string, actor Actor, recordEmployeeID, recordDepartment string) bool {
	switch scope {
	case "own":
		return actor.EmployeeID == "" || recordEmployeeID == actor.EmployeeID
	case "own_unit":
		return actor.DepartmentID == "" || recordDepartment == actor.DepartmentID || recordDepartment == ""
	default:
		return true
	}
}

// AvailableActions اقدام‌هایی که این کنشگر در این وضعیت مجاز است.
func (k Kind) AvailableActions(status string, actor Actor, recordEmployeeID, recordDepartment string) []Action {
	out := make([]Action, 0, len(k.Actions))
	for _, a := range k.Actions {
		if k.CanAct(status, a, actor, recordEmployeeID, recordDepartment) {
			out = append(out, a)
		}
	}
	return out
}

// Editable بررسی مجاز بودن ویرایش در یک وضعیت.
func (k Kind) Editable(status string) bool {
	if len(k.EditableStatuses) == 0 {
		return status == k.Initial
	}
	for _, s := range k.EditableStatuses {
		if s == status {
			return true
		}
	}
	return false
}

// ValidateInput اعتبارسنجی ورودی ایجاد/ویرایش بر پایهٔ اعلام ماژول.
// فیلدهای اجباری و قالب تاریخ (شمسی) اینجا بررسی می‌شوند؛ هیچ قاعده‌ای در UI نیست.
func (k Kind) ValidateInput(in contracts.RecordInput, partial bool) map[string]string {
	errs := map[string]string{}
	if !partial && strings.TrimSpace(in.Title) == "" {
		errs["title"] = "عنوان الزامی است."
	}
	if k.EmployeeScoped && !partial && strings.TrimSpace(in.EmployeeID) == "" {
		errs["employeeId"] = "انتخاب کارمند الزامی است."
	}
	for _, f := range k.Fields {
		v := strings.TrimSpace(in.Fields[f.Key])
		if v == "" {
			if f.Required && !partial {
				errs["fields."+f.Key] = fmt.Sprintf("«%s» الزامی است.", f.LabelFa)
			}
			continue
		}
		switch f.Type {
		case "date":
			if _, err := calendar.Parse(v); err != nil {
				errs["fields."+f.Key] = fmt.Sprintf("«%s» باید به قالب شمسی YYYY/MM/DD باشد.", f.LabelFa)
			}
		case "number", "money":
			if !isNumeric(v) {
				errs["fields."+f.Key] = fmt.Sprintf("«%s» باید عدد باشد.", f.LabelFa)
			}
		}
	}
	return errs
}

/* ------------------- سازنده‌های فشردهٔ اعلام قواعد ------------------- */
//
// این کمک‌ها به هر ماژول اجازه می‌دهند قواعدش را در چند خط و خوانا اعلام کند؛
// قالب رشته‌ها مستند است و در خود ماژول (بستهٔ domain) نوشته می‌شود.

// Statuses از قالب «value|برچسب فارسی» وضعیت‌ها را می‌سازد.
func Statuses(specs ...string) []Status {
	out := make([]Status, 0, len(specs))
	for _, s := range specs {
		parts := strings.SplitN(s, "|", 2)
		st := Status{Value: strings.TrimSpace(parts[0])}
		if len(parts) == 2 {
			st.LabelFa = strings.TrimSpace(parts[1])
		} else {
			st.LabelFa = st.Value
		}
		out = append(out, st)
	}
	return out
}

// Fields از قالب «key|برچسب فارسی|type|required» فیلدها را می‌سازد.
// type یکی از text|date|number|money|select و required با 1/0 مشخص می‌شود.
func Fields(specs ...string) []Field {
	out := make([]Field, 0, len(specs))
	for _, s := range specs {
		parts := strings.Split(s, "|")
		f := Field{Key: strings.TrimSpace(parts[0])}
		if len(parts) > 1 {
			f.LabelFa = strings.TrimSpace(parts[1])
		} else {
			f.LabelFa = f.Key
		}
		if len(parts) > 2 {
			f.Type = strings.TrimSpace(parts[2])
		}
		if f.Type == "" {
			f.Type = "text"
		}
		if len(parts) > 3 {
			f.Required = strings.TrimSpace(parts[3]) == "1"
		}
		out = append(out, f)
	}
	return out
}

// Actions از قالب «value|برچسب فارسی|style|مقصد|noteRequired|roles|scope» اقدام‌ها را
// می‌سازد. مقصد خالی یعنی اقدام وضعیت را عوض نمی‌کند و فقط رخداد ثبت می‌شود.
// roles فهرست نقش‌های مجاز این مرحله با کاما (خالی = همهٔ نقش‌های دارای دسترسی اقدام) و
// scope حداقل دامنهٔ لازم (all|own_unit|own) است.
func Actions(specs ...string) []Action {
	out := make([]Action, 0, len(specs))
	for _, s := range specs {
		parts := strings.Split(s, "|")
		a := Action{Value: strings.TrimSpace(parts[0])}
		if len(parts) > 1 {
			a.LabelFa = strings.TrimSpace(parts[1])
		} else {
			a.LabelFa = a.Value
		}
		if len(parts) > 2 && strings.TrimSpace(parts[2]) != "" {
			a.Style = strings.TrimSpace(parts[2])
		} else {
			a.Style = "ghost"
		}
		if len(parts) > 3 {
			a.To = strings.TrimSpace(parts[3])
		}
		if len(parts) > 4 {
			a.NoteRequired = strings.TrimSpace(parts[4]) == "1"
		}
		if len(parts) > 5 {
			a.Roles = parseRoles(parts[5])
		}
		if len(parts) > 6 {
			a.Scope = strings.TrimSpace(parts[6])
		}
		out = append(out, a)
	}
	return out
}

// parseRoles فهرست نقش‌های جدا‌شده با کاما را به نقش‌های معتبر تبدیل می‌کند.
func parseRoles(spec string) []contracts.Role {
	out := []contracts.Role{}
	for _, chunk := range strings.Split(spec, ",") {
		key := contracts.Role(strings.TrimSpace(chunk))
		if key == "" || !contracts.ValidRole(key) {
			continue
		}
		out = append(out, key)
	}
	return out
}

// roleKeys نقش‌های یک اقدام را برای DTO به رشته تبدیل می‌کند.
func roleKeys(roles []contracts.Role) []string {
	if len(roles) == 0 {
		return nil
	}
	out := make([]string, 0, len(roles))
	for _, r := range roles {
		out = append(out, string(r))
	}
	return out
}

// WorkflowSteps زنجیرهٔ مرحله‌های تأیید این نوع رکورد را می‌سازد: هر وضعیتی که از آن
// دست‌کم یک اقدام در وضعیت را عوض می‌کند، یک مرحله است؛ نقش‌های مرحله اجتماع نقش‌های
// اقدام‌های همان وضعیت است. UI از همین زنجیره نمودار «گردش‌کار» را می‌سازد.
func (k Kind) WorkflowSteps() []contracts.RecordWorkflowStep {
	steps := []contracts.RecordWorkflowStep{}
	for _, st := range k.Statuses {
		actions := []string{}
		roles := []string{}
		seen := map[contracts.Role]bool{}
		for _, a := range k.Actions {
			if !k.CanTransition(st.Value, a) {
				continue
			}
			actions = append(actions, a.Value)
			for _, r := range a.Roles {
				if !seen[r] {
					seen[r] = true
					roles = append(roles, string(r))
				}
			}
		}
		if len(actions) == 0 {
			continue
		}
		steps = append(steps, contracts.RecordWorkflowStep{Status: st.Value, StatusFa: st.LabelFa, Roles: roles, Actions: actions})
	}
	return steps
}

// Steps از قالب «وضعیت|مقصد۱,مقصد۲» جدول انتقال‌های مجاز را می‌سازد.
func Steps(specs ...string) map[string][]Status {
	out := map[string][]Status{}
	for _, s := range specs {
		parts := strings.SplitN(s, "|", 2)
		from := strings.TrimSpace(parts[0])
		targets := []Status{}
		if len(parts) == 2 {
			for _, t := range strings.Split(parts[1], ",") {
				t = strings.TrimSpace(t)
				if t != "" {
					targets = append(targets, Status{Value: t})
				}
			}
		}
		out[from] = targets
	}
	return out
}

func isNumeric(v string) bool {
	v = calendar.NormalizeDigits(strings.TrimSpace(v))
	if v == "" {
		return false
	}
	for _, c := range v {
		if (c < '0' || c > '9') && c != '.' && c != '-' {
			return false
		}
	}
	return true
}
