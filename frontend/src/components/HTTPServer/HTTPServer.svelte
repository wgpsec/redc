<script>
  import { GetHTTPServerConfig, SetHTTPServerConfig, StartHTTPServer, StopHTTPServer, GetHTTPServerStatus } from '../../../wailsjs/go/main/App.js';

  let { t = {} } = $props();

  let httpForm = $state({ enabled: false, port: 8899, host: '127.0.0.1', token: '' });
  let httpStatus = $state({ running: false, url: '', token: '' });
  let httpSaving = $state(false);
  let httpMessage = $state('');
  let httpMessageType = $state('');
  let httpLoaded = $state(false);

  async function loadConfig() {
    if (httpLoaded) return;
    try {
      const cfg = await GetHTTPServerConfig();
      httpForm = {
        enabled: cfg.enabled || false,
        port: cfg.port || 8899,
        host: cfg.host || '127.0.0.1',
        token: cfg.token || '',
      };
      const status = await GetHTTPServerStatus();
      httpStatus = { running: status.running || false, url: status.url || '', token: status.token || '' };
      httpLoaded = true;
    } catch(e) {
      console.error('Failed to load HTTP server config:', e);
    }
  }

  async function handleStart() {
    httpMessage = '';
    httpSaving = true;
    try {
      await StartHTTPServer(httpForm.port, httpForm.host, httpForm.token);
      const status = await GetHTTPServerStatus();
      httpStatus = { running: status.running || false, url: status.url || '', token: status.token || '' };
      httpMessage = t.httpServerStartSuccess || 'HTTP Server started';
      httpMessageType = 'success';
    } catch(e) {
      httpMessage = (t.httpServerStartFailed || 'Start failed') + ': ' + (e.message || String(e));
      httpMessageType = 'error';
    } finally {
      httpSaving = false;
      setTimeout(() => { httpMessage = ''; }, 4000);
    }
  }

  async function handleStop() {
    httpMessage = '';
    httpSaving = true;
    try {
      await StopHTTPServer();
      httpStatus = { running: false, url: '', token: '' };
      httpMessage = t.httpServerStopSuccess || 'HTTP Server stopped';
      httpMessageType = 'success';
    } catch(e) {
      httpMessage = (t.httpServerStopFailed || 'Stop failed') + ': ' + (e.message || String(e));
      httpMessageType = 'error';
    } finally {
      httpSaving = false;
      setTimeout(() => { httpMessage = ''; }, 3000);
    }
  }

  async function handleSave() {
    httpMessage = '';
    httpSaving = true;
    try {
      await SetHTTPServerConfig(httpForm.enabled, httpForm.port, httpForm.host, httpForm.token);
      httpMessage = t.httpServerSaveSuccess || 'Config saved';
      httpMessageType = 'success';
    } catch(e) {
      httpMessage = String(e.message || e);
      httpMessageType = 'error';
    } finally {
      httpSaving = false;
      setTimeout(() => { httpMessage = ''; }, 3000);
    }
  }

  function copyToClipboard(text) {
    navigator.clipboard.writeText(text).catch(() => {});
  }

  $effect(() => { loadConfig(); });
</script>

<div class="space-y-4">
  <!-- Status Banner -->
  {#if httpStatus.running}
  <div class="bg-emerald-50 border border-emerald-100 rounded-xl px-5 py-3 flex items-center justify-between flex-wrap gap-2">
    <div class="flex items-center gap-2.5">
      <span class="inline-block w-2.5 h-2.5 rounded-full bg-emerald-500 animate-pulse"></span>
      <span class="text-[13px] text-emerald-700 font-semibold">{t.httpServerRunning || 'Running'}</span>
      <span class="text-[13px] text-emerald-600 font-mono">{httpStatus.url}</span>
    </div>
    <div class="flex items-center gap-2">
      <button class="h-7 px-3 text-[11px] font-medium rounded-lg bg-emerald-100 hover:bg-emerald-200 text-emerald-700 cursor-pointer transition-colors" onclick={() => copyToClipboard(httpStatus.url)}>
        <span class="flex items-center gap-1">
          <svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M15.666 3.888A2.25 2.25 0 0 0 13.5 2.25h-3c-1.03 0-1.9.693-2.166 1.638m7.332 0c.055.194.084.4.084.612v0a.75.75 0 0 1-.75.75H9.75a.75.75 0 0 1-.75-.75v0c0-.212.03-.418.084-.612m7.332 0c.646.049 1.288.11 1.927.184 1.1.128 1.907 1.077 1.907 2.185V19.5a2.25 2.25 0 0 1-2.25 2.25H6.75A2.25 2.25 0 0 1 4.5 19.5V6.257c0-1.108.806-2.057 1.907-2.185a48.208 48.208 0 0 1 1.927-.184" /></svg>
          {t.httpServerCopyUrl || 'Copy URL'}
        </span>
      </button>
      <button class="h-7 px-3 text-[11px] font-medium rounded-lg bg-emerald-100 hover:bg-emerald-200 text-emerald-700 cursor-pointer transition-colors" onclick={() => copyToClipboard(httpStatus.token)}>
        <span class="flex items-center gap-1">
          <svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M15.75 5.25a3 3 0 0 1 3 3m3 0a6 6 0 0 1-7.029 5.912c-.563-.097-1.159.026-1.563.43L10.5 17.25H8.25v2.25H6v2.25H2.25v-2.818c0-.597.237-1.17.659-1.591l6.499-6.499c.404-.404.527-1 .43-1.563A6 6 0 1 1 21.75 8.25z" /></svg>
          {t.httpServerCopyToken || 'Copy Token'}
        </span>
      </button>
    </div>
  </div>
  {:else}
  <div class="bg-gray-50 border border-gray-100 rounded-xl px-5 py-3 flex items-center gap-2.5">
    <span class="inline-block w-2.5 h-2.5 rounded-full bg-gray-300"></span>
    <span class="text-[13px] text-gray-500">{t.httpServerStopped || '服务未运行'}</span>
  </div>
  {/if}

  <!-- Config Card -->
  <div class="bg-white rounded-xl border border-gray-100 overflow-hidden">
    <div class="px-5 py-3 border-b border-gray-100">
      <h3 class="text-[13px] font-semibold text-gray-900">{t.httpServerConfig || '服务配置'}</h3>
      <p class="text-[11px] text-gray-500 mt-0.5">{t.httpServerDesc || '通过浏览器访问 RedC GUI（无需桌面应用）'}</p>
    </div>
    <div class="px-5 py-4 space-y-3">
      <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
        <div>
          <label class="block text-[11px] font-medium text-gray-500 mb-1.5">{t.httpServerHost || '监听地址'}</label>
          <input type="text" bind:value={httpForm.host} placeholder="127.0.0.1"
            class="w-full h-9 px-3 text-[12px] bg-gray-50 border-0 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow font-mono" />
        </div>
        <div>
          <label class="block text-[11px] font-medium text-gray-500 mb-1.5">{t.httpServerPort || '端口'}</label>
          <input type="number" bind:value={httpForm.port} min="1024" max="65535" placeholder="8899"
            class="w-full h-9 px-3 text-[12px] bg-gray-50 border-0 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow font-mono" />
        </div>
      </div>
      <div>
        <label class="block text-[11px] font-medium text-gray-500 mb-1.5">{t.httpServerToken || 'Access Token'}</label>
        <input type="text" bind:value={httpForm.token} placeholder={t.httpServerTokenHint || '留空自动生成'}
          class="w-full h-9 px-3 text-[12px] bg-gray-50 border-0 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow font-mono" />
      </div>

      {#if httpMessage}
      <p class="text-[12px] rounded-lg px-3 py-2 {httpMessageType === 'success' ? 'text-emerald-600 bg-emerald-50' : 'text-red-600 bg-red-50'}">{httpMessage}</p>
      {/if}

      <div class="flex gap-2 pt-1">
        <button onclick={handleSave} disabled={httpSaving}
          class="h-8 px-4 text-[12px] font-medium rounded-lg bg-gray-100 hover:bg-gray-200 text-gray-700 transition-colors cursor-pointer disabled:opacity-50">{t.httpServerSaveConfig || '保存配置'}</button>
        {#if !httpStatus.running}
          <button onclick={handleStart} disabled={httpSaving}
            class="h-8 px-4 text-[12px] font-medium rounded-lg bg-gray-900 hover:bg-gray-800 text-white transition-colors cursor-pointer disabled:opacity-50 inline-flex items-center gap-1.5">
            <svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M5.25 5.653c0-.856.917-1.398 1.667-.986l11.54 6.348a1.125 1.125 0 0 1 0 1.971l-11.54 6.347a1.125 1.125 0 0 1-1.667-.985V5.653z" /></svg>
            {t.httpServerStart || '启动'}
          </button>
        {:else}
          <button onclick={handleStop} disabled={httpSaving}
            class="h-8 px-4 text-[12px] font-medium rounded-lg bg-red-500 hover:bg-red-600 text-white transition-colors cursor-pointer disabled:opacity-50 inline-flex items-center gap-1.5">
            <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M5.25 7.5A2.25 2.25 0 0 1 7.5 5.25h9a2.25 2.25 0 0 1 2.25 2.25v9a2.25 2.25 0 0 1-2.25 2.25h-9a2.25 2.25 0 0 1-2.25-2.25v-9z" /></svg>
            {t.httpServerStop || '停止'}
          </button>
        {/if}
      </div>
    </div>
  </div>

  <!-- Usage Guide -->
  <div class="bg-white rounded-xl border border-gray-100 overflow-hidden">
    <div class="px-5 py-3 border-b border-gray-100">
      <h3 class="text-[13px] font-semibold text-gray-900">{t.httpServerGuide || '使用说明'}</h3>
    </div>
    <div class="px-5 py-4 space-y-2 text-[12px] text-gray-600 leading-relaxed">
      <div class="flex items-start gap-2">
        <span class="text-gray-400 mt-0.5 flex-shrink-0">1.</span>
        <span>{t.httpServerGuide1 || '配置监听地址和端口，点击"启动"按钮'}</span>
      </div>
      <div class="flex items-start gap-2">
        <span class="text-gray-400 mt-0.5 flex-shrink-0">2.</span>
        <span>{t.httpServerGuide2 || '在浏览器中访问显示的 URL'}</span>
      </div>
      <div class="flex items-start gap-2">
        <span class="text-gray-400 mt-0.5 flex-shrink-0">3.</span>
        <span>{t.httpServerGuide3 || '使用 Access Token 进行认证登录'}</span>
      </div>
      <div class="flex items-start gap-2">
        <span class="text-gray-400 mt-0.5 flex-shrink-0">4.</span>
        <span>{t.httpServerGuide4 || '如需远程访问，将监听地址改为 0.0.0.0'}</span>
      </div>
    </div>
  </div>
</div>
