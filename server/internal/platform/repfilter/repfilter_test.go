// آزمون زیرساخت «repfilter» در پلتفرم سرور — «repfilter_test».

package repfilter

import (
	"testing"

	"hrms/internal/contracts"
)

func TestValidateRejectsUnknownOperator(t *testing.T) {
	err := Validate([]contracts.ReportCondition{{Field: "amount", Op: "wat", Value: "1"}})
	if err == nil {
		t.Fatal("عملگر ناشناخته باید خطا بدهد")
	}
}

func TestValidateRejectsEmptyValue(t *testing.T) {
	if err := Validate([]contracts.ReportCondition{{Field: "amount", Op: OpEq}}); err == nil {
		t.Fatal("مقدار خالی برای عملگر «برابر» باید خطا بدهد")
	}
}

func TestMatchesNumericAndText(t *testing.T) {
	values := map[string]string{"amount": "5000000", "typeFa": "وام مددکاری", "date": "1404/07/01"}

	cases := []struct {
		name string
		cond contracts.ReportCondition
		want bool
	}{
		{"برابر عددی", contracts.ReportCondition{Field: "amount", Op: OpEq, Value: "5000000"}, true},
		{"بزرگ‌تر", contracts.ReportCondition{Field: "amount", Op: OpGt, Value: "1000000"}, true},
		{"کوچک‌تر نادرست", contracts.ReportCondition{Field: "amount", Op: OpLt, Value: "1000000"}, false},
		{"شامل متن", contracts.ReportCondition{Field: "typeFa", Op: OpContains, Value: "مددکاری"}, true},
		{"بین بازه", contracts.ReportCondition{Field: "amount", Op: OpBetween, Value: "1000000", Value2: "9000000"}, true},
		{"یکی از مقادیر", contracts.ReportCondition{Field: "typeFa", Op: OpIn, Value: "وام بانکی, وام مددکاری"}, true},
		{"تاریخ بزرگ‌تر", contracts.ReportCondition{Field: "date", Op: OpGte, Value: "1404/06/01"}, true},
		{"مقدار دارد", contracts.ReportCondition{Field: "typeFa", Op: OpNotEmpty}, true},
		{"نابرابر", contracts.ReportCondition{Field: "typeFa", Op: OpNe, Value: "وام بانکی"}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Matches(values, []contracts.ReportCondition{c.cond}); got != c.want {
				t.Fatalf("Matches=%v، انتظار %v", got, c.want)
			}
		})
	}
}

func TestMatchesAllConditions(t *testing.T) {
	values := map[string]string{"amount": "5000000", "typeFa": "وام مددکاری", "bank": ""}
	conds := []contracts.ReportCondition{
		{Field: "typeFa", Op: OpContains, Value: "مددکاری"},
		{Field: "amount", Op: OpGte, Value: "5000000"},
	}
	if !Matches(values, conds) {
		t.Fatal("وقتی همهٔ شرط‌ها برقرارند نتیجه باید درست باشد")
	}
	conds = append(conds, contracts.ReportCondition{Field: "bank", Op: OpNotEmpty})
	if Matches(values, conds) {
		t.Fatal("با یک شرط نابرقرار، نتیجه باید نادرست باشد")
	}
}
