import RecordKindScreen from '../../components/RecordKindScreen';

/**
 * پارامترها و ضرائب حقوق — فرم واقعی تغذیهٔ پارامترهای حقوق روی API رکوردکیت.
 *
 * ساختار فرم، برچسب‌ها، وضعیت‌ها و اقدام‌ها (جاری‌سازی/غیرفعال‌سازی) همه از
 * /api/v1/payroll/schema می‌آید؛ کارشناس حقوق این‌جا پارامتر تعریف/ویرایش می‌کند،
 * تاریخ اجرا می‌گذارد و نسخه را جاری می‌سازد — بدون اینکه عددی در رابط هاردکد شود.
 * پیش‌نویس اثر عملیاتی ندارد و فقط نسخهٔ جاریِ رسیده به تاریخ اجرا در محاسبهٔ حقوق
 * مؤثر است (قاعده در سرور).
 */
export default function PayParamsScreen() {
  return <RecordKindScreen prefix="/api/v1/payroll" kindKey="payParams" />;
}
