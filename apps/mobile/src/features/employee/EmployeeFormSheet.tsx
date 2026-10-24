/**
 * فرم کارمند — دستور ۳-۳ سند طراحی.
 *
 * فیلدها بر پایهٔ منطق گروه‌بندی شده‌اند (اطلاعات شخصی / شغلی / قرارداد)، فیلدهای
 * اجباری با ستاره مشخص‌اند و پیام خطای هر فیلد کنار خودش نمایش داده می‌شود. اعتبارسنجی
 * نخست در هستهٔ بومی انجام می‌شود و پاسخ سرور بر آن غلبه می‌کند.
 */

import { useMemo, useState } from 'react';
import { ScrollView, StyleSheet, Text, View } from 'react-native';

import { useTheme } from '../../design/theme';
import { rtlText } from '../../design/rtl';
import { Button, ErrorBanner, FormField } from '../../components/ui';
import type { Department, Employee, EmployeeInput } from '../../api/contracts';

interface Props {
  departments: Department[];
  /** کارمند در حالت ویرایش؛ در حالت ایجاد مقدار ندارد. */
  employee?: Employee;
  busy: boolean;
  errorMessage?: string;
  onSubmit: (input: EmployeeInput) => void;
  onCancel: () => void;
}

type ContractType = EmployeeInput['contractType'];
type EmployeeStatus = NonNullable<EmployeeInput['status']>;

const CONTRACT_TYPES: ContractType[] = ['تمام‌وقت', 'پیمانکاری', 'پاره‌وقت'];
const STATUSES: { key: EmployeeStatus; labelFa: string }[] = [
  { key: 'active', labelFa: 'شاغل' },
  { key: 'leave', labelFa: 'مرخصی' },
  { key: 'probation', labelFa: 'آزمایشی' },
  { key: 'terminated', labelFa: 'ترک خدمت' },
];

export default function EmployeeFormSheet({ departments, employee, busy, errorMessage, onSubmit, onCancel }: Props) {
  const { theme } = useTheme();
  const editing = Boolean(employee);

  const [fullName, setFullName] = useState(employee?.fullName ?? '');
  const [nationalId, setNationalId] = useState(employee?.nationalId ?? '');
  const [departmentId, setDepartmentId] = useState(employee?.departmentId ?? departments[0]?.id ?? '');
  const [position, setPosition] = useState(employee?.position ?? '');
  const [contractType, setContractType] = useState<ContractType>(employee?.contractType ?? CONTRACT_TYPES[0] ?? 'تمام‌وقت');
  const [status, setStatus] = useState<EmployeeStatus>(employee?.status ?? 'active');
  const [hireDate, setHireDate] = useState(employee?.hireDate ?? '1404/07/01');
  const [baseSalary, setBaseSalary] = useState(String(employee?.baseSalary ?? 30000000));
  const [phone, setPhone] = useState(employee?.phone ?? '');
  const [email, setEmail] = useState(employee?.email ?? '');
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [serverMessage, setServerMessage] = useState<string | undefined>(undefined);

  const styles = useMemo(
    () =>
      StyleSheet.create({
        wrapper: { flex: 1 },
        group: {
          backgroundColor: theme.colors.surface,
          borderWidth: 1,
          borderColor: theme.colors.border,
          borderRadius: theme.layout.radius,
          padding: theme.spacing.lg,
          marginBottom: theme.spacing.md,
        },
        groupTitle: { color: theme.colors.text, fontWeight: '800', marginBottom: theme.spacing.md, ...rtlText },
        choices: { flexDirection: 'row-reverse', flexWrap: 'wrap', gap: theme.spacing.sm, marginBottom: theme.spacing.md },
        choice: {
          borderWidth: 1,
          borderColor: theme.colors.border,
          borderRadius: 999,
          paddingHorizontal: theme.spacing.md,
          paddingVertical: 6,
          backgroundColor: theme.colors.surface,
        },
        choiceActive: { backgroundColor: theme.colors.primaryLight, borderColor: theme.colors.primary },
        choiceText: { color: theme.colors.text, fontSize: theme.typography.size.xs },
        choiceTextActive: { color: theme.colors.primaryDark, fontWeight: '700' },
        label: { color: theme.colors.text, fontWeight: '700', marginBottom: 4, ...rtlText },
        footer: { flexDirection: 'row-reverse', gap: theme.spacing.md, marginBottom: theme.spacing.xl },
      }),
    [theme],
  );

  const submit = () => {
    const input: EmployeeInput = {
      fullName: fullName.trim(),
      nationalId: nationalId.trim(),
      departmentId,
      position: position.trim(),
      contractType,
      hireDate: hireDate.trim(),
      status,
      baseSalary: Number(baseSalary.replace(/[^0-9]/g, '')) || 0,
      phone: phone.trim(),
      email: email.trim(),
    };

    // خطاهای پیش از ارسال: تنها موارد ساختاری که در سرور هم بررسی می‌شوند.
    const localErrors: Record<string, string> = {};
    if (input.fullName.length < 3) {
      localErrors.fullName = 'نام و نام خانوادگی باید حداقل ۳ نویسه باشد.';
    }
    if (!/^\d{10}$/.test(input.nationalId)) {
      localErrors.nationalId = 'کد ملی باید ۱۰ رقم باشد.';
    }
    if (!input.position) {
      localErrors.position = 'عنوان پست الزامی است.';
    }
    setErrors(localErrors);
    setServerMessage(undefined);
    if (Object.keys(localErrors).length > 0) {
      return;
    }

    onSubmit(input);
  };

  return (
    <ScrollView style={styles.wrapper} keyboardShouldPersistTaps="handled">
      {serverMessage ? <ErrorBanner message={serverMessage} /> : null}
      {errorMessage ? <ErrorBanner message={errorMessage} /> : null}

      <View style={styles.group}>
        <Text style={styles.groupTitle}>اطلاعات شخصی</Text>
        <FormField label="نام و نام خانوادگی" value={fullName} onChangeText={setFullName} required error={errors.fullName} />
        <FormField
          label="کد ملی"
          value={nationalId}
          onChangeText={setNationalId}
          required
          numeric
          error={errors.nationalId}
          placeholder="۱۰ رقم بدون خط تیره"
        />
        <FormField label="تلفن همراه" value={phone} onChangeText={setPhone} error={errors.phone} />
        <FormField label="رایانامه" value={email} onChangeText={setEmail} error={errors.email} />
      </View>

      <View style={styles.group}>
        <Text style={styles.groupTitle}>اطلاعات شغلی</Text>

        <Text style={styles.label}>
          واحد سازمانی <Text style={{ color: theme.colors.red }}>*</Text>
        </Text>
        <View style={styles.choices}>
          {departments.map(department => (
            <View
              key={department.id}
              style={[styles.choice, departmentId === department.id ? styles.choiceActive : null]}>
              <Text
                style={[styles.choiceText, departmentId === department.id ? styles.choiceTextActive : null]}
                onPress={() => setDepartmentId(department.id)}>
                {department.name}
              </Text>
            </View>
          ))}
        </View>
        {errors.departmentId ? (
          <Text style={{ color: theme.colors.red, fontSize: theme.typography.size.xxs, ...rtlText }}>{errors.departmentId}</Text>
        ) : null}

        <FormField label="عنوان پست" value={position} onChangeText={setPosition} required error={errors.position} />
        <FormField
          label="تاریخ استخدام (شمسی)"
          value={hireDate}
          onChangeText={setHireDate}
          required
          error={errors.hireDate}
          placeholder="۱۴۰۴/۰۷/۰۱"
        />
      </View>

      <View style={styles.group}>
        <Text style={styles.groupTitle}>اطلاعات قرارداد</Text>

        <Text style={styles.label}>نوع قرارداد</Text>
        <View style={styles.choices}>
          {CONTRACT_TYPES.map(type => (
            <View key={type} style={[styles.choice, contractType === type ? styles.choiceActive : null]}>
              <Text
                style={[styles.choiceText, contractType === type ? styles.choiceTextActive : null]}
                onPress={() => setContractType(type)}>
                {type}
              </Text>
            </View>
          ))}
        </View>
        {errors.contractType ? (
          <Text style={{ color: theme.colors.red, fontSize: theme.typography.size.xxs, ...rtlText }}>{errors.contractType}</Text>
        ) : null}

        <Text style={styles.label}>وضعیت</Text>
        <View style={styles.choices}>
          {STATUSES.map(item => (
            <View key={item.key} style={[styles.choice, status === item.key ? styles.choiceActive : null]}>
              <Text
                style={[styles.choiceText, status === item.key ? styles.choiceTextActive : null]}
                onPress={() => setStatus(item.key)}>
                {item.labelFa}
              </Text>
            </View>
          ))}
        </View>

        <FormField
          label="حقوق پایه (ریال)"
          value={baseSalary}
          onChangeText={setBaseSalary}
          numeric
          error={errors.baseSalary}
        />
      </View>

      <View style={styles.footer}>
        <Button label={editing ? 'ذخیره تغییرات' : 'ایجاد کارمند'} onPress={submit} loading={busy} />
        <Button label="انصراف" variant="secondary" onPress={onCancel} />
      </View>
    </ScrollView>
  );
}
