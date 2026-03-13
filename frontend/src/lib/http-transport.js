/**
 * HTTP Transport Polyfill for RedC GUI
 *
 * When running in HTTP Server mode (not inside Wails desktop app),
 * this module installs shims for:
 *   - window['go']['main']['App'] → POST /api/call
 *   - window.runtime → SSE-based event system
 *
 * The polyfill is transparent: all existing wailsjs code works unchanged.
 */

let sseSource = null;
const sseListeners = {};  // { eventName: [{ cb, remaining }] }
let sseReconnectAttempts = 0;
const SSE_MAX_RECONNECT = 10;
let serverDisconnected = false;

function getToken() {
  return localStorage.getItem('redc_token') || '';
}

/** @returns {Record<string, string>} */
function buildHeaders() {
  const t = getToken();
  return t ? { 'Authorization': `Bearer ${t}`, 'Content-Type': 'application/json' } : { 'Content-Type': 'application/json' };
}

async function apiCall(method, args) {
  if (serverDisconnected) {
    throw new Error('服务器已断开连接 / Server disconnected');
  }
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), 15000);
  try {
    const resp = await fetch('/api/call', {
      method: 'POST',
      headers: buildHeaders(),
      body: JSON.stringify({ method, args }),
      signal: controller.signal,
    });
    clearTimeout(timeout);
    if (resp.status === 401) {
      promptToken();
      throw new Error('Unauthorized');
    }
    const data = await resp.json();
    if (data.error) throw new Error(data.error);
    // Successful call — reset disconnect state
    if (serverDisconnected) {
      serverDisconnected = false;
      sseReconnectAttempts = 0;
      hideDisconnectBanner();
      connectSSE();
    }
    return data.result;
  } catch (e) {
    clearTimeout(timeout);
    if (e.name === 'AbortError') {
      serverDisconnected = true;
      showDisconnectBanner();
      throw new Error('请求超时，服务器可能已关闭 / Request timeout');
    }
    throw e;
  }
}

function connectSSE() {
  if (sseSource) return;
  if (sseReconnectAttempts >= SSE_MAX_RECONNECT) {
    serverDisconnected = true;
    showDisconnectBanner();
    return;
  }
  const token = getToken();
  const url = token ? `/api/events?token=${token}` : '/api/events';
  sseSource = new EventSource(url);

  sseSource.onopen = () => {
    sseReconnectAttempts = 0;
    serverDisconnected = false;
    hideDisconnectBanner();
  };

  sseSource.onmessage = (e) => {
    try {
      const msg = JSON.parse(e.data);
      const { event, data } = msg;
      if (!event || event === 'connected') return;
      const listeners = sseListeners[event] || [];
      const remaining = [];
      for (const entry of listeners) {
        entry.cb(data);
        if (entry.remaining === -1 || --entry.remaining > 0) {
          remaining.push(entry);
        }
      }
      sseListeners[event] = remaining;
    } catch (_) {}
  };

  sseSource.onerror = () => {
    if (sseSource) {
      sseSource.close();
      sseSource = null;
    }
    sseReconnectAttempts++;
    if (sseReconnectAttempts >= SSE_MAX_RECONNECT) {
      serverDisconnected = true;
      showDisconnectBanner();
      return;
    }
    // Exponential backoff: 2s, 4s, 8s, ... up to 30s
    const delay = Math.min(2000 * Math.pow(2, sseReconnectAttempts - 1), 30000);
    setTimeout(connectSSE, delay);
  };
}

function subscribeEvent(name, cb, maxCallbacks) {
  if (!sseListeners[name]) sseListeners[name] = [];
  sseListeners[name].push({ cb, remaining: maxCallbacks });
  connectSSE();
  // Return unsubscribe function
  return () => {
    if (sseListeners[name]) {
      sseListeners[name] = sseListeners[name].filter(e => e.cb !== cb);
    }
  };
}

function promptToken() {
  const existing = localStorage.getItem('redc_token');
  const t = window.prompt(
    'RedC HTTP Server — 请输入访问 Token\n(Enter your access token)',
    existing || ''
  );
  if (t !== null && t.trim()) {
    localStorage.setItem('redc_token', t.trim());
    // Reconnect SSE with new token
    if (sseSource) {
      sseSource.close();
      sseSource = null;
    }
    connectSSE();
  }
}

function showDisconnectBanner() {
  if (document.getElementById('redc-disconnect-banner')) return;
  const banner = document.createElement('div');
  banner.id = 'redc-disconnect-banner';
  banner.style.cssText = 'position:fixed;top:0;left:0;right:0;z-index:99999;background:#dc2626;color:#fff;text-align:center;padding:10px 16px;font-size:14px;font-family:system-ui,sans-serif;';
  banner.innerHTML = '⚠️ 服务器连接已断开 / Server disconnected &nbsp;&nbsp;' +
    '<button onclick="window.__redcReconnect__()" style="background:#fff;color:#dc2626;border:none;padding:4px 12px;border-radius:4px;cursor:pointer;font-size:13px;">重新连接 / Reconnect</button>';
  document.body.prepend(banner);
}

function hideDisconnectBanner() {
  const el = document.getElementById('redc-disconnect-banner');
  if (el) el.remove();
}

// Global reconnect handler
window.__redcReconnect__ = function() {
  serverDisconnected = false;
  sseReconnectAttempts = 0;
  hideDisconnectBanner();
  connectSSE();
};

export function installHTTPTransport() {
  // Install window['go']['main']['App'] proxy
  if (!window['go']) window['go'] = {};
  if (!window['go']['main']) window['go']['main'] = {};
  window['go']['main']['App'] = new Proxy({}, {
    get(_, method) {
      return (...args) => apiCall(method, args);
    }
  });

  // Install window.runtime shim
  window.runtime = {
    EventsOnMultiple(name, cb, maxCallbacks) {
      return subscribeEvent(name, cb, maxCallbacks);
    },
    EventsOn(name, cb) {
      return subscribeEvent(name, cb, -1);
    },
    EventsOnce(name, cb) {
      return subscribeEvent(name, cb, 1);
    },
    EventsOff(...names) {
      for (const name of names) {
        delete sseListeners[name];
      }
    },
    EventsEmit() {
      // No-op from frontend in HTTP mode (backend handles broadcasts)
    },
    // Window controls — no-op in browser
    WindowMinimise() {},
    WindowMaximise() {},
    WindowUnmaximise() {},
    WindowIsMaximised() { return Promise.resolve(false); },
    WindowToggleMaximise() {},
    WindowSetSize() {},
    WindowSetPosition() {},
    Quit() { window.close(); },
    // Environment — report as "web" platform
    Environment() {
      return Promise.resolve({ platform: 'web', arch: 'unknown', buildType: 'production' });
    },
    // Logging no-ops
    LogPrint() {}, LogTrace() {}, LogDebug() {}, LogInfo() {}, LogWarning() {}, LogError() {}, LogFatal() {},
  };

  // Check token on first load
  if (!getToken()) {
    promptToken();
  } else {
    connectSSE();
  }
}

export function isHTTPMode() {
  return !window.__wails_loaded__;
}
