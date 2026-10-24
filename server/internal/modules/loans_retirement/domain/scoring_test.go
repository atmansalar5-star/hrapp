// آزمون دامنهٔ ماژول «loans_retirement» (scoring_test).

package domain

import (
	"testing"

	"hrms/internal/contracts"
	"hrms/internal/platform/recordkit"
)

func TestLoanCatalog(t *testing.T) {
	types := LoanTypes()
	if len(types) != 7 {
		t.Fatalf("باید ۷ نوع وام اعلام شود، %d شد", len(types))
	}
	if !IsSupplement("housingDepositSupplement") {
		t.Fatal("متمم ودیعهٔ مسکن باید متمم شناخته شود")
	}
	base, ok := LoanTypeByKey("housingDepositSupplement")
	if !ok || base.BaseKey != "housingDeposit" {
		t.Fatalf("وام پایهٔ متمم ودیعه باید housingDeposit باشد: %+v", base)
	}
	if BankTitleFa("melli") != "بانک ملی" || !ValidBank("mellat") {
		t.Fatal("عنوان/اعتبار بانک عامل درست نیست")
	}
	if ValidBank("unknown-bank") {
		t.Fatal("بانک ناشناخته نباید معتبر باشد")
	}
}

func TestValidateLoanRequiresSupplementBase(t *testing.T) {
	err := validateLoan(contracts.RecordInput{Fields: map[string]string{"loanType": "housingDepositSupplement"}})
	if err == nil {
		t.Fatal("متمم بدون وام پایه باید خطای قاعده بدهد")
	}
	if err := validateLoan(contracts.RecordInput{Fields: map[string]string{
		"loanType": "housingDepositSupplement", "supplementOf": "LOAN-0001",
	}}); err != nil {
		t.Fatalf("متمم با وام پایه باید معتبر باشد، خطا داد: %v", err)
	}
	if err := validateLoan(contracts.RecordInput{Fields: map[string]string{"loanType": "ghost"}}); err == nil {
		t.Fatal("نوع وام ناشناخته باید خطا بدهد")
	}
}

func TestScoringWeightsAndRank(t *testing.T) {
	crit := len(Criteria())
	if crit < 10 {
		t.Fatalf("دست‌کم ۱۰ معیار امتیازبندی لازم است، %d شد", crit)
	}
	a := Applicant{EmployeeID: "EMP-100", EmployeeName: "آزمایشی", Values: map[string]string{
		"education": "phd", "veteran": "freed", "shift": "night", "seniority": "gt15",
	}}
	res := Score(a)
	if len(res.Lines) != crit {
		t.Fatalf("ریز امتیاز باید به اندازهٔ معیارها باشد: %d != %d", len(res.Lines), crit)
	}
	// دکتری ۲۵×۲ + آزاده ۲۵×۳ + شب‌کار ۱۲×۲ + بیش از ۱۵ سال ۲۰×۲
	want := 25*2 + 25*3 + 12*2 + 20*2
	if res.Total != want {
		t.Fatalf("امتیاز کل %d بود، انتظار %d", res.Total, want)
	}

	ranked := Rank([]Applicant{
		{EmployeeID: "low", Values: map[string]string{"education": "diploma"}},
		{EmployeeID: "high", Values: map[string]string{"education": "phd", "veteran": "freed", "shift": "night"}},
	})
	if ranked[0].EmployeeID != "high" {
		t.Fatalf("متقاضی با امتیاز بیشتر باید اول باشد: %+v", ranked[0])
	}
}

func TestSeedCoversAllLoanTypes(t *testing.T) {
	seen := map[string]bool{}
	for _, rec := range Seed() {
		if rec.Kind == "loans" {
			lt := rec.Fields["loanType"]
			if !ValidLoanType(lt) {
				t.Fatalf("نوع وام نمونه «%s» در فهرست نیست", lt)
			}
			seen[lt] = true
		}
	}
	for _, lt := range LoanTypes() {
		if !seen[lt.Key] {
			t.Fatalf("برای نوع وام «%s» رکورد نمونه ثبت نشده", lt.Key)
		}
	}
	// قاعدهٔ متمم باید در دادهٔ نمونه هم رعایت شده باشد.
	for _, rec := range Seed() {
		if rec.Kind == "loans" && IsSupplement(rec.Fields["loanType"]) && rec.Fields["supplementOf"] == "" {
			t.Fatalf("متمم نمونه بدون وام پایه: %s", rec.Title)
		}
	}
}

// اطمینان از اینکه recordkit واقعاً فیلدهای وام را می‌شناسد (اعلام درست دامنه).
func TestDescriptorDeclaresLoanFields(t *testing.T) {
	kind, ok := Descriptor().KindByKey("loans")
	if !ok {
		t.Fatal("نوع رکورد loans در شناسهٔ ماژول نیست")
	}
	need := []string{"loanType", "bank", "feeRate", "feeAmount", "installmentAmount", "supplementOf", "guarantorName", "guarantorNationalId", "guarantorRelation", "receivedDate"}
	have := map[string]bool{}
	for _, f := range kind.Fields {
		have[f.Key] = true
	}
	for _, k := range need {
		if !have[k] {
			t.Fatalf("فیلد «%s» در دامنهٔ وام اعلام نشده", k)
		}
	}
	var _ recordkit.Kind = kind
}
