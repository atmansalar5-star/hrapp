"""مرجع مستقل (پایتون) برای تولید fixtureهای طلایی core/testdata/conformance/.

این فایل بخشی از بیلد نیست و فقط برای تولید مقدارهای مورد انتظار استفاده می‌شود؛
مقدارها در فایل‌های JSON کامیت می‌شوند و هم پیاده‌سازی Rust (hrms-rules) و هم Go
(internal/rules) با آن‌ها سنجیده می‌شوند. CLI هسته در CI با --check بازتولید می‌کند
و هر انحرافی را می‌گیرد.

ترتیب کلیدها در هر شیء JSON همان ترتیب اعلان فیلد در structهای Rust است.
اجرا: python tools/check/ref_fixtures.py
"""
import json
import math
import os

# ---------------------------------------------------------------- تقویم شمسی
breaks = [-61, 9, 38, 199, 426, 686, 756, 818, 1111, 1181, 1210, 1635, 2060, 2097, 2192, 2262, 2324, 2394, 2456, 3178]


def div(a, b):
    return int(a / b)


def mod(a, b):
    return a - div(a, b) * b


def jal_cal(jy):
    leapJ = -14
    jp = breaks[0]
    jump = 0
    for i in range(1, len(breaks)):
        jm = breaks[i]
        jump = jm - jp
        if jy < jm:
            break
        leapJ = leapJ + div(jump, 33) * 8 + div(mod(jump, 33), 4)
        jp = jm
    n = jy - jp
    leapJ = leapJ + div(n, 33) * 8 + div(mod(n, 33) + 3, 4)
    if mod(jump, 33) == 4 and jump - n == 4:
        leapJ += 1
    gy = jy + 621
    leapG = div(gy, 4) - div((div(gy, 100) + 1) * 3, 4) - 150
    march = 20 + leapJ - leapG
    if jump - n < 6:
        n = n - jump + div(jump + 4, 33) * 33
    leap = mod(mod(n + 1, 33) - 1, 4)
    if leap == -1:
        leap = 4
    return leap, gy, march


def g2d(gy, gm, gd):
    d = div((gy + div(gm - 8, 6) + 100100) * 1461, 4) + div(153 * mod(gm + 9, 12) + 2, 5) + gd - 34840408
    return d - div(div(gy + 100100 + div(gm - 8, 6), 100) * 3, 4) + 752


def d2g(jdn):
    j = 4 * jdn + 139361631
    j = j + div(div(4 * jdn + 183187720, 146097) * 3, 4) * 4 - 3908
    i = div(mod(j, 1461), 4) * 5 + 308
    return div(j, 1461) - 100100 + div(8 - (mod(div(i, 153), 12) + 1), 6), mod(div(i, 153), 12) + 1, div(mod(i, 153), 5) + 1


def j2d(jy, jm, jd):
    _, gy, march = jal_cal(jy)
    return g2d(gy, 3, march) + (jm - 1) * 31 - div(jm, 7) * (jm - 7) + jd - 1


def d2j(jdn):
    gy, _, _ = d2g(jdn)
    jy = gy - 621
    leap, _, march = jal_cal(jy)
    k = jdn - g2d(gy, 3, march)
    if k >= 0:
        if k <= 185:
            return jy, 1 + div(k, 31), mod(k, 31) + 1
        k -= 186
    else:
        jy -= 1
        k += 179
        if leap == 1:
            k += 1
    return jy, 7 + div(k, 30), mod(k, 30) + 1


def is_leap(jy):
    return jal_cal(jy)[0] == 0


def month_len(jy, jm):
    if jm <= 6:
        return 31
    if jm <= 11:
        return 30
    return 30 if is_leap(jy) else 29


def to_greg(d):
    return d2g(j2d(*d))


def from_greg(y, m, d):
    return d2j(g2d(y, m, d))


def fmt_j(d):
    return "%04d/%02d/%02d" % d


def fmt_g(d):
    return "%04d-%02d-%02d" % d


FA_DIGITS = "۰۱۲۳۴۵۶۷۸۹"
MONTHS_FA = ["فروردین", "اردیبهشت", "خرداد", "تیر", "مرداد", "شهریور", "مهر", "آبان", "آذر", "دی", "بهمن", "اسفند"]
WEEKDAYS_FA = ["شنبه", "یکشنبه", "دوشنبه", "سه‌شنبه", "چهارشنبه", "پنجشنبه", "جمعه"]


def fa_num(s):
    return "".join(FA_DIGITS[int(c)] if c.isdigit() else c for c in str(s))


def python_weekday_to_fa_index(y, m, d):
    import datetime
    return (datetime.date(y, m, d).weekday() + 2) % 7


def each_day(a, b):
    n = j2d(*b) - j2d(*a)
    return [d2j(j2d(*a) + i) for i in range(n + 1)]


def is_working(day, weekend, holidays):
    if fmt_j(day) in holidays:
        return False
    return python_weekday_to_fa_index(*to_greg(day)) not in weekend


def working_days(a, b, weekend, holidays):
    return sum(1 for d in each_day(a, b) if is_working(d, weekend, holidays))


def sandwich_days(a, b, weekend, holidays):
    """روزهای کاری از نخستین تا واپسین روز کاری بازه (روزهای میانی تعطیل هم حساب می‌شوند)."""
    days = each_day(a, b)
    work = [i for i, d in enumerate(days) if is_working(d, weekend, holidays)]
    if not work:
        return 0
    return work[-1] - work[0] + 1


def floor_div(a, b):
    return a // b


def round_half_up(x):
    return int(math.floor(x + 0.5)) if x >= 0 else -int(math.floor(-x + 0.5))


OUT = os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "..", "core", "testdata", "conformance")

# ---------------------------------------------------------------- calendar.json
years = []
for jy in [1398, 1399, 1400, 1401, 1402, 1403, 1404, 1405, 1408]:
    gy, gm, gd = to_greg((jy, 1, 1))
    years.append({
        "year": jy,
        "leap": is_leap(jy),
        "esfandLength": month_len(jy, 12),
        "farvardin1Gregorian": fmt_g((gy, gm, gd)),
    })

conversion_cases = [(1404, 1, 1), (1404, 6, 31), (1404, 7, 1), (1404, 7, 22), (1404, 12, 29),
                    (1403, 12, 30), (1400, 12, 29), (1399, 12, 30), (1398, 7, 14)]
conversions = []
for jy, jm, jd in conversion_cases:
    jdn = j2d(jy, jm, jd)
    gy, gm, gd = d2g(jdn)
    conversions.append({
        "jalali": fmt_j((jy, jm, jd)),
        "gregorian": fmt_g((gy, gm, gd)),
        "jdn": jdn,
        "weekdayIndex": python_weekday_to_fa_index(gy, gm, gd),
        "weekdayFa": WEEKDAYS_FA[python_weekday_to_fa_index(gy, gm, gd)],
        "monthLabelFa": "%s %s" % (MONTHS_FA[jm - 1], fa_num(jy)),
    })

month_lengths = [{"year": jy, "month": jm, "days": month_len(jy, jm)}
                 for jy, jm in [(1404, 1), (1404, 6), (1404, 7), (1404, 12), (1403, 12), (1400, 12)]]

work_windows = [
    {"name": "بازهٔ یک هفته‌ای با یک جمعه", "from": "1404/07/01", "to": "1404/07/05", "weekend": [6], "holidays": {}, "workingDays": 4, "sandwichDays": 5},
    {"name": "بازه‌ای که با تعطیل هفتگی آغاز می‌شود", "from": "1404/07/03", "to": "1404/07/06", "weekend": [6], "holidays": {}, "workingDays": 3, "sandwichDays": 4},
    {"name": "بازه با یک تعطیل رسمی در میانه", "from": "1404/07/01", "to": "1404/07/05", "weekend": [6], "holidays": {"1404/07/02": "تعطیل رسمی نمونه"}, "workingDays": 3, "sandwichDays": 5},
    {"name": "کل ماه مهر با پنجشنبه و جمعه تعطیل", "from": "1404/07/01", "to": "1404/07/30", "weekend": [5, 6], "holidays": {}, "workingDays": 21, "sandwichDays": 30},
]
for w in work_windows:
    a = tuple(int(x) for x in w["from"].split("/"))
    b = tuple(int(x) for x in w["to"].split("/"))
    w["workingDays"] = working_days(a, b, w["weekend"], w["holidays"])
    w["sandwichDays"] = sandwich_days(a, b, w["weekend"], w["holidays"])

calendar_fixture = {
    "id": "calendar",
    "description": "صحت تبدیل تقویم شمسی/میلادی، کبیسه، طول ماه و شمارش روز کاری",
    "years": years,
    "conversions": conversions,
    "monthLengths": month_lengths,
    "workWindows": work_windows,
}

# ---------------------------------------------------------------- leave.json
leave_params = {
    "accrualPerMonth": 2.5,
    "annualCap": 30,
    "carryCap": 9,
    "weekend": [6],
    "holidays": {"1404/07/02": "تعطیل رسمی نمونه"},
}


def count_leave_days(a, b, sandwich):
    if sandwich:
        return sandwich_days(a, b, leave_params["weekend"], leave_params["holidays"])
    return working_days(a, b, leave_params["weekend"], leave_params["holidays"])


leave_cases = [
    {"name": "مرخصی سه‌روزه با یک جمعه", "from": "1404/07/14", "to": "1404/07/16", "sandwich": False, "balance": 12.0},
    {"name": "مرخصی هفت‌روزه شامل جمعه", "from": "1404/07/05", "to": "1404/07/11", "sandwich": False, "balance": 12.0},
    {"name": "همان بازه با قاعدهٔ ساندویچ", "from": "1404/07/05", "to": "1404/07/11", "sandwich": True, "balance": 12.0},
    {"name": "مرخصی یک‌روزه روی تعطیل رسمی", "from": "1404/07/02", "to": "1404/07/02", "sandwich": False, "balance": 5.0},
    {"name": "ماندهٔ ناکافی", "from": "1404/07/01", "to": "1404/07/10", "sandwich": False, "balance": 2.0},
]
for c in leave_cases:
    a = tuple(int(x) for x in c["from"].split("/"))
    b = tuple(int(x) for x in c["to"].split("/"))
    n = float(count_leave_days(a, b, c["sandwich"]))
    c["days"] = n
    c["eligible"] = c["balance"] >= n
    c["balanceAfter"] = round(c["balance"] - n, 2) if c["eligible"] else c["balance"]

accrual_cases = []
for months in [0, 1, 6, 12, 18, 30]:
    raw = round(months * leave_params["accrualPerMonth"], 2)
    accrual_cases.append({"monthsWorked": months, "rawAccrual": raw, "cappedAccrual": min(raw, float(leave_params["annualCap"]))})

carry_cases = [{"balance": 14.5, "carryCap": 9, "carried": 9.0},
               {"balance": 6.0, "carryCap": 9, "carried": 6.0},
               {"balance": 9.0, "carryCap": 9, "carried": 9.0},
               {"balance": 0.0, "carryCap": 9, "carried": 0.0}]

leave_fixture = {
    "id": "leave",
    "description": "قواعد شمارش روز مرخصی، ساندویچ، مانده و ذخیره‌سازی سالانه",
    "params": leave_params,
    "cases": leave_cases,
    "accrualCases": accrual_cases,
    "carryCases": carry_cases,
}

# ---------------------------------------------------------------- payroll.json
payroll_params = {
    "insuranceEmployeeRate": 0.07,
    "insuranceEmployerRate": 0.23,
    "insuranceCeiling": 200000000,
    "housingAllowance": 9000000,
    "childAllowancePerChild": 7000000,
    "taxExemption": 24000000,
    "taxBrackets": [
        {"upto": 50000000, "rate": 0.10},
        {"upto": 100000000, "rate": 0.15},
        {"upto": 200000000, "rate": 0.20},
        {"upto": 0, "rate": 0.30},
    ],
}


def compute_tax(income, brackets):
    """مالیات پله‌ای: هر پله تا سقف خودش نرخ خودش را دارد. بات نهایی (upto=0) بی‌سقف است."""
    tax = 0.0
    prev = 0
    for b in brackets:
        if income <= prev:
            break
        upper = b["upto"] if b["upto"] > 0 else income
        slice_amt = min(income, upper) - prev
        if slice_amt > 0:
            tax += slice_amt * b["rate"]
        prev = upper
    return round_half_up(tax)


def compute_payroll(case):
    overtime_pay = case["overtimeHours"] * case["overtimeHourlyRate"]
    children_pay = case["children"] * payroll_params["childAllowancePerChild"]
    gross = case["baseSalary"] + payroll_params["housingAllowance"] + children_pay + overtime_pay
    insurance_base = min(gross, payroll_params["insuranceCeiling"])
    insurance_employee = round_half_up(insurance_base * payroll_params["insuranceEmployeeRate"])
    insurance_employer = round_half_up(insurance_base * payroll_params["insuranceEmployerRate"])
    taxable = max(0, gross - insurance_employee - payroll_params["taxExemption"])
    tax = compute_tax(taxable, payroll_params["taxBrackets"])
    net = gross - insurance_employee - tax - case["otherDeductions"]
    return {
        "overtimePay": overtime_pay,
        "gross": gross,
        "insuranceBase": insurance_base,
        "insuranceEmployee": insurance_employee,
        "insuranceEmployer": insurance_employer,
        "taxableIncome": taxable,
        "tax": tax,
        "net": net,
    }


payroll_inputs = [
    {"name": "کارمند بدون اضافه‌کاری", "baseSalary": 52000000, "children": 0, "overtimeHours": 0, "overtimeHourlyRate": 0, "otherDeductions": 0},
    {"name": "کارمند با دو فرزند و اضافه‌کاری", "baseSalary": 34000000, "children": 2, "overtimeHours": 12, "overtimeHourlyRate": 250000, "otherDeductions": 1500000},
    {"name": "حقوق بالای سقف بیمه", "baseSalary": 250000000, "children": 1, "overtimeHours": 0, "overtimeHourlyRate": 0, "otherDeductions": 0},
    {"name": "درآمد زیر سقف معافیت", "baseSalary": 15000000, "children": 0, "overtimeHours": 0, "overtimeHourlyRate": 0, "otherDeductions": 0},
]
payroll_cases = []
for c in payroll_inputs:
    case = dict(c)
    case["expected"] = compute_payroll(c)
    payroll_cases.append(case)

payroll_fixture = {
    "id": "payroll",
    "description": "زنجیرهٔ محاسبهٔ حقوق: ناخالص ← بیمه ← مالیات پله‌ای ← خالص",
    "params": payroll_params,
    "cases": payroll_cases,
}

# ---------------------------------------------------------------- kpi.json
kpi_cases = []
for name, items in [
    ("ارزیابی متوازن", [{"kpiId": "KPI-1", "weight": 30, "score": 4.5}, {"kpiId": "KPI-2", "weight": 30, "score": 3.0}, {"kpiId": "KPI-3", "weight": 40, "score": 4.0}]),
    ("وزن صفر", [{"kpiId": "KPI-1", "weight": 0, "score": 5.0}, {"kpiId": "KPI-2", "weight": 50, "score": 3.5}]),
    ("همه صفر", [{"kpiId": "KPI-1", "weight": 0, "score": 5.0}]),
    ("نمرهٔ کامل", [{"kpiId": "KPI-1", "weight": 100, "score": 5.0}]),
]:
    total_weight = sum(i["weight"] for i in items)
    weighted = 0.0 if total_weight == 0 else sum(i["weight"] * i["score"] for i in items) / total_weight
    grade = "بدون ارزیابی" if total_weight == 0 else ("عالی" if weighted >= 4.5 else "خوب" if weighted >= 3.5 else "قابل قبول" if weighted >= 2.5 else "نیازمند بهبود")
    kpi_cases.append({"name": name, "items": items, "totalWeight": total_weight, "weightedScore": round(weighted, 2), "gradeFa": grade})

kpi_fixture = {
    "id": "kpi",
    "description": "میانگین موزون شاخص‌های عملکرد و رتبه‌بندی فارسی",
    "cases": kpi_cases,
}

# ---------------------------------------------------------------- نوشتن
os.makedirs(OUT, exist_ok=True)
written = []
for name, data in [("calendar", calendar_fixture), ("leave", leave_fixture), ("payroll", payroll_fixture), ("kpi", kpi_fixture)]:
    path = os.path.join(OUT, name + ".json")
    with open(path, "w", encoding="utf-8", newline="\n") as f:
        json.dump(data, f, ensure_ascii=False, indent=2, sort_keys=False)
        f.write("\n")
    written.append(path)

for path in written:
    print("نوشته شد:", os.path.relpath(path))
summary = {k: (len(v) if isinstance(v, list) else len(v.get("cases", []))) for k, v in
           [("years", years), ("conversions", conversions), ("monthLengths", month_lengths),
            ("workWindows", work_windows), ("leaveCases", leave_cases), ("accrualCases", accrual_cases),
            ("payrollCases", payroll_cases), ("kpiCases", kpi_cases)]}
print(json.dumps(summary, ensure_ascii=False))
print("نمونهٔ حقوق:", json.dumps(payroll_cases[1]["expected"], ensure_ascii=False))
print("نمونهٔ مرخصی:", json.dumps(leave_cases[0], ensure_ascii=False))
