/**
 * ساختار سازمانی — نمودار درختی واحدها و پست‌ها (واحد برنامه‌ریزی نیروی انسانی).
 *
 * داده از سرور می‌آید که هر گره می‌تواند والد داشته باشد؛ درخت در همان‌جا ساخته
 * می‌شود تا کلاینت منطق سازمانی نسازد. مسیر داده در `features/orgstructure/api`
 * تعیین می‌شود و تا فاز ۳ به مخزن واحدهای ماژول امور کارکنان وصل است.
 */

import { useCallback, useEffect, useMemo, useState } from 'react';
import { ScrollView, StyleSheet, Text, View } from 'react-native';

import type { Department } from '../../api/contracts';
import { useAuth } from '../../auth/AuthContext';
import { ErrorBanner, Loading } from '../../components/ui';
import { faDigits, rtlText } from '../../design/rtl';
import { useTheme } from '../../design/theme';
import { fetchOrgUnits } from './api';

interface Node {
  department: Department;
  children: Node[];
}

/** ساخت درخت از فهرست تخت واحدها. */
export function buildTree(departments: Department[]): Node[] {
  const nodes = new Map<string, Node>();
  departments.forEach(department => nodes.set(department.id, { department, children: [] }));

  const roots: Node[] = [];
  nodes.forEach(node => {
    const parentId = node.department.parentId;
    const parent = parentId ? nodes.get(parentId) : undefined;
    if (parent) {
      parent.children.push(node);
    } else {
      roots.push(node);
    }
  });
  return roots;
}

export default function OrgChartScreen() {
  const { theme } = useTheme();
  const { client } = useAuth();
  const [departments, setDepartments] = useState<Department[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | undefined>(undefined);

  const load = useCallback(async () => {
    setLoading(true);
    setError(undefined);
    try {
      setDepartments(await fetchOrgUnits(client));
    } catch {
      setError('دریافت ساختار سازمانی ناموفق بود.');
      setDepartments([]);
    } finally {
      setLoading(false);
    }
  }, [client]);

  useEffect(() => {
    void load();
  }, [load]);

  const tree = useMemo(() => buildTree(departments), [departments]);

  const styles = useMemo(
    () =>
      StyleSheet.create({
        node: {
          backgroundColor: theme.colors.surface,
          borderWidth: 1,
          borderColor: theme.colors.border,
          borderRadius: theme.layout.radiusSm,
          padding: theme.spacing.md,
          marginBottom: theme.spacing.sm,
          flexDirection: 'row-reverse',
          alignItems: 'center',
          justifyContent: 'space-between',
          gap: theme.spacing.md,
        },
        child: { marginRight: theme.spacing.xl, borderRightWidth: 2, borderRightColor: theme.colors.border, paddingRight: theme.spacing.md },
        name: { color: theme.colors.text, fontWeight: '700', ...rtlText },
        meta: { color: theme.colors.muted, fontSize: theme.typography.size.xs, ...rtlText },
        headcount: {
          backgroundColor: theme.colors.bg,
          borderRadius: 999,
          paddingHorizontal: theme.spacing.md,
          paddingVertical: 2,
        },
        headcountText: { color: theme.colors.text, fontSize: theme.typography.size.xxs, fontWeight: '700' },
      }),
    [theme],
  );

  const renderNode = (node: Node) => (
    <View key={node.department.id}>
      <View style={styles.node}>
        <View>
          <Text style={styles.name}>{node.department.name}</Text>
          <Text style={styles.meta}>سرپرست: {node.department.head || '—'}</Text>
        </View>
        <View style={styles.headcount}>
          <Text style={styles.headcountText}>{faDigits(node.department.headcount ?? 0)} نفر</Text>
        </View>
      </View>
      {node.children.length > 0 ? <View style={styles.child}>{node.children.map(renderNode)}</View> : null}
    </View>
  );

  if (loading) {
    return <Loading label="در حال دریافت ساختار سازمانی…" />;
  }

  return (
    <ScrollView>
      {error ? <ErrorBanner message={error} onRetry={load} /> : null}
      {tree.map(renderNode)}
      <View style={styles.node}>
        <Text style={styles.meta}>
          افزودن و ویرایش واحد در فاز بعدی به همین صفحه اضافه می‌شود؛ قرارداد API و دسترسی نقش‌محور آن آماده است.
        </Text>
      </View>
    </ScrollView>
  );
}
