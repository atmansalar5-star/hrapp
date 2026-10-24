// قاعدهٔ محاسباتی موتور قواعد — «formula».

package rules

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"unicode"
)

// FormulaError خطای ارزیابی فرمول پارامتریک (متغیر ناشناخته، نماد نامعتبر و …).
type FormulaError struct{ Msg string }

func (e *FormulaError) Error() string { return e.Msg }

// EvalFormula یک عبارت حسابی پارامتریک را با متغیرهای داده‌شده حساب می‌کند.
//
// فقط اعداد، متغیرها (حرف لاتین، رقم، نقطه و زیرخط)، پرانتز و عملگرهای
// `+ - * / ^` پشتیبانی می‌شوند. هیچ کد یا تابعی اجرا نمی‌شود؛ پس فرمولی که کاربر در
// «قلم حقوق» ثبت می‌کند نمی‌تواند به سامانه دست بزند و نتیجه قابل پیش‌بینی می‌ماند.
// اگر فرمول به متغیری ارجاع دهد که در vars نیست، خطا برمی‌گردد تا caller بتواند
// صادقانه «حل‌نشده» را گزارش کند و کاربر متغیر گمشده را ببیند.
func EvalFormula(expr string, vars map[string]float64) (float64, error) {
	p := &formulaParser{src: expr, vars: vars}
	v, err := p.parseExpr()
	if err != nil {
		return 0, err
	}
	p.skipSpace()
	if p.pos < len(p.src) {
		return 0, &FormulaError{Msg: fmt.Sprintf("نماد ناشناخته در فرمول: %q", p.src[p.pos:])}
	}
	return v, nil
}

// FormulaVariables فهرست یکتای متغیرهای یک فرمول را برمی‌گرداند؛ برای پیام خطای
// دقیق و نمایش «متغیرهای لازم» یک قلم فرمولی به کار می‌رود.
func FormulaVariables(expr string) []string {
	seen := map[string]bool{}
	out := []string{}
	for i := 0; i < len(expr); {
		if isIdentStart(rune(expr[i])) {
			j := i
			for j < len(expr) && isIdentPart(rune(expr[j])) {
				j++
			}
			name := expr[i:j]
			if !seen[name] {
				seen[name] = true
				out = append(out, name)
			}
			i = j
			continue
		}
		i++
	}
	sort.Strings(out)
	return out
}

// formulaParser یک تحلیل‌گر نزولیِ بازگشتی روی عبارت؛ هیچ state بیرونی ندارد.
type formulaParser struct {
	src  string
	pos  int
	vars map[string]float64
}

func (p *formulaParser) skipSpace() {
	for p.pos < len(p.src) && unicode.IsSpace(rune(p.src[p.pos])) {
		p.pos++
	}
}

func (p *formulaParser) parseExpr() (float64, error) {
	v, err := p.parseTerm()
	if err != nil {
		return 0, err
	}
	for {
		p.skipSpace()
		if p.pos >= len(p.src) {
			return v, nil
		}
		op := p.src[p.pos]
		if op != '+' && op != '-' {
			return v, nil
		}
		p.pos++
		rhs, err := p.parseTerm()
		if err != nil {
			return 0, err
		}
		if op == '+' {
			v += rhs
		} else {
			v -= rhs
		}
	}
}

func (p *formulaParser) parseTerm() (float64, error) {
	v, err := p.parsePower()
	if err != nil {
		return 0, err
	}
	for {
		p.skipSpace()
		if p.pos >= len(p.src) {
			return v, nil
		}
		op := p.src[p.pos]
		if op != '*' && op != '/' {
			return v, nil
		}
		p.pos++
		rhs, err := p.parsePower()
		if err != nil {
			return 0, err
		}
		if op == '*' {
			v *= rhs
		} else {
			if rhs == 0 {
				return 0, &FormulaError{Msg: "تقسیم بر صفر در فرمول"}
			}
			v /= rhs
		}
	}
}

// parsePower توان را راست‌به‌چپ می‌خواند (a^b^c = a^(b^c)).
func (p *formulaParser) parsePower() (float64, error) {
	base, err := p.parseUnary()
	if err != nil {
		return 0, err
	}
	p.skipSpace()
	if p.pos < len(p.src) && p.src[p.pos] == '^' {
		p.pos++
		exp, err := p.parsePower()
		if err != nil {
			return 0, err
		}
		v := math.Pow(base, exp)
		if math.IsInf(v, 0) || math.IsNaN(v) {
			return 0, &FormulaError{Msg: "نتیجهٔ فرمول خارج از محدودهٔ عددی است"}
		}
		return v, nil
	}
	return base, nil
}

func (p *formulaParser) parseUnary() (float64, error) {
	p.skipSpace()
	if p.pos < len(p.src) {
		switch p.src[p.pos] {
		case '+':
			p.pos++
			return p.parseUnary()
		case '-':
			p.pos++
			v, err := p.parseUnary()
			return -v, err
		}
	}
	return p.parsePrimary()
}

func (p *formulaParser) parsePrimary() (float64, error) {
	p.skipSpace()
	if p.pos >= len(p.src) {
		return 0, &FormulaError{Msg: "فرمول ناقص است"}
	}
	c := p.src[p.pos]
	switch {
	case c == '(':
		p.pos++
		v, err := p.parseExpr()
		if err != nil {
			return 0, err
		}
		p.skipSpace()
		if p.pos >= len(p.src) || p.src[p.pos] != ')' {
			return 0, &FormulaError{Msg: "پرانتز بسته در فرمول نیست"}
		}
		p.pos++
		return v, nil
	case (c >= '0' && c <= '9') || c == '.':
		return p.parseNumber()
	case isIdentStart(rune(c)):
		return p.parseIdent()
	default:
		return 0, &FormulaError{Msg: fmt.Sprintf("نماد غیرمجاز در فرمول: %q", string(c))}
	}
}

func (p *formulaParser) parseNumber() (float64, error) {
	start := p.pos
	for p.pos < len(p.src) {
		c := p.src[p.pos]
		if (c >= '0' && c <= '9') || c == '.' {
			p.pos++
			continue
		}
		break
	}
	raw := p.src[start:p.pos]
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, &FormulaError{Msg: fmt.Sprintf("عدد نامعتبر در فرمول: %q", raw)}
	}
	return v, nil
}

func (p *formulaParser) parseIdent() (float64, error) {
	start := p.pos
	for p.pos < len(p.src) && isIdentPart(rune(p.src[p.pos])) {
		p.pos++
	}
	name := p.src[start:p.pos]
	v, ok := p.vars[name]
	if !ok {
		return 0, &FormulaError{Msg: fmt.Sprintf("متغیر ناشناخته در فرمول: %s", name)}
	}
	return v, nil
}

func isIdentStart(c rune) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isIdentPart(c rune) bool {
	return isIdentStart(c) || c == '.' || (c >= '0' && c <= '9')
}
