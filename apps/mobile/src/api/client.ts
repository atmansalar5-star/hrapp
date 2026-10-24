/**
 * کلاینت HTTP سامانه.
 *
 * نکات طراحی:
 * - خطاها به `ApiError` تبدیل می‌شوند که کد ماشین‌خوان، پیام فارسی و خطاهای
 *   فیلد‌به‌فیلد سرور را نگه می‌دارد؛ فرم‌ها همان `details` را کنار فیلدها نشان می‌دهند.
 * - نبود شبکه به `OfflineError` تبدیل می‌شود تا لایهٔ بالاتر تغییر را در صف آفلاین
 *   هسته ثبت کند، نه اینکه خطای فنی به کاربر نشان دهد.
 * - توکن دسترسی از بیرون تزریق می‌شود (هیچ نشستی در حافظهٔ پنهان نمی‌ماند).
 */

import { API_ROUTES } from './contracts';
import type {
  ChangeSet,
  Department,
  Employee,
  EmployeeInput,
  EmployeePage,
  Mutation,
  MutationResult,
  Session,
  Widget,
} from './contracts';
import type { Role } from '../core/types';

/** خطای برگشتی سرور با پیام فارسی. */
export class ApiError extends Error {
  readonly status: number;
  readonly code: string;
  readonly details: Record<string, string>;

  constructor(status: number, code: string, message: string, details: Record<string, string> = {}) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
    this.code = code;
    this.details = details;
  }

  /** آیا خطای اعتبارسنجی فیلد‌به‌فیلد است. */
  get isValidation(): boolean {
    return this.code === 'validation_failed';
  }

  /** آیا دسترسی نقش کافی نبوده است. */
  get isForbidden(): boolean {
    return this.code === 'forbidden';
  }
}

/** نبود ارتباط با سرور. */
export class OfflineError extends Error {
  constructor(cause?: unknown) {
    super('ارتباط با سرور برقرار نشد.');
    this.name = 'OfflineError';
    if (cause !== undefined) {
      (this as { cause?: unknown }).cause = cause;
    }
  }
}

/** پیکربندی کلاینت. */
export interface ClientConfig {
  baseUrl: string;
  /** توکن دسترسی جاری (اختیاری). */
  token?: string;
  /** تابع دریافت توکن تازه در صورت انقضای نشست. */
  onUnauthorized?: () => void;
}

/** تطبیق مسیر قراردادی با بخش‌های {پارامتر} — مسیر `/records/{id}` با شناسهٔ واقعی می‌خواند. */
function routeMatches(pattern: string, path: string): boolean {
  if (pattern === path) {
    return true;
  }
  const patternParts = pattern.split('/');
  const pathParts = path.split('/');
  if (patternParts.length !== pathParts.length) {
    return false;
  }
  return patternParts.every((segment, index) =>
    segment.startsWith('{') && segment.endsWith('}') ? (pathParts[index] ?? '').length > 0 : segment === pathParts[index],
  );
}

/** مسیرها را از قرارداد تولیدشده برمی‌دارد تا رشته دستی نوشته نشود. */
function ensureRoute(method: string, path: string): void {
  const known = API_ROUTES.some(route => route.method === method && routeMatches(route.path, path));
  if (!known) {
    throw new Error(`مسیر در قرارداد API نیست: ${method} ${path} — ابتدا contracts/openapi/hrms.json را به‌روز کنید.`);
  }
}

/** کلاینت API. */
export function createClient(config: ClientConfig) {
  let token = config.token;

  const setToken = (next?: string) => {
    token = next;
  };

  async function request<T>(method: string, path: string, body?: unknown, query?: Record<string, string | number | undefined>): Promise<T> {
    ensureRoute(method, path);

    let url = `${config.baseUrl.replace(/\/$/, '')}${path}`;
    if (query) {
      const params = Object.entries(query)
        .filter(([, v]) => v !== undefined && v !== '')
        .map(([k, v]) => `${encodeURIComponent(k)}=${encodeURIComponent(String(v))}`);
      if (params.length > 0) {
        url += `?${params.join('&')}`;
      }
    }

    const headers: Record<string, string> = { Accept: 'application/json' };
    if (body !== undefined) {
      headers['Content-Type'] = 'application/json; charset=utf-8';
    }
    if (token) {
      headers.Authorization = `Bearer ${token}`;
    }

    let response: Response;
    try {
      response = await fetch(url, {
        method,
        headers,
        body: body === undefined ? undefined : JSON.stringify(body),
      });
    } catch (error) {
      throw new OfflineError(error);
    }

    if (response.status === 401) {
      config.onUnauthorized?.();
    }

    if (response.status === 204) {
      return undefined as T;
    }

    const text = await response.text();
    const payload = text ? (JSON.parse(text) as Record<string, unknown>) : {};

    if (!response.ok) {
      const code = typeof payload.code === 'string' ? payload.code : `http_${response.status}`;
      const message = typeof payload.message === 'string' ? payload.message : 'خطای نامشخص سرور';
      const details = (payload.details ?? {}) as Record<string, string>;
      throw new ApiError(response.status, code, message, details);
    }

    return payload as T;
  }

  return {
    setToken,

    /** ورود و دریافت نشست. */
    async login(username: string, password: string): Promise<Session> {
      const session = await request<Session>('POST', '/api/v1/auth/login', { username, password });
      setToken(session.accessToken);
      return session;
    },

    /** تمدید نشست. */
    async refresh(refreshToken: string): Promise<Session> {
      const session = await request<Session>('POST', '/api/v1/auth/refresh', { refreshToken });
      setToken(session.accessToken);
      return session;
    },

    /** کاربر جاری، منوی نقش‌محور و دامنهٔ دادهٔ هر ماژول. */
    me(): Promise<{
      user: { username: string; displayName: string; role: Role; employeeId?: string; departmentId?: string };
      roleFa: string;
      menu: Array<{
        moduleId: string;
        titleFa: string;
        order: number;
        screens: Array<{ key: string; titleFa: string; component: string; icon: string }>;
        /** فیلدهای گروه‌بندی منو (حوزه > اداره > زیرواحد) — همان چیزی که سرور می‌فرستد. */
        directorate?: string;
        department?: string;
        departmentTitleFa?: string;
        subUnit?: string;
        subUnitTitleFa?: string;
        shared?: boolean;
      }>;
      scopes: Record<string, string>;
      syncSeq: number;
      modules: number;
    }> {
      return request('GET', '/api/v1/auth/me');
    },

    /** فهرست کارکنان با جستجو، فیلتر، مرتب‌سازی و صفحه‌بندی. */
    listEmployees(params: {
      query?: string;
      departmentId?: string;
      status?: string;
      contractType?: string;
      sort?: string;
      page?: number;
      size?: number;
    }): Promise<EmployeePage> {
      return request('GET', '/api/v1/employee/employees', undefined, params);
    },

    /** پروندهٔ کارمند. */
    getEmployee(id: string): Promise<Employee> {
      return request('GET', '/api/v1/employee/employees/{id}'.replace('{id}', encodeURIComponent(id)));
    },

    /** ایجاد کارمند. */
    createEmployee(input: EmployeeInput): Promise<Employee> {
      return request('POST', '/api/v1/employee/employees', input);
    },

    /** ویرایش کارمند. */
    updateEmployee(id: string, input: EmployeeInput): Promise<Employee> {
      return request('PATCH', '/api/v1/employee/employees/{id}'.replace('{id}', encodeURIComponent(id)), input);
    },

    /** حذف کارمند. */
    deleteEmployee(id: string): Promise<void> {
      return request('DELETE', '/api/v1/employee/employees/{id}'.replace('{id}', encodeURIComponent(id)));
    },

    /** واحدهای سازمانی (مخزن اصلی تا پایان فاز ۳ در ماژول امور کارکنان). */
    departments(): Promise<Department[]> {
      return request('GET', '/api/v1/employee/departments');
    },

    /** درخت واحدهای سازمانی از ماژول ساختار سازمانی. */
    orgUnits(): Promise<Department[]> {
      return request('GET', '/api/v1/orgstructure/units');
    },

    /** شناسنامهٔ پست‌ها. */
    positions(): Promise<Department[]> {
      return request('GET', '/api/v1/orgstructure/positions');
    },

    /**
     * پایانه‌های رکوردکیت (recordkit) هر ماژول — مبنای صفحات پویا.
     *
     * شناسهٔ ماژول (`recordSchema`) فیلدها، وضعیت‌ها و اقدام‌ها را می‌دهد تا فرم و
     * جدول از همان ساخته شود؛ هیچ برچسب یا قاعده‌ای در کلاینت تکرار نمی‌شود.
     */
    /** شناسهٔ ماژول رکوردگان (انواع رکورد، فیلدها، وضعیت‌ها، اقدام‌ها). */
    recordSchema<T>(prefix: string): Promise<T> {
      return request<T>('GET', `${prefix}/schema`);
    },

    /** فهرست رکوردهای یک نوع با فیلتر سمت سرور. */
    listRecords<T>(prefix: string, kind: string, params: { query?: string; status?: string; size?: number } = {}): Promise<T> {
      return request<T>('GET', `${prefix}/records`, undefined, { kind, ...params });
    },

    /** ایجاد رکورد تازه (ورودی همان RecordInput سرور: kind/title/employeeId/fields/noteFa). */
    createRecord<T>(prefix: string, input: unknown): Promise<T> {
      return request<T>('POST', `${prefix}/records`, input);
    },

    /** ویرایش رکورد (تنها در وضعیت‌های مجاز ویرایش — قاعده در سرور اعمال می‌شود). */
    updateRecord<T>(prefix: string, id: string, input: unknown): Promise<T> {
      return request<T>('PATCH', `${prefix}/records/${encodeURIComponent(id)}`, input);
    },

    /** اقدام گردش‌کار روی رکورد (جاری‌سازی، تأیید، رد، قفل و …). */
    actRecord<T>(prefix: string, id: string, action: string, noteFa?: string): Promise<T> {
      return request<T>('POST', `${prefix}/records/${encodeURIComponent(id)}/action`, { action, noteFa });
    },

    /** حذف رکورد. */
    deleteRecord(prefix: string, id: string): Promise<void> {
      return request<void>('DELETE', `${prefix}/records/${encodeURIComponent(id)}`);
    },

    /** کارت‌های آماری داشبورد بر اساس نقش. */
    widgets(): Promise<Widget[]> {
      return request('GET', '/api/v1/dashboard/widgets');
    },

    /** دلتای تغییرات برای هم‌گام‌سازی. */
    syncChanges(since: number): Promise<ChangeSet> {
      return request('GET', '/api/v1/sync/changes', undefined, { since });
    },

    /** فرستادن عملیات‌های ذخیره‌شده در حالت آفلاین. */
    syncMutations(mutations: Mutation[]): Promise<MutationResult> {
      return request('POST', '/api/v1/sync/mutations', mutations);
    },
  };
}

/** نوع کلاینت برای تزریق در کامپوننت‌ها. */
export type Client = ReturnType<typeof createClient>;
