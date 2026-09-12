import { writeFileSync, readFileSync } from 'node:fs';

// ---------- tokens (lifted from frontend/src/lib/theme.css, warm-dark) ----------
const T = {
  bg: '#040404', surface: '#121009', surface2: '#1d1913', border: '#322b22',
  text: '#f2e5d5', muted: '#a8917a', accent: '#ff7700', accent2: '#ffa149',
  accentFg: '#1c0e00', peach: '#ffd7b4', live: '#ff3d2e', success: '#4fb56d', gold: '#ffc633', warning: '#ffb43d',
};
const DSEG = readFileSync('dseg7.woff2').toString('base64');
const LED_PRE = `@font-face{font-family:'DSEG7';src:url(data:font/woff2;base64,${DSEG}) format('woff2');font-weight:700}`;
const LED_CSS = `
.led{font-family:'DSEG7',monospace;font-size:15px;color:${T.warning};background:#000;border-radius:9px;padding:5px 0;width:44px;gap:6px;box-shadow:inset 0 0 0 1px #221c14,inset 0 2px 6px rgba(0,0,0,.8)}
.led span{position:relative;align-self:stretch;height:17px;line-height:17px;display:flex;align-items:center;justify-content:center;text-shadow:0 0 7px rgba(255,180,61,.55)}
.led i{position:absolute;left:0;right:0;top:0;bottom:0;display:flex;align-items:center;justify-content:center;font-style:normal;color:${T.warning};opacity:.2;text-shadow:none}
.led.live{color:${T.live}}.led.live i{color:${T.live}}.led.live span{text-shadow:0 0 7px rgba(255,61,46,.6)}
.led.off span{color:transparent;text-shadow:none}
`;

const CSS = `
@import url('https://fonts.googleapis.com/css2?family=Baloo+2:wght@700;800&family=Figtree:wght@400;500;600;700;800&family=Red+Hat+Mono:wght@700&display=swap');
body{margin:0;background:${T.bg};color:${T.text};font-family:'Figtree',system-ui,-apple-system,'Segoe UI',sans-serif;-webkit-font-smoothing:antialiased;font-size:14px}
a{color:${T.accent};text-decoration:none}a:hover{color:${T.accent2}}
svg{flex:none}
.screen{position:relative;overflow:hidden;background:${T.bg};background-image:radial-gradient(110% 70% at 50% -20%,rgba(255,119,0,.10),transparent 55%),radial-gradient(80% 60% at 100% 0%,rgba(255,215,180,.05),transparent 50%)}
.display{font-family:'Baloo 2','Figtree',sans-serif;font-weight:700;letter-spacing:-.015em;line-height:1.04}
.digits{font-family:'Red Hat Mono',ui-monospace,monospace;font-weight:700;font-variant-numeric:tabular-nums;letter-spacing:-.02em}
.kicker{font-weight:700;font-size:11px;letter-spacing:.22em;text-transform:uppercase;color:${T.accent}}
.muted{color:${T.muted}}
.card{background:linear-gradient(180deg,rgba(255,255,255,.025),transparent 40%),${T.surface};border:1px solid ${T.border};border-radius:20px;overflow:clip}
.pill{display:inline-flex;align-items:center;gap:4px;white-space:nowrap;font-weight:700;font-size:10.5px;letter-spacing:.1em;text-transform:uppercase;background:${T.surface2};border:1px solid ${T.border};color:${T.muted};padding:4px 9px;border-radius:999px}
.pill.ok{color:${T.accent};border-color:rgba(255,119,0,.45)}
.pill.live{color:${T.bg};background:${T.live};border-color:${T.live}}
.btn{display:inline-flex;align-items:center;justify-content:center;gap:8px;padding:13px 18px;border-radius:12px;background:${T.accent};color:${T.accentFg};font-weight:800;font-size:14px;letter-spacing:.06em;text-transform:uppercase;border:1px solid transparent}
.btn.secondary{background:${T.surface2};border-color:${T.border};color:${T.text}}
.topbar{position:absolute;top:0;left:0;right:0;height:58px;display:flex;align-items:center;gap:10px;padding:0 16px;background:rgba(4,4,4,.82);backdrop-filter:blur(14px) saturate(1.4);border-bottom:1px solid ${T.border};z-index:5}
.subbar{position:absolute;left:0;right:0;top:58px;background:rgba(4,4,4,.82);backdrop-filter:blur(14px);border-bottom:1px solid ${T.border};z-index:4}
.tabbar{position:absolute;bottom:0;left:0;right:0;height:66px;display:flex;background:rgba(4,4,4,.86);backdrop-filter:blur(14px);border-top:1px solid ${T.border};z-index:5}
.tab{flex:1;display:flex;flex-direction:column;align-items:center;justify-content:center;gap:4px;color:${T.muted};font-size:9.6px;font-weight:700;letter-spacing:.04em;text-transform:uppercase;position:relative}
.tab.on{color:${T.accent}}
.tab.on::before{content:'';position:absolute;top:0;left:50%;transform:translateX(-50%);width:26px;height:3px;border-radius:0 0 3px 3px;background:${T.accent};box-shadow:0 0 12px 1px ${T.accent}}
.iconbtn{width:34px;height:34px;border-radius:12px;display:inline-flex;align-items:center;justify-content:center;color:${T.muted}}
.avatar{border-radius:50%;background:${T.peach};color:${T.accentFg};font-weight:800;display:inline-flex;align-items:center;justify-content:center;flex:none}
.chips{display:flex;gap:8px;align-items:center}
.chip{height:34px;padding:0 14px;border-radius:999px;border:1px solid ${T.border};background:${T.surface};color:${T.muted};font-weight:700;font-size:13px;display:inline-flex;align-items:center;gap:6px;white-space:nowrap}
.chip.on{background:${T.accent};color:${T.accentFg};border-color:${T.accent}}
.chip.livechip{color:${T.live};border-color:rgba(255,61,46,.45)}
.dot{width:7px;height:7px;border-radius:50%;background:${T.live};box-shadow:0 0 8px ${T.live}}
.days{display:flex;gap:6px;overflow:hidden}
.day{flex:none;width:52px;height:50px;border-radius:12px;display:flex;flex-direction:column;align-items:center;justify-content:center;gap:2px;color:${T.muted};font-size:11px;font-weight:600}
.day b{font-size:15px;color:${T.text};font-family:'Red Hat Mono',monospace}
.day.on{background:${T.accent};color:${T.accentFg}}.day.on b{color:${T.accentFg}}
.day .mark{width:5px;height:5px;border-radius:50%;background:${T.accent}}
.sec{display:flex;align-items:baseline;gap:8px;margin:18px 2px 8px}
.sec h2{margin:0;font-size:19px}
.sec .more{margin-left:auto;font-size:13px;font-weight:600;color:${T.accent};display:inline-flex;align-items:center;gap:2px}
.dayh{font-size:12px;font-weight:700;letter-spacing:.12em;text-transform:uppercase;color:${T.muted};margin:16px 2px 8px;display:flex;gap:8px;align-items:center}
.dayh.today{color:${T.accent}}
.comp{margin-bottom:12px}
.comph{display:flex;align-items:center;gap:10px;padding:10px 12px 10px 14px;border-bottom:1px solid ${T.border}}
.comph b{font-size:13px;font-weight:700}
.comph .rnd{font-size:12px;color:${T.muted}}
.cols{margin-left:auto;display:flex;gap:0;font-size:9px;font-weight:700;letter-spacing:.1em;text-transform:uppercase;color:${T.muted}}
.cols span{width:40px;text-align:center}
.row{display:grid;grid-template-columns:46px 1fr auto;align-items:center;min-height:60px;padding:7px 10px 7px 12px;border-bottom:1px solid ${T.border};gap:6px}
.row:last-child{border-bottom:none}
.when{display:flex;flex-direction:column;gap:1px;font-size:12px;color:${T.muted};line-height:1.15}
.when b{color:${T.text};font-weight:700}
.when.live{color:${T.live}}.when.live b{color:${T.live}}
.teams{display:flex;flex-direction:column;gap:7px;min-width:0}
.team{display:flex;align-items:center;gap:8px;font-weight:600;font-size:14px;line-height:19px;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}
.team.dim{color:${T.muted}}
.crest{width:20px;height:20px;border-radius:50%;font-size:7.5px;font-weight:800;display:inline-flex;align-items:center;justify-content:center;color:#111;flex:none;letter-spacing:.02em}
.right{display:flex;align-items:center;gap:8px}
.score{display:flex;flex-direction:column;gap:7px;align-items:center;width:40px;font-size:15px;line-height:19px;color:${T.text}}
.score.live{color:${T.live}}
.score .w{color:${T.text}}.score .l{color:${T.muted}}
.tip{display:flex;flex-direction:column;width:40px;height:46px;border:1.5px solid ${T.accent};border-radius:12px;overflow:hidden;background:rgba(255,119,0,.10);justify-content:center;align-items:center;gap:1px}
.tip span{height:19px;display:flex;align-items:center;justify-content:center;font-size:14px;color:${T.text}}
.tip.empty{border-style:dashed;border-color:rgba(255,119,0,.6);background:transparent;color:${T.accent}}
.tip.done{border-color:${T.border};background:${T.surface2}}
.tip.done span{color:${T.muted}}
.tip.none{border-style:dashed;border-color:${T.border};background:transparent;color:${T.muted}}
.tip.hit{border-color:${T.accent};background:rgba(255,119,0,.14)}.tip.hit span{color:${T.text}}
.pts{width:34px;text-align:center;font-size:12px;color:${T.muted}}
.pts.ok{color:${T.accent}}
.chev{color:${T.muted}}
.drawer{grid-column:1/-1;display:flex;align-items:center;justify-content:center;gap:14px;padding:10px 0 6px;border-top:1px dashed ${T.border};margin-top:6px}
.step{display:flex;align-items:center;gap:6px}
.step .b{width:40px;height:40px;border-radius:12px;background:${T.surface2};border:1px solid ${T.border};display:inline-flex;align-items:center;justify-content:center;color:${T.text}}
.step .v{width:34px;text-align:center;font-size:22px}
.vs{font-weight:800;opacity:.5}
.lrow{display:flex;align-items:center;gap:12px;padding:12px 14px;border-bottom:1px solid ${T.border}}
.lrow:last-child{border-bottom:none}
.rank{font-size:22px;width:44px}
.rank small{font-size:12px;color:${T.muted};font-weight:600}
.delta{font-size:11px;font-weight:700;display:inline-flex;align-items:center;gap:2px}
.delta.up{color:${T.success}}.delta.down{color:${T.live}}
.utabs{display:flex;gap:0;padding:0 8px;overflow:hidden}
.utab{padding:12px 12px 10px;font-weight:700;font-size:13px;color:${T.muted};border-bottom:2px solid transparent;white-space:nowrap}
.utab.on{color:${T.text};border-bottom-color:${T.accent}}
.navlink{display:inline-flex;align-items:center;gap:8px;padding:7px 14px;border-radius:999px;color:${T.muted};font-weight:700;font-size:12.5px;letter-spacing:.04em;text-transform:uppercase}
.navlink.on{color:${T.accent};background:rgba(255,119,0,.12)}
.raillink{display:flex;align-items:center;gap:10px;padding:9px 12px;border-radius:12px;color:${T.muted};font-weight:600;font-size:13.5px}
.raillink.on{background:rgba(255,119,0,.12);color:${T.accent}}
.railh{font-size:10.5px;font-weight:700;letter-spacing:.14em;text-transform:uppercase;color:${T.muted};margin:18px 12px 6px}
`;

const I = {
  home: '<path d="M3 11l9-8 9 8v9a2 2 0 0 1-2 2h-4v-6H9v6H5a2 2 0 0 1-2-2z"/>',
  matches: '<path d="M8 6h13M8 12h13M8 18h13M3 6h.01M3 12h.01M3 18h.01"/>',
  trophy: '<path d="M8 21h8M12 17v4M7 4h10v5a5 5 0 0 1-10 0z"/><path d="M7 6H4a2 2 0 0 0 0 4h3M17 6h3a2 2 0 0 1 0 4h-3"/>',
  users: '<path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87M16 3.13a4 4 0 0 1 0 7.75"/>',
  right: '<path d="M9 18l6-6-6-6"/>', left: '<path d="M15 18l-6-6 6-6"/>', down: '<path d="M6 9l6 6 6-6"/>',
  plus: '<path d="M12 5v14M5 12h14"/>', minus: '<path d="M5 12h14"/>',
  bell: '<path d="M18 8a6 6 0 0 0-12 0c0 7-3 9-3 9h18s-3-2-3-9M13.7 21a2 2 0 0 1-3.4 0"/>',
  share: '<circle cx="18" cy="5" r="3"/><circle cx="6" cy="12" r="3"/><circle cx="18" cy="19" r="3"/><path d="M8.6 13.5l6.8 4M15.4 6.5l-6.8 4"/>',
  lock: '<rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/>',
  search: '<circle cx="11" cy="11" r="8"/><path d="M21 21l-4.35-4.35"/>',
  up: '<path d="M12 19V5M5 12l7-7 7 7"/>', check: '<path d="M20 6L9 17l-5-5"/>',
  target: '<circle cx="12" cy="12" r="10"/><circle cx="12" cy="12" r="6"/><circle cx="12" cy="12" r="2"/>',
  chat: '<path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/>',
  filter: '<path d="M4 21v-7M4 10V3M12 21v-9M12 8V3M20 21v-5M20 12V3M1 14h6M9 8h6M17 16h6"/>',
  help: '<circle cx="12" cy="12" r="10"/><path d="M9.1 9a3 3 0 0 1 5.8 1c0 2-3 3-3 3M12 17h.01"/>',
  checkcircle: '<circle cx="12" cy="12" r="10"/><path d="M8 12l3 3 5-6"/>',
  caretup: '<path d="M6 15l6-6 6 6"/>', caretdown: '<path d="M6 9l6 6 6-6"/>',
  x: '<path d="M18 6L6 18M6 6l12 12"/>',
  globe: '<circle cx="12" cy="12" r="10"/><path d="M2 12h20M12 2a15 15 0 0 1 0 20M12 2a15 15 0 0 0 0 20"/>',
};
const ic = (n, s = 20, extra = '') => `<svg width="${s}" height="${s}" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" ${extra}>${I[n]}</svg>`;
const owl = (s = 28) => `<svg width="${s}" height="${s}" viewBox="0 0 28 28"><circle cx="14" cy="14" r="14" fill="${T.accent}"/><circle cx="9.5" cy="13" r="4.6" fill="${T.accentFg}"/><circle cx="18.5" cy="13" r="4.6" fill="${T.accentFg}"/><circle cx="9.5" cy="13" r="2" fill="${T.peach}"/><circle cx="18.5" cy="13" r="2" fill="${T.peach}"/><path d="M12.6 18.5l1.4 2.4 1.4-2.4z" fill="${T.accentFg}"/></svg>`;
const wordmark = () => `<div style="display:flex;align-items:center;gap:8px">${owl(28)}<span class="display" style="font-size:20px;letter-spacing:-.02em">matchowl</span></div>`;

const C = {
  FCB: ['FCB', '#dc052d', '#fff'], ARS: ['ARS', '#ef0107', '#fff'], RMA: ['RMA', '#f5f5f5', '#222'], LIV: ['LIV', '#c8102e', '#fff'],
  PSG: ['PSG', '#1a3a7a', '#fff'], INT: ['INT', '#0b1c3f', '#fff'], BAR: ['BAR', '#a50044', '#fff'], BVB: ['BVB', '#fde100', '#111'],
  JUV: ['JUV', '#e9e9e9', '#111'], ATM: ['ATM', '#cb3524', '#fff'], MCI: ['MCI', '#6cabdd', '#111'], CHE: ['CHE', '#034694', '#fff'],
  LEV: ['LEV', '#e32221', '#fff'], SGE: ['SGE', '#c8c8c8', '#111'], VFB: ['VFB', '#e2e2e2', '#c00'], WOB: ['WOB', '#65b32e', '#111'],
  FCU: ['FCU', '#eb1923', '#fff'], BMG: ['BMG', '#1f9a48', '#fff'], SCF: ['SCF', '#d9d9d9', '#111'], RBL: ['RBL', '#e7e7e7', '#c00'],
  NAP: ['NAP', '#12a0d7', '#111'], AJX: ['AJX', '#d2122e', '#fff'],
  UCL: ['UCL', '#0b1c3f', '#9fc8ff'], BL: ['BL', '#d20515', '#fff'], WC: ['WC', '#c9a227', '#111'], PL: ['PL', '#38003c', '#fff'], EM: ['EM', '#0a3f9e', '#fff'],
};
const crest = (k, s = 20) => { const [c, bg, fg] = C[k]; return `<span class="crest" style="width:${s}px;height:${s}px;background:${bg};color:${fg};font-size:${Math.round(s * .38)}px">${c}</span>`; };
const NAMES = { FCB: 'Bayern', ARS: 'Arsenal', RMA: 'Real Madrid', LIV: 'Liverpool', PSG: 'PSG', INT: 'Inter', BAR: 'Barcelona', BVB: 'Dortmund', JUV: 'Juventus', ATM: 'Atlético', MCI: 'Man City', CHE: 'Chelsea', LEV: 'Leverkusen', SGE: 'Frankfurt', VFB: 'Stuttgart', WOB: 'Wolfsburg', FCU: 'Union Berlin', BMG: 'Gladbach', SCF: 'Freiburg', RBL: 'Leipzig', NAP: 'Napoli', AJX: 'Ajax' };

// ---------- match row (score column = LED stadium board, option A) ----------
const led = (s, live) => s
  ? `<div class="score led${live ? ' live' : ''}"><span><i>8</i>${s[0]}</span><span><i>8</i>${s[1]}</span></div>`
  : `<div class="score led off"><span><i>8</i>8</span><span><i>8</i>8</span></div>`;
function row(o) {
  const when = `<div class="when${o.live ? ' live' : ''}"><b>${o.when[0]}</b><span>${o.when[1] ?? ''}</span></div>`;
  const teams = `<div class="teams"><span class="team">${crest(o.h)}${NAMES[o.h]}</span><span class="team">${crest(o.a)}${NAMES[o.a]}</span></div>`;
  const score = led(o.score, o.live);
  let tip;
  if (o.tip === 'empty') tip = `<div class="tip empty">${ic('plus', 18)}</div>`;
  else if (o.tip === 'none') tip = `<div class="tip none">${ic('lock', 14)}</div>`;
  else if (o.tip === 'locked') tip = `<div class="tip none"><span style="color:${T.muted}">—</span></div>`;
  else { const cls = o.score ? (o.pts > 0 ? ' hit' : ' done') : ''; tip = `<div class="tip digits${cls}"><span>${o.tip[0]}</span><span>${o.tip[1]}</span></div>`; }
  let pts = '';
  if (o.pts !== undefined) pts = `<div class="pts digits${o.pts > 0 ? ' ok' : ''}">${o.pts > 0 ? '+' : ''}${o.pts}</div>`;
  else if (o.chev !== false) pts = `<div class="pts">${ic('right', 16, 'style="color:' + T.muted + '"')}</div>`;
  const drawer = o.drawer ? `<div class="drawer"><div class="step"><span class="b">${ic('minus', 18)}</span><span class="v digits">${o.tip[0]}</span><span class="b">${ic('plus', 18)}</span></div><span class="vs">:</span><div class="step"><span class="b">${ic('minus', 18)}</span><span class="v digits">${o.tip[1]}</span><span class="b">${ic('plus', 18)}</span></div></div>` : '';
  return `<div class="row"${o.style ? ` style="${o.style}"` : ''}>${when}${teams}<div class="right">${score}${tip}${pts}</div>${drawer}</div>`;
}
function comp(o) {
  const cols = o.cols === false ? '' : `<div class="cols"><span>Score</span><span>Tip</span><span></span></div>`;
  return `<div class="card comp"><div class="comph">${crest(o.crestKey, 22)}<div style="display:flex;flex-direction:column;gap:1px;min-width:0"><b>${o.name}</b><span class="rnd">${o.round}</span></div>${cols}</div>${o.rows.join('')}</div>`;
}
const tabbar = (on) => `<nav class="tabbar">${[['home', 'Home'], ['matches', 'Matches'], ['trophy', 'Competitions'], ['users', 'Friends']].map(([i, l]) => `<a class="tab${on === l ? ' on' : ''}">${ic(i, 22)}<span>${l}</span></a>`).join('')}</nav>`;
const avatar = (s = 34, n = 'FH') => `<span class="avatar" style="width:${s}px;height:${s}px;font-size:${Math.round(s * .38)}px">${n}</span>`;
const phone = (inner) => `<div class="screen" style="width:390px;height:844px">${inner}</div>`;
const wrap = (body, extraCss = '', pre = '') => `<!doctype html>
<html>
<head>
  <meta charset="utf-8">
  <script src="./support.js"></script>
</head>
<body>
<x-dc>
<helmet>
  <style>${pre}${LED_PRE}${CSS}${LED_CSS}${extraCss}</style>
</helmet>
${body}
</x-dc>
</body>
</html>`;

// ================= Matches (Main) =================
const matchesMobile = phone(`
<header class="topbar"><span class="display" style="font-size:22px">Matches</span><span style="flex:1"></span><span class="iconbtn">${ic('search', 20)}</span>${avatar()}</header>
<div class="subbar" style="padding:10px 16px 10px;display:flex;flex-direction:column;gap:10px">
  <div class="chips"><span class="chip on">${ic('check', 14)} Mine</span><span class="chip">All</span><span style="flex:1"></span><span class="chip livechip"><span class="dot"></span> Live · 1</span><span class="iconbtn" style="width:34px">${ic('filter', 18)}</span></div>
  <div class="days">
    <span class="day"><span>Thu</span><b>10</b></span><span class="day"><span>Fri</span><b>11</b></span>
    <span class="day on"><span>Today</span><b>12</b></span>
    <span class="day"><span>Sun</span><b>13</b><span class="mark"></span></span><span class="day"><span>Mon</span><b>14</b></span>
    <span class="day"><span>Tue</span><b>15</b><span class="mark"></span></span><span class="day"><span>Wed</span><b>16</b><span class="mark"></span></span>
  </div>
</div>
<main style="position:absolute;top:174px;bottom:66px;left:0;right:0;padding:4px 16px;overflow:hidden">
  <div class="dayh today">Today · Saturday 12 Sep</div>
  ${comp({ crestKey: 'BL', name: 'Bundesliga', round: 'Matchday 3', rows: [
    row({ when: ['67’', 'Live'], live: true, h: 'FCU', a: 'BMG', score: [1, 1], tip: [2, 1] }),
    row({ when: ['15:30'], h: 'FCB', a: 'LEV', tip: 'empty' }),
    row({ when: ['15:30'], h: 'BVB', a: 'SGE', tip: [2, 1] }),
    row({ when: ['18:30'], h: 'VFB', a: 'WOB', tip: 'empty' }),
  ] })}
  <div class="dayh">Tuesday 15 Sep</div>
  ${comp({ crestKey: 'UCL', name: 'Champions League', round: 'League phase · Matchday 1', rows: [
    row({ when: ['18:45'], h: 'JUV', a: 'BVB', tip: [1, 1] }),
    row({ when: ['21:00'], h: 'FCB', a: 'ARS', tip: 'empty' }),
    row({ when: ['21:00'], h: 'RMA', a: 'LIV', tip: 'empty' }),
    row({ when: ['21:00'], h: 'ATM', a: 'PSG', tip: 'empty' }),
  ] })}
  <div class="dayh">Wednesday 16 Sep</div>
  ${comp({ crestKey: 'UCL', name: 'Champions League', round: 'League phase · Matchday 1', rows: [
    row({ when: ['21:00'], h: 'BAR', a: 'INT', tip: 'empty' }),
  ] })}
</main>
${tabbar('Matches')}`);

// ================= Home =================
const homeMobile = phone(`
<header class="topbar">${wordmark()}<span style="flex:1"></span><span class="iconbtn">${ic('bell', 20)}</span>${avatar()}</header>
<main style="position:absolute;top:58px;bottom:66px;left:0;right:0;padding:8px 16px;overflow:hidden">
  <div class="sec" style="margin-top:10px"><h2 class="display">Tip now</h2><span class="muted" style="font-size:13px">2 open · locks 15:30</span><a class="more">Today ${ic('right', 14)}</a></div>
  <div class="card comp">
    ${row({ when: ['15:30', 'BL'], h: 'FCB', a: 'LEV', tip: 'empty' })}
    ${row({ when: ['18:30', 'BL'], h: 'VFB', a: 'WOB', tip: 'empty' })}
    ${row({ when: ['Tue', '21:00'], h: 'FCB', a: 'ARS', tip: 'empty' })}
  </div>
  <div class="card" style="margin-top:12px;display:flex;align-items:center;gap:12px;padding:12px 14px;border-color:rgba(255,119,0,.35)">
    <span style="width:38px;height:38px;border-radius:12px;background:rgba(255,119,0,.14);color:${T.accent};display:inline-flex;align-items:center;justify-content:center">${ic('target', 20)}</span>
    <span style="display:flex;flex-direction:column;gap:2px;min-width:0"><b style="font-size:14px">Make your Champions League forecast</b><span class="muted" style="font-size:12.5px">One shot before kickoff · locks in 3 days</span></span>
    <span style="margin-left:auto;color:${T.muted}">${ic('right', 18)}</span>
  </div>
  <div class="sec"><h2 class="display">Live</h2><a class="more">Matches ${ic('right', 14)}</a></div>
  <div class="card comp">${row({ when: ['67’', 'BL'], live: true, h: 'FCU', a: 'BMG', score: [1, 1], tip: [2, 1] })}</div>
  <div class="sec"><h2 class="display">Your leagues</h2><a class="more">Friends ${ic('right', 14)}</a></div>
  <div class="card">
    <div class="lrow"><span class="rank digits">#2<small>/8</small></span><span style="display:flex;flex-direction:column;gap:2px;min-width:0"><b>Bürocup</b><span class="muted" style="font-size:12px">Lena leads · 101 pts</span></span><span style="margin-left:auto;display:flex;flex-direction:column;align-items:flex-end;gap:2px"><span class="digits" style="font-size:15px">94</span><span class="delta up">${ic('caretup', 12)} 1</span></span></div>
    <div class="lrow"><span class="rank digits">#5<small>/12</small></span><span style="display:flex;flex-direction:column;gap:2px;min-width:0"><b>Stammtisch</b><span class="muted" style="font-size:12px">Tom leads · 98 pts</span></span><span style="margin-left:auto;display:flex;flex-direction:column;align-items:flex-end;gap:2px"><span class="digits" style="font-size:15px">94</span><span class="delta down">${ic('caretdown', 12)} 2</span></span></div>
  </div>
  <div class="sec"><h2 class="display">Yesterday</h2><span class="pill ok">+4 pts</span></div>
  <div class="card comp">${row({ when: ['FT', 'BL'], h: 'RBL', a: 'SCF', score: [2, 1], tip: [2, 1], pts: 4 })}</div>
</main>
${tabbar('Home')}`);

// ================= Competition hub =================
const compMobile = phone(`
<header class="topbar" style="height:66px"><span class="iconbtn" style="margin-left:-8px">${ic('left', 22)}</span>${crest('UCL', 34)}
  <div style="display:flex;flex-direction:column;gap:1px;min-width:0"><b style="font-size:15px;white-space:nowrap;overflow:hidden;text-overflow:ellipsis">Champions League</b><span class="muted" style="font-size:12.5px;display:inline-flex;align-items:center;gap:3px;font-weight:600">2026/27 ${ic('down', 13)} <span class="pill live" style="margin-left:6px;font-size:9px;padding:2px 6px">Live</span></span></div>
  <span style="flex:1"></span>
  <span class="chip on" style="height:32px;padding:0 12px;font-size:12px">${ic('checkcircle', 15)} Playing</span>
</header>
<div class="subbar" style="top:66px">
  <div class="utabs"><span class="utab">Overview</span><span class="utab on">Matches</span><span class="utab">Table</span><span class="utab">Knockout</span><span class="utab">Forecast</span></div>
</div>
<main style="position:absolute;top:112px;bottom:66px;left:0;right:0;padding:12px 16px;overflow:hidden">
  <div class="chips" style="margin-bottom:4px"><span class="chip on">Matchday 1 ${ic('down', 14)}</span><span class="chip">By team</span><span style="flex:1"></span><span class="chip">${ic('target', 14)} Now</span></div>
  <div class="dayh">Tuesday 15 Sep</div>
  <div class="card comp">
    ${row({ when: ['18:45'], h: 'JUV', a: 'BVB', tip: [1, 1] })}
    ${row({ when: ['21:00'], h: 'FCB', a: 'ARS', tip: 'empty' })}
    ${row({ when: ['21:00'], h: 'RMA', a: 'LIV', tip: 'empty' })}
    ${row({ when: ['21:00'], h: 'ATM', a: 'PSG', tip: 'empty' })}
  </div>
  <div class="dayh">Wednesday 16 Sep</div>
  <div class="card comp">
    ${row({ when: ['18:45'], h: 'NAP', a: 'AJX', tip: 'empty' })}
    ${row({ when: ['21:00'], h: 'BAR', a: 'INT', tip: 'empty' })}
    ${row({ when: ['21:00'], h: 'MCI', a: 'CHE', tip: 'empty' })}
  </div>
  <div class="dayh">Thursday 17 Sep</div>
  <div class="card comp">${row({ when: ['21:00'], h: 'PSG', a: 'RMA', tip: 'empty' })}</div>
</main>
${tabbar('Competitions')}`);

// ================= Match page =================
const matchMobile = phone(`
<header class="topbar"><span class="iconbtn" style="margin-left:-8px">${ic('left', 22)}</span>
  <span style="display:flex;flex-direction:column;gap:1px;min-width:0"><b style="font-size:14px">Champions League</b><span class="muted" style="font-size:12px">League phase · Matchday 1</span></span>
  <span style="flex:1"></span><span class="iconbtn">${ic('bell', 20)}</span><span class="iconbtn">${ic('share', 20)}</span></header>
<main style="position:absolute;top:58px;bottom:66px;left:0;right:0;padding:12px 16px;overflow:hidden">
  <div style="display:grid;grid-template-columns:1fr auto 1fr;align-items:center;gap:8px;padding:14px 6px 18px">
    <div style="display:flex;flex-direction:column;align-items:center;gap:8px">${crest('FCB', 56)}<b style="font-size:14px">Bayern</b></div>
    <div style="display:flex;flex-direction:column;align-items:center;gap:3px;min-width:96px"><span class="digits" style="font-size:26px">21:00</span><span class="muted" style="font-size:12.5px;font-weight:600">Tue 15 Sep</span><span class="pill ok" style="margin-top:4px">locks in 3d 4h</span></div>
    <div style="display:flex;flex-direction:column;align-items:center;gap:8px">${crest('ARS', 56)}<b style="font-size:14px">Arsenal</b></div>
  </div>
  <div class="card" style="padding:14px 16px 16px;border-color:rgba(255,119,0,.45);box-shadow:0 0 0 1px rgba(255,119,0,.18),0 8px 30px -8px rgba(255,119,0,.3)">
    <div style="display:flex;align-items:center;gap:8px"><span class="kicker">Your tip</span><span style="flex:1"></span><span class="muted" style="font-size:12px;display:inline-flex;align-items:center;gap:4px">${ic('check', 13)} saved</span></div>
    <div style="display:flex;align-items:center;justify-content:center;gap:18px;margin:16px 0 12px">
      <div class="step"><span class="b" style="width:44px;height:44px">${ic('minus', 20)}</span><span class="v digits" style="font-size:34px;width:44px">2</span><span class="b" style="width:44px;height:44px">${ic('plus', 20)}</span></div>
      <span class="vs" style="font-size:22px">:</span>
      <div class="step"><span class="b" style="width:44px;height:44px">${ic('minus', 20)}</span><span class="v digits" style="font-size:34px;width:44px">1</span><span class="b" style="width:44px;height:44px">${ic('plus', 20)}</span></div>
    </div>
    <div style="display:flex;justify-content:center;gap:14px;font-size:12px" class="muted"><span><b style="color:${T.text}">Bayern win</b> · tendency 2 pts</span><span>exact 4 pts</span></div>
  </div>
  <div class="sec"><h2 class="display" style="font-size:17px">Friends’ picks</h2><span class="muted" style="font-size:12.5px">shown at kickoff</span></div>
  <div class="card" style="padding:12px 14px;display:flex;align-items:center;gap:10px">
    <span style="display:flex">${avatar(28, 'LE')}<span style="margin-left:-8px">${avatar(28, 'TO')}</span><span style="margin-left:-8px">${avatar(28, 'MK')}</span><span style="margin-left:-8px">${avatar(28, 'JS')}</span></span>
    <span style="font-size:13px"><b>4 of 7</b> <span class="muted">league mates have tipped</span></span><span style="margin-left:auto;color:${T.muted}">${ic('lock', 16)}</span>
  </div>
  <div class="sec"><h2 class="display" style="font-size:17px">League phase</h2><a class="more">Table ${ic('right', 14)}</a></div>
  <div class="card" style="padding:4px 0">
    ${[[6, 'FCB', 3, '+4', 7], [7, 'ARS', 3, '+3', 7]].map(([p, k, pl, d, pts]) => `<div style="display:grid;grid-template-columns:28px 1fr 36px 36px 40px;align-items:center;padding:8px 14px;font-size:13.5px;background:rgba(255,119,0,.06)"><span class="muted digits">${p}</span><span style="display:flex;align-items:center;gap:8px;font-weight:600">${crest(k, 18)}${NAMES[k]}</span><span class="muted digits" style="text-align:right">${pl}</span><span class="muted digits" style="text-align:right">${d}</span><span class="digits" style="text-align:right">${pts}</span></div>`).join('')}
    <div style="padding:6px 14px 8px;font-size:12px" class="muted">Both in the R16 zone · 1–8 direct, 9–24 play-off</div>
  </div>
</main>
${tabbar('Matches')}`);

// ================= Friends =================
const leagueCard = (o) => `<div class="card" style="margin-bottom:12px">
  <div style="display:flex;align-items:center;gap:10px;padding:12px 14px 8px"><b style="font-size:16px">${o.name}</b>${o.owner ? `<span class="pill">owner</span>` : ''}<span style="flex:1"></span>${o.unread ? `<span class="pill ok">${ic('chat', 12)} ${o.unread}</span>` : `<span class="muted">${ic('chat', 16)}</span>`}</div>
  <div style="display:grid;grid-template-columns:1fr 1fr 1fr;padding:2px 14px 12px;gap:8px">
    <span style="display:flex;flex-direction:column;gap:1px"><span class="digits" style="font-size:24px">#${o.rank}<small class="muted" style="font-size:12px">/${o.n}</small></span><span class="delta ${o.dir}">${ic(o.dir === 'up' ? 'caretup' : 'caretdown', 12)} ${o.d} this matchday</span></span>
    <span style="display:flex;flex-direction:column;gap:1px"><span class="digits" style="font-size:24px">${o.pts}</span><span class="muted" style="font-size:11.5px">your points</span></span>
    <span style="display:flex;flex-direction:column;gap:1px;align-items:flex-end;text-align:right"><span style="display:flex;align-items:center;gap:6px">${avatar(22, o.leaderIn)}<span class="digits" style="font-size:15px">${o.leaderPts}</span></span><span class="muted" style="font-size:11.5px">${o.leader} leads · ${o.gap} behind</span></span>
  </div>
  <div style="display:flex;align-items:center;gap:8px;padding:10px 14px;border-top:1px solid ${T.border};font-size:12.5px" class="muted">${o.foot}<span style="flex:1"></span>${ic('right', 16)}</div>
</div>`;
const friendsMobile = phone(`
<header class="topbar"><span class="display" style="font-size:22px">Friends</span><span style="flex:1"></span><span class="chip" style="height:32px;padding:0 12px;font-size:12px">${ic('plus', 15)} New league</span></header>
<main style="position:absolute;top:58px;bottom:66px;left:0;right:0;padding:12px 16px;overflow:hidden">
  ${leagueCard({ name: 'Bürocup', owner: true, unread: 3, rank: 2, n: 8, dir: 'up', d: 1, pts: 94, leader: 'Lena', leaderIn: 'LE', leaderPts: 101, gap: 7, foot: 'Matchday 3 · Tom leads the round with 9 pts' })}
  ${leagueCard({ name: 'Stammtisch', rank: 5, n: 12, dir: 'down', d: 2, pts: 94, leader: 'Tom', leaderIn: 'TO', leaderPts: 98, gap: 4, foot: 'Champions League starts Tue · 3 of 12 tipped' })}
  <div class="lrow card" style="padding:12px 14px;margin-bottom:12px"><span style="color:${T.muted}">${ic('globe', 18)}</span><span style="display:flex;flex-direction:column;gap:2px"><b>Global</b><span class="muted" style="font-size:12px">everyone on Matchowl</span></span><span style="margin-left:auto" class="digits">#148<small class="muted" style="font-size:12px">/2,310</small></span><span style="color:${T.muted}">${ic('right', 16)}</span></div>
  <div class="sec"><h2 class="display" style="font-size:17px">Activity</h2></div>
  <div class="card">
    ${[['LE', '<b>Lena</b> took 1st in Bürocup', '2h'], ['TO', '<b>Tom</b> hit the exact score · Leipzig 2–1', 'yesterday'], ['MK', '<b>Mara</b> joined Stammtisch', '2d'], ['JS', '<b>Jonas</b> placed the Champions League forecast', '3d']].map(([a, t, w]) => `<div class="lrow" style="padding:10px 14px">${avatar(30, a)}<span style="font-size:13.5px;line-height:1.3">${t}</span><span class="muted" style="margin-left:auto;font-size:12px;white-space:nowrap">${w}</span></div>`).join('')}
  </div>
</main>
${tabbar('Friends')}`);

// ================= Row states sheet =================
const rowStates = `<div class="screen" style="width:390px;height:1080px;padding:20px 16px;background-image:none">
  <div class="kicker">Match row</div>
  <h1 class="display" style="margin:4px 0 2px;font-size:26px">Score vs. tip</h1>
  <p class="muted" style="margin:0 0 6px;font-size:13px;line-height:1.4">Result = the stadium board: seven-segment LEDs on black, amber, red while live, unlit before kickoff. Tip = the orange capsule in the app’s mono, always the same spot, always yours. Both stack home over away, aligned with the team names.</p>
  <div class="card comp" style="margin-top:12px"><div class="comph">${crest('BL', 22)}<div style="display:flex;flex-direction:column;gap:1px"><b>Bundesliga</b><span class="rnd">Matchday 3</span></div><div class="cols"><span>Score</span><span>Tip</span><span></span></div></div>
  ${row({ when: ['15:30'], h: 'FCB', a: 'LEV', tip: 'empty' })}
  ${row({ when: ['15:30'], h: 'BVB', a: 'SGE', tip: [2, 1] })}
  ${row({ when: ['15:30'], h: 'VFB', a: 'WOB', tip: [1, 0], drawer: true, style: 'background:rgba(255,119,0,.05)' })}
  ${row({ when: ['67’', 'Live'], live: true, h: 'FCU', a: 'BMG', score: [1, 1], tip: [2, 1] })}
  ${row({ when: ['FT'], h: 'RBL', a: 'SCF', score: [2, 1], tip: [2, 1], pts: 4 })}
  ${row({ when: ['FT'], h: 'MCI', a: 'CHE', score: [0, 3], tip: [2, 1], pts: 0 })}
  ${row({ when: ['AET'], h: 'RMA', a: 'LIV', score: [2, 1], tip: 'locked', pts: 0 })}
  ${row({ when: ['Sat', '21:00'], h: 'PSG', a: 'INT', tip: 'none' })}
  </div>
  <div style="display:grid;grid-template-columns:46px 1fr;gap:4px 10px;margin:14px 4px 0;font-size:12.5px;line-height:1.35" class="muted">
    <span style="color:${T.text};font-weight:700">1</span><span>Untipped, open: unlit board, dashed capsule with a plus. Tap it to edit in place.</span>
    <span style="color:${T.text};font-weight:700">2</span><span>Tipped: solid capsule, your numbers. Tap to change until lock.</span>
    <span style="color:${T.text};font-weight:700">3</span><span>Editing: the row opens a stepper drawer, saves on close.</span>
    <span style="color:${T.text};font-weight:700">4</span><span>Live: minute and board in live red, capsule frozen.</span>
    <span style="color:${T.text};font-weight:700">5–6</span><span>Played: board lit in amber. Capsule filled; hit stays orange, miss goes grey. Points on the right.</span>
    <span style="color:${T.text};font-weight:700">7</span><span>Locked without a tip: dash, no points.</span>
    <span style="color:${T.text};font-weight:700">8</span><span>Not tippable yet (KO pairing undecided): lock in the capsule.</span>
  </div>
</div>`;

// ================= Desktop =================
const dtop = (on) => `<header class="topbar" style="padding:0 40px;gap:16px">${wordmark()}<nav style="display:flex;gap:6px;margin-left:16px">${[['home', 'Home'], ['matches', 'Matches'], ['trophy', 'Competitions'], ['users', 'Friends']].map(([i, l]) => `<a class="navlink${on === l ? ' on' : ''}">${ic(i, 17)}${l}</a>`).join('')}</nav><span style="flex:1"></span><span class="iconbtn">${ic('search', 20)}</span><span class="iconbtn">${ic('help', 20)}</span><span class="iconbtn">${ic('bell', 20)}</span><span style="display:flex;align-items:center;gap:8px">${avatar(34)}<b style="font-size:13.5px">floholz</b>${ic('down', 14, `style="color:${T.muted}"`)}</span></header>`;

const desktopMatches = `<div class="screen" style="width:1440px;height:900px">
${dtop('Matches')}
<div style="position:absolute;top:58px;left:0;right:0;bottom:0;display:grid;grid-template-columns:240px minmax(0,1fr) 400px;gap:28px;padding:20px 40px 0">
  <aside style="display:flex;flex-direction:column">
    <div class="railh" style="margin-top:6px">Show</div>
    <a class="raillink on">${ic('check', 16)} My competitions</a>
    <a class="raillink">${ic('globe', 16)} Everything</a>
    <a class="raillink" style="color:${T.live}"><span class="dot"></span> Live · 1</a>
    <div class="railh">Playing</div>
    <a class="raillink">${crest('BL', 20)} Bundesliga</a>
    <a class="raillink">${crest('UCL', 20)} Champions League</a>
    <a class="raillink">${crest('PL', 20)} Premier League</a>
    <div class="railh">Not playing</div>
    <a class="raillink" style="opacity:.7">${crest('WC', 20)} World Cup 2026 <span class="pill" style="margin-left:auto">done</span></a>
    <a class="raillink" style="opacity:.7">${crest('EM', 20)} Euro 2028 <span class="pill" style="margin-left:auto">soon</span></a>
  </aside>
  <section style="min-width:0;overflow:hidden">
    <div class="days" style="margin-bottom:6px">
      ${[['Thu', 10], ['Fri', 11]].map(([d, n]) => `<span class="day"><span>${d}</span><b>${n}</b></span>`).join('')}
      <span class="day on"><span>Today</span><b>12</b></span>
      ${[['Sun', 13, 1], ['Mon', 14], ['Tue', 15, 1], ['Wed', 16, 1], ['Thu', 17, 1], ['Fri', 18], ['Sat', 19, 1], ['Sun', 20, 1]].map(([d, n, m]) => `<span class="day"><span>${d}</span><b>${n}</b>${m ? '<span class="mark"></span>' : ''}</span>`).join('')}
    </div>
    <div class="dayh today">Today · Saturday 12 Sep</div>
    ${comp({ crestKey: 'BL', name: 'Bundesliga', round: 'Matchday 3', rows: [
      row({ when: ['67’', 'Live'], live: true, h: 'FCU', a: 'BMG', score: [1, 1], tip: [2, 1] }),
      row({ when: ['15:30'], h: 'FCB', a: 'LEV', tip: 'empty' }),
      row({ when: ['15:30'], h: 'BVB', a: 'SGE', tip: [2, 1] }),
      row({ when: ['18:30'], h: 'VFB', a: 'WOB', tip: 'empty' }),
    ] })}
    <div class="dayh">Tuesday 15 Sep</div>
    ${comp({ crestKey: 'UCL', name: 'Champions League', round: 'League phase · Matchday 1', rows: [
      row({ when: ['18:45'], h: 'JUV', a: 'BVB', tip: [1, 1] }),
      row({ when: ['21:00'], h: 'FCB', a: 'ARS', tip: 'empty', style: `background:rgba(255,119,0,.07);box-shadow:inset 3px 0 0 ${T.accent}` }),
      row({ when: ['21:00'], h: 'RMA', a: 'LIV', tip: 'empty' }),
      row({ when: ['21:00'], h: 'ATM', a: 'PSG', tip: 'empty' }),
    ] })}
  </section>
  <aside>
    <div class="card" style="padding:0">
      <div style="display:flex;align-items:center;gap:8px;padding:12px 16px;border-bottom:1px solid ${T.border}"><span style="display:flex;flex-direction:column;gap:1px"><b style="font-size:13.5px">Champions League</b><span class="muted" style="font-size:12px">League phase · Matchday 1</span></span><span style="flex:1"></span><span class="iconbtn">${ic('share', 18)}</span><span class="iconbtn">${ic('x', 18)}</span></div>
      <div style="display:grid;grid-template-columns:1fr auto 1fr;align-items:center;gap:8px;padding:18px 12px 16px">
        <div style="display:flex;flex-direction:column;align-items:center;gap:8px">${crest('FCB', 52)}<b style="font-size:13.5px">Bayern</b></div>
        <div style="display:flex;flex-direction:column;align-items:center;gap:3px;min-width:96px"><span class="digits" style="font-size:24px">21:00</span><span class="muted" style="font-size:12px;font-weight:600">Tue 15 Sep</span><span class="pill ok" style="margin-top:4px">locks in 3d 4h</span></div>
        <div style="display:flex;flex-direction:column;align-items:center;gap:8px">${crest('ARS', 52)}<b style="font-size:13.5px">Arsenal</b></div>
      </div>
      <div style="margin:0 14px 14px;padding:12px 14px 14px;border:1px solid rgba(255,119,0,.45);border-radius:14px;background:rgba(255,119,0,.05)">
        <div style="display:flex;align-items:center;gap:8px"><span class="kicker">Your tip</span><span style="flex:1"></span><span class="muted" style="font-size:12px">not placed</span></div>
        <div style="display:flex;align-items:center;justify-content:center;gap:16px;margin:14px 0 8px">
          <div class="step"><span class="b">${ic('minus', 18)}</span><span class="v digits" style="font-size:30px;width:40px;color:${T.muted}">–</span><span class="b">${ic('plus', 18)}</span></div>
          <span class="vs" style="font-size:20px">:</span>
          <div class="step"><span class="b">${ic('minus', 18)}</span><span class="v digits" style="font-size:30px;width:40px;color:${T.muted}">–</span><span class="b">${ic('plus', 18)}</span></div>
        </div>
      </div>
      <div style="padding:0 16px 14px;display:flex;flex-direction:column;gap:10px">
        <div style="display:flex;align-items:center;gap:10px;font-size:13px"><span style="display:flex">${avatar(26, 'LE')}<span style="margin-left:-8px">${avatar(26, 'TO')}</span><span style="margin-left:-8px">${avatar(26, 'MK')}</span></span><span><b>4 of 7</b> <span class="muted">league mates have tipped</span></span><span style="margin-left:auto;color:${T.muted}">${ic('lock', 15)}</span></div>
        <div style="border-top:1px solid ${T.border};padding-top:10px">
          <div style="display:flex;align-items:baseline;gap:8px;margin-bottom:6px"><b style="font-size:13px">League phase</b><a class="more" style="margin-left:auto;font-size:12.5px;display:inline-flex;align-items:center">Table ${ic('right', 13)}</a></div>
          ${[[6, 'FCB', 3, '+4', 7], [7, 'ARS', 3, '+3', 7]].map(([p, k, pl, d, pts]) => `<div style="display:grid;grid-template-columns:24px 1fr 32px 32px 36px;align-items:center;padding:6px 8px;font-size:13px;background:rgba(255,119,0,.06);border-radius:8px;margin-bottom:3px"><span class="muted digits">${p}</span><span style="display:flex;align-items:center;gap:8px;font-weight:600">${crest(k, 18)}${NAMES[k]}</span><span class="muted digits" style="text-align:right">${pl}</span><span class="muted digits" style="text-align:right">${d}</span><span class="digits" style="text-align:right">${pts}</span></div>`).join('')}
        </div>
      </div>
    </div>
  </aside>
</div>
</div>`;

const desktopHome = `<div class="screen" style="width:1440px;height:900px">
${dtop('Home')}
<div style="position:absolute;top:58px;left:0;right:0;bottom:0;display:grid;grid-template-columns:minmax(0,1fr) 380px;gap:28px;padding:20px 40px 0;max-width:1260px;margin:0 auto">
  <section style="min-width:0">
    <div class="sec" style="margin-top:6px"><h2 class="display" style="font-size:22px">Tip now</h2><span class="muted" style="font-size:13px">2 open today · locks 15:30</span><a class="more">All of today ${ic('right', 14)}</a></div>
    <div class="card comp">
      <div class="comph">${crest('BL', 22)}<div style="display:flex;flex-direction:column;gap:1px"><b>Bundesliga</b><span class="rnd">Matchday 3 · today</span></div><div class="cols"><span>Score</span><span>Tip</span><span></span></div></div>
      ${row({ when: ['15:30'], h: 'FCB', a: 'LEV', tip: 'empty' })}
      ${row({ when: ['18:30'], h: 'VFB', a: 'WOB', tip: 'empty' })}
    </div>
    ${comp({ crestKey: 'UCL', name: 'Champions League', round: 'Matchday 1 · Tuesday', rows: [row({ when: ['21:00'], h: 'FCB', a: 'ARS', tip: 'empty' }), row({ when: ['21:00'], h: 'RMA', a: 'LIV', tip: 'empty' })] })}
    <div class="sec"><h2 class="display" style="font-size:22px">Live</h2></div>
    <div class="card comp">${row({ when: ['67’', 'BL'], live: true, h: 'FCU', a: 'BMG', score: [1, 1], tip: [2, 1] })}</div>
    <div class="sec"><h2 class="display" style="font-size:22px">Yesterday</h2><span class="pill ok">+4 pts</span><a class="more">Results ${ic('right', 14)}</a></div>
    <div class="card comp">${row({ when: ['FT', 'BL'], h: 'RBL', a: 'SCF', score: [2, 1], tip: [2, 1], pts: 4 })}</div>
  </section>
  <aside style="display:flex;flex-direction:column;gap:12px;padding-top:6px">
    <div class="card" style="display:flex;align-items:center;gap:12px;padding:12px 14px;border-color:rgba(255,119,0,.35)">
      <span style="width:38px;height:38px;border-radius:12px;background:rgba(255,119,0,.14);color:${T.accent};display:inline-flex;align-items:center;justify-content:center">${ic('target', 20)}</span>
      <span style="display:flex;flex-direction:column;gap:2px;min-width:0"><b style="font-size:14px">Make your Champions League forecast</b><span class="muted" style="font-size:12.5px">One shot · locks in 3 days</span></span><span style="margin-left:auto;color:${T.muted}">${ic('right', 18)}</span>
    </div>
    <div class="sec" style="margin:8px 2px 0"><h2 class="display" style="font-size:18px">Your leagues</h2><a class="more">Friends ${ic('right', 14)}</a></div>
    <div class="card">
      <div class="lrow"><span class="rank digits">#2<small>/8</small></span><span style="display:flex;flex-direction:column;gap:2px"><b>Bürocup</b><span class="muted" style="font-size:12px">Lena leads · 101 pts</span></span><span style="margin-left:auto;display:flex;flex-direction:column;align-items:flex-end;gap:2px"><span class="digits" style="font-size:15px">94</span><span class="delta up">${ic('caretup', 12)} 1</span></span></div>
      <div class="lrow"><span class="rank digits">#5<small>/12</small></span><span style="display:flex;flex-direction:column;gap:2px"><b>Stammtisch</b><span class="muted" style="font-size:12px">Tom leads · 98 pts</span></span><span style="margin-left:auto;display:flex;flex-direction:column;align-items:flex-end;gap:2px"><span class="digits" style="font-size:15px">94</span><span class="delta down">${ic('caretdown', 12)} 2</span></span></div>
      <div class="lrow"><span style="color:${T.muted}">${ic('globe', 18)}</span><b>Global</b><span style="margin-left:auto" class="digits">#148<small class="muted" style="font-size:12px">/2,310</small></span></div>
    </div>
    <div class="sec" style="margin:8px 2px 0"><h2 class="display" style="font-size:18px">Activity</h2></div>
    <div class="card">
      ${[['LE', '<b>Lena</b> took 1st in Bürocup', '2h'], ['TO', '<b>Tom</b> hit the exact score · Leipzig 2–1', '1d'], ['JS', '<b>Jonas</b> placed the CL forecast', '3d']].map(([a, t, w]) => `<div class="lrow" style="padding:10px 14px">${avatar(28, a)}<span style="font-size:13px;line-height:1.3">${t}</span><span class="muted" style="margin-left:auto;font-size:12px;white-space:nowrap">${w}</span></div>`).join('')}
    </div>
  </aside>
</div>
</div>`;

const files = {
  'Main.dc.html': matchesMobile, 'Home.dc.html': homeMobile, 'Competition.dc.html': compMobile, 'Match.dc.html': matchMobile,
  'Friends.dc.html': friendsMobile, 'RowStates.dc.html': rowStates, 'DesktopMatches.dc.html': desktopMatches, 'DesktopHome.dc.html': desktopHome,
};
for (const [n, b] of Object.entries(files)) writeFileSync(n, wrap(b));

// ======================= Score vs tip — options (page 2, A chosen) =======================
const OPT_PRE = `@import url('https://fonts.googleapis.com/css2?family=Bebas+Neue&family=Teko:wght@600&display=swap');`;
const OPT_CSS = `
.tall{font-family:'Bebas Neue','Teko',sans-serif;font-weight:400;font-size:24px;letter-spacing:.02em;gap:3px}
.tall span{height:21px;line-height:21px}
.tile{font-family:'Red Hat Mono',monospace;font-weight:700;color:${T.warning};background:#000;border-radius:9px;padding:5px 0;width:44px;gap:6px;box-shadow:inset 0 0 0 1px #221c14}
.tile span{height:17px;line-height:17px;font-size:14px}
.tile.live{color:${T.live}}
.tip.lbl{position:relative;height:50px;padding-top:6px}
.tip.lbl .tag{position:absolute;top:-1px;left:50%;transform:translateX(-50%);font-size:7.5px;font-weight:800;letter-spacing:.12em;text-transform:uppercase;color:${T.accentFg};background:${T.accent};padding:1px 5px 0;border-radius:0 0 5px 5px;line-height:9px;font-style:normal}
.tip.lbl.done .tag,.tip.lbl.none .tag{background:${T.border};color:${T.muted}}
.opt h1{font-size:22px;margin:2px 0 4px}
.opt p{margin:0 0 12px;font-size:12.5px;line-height:1.4}
`;
function rowV(v, o) {
  const s = o.score, lv = o.live ? ' live' : '';
  const dimH = s && s[0] < s[1] && !o.live && v !== 'led' ? ' dim' : '';
  const dimA = s && s[1] < s[0] && !o.live && v !== 'led' ? ' dim' : '';
  const when = `<div class="when${lv}"><b>${o.when[0]}</b><span>${o.when[1] ?? ''}</span></div>`;
  const teams = `<div class="teams"><span class="team${dimH}">${crest(o.h)}${NAMES[o.h]}</span><span class="team${dimA}">${crest(o.a)}${NAMES[o.a]}</span></div>`;
  let score;
  if (v === 'led') score = led(s, o.live);
  else if (v === 'tall') score = s ? `<div class="score tall${lv}"><span class="${s[0] < s[1] ? 'l' : 'w'}">${s[0]}</span><span class="${s[1] < s[0] ? 'l' : 'w'}">${s[1]}</span></div>` : `<div class="score tall" style="color:transparent"><span>0</span><span>0</span></div>`;
  else if (v === 'tile') score = s ? `<div class="score tile${lv}"><span>${s[0]}</span><span>${s[1]}</span></div>` : `<div class="score tile" style="color:${T.border}"><span>–</span><span>–</span></div>`;
  else score = s ? `<div class="score digits${lv}"><span class="${s[0] < s[1] ? 'l' : 'w'}">${s[0]}</span><span class="${s[1] < s[0] ? 'l' : 'w'}">${s[1]}</span></div>` : `<div class="score" style="color:transparent"><span>·</span><span>·</span></div>`;
  const lbl = v === 'label' ? ' lbl' : '';
  const tag = v === 'label' ? '<i class="tag">tip</i>' : '';
  let tip;
  if (o.tip === 'empty') tip = `<div class="tip empty${lbl}">${tag}${ic('plus', 18)}</div>`;
  else { const cls = s ? (o.pts > 0 ? ' hit' : ' done') : ''; tip = `<div class="tip digits${cls}${lbl}">${tag}<span>${o.tip[0]}</span><span>${o.tip[1]}</span></div>`; }
  const pts = o.pts !== undefined ? `<div class="pts digits${o.pts > 0 ? ' ok' : ''}">${o.pts > 0 ? '+' : ''}${o.pts}</div>` : `<div class="pts">${ic('right', 16, 'style="color:' + T.muted + '"')}</div>`;
  return `<div class="row">${when}${teams}<div class="right">${score}${tip}${pts}</div></div>`;
}
const sample = (v) => [
  rowV(v, { when: ['15:30'], h: 'FCB', a: 'LEV', tip: 'empty' }),
  rowV(v, { when: ['15:30'], h: 'BVB', a: 'SGE', tip: [2, 1] }),
  rowV(v, { when: ['67’', 'Live'], live: true, h: 'FCU', a: 'BMG', score: [1, 1], tip: [2, 1] }),
  rowV(v, { when: ['FT'], h: 'RBL', a: 'SCF', score: [2, 1], tip: [2, 1], pts: 4 }),
  rowV(v, { when: ['FT'], h: 'MCI', a: 'CHE', score: [0, 3], tip: [2, 1], pts: 0 }),
  rowV(v, { when: ['AET'], h: 'RMA', a: 'LIV', score: [2, 1], tip: [1, 1], pts: 2 }),
].join('');
const hdrCols = `<div class="cols"><span>Score</span><span>Tip</span><span></span></div>`;
const optBoard = (letter, title, blurb, v, hdr) => `<div class="screen opt" style="width:390px;height:640px;padding:20px 16px;background-image:none">
  <div class="kicker">Option ${letter}</div><h1 class="display">${title}</h1><p class="muted">${blurb}</p>
  <div class="card comp"><div class="comph">${crest('BL', 22)}<div style="display:flex;flex-direction:column;gap:1px"><b>Bundesliga</b><span class="rnd">Matchday 3</span></div>${hdr}</div>${sample(v)}</div>
</div>`;
const opts = {
  'OptionA.dc.html': optBoard('A', 'Stadium board', 'Chosen. The result is a seven-segment LED panel in amber, live in red. Unlit segments show on upcoming matches, so the board reads as “result goes here” before kickoff. The tip keeps the app’s mono digits in the orange capsule. Two different objects, not two scorelines.', 'led', hdrCols),
  'OptionB.dc.html': optBoard('B', 'Broadcast digits', 'The result in a tall condensed broadcast face, bare and big, loser dimmed. The tip stays small mono in the capsule. Different font, different size, same column logic.', 'tall', hdrCols),
  'OptionC.dc.html': optBoard('C', 'Board tile, house font', 'Same Red Hat Mono for both, but the result sits on a black inset tile in amber while the tip is the outlined orange capsule. Shape and colour do the separating; nothing new in the type system.', 'tile', hdrCols),
  'OptionD.dc.html': optBoard('D', 'Labelled capsule', 'Cheapest fix: the capsule carries a tiny “tip” tab on its top edge, the result stays bare mono. No new font, no tile. Relies on the label being read.', 'label', ''),
};
for (const [n, b] of Object.entries(opts)) writeFileSync(n, wrap(b, OPT_CSS, OPT_PRE));

// ======================= canvas =======================
const canvas = {
  pages: [{ id: 'page-1', name: 'Screens' }, { id: 'page-2', name: 'Score vs tip' }],
  artboards: [
    { file: 'Home.dc.html', title: 'Home', x: 0, y: 0, w: 390, h: 844, page: 'page-1' },
    { file: 'Main.dc.html', title: 'Matches', x: 480, y: 0, w: 390, h: 844, page: 'page-1' },
    { file: 'Competition.dc.html', title: 'Competition hub', x: 960, y: 0, w: 390, h: 844, page: 'page-1' },
    { file: 'Match.dc.html', title: 'Match page', x: 1440, y: 0, w: 390, h: 844, page: 'page-1' },
    { file: 'Friends.dc.html', title: 'Friends', x: 1920, y: 0, w: 390, h: 844, page: 'page-1' },
    { file: 'RowStates.dc.html', title: 'Match row states', x: 2400, y: 0, w: 390, h: 1080, page: 'page-1' },
    { file: 'DesktopMatches.dc.html', title: 'Desktop · Matches', x: 0, y: 1260, w: 1440, h: 900, page: 'page-1' },
    { file: 'DesktopHome.dc.html', title: 'Desktop · Home', x: 1520, y: 1260, w: 1440, h: 900, page: 'page-1' },
    { file: 'OptionA.dc.html', title: 'A · Stadium board · chosen', x: 0, y: 0, w: 390, h: 640, page: 'page-2' },
    { file: 'OptionB.dc.html', title: 'B · Broadcast digits', x: 480, y: 0, w: 390, h: 640, page: 'page-2' },
    { file: 'OptionC.dc.html', title: 'C · Board tile', x: 960, y: 0, w: 390, h: 640, page: 'page-2' },
    { file: 'OptionD.dc.html', title: 'D · Labelled capsule', x: 1440, y: 0, w: 390, h: 640, page: 'page-2' },
  ],
  annotations: [
    { id: 'shell', page: 'page-1', x: 0, y: -260, w: 440, text: 'Shell\n4 tabs: Home · Matches · Competitions · Friends. Mobile top bar is contextual (page title or back + context), page-owned second row under it (filters, date strip, tabs). No more kicker + h1 slabs on list pages; the owl lives in the wordmark and the accent.' },
    { id: 'row', page: 'page-1', x: 480, y: -260, w: 440, text: 'The match row\nScore = the LED stadium board (option A, chosen): amber on black, red while live, unlit before kickoff. Tip = orange capsule in the app’s mono, same column on every screen, tap to edit in place. Stacked home over away, so a matchday is one column of tips to fill. Points sit right of the capsule after FT. See the states sheet →' },
    { id: 'matches-note', page: 'page-1', x: 960, y: -260, w: 440, text: 'Matches\nOnly lists matches. "Mine" is on by default (competitions you play); "All" is one tap away. Day strip scrolls the list; dots mark days with matches. Live chip appears only when something is live.' },
    { id: 'match-note', page: 'page-1', x: 1440, y: -260, w: 440, text: 'Match page /m/{id}\nHero + big steppers replace the growing inline card. Friends’ picks, bots and the mini table move here. Two-legged ties add an "Other leg · aggregate" row directly under the hero. After FT the hero shows the LED board large.' },
    { id: 'friends-note', page: 'page-1', x: 1920, y: -260, w: 440, text: 'Friends\nEach league card answers: where am I, who leads, how far, what happened this matchday. Activity feed kept as a secondary section.' },
    { id: 'desktop-note', page: 'page-1', x: 0, y: 2240, w: 640, text: 'Desktop rule\n≥ 900px: the tab bar becomes top-bar links. Matches = rail (240) · list · detail panel (400). The detail panel IS the match page, so nothing is designed twice. Home = content column + 380px side rail. Everything caps at 1260px and centres; between 600 and 900 the rail collapses and the detail panel becomes the match page route.' },
    { id: 'opts-note', page: 'page-2', x: 0, y: -230, w: 900, text: 'Score vs tip\nProblem: two stacked digit pairs in the same face read like HT / FT. Each option changes only the SCORE column; the tip capsule is the constant across the whole app (A–C) so your input never moves or changes shape. Rows: open, tipped, live, exact hit, miss, tendency hit.\n\nChosen: A, rolled through every screen on the Screens page. The LED board is unmistakably “the stadium’s number”, the unlit segments give upcoming rows a natural empty state, and live red on a black panel is exactly the ticker feeling. Cost: one extra 5 KB font, digits only.' },
  ],
  launch: { view: 'canvas', page: 'page-1' },
};
writeFileSync('canvas.json', JSON.stringify(canvas, null, 2));
console.log('built', Object.keys(files).length + Object.keys(opts).length, 'artboards');

// ======================= Knockout: AET / penalties / two legs =======================
const KO_CSS = `
.led span{position:relative}
.led .mk{position:absolute;left:5px;top:50%;width:5px;height:5px;margin-top:-2.5px;border-radius:50%;background:currentColor;box-shadow:0 0 6px currentColor;opacity:1}
.tip{position:relative}
.tip .mk{position:absolute;left:4px;width:4px;height:4px;border-radius:50%;background:${T.accent};box-shadow:0 0 5px ${T.accent}}
.tip.done .mk{background:${T.muted};box-shadow:none}
.tip .mk.h{top:11px}.tip .mk.a{top:31px}
.when small{font-size:10.5px;color:${T.muted};font-family:'Red Hat Mono',monospace;font-weight:700}
.legtab{display:flex;flex-direction:column;align-items:center;justify-content:center;gap:1px;height:28px;width:24px;margin-right:-8px;padding-right:2px;background:#000;border-radius:7px 0 0 7px;box-shadow:inset 1px 0 0 #221c14,inset 0 1px 0 #221c14,inset 0 -1px 0 #221c14;line-height:1;font-size:6.5px;font-weight:800;letter-spacing:.06em;text-transform:uppercase;color:${T.muted}}.legtab b{color:${T.warning};font-size:9px;font-family:'Red Hat Mono',monospace}.legtab.live b{color:${T.live}}.led.docked{position:relative;z-index:1}
.strip{grid-column:1/-1;display:flex;align-items:center;gap:8px;font-size:11.5px;color:${T.muted};padding:7px 2px 1px;margin-top:3px;border-top:1px dashed ${T.border}}
.strip b{color:${T.text};font-weight:700}
.strip .agg{font-family:'Red Hat Mono',monospace;font-weight:700;color:${T.text}}
.strip .cv{margin-left:auto;color:${T.muted}}
`;
const ledK = (s, o) => {
  if (!s) return `<div class="score led off"><span><i>8</i>8</span><span><i>8</i>8</span></div>`;
  const m = (side) => o.adv === side ? '<em class="mk"></em>' : '';
  return `<div class="score led${o.live ? ' live' : ''}"><span><i>8</i>${s[0]}${m('h')}</span><span><i>8</i>${s[1]}${m('a')}</span></div>`;
};
function rowK(o) {
  const when = `<div class="when${o.live ? ' live' : ''}"><b>${o.when[0]}</b>${o.when[1] ? `<small>${o.when[1]}</small>` : ''}</div>`;
  const legtab = o.leg ? `<div class="legtab${o.live ? ' live' : ''}"><b>${o.leg}</b><span>leg</span></div>` : '';
  const teams = `<div class="teams"><span class="team">${crest(o.h)}${NAMES[o.h]}</span><span class="team">${crest(o.a)}${NAMES[o.a]}</span></div>`;
  const score = ledK(o.score, o).replace('class="score led', o.leg ? 'class="score led docked' : 'class="score led');
  let tip;
  const tm = o.tipAdv ? `<em class="mk ${o.tipAdv}"></em>` : '';
  if (o.tip === 'empty') tip = `<div class="tip empty">${ic('plus', 18)}</div>`;
  else { const cls = o.score ? (o.pts > 0 ? ' hit' : ' done') : ''; tip = `<div class="tip digits${cls}">${tm}<span>${o.tip[0]}</span><span>${o.tip[1]}</span></div>`; }
  const pts = o.pts !== undefined ? `<div class="pts digits${o.pts > 0 ? ' ok' : ''}">${o.pts > 0 ? '+' : ''}${o.pts}</div>` : `<div class="pts">${ic('right', 16, 'style="color:' + T.muted + '"')}</div>`;
  const strip = o.strip ? `<div class="strip">${o.strip}${ic('right', 14, 'class="cv"')}</div>` : '';
  return `<div class="row"${o.style ? ` style="${o.style}"` : ''}>${when}${teams}<div class="right">${legtab}${score}${tip}${pts}</div>${strip}</div>`;
}
const koStates = `<div class="screen" style="width:390px;height:1180px;padding:20px 16px;background-image:none">
  <div class="kicker">Match row · knockout</div>
  <h1 class="display" style="margin:4px 0 2px;font-size:26px">Extra time, pens, two legs</h1>
  <p class="muted" style="margin:0 0 6px;font-size:13px;line-height:1.4">The board always shows the final score, the status says how it ended: FT, AET or PEN. The 90’ and 120’ detail lives on the match page. A small LED dot marks who goes through: on the board the real advancer, on the capsule your pick. Ties get a strip under the row.</p>
  <div class="card comp" style="margin-top:12px"><div class="comph">${crest('UCL', 22)}<div style="display:flex;flex-direction:column;gap:1px"><b>Champions League</b><span class="rnd">Round of 16</span></div><div class="cols"><span>Score</span><span>Tip</span><span></span></div></div>
  ${rowK({ when: ['21:00'], leg: '1st', h: 'ARS', a: 'FCB', tip: [2, 1], strip: `2nd leg <b>Wed 18 Mar</b> · in Munich` })}
  ${rowK({ when: ['FT'], leg: '1st', h: 'ARS', a: 'FCB', score: [2, 1], tip: [2, 1], pts: 4, strip: `2nd leg <b>Wed 18 Mar</b> · Arsenal lead <span class="agg">2–1</span>` })}
  ${rowK({ when: ['ET', '97’'], leg: '2nd', live: true, h: 'FCB', a: 'ARS', score: [1, 0], tip: [1, 0], tipAdv: 'h', strip: `Agg <span class="agg">2–2</span> · 1st leg 1–2 · extra time` })}
  ${rowK({ when: ['PEN'], leg: '2nd', h: 'FCB', a: 'ARS', score: [2, 1], adv: 'h', tip: [1, 0], tipAdv: 'h', pts: 3, strip: `Agg <span class="agg">3–3</span> · 1st leg 1–2 · <b>Bayern</b> advance on penalties` })}
  </div>
  <div class="card comp" style="margin-top:12px"><div class="comph">${crest('WC', 22)}<div style="display:flex;flex-direction:column;gap:1px"><b>World Cup</b><span class="rnd">Final · single match</span></div><div class="cols"><span>Score</span><span>Tip</span><span></span></div></div>
  ${rowK({ when: ['Sun', '21:00'], h: 'RMA', a: 'LIV', tip: [1, 1], tipAdv: 'a' })}
  ${rowK({ when: ['AET'], h: 'RMA', a: 'LIV', score: [2, 1], adv: 'h', tip: [1, 1], tipAdv: 'a', pts: 0 })}
  ${rowK({ when: ['PEN'], h: 'PSG', a: 'INT', score: [0, 0], adv: 'a', tip: [0, 0], tipAdv: 'a', pts: 5 })}
  </div>
  <div style="display:grid;grid-template-columns:46px 1fr;gap:4px 10px;margin:14px 4px 0;font-size:12.5px;line-height:1.35" class="muted">
    <span style="color:${T.text};font-weight:700">1–2</span><span>First leg: tipped like a group match, draws allowed. The strip points to the second leg and, once played, carries the lead.</span>
    <span style="color:${T.text};font-weight:700">3</span><span>Live in extra time: status “ET 97’”, board red, strip shows the aggregate so far.</span>
    <span style="color:${T.text};font-weight:700">4</span><span>Deciding leg after pens: status “PEN”, board shows the final score, dot on the tie winner. Capsule shows your tip, dot on your pick. Strip: aggregate, first leg, who advances and how.</span>
    <span style="color:${T.text};font-weight:700">5</span><span>KO tip that predicts a draw carries your penalty pick as the dot; the capsule alone tells the whole tip.</span>
    <span style="color:${T.text};font-weight:700">6–7</span><span>Single match after ET or pens: same language without the strip. Status “AET” or “PEN” is the only extra signal in the row.</span>
  </div>
</div>`;

// ---- Match page after full time (deciding leg, penalties) ----
const matchFT = phone(`
<header class="topbar"><span class="iconbtn" style="margin-left:-8px">${ic('left', 22)}</span>
  <span style="display:flex;flex-direction:column;gap:1px;min-width:0"><b style="font-size:14px">Champions League</b><span class="muted" style="font-size:12px">Round of 16 · 2nd leg</span></span>
  <span style="flex:1"></span><span class="iconbtn">${ic('share', 20)}</span></header>
<main style="position:absolute;top:58px;bottom:66px;left:0;right:0;padding:12px 16px;overflow:hidden">
  <div style="display:grid;grid-template-columns:1fr auto 1fr;align-items:center;gap:8px;padding:14px 0 10px">
    <div style="display:flex;flex-direction:column;align-items:center;gap:8px">${crest('FCB', 56)}<b style="font-size:14px">Bayern</b></div>
    <div style="display:flex;flex-direction:column;align-items:center;gap:6px">
      <div class="led" style="display:flex;flex-direction:row;width:auto;padding:8px 12px;gap:10px;font-size:30px;align-items:center;border-radius:12px"><span style="height:34px;line-height:34px;width:26px"><i>8</i>2</span><span style="height:34px;line-height:34px;font-size:22px;opacity:.55">:</span><span style="height:34px;line-height:34px;width:26px"><i>8</i>1</span></div>
      <div style="display:flex;gap:5px"><span class="pill">AET</span><span class="pill">PEN</span></div>
      <span class="muted" style="font-size:12px;font-weight:600">after 90’ <span class="digits" style="color:${T.text}">1:0</span> · after 120’ <span class="digits" style="color:${T.text}">2:1</span></span>
    </div>
    <div style="display:flex;flex-direction:column;align-items:center;gap:8px">${crest('ARS', 56)}<b style="font-size:14px">Arsenal</b></div>
  </div>
  <div class="card" style="padding:10px 14px;display:flex;align-items:center;gap:8px;font-size:13px;white-space:nowrap"><span class="pill ok">agg 3–3</span><span><b>Bayern</b> advance on pens</span><span style="flex:1"></span><span class="muted" style="font-size:12px;display:inline-flex;align-items:center;gap:4px">1st leg <b class="digits" style="color:${T.text}">1–2</b> ${ic('right', 14)}</span></div>
  <div class="sec"><h2 class="display" style="font-size:17px">Your tip</h2><span style="flex:1"></span><span class="digits" style="color:${T.accent};font-size:18px">+3</span></div>
  <div class="card" style="padding:12px 14px;display:flex;align-items:center;gap:14px">
    <div class="tip digits hit" style="width:44px;height:50px"><em class="mk h"></em><span>1</span><span>0</span></div>
    <div style="display:flex;flex-direction:column;gap:4px;font-size:13px;min-width:0"><span><b>1:0</b>, <b>Bayern</b> on penalties</span><span class="muted" style="font-size:12.5px">tendency <b style="color:${T.accent}">+2</b> · advancer <b style="color:${T.accent}">+1</b> · exact 0</span></div>
  </div>
  <div class="sec"><h2 class="display" style="font-size:17px">Friends’ picks</h2><span class="muted" style="font-size:12.5px">Bürocup</span></div>
  <div class="card" style="padding:4px 0">
    ${[['LE', 'Lena', [2, 1], 'h', 5, true], ['TO', 'Tom', [0, 1], 'a', 0, false], ['MK', 'Mara', [1, 1], 'h', 1, false], ['JS', 'Jonas', [1, 0], 'h', 3, false]].map(([a, n, t, adv, p, ex]) => `<div style="display:flex;align-items:center;gap:10px;padding:7px 14px;font-size:13.5px">${avatar(26, a)}<span style="font-weight:600">${n}</span>${ex ? `<span class="pill ok" style="font-size:9px">${ic('target', 10)} exact</span>` : ''}<span style="flex:1"></span><div class="tip digits${p > 0 ? ' hit' : ' done'}" style="width:36px;height:40px;transform:scale(.9)"><em class="mk ${adv}" style="${adv === 'h' ? 'top:9px' : 'top:26px'}"></em><span style="height:16px;font-size:12.5px">${t[0]}</span><span style="height:16px;font-size:12.5px">${t[1]}</span></div><span class="pts digits${p > 0 ? ' ok' : ''}" style="width:30px">${p > 0 ? '+' : ''}${p}</span></div>`).join('')}
  </div>
</main>
${tabbar('Matches')}`);

writeFileSync('KOStates.dc.html', wrap(koStates, KO_CSS));
writeFileSync('MatchFT.dc.html', wrap(matchFT, KO_CSS));
canvas.artboards.push(
  { file: 'KOStates.dc.html', title: 'Knockout row states', x: 2880, y: 0, w: 390, h: 1180, page: 'page-1' },
  { file: 'MatchFT.dc.html', title: 'Match page · after FT', x: 3360, y: 0, w: 390, h: 844, page: 'page-1' },
);
canvas.annotations.push({ id: 'ko-note', page: 'page-1', x: 2880, y: -260, w: 860, text: 'Knockout rules\n• The board always shows the FINAL score; the status column says how it ended: FT, AET or PEN. The 90’ / 120’ timeline lives on the match page, never in the row.\n• A small LED dot marks the advancer: real one on the board, your pick on the capsule. A drawn KO tip + dot = your penalty pick, so the capsule alone is the full tip.\n• Two legs: a “1st / leg” tab docked to the front of the scoreboard (one black block with the board), and a strip under the row: next leg date, or aggregate · first-leg score · who advances and how. The strip links to the other leg.\n• Match page after FT: the board goes big in the hero, AET/PEN pills, aggregate strip, your tip card turns into the points breakdown, and friends’ picks become visible with the same capsules.' });
writeFileSync('canvas.json', JSON.stringify(canvas, null, 2));
console.log('built KO boards');
