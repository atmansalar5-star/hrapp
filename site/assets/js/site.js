/* =====================================================================
   رفتار مشترک وب‌سایت معرفی سامانهٔ منابع انسانی
   ---------------------------------------------------------------------
   • منوی موبایل
   • نمایان‌شدن نرم بخش‌ها هنگام اسکرول
   • شبکهٔ تعاملی «نقش‌ها ↔ ماژول‌ها» در قهرمان صفحهٔ خانه
   • کمکی‌های نمایشی (ارقام فارسی، جداکنندهٔ هزارگان، برچسب ماژول)
   هیچ وابستگی بیرونی و هیچ درخواست شبکه‌ای در این فایل نیست.
   ===================================================================== */
(function () {
  'use strict';

  /* ---------- کمکی‌های نمایشی ---------- */
  var FA_DIGITS = ['۰','۱','۲','۳','۴','۵','۶','۷','۸','۹'];

  /** ارقام لاتین را به ارقام فارسی برمی‌گرداند. */
  function faNum(value) {
    return String(value).replace(/[0-9]/g, function (d) { return FA_DIGITS[+d]; });
  }

  /** عدد را با جداکنندهٔ هزارگان و ارقام فارسی برمی‌گرداند. */
  function faNumGroup(value) {
    var n = Number(value);
    if (!isFinite(n)) return faNum(value);
    return faNum(n.toLocaleString('en-US'));
  }

  window.HRApp = { faNum: faNum, faNumGroup: faNumGroup };

  /* اعداد عناصر نشان‌دار را فارسی کن */
  function localizeNumbers(root) {
    var nodes = (root || document).querySelectorAll('[data-fa-num]');
    Array.prototype.forEach.call(nodes, function (el) {
      var raw = el.getAttribute('data-fa-num');
      if (raw === null || raw === '') return;
      el.textContent = el.hasAttribute('data-fa-group') ? faNumGroup(raw) : faNum(raw);
    });
  }

  /* ---------- منوی موبایل ---------- */
  function initNav() {
    var toggle = document.querySelector('.nav-toggle');
    var nav = document.getElementById('site-nav');
    if (!toggle || !nav) return;
    toggle.addEventListener('click', function () {
      var open = nav.classList.toggle('open');
      toggle.setAttribute('aria-expanded', open ? 'true' : 'false');
      toggle.textContent = open ? '✕' : '☰';
    });
    nav.addEventListener('click', function (e) {
      if (e.target.tagName === 'A' && nav.classList.contains('open')) {
        nav.classList.remove('open');
        toggle.setAttribute('aria-expanded', 'false');
        toggle.textContent = '☰';
      }
    });
  }

  /* ---------- نمایان‌شدن هنگام اسکرول ---------- */
  function initReveal() {
    var items = document.querySelectorAll('.reveal');
    if (!items.length) return;
    if (!('IntersectionObserver' in window)) {
      Array.prototype.forEach.call(items, function (el) { el.classList.add('in'); });
      return;
    }
    var io = new IntersectionObserver(function (entries) {
      entries.forEach(function (entry) {
        if (entry.isIntersecting) {
          entry.target.classList.add('in');
          io.unobserve(entry.target);
        }
      });
    }, { rootMargin: '0px 0px -8% 0px', threshold: 0.08 });
    Array.prototype.forEach.call(items, function (el) { io.observe(el); });
  }

  /* ---------- شبکهٔ تعاملی قهرمان صفحه ----------
     گره‌های درون‌شبکه یک نمودار سازمانی انتزاعی‌اند: چند کارمند/نقش به مرکز
     «هستهٔ قواعد» وصل می‌شوند. هدف، بیان یک پیام است: هر نقش، به‌اندازهٔ
     دسترسی خودش، به همان هستهٔ واحد وصل می‌شود — نه به نسخه‌ای جداگانه از قواعد. */
  function initHeroGraph() {
    var canvas = document.getElementById('heroGraph');
    if (!canvas) return;
    var ctx = canvas.getContext('2d');
    if (!ctx) return;

    var reduce = window.matchMedia && window.matchMedia('(prefers-reduced-motion: reduce)').matches;
    var W = 0, H = 0, dpr = 1;
    var t = 0;
    var roles = [
      { label: 'مدیر منابع انسانی', r: 0.34, color: '#5EEAD4' },
      { label: 'کارشناس منابع انسانی', r: 0.60, color: '#7DD3FC' },
      { label: 'مدیر واحد', r: 0.86, color: '#C4B5FD' },
      { label: 'کارمند', r: 1.12, color: '#FCD34D' },
      { label: 'مدیر سیستم', r: 1.38, color: '#F9A8D4' }
    ];

    function resize() {
      dpr = Math.min(window.devicePixelRatio || 1, 2);
      var rect = canvas.getBoundingClientRect();
      W = Math.max(rect.width, 240);
      H = Math.max(rect.height, 240);
      canvas.width = Math.round(W * dpr);
      canvas.height = Math.round(H * dpr);
      ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
    }

    function cx() { return W / 2; }
    function cy() { return H / 2; }
    function unit() { return Math.min(W, H) / 2; }

    function draw() {
      var cxv = cx(), cyv = cy(), u = unit();
      ctx.clearRect(0, 0, W, H);

      /* حلقه‌های مدار: هر حلقه = یک لایهٔ دسترسی */
      ctx.lineWidth = 1;
      for (var i = 1; i <= 5; i++) {
        ctx.beginPath();
        ctx.strokeStyle = 'rgba(255,255,255,' + (0.10 - i * 0.012) + ')';
        ctx.arc(cxv, cyv, u * 0.16 * i, 0, Math.PI * 2);
        ctx.stroke();
      }

      /* اتصال هر نقش به مرکز */
      roles.forEach(function (role, idx) {
        var angle = (idx / roles.length) * Math.PI * 2 + t * 0.00016;
        var rad = u * role.r * 0.62 + u * 0.16;
        var x = cxv + Math.cos(angle) * rad;
        var y = cyv + Math.sin(angle) * rad;

        ctx.beginPath();
        ctx.strokeStyle = role.color + '55';
        ctx.lineWidth = 1.4;
        ctx.moveTo(cxv, cyv);
        ctx.lineTo(x, y);
        ctx.stroke();

        /* پالس درخواست روی خط */
        var pulse = (t * 0.00045 + idx * 0.2) % 1;
        var px = cxv + (x - cxv) * pulse;
        var py = cyv + (y - cyv) * pulse;
        ctx.beginPath();
        ctx.fillStyle = role.color;
        ctx.arc(px, py, 2.4, 0, Math.PI * 2);
        ctx.fill();

        /* گرهٔ نقش */
        ctx.beginPath();
        ctx.fillStyle = 'rgba(255,255,255,0.10)';
        ctx.arc(x, y, 13, 0, Math.PI * 2);
        ctx.fill();
        ctx.beginPath();
        ctx.fillStyle = role.color;
        ctx.arc(x, y, 5.4, 0, Math.PI * 2);
        ctx.fill();

        /* برچسب نقش */
        ctx.font = '600 11px Vazirmatn, Tahoma, sans-serif';
        ctx.fillStyle = 'rgba(233,246,245,0.92)';
        ctx.textAlign = 'center';
        ctx.textBaseline = 'middle';
        ctx.fillText(role.label, x, y + 26);
      });

      /* هستهٔ مرکزی */
      ctx.beginPath();
      ctx.fillStyle = 'rgba(45,212,191,0.22)';
      ctx.arc(cxv, cyv, 30 + Math.sin(t * 0.0009) * 2.5, 0, Math.PI * 2);
      ctx.fill();
      ctx.beginPath();
      ctx.fillStyle = 'rgba(255,255,255,0.95)';
      ctx.arc(cxv, cyv, 17, 0, Math.PI * 2);
      ctx.fill();
      ctx.font = '800 10px Vazirmatn, Tahoma, sans-serif';
      ctx.fillStyle = '#0B2A33';
      ctx.textAlign = 'center';
      ctx.textBaseline = 'middle';
      ctx.fillText('هستهٔ قواعد', cxv, cyv);
    }

    function loop() {
      t += 16;
      draw();
      window.requestAnimationFrame(loop);
    }

    resize();
    draw();
    window.addEventListener('resize', function () { resize(); draw(); }, { passive: true });
    if (!reduce) window.requestAnimationFrame(loop);
  }

  /* ---------- تزئین قهرمان: نورافکن نشانگر و نشانگر اسکرول ---------- */
  function initHeroDeco() {
    var hero = document.querySelector('.hero');
    if (!hero) return;

    /* لایهٔ نورافکن: مختصات نشانگر را در دو متغیر CSS می‌ریزد. */
    var spot = document.createElement('div');
    spot.className = 'hero__spot';
    spot.setAttribute('aria-hidden', 'true');
    hero.appendChild(spot);

    hero.addEventListener('pointermove', function (e) {
      var box = hero.getBoundingClientRect();
      if (!box.width || !box.height) return;
      hero.style.setProperty('--mx', ((e.clientX - box.left) / box.width * 100).toFixed(1) + '%');
      hero.style.setProperty('--my', ((e.clientY - box.top) / box.height * 100).toFixed(1) + '%');
    }, { passive: true });

    /* نشانگر اسکرول: در موبایل با CSS پنهان می‌شود. */
    var cue = document.createElement('div');
    cue.className = 'hero__cue';
    cue.setAttribute('aria-hidden', 'true');
    cue.innerHTML = '<span>اسکرول کنید</span><i></i>';
    hero.appendChild(cue);
  }

  /* ---------- نشانگر پیشرفت، سرصفحهٔ شیشه‌ای و بازگشت به بالا ---------- */
  function initScrollUI() {
    var root = document.documentElement;
    var header = document.querySelector('.site-header');

    var bar = document.createElement('div');
    bar.className = 'scroll-progress';
    bar.setAttribute('aria-hidden', 'true');
    bar.appendChild(document.createElement('i'));
    document.body.appendChild(bar);

    var toTop = document.createElement('button');
    toTop.type = 'button';
    toTop.className = 'to-top';
    toTop.setAttribute('aria-label', 'بازگشت به بالای صفحه');
    toTop.textContent = '↑';
    toTop.addEventListener('click', function () {
      window.scrollTo({ top: 0, behavior: prefersReduced() ? 'auto' : 'smooth' });
    });
    document.body.appendChild(toTop);

    var ticking = false;
    function update() {
      ticking = false;
      var span = root.scrollHeight - window.innerHeight;
      var p = span > 0 ? Math.min(1, Math.max(0, window.scrollY / span)) : 0;
      root.style.setProperty('--p', p.toFixed(4));
      if (header) header.classList.toggle('is-scrolled', window.scrollY > 6);
      toTop.classList.toggle('show', window.scrollY > 760);
    }
    function schedule() {
      if (ticking) return;
      ticking = true;
      window.requestAnimationFrame(update);
    }
    window.addEventListener('scroll', schedule, { passive: true });
    window.addEventListener('resize', schedule, { passive: true });
    update();
  }

  /* ---------- شمارش صعودی اعداد آماری ----------
     عدد نهایی همیشه همان چیزی است که در data-fa-num آمده؛ انیمیشن فقط
     نمایشی است و در پایان، مقدار دقیق دوباره نوشته می‌شود. */
  function initCounters() {
    var nodes = document.querySelectorAll('[data-fa-num]');
    if (!nodes.length || prefersReduced() || !('IntersectionObserver' in window)) return;

    var targets = [];
    Array.prototype.forEach.call(nodes, function (el) {
      var raw = el.getAttribute('data-fa-num');
      if (!raw || !/^[0-9]+$/.test(raw)) return;
      var n = parseInt(raw, 10);
      if (n < 8) return;                    /* اعداد کوچک ارزش شمردن ندارند */
      targets.push({ el: el, n: n, group: el.hasAttribute('data-fa-group') });
    });
    if (!targets.length) return;

    function run(item) {
      var started = 0;
      var dur = 900;
      function frame(now) {
        if (!started) started = now;
        var k = Math.min(1, (now - started) / dur);
        var eased = 1 - Math.pow(1 - k, 3);
        var v = Math.round(item.n * eased);
        item.el.textContent = item.group ? faNumGroup(v) : faNum(v);
        if (k < 1) window.requestAnimationFrame(frame);
        else item.el.textContent = item.group ? faNumGroup(item.n) : faNum(item.n);
      }
      window.requestAnimationFrame(frame);
    }

    var io = new IntersectionObserver(function (entries) {
      entries.forEach(function (entry) {
        if (!entry.isIntersecting) return;
        io.unobserve(entry.target);
        for (var i = 0; i < targets.length; i++) {
          if (targets[i].el === entry.target) { run(targets[i]); return; }
        }
      });
    }, { threshold: 0.7 });

    targets.forEach(function (item) { io.observe(item.el); });
  }

  function prefersReduced() {
    return !!(window.matchMedia && window.matchMedia('(prefers-reduced-motion: reduce)').matches);
  }

  /* ---------- آماده‌سازی ---------- */
  function boot() {
    localizeNumbers(document);
    initNav();
    initReveal();
    initHeroGraph();
    initHeroDeco();
    initScrollUI();
    initCounters();
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', boot);
  } else {
    boot();
  }
})();
