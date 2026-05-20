// Daily English — Shared API Helper
const API_BASE = '/api';

async function api(path, opts = {}) {
  const res = await fetch(API_BASE + path, {
    headers: { 'Content-Type': 'application/json' },
    ...opts,
    body: opts.body ? JSON.stringify(opts.body) : undefined,
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(err.error || err.message || 'request failed');
  }
  return res.json();
}

// shorthand
const GET  = (path) => api(path);
const POST = (path, body) => api(path, { method: 'POST', body });
const PUT  = (path, body) => api(path, { method: 'PUT', body });

// Apply font size and theme settings globally
// Synchronous: restore theme from localStorage to avoid flash
(function() {
  try {
    var t = localStorage.getItem('de_theme');
    if (t) document.documentElement.setAttribute('data-theme', t);
  } catch(e) {}
})();
// Async: load from server and sync
(async function() {
  try {
    const settings = await GET('/settings');
    if (settings.settings_json) {
      const parsed = JSON.parse(settings.settings_json);
      if (parsed.font_size) {
        const sizes = { small: '14px', medium: '15px', large: '17px', xlarge: '19px' };
        document.documentElement.style.setProperty('--text-base', sizes[parsed.font_size] || '15px');
      }
      if (parsed.theme) {
        document.documentElement.setAttribute('data-theme', parsed.theme);
        try { localStorage.setItem('de_theme', parsed.theme); } catch(e) {}
      }
    }
  } catch(e) {}
})();
