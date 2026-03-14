<script>
  import { SaveProxyConfig, SetDebugLogging, GetTerraformMirrorConfig, SaveTerraformMirrorConfig, TestTerraformEndpoints, SetNotificationEnabled, SetSpotMonitorEnabled, SetSpotAutoRecoverEnabled, GetWebhookConfig, SetWebhookConfig, TestWebhook } from '../../../wailsjs/go/main/App.js';

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

<div class="space-y-4">
    <!-- 基本信息 + 开关 -->
    <div class="bg-white rounded-xl border border-gray-100 overflow-hidden">
      <div class="px-5 py-3 border-b border-gray-100">
        <h3 class="text-[13px] font-semibold text-gray-900">{t.settingsGeneral || '通用设置'}</h3>
      </div>
      <!-- Path info -->
      <div class="px-5 py-3 border-b border-gray-50">
        <div class="space-y-2">
          <div class="flex items-start gap-3">
            <span class="text-[11px] text-gray-500 w-20 flex-shrink-0 pt-0.5">{t.redcPath}</span>
            <span class="text-[12px] text-gray-900 font-mono break-all">{config.redcPath || '-'}</span>
          </div>
          <div class="flex items-start gap-3">
            <span class="text-[11px] text-gray-500 w-20 flex-shrink-0 pt-0.5">{t.projectPath}</span>
            <span class="text-[12px] text-gray-900 font-mono break-all">{config.projectPath || '-'}</span>
          </div>
          <div class="flex items-start gap-3">
            <span class="text-[11px] text-gray-500 w-20 flex-shrink-0 pt-0.5">{t.logPath}</span>
            <span class="text-[12px] text-gray-900 font-mono break-all">{config.logPath || '-'}</span>
          </div>
        </div>
      </div>
      <!-- Toggle switches -->
      <div class="divide-y divide-gray-50">
        <!-- 调试日志 -->
        <div class="flex items-center justify-between px-5 py-3">
          <div>
            <div class="text-[13px] font-medium text-gray-900">{t.debugLogs}</div>
            <div class="text-[11px] text-gray-500 mt-0.5">{t.debugLogsDesc}</div>
          </div>
          <button
            class="relative inline-flex h-6 w-11 flex-shrink-0 items-center rounded-full transition-colors disabled:opacity-50 disabled:cursor-not-allowed cursor-pointer"
            class:bg-emerald-500={debugEnabled}
            class:bg-gray-300={!debugEnabled}
            onclick={handleToggleDebug}
            disabled={debugSaving}
          >
            <span class="inline-block h-4 w-4 transform rounded-full bg-white transition-transform"
              class:translate-x-6={debugEnabled} class:translate-x-1={!debugEnabled}></span>
          </button>
        </div>
        <!-- 系统通知 -->
        <div class="flex items-center justify-between px-5 py-3">
          <div>
            <div class="text-[13px] font-medium text-gray-900">{t.systemNotification}</div>
            <div class="text-[11px] text-gray-500 mt-0.5">{t.systemNotificationDesc}</div>
          </div>
          <button
            class="relative inline-flex h-6 w-11 flex-shrink-0 items-center rounded-full transition-colors disabled:opacity-50 disabled:cursor-not-allowed cursor-pointer"
            class:bg-emerald-500={notificationEnabled}
            class:bg-gray-300={!notificationEnabled}
            onclick={handleToggleNotification}
            disabled={notificationSaving}
          >
            <span class="inline-block h-4 w-4 transform rounded-full bg-white transition-transform"
              class:translate-x-6={notificationEnabled} class:translate-x-1={!notificationEnabled}></span>
          </button>
        </div>
        <!-- Spot 实例监控 -->
        <div class="flex items-center justify-between px-5 py-3">
          <div>
            <div class="text-[13px] font-medium text-gray-900">{t.spotMonitor || 'Spot 实例监控'}</div>
            <div class="text-[11px] text-gray-500 mt-0.5">{t.spotMonitorDesc || '定期检测运行中的抢占式实例是否被云厂商回收'}</div>
          </div>
          <button
            class="relative inline-flex h-6 w-11 flex-shrink-0 items-center rounded-full transition-colors disabled:opacity-50 disabled:cursor-not-allowed cursor-pointer"
            class:bg-emerald-500={spotMonitorEnabled}
            class:bg-gray-300={!spotMonitorEnabled}
            onclick={handleToggleSpotMonitor}
            disabled={spotMonitorSaving}
          >
            <span class="inline-block h-4 w-4 transform rounded-full bg-white transition-transform"
              class:translate-x-6={spotMonitorEnabled} class:translate-x-1={!spotMonitorEnabled}></span>
          </button>
        </div>
        <!-- Spot 自动恢复 -->
        {#if spotMonitorEnabled}
        <div class="flex items-center justify-between px-5 py-3 ml-4 border-l-2 border-gray-200">
          <div>
            <div class="text-[13px] font-medium text-gray-900">{t.spotAutoRecover || 'Spot 自动恢复'}</div>
            <div class="text-[11px] text-gray-500 mt-0.5">{t.spotAutoRecoverDesc || '检测到实例被回收时自动执行 terraform apply 补齐'}</div>
          </div>
          <button
            class="relative inline-flex h-6 w-11 flex-shrink-0 items-center rounded-full transition-colors disabled:opacity-50 disabled:cursor-not-allowed cursor-pointer"
            class:bg-emerald-500={spotAutoRecoverEnabled}
            class:bg-gray-300={!spotAutoRecoverEnabled}
            onclick={handleToggleSpotAutoRecover}
            disabled={spotAutoRecoverSaving}
          >
            <span class="inline-block h-4 w-4 transform rounded-full bg-white transition-transform"
              class:translate-x-6={spotAutoRecoverEnabled} class:translate-x-1={!spotAutoRecoverEnabled}></span>
          </button>
        </div>
        {/if}
      </div>
    </div>

    <!-- 代理配置 -->
    <div class="bg-white rounded-xl border border-gray-100 overflow-hidden">
      <div class="px-5 py-3 border-b border-gray-100">
        <h3 class="text-[13px] font-semibold text-gray-900">{t.proxyConfig}</h3>
      </div>
      <div class="px-5 py-4">
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
          <div>
            <label for="httpProxy" class="block text-[11px] font-medium text-gray-500 mb-1.5">{t.httpProxy}</label>
            <input id="httpProxy" type="text" placeholder="http://127.0.0.1:7890"
              class="w-full h-9 px-3 text-[12px] bg-gray-50 border-0 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow font-mono"
              bind:value={proxyForm.httpProxy} />
          </div>
          <div>
            <label for="httpsProxy" class="block text-[11px] font-medium text-gray-500 mb-1.5">{t.httpsProxy}</label>
            <input id="httpsProxy" type="text" placeholder="http://127.0.0.1:7890"
              class="w-full h-9 px-3 text-[12px] bg-gray-50 border-0 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow font-mono"
              bind:value={proxyForm.httpsProxy} />
          </div>
          <div>
            <label for="socks5Proxy" class="block text-[11px] font-medium text-gray-500 mb-1.5">{t.socks5Proxy}</label>
            <input id="socks5Proxy" type="text" placeholder="socks5://127.0.0.1:1080"
              class="w-full h-9 px-3 text-[12px] bg-gray-50 border-0 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow font-mono"
              bind:value={proxyForm.socks5Proxy} />
          </div>
          <div>
            <label for="noProxy" class="block text-[11px] font-medium text-gray-500 mb-1.5">{t.noProxyLabel}</label>
            <input id="noProxy" type="text" placeholder="localhost,127.0.0.1,.local"
              class="w-full h-9 px-3 text-[12px] bg-gray-50 border-0 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow font-mono"
              bind:value={proxyForm.noProxy} />
          </div>
        </div>
        <div class="mt-4 flex items-center gap-3">
          <button
            class="h-8 px-4 bg-gray-900 text-white text-[12px] font-medium rounded-lg hover:bg-gray-800 transition-colors disabled:opacity-50 disabled:cursor-not-allowed cursor-pointer inline-flex items-center gap-1.5"
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
          <span class="text-[11px] text-gray-500">{t.proxyHint}</span>
        </div>
      </div>
    </div>

    <!-- Terraform 镜像加速 -->
    <div class="bg-white rounded-xl border border-gray-100 overflow-hidden">
      <div class="px-5 py-3 border-b border-gray-100 flex items-center justify-between">
        <div>
          <h3 class="text-[13px] font-semibold text-gray-900">{t.terraformMirror}</h3>
          <p class="text-[11px] text-gray-500 mt-0.5">{t.mirrorConfigHint}</p>
        </div>
        <button
          class="relative inline-flex h-6 w-11 flex-shrink-0 items-center rounded-full transition-colors disabled:opacity-50 disabled:cursor-not-allowed cursor-pointer"
          class:bg-emerald-500={terraformMirrorForm.enabled}
          class:bg-gray-300={!terraformMirrorForm.enabled}
          onclick={handleToggleTerraformMirror}
          disabled={terraformMirrorSaving}
        >
          <span class="inline-block h-4 w-4 transform rounded-full bg-white transition-transform"
            class:translate-x-6={terraformMirrorForm.enabled} class:translate-x-1={!terraformMirrorForm.enabled}></span>
        </button>
      </div>
      <div class="px-5 py-4 space-y-3">
        <div class="text-[11px] text-gray-600 flex items-start gap-1.5 bg-gray-50 p-2.5 rounded-lg">
          <svg class="w-3.5 h-3.5 mt-0.5 flex-shrink-0 text-gray-400" fill="currentColor" viewBox="0 0 20 20">
            <path fill-rule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z" clip-rule="evenodd" />
          </svg>
          <span>{t.mirrorCachePriorityHint}</span>
        </div>
        <div>
          <span class="block text-[11px] font-medium text-gray-500 mb-1.5">{t.mirrorProviders}</span>
          <div class="flex flex-wrap items-center gap-3 text-[12px] text-gray-700">
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
          <div class="mt-1.5 text-[10px] text-gray-500">{t.mirrorProvidersDesc}</div>
        </div>
        <div>
          <label for="mirrorConfigPath" class="block text-[11px] font-medium text-gray-500 mb-1.5">{t.mirrorConfigPath}</label>
          <input id="mirrorConfigPath" type="text" placeholder={terraformMirror.configPath || t.mirrorConfigHint}
            class="w-full h-9 px-3 text-[12px] bg-gray-50 border-0 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow font-mono"
            bind:value={terraformMirrorForm.configPath} />
          {#if terraformMirror.fromEnv}
            <div class="mt-1 text-[10px] text-amber-600">{t.mirrorConfigFromEnv}</div>
          {/if}
        </div>
        <div class="flex items-center gap-2 text-[12px] text-gray-600">
          <input type="checkbox" class="rounded" bind:checked={terraformMirrorForm.setEnv} />
          <span>{t.mirrorSetEnv}</span>
        </div>
        <div class="pt-1 flex flex-wrap gap-2 items-center">
          <button
            class="h-8 px-4 bg-gray-900 text-white text-[12px] font-medium rounded-lg hover:bg-gray-800 transition-colors disabled:opacity-50 cursor-pointer inline-flex items-center gap-1.5"
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
          <button class="h-8 px-3 text-gray-700 bg-white border border-gray-200 hover:bg-gray-50 text-[12px] font-medium rounded-lg transition-colors cursor-pointer" onclick={enableAliyunMirrorQuick}>{t.mirrorAliyunPreset}</button>
          <button class="h-8 px-3 text-gray-700 bg-white border border-gray-200 hover:bg-gray-50 text-[12px] font-medium rounded-lg transition-colors cursor-pointer" onclick={enableTencentMirrorQuick}>{t.mirrorTencentPreset}</button>
          <button class="h-8 px-3 text-gray-700 bg-white border border-gray-200 hover:bg-gray-50 text-[12px] font-medium rounded-lg transition-colors cursor-pointer" onclick={enableVolcMirrorQuick}>{t.mirrorVolcPreset}</button>
          {#if terraformMirrorError}
            <span class="text-[11px] text-red-500">{terraformMirrorError}</span>
          {:else if terraformMirror.managed}
            <span class="text-[11px] text-emerald-600">OK</span>
          {/if}
        </div>
        <div class="text-[10px] text-gray-500 leading-relaxed">
          <span class="font-medium text-gray-600">{t.mirrorLimitTitle}</span>
          <span class="ml-1">{t.mirrorLimitDesc}</span>
        </div>
      </div>
    </div>

    <!-- 网络诊断 -->
    <div class="bg-white rounded-xl border border-gray-100 overflow-hidden">
      <div class="px-5 py-3 border-b border-gray-100 flex items-center justify-between">
        <h3 class="text-[13px] font-semibold text-gray-900">{t.networkCheck}</h3>
        <button
          class="h-7 px-3 text-gray-700 bg-white border border-gray-200 hover:bg-gray-50 text-[11px] font-medium rounded-lg transition-colors disabled:opacity-50 cursor-pointer inline-flex items-center gap-1.5"
          onclick={runTerraformNetworkCheck}
          disabled={networkCheckLoading}
        >
          <svg class="w-3 h-3 {networkCheckLoading ? 'animate-spin' : ''}" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M16.023 9.348h4.992v-.001M2.985 19.644v-4.992m0 0h4.992m-4.993 0l3.181 3.183a8.25 8.25 0 0013.803-3.7M4.031 9.865a8.25 8.25 0 0113.803-3.7l3.181 3.182" /></svg>
          {networkCheckLoading ? t.networkChecking : t.networkCheckBtn}
        </button>
      </div>
      <div class="px-5 py-3">
        {#if networkCheckError}
          <div class="text-[12px] text-red-500 mb-3">{networkCheckError}</div>
        {/if}
        {#if networkChecks.length > 0}
          <div class="divide-y divide-gray-50">
            {#each networkChecks as item}
              <div class="flex items-center justify-between py-2">
                <div class="flex items-center gap-2 flex-1 min-w-0">
                  <div class="w-1.5 h-1.5 rounded-full flex-shrink-0 {item.ok ? 'bg-emerald-500' : 'bg-red-500'}"></div>
                  <span class="text-[12px] text-gray-700 truncate">{item.name}</span>
                </div>
                <div class="flex items-center gap-3 flex-shrink-0">
                  <span class="text-[11px] text-gray-400 tabular-nums">{item.latencyMs}ms</span>
                  <span class="text-[11px] font-medium w-8 text-right {item.ok ? 'text-emerald-500' : 'text-red-500'}">
                    {item.ok ? 'OK' : 'FAIL'}
                  </span>
                  {#if item.error}
                    <span class="text-[10px] text-gray-400 truncate max-w-[200px]" title={item.error}>{item.error}</span>
                  {/if}
                </div>
              </div>
            {/each}
          </div>
        {:else if !networkCheckLoading}
          <div class="text-center py-4 text-[12px] text-gray-400">{t.settingsClickTest || '点击上方按钮进行网络诊断'}</div>
        {:else}
          <div class="text-center py-4 text-[12px] text-gray-400">{t.loading || '加载中...'}</div>
        {/if}
      </div>
    </div>

    <!-- Webhook 通知 -->
    <div class="bg-white rounded-xl border border-gray-100 overflow-hidden">
      <div class="px-5 py-3 border-b border-gray-100 flex items-center justify-between">
        <div>
          <h3 class="text-[13px] font-semibold text-gray-900">{t.webhookNotification || 'Webhook 通知'}</h3>
          <p class="text-[11px] text-gray-500 mt-0.5">{t.webhookNotificationDesc || '场景状态变化时推送消息'}</p>
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
      <div class="px-5 py-4 space-y-3">
        {#each [
          { key: 'slack', label: t.webhookSlack || 'Slack Webhook URL', hasSecret: false },
          { key: 'dingtalk', label: t.webhookDingtalk || '钉钉 Webhook URL', hasSecret: true, secretKey: 'dingtalkSecret' },
          { key: 'feishu', label: t.webhookFeishu || '飞书 Webhook URL', hasSecret: true, secretKey: 'feishuSecret' },
          { key: 'discord', label: t.webhookDiscord || 'Discord Webhook URL', hasSecret: false },
          { key: 'wecom', label: t.webhookWecom || '企业微信 Webhook URL', hasSecret: false },
        ] as wh}
          <div>
            <label class="text-[11px] text-gray-500 mb-1 block">{wh.label}</label>
            <div class="flex gap-2">
              <input type="text" bind:value={webhookForm[wh.key]} placeholder={t.webhookUrlHint || '留空则不推送'}
                class="flex-1 h-9 px-3 text-[12px] bg-gray-50 border-0 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow font-mono" />
              <button onclick={() => handleTestWebhook(wh.key)} disabled={webhookTesting[wh.key] || !webhookForm[wh.key]}
                class="h-9 px-3 text-[11px] font-medium bg-gray-100 hover:bg-gray-200 rounded-lg disabled:opacity-40 cursor-pointer disabled:cursor-not-allowed transition-colors">
                {webhookTesting[wh.key] ? '...' : (t.webhookTest || '测试')}
              </button>
            </div>
            {#if wh.hasSecret}
              <input type="text" bind:value={webhookForm[wh.secretKey]} placeholder={t.webhookSecretHint || '签名密钥（可选）'}
                class="mt-1.5 w-full h-9 px-3 text-[12px] bg-gray-50 border-0 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow font-mono" />
            {/if}
          </div>
        {/each}
        <div class="flex items-center gap-3 pt-1">
          <button onclick={handleSaveWebhook} disabled={webhookSaving}
            class="h-8 px-4 bg-gray-900 text-white text-[12px] font-medium rounded-lg hover:bg-gray-800 transition-colors disabled:opacity-50 cursor-pointer disabled:cursor-not-allowed">
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
</div>
