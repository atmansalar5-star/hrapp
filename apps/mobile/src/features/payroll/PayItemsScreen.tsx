import RecordKindScreen from '../../components/RecordKindScreen';

/**
 * اجزای حقوق — فرم واقعی تعریف اقلام مزد و کسور (مزایا/کسور، ثابت/درصدی/فرمولی)
 * روی API رکوردکیت. روش محاسبه، مبنا و ضریب هر قلم از شناسهٔ سرور در فرم دیده
 * می‌شود و اعتبارسنجی (مبنا برای درصدی، فرمول برای فرمولی) در سرور اعمال می‌شود.
 */
export default function PayItemsScreen() {
  return <RecordKindScreen prefix="/api/v1/payroll" kindKey="payItems" />;
}
