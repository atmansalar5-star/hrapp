//! تقویم هجری شمسی، روز کاری و تعطیلات.
//!
//! این crate مبنای همهٔ محاسبات زمانی سامانه است: شمارش روز مرخصی، مهلت‌های تأیید،
//! برش دورهٔ حقوق و نمایش تقویم ماهانه. همهٔ محاسبات با عدد صحیح انجام می‌شود و
//! «شمارهٔ روز مطلق» (jdn) به‌عنوان ساعت پایه استفاده می‌شود تا محاسبات روزانه در
//! سراسر سامانه یکدست بماند.
//!
//! صحت این پیاده‌سازی با `core/testdata/conformance/calendar.json` سنجیده می‌شود؛
//! همان فایل را سرور Go هم مصرف می‌کند تا دو میزبان هرگز از هم جدا نیفتند.

use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::fmt;

/// سقف‌های تقویم شمسی طبق الگوریتم بورکوفسکی (مبنای jalaali).
const BREAKS: [i32; 20] = [
    -61, 9, 38, 199, 426, 686, 756, 818, 1111, 1181, 1210, 1635, 2060, 2097, 2192, 2262, 2324, 2394, 2456, 3178,
];

/// خطاهای تقویم با پیام فارسی قابل نمایش به کاربر.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum CalendarError {
    OutOfRange(i32),
    InvalidMonth(u32),
    InvalidDay { day: u32, year: i32, month: u32, max: u32 },
    BadFormat(String),
    InvalidRange { from: String, to: String },
}

impl fmt::Display for CalendarError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            CalendarError::OutOfRange(y) => write!(f, "سال شمسی خارج از بازهٔ پشتیبانی‌شده است: {y}"),
            CalendarError::InvalidMonth(m) => write!(f, "شمارهٔ ماه نامعتبر است: {m}"),
            CalendarError::InvalidDay { day, year, month, max } => {
                write!(f, "روز {day} برای ماه {month:02} سال {year} نامعتبر است (بیشینه {max})")
            }
            CalendarError::BadFormat(s) => write!(f, "قالب تاریخ باید YYYY/MM/DD باشد، دریافت‌شده: {s}"),
            CalendarError::InvalidRange { from, to } => {
                write!(f, "بازهٔ تاریخ نامعتبر است: {from} تا {to}")
            }
        }
    }
}

impl std::error::Error for CalendarError {}

/// یک تاریخ شمسی.
#[derive(Debug, Clone, Copy, PartialEq, Eq, PartialOrd, Ord, Serialize, Deserialize)]
pub struct JalaliDate {
    pub year: i32,
    pub month: u32,
    pub day: u32,
}

struct JalCal {
    leap: i32,
    gy: i32,
    march: i32,
}

fn div(a: i64, b: i64) -> i64 {
    a / b
}

fn modr(a: i64, b: i64) -> i64 {
    a - div(a, b) * b
}

fn jal_cal(jy: i32) -> Result<JalCal, CalendarError> {
    if jy < BREAKS[0] || jy >= BREAKS[BREAKS.len() - 1] {
        return Err(CalendarError::OutOfRange(jy));
    }
    let gy = jy + 621;
    let mut leap_j: i64 = -14;
    let mut jp = BREAKS[0];
    let mut jump = 0i32;

    for &jm in BREAKS.iter().skip(1) {
        jump = jm - jp;
        if jy < jm {
            break;
        }
        leap_j += div(jump as i64, 33) * 8 + div(modr(jump as i64, 33), 4);
        jp = jm;
    }
    let mut n = jy - jp;
    leap_j += div(n as i64, 33) * 8 + div(modr(n as i64, 33) + 3, 4);
    if modr(jump as i64, 33) == 4 && jump - n == 4 {
        leap_j += 1;
    }
    let leap_g = div(gy as i64, 4) - div((div(gy as i64, 100) + 1) * 3, 4) - 150;
    let march = 20 + leap_j - leap_g;

    if jump - n < 6 {
        n = n - jump + div((jump + 4) as i64, 33) as i32 * 33;
    }
    let mut leap = modr(modr((n + 1) as i64, 33) - 1, 4) as i32;
    if leap == -1 {
        leap = 4;
    }
    Ok(JalCal { leap, gy, march })
}

fn g2d(gy: i32, gm: u32, gd: u32) -> i64 {
    let gm = gm as i64;
    let gy = gy as i64;
    let gd = gd as i64;
    let mut d = div((gy + div(gm - 8, 6) + 100100) * 1461, 4) + div(153 * modr(gm + 9, 12) + 2, 5) + gd - 34840408;
    d = d - div(div(gy + 100100 + div(gm - 8, 6), 100) * 3, 4) + 752;
    d
}

fn d2g(jdn: i64) -> (i32, u32, u32) {
    let mut j = 4 * jdn + 139361631;
    j += div(div(4 * jdn + 183187720, 146097) * 3, 4) * 4 - 3908;
    let i = div(modr(j, 1461), 4) * 5 + 308;
    let gd = div(modr(i, 153), 5) + 1;
    let gm = modr(div(i, 153), 12) + 1;
    let gy = div(j, 1461) - 100100 + div(8 - gm, 6);
    (gy as i32, gm as u32, gd as u32)
}

/// ارقام فارسی و عربی را به لاتین برمی‌گرداند (ورودی کاربر همیشه تمیز نیست).
pub fn normalize_digits(s: &str) -> String {
    s.chars()
        .map(|c| match c {
            '۰'..='۹' => char::from(b'0' + (c as u32 - '۰' as u32) as u8),
            '٠'..='٩' => char::from(b'0' + (c as u32 - '٠' as u32) as u8),
            other => other,
        })
        .collect()
}

/// ارقام لاتین را به فارسی تبدیل می‌کند.
pub fn to_persian_digits(s: &str) -> String {
    s.chars()
        .map(|c| match c {
            '0'..='9' => char::from_u32('۰' as u32 + (c as u32 - '0' as u32)).unwrap_or(c),
            other => other,
        })
        .collect()
}

/// نام فارسی ماه‌ها.
pub const MONTHS_FA: [&str; 12] = [
    "فروردین", "اردیبهشت", "خرداد", "تیر", "مرداد", "شهریور", "مهر", "آبان", "آذر", "دی", "بهمن", "اسفند",
];

/// نام فارسی روزهای هفته؛ اندیس ۰ شنبه و ۶ جمعه است.
pub const WEEKDAYS_FA: [&str; 7] = ["شنبه", "یکشنبه", "دوشنبه", "سه‌شنبه", "چهارشنبه", "پنجشنبه", "جمعه"];

impl JalaliDate {
    /// ساخت تاریخ با اعتبارسنجی کامل.
    pub fn new(year: i32, month: u32, day: u32) -> Result<Self, CalendarError> {
        let max = Self::month_length(year, month)?;
        if day < 1 || day > max {
            return Err(CalendarError::InvalidDay { day, year, month, max });
        }
        Ok(JalaliDate { year, month, day })
    }

    /// تجزیهٔ رشتهٔ تاریخ به قالب YYYY/MM/DD با پذیرش ارقام فارسی.
    pub fn parse(input: &str) -> Result<Self, CalendarError> {
        let cleaned = normalize_digits(input.trim());
        let parts: Vec<&str> = cleaned.split(['/', '-']).collect();
        if parts.len() != 3 || parts.iter().any(|p| p.is_empty() || !p.chars().all(|c| c.is_ascii_digit())) {
            return Err(CalendarError::BadFormat(input.to_string()));
        }
        let year: i32 = parts[0].parse().map_err(|_| CalendarError::BadFormat(input.to_string()))?;
        let month: u32 = parts[1].parse().map_err(|_| CalendarError::BadFormat(input.to_string()))?;
        let day: u32 = parts[2].parse().map_err(|_| CalendarError::BadFormat(input.to_string()))?;
        Self::new(year, month, day)
    }

    /// آیا سال کبیسه است.
    pub fn is_leap(year: i32) -> Result<bool, CalendarError> {
        Ok(jal_cal(year)?.leap == 0)
    }

    /// طول ماه شمسی.
    pub fn month_length(year: i32, month: u32) -> Result<u32, CalendarError> {
        match month {
            1..=6 => Ok(31),
            7..=11 => Ok(30),
            12 => {
                if Self::is_leap(year)? {
                    Ok(30)
                } else {
                    Ok(29)
                }
            }
            m => Err(CalendarError::InvalidMonth(m)),
        }
    }

    /// شمارهٔ روز مطلق — ساعت پایهٔ همهٔ محاسبات روزانه.
    pub fn day_number(&self) -> Result<i64, CalendarError> {
        let cal = jal_cal(self.year)?;
        Self::month_length(self.year, self.month)?;
        Ok(g2d(cal.gy, 3, cal.march as u32) + (self.month as i64 - 1) * 31 - div(self.month as i64, 7) * (self.month as i64 - 7)
            + self.day as i64
            - 1)
    }

    /// بازسازی تاریخ از شمارهٔ روز مطلق.
    pub fn from_day_number(jdn: i64) -> Result<Self, CalendarError> {
        let (gy, _, _) = d2g(jdn);
        let mut jy = gy - 621;
        let cal = jal_cal(jy)?;
        let jdn1f = g2d(gy, 3, cal.march as u32);
        let mut k = jdn - jdn1f;
        if k >= 0 {
            if k <= 185 {
                return Ok(JalaliDate { year: jy, month: (1 + div(k, 31)) as u32, day: (modr(k, 31) + 1) as u32 });
            }
            k -= 186;
        } else {
            jy -= 1;
            k += 179;
            if cal.leap == 1 {
                k += 1;
            }
        }
        Ok(JalaliDate { year: jy, month: (7 + div(k, 30)) as u32, day: (modr(k, 30) + 1) as u32 })
    }

    /// تاریخ میلادی متناظر به قالب ISO.
    pub fn to_gregorian(&self) -> Result<(i32, u32, u32), CalendarError> {
        Ok(d2g(self.day_number()?))
    }

    /// ساخت تاریخ شمسی از تاریخ میلادی.
    pub fn from_gregorian(year: i32, month: u32, day: u32) -> Result<Self, CalendarError> {
        Self::from_day_number(g2d(year, month, day))
    }

    /// اندیس روز هفته: ۰ شنبه … ۶ جمعه.
    pub fn weekday_index(&self) -> Result<u32, CalendarError> {
        // رابطهٔ ثابت بین شمارهٔ روز مطلق و روز هفتهٔ شمسی؛ با fixtureها تأیید شده است.
        Ok((modr(self.day_number()? + 2, 7)) as u32)
    }

    /// نام فارسی روز هفته.
    pub fn weekday_fa(&self) -> Result<&'static str, CalendarError> {
        Ok(WEEKDAYS_FA[self.weekday_index()? as usize])
    }

    /// برچسب فارسی ماه مانند «مهر ۱۴۰۴».
    pub fn month_label_fa(&self) -> Result<String, CalendarError> {
        let name = MONTHS_FA
            .get(self.month.saturating_sub(1) as usize)
            .ok_or(CalendarError::InvalidMonth(self.month))?;
        Ok(format!("{} {}", name, to_persian_digits(&self.year.to_string())))
    }

    /// افزودن روز (منفی هم مجاز است).
    pub fn add_days(&self, days: i64) -> Result<Self, CalendarError> {
        Self::from_day_number(self.day_number()? + days)
    }

    /// فاصلهٔ روزها نسبت به تاریخ دیگر (other - self).
    pub fn days_until(&self, other: &JalaliDate) -> Result<i64, CalendarError> {
        Ok(other.day_number()? - self.day_number()?)
    }

    /// پیمایش روزهای بازه شامل هر دو سر.
    pub fn each_day(&self, to: &JalaliDate) -> Result<Vec<JalaliDate>, CalendarError> {
        let n = self.days_until(to)?;
        if n < 0 {
            return Err(CalendarError::InvalidRange { from: self.to_string(), to: to.to_string() });
        }
        (0..=n).map(|i| self.add_days(i)).collect()
    }

    /// ماه بعدی و سال متناظر.
    pub fn next_month(&self) -> Result<Self, CalendarError> {
        if self.month == 12 {
            Ok(JalaliDate { year: self.year + 1, month: 1, day: 1 })
        } else {
            Ok(JalaliDate { year: self.year, month: self.month + 1, day: 1 })
        }
    }
}

impl fmt::Display for JalaliDate {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "{:04}/{:02}/{:02}", self.year, self.month, self.day)
    }
}

/// تقویم کاری سازمان: روزهای تعطیل هفتگی و تعطیلات رسمی.
#[derive(Debug, Clone, Default)]
pub struct WorkCalendar {
    /// اندیس روزهای تعطیل هفتگی (۰ شنبه … ۶ جمعه).
    pub weekend: Vec<u32>,
    /// تعطیلات رسمی به قالب YYYY/MM/DD.
    pub holidays: HashMap<String, String>,
}

impl WorkCalendar {
    /// تقویم پیش‌فرض: تنها جمعه تعطیل است.
    pub fn default_iran() -> Self {
        WorkCalendar { weekend: vec![6], holidays: HashMap::new() }
    }

    /// آیا روز تعطیل هفتگی است.
    pub fn is_weekend(&self, d: &JalaliDate) -> Result<bool, CalendarError> {
        Ok(self.weekend.contains(&d.weekday_index()?))
    }

    /// آیا روز تعطیل رسمی است.
    pub fn is_holiday(&self, d: &JalaliDate) -> bool {
        self.holidays.contains_key(&d.to_string())
    }

    /// روز کاری = نه تعطیل هفتگی و نه تعطیل رسمی.
    pub fn is_working_day(&self, d: &JalaliDate) -> Result<bool, CalendarError> {
        if self.is_holiday(d) {
            return Ok(false);
        }
        Ok(!self.is_weekend(d)?)
    }

    /// شمار روزهای کاری بازه.
    pub fn working_days(&self, from: &JalaliDate, to: &JalaliDate) -> Result<u32, CalendarError> {
        let mut n = 0;
        for d in from.each_day(to)? {
            if self.is_working_day(&d)? {
                n += 1;
            }
        }
        Ok(n)
    }

    /// روزهای کاری «ساندویچی»: از نخستین تا واپسین روز کاری بازه؛
    /// تعطیل‌های میانی هم به حساب می‌آیند ولی تعطیل‌های دو سر نه.
    pub fn sandwich_days(&self, from: &JalaliDate, to: &JalaliDate) -> Result<u32, CalendarError> {
        let days = from.each_day(to)?;
        let mut first = None;
        let mut last = 0usize;
        for (i, d) in days.iter().enumerate() {
            if self.is_working_day(d)? {
                if first.is_none() {
                    first = Some(i);
                }
                last = i;
            }
        }
        match first {
            None => Ok(0),
            Some(f) => Ok((last - f + 1) as u32),
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn nowruz_anchors() {
        // نوروز و کبیسه‌بودن سال‌ها — مقادیر از fixture طلایی.
        let cases = [
            (1399, 2020, 3, 20, true),
            (1400, 2021, 3, 21, false),
            (1401, 2022, 3, 21, false),
            (1402, 2023, 3, 21, false),
            (1403, 2024, 3, 20, true),
            (1404, 2025, 3, 21, false),
            (1405, 2026, 3, 21, false),
            (1408, 2029, 3, 21, true),
        ];
        for (jy, gy, gm, gd, leap) in cases {
            let d = JalaliDate::new(jy, 1, 1).unwrap();
            assert_eq!(d.to_gregorian().unwrap(), (gy, gm, gd), "نوروز {jy}");
            assert_eq!(JalaliDate::is_leap(jy).unwrap(), leap, "کبیسهٔ {jy}");
        }
    }

    #[test]
    fn round_trip_and_weekday() {
        let d = JalaliDate::parse("1404/07/01").unwrap();
        assert_eq!(d.to_string(), "1404/07/01");
        assert_eq!(d.to_gregorian().unwrap(), (2025, 9, 23));
        assert_eq!(d.month_label_fa().unwrap(), "مهر ۱۴۰۴");
        assert_eq!(d.weekday_fa().unwrap(), "سه‌شنبه");
        assert_eq!(JalaliDate::from_gregorian(2025, 9, 23).unwrap(), d);
        assert_eq!(JalaliDate::parse("۱۴۰۴/۰۷/۰۱").unwrap(), d);
    }

    #[test]
    fn add_and_diff_days() {
        let a = JalaliDate::parse("1404/12/29").unwrap();
        let b = a.add_days(1).unwrap();
        assert_eq!(b.to_string(), "1405/01/01");
        assert_eq!(b.to_gregorian().unwrap(), (2026, 3, 21));
        assert_eq!(a.days_until(&b).unwrap(), 1);
        assert_eq!(JalaliDate::new(1404, 12, 30).is_err(), true, "۱۴۰۴ کبیسه نیست");
        assert!(JalaliDate::new(1403, 12, 30).is_ok(), "۱۴۰۳ کبیسه است");
    }

    #[test]
    fn work_calendar_counts() {
        let cal = WorkCalendar::default_iran();
        let from = JalaliDate::parse("1404/07/01").unwrap();
        let to = JalaliDate::parse("1404/07/05").unwrap();
        assert_eq!(cal.working_days(&from, &to).unwrap(), 4);
        assert_eq!(cal.sandwich_days(&from, &to).unwrap(), 5);

        let mut with_holiday = WorkCalendar::default_iran();
        with_holiday.holidays.insert("1404/07/02".to_string(), "تعطیل رسمی نمونه".to_string());
        assert_eq!(with_holiday.working_days(&from, &to).unwrap(), 3);
        assert_eq!(with_holiday.sandwich_days(&from, &to).unwrap(), 5);
    }

    #[test]
    fn month_switches_and_lengths() {
        let d = JalaliDate::new(1404, 6, 31).unwrap();
        assert_eq!(d.next_month().unwrap().to_string(), "1404/07/01");
        let d = JalaliDate::new(1404, 12, 29).unwrap();
        assert_eq!(d.next_month().unwrap().to_string(), "1405/01/01");
        assert_eq!(JalaliDate::month_length(1404, 12).unwrap(), 29);
        assert_eq!(JalaliDate::month_length(1403, 12).unwrap(), 30);
    }
}
