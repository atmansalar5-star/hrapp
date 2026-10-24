# فونت‌ها

فونتی که در بستهٔ طراحی استفاده شده **Vazirmatn** است (پروانهٔ OFL). فایل‌های زیر را در
همین پوشه قرار دهید و سپس یک‌بار `npx react-native-asset` را اجرا کنید تا در پروژه‌های
بومی ثبت شوند:

```
Vazirmatn-Regular.ttf
Vazirmatn-Medium.ttf
Vazirmatn-Bold.ttf
Vazirmatn-Black.ttf
```

تا زمانی که این فایل‌ها اضافه نشوند، `apps/mobile/src/design/tokens.ts` مقدار جایگزین
`Tahoma` را اعلام می‌کند و چیدمان راست‌چین دست‌نخورده می‌ماند؛ فقط شکل قلم تغییر می‌کند.

برای Web/دسکتاپ همان فونت را از CDN رسمی Vazirmatn بار کنید تا خروجی سه پلتفرم یکدست بماند.
