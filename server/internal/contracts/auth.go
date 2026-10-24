package contracts

// Role نقش‌های سامانه — هم‌خوان با enum در قرارداد OpenAPI و مانیفست ماژول‌ها.
type Role string

const (
	RoleAdmin        Role = "admin"
	RoleHRManager    Role = "hr_manager"
	RoleHRSpecialist Role = "hr_spec"
	RoleLineManager  Role = "line_manager"
	RoleEmployee     Role = "employee"
)

// Roles همهٔ نقش‌های معتبر به ترتیب سند طراحی.
var Roles = []Role{RoleAdmin, RoleHRManager, RoleHRSpecialist, RoleLineManager, RoleEmployee}

// ValidRole بررسی می‌کند نقش در فهرست مجاز است.
func ValidRole(r Role) bool {
	for _, x := range Roles {
		if x == r {
			return true
		}
	}
	return false
}

// RoleFa نام فارسی نقش برای پیام‌ها و مستندات.
var RoleFa = map[Role]string{
	RoleAdmin:        "مدیر سیستم",
	RoleHRManager:    "مدیر منابع انسانی",
	RoleHRSpecialist: "کارشناس منابع انسانی",
	RoleLineManager:  "مدیر واحد",
	RoleEmployee:     "کارمند",
}

// LoginRequest ورودی ورود به سامانه.
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// User کاربر جاری و نقش او.
type User struct {
	Username     string `json:"username"`
	DisplayName  string `json:"displayName"`
	Role         Role   `json:"role"`
	EmployeeID   string `json:"employeeId,omitempty"`
	DepartmentID string `json:"departmentId,omitempty"`
}

// Session نشست صادرشده برای کلاینت.
type Session struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresAt    int64  `json:"expiresAt"`
	User         User   `json:"user"`
}

// RefreshRequest ورودی تمدید نشست.
type RefreshRequest struct {
	RefreshToken string `json:"refreshToken"`
}
