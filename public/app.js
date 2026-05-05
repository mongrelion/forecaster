// Sites direction ranges kept as reference only — actual site data comes from backend

// ── Compass utilities ──────────────────────────────────────────────────────────

const COMPASS_POINTS = ['N','NNE','NE','ENE','E','ESE','SE','SSE','S','SSW','SW','WSW','W','WNW','NW','NNW'];

// Centre degree and ±11.25° sector boundaries for each 16-point compass direction
const COMPASS = {};
COMPASS_POINTS.forEach((name, i) => {
  const center = i * 22.5;
  COMPASS[name] = { center, min: ((center - 11.25) + 360) % 360, max: (center + 11.25) % 360 };
});

// Convert a named FROM–TO range to a [minDeg, maxDeg] pair.
// isWindInRange() handles wrap-around when minDeg > maxDeg (range crosses 0°/360°).
function compassToRange(from, to) {
  return [COMPASS[from].min, COMPASS[to].max];
}

// Nearest 16-point compass name for a bearing in degrees
function degToCompass(deg) {
  return COMPASS_POINTS[Math.round(((deg % 360) + 360) % 360 / 22.5) % 16];
}

// ── State ──────────────────────────────────────────────────────────────────────

const state = {
  rainThreshold:  30,
  cloudThreshold: 75,
  sortBy:         'flyability', // 'flyability' | 'alpha'
};

// ── Tooltip ────────────────────────────────────────────────────────────────────

const tooltipEl = document.getElementById('tooltip');

function showTooltip(e, html) {
  tooltipEl.innerHTML = html;
  tooltipEl.classList.add('visible');
  tooltipEl.removeAttribute('aria-hidden');
  positionTooltip(e);
}

function positionTooltip(e) {
  const margin = 14;
  tooltipEl.style.left = '0px';
  tooltipEl.style.top  = '0px';
  const { width, height } = tooltipEl.getBoundingClientRect();
  const vw = window.innerWidth;
  const vh = window.innerHeight;

  let left = e.clientX + margin;
  let top  = e.clientY - height - margin;

  if (left + width  > vw - margin) left = e.clientX - width  - margin;
  if (top           < margin)      top  = e.clientY + margin;
  if (top + height  > vh - margin) top  = vh - height - margin;

  tooltipEl.style.left = `${left}px`;
  tooltipEl.style.top  = `${top}px`;
}

function hideTooltip() {
  tooltipEl.classList.remove('visible');
  tooltipEl.setAttribute('aria-hidden', 'true');
}

// ── Data processing ────────────────────────────────────────────────────────────

function isWindInRange(dir, [min, max]) {
  if (min <= max) return dir >= min && dir <= max;
  return dir >= min || dir <= max; // wraps around 0°/360°
}

function blockColor(hour) {
  if (hour.isDay === 0) return '#0c1526';                      // night — nearly invisible
  if (hour.flyable)     return '#4ade80';                     // all conditions met
  if (hour.marginal)    return 'rgba(251,191,36,0.65)';       // close but not quite
  return '#1c2b42';                                           // daytime, not flyable
}

/**
 * Compute flyability flags for each hour in the backend's response.
 * direction comes from the backend (not SITES reference array).
 * gusts check uses the server-provided max gusts threshold.
 */
function processResponse(siteData) {
  const maxGusts = window._maxGusts || 25;
  return siteData.hours.map(h => {
    const dirOk   = isWindInRange(h.wind_dir, compassToRange(...siteData.direction));
    const gustsOk = h.gusts     <= maxGusts;
    const cloudOk = h.cloud     <= state.cloudThreshold;
    const rainOk  = h.rain      <= state.rainThreshold;

    const flyable = h.is_day === 1 && dirOk && gustsOk && cloudOk && rainOk;

    // Marginal: safe to fly (dir + gusts OK) but exactly one weather condition failing
    const weatherFails = [cloudOk, rainOk].filter(v => !v).length;
    const marginal = !flyable && h.is_day === 1 && dirOk && gustsOk && weatherFails === 1;

    return {
      time:    h.time,
      isDay:   h.is_day,
      windDir: h.wind_dir,
      gusts:   h.gusts,
      cloud:   h.cloud,
      rain:    h.rain,
      windSpeed: h.wind_speed,
      temp:    h.temp,
      dirOk, gustsOk, cloudOk, rainOk, flyable, marginal,
    };
  });
}

function groupByDay(hours) {
  const map = new Map();
  for (const h of hours) {
    const day = h.time.slice(0, 10);
    if (!map.has(day)) map.set(day, []);
    map.get(day).push(h);
  }
  return Array.from(map.entries()).map(([date, dayHours]) => ({ date, dayHours }));
}

function findWindows(hours) {
  const windows = [];
  let current   = null;

  for (const hour of hours) {
    if (hour.flyable) {
      if (!current) current = [];
      current.push(hour);
    } else {
      if (current) { windows.push(current); current = null; }
    }
  }
  if (current) windows.push(current);

  return windows.map(group => {
    const avg = arr => arr.reduce((a, b) => a + b, 0) / arr.length;
    return {
      startTime: group[0].time,
      endTime:   group[group.length - 1].time,
      count:     group.length,
      avgDir:    Math.round(avg(group.map(h => h.windDir))),
      avgWind:   Math.round(avg(group.map(h => h.windSpeed))),
      maxGusts:  Math.round(Math.max(...group.map(h => h.gusts))),
      avgCloud:  Math.round(avg(group.map(h => h.cloud))),
      avgRain:   Math.round(avg(group.map(h => h.rain))),
    };
  });
}

function sortResults(results) {
  if (state.sortBy === 'alpha') {
    return [...results].sort((a, b) => a.site.name.localeCompare(b.site.name));
  }
  // Flyability: most windows first; no-window sites after; errors last
  return [...results].sort((a, b) => {
    const aW = a.error ? -1 : a.windows.length;
    const bW = b.error ? -1 : b.windows.length;
    return bW - aW;
  });
}

function findBestBet(results) {
  const candidates = results.filter(r => !r.error && r.windows.length > 0);
  if (candidates.length === 0) return null;

  candidates.sort((a, b) => {
    const aHours = a.hours.filter(h => h.flyable).length;
    const bHours = b.hours.filter(h => h.flyable).length;
    if (bHours !== aHours) return bHours - aHours;
    const aMax = Math.max(...a.windows.map(w => w.count));
    const bMax = Math.max(...b.windows.map(w => w.count));
    return bMax - aMax;
  });

  const best       = candidates[0];
  const bestWindow = best.windows.reduce((a, b) => a.count >= b.count ? a : b);
  return { site: best.site, window: bestWindow };
}

// ── Formatting helpers ──────────────────────────────────────────────────────────

const DAYS_SHORT   = ['Sun','Mon','Tue','Wed','Thu','Fri','Sat'];
const MONTHS_SHORT  = ['Jan','Feb','Mar','Apr','May','Jun','Jul','Aug','Sep','Oct','Nov','Dec'];

function parseDateStr(isoStr) {
  const [y, m, d] = isoStr.slice(0, 10).split('-').map(Number);
  return { y, m, d };
}

// "Wed 23"
function formatDayLabel(dateStr) {
  const { y, m, d } = parseDateStr(dateStr);
  const date = new Date(y, m - 1, d);
  return `${DAYS_SHORT[date.getDay()]} ${d}`;
}

// "Wed 23 Apr"
function formatDate(isoStr) {
  const { y, m, d } = parseDateStr(isoStr);
  const date = new Date(y, m - 1, d);
  return `${DAYS_SHORT[date.getDay()]} ${d} ${MONTHS_SHORT[m - 1]}`;
}

// "14:00"
function formatTime(isoStr) {
  return isoStr.slice(11, 16);
}

function endTime(lastHourIso) {
  const hour = parseInt(lastHourIso.slice(11, 13), 10);
  return `${String((hour + 1) % 24).padStart(2, '0')}:00`;
}

// "2026-04-22" for today in local time
function todayStr() {
  const n = new Date();
  return `${n.getFullYear()}-${String(n.getMonth() + 1).padStart(2, '0')}-${String(n.getDate()).padStart(2, '0')}`;
}

// ── SVG helpers ────────────────────────────────────────────────────────────────

function windArrowSvg(degrees, cls = 'wind-arrow') {
  return `<svg class="${cls}" viewBox="0 0 14 18" fill="none" xmlns="http://www.w3.org/2000/svg"
    style="transform:rotate(${(degrees + 180) % 360}deg)" aria-hidden="true">
    <line x1="7" y1="16" x2="7" y2="5"  stroke="currentColor" stroke-width="1.8" stroke-linecap="round"/>
    <path d="M3 9 L7 3 L11 9"           stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" fill="none"/>
  </svg>`;
}

// ── Tooltip content ────────────────────────────────────────────────────────────

function buildTooltipHTML(hour) {
  const dateLabel = formatDate(hour.time);
  const timeLabel = formatTime(hour.time);
  const maxGusts = window._maxGusts || 25;

  if (hour.isDay === 0) {
    return `<div class="tt-header">${dateLabel} · ${timeLabel}</div>
            <div class="tt-night">Night</div>`;
  }

  const row = (ok, label, value) =>
    `<div class="tt-row ${ok ? 'ok' : 'fail'}">
       <span class="tt-key">${label}</span>
       <span class="tt-val">${value}</span>
     </div>`;

  return `
    <div class="tt-header">${dateLabel} · ${timeLabel}</div>
    ${row(hour.dirOk,                 'Direction', degToCompass(hour.windDir))}
    ${row(hour.windSpeed <= maxGusts,       'Wind',      `${Math.round(hour.windSpeed)} km/h`)}
    ${row(hour.gustsOk,              'Gusts',     `${Math.round(hour.gusts)} km/h`)}
    ${row(hour.cloudOk,              'Cloud',     `${hour.cloud}%`)}
    ${row(hour.rainOk,               'Rain',      `${hour.rain}%`)}
    <div class="tt-row neutral">
      <span class="tt-key">Temp</span>
      <span class="tt-val">${hour.temp.toFixed(1)}°C</span>
    </div>
  `;
}

// ── Rendering ──────────────────────────────────────────────────────────────────

function createHourBlock(hour) {
  const block = document.createElement('div');
  block.className = 'hour-block';
  block.style.backgroundColor = blockColor(hour);

  block.addEventListener('mouseenter', e => showTooltip(e, buildTooltipHTML(hour)));
  block.addEventListener('mousemove',  positionTooltip);
  block.addEventListener('mouseleave', hideTooltip);

  return block;
}

function renderWindowRow(w) {
  const div = document.createElement('div');
  div.className = 'window-row';

  const today     = todayStr();
  const dateLabel = w.startTime.slice(0, 10) === today ? 'Today' : formatDate(w.startTime);

  div.innerHTML = `
    <div class="win-time">
      <span class="win-date">${dateLabel}</span>
      <span class="win-range">${formatTime(w.startTime)}–${endTime(w.endTime)}</span>
      <span class="win-dur">${w.count}h</span>
    </div>
    <div class="win-stats">
      <span class="win-stat">
        ${windArrowSvg(w.avgDir)}
        <span class="data-val">${degToCompass(w.avgDir)}</span>
      </span>
      <span class="win-stat">
        <span class="data-val">${w.avgWind}</span>
        <span class="stat-unit">/ ${w.maxGusts} km/h</span>
      </span>
      <span class="win-stat">
        <span class="data-val">${w.avgCloud}</span><span class="stat-unit">% ☁</span>
      </span>
      <span class="win-stat">
        <span class="data-val">${w.avgRain}</span><span class="stat-unit">% 🌧</span>
      </span>
    </div>
  `;
  return div;
}

function renderCard(result) {
  const { site, hours, windows, error } = result;
  const card = document.createElement('div');

  if (error) {
    card.className = 'site-card error-card';
    card.innerHTML = `
      <div class="card-header">
        <div class="card-title"><span class="site-name">${site.name}</span></div>
      </div>
      <div class="error-msg">Could not load forecast: ${error}</div>`;
    return card;
  }

  const hasFlyable = windows.length > 0;
  card.className = `site-card${hasFlyable ? ' has-windows' : ''}`;

  // Header
  const header = document.createElement('div');
  header.className = 'card-header';
  header.innerHTML = `
    <div class="card-title">
      <span class="site-name">${site.name}</span>
      <span class="dir-badge">${site.direction[0]}–${site.direction[1]}</span>
    </div>
    <div class="card-status ${hasFlyable ? 'status-good' : 'status-none'}">
      ${hasFlyable ? `${windows.length} window${windows.length > 1 ? 's' : ''}` : 'no windows'}
    </div>`;
  card.appendChild(header);

  // Hour strip
  const stripArea = document.createElement('div');
  stripArea.className = 'strip-area';

  for (const { date, dayHours } of groupByDay(hours)) {
    const row = document.createElement('div');
    row.className = 'day-row';

    const label = document.createElement('span');
    label.className = 'strip-day-label';
    label.textContent = formatDayLabel(date);
    row.appendChild(label);

    const blocks = document.createElement('div');
    blocks.className = 'blocks';
    for (const hour of dayHours) blocks.appendChild(createHourBlock(hour));
    row.appendChild(blocks);

    stripArea.appendChild(row);
  }

  const stripFooter = document.createElement('div');
  stripFooter.className = 'strip-footer';
  stripFooter.innerHTML = `
    <span></span>
    <div class="time-labels">
      <span>00</span><span>06</span><span>12</span><span>18</span><span>24</span>
    </div>`;
  stripArea.appendChild(stripFooter);
  card.appendChild(stripArea);

  // Windows
  const windowsArea = document.createElement('div');
  windowsArea.className = 'windows-area';

  if (hasFlyable) {
    for (const w of windows) windowsArea.appendChild(renderWindowRow(w));
  } else {
    const noWin = document.createElement('div');
    noWin.className = 'no-windows';
    noWin.textContent = 'No flyable conditions in this period';
    windowsArea.appendChild(noWin);
  }
  card.appendChild(windowsArea);

  return card;
}

function renderBestBet(bestBet) {
  const container = document.getElementById('best-bet-container');
  container.innerHTML = '';
  if (!bestBet) return;

  const { site, window: w } = bestBet;
  const today     = todayStr();
  const dateLabel = w.startTime.slice(0, 10) === today ? 'Today' : formatDate(w.startTime);

  const banner = document.createElement('div');
  banner.className = 'best-bet';
  banner.innerHTML = `
    <div class="best-bet-inner">
      <span class="best-icon">★</span>
      <span class="best-label">Best conditions</span>
      <span class="best-site">${site.name}</span>
      <span class="best-sep">·</span>
      <span class="best-time">${dateLabel} · ${formatTime(w.startTime)}–${endTime(w.endTime)} (${w.count}h)</span>
    </div>`;
  container.appendChild(banner);
}

function updateFooterModel() {
  const el = document.getElementById('model-name');
  if (el && window._modelName) {
    el.textContent = window._modelName;
  }
}

function renderAll(results) {
  const grid = document.getElementById('site-grid');
  grid.innerHTML = '';

  renderBestBet(findBestBet(results));

  for (const result of sortResults(results)) grid.appendChild(renderCard(result));

  // Update timestamp from backend
  const now = new Date();
  document.getElementById('updated-label').textContent =
    `Updated ${String(now.getHours()).padStart(2, '0')}:${String(now.getMinutes()).padStart(2, '0')}`;

  document.getElementById('refresh-btn').classList.remove('loading');
}

function showLoadingState() {
  document.getElementById('best-bet-container').innerHTML = '';
  document.getElementById('site-grid').innerHTML =
    `<div class="loading-state"><div class="spinner"></div><span>Fetching forecasts…</span></div>`;
  document.getElementById('refresh-btn').classList.add('loading');
}

// ── Data pipeline ──────────────────────────────────────────────────────────────

/**
 * Process backend sites into { site, hours, windows, error } results,
 * then render. Re-runs without network call when thresholds change.
 */
function processAndRender(backendSites) {
  window._backendSites = backendSites; // persist for threshold re-evaluation
  const results = backendSites.map(siteData => {
    if (siteData.error) {
      const site = { name: siteData.name, direction: siteData.direction };
      return { site, hours: [], windows: [], error: siteData.error };
    }
    const hours   = processResponse(siteData);
    const windows = findWindows(hours);
    const site    = { name: siteData.name, direction: siteData.direction };
    return { site, hours, windows, error: null };
  });
  renderAll(results);
}

async function loadData() {
  showLoadingState();
  try {
    const resp = await fetch('/api/forecast');
    if (!resp.ok) throw new Error(`HTTP ${resp.status}`);
    const data = await resp.json();
    window._modelName = data.model || null;
    window._maxGusts = data.max_gusts || 25;
    updateFooterModel();
    processAndRender(data.sites);
  } catch (err) {
    document.getElementById('site-grid').innerHTML =
      `<div class="error-card" style="max-width:600px;margin:2rem auto;text-align:center;padding:1.5rem;border-radius:8px;color:#ef4444;background:#1c1c1c">
         <strong>Could not load forecasts</strong><br>
         <span style="color:#9ca3af;font-size:0.875rem">${err.message}</span>
       </div>`;
    document.getElementById('refresh-btn').classList.remove('loading');
  }
}

// ── Controls ───────────────────────────────────────────────────────────────────

function setupControls() {
  // Sort control
  document.getElementById('sort-control').addEventListener('click', e => {
    const btn = e.target.closest('.sort-btn');
    if (!btn || btn.classList.contains('active')) return;
    document.querySelectorAll('.sort-btn').forEach(b => b.classList.remove('active'));
    btn.classList.add('active');
    state.sortBy = btn.dataset.sort;
    // Re-sort and re-render from stored site data without refetching
    if (window._backendSites) processAndRender(window._backendSites);
  });

  // Thresholds toggle
  const panel  = document.getElementById('thresholds-panel');
  const toggle = document.getElementById('thresholds-toggle');
  toggle.addEventListener('click', () => {
    const open = !panel.hidden;
    panel.hidden = open;
    toggle.classList.toggle('active', !open);
    toggle.setAttribute('aria-expanded', String(!open));
  });

  // Threshold inputs — re-process without refetching
  document.getElementById('rain-input').addEventListener('change', e => {
    state.rainThreshold = Math.max(0, Math.min(100, Number(e.target.value)));
    if (window._backendSites) processAndRender(window._backendSites);
  });

  document.getElementById('cloud-input').addEventListener('change', e => {
    state.cloudThreshold = Math.max(0, Math.min(100, Number(e.target.value)));
    if (window._backendSites) processAndRender(window._backendSites);
  });

  // Refresh button
  document.getElementById('refresh-btn').addEventListener('click', loadData);
}

// ── Init ───────────────────────────────────────────────────────────────────────

setupControls();
loadData();