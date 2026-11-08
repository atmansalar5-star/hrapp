/**
 * ثبت کامپوننت صفحه‌ها.
 *
 * کلید این فایل، رشتهٔ `component` در مانیفست ماژول‌ها است. ابزار بررسی
 * (`tools/check/verify.mjs`) وجود هر فایل کامپوننت را در پوشهٔ ماژول متناظر الزام
 * می‌کند، پس امکان ندارد مانیفست صفحه‌ای را اعلام کند که کامپوننتش ساخته نشده باشد.
 */

import type { ComponentType } from 'react';

// داشبورد
import DashboardScreen from '../features/dashboard/DashboardScreen';

// واحد امور کارکنان — کارکنان
import EmployeeListScreen from '../features/employee/EmployeeListScreen';
import EmployeeProfileScreen from '../features/employee/EmployeeProfileScreen';
import PersonnelServicesScreen from '../features/employee/PersonnelServicesScreen';

// واحد امور کارکنان — کارگزینی و انعقاد قراردادها
import ContractsScreen from '../features/contracts/ContractsScreen';
import ContractLifecycleScreen from '../features/contracts/ContractLifecycleScreen';
import ContractTemplatesScreen from '../features/contracts/ContractTemplatesScreen';

// واحد امور کارکنان — افراد تحت تکفل و درخواست پرواز
import DependentsScreen from '../features/staff_services/DependentsScreen';
import FlightRequestsScreen from '../features/staff_services/FlightRequestsScreen';

// واحد برنامه‌ریزی نیروی انسانی — ساختار سازمانی
import OrgChartScreen from '../features/orgstructure/OrgChartScreen';
import PositionsScreen from '../features/orgstructure/PositionsScreen';

// واحد برنامه‌ریزی نیروی انسانی — انتصابات، ترفیعات و کارراههٔ شغلی
import AppointmentFormScreen from '../features/career/AppointmentFormScreen';
import AppointmentsScreen from '../features/career/AppointmentsScreen';
import PromotionsScreen from '../features/career/PromotionsScreen';
import CareerPathScreen from '../features/career/CareerPathScreen';

// استخدام
import HiringRequestsScreen from '../features/recruitment/HiringRequestsScreen';
import JobPostingsScreen from '../features/recruitment/JobPostingsScreen';
import CandidatesScreen from '../features/recruitment/CandidatesScreen';
import HiringPipelineScreen from '../features/recruitment/HiringPipelineScreen';

// حضور و غیاب
import AttendanceScreen from '../features/attendance/AttendanceScreen';
import LeaveRequestsScreen from '../features/attendance/LeaveRequestsScreen';
import LeaveApprovalScreen from '../features/attendance/LeaveApprovalScreen';
import OvertimeScreen from '../features/attendance/OvertimeScreen';
import WorkCalendarScreen from '../features/attendance/WorkCalendarScreen';
import MissionsScreen from '../features/attendance/MissionsScreen';
import MissionCalcScreen from '../features/attendance/MissionCalcScreen';

// حقوق و دستمزد
import PayrollRunScreen from '../features/payroll/PayrollRunScreen';
import OilPayslipScreen from '../features/payroll/OilPayslipScreen';
import PayslipScreen from '../features/payroll/PayslipScreen';
import PayItemsScreen from '../features/payroll/PayItemsScreen';
import KaranaScreen from '../features/payroll/KaranaScreen';
import PayReportsScreen from '../features/payroll/PayReportsScreen';
import PayParamsScreen from '../features/payroll/PayParamsScreen';
import ServiceBonusScreen from '../features/payroll/ServiceBonusScreen';
import PayGapScreen from '../features/payroll/PayGapScreen';
// فاز ۱: فایل قانونی بیمه/مالیات و فایل پرداخت بانکی (زیرکارهای ۱.۴ و ۱.۵).
import StatutoryFilesScreen from '../features/payroll/StatutoryFilesScreen';
import BankFilesScreen from '../features/payroll/BankFilesScreen';

// جبران خدمات — احکام حقوق و گواهی اشتغال
import SalaryDecreesScreen from '../features/salary_admin/SalaryDecreesScreen';
import EmploymentCertsScreen from '../features/salary_admin/EmploymentCertsScreen';

// واحد روابط کار و مددکاری اجتماعی — کمیتهٔ انضباط کار
import DisciplineCasesScreen from '../features/discipline/DisciplineCasesScreen';
import DisciplineMeetingsScreen from '../features/discipline/DisciplineMeetingsScreen';

// واحد روابط کار و مددکاری اجتماعی — مددکاری
import SocialWorkCasesScreen from '../features/socialwork/SocialWorkCasesScreen';
import AidRequestsScreen from '../features/socialwork/AidRequestsScreen';

// واحد روابط کار و مددکاری اجتماعی — درمان و خدمات درمانی
import TreatmentCasesScreen from '../features/medical/TreatmentCasesScreen';
import PeriodicExamsScreen from '../features/medical/PeriodicExamsScreen';
import MedicalServicesScreen from '../features/medical/MedicalServicesScreen';
import SupplementaryInsuranceScreen from '../features/medical/SupplementaryInsuranceScreen';

// آموزش (واحد هماهنگی آموزش)
import TrainingPlanScreen from '../features/training/TrainingPlanScreen';
import TrainingNeedsScreen from '../features/training/TrainingNeedsScreen';
import TrainingExecutionScreen from '../features/training/TrainingExecutionScreen';
import TrainingCoursesScreen from '../features/training/TrainingCoursesScreen';
import TrainingHistoryScreen from '../features/training/TrainingHistoryScreen';

// رفاه
import WelfareFacilitiesScreen from '../features/welfare/WelfareFacilitiesScreen';
import WelfareRequestsScreen from '../features/welfare/WelfareRequestsScreen';
import SchoolServiceScreen from '../features/welfare/SchoolServiceScreen';
import WelfareToursScreen from '../features/welfare/WelfareToursScreen';
import AccommodationScreen from '../features/welfare/AccommodationScreen';

// عملکرد
import PerfCyclesScreen from '../features/performance/PerfCyclesScreen';
import PerfFormsScreen from '../features/performance/PerfFormsScreen';
import PerfResultsScreen from '../features/performance/PerfResultsScreen';

// واحد امور کارکنان — کارت‌های رفاهی و شارژ (پتروکارت، نفت‌کارت)
import PetrocardScreen from '../features/cards/PetrocardScreen';
import CardAllocationsScreen from '../features/cards/CardAllocationsScreen';
import NaftmartScreen from '../features/cards/NaftmartScreen';
import CardRatesScreen from '../features/cards/CardRatesScreen';
import OccasionsScreen from '../features/cards/OccasionsScreen';
import ChargeFilesScreen from '../features/cards/ChargeFilesScreen';

// واحد امور کارکنان — پاداش تحصیلی فرزندان ممتاز (مستقل از کارت‌های رفاهی)
import OutstandingStudentsScreen from '../features/education_bonus/OutstandingStudentsScreen';
import EducationBonusRatesScreen from '../features/education_bonus/EducationBonusRatesScreen';
import EducationBonusFilesScreen from '../features/education_bonus/EducationBonusFilesScreen';

// واحد امور کارکنان — پاداش‌ها (داشبورد مدیریتی، پرداخت و گزارش)
import BonusDashboardScreen from '../features/bonus/BonusDashboardScreen';
import BonusPaymentsScreen from '../features/bonus/BonusPaymentsScreen';
import BonusReportsScreen from '../features/bonus/BonusReportsScreen';

// گزارش‌ها
import ReportsScreen from '../features/reports/ReportsScreen';

// تنظیمات
import UsersScreen from '../features/settings/UsersScreen';
import RolesScreen from '../features/settings/RolesScreen';
import OrgProfileScreen from '../features/settings/OrgProfileScreen';
import OrgSettingsScreen from '../features/settings/OrgSettingsScreen';
import SysParamsScreen from '../features/settings/SysParamsScreen';
// نشان شرکت و QR کد آن: بارگذاری لوگو و QR شناسهٔ شرکت.
import BrandingScreen from '../features/settings/BrandingScreen';
// تبادل داده: اتصال به منابع بیرونی، ساختار داده و قالب‌های آمادهٔ بارگذاری.
import DataExchangeScreen from '../features/settings/DataExchangeScreen';
import FormParamsScreen from '../features/settings/FormParamsScreen';
import AuditLogScreen from '../features/settings/AuditLogScreen';
import DocTemplatesScreen from '../features/settings/DocTemplatesScreen';
import CatalogsScreen from '../features/settings/CatalogsScreen';
// شرکت‌ها و استقرارها: ورود شرکت تازه و ویرایش اطلاعات شرکت‌های موجود (رکورد companyProfiles).
import CompaniesScreen from '../features/settings/CompaniesScreen';

// واحد امور کارکنان — بایگانی و شناسایی کارکنان
import PersonnelFilesScreen from '../features/archive/PersonnelFilesScreen';
import EmployeeIdentityScreen from '../features/archive/EmployeeIdentityScreen';
import ArchiveDocumentsScreen from '../features/archive/ArchiveDocumentsScreen';

// واحد امور کارکنان — طبقه‌بندی مشاغل (مشترک با واحد برون‌سپاری)
import JobCategoriesScreen from '../features/jobclass/JobCategoriesScreen';
import JobEvaluationScreen from '../features/jobclass/JobEvaluationScreen';
import JobDescriptionsScreen from '../features/jobclass/JobDescriptionsScreen';
import JobRegistryScreen from '../features/jobclass/JobRegistryScreen';
import JobCardScreen from '../features/jobclass/JobCardScreen';
import JobQualificationScreen from '../features/jobclass/JobQualificationScreen';

// واحد امور کارکنان — امور وام و بازنشستگی
import LoansScreen from '../features/loans_retirement/LoansScreen';
import LoanInstallmentsScreen from '../features/loans_retirement/LoanInstallmentsScreen';
import RetirementFilesScreen from '../features/loans_retirement/RetirementFilesScreen';

// واحد روابط کار و مددکاری اجتماعی — شکایات و امور ایثارگران
import ComplaintsScreen from '../features/grievances/ComplaintsScreen';
import VeteransAffairsScreen from '../features/grievances/VeteransAffairsScreen';

// واحد آمار و گزارشات — آمار، پایش و تأیید اطلاعات پایه
import StatsOverviewScreen from '../features/statistics/StatsOverviewScreen';
import StatsSectionScreen from '../features/statistics/StatsSectionScreen';
import StatsAttendanceScreen from '../features/statistics/StatsAttendanceScreen';
import StatsLeaveScreen from '../features/statistics/StatsLeaveScreen';
import StatsMissionScreen from '../features/statistics/StatsMissionScreen';
import StatsBasePayScreen from '../features/statistics/StatsBasePayScreen';
import StatsKaranaScreen from '../features/statistics/StatsKaranaScreen';
import TransfersScreen from '../features/staffing_moves/TransfersScreen';
import IntroductionsScreen from '../features/staffing_moves/IntroductionsScreen';
import MgmtReportsScreen from '../features/reports/MgmtReportsScreen';
import ReportBuilderScreen from '../features/reports/ReportBuilderScreen';

// واحد برون‌سپاری — نیروهای ارکان ثالث و مشاغل سخت و زیان‌آور
import ContractorsScreen from '../features/outsourcing/ContractorsScreen';
import ThirdPartyPayrollScreen from '../features/outsourcing/ThirdPartyPayrollScreen';
import OutsourceAnalysisScreen from '../features/outsourcing/OutsourceAnalysisScreen';
import HardshipFilesScreen from '../features/outsourcing/HardshipFilesScreen';

// واحد مطالعات منابع انسانی
import StudiesScreen from '../features/hrstudies/StudiesScreen';
import SurveysScreen from '../features/hrstudies/SurveysScreen';
import HrIndicesScreen from '../features/hrstudies/HrIndicesScreen';
import SuggestionsScreen from '../features/hrstudies/SuggestionsScreen';

// واحد امور ورزش
import SportsProgramsScreen from '../features/sports/SportsProgramsScreen';
import SportsEventsScreen from '../features/sports/SportsEventsScreen';

/** نگاشت نام کامپوننت مانیفست به کامپوننت واقعی. */
export const SCREEN_REGISTRY: Record<string, ComponentType> = {
  DashboardScreen,
  EmployeeListScreen,
  EmployeeProfileScreen,
  PersonnelServicesScreen,
  ContractsScreen,
  ContractLifecycleScreen,
  ContractTemplatesScreen,
  DependentsScreen,
  FlightRequestsScreen,
  OrgChartScreen,
  PositionsScreen,
  AppointmentsScreen,
  AppointmentFormScreen,
  PromotionsScreen,
  CareerPathScreen,
  HiringRequestsScreen,
  JobPostingsScreen,
  CandidatesScreen,
  HiringPipelineScreen,
  AttendanceScreen,
  LeaveRequestsScreen,
  LeaveApprovalScreen,
  OvertimeScreen,
  WorkCalendarScreen,
  MissionCalcScreen,
  PayrollRunScreen,
  OilPayslipScreen,
  PayslipScreen,
  PayItemsScreen,
  PayParamsScreen,
  ServiceBonusScreen,
  PayGapScreen,
  StatutoryFilesScreen,
  BankFilesScreen,
  PayReportsScreen,
  SalaryDecreesScreen,
  EmploymentCertsScreen,
  DisciplineCasesScreen,
  DisciplineMeetingsScreen,
  SocialWorkCasesScreen,
  AidRequestsScreen,
  TreatmentCasesScreen,
  PeriodicExamsScreen,
  MedicalServicesScreen,
  SupplementaryInsuranceScreen,
  PerfCyclesScreen,
  PerfFormsScreen,
  PerfResultsScreen,
  TrainingPlanScreen,
  TrainingNeedsScreen,
  TrainingExecutionScreen,
  TrainingCoursesScreen,
  TrainingHistoryScreen,
  WelfareFacilitiesScreen,
  WelfareRequestsScreen,
  PetrocardScreen,
  CardAllocationsScreen,
  NaftmartScreen,
  CardRatesScreen,
  OccasionsScreen,
  ChargeFilesScreen,
  OutstandingStudentsScreen,
  EducationBonusRatesScreen,
  EducationBonusFilesScreen,
  BonusDashboardScreen,
  BonusPaymentsScreen,
  BonusReportsScreen,
  ReportsScreen,
  ReportBuilderScreen,
  UsersScreen,
  RolesScreen,
  OrgSettingsScreen,
  SysParamsScreen,
  BrandingScreen,
  DataExchangeScreen,
  FormParamsScreen,
  AuditLogScreen,
  DocTemplatesScreen,
  CatalogsScreen,
  CompaniesScreen,
  OrgProfileScreen,
  PersonnelFilesScreen,
  EmployeeIdentityScreen,
  ArchiveDocumentsScreen,
  JobCategoriesScreen,
  JobEvaluationScreen,
  JobDescriptionsScreen,
  JobRegistryScreen,
  JobCardScreen,
  JobQualificationScreen,
  LoansScreen,
  LoanInstallmentsScreen,
  RetirementFilesScreen,
  ComplaintsScreen,
  VeteransAffairsScreen,
  StatsOverviewScreen,
  StatsSectionScreen,
  StatsAttendanceScreen,
  StatsLeaveScreen,
  StatsMissionScreen,
  StatsBasePayScreen,
  StatsKaranaScreen,
  TransfersScreen,
  IntroductionsScreen,
  MgmtReportsScreen,
  ContractorsScreen,
  ThirdPartyPayrollScreen,
  OutsourceAnalysisScreen,
  HardshipFilesScreen,
  StudiesScreen,
  SurveysScreen,
  HrIndicesScreen,
  SuggestionsScreen,
  SportsProgramsScreen,
  SportsEventsScreen,
  MissionsScreen,
  KaranaScreen,
  SchoolServiceScreen,
  WelfareToursScreen,
  AccommodationScreen,
};

/** یافتن کامپوننت یک صفحه؛ نبودن آن یعنی ناهم‌خوانی مانیفست و کد (CI آن را می‌گیرد). */
export function resolveScreen(component: string): ComponentType | undefined {
  return SCREEN_REGISTRY[component];
}
