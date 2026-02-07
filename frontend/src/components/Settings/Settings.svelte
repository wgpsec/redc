<script>
  import { SaveProxyConfig, SetDebugLogging, GetTerraformMirrorConfig, SaveTerraformMirrorConfig, TestTerraformEndpoints, SetNotificationEnabled } from '../../../wailsjs/go/main/App.js';
  
  export let t;
  export let config = { redcPath: '', projectPath: '', logPath: '' };
  export let terraformMirror = { enabled: false, configPath: '', managed: false, fromEnv: false, providers: [] };
  export let debugEnabled = false;
  export let notificationEnabled = false;
  
  let proxyForm = { httpProxy: '', httpsProxy: '', noProxy: '' };
  let proxySaving = false;
  let terraformMirrorForm = { enabled: false, configPath: '', setEnv: false, providers: { aliyun: true, tencent: false, volc: false } };
  let terraformMirrorSaving = false;
  let terraformMirrorError = '';
  let networkChecks = [];
  let networkCheckLoading = false;
  let networkCheckError = '';
  let debugSaving = false;
  let notificationSaving = false;
  
  // Initialize forms when props change
  $: {
    proxyForm = {
      httpProxy: config.httpProxy || '',
      httpsProxy: config.httpsProxy || '',
      noProxy: config.noProxy || ''
    };
  }
  
  $: {
    terraformMirrorForm = {
      enabled: !!terraformMirror.enabled,
      configPath: terraformMirror.configPath || '',
      setEnv: !!terraformMirror.fromEnv,
      providers: {
        aliyun: terraformMirror.providers?.includes('aliyun'),
        tencent: terraformMirror.providers?.includes('tencent'),
        volc: terraformMirror.providers?.includes('volc')
      }
    };
  }
  
  async function handleSaveProxy() {
    proxySaving = true;
    try {
      await SaveProxyConfig(proxyForm.httpProxy, proxyForm.httpsProxy, proxyForm.noProxy);
      config.httpProxy = proxyForm.httpProxy;
      config.httpsProxy = proxyForm.httpsProxy;
      config.noProxy = proxyForm.noProxy;
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
      terraformMirrorForm = {
        enabled: !!terraformMirror.enabled,
        configPath: terraformMirror.configPath || '',
        setEnv: !!terraformMirror.fromEnv,
        providers: {
          aliyun: terraformMirror.providers?.includes('aliyun'),
          tencent: terraformMirror.providers?.includes('tencent'),
          volc: terraformMirror.providers?.includes('volc')
        }
      };
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
</script>


<div class="max-w-xl space-y-4">
  <!-- 基本信息 -->
  <div class="bg-white rounded-xl border border-gray-100 divide-y divide-gray-100">
    <div class="px-5 py-4">
      <div class="text-[12px] font-medium text-gray-500 mb-1">{t.redcPath}</div>
      <div class="text-[13px] text-gray-900 font-mono">{config.redcPath || '-'}</div>
    </div>
    <div class="px-5 py-4">
      <div class="text-[12px] font-medium text-gray-500 mb-1">{t.projectPath}</div>
      <div class="text-[13px] text-gray-900 font-mono">{config.projectPath || '-'}</div>
    </div>
    <div class="px-5 py-4">
      <div class="text-[12px] font-medium text-gray-500 mb-1">{t.logPath}</div>
      <div class="text-[13px] text-gray-900 font-mono">{config.logPath || '-'}</div>
    </div>
  </div>

  <!-- 代理配置 -->
  <div class="bg-white rounded-xl border border-gray-100 p-5">
    <div class="text-[14px] font-medium text-gray-900 mb-4">{t.proxyConfig}</div>
    <div class="space-y-4">
      <div>
        <label class="block text-[12px] font-medium text-gray-500 mb-1.5">{t.httpProxy}</label>
        <input 
          type="text" 
          placeholder="http://127.0.0.1:7890" 
          class="w-full h-10 px-3 text-[13px] bg-gray-50 border-0 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow font-mono"
          bind:value={proxyForm.httpProxy} 
        />
      </div>
      <div>
        <label class="block text-[12px] font-medium text-gray-500 mb-1.5">{t.httpsProxy}</label>
        <input 
          type="text" 
          placeholder="http://127.0.0.1:7890" 
          class="w-full h-10 px-3 text-[13px] bg-gray-50 border-0 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow font-mono"
          bind:value={proxyForm.httpsProxy} 
        />
      </div>
      <div>
        <label class="block text-[12px] font-medium text-gray-500 mb-1.5">{t.noProxyLabel}</label>
        <input 
          type="text" 
          placeholder="localhost,127.0.0.1,.local" 
          class="w-full h-10 px-3 text-[13px] bg-gray-50 border-0 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow font-mono"
          bind:value={proxyForm.noProxy} 
        />
      </div>
      <div class="pt-2">
        <button 
          class="h-10 px-5 bg-gray-900 text-white text-[13px] font-medium rounded-lg hover:bg-gray-800 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          on:click={handleSaveProxy}
          disabled={proxySaving}
        >
          {proxySaving ? t.saving : t.saveProxy}
        </button>
        <span class="ml-3 text-[12px] text-gray-500">{t.proxyHint}</span>
      </div>
    </div>
  </div>

  <!-- Terraform 镜像加速 -->
  <div class="bg-white rounded-xl border border-gray-100 p-5">
    <div class="flex items-start justify-between mb-4">
      <div>
        <div class="text-[14px] font-medium text-gray-900">{t.terraformMirror}</div>
        <div class="text-[12px] text-gray-500 mt-1">{t.mirrorConfigHint}</div>
      </div>
      <button
        class="relative inline-flex h-6 w-11 items-center rounded-full transition-colors"
        class:bg-emerald-500={terraformMirrorForm.enabled}
        class:bg-gray-300={!terraformMirrorForm.enabled}
        on:click={() => terraformMirrorForm = { ...terraformMirrorForm, enabled: !terraformMirrorForm.enabled }}
        aria-label={t.mirrorEnabled}
      >
        <span
          class="inline-block h-4 w-4 transform rounded-full bg-white transition-transform"
          class:translate-x-6={terraformMirrorForm.enabled}
          class:translate-x-1={!terraformMirrorForm.enabled}
        ></span>
      </button>
    </div>
    <div class="space-y-4">
      <div>
        <label class="block text-[12px] font-medium text-gray-500 mb-1.5">{t.mirrorProviders}</label>
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
        <div class="mt-2 text-[11px] text-gray-500">
          {t.mirrorProvidersDesc}
        </div>
      </div>
      <div>
        <label class="block text-[12px] font-medium text-gray-500 mb-1.5">{t.mirrorConfigPath}</label>
        <input
          type="text"
          placeholder={terraformMirror.configPath || t.mirrorConfigHint}
          class="w-full h-10 px-3 text-[13px] bg-gray-50 border-0 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow font-mono"
          bind:value={terraformMirrorForm.configPath}
        />
        {#if terraformMirror.fromEnv}
          <div class="mt-1 text-[11px] text-amber-600">{t.mirrorConfigFromEnv}</div>
        {/if}
      </div>
      <div class="flex items-center gap-2 text-[12px] text-gray-600">
        <input type="checkbox" class="rounded" bind:checked={terraformMirrorForm.setEnv} />
        <span>{t.mirrorSetEnv}</span>
      </div>
      <div class="pt-1 flex flex-wrap gap-2 items-center">
        <button
          class="h-9 px-4 bg-gray-900 text-white text-[12px] font-medium rounded-lg hover:bg-gray-800 transition-colors disabled:opacity-50"
          on:click={handleSaveTerraformMirror}
          disabled={terraformMirrorSaving}
        >
          {terraformMirrorSaving ? t.saving : t.mirrorSave}
        </button>
        <button
          class="h-9 px-4 bg-amber-500 text-white text-[12px] font-medium rounded-lg hover:bg-amber-600 transition-colors"
          on:click={enableAliyunMirrorQuick}
        >
          {t.mirrorAliyunPreset}
        </button>
        <button
          class="h-9 px-4 bg-sky-500 text-white text-[12px] font-medium rounded-lg hover:bg-sky-600 transition-colors"
          on:click={enableTencentMirrorQuick}
        >
          {t.mirrorTencentPreset}
        </button>
        <button
          class="h-9 px-4 bg-violet-500 text-white text-[12px] font-medium rounded-lg hover:bg-violet-600 transition-colors"
          on:click={enableVolcMirrorQuick}
        >
          {t.mirrorVolcPreset}
        </button>
        {#if terraformMirrorError}
          <span class="text-[12px] text-red-500">{terraformMirrorError}</span>
        {:else if terraformMirror.managed}
          <span class="text-[12px] text-emerald-600">OK</span>
        {/if}
      </div>
      <div class="mt-2 text-[11px] text-gray-500 leading-relaxed">
        <span class="font-medium text-gray-600">{t.mirrorLimitTitle}</span>
        <span class="ml-1">{t.mirrorLimitDesc}</span>
      </div>
    </div>
  </div>

  <!-- 网络诊断 -->
  <div class="bg-white rounded-xl border border-gray-100 p-5">
    <div class="flex items-center justify-between">
      <div class="text-[14px] font-medium text-gray-900">{t.networkCheck}</div>
      <button
        class="h-9 px-4 bg-gray-900 text-white text-[12px] font-medium rounded-lg hover:bg-gray-800 transition-colors disabled:opacity-50"
        on:click={runTerraformNetworkCheck}
        disabled={networkCheckLoading}
      >
        {networkCheckLoading ? t.networkChecking : t.networkCheckBtn}
      </button>
    </div>
    {#if networkCheckError}
      <div class="mt-3 text-[12px] text-red-500">{networkCheckError}</div>
    {/if}
    {#if networkChecks.length > 0}
      <div class="mt-4 border border-gray-100 rounded-lg overflow-hidden">
        <table class="w-full text-[12px]">
          <thead>
            <tr class="bg-gray-50 border-b border-gray-100">
              <th class="text-left px-4 py-2.5 font-semibold text-gray-600">{t.networkEndpoint}</th>
              <th class="text-right px-4 py-2.5 font-semibold text-gray-600">{t.networkStatus}</th>
              <th class="text-right px-4 py-2.5 font-semibold text-gray-600">{t.networkLatency}</th>
              <th class="text-left px-4 py-2.5 font-semibold text-gray-600">{t.networkError}</th>
            </tr>
          </thead>
          <tbody>
            {#each networkChecks as item}
              <tr class="border-b border-gray-50">
                <td class="px-4 py-3 text-gray-700">{item.name}</td>
                <td class="px-4 py-3 text-right {item.ok ? 'text-emerald-600' : 'text-red-600'}">{item.ok ? 'OK' : item.status || '-'}</td>
                <td class="px-4 py-3 text-right text-gray-700">{item.latencyMs} ms</td>
                <td class="px-4 py-3 text-gray-500 truncate" title={item.error}>{item.error || '-'}</td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/if}
  </div>

  <!-- 调试日志 -->
  <div class="bg-white rounded-xl border border-gray-100 p-5">
    <div class="flex items-center justify-between">
      <div>
        <div class="text-[14px] font-medium text-gray-900">{t.debugLogs}</div>
        <div class="text-[12px] text-gray-500 mt-1">{t.debugLogsDesc}</div>
      </div>
      <button
        class="relative inline-flex h-6 w-11 items-center rounded-full transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
        class:bg-emerald-500={debugEnabled}
        class:bg-gray-300={!debugEnabled}
        on:click={handleToggleDebug}
        disabled={debugSaving}
        aria-label={debugEnabled ? t.disable : t.enable}
      >
        <span
          class="inline-block h-4 w-4 transform rounded-full bg-white transition-transform"
          class:translate-x-6={debugEnabled}
          class:translate-x-1={!debugEnabled}
        ></span>
      </button>
    </div>
  </div>

  <!-- 系统通知 -->
  <div class="bg-white rounded-xl border border-gray-100 p-5">
    <div class="flex items-center justify-between">
      <div>
        <div class="text-[14px] font-medium text-gray-900">{t.systemNotification}</div>
        <div class="text-[12px] text-gray-500 mt-1">{t.systemNotificationDesc}</div>
      </div>
      <button
        class="relative inline-flex h-6 w-11 items-center rounded-full transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
        class:bg-emerald-500={notificationEnabled}
        class:bg-gray-300={!notificationEnabled}
        on:click={handleToggleNotification}
        disabled={notificationSaving}
        aria-label={notificationEnabled ? t.disable : t.enable}
      >
        <span
          class="inline-block h-4 w-4 transform rounded-full bg-white transition-transform"
          class:translate-x-6={notificationEnabled}
          class:translate-x-1={!notificationEnabled}
        ></span>
      </button>
    </div>
  </div>
</div>
