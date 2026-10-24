/**
 * تنگهٔ ورود انواع API.
 *
 * انواع از `contracts/generated/api.types.ts` می‌آیند که با `npm run codegen` از همان
 * OpenAPI سرور تولید می‌شود. هیچ نوعی دستی نوشته نمی‌شود تا کلاینت و سرور از هم جدا
 * نیفتند؛ دروازهٔ CI هم انحراف قرارداد را می‌گیرد.
 */
export type {
  Employee,
  EmployeeInput,
  EmployeePage,
  Department,
  Session,
  User,
  Widget,
  ChangeItem,
  ChangeSet,
  Mutation,
  RejectedMutation,
  MutationResult,
  Error as ApiErrorBody,
} from '@contracts/api.types';

export { API_ROUTES } from '@contracts/api.types';
