/**
 * آزمون کلاینت API.
 *
 * نکتهٔ مهم آزمون‌شده: مسیرها از قرارداد تولیدشده خوانده می‌شوند، پس اگر کسی مسیری
 * خارج از قرارداد بنویسد، در همان‌جا خطا گرفته می‌شود (و در CI هم شکست می‌خورد).
 */

import { ApiError, OfflineError, createClient } from '../api/client';
import { API_ROUTES } from '../api/contracts';
import { MODULES } from '../navigation/modules.generated';

const originalFetch = globalThis.fetch;

afterEach(() => {
  globalThis.fetch = originalFetch;
  jest.clearAllMocks();
});

describe('کلاینت API', () => {
  it('همهٔ مسیرهای مورد استفادهٔ کلاینت در قرارداد هستند', () => {
    const routes = API_ROUTES.map(r => r.path);
    expect(routes).toContain('/api/v1/auth/login');
    expect(routes).toContain('/api/v1/auth/me');
    expect(routes).toContain('/api/v1/employee/employees');
    expect(routes).toContain('/api/v1/employee/employees/{id}');
    expect(routes).toContain('/api/v1/dashboard/widgets');
    expect(routes).toContain('/api/v1/sync/mutations');
    expect(routes).not.toContain('/api/v1/nope');

    // و هر مسیر قرارداد به یک پیشوند ماژولِ اعلام‌شده در مانیفست (یا auth/sync) تعلق دارد؛
    // پیشوندها از خود مانیفست ساخته می‌شوند تا با افزودن ماژول تازه کهنه نشود.
    // `/api/v1/orgs` هم سرویس سطح سکو (فهرست شرکت‌ها/هلدینگ‌ها روی bootstrap) است، نه ماژول منو.
    const prefixes = ['/api/v1/auth', '/api/v1/sync', '/api/v1/orgs', ...MODULES.map(module => module.apiPrefix)];
    routes.forEach(path => {
      const known = path === '/healthz' || prefixes.some(prefix => path.startsWith(prefix));
      expect(`${path} → ${known}`).toBe(`${path} → true`);
    });
  });

  it('پاسخ موفق فهرست کارکنان را برمی‌گرداند', async () => {
    globalThis.fetch = jest.fn().mockResolvedValue({
      ok: true,
      status: 200,
      text: async () =>
        JSON.stringify({ items: [{ id: 'EMP-001', fullName: 'سارا محمدی', status: 'active' }], total: 1, page: 1, size: 20 }),
    }) as unknown as typeof fetch;

    const client = createClient({ baseUrl: 'http://localhost:8080' });
    const page = await client.listEmployees({ page: 1 });
    expect(page.total).toBe(1);
    expect(page.items[0]!.fullName).toBe('سارا محمدی');
  });

  it('خطای اعتبارسنجی سرور را با جزئیات فیلد برمی‌گرداند', async () => {
    globalThis.fetch = jest.fn().mockResolvedValue({
      ok: false,
      status: 422,
      text: async () =>
        JSON.stringify({ code: 'validation_failed', message: 'اطلاعات ارسالی معتبر نیست.', details: { nationalId: 'کد ملی نامعتبر است.' } }),
    }) as unknown as typeof fetch;

    const client = createClient({ baseUrl: 'http://localhost:8080' });
    try {
      await client.createEmployee({
        fullName: 'الف',
        nationalId: '1',
        departmentId: 'D1',
        position: 'کارشناس',
        contractType: 'تمام‌وقت',
        hireDate: '1404/07/01',
      });
      throw new Error('باید خطا پرتاب می‌شد');
    } catch (error) {
      expect(error).toBeInstanceOf(ApiError);
      const apiError = error as ApiError;
      expect(apiError.isValidation).toBe(true);
      expect(apiError.details.nationalId).toContain('کد ملی');
      expect(apiError.status).toBe(422);
    }
  });

  it('خطای دسترسی را به forbidden تبدیل می‌کند', async () => {
    globalThis.fetch = jest.fn().mockResolvedValue({
      ok: false,
      status: 403,
      text: async () => JSON.stringify({ code: 'forbidden', message: 'نقش شما اجازهٔ این عملیات را ندارد.' }),
    }) as unknown as typeof fetch;

    const client = createClient({ baseUrl: 'http://localhost:8080' });
    await expect(client.deleteEmployee('EMP-001')).rejects.toMatchObject({ code: 'forbidden' });
  });

  it('قطع ارتباط را به OfflineError تبدیل می‌کند', async () => {
    globalThis.fetch = jest.fn().mockRejectedValue(new Error('network down')) as unknown as typeof fetch;
    const client = createClient({ baseUrl: 'http://localhost:8080' });
    await expect(client.widgets()).rejects.toBeInstanceOf(OfflineError);
  });

  it('توکن را در هدر Authorization می‌فرستد', async () => {
    const fetchMock = jest.fn().mockResolvedValue({
      ok: true,
      status: 200,
      text: async () => JSON.stringify({ accessToken: 'a', refreshToken: 'b', expiresAt: 1, user: { username: 'x', displayName: 'y', role: 'admin' } }),
    });
    globalThis.fetch = fetchMock as unknown as typeof fetch;

    const client = createClient({ baseUrl: 'http://localhost:8080' });
    await client.login('admin', 'demo');
    await client.me();

    const lastCall = fetchMock.mock.calls[fetchMock.mock.calls.length - 1];
    expect((lastCall?.[1] as { headers: Record<string, string> }).headers.Authorization).toBe('Bearer a');
  });
});
