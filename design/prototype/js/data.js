/* =====================================================================
   داده‌های نمونه (Demo Data) — سامانه جامع منابع انسانی
   ===================================================================== */

const DEPTS = {
  D1:{name:'فناوری اطلاعات', code:'IT'},
  D2:{name:'منابع انسانی', code:'HR'},
  D3:{name:'مالی و حسابداری', code:'FIN'},
  D4:{name:'فروش و بازاریابی', code:'SAL'},
  D5:{name:'تولید و عملیات', code:'OPS'},
  D6:{name:'تحقیق و توسعه', code:'R&D'},
};
const deptName = id => (DEPTS[id] ? DEPTS[id].name : '—');

const DEPT_TREE = [
  {id:'D1', name:'فناوری اطلاعات', head:'آرش رضایی', count:14, children:[
    {id:'D6', name:'تحقیق و توسعه', head:'آرش رضایی', count:6, children:[]}
  ]},
  {id:'D2', name:'منابع انسانی', head:'سارا محمدی', count:5, children:[]},
  {id:'D3', name:'مالی و حسابداری', head:'نرگس کریمی', count:8, children:[]},
  {id:'D4', name:'فروش و بازاریابی', head:'محمد احمدی', count:11, children:[]},
  {id:'D5', name:'تولید و عملیات', head:'حسین موسوی', count:22, children:[]},
];

const EMPS = [
  {id:'EMP-001', name:'سارا محمدی',  nid:'0012345678', dept:'D2', pos:'مدیر منابع انسانی', contract:'تمام‌وقت', status:'active', hire:'1398/04/12', birth:'1368/02/20', edu:'کارشناسی ارشد MBA', phone:'0912-0000001', email:'sara.m@demo.ir', mgr:'مدیرعامل', base:52000000, leaveBal:12, acc:'6037-9911-0001', tel:'021-8811001', marital:'متأهل', addr:'تهران، سعادت‌آباد'},
  {id:'EMP-002', name:'آرش رضایی',   nid:'0023456789', dept:'D1', pos:'مدیر فناوری اطلاعات', contract:'تمام‌وقت', status:'active', hire:'1397/09/01', birth:'1364/07/11', edu:'کارشناسی ارشد نرم‌افزار', phone:'0912-0000002', email:'arash.r@demo.ir', mgr:'مدیرعامل', base:58000000, leaveBal:7, acc:'6037-9911-0002', tel:'021-8811002', marital:'مجرد', addr:'تهران، ونک'},
  {id:'EMP-003', name:'نگار حسینی',  nid:'0034567890', dept:'D4', pos:'کارشناس فروش', contract:'تمام‌وقت', status:'active', hire:'1401/11/20', birth:'1373/03/30', edu:'کارشناسی مدیریت بازرگانی', phone:'0912-0000003', email:'negar.h@demo.ir', mgr:'محمد احمدی', base:34000000, leaveBal:15, acc:'6037-9911-0003', tel:'021-8811003', marital:'مجرد', addr:'تهران، نارمک'},
  {id:'EMP-004', name:'علی مرادی',   nid:'0045678901', dept:'D3', pos:'کارشناس ارشد مالی', contract:'تمام‌وقت', status:'active', hire:'1399/06/15', birth:'1369/11/05', edu:'کارشناسی حسابداری', phone:'0912-0000004', email:'ali.m@demo.ir', mgr:'نرگس کریمی', base:41000000, leaveBal:4, acc:'6037-9911-0004', tel:'021-8811004', marital:'متأهل', addr:'کرج، مهرشهر'},
  {id:'EMP-005', name:'محمد احمدی',  nid:'0056789012', dept:'D4', pos:'مدیر فروش', contract:'تمام‌وقت', status:'active', hire:'1396/02/10', birth:'1362/09/18', edu:'کارشناسی ارشد بازاریابی', phone:'0912-0000005', email:'m.ahmadi@demo.ir', mgr:'مدیرعامل', base:55000000, leaveBal:9, acc:'6037-9911-0005', tel:'021-8811005', marital:'متأهل', addr:'تهران، جردن'},
  {id:'EMP-006', name:'نرگس کریمی',  nid:'0067890123', dept:'D3', pos:'مدیر مالی', contract:'تمام‌وقت', status:'active', hire:'1395/08/03', birth:'1363/05/22', edu:'کارشناسی ارشد حسابداری', phone:'0912-0000006', email:'n.karimi@demo.ir', mgr:'مدیرعامل', base:56000000, leaveBal:10, acc:'6037-9911-0006', tel:'021-8811006', marital:'متأهل', addr:'تهران، یوسف‌آباد'},
  {id:'EMP-007', name:'رضا کاظمی',   nid:'0078901234', dept:'D5', pos:'سرپرست تولید', contract:'تمام‌وقت', status:'active', hire:'1400/03/08', birth:'1367/01/14', edu:'کارشناسی مهندسی صنایع', phone:'0912-0000007', email:'r.kazemi@demo.ir', mgr:'حسین موسوی', base:38000000, leaveBal:2, acc:'6037-9911-0007', tel:'021-8811007', marital:'متأهل', addr:'تهران، اکباتان'},
  {id:'EMP-008', name:'مریم نوری',   nid:'0089012345', dept:'D2', pos:'کارشناس منابع انسانی', contract:'تمام‌وقت', status:'leave', hire:'1402/05/01', birth:'1374/12/02', edu:'کارشناسی روانشناسی صناعتی', phone:'0912-0000008', email:'m.nouri@demo.ir', mgr:'سارا محمدی', base:31000000, leaveBal:6, acc:'6037-9911-0008', tel:'021-8811008', marital:'مجرد', addr:'تهران، پونک'},
  {id:'EMP-009', name:'حسین موسوی',  nid:'0090123456', dept:'D5', pos:'مدیر تولید', contract:'تمام‌وقت', status:'active', hire:'1394/10/25', birth:'1361/06/09', edu:'کارشناسی ارشد صنایع', phone:'0912-0000009', email:'h.mousavi@demo.ir', mgr:'مدیرعامل', base:54000000, leaveBal:11, acc:'6037-9911-0009', tel:'021-8811009', marital:'متأهل', addr:'تهران، شهرک غرب'},
  {id:'EMP-010', name:'فاطمه صادقی', nid:'0101234567', dept:'D1', pos:'برنامه‌نویس', contract:'پیمانکاری', status:'active', hire:'1403/01/15', birth:'1376/08/27', edu:'کارشناسی مهندسی کامپیوتر', phone:'0912-0000010', email:'f.sadeghi@demo.ir', mgr:'آرش رضایی', base:36000000, leaveBal:14, acc:'6037-9911-0010', tel:'021-8811010', marital:'مجرد', addr:'تهران، سعادت‌آباد'},
  {id:'EMP-011', name:'امیر قاسمی',  nid:'0112345678', dept:'D1', pos:'کارشناس شبکه', contract:'تمام‌وقت', status:'probation', hire:'1404/06/01', birth:'1377/04/16', edu:'کارشناسی فناوری اطلاعات', phone:'0912-0000011', email:'a.ghasemi@demo.ir', mgr:'آرش رضایی', base:29000000, leaveBal:18, acc:'6037-9911-0011', tel:'021-8811011', marital:'مجرد', addr:'تهران، تهرانپارس'},
  {id:'EMP-012', name:'الهام رستمی', nid:'0123456789', dept:'D4', pos:'کارشناس بازاریابی', contract:'تمام‌وقت', status:'terminated', hire:'1399/01/20', birth:'1370/10/08', edu:'کارشناسی مدیریت تبلیغات', phone:'0912-0000012', email:'e.rostami@demo.ir', mgr:'محمد احمدی', base:33000000, leaveBal:0, acc:'6037-9911-0012', tel:'021-8811012', marital:'متأهل', addr:'تهران، نازی‌آباد'},
];

const EMPS_EXTRA = {
  D2:['زهرا کریمی','هدی صالحی'], D3:['مهدی توکلی','سمانه غفاری','پویا امینی'],
  D4:['کیانا شجاعی','رضا فرهادی','سعید مدنی'], D5:['جواد رحیمی','اکبر نجفی','مرتضی سلطانی','یاسر عبدی'],
  D6:['شیرین یزدانی','کامران داوری'],
};

let LEAVES = [
  {id:'LV-201', emp:'EMP-003', type:'استحقاقی', from:'1404/07/14', to:'1404/07/16', days:3, status:'pending', reason:'سفر خانوادگی', ts:'1404/07/09'},
  {id:'LV-202', emp:'EMP-010', type:'استحقاقی', from:'1404/07/21', to:'1404/07/22', days:2, status:'pending', reason:'امور شخصی', ts:'1404/07/10'},
  {id:'LV-203', emp:'EMP-004', type:'استحقاقی', from:'1404/07/12', to:'1404/07/12', days:1, status:'pending', reason:'مراجعه به پزشک', ts:'1404/07/08'},
  {id:'LV-204', emp:'EMP-008', type:'استحقاقی', from:'1404/07/05', to:'1404/07/11', days:7, status:'approved', reason:'مرخصی سالانه', ts:'1404/07/01', approver:'سارا محمدی'},
  {id:'LV-205', emp:'EMP-007', type:'مریضی', from:'1404/07/02', to:'1404/07/02', days:1, status:'rejected', reason:'تداخل با شیفت تولید', ts:'1404/06/30', approver:'حسین موسوی'},
  {id:'LV-206', emp:'EMP-004', type:'استحقاقی', from:'1404/06/15', to:'1404/06/15', days:1, status:'approved', reason:'امور اداری', ts:'1404/06/10', approver:'نرگس کریمی'},
  {id:'LV-207', emp:'EMP-011', type:'مریضی', from:'1404/07/01', to:'1404/07/03', days:3, status:'approved', reason:'گواهی پزشک', ts:'1404/06/28', approver:'آرش رضایی'},
];
let leaveSeq = 211;

const PAYRUNS = [
  {id:'PAY-1404-06', month:'خرداد ۱۴۰۴', status:'پردازش‌شده', count:66, gross:2850, tax:312, ins:228, net:2198, ts:'1404/07/01'},
  {id:'PAY-1404-07', month:'تیر ۱۴۰۴', status:'پردازش‌شده', count:66, gross:2890, tax:318, ins:231, net:2223, ts:'1404/07/31'},
  {id:'PAY-1404-08', month:'مرداد ۱۴۰۴', status:'پردازش‌شده', count:66, gross:2890, tax:318, ins:231, net:2223, ts:'1404/08/31'},
  {id:'PAY-1404-09', month:'شهریور ۱۴۰۴', status:'پیش‌نویس', count:66, gross:2910, tax:320, ins:233, net:2237, ts:'—'},
];

const PAYSLIP = {
  emp:'EMP-003', month:'مهر ۱۴۰۴', bank:'بانک ملت', acc:'6037-9911-0003',
  workdays:22, payDate:'۱۵ آبان ۱۴۰۴',
  earnings:[['حقوق مبنا',34000000],['حق مسکن',9000000],['بن کارگری',14000000],['عیدی و سنوات (اقساطی)',5800000],['اضافه‌کاری (۱۸ ساعت)',6120000]],
  deductions:[['مالیات حقوق',4150000],['بیمه سهم کارمند (۷٪)',3528000],['قسط وام مسکن',2500000]],
};

const RECRUIT_REQS = [
  {id:'REQ-21', dept:'D1', pos:'برنامه‌نویس ارشد', count:2, reason:'توسعه تیم محصول', status:'approved', date:'1404/06/28', by:'آرش رضایی'},
  {id:'REQ-22', dept:'D4', pos:'کارشناس فروش', count:1, reason:'افزایش پوشش منطقه ۲', status:'pending', date:'1404/07/05', by:'محمد احمدی'},
  {id:'REQ-23', dept:'D5', pos:'اپراتور خط تولید', count:4, reason:'راه‌اندازی خط جدید', status:'approved', date:'1404/05/12', by:'حسین موسوی'},
  {id:'REQ-24', dept:'D3', pos:'حسابدار', count:1, reason:'جانشین‌پروری', status:'rejected', date:'1404/04/30', by:'نرگس کریمی'},
];

const JOBS = [
  {id:'JOB-31', title:'برنامه‌نویس ارشد فرانت‌اند', dept:'D1', type:'تمام‌وقت', channel:'سایت سازمان، LinkedIn', status:'open', date:'1404/07/01', apps:24},
  {id:'JOB-32', title:'کارشناس فروش', dept:'D4', type:'تمام‌وقت', channel:'سایت سازمان', status:'open', date:'1404/07/06', apps:9},
  {id:'JOB-33', title:'اپراتور تولید', dept:'D5', type:'تمام‌وقت', channel:'جاب‌ویژن، آگهی محلی', status:'open', date:'1404/06/20', apps:41},
  {id:'JOB-34', title:'حسابدار', dept:'D3', type:'تمام‌وقت', channel:'سایت سازمان', status:'paused', date:'1404/05/15', apps:7},
];

const CANDS = [
  {id:'CAN-41', name:'سپهر نیک‌فر', job:'JOB-31', src:'LinkedIn', stage:'فنی', score:85, ts:'1404/07/04'},
  {id:'CAN-42', name:'متین شریفی', job:'JOB-31', src:'معرفی همکار', stage:'منابع انسانی', score:72, ts:'1404/07/03'},
  {id:'CAN-43', name:'درسا اکبری', job:'JOB-32', src:'سایت سازمان', stage:'فنی', score:64, ts:'1404/07/08'},
  {id:'CAN-44', name:'سینا پارسا', job:'JOB-31', src:'جاب‌ویژن', stage:'منابع انسانی', score:58, ts:'1404/07/02'},
  {id:'CAN-45', name:'نیلوفر بهرامی', job:'JOB-31', src:'LinkedIn', stage:'پیشنهاد شغلی', score:91, ts:'1404/06/28'},
  {id:'CAN-46', name:'کیان توکلی', job:'JOB-31', src:'سایت سازمان', stage:'حذف‌شده', score:35, ts:'1404/07/01'},
  {id:'CAN-47', name:'آوا مقدم', job:'JOB-33', src:'آگهی محلی', stage:'منابع انسانی', score:70, ts:'1404/07/07'},
  {id:'CAN-48', name:'پارسا نادری', job:'JOB-33', src:'جاب‌ویژن', stage:'فنی', score:75, ts:'1404/07/09'},
];
const STAGES = ['منابع انسانی','فنی','مدیریت','پیشنهاد شغلی','حذف‌شده'];

const PERF_CYCLES = [
  {id:'CYC-51', name:'نیم‌سال اول ۱۴۰۴', period:'۱۴۰۴/۰۱ تا ۱۴۰۴/۰۶', status:'پایان‌یافته', people:66, ts:'1404/07/02'},
  {id:'CYC-52', name:'نیم‌سال دوم ۱۴۰۴', period:'۱۴۰۴/۰۷ تا ۱۴۰۴/۱۲', status:'در حال اجرا', people:66, ts:'1404/07/10'},
];
const PERF_ROWS = [
  {emp:'EMP-003', cycle:'نیم‌سال اول ۱۴۰۴', score:3.8, status:'نهایی'},
  {emp:'EMP-004', cycle:'نیم‌سال اول ۱۴۰۴', score:4.3, status:'نهایی'},
  {emp:'EMP-007', cycle:'نیم‌سال اول ۱۴۰۴', score:4.0, status:'نهایی'},
  {emp:'EMP-010', cycle:'نیم‌سال اول ۱۴۰۴', score:3.1, status:'نهایی'},
  {emp:'EMP-011', cycle:'نیم‌سال دوم ۱۴۰۴', score:0, status:'در حال ارزیابی'},
];
const PERF_TPL = [
  {name:'تعهد به نتایج', cat:'رفتاری', w:25},
  {name:'کار تیمی و همکاری', cat:'رفتاری', w:20},
  {name:'کیفیت خروجی کار', cat:'فنی', w:30},
  {name:'رشد فردی و یادگیری', cat:'رفتاری', w:15},
  {name:'پیروی از ارزش‌های سازمان', cat:'ارزشی', w:10},
];

const TRAININGS = [
  {id:'TRN-81', title:'اصول حمل بار برای اپراتورها', cat:'ایمنی', cap:15, enrolled:12, dates:'۲۶–۲۷ مهر ۱۴۰۴', mode:'حضوری'},
  {id:'TRN-82', title:'پایتون برای تحلیل داده', cat:'فنی', cap:12, enrolled:9, dates:'آبان ۱۴۰۴', mode:'آنلاین'},
  {id:'TRN-83', title:'مبانی ایمنی صنعتی HSE', cat:'ایمنی', cap:20, enrolled:15, dates:'آبان ۱۴۰۴', mode:'حضوری'},
  {id:'TRN-84', title:'اصول مذاکره فروش', cat:'مهارت نرم', cap:10, enrolled:2, dates:'آذر ۱۴۰۴', mode:'حضوری'},
  {id:'TRN-85', title:'انگلیسی تجاری', cat:'زبان', cap:14, enrolled:8, dates:'آذر ۱۴۰۴', mode:'آنلاین'},
];
const TRAIN_HIST = [
  {emp:'EMP-003', course:'اصول مذاکره فروش', date:'1403/11/20', hours:16, result:'قبولی', cert:'CERT-1024'},
  {emp:'EMP-003', course:'مبانی ایمنی صنعتی HSE', date:'1403/08/10', hours:8, result:'قبولی', cert:'CERT-0992'},
  {emp:'EMP-003', course:'اکسل پیشرفته', date:'1402/12/15', hours:24, result:'قبولی', cert:'CERT-0771'},
];
const SKILL_GAP = [
  {skill:'تحلیل داده با پایتون', need:'پیشرفته', have:'مبتدی', gap:'زیاد', pri:'فوری'},
  {skill:'اصول مذاکره فروش', need:'پیشرفته', have:'متوسط', gap:'متوسط', pri:'فوری'},
  {skill:'انگلیسی تجاری', need:'متوسط', have:'مبتدی', gap:'متوسط', pri:'معمولی'},
];

const WELFARE = [
  {id:'WLF-11', emp:'EMP-007', type:'وام ضروری', amount:'۲۰۰ میلیون ریال', status:'approved', ts:'1404/06/18', desc:'بازپرداخت ۱۸ ماهه'},
  {id:'WLF-12', emp:'EMP-003', type:'بیمه تکمیلی', amount:'—', status:'pending', ts:'1404/07/09', desc:'بیمه همسر و فرزند'},
  {id:'WLF-13', emp:'EMP-010', type:'وام مسکن', amount:'۱۵۰ میلیون ریال', status:'pending', ts:'1404/07/10', desc:'وام ساخت مسکن'},
];

const USERS = [
  {id:'USR-01', username:'admin', name:'مدیر سیستم', role:'admin', status:'active', last:'1404/07/12 08:01'},
  {id:'USR-02', username:'hr.manager', name:'سارا محمدی', role:'hr_manager', status:'active', last:'1404/07/12 07:45'},
  {id:'USR-03', username:'hr.spec', name:'مریم نوری', role:'hr_spec', status:'active', last:'1404/07/11 14:40'},
  {id:'USR-04', username:'line.mgr', name:'محمد احمدی', role:'line_manager', status:'active', last:'1404/07/12 08:10'},
  {id:'USR-05', username:'employee', name:'نگار حسینی', role:'employee', status:'active', last:'1404/07/12 08:30'},
  {id:'USR-06', username:'r.kazemi', name:'رضا کاظمی', role:'employee', status:'locked', last:'1404/07/05 18:00'},
];

const ROLES = [
  {id:'admin', name:'مدیر سیستم', users:1, desc:'مدیریت کاربران، نقش‌ها، تنظیمات و لاگ حسابرسی'},
  {id:'hr_manager', name:'مدیر منابع انسانی', users:1, desc:'دسترسی کامل به تمام ماژول‌ها، تأیید نهایی، گزارش‌های کلان'},
  {id:'hr_spec', name:'کارشناس منابع انسانی', users:1, desc:'مدیریت کارکنان، استخدام، آموزش، ورود داده'},
  {id:'line_manager', name:'مدیر واحد', users:1, desc:'تأیید مرخصی، ارزیابی عملکرد، مشاهده کارکنان واحد خود'},
  {id:'employee', name:'کارمند', users:2, desc:'اطلاعات شخصی، درخواست مرخصی، فیش حقوقی، درخواست‌های رفاهی'},
];

const SYS_PARAMS = [
  {id:'P1', name:'سقف مرخصی استحقاقی سالانه', value:'۲۶ روز', group:'مرخصی'},
  {id:'P2', name:'نرخ اضافه‌کاری (هر ساعت)', value:'۱۴۰٪ نرخ ساعتی', group:'حقوق و دستمزد'},
  {id:'P3', name:'حد نصاب کارکرد ماه', value:'۱۷۶ ساعت', group:'حقوق و دستمزد'},
  {id:'P4', name:'سطوح تأیید مرخصی (بیش از ۳ روز)', value:'مدیر واحد + منابع انسانی', group:'گردش کار'},
  {id:'P5', name:'سقف وام ضروری', value:'۳ برابر حقوق پایه', group:'رفاهی'},
  {id:'P6', name:'قفل حساب پس از', value:'۵ تلاش ناموفق', group:'امنیت'},
];

const AUDIT = [
  {id:'LOG-01', user:'admin', action:'ورود موفق به سامانه', ts:'1404/07/12 08:01', ip:'10.0.0.11'},
  {id:'LOG-02', user:'hr.manager', action:'ویرایش حقوق پایه — EMP-011', ts:'1404/07/12 09:15', ip:'10.0.0.23'},
  {id:'LOG-03', user:'line.mgr', action:'تأیید مرخصی LV-204', ts:'1404/07/12 10:02', ip:'10.0.0.34'},
  {id:'LOG-04', user:'hr.spec', action:'ایجاد کارمند EMP-012', ts:'1404/07/11 14:40', ip:'10.0.0.42'},
  {id:'LOG-05', user:'admin', action:'تغییر نقش کاربر r.kazemi', ts:'1404/07/11 16:22', ip:'10.0.0.11'},
  {id:'LOG-06', user:'employee', action:'ثبت درخواست مرخصی LV-201', ts:'1404/07/11 11:30', ip:'10.0.0.77'},
];

const EMP_DOCS = [
  {type:'قرارداد کاری', name:'قرارداد_1404.pdf', size:'1.2MB', ts:'1404/01/10'},
  {type:'کارت ملی', name:'cart_melli.jpg', size:'0.4MB', ts:'1398/04/12'},
  {type:'مدرک تحصیلی', name:'modarres.pdf', size:'2.1MB', ts:'1398/04/15'},
];

const NOTIFS = [
  {ico:'🌤️', title:'درخواست مرخصی نگار حسینی در انتظار تأیید شماست', time:'۱۰ دقیقه پیش', unread:true, page:'leaveApprove'},
  {ico:'💰', title:'فیش حقوقی مهر ۱۴۰۴ صادر شد', time:'۲ ساعت پیش', unread:true, page:'payslip'},
  {ico:'🎓', title:'ثبت‌نام دوره «مبانی ایمنی صنعتی HSE» آغاز شد', time:'دیروز', unread:true, page:'training'},
  {ico:'📈', title:'دوره ارزیابی «نیم‌سال دوم ۱۴۰۴» آغاز شد', time:'۳ روز پیش', unread:false, page:'perfCycles'},
  {ico:'👥', title:'۲ متقاضی جدید برای آگهی «برنامه‌نویس ارشد» ثبت‌نام کردند', time:'۳ روز پیش', unread:false, page:'candidates'},
];

const EMPLOYEES = EMPS;
const CANDIDATES = CANDS;
