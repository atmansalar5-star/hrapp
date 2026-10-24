/**
 * لایهٔ دادهٔ ماژول کارکنان.
 *
 * ترتیب کار عمدی است:
 * ۱. اعتبارسنجی محلی با **هستهٔ بومی** (همان قواعدی که سرور دارد) تا خطاها فوری و
 *    کنار فیلد نمایش داده شوند و رفت‌وبرگشت شبکه لازم نباشد.
 * ۲. ارسال به سرور که مرجع نهایی است؛ خطاهای اعتبارسنجی سرور روی خطاهای محلی غلبه می‌کنند.
 * ۳. اگر شبکه نبود، تغییر در صف آفلاین هسته ثبت می‌شود و با بازگشت اتصال هم‌گام می‌گردد.
 */

import { ApiError, OfflineError, type Client } from '../../api/client';
import type { Employee, EmployeeInput, EmployeePage } from '../../api/contracts';
import { enqueueMutation, pendingMutations, markMutationsSent, markMutationRejected, validateEmployee } from '../../core/hrmsCore';

/** نتیجهٔ یک عملیات نوشتن. */
export interface WriteResult {
  employee?: Employee;
  /** خطاهای فیلد‌به‌فیلد برای نمایش کنار فیلد. */
  errors?: Record<string, string>;
  /** پیام کلی برای نمایش در بنر یا Toast. */
  message?: string;
  /** آیا تغییر در صف آفلاین ذخیره شد. */
  queued?: boolean;
}

/** فهرست کارکنان. */
export async function fetchEmployees(
  client: Client,
  params: {
    query?: string;
    departmentId?: string;
    status?: string;
    contractType?: string;
    sort?: string;
    page?: number;
    size?: number;
  },
): Promise<EmployeePage> {
  return client.listEmployees(params);
}

/** پروندهٔ کارمند. */
export async function fetchEmployee(client: Client, id: string): Promise<Employee> {
  return client.getEmployee(id);
}

/** واحدهای سازمانی. */
export async function fetchDepartments(client: Client) {
  return client.departments();
}

/** اعتبارسنجی محلی با هستهٔ بومی؛ در نبود هسته، `undefined` یعنی «اعتبارسنجی به سرور سپرده شد». */
export function localValidation(input: EmployeeInput): Record<string, string> | undefined {
  try {
    const result = validateEmployee(input);
    return result.valid ? undefined : result.errors;
  } catch {
    // هستهٔ بومی بار نشده است؛ اعتبارسنجی سرور انجام می‌شود و به‌جای قاعده‌نویسی در
    // جاوااسکریپت، هیچ بررسی محلی انجام نمی‌دهیم.
    return undefined;
  }
}

/** ایجاد کارمند. */
export async function createEmployee(client: Client, input: EmployeeInput): Promise<WriteResult> {
  const localErrors = localValidation(input);
  if (localErrors) {
    return { errors: localErrors, message: 'برخی فیلدها نامعتبر هستند.' };
  }
  try {
    return { employee: await client.createEmployee(input) };
  } catch (error) {
    return handleWriteError(error, 'create', input);
  }
}

/** ویرایش کارمند. */
export async function updateEmployee(client: Client, id: string, input: EmployeeInput): Promise<WriteResult> {
  const localErrors = localValidation(input);
  if (localErrors) {
    return { errors: localErrors, message: 'برخی فیلدها نامعتبر هستند.' };
  }
  try {
    return { employee: await client.updateEmployee(id, input) };
  } catch (error) {
    return handleWriteError(error, 'update', input, id);
  }
}

/** حذف کارمند. */
export async function deleteEmployee(client: Client, id: string): Promise<WriteResult> {
  try {
    await client.deleteEmployee(id);
    return { message: 'کارمند حذف شد.' };
  } catch (error) {
    if (error instanceof OfflineError) {
      return queueOffline({ entity: 'Employee', op: 'delete', entityId: id, payload: { id } });
    }
    if (error instanceof ApiError) {
      return { message: error.message };
    }
    throw error;
  }
}

/** فرستادن عملیات‌های صف آفلاین به سرور. */
export async function flushOfflineQueue(client: Client): Promise<{ sent: number; rejected: number; pending: number }> {
  let entries: Array<{
    opId: string;
    entity: string;
    op: 'create' | 'update' | 'delete';
    entityId: string;
    payload: unknown;
    clientTs: string;
    baseVersion?: number;
  }> = [];
  try {
    entries = pendingMutations().entries as typeof entries;
  } catch {
    return { sent: 0, rejected: 0, pending: 0 };
  }
  if (entries.length === 0) {
    return { sent: 0, rejected: 0, pending: 0 };
  }

  const result = await client.syncMutations(
    entries.map(entry => ({
      opId: entry.opId,
      entity: entry.entity,
      op: entry.op,
      id: entry.entityId,
      payload: (entry.payload ?? {}) as Record<string, unknown>,
      clientTs: entry.clientTs,
      baseVersion: entry.baseVersion,
    })),
  );

  markMutationsSent(result.accepted);
  result.rejected.forEach(rejection => markMutationRejected(rejection.opId, rejection.message ?? rejection.reason));

  return {
    sent: result.accepted.length,
    rejected: result.rejected.length,
    pending: pendingMutations().pending,
  };
}

function handleWriteError(error: unknown, op: 'create' | 'update', input: EmployeeInput, id?: string): WriteResult {
  if (error instanceof ApiError) {
    return {
      errors: error.isValidation ? error.details : undefined,
      message: error.message,
    };
  }
  if (error instanceof OfflineError) {
    return queueOffline({ entity: 'Employee', op, entityId: id ?? '', payload: input });
  }
  throw error;
}

/** ثبت تغییر در صف آفلاین هسته. */
function queueOffline(mutation: {
  entity: string;
  op: 'create' | 'update' | 'delete';
  entityId: string;
  payload: unknown;
}): WriteResult {
  const clientTs = new Date().toISOString();
  const opId = `${mutation.entity}:${mutation.entityId || 'new'}:${clientTs}`;
  try {
    const result = enqueueMutation({
      opId,
      entity: mutation.entity,
      op: mutation.op,
      entityId: mutation.entityId,
      payload: mutation.payload,
      clientTs,
    });
    return {
      queued: true,
      message: result.enqueued
        ? `اتصال برقرار نیست؛ تغییر روی دستگاه ذخیره شد و با بازگشت شبکه هم‌گام می‌شود. (${result.pending} تغییر در انتظار)`
        : 'این تغییر پیش‌تر در صف هم‌گام‌سازی ثبت شده است.',
    };
  } catch (error) {
    return { message: `اتصال برقرار نیست و ذخیرهٔ محلی ممکن نشد: ${String(error)}` };
  }
}
