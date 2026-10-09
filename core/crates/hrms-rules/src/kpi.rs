//! قواعد ارزیابی عملکرد: میانگین موزون شاخص‌ها و رتبه‌بندی فارسی.
//!
//! وزن‌ها از فرم ارزیابی (ماژول عملکرد) می‌آید؛ اینجا فقط ریاضیات و آستانه‌های
//! رتبه‌بندی وجود دارد تا نتیجه در کلاینت و سرور یکسان محاسبه شود.

use crate::round2;
use serde::{Deserialize, Serialize};

/// یک شاخص ارزیابی با وزن و نمره (مقیاس ۰ تا ۵).
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct KpiItem {
    pub kpi_id: String,
    pub weight: f64,
    pub score: f64,
}

/// میانگین موزون نمره‌ها؛ اگر جمع وزن صفر باشد مقدار صفر برمی‌گردد.
pub fn weighted_score(items: &[KpiItem]) -> f64 {
    let total_weight: f64 = items.iter().map(|i| i.weight).sum();
    if total_weight <= 0.0 {
        return 0.0;
    }
    let sum: f64 = items.iter().map(|i| i.weight * i.score).sum();
    round2(sum / total_weight)
}

/// رتبهٔ فارسی بر پایهٔ نمرهٔ موزون.
pub fn grade_fa(score: f64, has_weight: bool) -> &'static str {
    if !has_weight {
        return "بدون ارزیابی";
    }
    if score >= 4.5 {
        "عالی"
    } else if score >= 3.5 {
        "خوب"
    } else if score >= 2.5 {
        "قابل قبول"
    } else {
        "نیازمند بهبود"
    }
}

/// نتیجهٔ ارزیابی یک کارمند.
#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct KpiOutcome {
    pub total_weight: f64,
    pub weighted_score: f64,
    pub grade_fa: String,
}

/// ارزیابی کامل یک فرم.
pub fn evaluate(items: &[KpiItem]) -> KpiOutcome {
    let total_weight: f64 = items.iter().map(|i| i.weight).sum();
    let score = weighted_score(items);
    KpiOutcome {
        total_weight,
        weighted_score: score,
        grade_fa: grade_fa(score, total_weight > 0.0).to_string(),
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn matches_golden_fixture_cases() {
        let balanced = evaluate(&[
            KpiItem { kpi_id: "KPI-1".into(), weight: 30.0, score: 4.5 },
            KpiItem { kpi_id: "KPI-2".into(), weight: 30.0, score: 3.0 },
            KpiItem { kpi_id: "KPI-3".into(), weight: 40.0, score: 4.0 },
        ]);
        assert_eq!(balanced.weighted_score, 3.85);
        assert_eq!(balanced.grade_fa, "خوب");

        let zero_weight = evaluate(&[
            KpiItem { kpi_id: "KPI-1".into(), weight: 0.0, score: 5.0 },
            KpiItem { kpi_id: "KPI-2".into(), weight: 50.0, score: 3.5 },
        ]);
        assert_eq!(zero_weight.weighted_score, 3.5);
        assert_eq!(zero_weight.grade_fa, "خوب");

        let none = evaluate(&[KpiItem { kpi_id: "KPI-1".into(), weight: 0.0, score: 5.0 }]);
        assert_eq!(none.weighted_score, 0.0);
        assert_eq!(none.grade_fa, "بدون ارزیابی");

        let full = evaluate(&[KpiItem { kpi_id: "KPI-1".into(), weight: 100.0, score: 5.0 }]);
        assert_eq!(full.weighted_score, 5.0);
        assert_eq!(full.grade_fa, "عالی");
    }
}
