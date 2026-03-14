<script>
  import { SaveProxyConfig, SetDebugLogging, GetTerraformMirrorConfig, SaveTerraformMirrorConfig, TestTerraformEndpoints, SetNotificationEnabled, SetSpotMonitorEnabled, SetSpotAutoRecoverEnabled, GetWebhookConfig, SetWebhookConfig, TestWebhook, GetHTTPServerConfig, SetHTTPServerConfig, StartHTTPServer, StopHTTPServer, GetHTTPServerStatus } from '../../../wailsjs/go/main/App.js';

let { t, config = $bindable({ redcPath: '', projectPath: '', logPath: '' }), terraformMirror = $bindable({ enabled: false, configPath: '', managed: false, fromEnv: false, providers: [] }), debugEnabled = $bindable(false), notificationEnabled = $bindable(false), spotMonitorEnabled = $bindable(false), spotAutoRecoverEnabled = $bindable(false) } = $props();
  let proxyForm = $state({ httpProxy: '', httpsProxy: '', socks5Proxy: '', noProxy: '' });
  let proxySaving = $state(false);
  let proxySaved = $state(false);
  let terraformMirrorForm = $state({ enabled: false, configPath: '', setEnv: false, providers: { aliyun: true, tencent: false, volc: false } });
  let terraformMirrorSaving = $state(false);
  let terraformMirrorSaved = $state(false);
  let terraformMirrorError = $state('');
  let networkChecks = $state([]);
  let networkCheckLoading = $state(false);
  let networkCheckError = $state('');
  let debugSaving = $state(false);
  let notificationSaving = $state(false);
  let spotMonitorSaving = $state(false);
  let spotAutoRecoverSaving = $state(false);
  let webhookForm = $state({ enabled: false, slack: '', dingtalk: '', dingtalkSecret: '', feishu: '', feishuSecret: '', discord: '', wecom: '' });
  let webhookSaving = $state(false);
  let webhookMessage = $state('');
  let webhookMessageType = $state('');
  let webhookTesting = $state({ slack: false, dingtalk: false, feishu: false, discord: false, wecom: false });
  let webhookLoaded = $state(false);
  
  // HTTP Server state
  let httpForm = $state({ enabled: false, port: 8899, host: '127.0.0.1', token: '' });
  let httpStatus = $state({ running: false, url: '', token: '' });
  let httpSaving = $state(false);
  let httpMessage = $state('');
  let httpMessageType = $state('');
  let httpLoaded = $state(false);
  
  async function loadHTTPServerConfig() {
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
  
  async function handleStartHTTPServer() {
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
  
  async function handleStopHTTPServer() {
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
  
  async function handleSaveHTTPConfig() {
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
  
  $effect(() => {
    loadHTTPServerConfig();
  });
  
  // Initialize forms when props change
  $effect(() => {
    proxyForm = {
      httpProxy: config.httpProxy || '',
      httpsProxy: config.httpsProxy || '',
      socks5Proxy: config.socks5Proxy || '',
      noProxy: config.noProxy || ''
    };
  });

  $effect(() => {
    if (terraformMirror) {
      terraformMirrorForm.enabled = !!terraformMirror.enabled;
      terraformMirrorForm.configPath = terraformMirror.configPath || '';
      terraformMirrorForm.setEnv = !!terraformMirror.fromEnv;
      terraformMirrorForm.providers.aliyun = terraformMirror.providers?.includes('aliyun');
      terraformMirrorForm.providers.tencent = terraformMirror.providers?.includes('tencent');
      terraformMirrorForm.providers.volc = terraformMirror.providers?.includes('volc');
    }
  });

  async function handleSaveProxy() {
    proxySaving = true;
    try {
      await SaveProxyConfig(proxyForm.httpProxy, proxyForm.httpsProxy, proxyForm.socks5Proxy, proxyForm.noProxy);
      config.httpProxy = proxyForm.httpProxy;
      config.httpsProxy = proxyForm.httpsProxy;
      config.socks5Proxy = proxyForm.socks5Proxy;
      config.noProxy = proxyForm.noProxy;
      proxySaved = true;
      setTimeout(() => { proxySaved = false; }, 1500);
    } catch (e) {
      console.error('Failed to save proxy:', e);
    } finally {
      proxySaving = false;
    }
  }
  
  async function handleSaveTerraformMirror() {
    terraformMirrorSaving = true;
    terraformMirrorError = '';
    try {
      const providers = Object.entries(terraformMirrorForm.providers)
        .filter(([, enabled]) => enabled)
        .map(([key]) => key);
      await SaveTerraformMirrorConfig(
        terraformMirrorForm.enabled,
        providers,
        terraformMirrorForm.configPath,
        terraformMirrorForm.setEnv
      );
      terraformMirror = await GetTerraformMirrorConfig();
      terraformMirrorForm.enabled = !!terraformMirror.enabled;
      terraformMirrorForm.configPath = terraformMirror.configPath || '';
      terraformMirrorForm.setEnv = !!terraformMirror.fromEnv;
      terraformMirrorForm.providers.aliyun = terraformMirror.providers?.includes('aliyun');
      terraformMirrorForm.providers.tencent = terraformMirror.providers?.includes('tencent');
      terraformMirrorForm.providers.volc = terraformMirror.providers?.includes('volc');
      terraformMirrorForm.providers.wgpsec = terraformMirror.providers?.includes('wgpsec');
      terraformMirrorSaved = true;
      setTimeout(() => { terraformMirrorSaved = false; }, 1500);
    } catch (e) {
      terraformMirrorError = e.message || String(e);
    } finally {
      terraformMirrorSaving = false;
    }
  }
  
  async function enableAliyunMirrorQuick() {
    terraformMirrorForm = {
      ...terraformMirrorForm,
      enabled: true,
      setEnv: true,
      providers: { ...terraformMirrorForm.providers, aliyun: true }
    };
    await handleSaveTerraformMirror();
  }
  
  async function enableTencentMirrorQuick() {
    terraformMirrorForm = {
      ...terraformMirrorForm,
      enabled: true,
      setEnv: true,
      providers: { ...terraformMirrorForm.providers, tencent: true }
    };
    await handleSaveTerraformMirror();
  }
  
  async function enableVolcMirrorQuick() {
    terraformMirrorForm = {
      ...terraformMirrorForm,
      enabled: true,
      setEnv: true,
      providers: { ...terraformMirrorForm.providers, volc: true }
    };
    await handleSaveTerraformMirror();
  }

  async function runTerraformNetworkCheck() {
    networkCheckLoading = true;
    networkCheckError = '';
    try {
      networkChecks = await TestTerraformEndpoints();
    } catch (e) {
      networkCheckError = e.message || String(e);
    } finally {
      networkCheckLoading = false;
    }
  }
  
  async function handleToggleDebug() {
    const nextValue = !debugEnabled;
    debugSaving = true;
    try {
      await SetDebugLogging(nextValue);
      debugEnabled = nextValue;
      config.debugEnabled = nextValue;
    } catch (e) {
      console.error('Failed to toggle debug:', e);
    } finally {
      debugSaving = false;
    }
  }
  
  async function handleToggleNotification() {
    const nextValue = !notificationEnabled;
    notificationSaving = true;
    try {
      await SetNotificationEnabled(nextValue);
      notificationEnabled = nextValue;
    } catch (e) {
      console.error('Failed to toggle notification:', e);
    } finally {
      notificationSaving = false;
    }
  }

  async function handleToggleSpotMonitor() {
    const nextValue = !spotMonitorEnabled;
    spotMonitorSaving = true;
    try {
      await SetSpotMonitorEnabled(nextValue);
      spotMonitorEnabled = nextValue;
    } catch (e) {
      console.error('Failed to toggle spot monitor:', e);
    } finally {
      spotMonitorSaving = false;
    }
  }

  async function handleToggleSpotAutoRecover() {
    const nextValue = !spotAutoRecoverEnabled;
    spotAutoRecoverSaving = true;
    try {
      await SetSpotAutoRecoverEnabled(nextValue);
      spotAutoRecoverEnabled = nextValue;
    } catch (e) {
      console.error('Failed to toggle spot auto recover:', e);
    } finally {
      spotAutoRecoverSaving = false;
    }
  }
  
  async function handleToggleTerraformMirror() {
    const nextValue = !terraformMirrorForm.enabled;
    terraformMirrorSaving = true;
    terraformMirrorError = '';
    try {
      const providers = Object.entries(terraformMirrorForm.providers)
        .filter(([, enabled]) => enabled)
        .map(([key]) => key);
      await SaveTerraformMirrorConfig(
        nextValue,
        providers,
        terraformMirrorForm.configPath,
        terraformMirrorForm.setEnv
      );
      terraformMirror = await GetTerraformMirrorConfig();
      terraformMirrorForm.enabled = !!terraformMirror.enabled;
      terraformMirrorForm.configPath = terraformMirror.configPath || '';
      terraformMirrorForm.setEnv = !!terraformMirror.fromEnv;
      terraformMirrorForm.providers.aliyun = terraformMirror.providers?.includes('aliyun');
      terraformMirrorForm.providers.tencent = terraformMirror.providers?.includes('tencent');
      terraformMirrorForm.providers.volc = terraformMirror.providers?.includes('volc');
      terraformMirrorForm.providers.wgpsec = terraformMirror.providers?.includes('wgpsec');
    } catch (e) {
      terraformMirrorError = e.message || String(e);
    } finally {
      terraformMirrorSaving = false;
    }
  }

  async function loadWebhookConfig() {
    if (webhookLoaded) return;
    try {
      const cfg = await GetWebhookConfig();
      webhookForm = {
        enabled: cfg.enabled || false,
        slack: cfg.slack || '',
        dingtalk: cfg.dingtalk || '',
        dingtalkSecret: cfg.dingtalkSecret || '',
        feishu: cfg.feishu || '',
        feishuSecret: cfg.feishuSecret || '',
        discord: cfg.discord || '',
        wecom: cfg.wecom || ''
      };
      webhookLoaded = true;
    } catch (e) {
      console.error('Failed to load webhook config:', e);
    }
  }

  async function handleSaveWebhook() {
    webhookSaving = true;
    webhookMessage = '';
    try {
      await SetWebhookConfig(webhookForm);
      webhookMessage = t.webhookSaveSuccess || 'Webhook config saved';
      webhookMessageType = 'success';
    } catch (e) {
      webhookMessage = (t.webhookSaveFailed || 'Failed to save') + ': ' + (e.message || String(e));
      webhookMessageType = 'error';
    } finally {
      webhookSaving = false;
      setTimeout(() => { webhookMessage = ''; }, 3000);
    }
  }

  async function handleTestWebhook(platform) {
    webhookTesting = { ...webhookTesting, [platform]: true };
    webhookMessage = '';
    try {
      let url = '', secret = '';
      switch (platform) {
        case 'slack': url = webhookForm.slack; break;
        case 'dingtalk': url = webhookForm.dingtalk; secret = webhookForm.dingtalkSecret; break;
        case 'feishu': url = webhookForm.feishu; secret = webhookForm.feishuSecret; break;
        case 'discord': url = webhookForm.discord; break;
        case 'wecom': url = webhookForm.wecom; break;
      }
      if (!url) {
        webhookMessage = 'URL is empty';
        webhookMessageType = 'error';
        setTimeout(() => { webhookMessage = ''; }, 3000);
        return;
      }
      await TestWebhook(platform, url, secret);
      webhookMessage = t.webhookTestSuccess || 'Test message sent';
      webhookMessageType = 'success';
    } catch (e) {
      webhookMessage = (t.webhookTestFailed || 'Test failed') + ': ' + (e.message || String(e));
      webhookMessageType = 'error';
    } finally {
      webhookTesting = { ...webhookTesting, [platform]: false };
      setTimeout(() => { webhookMessage = ''; }, 4000);
    }
  }

  $effect(() => { loadWebhookConfig(); });
</script>

<div class="w-full max-w-2xl mx-auto space-y-4">
  <div class="bg-white rounded-xl border border-gray-100 p-4 sm:p-5">
    <div class="text-[13px] sm:text-[14px] font-medium text-gray-900 mb-3">{t.redcPath?.replace('路径', '') || '基本信息'}</div>
    <div class="space-y-2">
      <div class="flex items-start gap-3">
        <span class="text-[11px] sm:text-[12px] text-gray-500 w-20 flex-shrink-0 pt-0.5">{t.redcPath}</span>
        <span class="text-[12px] sm:text-[13px] text-gray-900 font-mono break-all">{config.redcPath || '-'}</span>
      </div>
      <div class="flex items-start gap-3">
        <span class="text-[11px] sm:text-[12px] text-gray-500 w-20 flex-shrink-0 pt-0.5">{t.projectPath}</span>
        <span class="text-[12px] sm:text-[13px] text-gray-900 font-mono break-all">{config.projectPath || '-'}</span>
      </div>
      <div class="flex items-start gap-3">
        <span class="text-[11px] sm:text-[12px] text-gray-500 w-20 flex-shrink-0 pt-0.5">{t.logPath}</span>
        <span class="text-[12px] sm:text-[13px] text-gray-900 font-mono break-all">{config.logPath || '-'}</span>
      </div>
    </div>
  </div>

  <!-- 代理配置 -->
  <div class="bg-white rounded-xl border border-gray-100 p-4 sm:p-5">
    <div class="text-[13px] sm:text-[14px] font-medium text-gray-900 mb-4">{t.proxyConfig}</div>
    <div class="grid grid-cols-1 sm:grid-cols-2 gap-3 sm:gap-4">
      <div>
        <label for="httpProxy" class="block text-[11px] sm:text-[12px] font-medium text-gray-500 mb-1.5">{t.httpProxy}</label>
        <input 
          id="httpProxy"
          type="text" 
          placeholder="http://127.0.0.1:7890" 
          class="w-full h-9 px-3 text-[12px] sm:text-[13px] bg-gray-50 border-0 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow font-mono"
          bind:value={proxyForm.httpProxy} 
        />
      </div>
      <div>
        <label for="httpsProxy" class="block text-[11px] sm:text-[12px] font-medium text-gray-500 mb-1.5">{t.httpsProxy}</label>
        <input 
          id="httpsProxy"
          type="text" 
          placeholder="http://127.0.0.1:7890" 
          class="w-full h-9 px-3 text-[12px] sm:text-[13px] bg-gray-50 border-0 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow font-mono"
          bind:value={proxyForm.httpsProxy} 
        />
      </div>
      <div>
        <label for="socks5Proxy" class="block text-[11px] sm:text-[12px] font-medium text-gray-500 mb-1.5">{t.socks5Proxy}</label>
        <input 
          id="socks5Proxy"
          type="text" 
          placeholder="socks5://127.0.0.1:1080" 
          class="w-full h-9 px-3 text-[12px] sm:text-[13px] bg-gray-50 border-0 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow font-mono"
          bind:value={proxyForm.socks5Proxy} 
        />
      </div>
      <div>
        <label for="noProxy" class="block text-[11px] sm:text-[12px] font-medium text-gray-500 mb-1.5">{t.noProxyLabel}</label>
        <input 
          id="noProxy"
          type="text" 
          placeholder="localhost,127.0.0.1,.local" 
          class="w-full h-9 px-3 text-[12px] sm:text-[13px] bg-gray-50 border-0 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow font-mono"
          bind:value={proxyForm.noProxy} 
        />
      </div>
    </div>
    <div class="mt-4 flex items-center gap-3">
      <button 
        class="h-9 px-4 bg-emerald-500 text-white text-[12px] font-medium rounded-lg hover:bg-emerald-600 transition-colors disabled:opacity-50 disabled:cursor-not-allowed cursor-pointer inline-flex items-center gap-1.5"
        onclick={handleSaveProxy}
        disabled={proxySaving || proxySaved}
      >
        {#if proxySaving}
          <svg class="animate-spin h-3.5 w-3.5" viewBox="0 0 24 24" fill="none"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path></svg>
          {t.saving}
        {:else if proxySaved}
          <svg class="h-3.5 w-3.5" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd" /></svg>
          {t.saved || '已保存'}
        {:else}
          {t.saveProxy}
        {/if}
      </button>
      <span class="text-[11px] sm:text-[12px] text-gray-500">{t.proxyHint}</span>
    </div>
  </div>

  <!-- Terraform 镜像加速 -->
  <div class="bg-white rounded-xl border border-gray-100 p-4 sm:p-5">
    <div class="flex flex-col sm:flex-row sm:items-start justify-between gap-3 sm:gap-0 mb-4">
      <div class="flex-1">
        <div class="text-[13px] sm:text-[14px] font-medium text-gray-900">{t.terraformMirror}</div>
        <div class="text-[11px] sm:text-[12px] text-gray-500 mt-1">{t.mirrorConfigHint}</div>
      </div>
      <button
        class="relative inline-flex h-6 w-11 flex-shrink-0 items-center rounded-full transition-colors disabled:opacity-50 disabled:cursor-not-allowed cursor-pointer"
        class:bg-emerald-500={terraformMirrorForm.enabled}
        class:bg-gray-300={!terraformMirrorForm.enabled}
        onclick={handleToggleTerraformMirror}
        disabled={terraformMirrorSaving}
        aria-label={t.mirrorEnabled}
      >
        <span
          class="inline-block h-4 w-4 transform rounded-full bg-white transition-transform"
          class:translate-x-6={terraformMirrorForm.enabled}
          class:translate-x-1={!terraformMirrorForm.enabled}
        ></span>
      </button>
    </div>
    <div class="mb-4 text-[10px] sm:text-[11px] text-blue-600 flex items-start gap-1.5 bg-blue-50 p-2.5 rounded-lg">
      <svg class="w-3.5 h-3.5 mt-0.5 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
        <path fill-rule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z" clip-rule="evenodd" />
      </svg>
      <span>{t.mirrorCachePriorityHint}</span>
    </div>
    <div class="space-y-3 sm:space-y-4">
      <div>
        <span id="mirrorProvidersLabel" class="block text-[11px] sm:text-[12px] font-medium text-gray-500 mb-1.5">{t.mirrorProviders}</span>
        <div class="flex flex-wrap items-center gap-2 sm:gap-3 text-[11px] sm:text-[12px] text-gray-700" role="group" aria-labelledby="mirrorProvidersLabel">
          <label class="inline-flex items-center gap-2">
            <input type="checkbox" class="rounded" bind:checked={terraformMirrorForm.providers.aliyun} />
            <span>{t.mirrorAliyun}</span>
          </label>
          <label class="inline-flex items-center gap-2">
            <input type="checkbox" class="rounded" bind:checked={terraformMirrorForm.providers.tencent} />
            <span>{t.mirrorTencent}</span>
          </label>
          <label class="inline-flex items-center gap-2">
            <input type="checkbox" class="rounded" bind:checked={terraformMirrorForm.providers.volc} />
            <span>{t.mirrorVolc}</span>
          </label>
        </div>
        <div class="mt-2 text-[10px] sm:text-[11px] text-gray-500">
          {t.mirrorProvidersDesc}
        </div>
      </div>
      <div>
        <label for="mirrorConfigPath" class="block text-[11px] sm:text-[12px] font-medium text-gray-500 mb-1.5">{t.mirrorConfigPath}</label>
        <input
          id="mirrorConfigPath"
          type="text"
          placeholder={terraformMirror.configPath || t.mirrorConfigHint}
          class="w-full h-9 sm:h-10 px-3 text-[12px] sm:text-[13px] bg-gray-50 border-0 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow font-mono"
          bind:value={terraformMirrorForm.configPath}
        />
        {#if terraformMirror.fromEnv}
          <div class="mt-1 text-[10px] sm:text-[11px] text-amber-600">{t.mirrorConfigFromEnv}</div>
        {/if}
      </div>
      <div class="flex items-center gap-2 text-[11px] sm:text-[12px] text-gray-600">
        <input type="checkbox" class="rounded" bind:checked={terraformMirrorForm.setEnv} />
        <span>{t.mirrorSetEnv}</span>
      </div>
      <div class="pt-1 flex flex-wrap gap-2 items-center">
        <button
          class="h-8 sm:h-9 px-3 sm:px-4 bg-emerald-500 text-white text-[11px] sm:text-[12px] font-medium rounded-lg hover:bg-emerald-600 transition-colors disabled:opacity-50 cursor-pointer inline-flex items-center gap-1.5"
          onclick={handleSaveTerraformMirror}
          disabled={terraformMirrorSaving || terraformMirrorSaved}
        >
          {#if terraformMirrorSaving}
            <svg class="animate-spin h-3.5 w-3.5" viewBox="0 0 24 24" fill="none"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path></svg>
            {t.saving}
          {:else if terraformMirrorSaved}
            <svg class="h-3.5 w-3.5" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd" /></svg>
            {t.saved || '已保存'}
          {:else}
            {t.mirrorSave}
          {/if}
        </button>
        <button
          class="h-8 sm:h-9 px-3 sm:px-4 text-gray-700 bg-white border border-gray-300 hover:bg-gray-50 text-[11px] sm:text-[12px] font-medium rounded-lg transition-colors cursor-pointer"
          onclick={enableAliyunMirrorQuick}
        >
          {t.mirrorAliyunPreset}
        </button>
        <button
          class="h-8 sm:h-9 px-3 sm:px-4 text-gray-700 bg-white border border-gray-300 hover:bg-gray-50 text-[11px] sm:text-[12px] font-medium rounded-lg transition-colors cursor-pointer"
          onclick={enableTencentMirrorQuick}
        >
          {t.mirrorTencentPreset}
        </button>
        <button
          class="h-8 sm:h-9 px-3 sm:px-4 text-gray-700 bg-white border border-gray-300 hover:bg-gray-50 text-[11px] sm:text-[12px] font-medium rounded-lg transition-colors cursor-pointer"
          onclick={enableVolcMirrorQuick}
        >
          {t.mirrorVolcPreset}
        </button>
        {#if terraformMirrorError}
          <span class="text-[11px] sm:text-[12px] text-red-500">{terraformMirrorError}</span>
        {:else if terraformMirror.managed}
          <span class="text-[11px] sm:text-[12px] text-emerald-600">OK</span>
        {/if}
      </div>
      <div class="mt-2 text-[10px] sm:text-[11px] text-gray-500 leading-relaxed">
        <span class="font-medium text-gray-600">{t.mirrorLimitTitle}</span>
        <span class="ml-1">{t.mirrorLimitDesc}</span>
      </div>
    </div>
  </div>

  <!-- 网络诊断 -->
  <div class="bg-white rounded-xl border border-gray-100 p-4 sm:p-5">
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 sm:gap-0">
      <div class="text-[13px] sm:text-[14px] font-medium text-gray-900">{t.networkCheck}</div>
      <button
        class="h-8 sm:h-9 px-3 sm:px-4 text-gray-700 bg-white border border-gray-300 hover:bg-gray-50 text-[11px] sm:text-[12px] font-medium rounded-lg transition-colors disabled:opacity-50 cursor-pointer"
        onclick={runTerraformNetworkCheck}
        disabled={networkCheckLoading}
      >
        {networkCheckLoading ? t.networkChecking : t.networkCheckBtn}
      </button>
    </div>
    {#if networkCheckError}
      <div class="mt-3 text-[11px] sm:text-[12px] text-red-500">{networkCheckError}</div>
    {/if}
    {#if networkChecks.length > 0}
      <div class="mt-4 border border-gray-100 rounded-lg overflow-hidden">
        <table class="w-full text-[11px] sm:text-[12px]">
          <thead>
            <tr class="bg-gray-50 border-b border-gray-100">
              <th class="text-left px-3 sm:px-4 py-2 sm:py-2.5 font-semibold text-gray-600">{t.networkEndpoint}</th>
              <th class="text-right px-3 sm:px-4 py-2 sm:py-2.5 font-semibold text-gray-600">{t.networkStatus}</th>
              <th class="text-right px-3 sm:px-4 py-2 sm:py-2.5 font-semibold text-gray-600">{t.networkLatency}</th>
              <th class="text-left px-3 sm:px-4 py-2 sm:py-2.5 font-semibold text-gray-600">{t.networkError}</th>
            </tr>
          </thead>
          <tbody>
            {#each networkChecks as item}
              <tr class="border-b border-gray-50">
                <td class="px-3 sm:px-4 py-2.5 sm:py-3 text-gray-700">{item.name}</td>
                <td class="px-3 sm:px-4 py-2.5 sm:py-3 text-right {item.ok ? 'text-emerald-600' : 'text-red-600'}">{item.ok ? 'OK' : item.status || '-'}</td>
                <td class="px-3 sm:px-4 py-2.5 sm:py-3 text-right text-gray-700">{item.latencyMs} ms</td>
                <td class="px-3 sm:px-4 py-2.5 sm:py-3 text-gray-500 truncate" title={item.error}>{item.error || '-'}</td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/if}
  </div>

  <!-- 通用设置（调试/通知/右键） -->
  <div class="bg-white rounded-xl border border-gray-100 divide-y divide-gray-100">
    <!-- 调试日志 -->
    <div class="flex items-center justify-between px-4 sm:px-5 py-3.5">
      <div>
        <div class="text-[13px] sm:text-[14px] font-medium text-gray-900">{t.debugLogs}</div>
        <div class="text-[11px] sm:text-[12px] text-gray-500 mt-0.5">{t.debugLogsDesc}</div>
      </div>
      <button
        class="relative inline-flex h-6 w-11 flex-shrink-0 items-center rounded-full transition-colors disabled:opacity-50 disabled:cursor-not-allowed cursor-pointer"
        class:bg-emerald-500={debugEnabled}
        class:bg-gray-300={!debugEnabled}
        onclick={handleToggleDebug}
        disabled={debugSaving}
        aria-label={debugEnabled ? t.disable : t.enable}
      >
        <span class="inline-block h-4 w-4 transform rounded-full bg-white transition-transform"
          class:translate-x-6={debugEnabled} class:translate-x-1={!debugEnabled}></span>
      </button>
    </div>
    <!-- 系统通知 -->
    <div class="flex items-center justify-between px-4 sm:px-5 py-3.5">
      <div>
        <div class="text-[13px] sm:text-[14px] font-medium text-gray-900">{t.systemNotification}</div>
        <div class="text-[11px] sm:text-[12px] text-gray-500 mt-0.5">{t.systemNotificationDesc}</div>
      </div>
      <button
        class="relative inline-flex h-6 w-11 flex-shrink-0 items-center rounded-full transition-colors disabled:opacity-50 disabled:cursor-not-allowed cursor-pointer"
        class:bg-emerald-500={notificationEnabled}
        class:bg-gray-300={!notificationEnabled}
        onclick={handleToggleNotification}
        disabled={notificationSaving}
        aria-label={notificationEnabled ? t.disable : t.enable}
      >
        <span class="inline-block h-4 w-4 transform rounded-full bg-white transition-transform"
          class:translate-x-6={notificationEnabled} class:translate-x-1={!notificationEnabled}></span>
      </button>
    </div>
    <!-- Webhook 通知 -->
    <div class="px-4 sm:px-5 py-3.5">
      <div class="flex items-center justify-between">
        <div>
          <div class="text-[13px] sm:text-[14px] font-medium text-gray-900">{t.webhookNotification || 'Webhook 通知'}</div>
          <div class="text-[11px] sm:text-[12px] text-gray-500 mt-0.5">{t.webhookNotificationDesc || '场景状态变化时推送消息'}</div>
        </div>
        <button
          class="relative inline-flex h-6 w-11 flex-shrink-0 items-center rounded-full transition-colors cursor-pointer"
          class:bg-emerald-500={webhookForm.enabled}
          class:bg-gray-300={!webhookForm.enabled}
          onclick={() => { webhookForm.enabled = !webhookForm.enabled; handleSaveWebhook(); }}
        >
          <span class="inline-block h-4 w-4 transform rounded-full bg-white transition-transform"
            class:translate-x-6={webhookForm.enabled} class:translate-x-1={!webhookForm.enabled}></span>
        </button>
      </div>
      {#if webhookForm.enabled}
      <div class="mt-3 space-y-3 ml-1 border-l-2 border-gray-200 pl-3">
        <!-- Slack -->
        <div>
          <label class="text-[11px] text-gray-500 mb-1 block">{t.webhookSlack || 'Slack Webhook URL'}</label>
          <div class="flex gap-2">
            <input type="text" bind:value={webhookForm.slack} placeholder={t.webhookUrlHint || '留空则不推送'}
              class="flex-1 text-[12px] px-3 py-1.5 border border-gray-200 rounded-lg focus:outline-none focus:ring-1 focus:ring-red-400" />
            <button onclick={() => handleTestWebhook('slack')} disabled={webhookTesting.slack || !webhookForm.slack}
              class="text-[11px] px-3 py-1.5 bg-gray-100 hover:bg-gray-200 rounded-lg disabled:opacity-40 cursor-pointer disabled:cursor-not-allowed">
              {webhookTesting.slack ? '...' : (t.webhookTest || '测试')}
            </button>
          </div>
        </div>
        <!-- 钉钉 -->
        <div>
          <label class="text-[11px] text-gray-500 mb-1 block">{t.webhookDingtalk || '钉钉 Webhook URL'}</label>
          <div class="flex gap-2">
            <input type="text" bind:value={webhookForm.dingtalk} placeholder={t.webhookUrlHint || '留空则不推送'}
              class="flex-1 text-[12px] px-3 py-1.5 border border-gray-200 rounded-lg focus:outline-none focus:ring-1 focus:ring-red-400" />
            <button onclick={() => handleTestWebhook('dingtalk')} disabled={webhookTesting.dingtalk || !webhookForm.dingtalk}
              class="text-[11px] px-3 py-1.5 bg-gray-100 hover:bg-gray-200 rounded-lg disabled:opacity-40 cursor-pointer disabled:cursor-not-allowed">
              {webhookTesting.dingtalk ? '...' : (t.webhookTest || '测试')}
            </button>
          </div>
          <input type="text" bind:value={webhookForm.dingtalkSecret} placeholder={t.webhookSecretHint || '签名密钥（可选）'}
            class="mt-1 w-full text-[12px] px-3 py-1.5 border border-gray-200 rounded-lg focus:outline-none focus:ring-1 focus:ring-red-400" />
        </div>
        <!-- 飞书 -->
        <div>
          <label class="text-[11px] text-gray-500 mb-1 block">{t.webhookFeishu || '飞书 Webhook URL'}</label>
          <div class="flex gap-2">
            <input type="text" bind:value={webhookForm.feishu} placeholder={t.webhookUrlHint || '留空则不推送'}
              class="flex-1 text-[12px] px-3 py-1.5 border border-gray-200 rounded-lg focus:outline-none focus:ring-1 focus:ring-red-400" />
            <button onclick={() => handleTestWebhook('feishu')} disabled={webhookTesting.feishu || !webhookForm.feishu}
              class="text-[11px] px-3 py-1.5 bg-gray-100 hover:bg-gray-200 rounded-lg disabled:opacity-40 cursor-pointer disabled:cursor-not-allowed">
              {webhookTesting.feishu ? '...' : (t.webhookTest || '测试')}
            </button>
          </div>
          <input type="text" bind:value={webhookForm.feishuSecret} placeholder={t.webhookSecretHint || '签名密钥（可选）'}
            class="mt-1 w-full text-[12px] px-3 py-1.5 border border-gray-200 rounded-lg focus:outline-none focus:ring-1 focus:ring-red-400" />
        </div>
        <!-- Discord -->
        <div>
          <label class="text-[11px] text-gray-500 mb-1 block">{t.webhookDiscord || 'Discord Webhook URL'}</label>
          <div class="flex gap-2">
            <input type="text" bind:value={webhookForm.discord} placeholder={t.webhookUrlHint || '留空则不推送'}
              class="flex-1 text-[12px] px-3 py-1.5 border border-gray-200 rounded-lg focus:outline-none focus:ring-1 focus:ring-red-400" />
            <button onclick={() => handleTestWebhook('discord')} disabled={webhookTesting.discord || !webhookForm.discord}
              class="text-[11px] px-3 py-1.5 bg-gray-100 hover:bg-gray-200 rounded-lg disabled:opacity-40 cursor-pointer disabled:cursor-not-allowed">
              {webhookTesting.discord ? '...' : (t.webhookTest || '测试')}
            </button>
          </div>
        </div>
        <!-- 企业微信 -->
        <div>
          <label class="text-[11px] text-gray-500 mb-1 block">{t.webhookWecom || '企业微信 Webhook URL'}</label>
          <div class="flex gap-2">
            <input type="text" bind:value={webhookForm.wecom} placeholder={t.webhookUrlHint || '留空则不推送'}
              class="flex-1 text-[12px] px-3 py-1.5 border border-gray-200 rounded-lg focus:outline-none focus:ring-1 focus:ring-red-400" />
            <button onclick={() => handleTestWebhook('wecom')} disabled={webhookTesting.wecom || !webhookForm.wecom}
              class="text-[11px] px-3 py-1.5 bg-gray-100 hover:bg-gray-200 rounded-lg disabled:opacity-40 cursor-pointer disabled:cursor-not-allowed">
              {webhookTesting.wecom ? '...' : (t.webhookTest || '测试')}
            </button>
          </div>
        </div>
        <!-- 保存按钮 + 状态信息 -->
        <div class="flex items-center gap-3 pt-1">
          <button onclick={handleSaveWebhook} disabled={webhookSaving}
            class="text-[12px] px-4 py-1.5 bg-red-500 hover:bg-red-600 text-white rounded-lg disabled:opacity-50 cursor-pointer disabled:cursor-not-allowed">
            {webhookSaving ? '...' : (t.webhookSave || '保存')}
          </button>
          {#if webhookMessage}
            <span class="text-[11px]" class:text-emerald-600={webhookMessageType === 'success'} class:text-red-500={webhookMessageType === 'error'}>
              {webhookMessage}
            </span>
          {/if}
        </div>
      </div>
      {/if}
    </div>
    <!-- Spot 实例监控 -->
    <div class="flex items-center justify-between px-4 sm:px-5 py-3.5">
      <div>
        <div class="text-[13px] sm:text-[14px] font-medium text-gray-900">{t.spotMonitor || 'Spot 实例监控'}</div>
        <div class="text-[11px] sm:text-[12px] text-gray-500 mt-0.5">{t.spotMonitorDesc || '定期检测运行中的抢占式实例是否被云厂商回收'}</div>
      </div>
      <button
        class="relative inline-flex h-6 w-11 flex-shrink-0 items-center rounded-full transition-colors disabled:opacity-50 disabled:cursor-not-allowed cursor-pointer"
        class:bg-emerald-500={spotMonitorEnabled}
        class:bg-gray-300={!spotMonitorEnabled}
        onclick={handleToggleSpotMonitor}
        disabled={spotMonitorSaving}
        aria-label={spotMonitorEnabled ? t.disable : t.enable}
      >
        <span class="inline-block h-4 w-4 transform rounded-full bg-white transition-transform"
          class:translate-x-6={spotMonitorEnabled} class:translate-x-1={!spotMonitorEnabled}></span>
      </button>
    </div>
    <!-- Spot 自动恢复（仅在监控开启时显示） -->
    {#if spotMonitorEnabled}
    <div class="flex items-center justify-between px-4 sm:px-5 py-3.5 ml-4 border-l-2 border-gray-200">
      <div>
        <div class="text-[13px] sm:text-[14px] font-medium text-gray-900">{t.spotAutoRecover || 'Spot 自动恢复'}</div>
        <div class="text-[11px] sm:text-[12px] text-gray-500 mt-0.5">{t.spotAutoRecoverDesc || '检测到实例被回收时自动执行 terraform apply 补齐'}</div>
      </div>
      <button
        class="relative inline-flex h-6 w-11 flex-shrink-0 items-center rounded-full transition-colors disabled:opacity-50 disabled:cursor-not-allowed cursor-pointer"
        class:bg-emerald-500={spotAutoRecoverEnabled}
        class:bg-gray-300={!spotAutoRecoverEnabled}
        onclick={handleToggleSpotAutoRecover}
        disabled={spotAutoRecoverSaving}
        aria-label={spotAutoRecoverEnabled ? t.disable : t.enable}
      >
        <span class="inline-block h-4 w-4 transform rounded-full bg-white transition-transform"
          class:translate-x-6={spotAutoRecoverEnabled} class:translate-x-1={!spotAutoRecoverEnabled}></span>
      </button>
    </div>
    {/if}
  </div>

  <!-- HTTP Server Section -->
  <div class="bg-white rounded-xl border border-gray-100 overflow-hidden">
    <div class="px-4 sm:px-5 py-3 border-b border-gray-100 bg-gray-50/50">
      <h3 class="text-[13px] sm:text-[14px] font-semibold text-gray-700">{t.httpServer || 'HTTP Server'}</h3>
      <p class="text-[11px] sm:text-[12px] text-gray-500 mt-0.5">{t.httpServerDesc || 'Access RedC GUI via browser'}</p>
    </div>
    
    <!-- Running status banner -->
    {#if httpStatus.running}
    <div class="px-4 sm:px-5 py-2.5 bg-emerald-50 border-b border-emerald-100 flex items-center justify-between gap-2 flex-wrap">
      <div class="flex items-center gap-2">
        <span class="inline-block w-2 h-2 rounded-full bg-emerald-500 animate-pulse"></span>
        <span class="text-[12px] text-emerald-700 font-medium">{t.httpServerRunning || 'Running'}</span>
        <span class="text-[12px] text-emerald-600">{httpStatus.url}</span>
      </div>
      <div class="flex items-center gap-1.5">
        <button class="text-[11px] px-2 py-0.5 rounded bg-emerald-100 hover:bg-emerald-200 text-emerald-700 cursor-pointer transition-colors" onclick={() => copyToClipboard(httpStatus.url)}>{t.httpServerCopyUrl || 'Copy URL'}</button>
        <button class="text-[11px] px-2 py-0.5 rounded bg-emerald-100 hover:bg-emerald-200 text-emerald-700 cursor-pointer transition-colors" onclick={() => copyToClipboard(httpStatus.token)}>{t.httpServerCopyToken || 'Copy Token'}</button>
      </div>
    </div>
    {/if}
    
    <!-- Config form -->
    <div class="px-4 sm:px-5 py-3 space-y-3">
      <div class="grid grid-cols-2 gap-3">
        <div>
          <label class="block text-[11px] text-gray-500 mb-1">{t.httpServerHost || 'Listen host'}</label>
          <input type="text" bind:value={httpForm.host} placeholder="127.0.0.1"
            class="w-full px-2.5 py-1.5 text-[12px] border border-gray-200 rounded-lg focus:outline-none focus:border-blue-400 bg-gray-50" />
        </div>
        <div>
          <label class="block text-[11px] text-gray-500 mb-1">{t.httpServerPort || 'Port'}</label>
          <input type="number" bind:value={httpForm.port} min="1024" max="65535" placeholder="8899"
            class="w-full px-2.5 py-1.5 text-[12px] border border-gray-200 rounded-lg focus:outline-none focus:border-blue-400 bg-gray-50" />
        </div>
      </div>
      <div>
        <label class="block text-[11px] text-gray-500 mb-1">{t.httpServerToken || 'Access Token'}</label>
        <input type="text" bind:value={httpForm.token} placeholder={t.httpServerTokenHint || 'Leave empty to auto-generate'}
          class="w-full px-2.5 py-1.5 text-[12px] border border-gray-200 rounded-lg focus:outline-none focus:border-blue-400 bg-gray-50 font-mono" />
      </div>
      
      <!-- Message -->
      {#if httpMessage}
      <p class="text-[12px] rounded px-2 py-1" class:text-emerald-600={httpMessageType === 'success'} class:bg-emerald-50={httpMessageType === 'success'} class:text-red-600={httpMessageType === 'error'} class:bg-red-50={httpMessageType === 'error'}>{httpMessage}</p>
      {/if}
      
      <!-- Action buttons -->
      <div class="flex gap-2 pt-0.5">
        <button onclick={handleSaveHTTPConfig} disabled={httpSaving}
          class="px-3 py-1.5 text-[12px] rounded-lg bg-gray-100 hover:bg-gray-200 text-gray-700 transition-colors cursor-pointer disabled:opacity-50">{t.httpServerSaveConfig || 'Save config'}</button>
        {#if !httpStatus.running}
          <button onclick={handleStartHTTPServer} disabled={httpSaving}
            class="px-3 py-1.5 text-[12px] rounded-lg bg-emerald-500 hover:bg-emerald-600 text-white transition-colors cursor-pointer disabled:opacity-50">{t.httpServerStart || 'Start'}</button>
        {:else}
          <button onclick={handleStopHTTPServer} disabled={httpSaving}
            class="px-3 py-1.5 text-[12px] rounded-lg bg-red-500 hover:bg-red-600 text-white transition-colors cursor-pointer disabled:opacity-50">{t.httpServerStop || 'Stop'}</button>
        {/if}
      </div>
    </div>
  </div>
</div>
