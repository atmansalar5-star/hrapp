// فایل پرداخت بانکی — زیرکار ۱.۵ فاز ۱ نقشهٔ راه.
//
// قاعده‌های این فایل: گرد کردن مبلغ هر کارمند به مضرب پارامتریک (پیش‌فرض ۱۰۰۰ ریال) و
// نگه‌داشتن جمع تفاوتِ گرد کردن، جمع کنترلی (شمار ردیف + جمع مبلغ) برای مقایسه با رسید
// بانک، سنجش شمارهٔ شبا (IR + ۲۴ رقم با رقم کنترل استاندارد) و پیگیری واریزهای برگشتی.
//
// حساب‌های بانکی کارکنان دادهٔ رکوردی ماژول حقوق است (`bankAccounts`)؛ کارمندی که حساب
// فعال ندارد در گزارش «بدون حساب» می‌آید و ردیف پرداخت برایش ساخته نمی‌شود.
package domain

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"hrms/internal/contracts"
	"hrms/internal/platform/calendar"
	"hrms/internal/platform/recordkit"
)

// DefaultBankRoundUnit مبنای گرد کردن پیش‌فرض مبلغ پرداخت (ریال).
const DefaultBankRoundUnit int64 = 1000

// BankRow یک ردیف فایل پرداخت بانکی.
type BankRow struct {
	RowNo        int    `json:"rowNo"`
	EmployeeID   string `json:"employeeId"`
	EmployeeName string `json:"employeeName"`
	NationalID   string `json:"nationalId,omitempty"`
	BankNameFa   string `json:"bankNameFa"`
	IBAN         string `json:"iban"`
	AccountNo    string `json:"accountNo,omitempty"`
	// Amount مبلغ پرداختی (پس از گرد کردن) و RawAmount مبلغ محاسبه‌شدهٔ پیش از گرد کردن.
	Amount    int64 `json:"amount"`
	RawAmount int64 `json:"rawAmount"`
	Rounding  int64 `json:"rounding"`
	// StatusFa وضعیت واریز: آماده، واریزشده، برگشتی (برای پیگیری مغایرت بانکی).
	StatusFa string   `json:"statusFa"`
	IssuesFa []string `json:"issuesFa,omitempty"`
}

// BankTotals جمع‌های کنترلی فایل پرداخت.
type BankTotals struct {
	Rows          int   `json:"rows"`
	Amount        int64 `json:"amount"`
	RawAmount     int64 `json:"rawAmount"`
	Rounding      int64 `json:"rounding"`
	ReturnedRows  int   `json:"returnedRows"`
	ReturnedSum   int64 `json:"returnedAmount"`
	MissingInputs int   `json:"missingInputs"`
}

// BankFile فایل پرداخت بانکی یک دوره برای یک بانک.
type BankFile struct {
	Period     string     `json:"period"`
	BankNameFa string     `json:"bankNameFa"`
	ColumnsFa  []string   `json:"columnsFa"`
	Rows       []BankRow  `json:"rows"`
	Totals     BankTotals `json:"totals"`
	// ControlFa جمع کنترلی برای مقایسه با رسید بانک.
	ControlFa   string   `json:"controlFa"`
	RoundUnit   int64    `json:"roundUnit"`
	IssuesFa    []string `json:"issuesFa,omitempty"`
	MissingFa   []string `json:"missingAccountsFa,omitempty"`
	CreatedAtFa string   `json:"createdAtFa,omitempty"`
	NoteFa      string   `json:"noteFa,omitempty"`
}

// BankInput ورودی یک ردیف پرداخت (از کارکرد/ماشین حقوق).
type BankInput struct {
	EmployeeID   string
	EmployeeName string
	NationalID   string
	BankNameFa   string
	IBAN         string
	AccountNo    string
	Amount       int64
	Returned     bool
}

// BankColumns ستون‌های فایل پرداخت به ترتیب خروجی.
func BankColumns() []string {
	return []string{"ردیف", "کد پرسنلی", "نام کارمند", "کد ملی", "بانک", "شمارهٔ شبا (IR)", "مبلغ (ریال)", "وضعیت"}
}

// BankRowValues مقادیر یک ردیف به ترتیب ستون‌ها.
func BankRowValues(row BankRow) []string {
	return []string{
		strconv.Itoa(row.RowNo), row.EmployeeID, row.EmployeeName, row.NationalID,
		row.BankNameFa, row.IBAN, strconv.FormatInt(row.Amount, 10), row.StatusFa,
	}
}

// BankFooter ردیف جمع پایانی فایل.
func BankFooter(file BankFile) [][]string {
	return [][]string{
		{"", "", "جمع کل", "", file.BankNameFa, strconv.Itoa(file.Totals.Rows) + " ردیف",
			strconv.FormatInt(file.Totals.Amount, 10), ""},
		{"", "", "جمع تفاوت گرد کردن", "", "", "", strconv.FormatInt(file.Totals.Rounding, 10), ""},
	}
}

// ValidateIBAN شمارهٔ شبا ایران را می‌سنجد: «IR» + ۲۴ رقم و رقم کنترل استاندارد (mod ۹۷).
func ValidateIBAN(raw string) error {
	code := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(calendar.NormalizeDigits(raw)), " ", ""))
	if code == "" {
		return fmt.Errorf("شمارهٔ شبا خالی است")
	}
	if !strings.HasPrefix(code, "IR") {
		return fmt.Errorf("شمارهٔ شبا باید با IR آغاز شود («%s»)", code)
	}
	body := code[2:]
	if len(body) != 24 {
		return fmt.Errorf("شمارهٔ شبا باید ۲۴ رقم پس از IR داشته باشد («%s»)", code)
	}
	for _, ch := range body {
		if ch < '0' || ch > '9' {
			return fmt.Errorf("شمارهٔ شبا باید فقط رقم باشد («%s»)", code)
		}
	}
	// الگوریتم استاندارد: جابه‌جایی چهار نویسهٔ نخست به پایان، تبدیل حروف به عدد و mod ۹۷.
	rearranged := body + "1827" // I=18، R=27
	remainder := 0
	for _, ch := range rearranged {
		remainder = (remainder*10 + int(ch-'0')) % 97
	}
	if remainder != 1 {
		return fmt.Errorf("رقم کنترل شمارهٔ شبا درست نیست («%s»)", code)
	}
	return nil
}

// BuildBankFile ردیف‌های پرداخت را می‌سازد: گرد کردن، جمع کنترلی، سنجش شبا و برگشتی‌ها.
func BuildBankFile(period, bankNameFa string, inputs []BankInput, roundUnit int64, today string) BankFile {
	if roundUnit <= 0 {
		roundUnit = DefaultBankRoundUnit
	}
	file := BankFile{Period: period, BankNameFa: bankNameFa, RoundUnit: roundUnit, CreatedAtFa: today}
	file.ColumnsFa = BankColumns()

	sort.SliceStable(inputs, func(i, j int) bool {
		if inputs[i].EmployeeID != inputs[j].EmployeeID {
			return inputs[i].EmployeeID < inputs[j].EmployeeID
		}
		return inputs[i].EmployeeName < inputs[j].EmployeeName
	})

	seenIBAN := map[string]bool{}
	for _, in := range inputs {
		row := BankRow{
			RowNo:      len(file.Rows) + 1,
			EmployeeID: in.EmployeeID, EmployeeName: in.EmployeeName,
			NationalID: strings.TrimSpace(calendar.NormalizeDigits(in.NationalID)),
			BankNameFa: strings.TrimSpace(in.BankNameFa), IBAN: strings.ToUpper(strings.TrimSpace(in.IBAN)),
			AccountNo: strings.TrimSpace(in.AccountNo), RawAmount: in.Amount,
			StatusFa: "آمادهٔ پرداخت",
		}
		row.Amount, row.Rounding = roundTo(in.Amount, roundUnit)
		if in.Returned {
			row.StatusFa = "برگشتی — نیازمند پیگیری"
			file.Totals.ReturnedRows++
			file.Totals.ReturnedSum += row.Amount
		}
		if in.IBAN == "" {
			row.IssuesFa = append(row.IssuesFa, "شمارهٔ شبا ثبت نشده است؛ واریز ممکن نیست.")
			file.Totals.MissingInputs++
		} else if err := ValidateIBAN(in.IBAN); err != nil {
			row.IssuesFa = append(row.IssuesFa, "مغایرت شمارهٔ شبا: "+err.Error()+".")
		} else if seenIBAN[row.IBAN] {
			row.IssuesFa = append(row.IssuesFa, "این شمارهٔ شبا در همین فایل برای کارمند دیگری هم آمده است؛ واریز تکراری/اشتباه ممکن است.")
		}
		seenIBAN[row.IBAN] = true
		if row.Amount <= 0 {
			row.IssuesFa = append(row.IssuesFa, "مبلغ پرداخت صفر یا منفی است؛ ردیف در فایل بانک نمی‌آید.")
		}
		if row.BankNameFa == "" {
			row.IssuesFa = append(row.IssuesFa, "نام بانک در پروندهٔ حساب ثبت نشده است.")
		}
		if len(row.IssuesFa) > 0 {
			file.IssuesFa = append(file.IssuesFa, fmt.Sprintf("ردیف %d (%s): %s",
				row.RowNo, row.EmployeeName, strings.Join(row.IssuesFa, " ")))
		}
		file.Rows = append(file.Rows, row)
		file.Totals.Rows++
		file.Totals.Amount += row.Amount
		file.Totals.RawAmount += row.RawAmount
		file.Totals.Rounding += row.Rounding
	}
	file.ControlFa = fmt.Sprintf("شمار ردیف: %d | جمع مبلغ: %s ریال | جمع تفاوت گرد کردن: %s ریال",
		file.Totals.Rows, formatFa(file.Totals.Amount), formatFa(file.Totals.Rounding))
	file.NoteFa = bankNote(file, roundUnit)
	return file
}

// BuildBankFileFromAccounts فایل پرداخت را از رکوردهای «حساب بانکی کارمند» می‌سازد.
//
// قاعده: هر کارمند فقط با حساب فعال در فایل می‌آید؛ کارمند بدون حساب در فهرست
// «بدون حساب» گزارش می‌شود تا حقوقش بی‌صدا جا نماند. مبلغ هر کارمند از نقشهٔ مبالغ
// (خروجی محاسبهٔ دوره) می‌آید.
func BuildBankFileFromAccounts(period, bankNameFa string, accounts []contracts.Record, amounts map[string]int64, returned map[string]bool, roundUnit int64, today string) BankFile {
	inputs := make([]BankInput, 0, len(accounts))
	seen := map[string]bool{}
	for _, rec := range accounts {
		if rec.Status != "active" {
			continue
		}
		employeeID := strings.TrimSpace(rec.EmployeeID)
		if employeeID == "" {
			continue
		}
		if bankNameFa != "" && !strings.EqualFold(strings.TrimSpace(rec.Fields["bankName"]), bankNameFa) {
			continue
		}
		amount, ok := amounts[employeeID]
		seen[employeeID] = true
		if !ok || amount <= 0 {
			continue
		}
		inputs = append(inputs, BankInput{
			EmployeeID: employeeID, EmployeeName: rec.EmployeeName,
			NationalID: rec.Fields["nationalId"], BankNameFa: rec.Fields["bankName"],
			IBAN: rec.Fields["iban"], AccountNo: rec.Fields["accountNo"],
			Amount: amount, Returned: returned[employeeID],
		})
	}
	file := BuildBankFile(period, bankNameFa, inputs, roundUnit, today)
	// کارمندانی که مبلغ پرداختی دارند ولی حساب فعال ثبت‌شده ندارند.
	missing := make([]string, 0)
	for employeeID, amount := range amounts {
		if amount <= 0 || seen[employeeID] {
			continue
		}
		missing = append(missing, employeeID)
	}
	sort.Strings(missing)
	for _, employeeID := range missing {
		file.MissingFa = append(file.MissingFa, "کارمند "+employeeID+" حساب بانکی فعال ندارد؛ در فایل پرداخت نیامد.")
	}
	return file
}

// roundTo مبلغ را به مضرب واحد داده‌شده گرد می‌کند و تفاوت را برمی‌گرداند.
func roundTo(amount, unit int64) (int64, int64) {
	if unit <= 1 || amount == 0 {
		return amount, 0
	}
	rounded := int64(float64(amount)/float64(unit)+0.5) * unit
	return rounded, rounded - amount
}

// bankNote یادداشت فارسی فایل پرداخت: جمع کنترلی، برگشتی‌ها و هشدارها.
func bankNote(file BankFile, roundUnit int64) string {
	parts := []string{fmt.Sprintf("مبلغ هر ردیف به مضرب %s ریال گرد شده و جمع تفاوت گرد کردن نگه داشته شده است.",
		formatFa(roundUnit))}
	if file.Totals.ReturnedRows > 0 {
		parts = append(parts, fmt.Sprintf("%d ردیف برگشتی با جمع %s ریال در فایل هست؛ پیگیری و پرداخت مجدد لازم است.",
			file.Totals.ReturnedRows, formatFa(file.Totals.ReturnedSum)))
	}
	if len(file.MissingFa) > 0 {
		parts = append(parts, fmt.Sprintf("%d کارمند حساب بانکی فعال ندارند.", len(file.MissingFa)))
	}
	if len(file.IssuesFa) > 0 {
		parts = append(parts, fmt.Sprintf("%d ردیف مغایرت دارد (شبا/بانک/مبلغ)؛ پیش از ارسال به بانک رفع شود.", len(file.IssuesFa)))
	}
	parts = append(parts, "جمع کنترلی را با رسید بانک مقایسه کنید.")
	return strings.Join(parts, " ")
}

// BankFileKindReady خطای قاعده برای فایل بدون ردیف.
func BankFileKindReady(file BankFile) error {
	if file.Totals.Rows == 0 {
		return &recordkit.RuleError{Message: "هیچ ردیف پرداختی برای این دوره و بانک ساخته نشد؛ حساب بانکی کارکنان یا مبالغ دوره را بررسی کنید."}
	}
	return nil
}
