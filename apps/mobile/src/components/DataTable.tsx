/**
 * جدول داده — مطابق دستور ۳-۲ سند طراحی.
 *
 * الزامات پیاده‌شده:
 * - سرستون‌های مرتب‌شدنی (با کلیک روی عنوان ستون)
 * - شمارش رکورد در بالای جدول
 * - صفحه‌بندی در پایین جدول
 * - ترتیب ستون‌ها از راست به چپ (`row-reverse`)
 * - ردیف قابل کلیک برای باز شدن پرونده
 */

import { useMemo, type ReactNode } from 'react';
import { Pressable, ScrollView, StyleSheet, Text, View } from 'react-native';

import { useTheme } from '../design/theme';
import { faDigits } from '../design/rtl';

/** تعریف یک ستون. */
export interface Column<T> {
  key: string;
  title: string;
  /** آیا این ستون مرتب‌شدنی است. */
  sortable?: boolean;
  /** عرض نسبی ستون. */
  flex?: number;
  /** قالب‌بندی مقدار سلول. */
  render?: (row: T) => ReactNode;
}

interface Props<T> {
  columns: Column<T>[];
  rows: T[];
  rowKey: (row: T) => string;
  total: number;
  page: number;
  size: number;
  sort?: string;
  onSortChange?: (sort: string) => void;
  onPageChange?: (page: number) => void;
  onRowPress?: (row: T) => void;
  /** پیام نمایش در حالت خالی. */
  emptyMessage?: string;
}

export default function DataTable<T>({
  columns,
  rows,
  rowKey,
  total,
  page,
  size,
  sort,
  onSortChange,
  onPageChange,
  onRowPress,
  emptyMessage = 'رکوردی برای نمایش وجود ندارد.',
}: Props<T>) {
  const { theme } = useTheme();
  const totalPages = Math.max(1, Math.ceil(total / Math.max(1, size)));

  const styles = useMemo(
    () =>
      StyleSheet.create({
        wrapper: {
          backgroundColor: theme.colors.surface,
          borderWidth: 1,
          borderColor: theme.colors.border,
          borderRadius: theme.layout.radius,
          overflow: 'hidden',
        },
        header: {
          flexDirection: 'row-reverse',
          justifyContent: 'space-between',
          alignItems: 'center',
          paddingHorizontal: theme.spacing.lg,
          paddingVertical: theme.spacing.md,
          borderBottomWidth: 1,
          borderBottomColor: theme.colors.border,
        },
        count: { color: theme.colors.muted, fontSize: theme.typography.size.xs },
        head: {
          flexDirection: 'row-reverse',
          backgroundColor: theme.colors.bg,
          paddingHorizontal: theme.spacing.md,
          paddingVertical: theme.spacing.sm,
        },
        headCell: { flexDirection: 'row-reverse', alignItems: 'center', gap: 4 },
        headText: { color: theme.colors.muted, fontWeight: '700', fontSize: theme.typography.size.xs, textAlign: 'right' },
        sortMark: { color: theme.colors.primary, fontSize: theme.typography.size.xxs },
        row: {
          flexDirection: 'row-reverse',
          paddingHorizontal: theme.spacing.md,
          paddingVertical: theme.spacing.md,
          borderTopWidth: 1,
          borderTopColor: theme.colors.border,
        },
        rowPressed: { backgroundColor: theme.colors.primaryLight },
        cell: { justifyContent: 'center' },
        cellText: { color: theme.colors.text, textAlign: 'right', fontSize: theme.typography.size.sm },
        empty: { padding: theme.spacing.xl, alignItems: 'center' },
        emptyText: { color: theme.colors.muted, textAlign: 'center' },
        footer: {
          flexDirection: 'row-reverse',
          justifyContent: 'space-between',
          alignItems: 'center',
          paddingHorizontal: theme.spacing.lg,
          paddingVertical: theme.spacing.md,
          borderTopWidth: 1,
          borderTopColor: theme.colors.border,
        },
        pageInfo: { color: theme.colors.muted, fontSize: theme.typography.size.xs },
        pager: { flexDirection: 'row-reverse', gap: theme.spacing.sm },
        pagerButton: {
          borderWidth: 1,
          borderColor: theme.colors.border,
          borderRadius: theme.layout.radiusSm,
          paddingHorizontal: theme.spacing.md,
          paddingVertical: 4,
        },
        pagerButtonText: { color: theme.colors.text, fontSize: theme.typography.size.xs },
        pagerDisabled: { opacity: 0.4 },
      }),
    [theme],
  );

  const toggleSort = (column: Column<T>) => {
    if (!column.sortable || !onSortChange) {
      return;
    }
    const [currentField, currentDir] = (sort ?? '').split(':');
    const nextDir = currentField === column.key && currentDir !== 'desc' ? 'desc' : 'asc';
    onSortChange(`${column.key}:${nextDir}`);
  };

  const sortMark = (column: Column<T>): string => {
    const [field, dir] = (sort ?? '').split(':');
    if (field !== column.key) {
      return '';
    }
    return dir === 'desc' ? '▼' : '▲';
  };

  return (
    <View style={styles.wrapper}>
      <View style={styles.header}>
        <Text style={styles.count}>
          {faDigits(total)} رکورد — صفحهٔ {faDigits(page)} از {faDigits(totalPages)}
        </Text>
      </View>

      <View style={styles.head}>
        {columns.map(column => (
          <Pressable
            key={column.key}
            style={[styles.headCell, { flex: column.flex ?? 1 }]}
            onPress={() => toggleSort(column)}
            disabled={!column.sortable}>
            <Text style={styles.headText}>{column.title}</Text>
            {column.sortable ? <Text style={styles.sortMark}>{sortMark(column) || '⇅'}</Text> : null}
          </Pressable>
        ))}
      </View>

      {rows.length === 0 ? (
        <View style={styles.empty}>
          <Text style={styles.emptyText}>{emptyMessage}</Text>
        </View>
      ) : (
        <ScrollView>
          {rows.map(row => (
            <Pressable
              key={rowKey(row)}
              style={({ pressed }) => [styles.row, pressed && onRowPress ? styles.rowPressed : null]}
              onPress={() => onRowPress?.(row)}
              accessibilityRole={onRowPress ? 'button' : undefined}>
              {columns.map(column => (
                <View key={column.key} style={[styles.cell, { flex: column.flex ?? 1 }]}>
                  {column.render ? column.render(row) : <Text style={styles.cellText}>—</Text>}
                </View>
              ))}
            </Pressable>
          ))}
        </ScrollView>
      )}

      {onPageChange ? (
        <View style={styles.footer}>
          <Text style={styles.pageInfo}>
            نمایش {faDigits(rows.length)} رکورد از {faDigits(total)}
          </Text>
          <View style={styles.pager}>
            <Pressable
              style={[styles.pagerButton, page <= 1 ? styles.pagerDisabled : null]}
              disabled={page <= 1}
              onPress={() => onPageChange(page - 1)}>
              <Text style={styles.pagerButtonText}>قبلی</Text>
            </Pressable>
            <Pressable
              style={[styles.pagerButton, page >= totalPages ? styles.pagerDisabled : null]}
              disabled={page >= totalPages}
              onPress={() => onPageChange(page + 1)}>
              <Text style={styles.pagerButtonText}>بعدی</Text>
            </Pressable>
          </View>
        </View>
      ) : null}
    </View>
  );
}
