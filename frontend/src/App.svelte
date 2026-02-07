<script>
  import { onMount, onDestroy } from 'svelte';
  import { i18n as i18nData } from './lib/i18n.js';
  import { EventsOn, EventsOff, BrowserOpenURL } from '../wailsjs/runtime/runtime.js';
  import { ListCases, ListTemplates, GetConfig, SaveProxyConfig, GetMCPStatus, StartMCPServer, StopMCPServer, SetDebugLogging, GetResourceSummary, GetBalances, GetTerraformMirrorConfig, SaveTerraformMirrorConfig, TestTerraformEndpoints, SetNotificationEnabled, GetNotificationEnabled } from '../wailsjs/go/main/App.js';
  import Console from './components/Console/Console.svelte';
  import CloudResources from './components/Resources/CloudResources.svelte';
  import Compose from './components/Compose/Compose.svelte';
  import AIIntegration from './components/AI/AIIntegration.svelte';
  import SpecialModules from './components/SpecialModules/SpecialModules.svelte';
  import Registry from './components/Registry/Registry.svelte';
  import Credentials from './components/Credentials/Credentials.svelte';
  import LocalTemplates from './components/LocalTemplates/LocalTemplates.svelte';
  import Dashboard from './components/Dashboard/Dashboard.svelte';

  let cases = [];
  let templates = [];
  let logs = [];
  let config = { redcPath: '', projectPath: '', logPath: '', httpProxy: '', httpsProxy: '', noProxy: '', debugEnabled: false };
  let activeTab = 'dashboard';
  let specialModuleTab = 'vulhub';
  let isLoading = false;
  let error = '';
  let proxyForm = { httpProxy: '', httpsProxy: '', noProxy: '' };
  let proxySaving = false;
  let debugEnabled = false;
  let debugSaving = false;
  let notificationEnabled = false;
  let notificationSaving = false;
  let terraformMirror = { enabled: false, configPath: '', managed: false, fromEnv: false, providers: [] };
  let terraformMirrorForm = { enabled: false, configPath: '', setEnv: false, providers: { aliyun: true, tencent: false, volc: false } };
  let terraformMirrorSaving = false;
  let terraformMirrorError = '';
  let networkChecks = [];
  let networkCheckLoading = false;
  let networkCheckError = '';

  // MCP state
  let mcpStatus = { running: false, mode: '', address: '', protocolVersion: '' };
  let mcpForm = { mode: 'sse', address: 'localhost:8080' };
  let mcpLoading = false;

  // Resources state
  let resourceSummary = [];
  let resourcesLoading = false;
  let resourcesError = '';
  let balanceResults = [];
  let balanceLoading = false;
  let balanceError = '';
  let balanceCooldown = 0;
  let balanceCooldownTimer = null;

  // i18n state
  let lang = localStorage.getItem('lang') || 'zh';
  const i18n = { ...i18nData };
  $: t = i18n[lang];
  
  // Dashboard component reference
  let dashboardComponent;

  function toggleLang() {
    lang = lang === 'zh' ? 'en' : 'zh';
    localStorage.setItem('lang', lang);
  }

  function openGitHub() {
    BrowserOpenURL('https://github.com/wgpsec/redc');
  }

  function normalizeVersion(value) {
    if (!value) return '';
    return String(value).trim().replace(/^v/i, '');
  }

  function compareVersions(a, b) {
    const va = normalizeVersion(a).split('.').map(part => parseInt(part, 10));
    const vb = normalizeVersion(b).split('.').map(part => parseInt(part, 10));
    const maxLen = Math.max(va.length, vb.length);
    for (let i = 0; i < maxLen; i += 1) {
      const na = Number.isFinite(va[i]) ? va[i] : 0;
      const nb = Number.isFinite(vb[i]) ? vb[i] : 0;
      if (na > nb) return 1;
      if (na < nb) return -1;
    }
    return 0;
  }

  function hasUpdate(tmpl) {
    if (!tmpl || !tmpl.installed) return false;
    if (!tmpl.latest || !tmpl.localVersion) return false;
    return compareVersions(tmpl.latest, tmpl.localVersion) > 0;
  }

  onMount(async () => {
    EventsOn('log', (message) => {
      logs = [...logs, { time: new Date().toLocaleTimeString(), message }];
      if (dashboardComponent && dashboardComponent.updateCreateStatusFromLog) {
        dashboardComponent.updateCreateStatusFromLog(message);
      }
    });
    EventsOn('refresh', async () => {
      await refreshData();
    });
    await refreshData();
  });

  onDestroy(() => {
    EventsOff('log');
    EventsOff('refresh');
    if (balanceCooldownTimer) {
      clearInterval(balanceCooldownTimer);
      balanceCooldownTimer = null;
    }
  });

  async function refreshData() {
    isLoading = true;
    error = '';
    try {
      [cases, templates, config, terraformMirror, notificationEnabled] = await Promise.all([
        ListCases(),
        ListTemplates(),
        GetConfig(),
        GetTerraformMirrorConfig(),
        GetNotificationEnabled()
      ]);
      proxyForm = {
        httpProxy: config.httpProxy || '',
        httpsProxy: config.httpsProxy || '',
        noProxy: config.noProxy || ''
      };
      debugEnabled = !!config.debugEnabled;
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
      // Refresh dashboard component if it exists
      if (dashboardComponent && dashboardComponent.refresh) {
        await dashboardComponent.refresh();
      }
    } catch (e) {
      error = e.message || String(e);
      cases = [];
      templates = [];
    } finally {
      isLoading = false;
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

  async function handleSaveProxy() {
    proxySaving = true;
    try {
      await SaveProxyConfig(proxyForm.httpProxy, proxyForm.httpsProxy, proxyForm.noProxy);
      config.httpProxy = proxyForm.httpProxy;
      config.httpsProxy = proxyForm.httpsProxy;
      config.noProxy = proxyForm.noProxy;
    } catch (e) {
      error = e.message || String(e);
    } finally {
      proxySaving = false;
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
      error = e.message || String(e);
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
      error = e.message || String(e);
    } finally {
      notificationSaving = false;
    }
  }



  async function syncLocalTemplates() {
    try {
      const list = await ListTemplates();
      templates = list || [];
    } catch (e) {
      error = e.message || String(e);
    }
  }

  // MCP functions
  async function loadMCPStatus() {
    try {
      mcpStatus = await GetMCPStatus();
    } catch (e) {
      console.error('Failed to load MCP status:', e);
    }
  }

  async function handleStartMCP() {
    mcpLoading = true;
    try {
      mcpForm.mode = 'sse';
      await StartMCPServer(mcpForm.mode, mcpForm.address);
      await loadMCPStatus();
    } catch (e) {
      error = e.message || String(e);
    } finally {
      mcpLoading = false;
    }
  }

  async function handleStopMCP() {
    mcpLoading = true;
    try {
      await StopMCPServer();
      await loadMCPStatus();
    } catch (e) {
      error = e.message || String(e);
    } finally {
      mcpLoading = false;
    }
  }

  // Resources functions
  async function loadResourceSummary() {
    resourcesLoading = true;
    resourcesError = '';
    try {
      resourceSummary = await GetResourceSummary() || [];
    } catch (e) {
      resourcesError = e.message || String(e);
      resourceSummary = [];
    } finally {
      resourcesLoading = false;
    }
  }

  async function queryBalances() {
    if (balanceCooldown > 0) return;
    balanceLoading = true;
    balanceError = '';
    try {
      balanceResults = await GetBalances(['aliyun', 'tencentcloud', 'volcengine', 'huaweicloud']) || [];
      balanceCooldown = 5;
      if (balanceCooldownTimer) {
        clearInterval(balanceCooldownTimer);
      }
      balanceCooldownTimer = setInterval(() => {
        balanceCooldown = Math.max(0, balanceCooldown - 1);
        if (balanceCooldown === 0 && balanceCooldownTimer) {
          clearInterval(balanceCooldownTimer);
          balanceCooldownTimer = null;
        }
      }, 1000);
    } catch (e) {
      balanceError = e.message || String(e);
    } finally {
      balanceLoading = false;
    }
  }

</script>

<div class="h-screen flex bg-[#fafbfc]">
  <!-- Sidebar -->
  <aside class="w-44 bg-white border-r border-gray-100 flex flex-col">
    <div class="h-14 flex items-center px-4 border-b border-gray-100">
      <div class="flex items-center gap-2">
        <div class="w-6 h-6 rounded-md bg-gradient-to-br from-rose-500 to-red-600 flex items-center justify-center">
          <span class="text-white text-[10px] font-bold">R</span>
        </div>
        <span class="text-[14px] font-semibold text-gray-900">RedC</span>
      </div>
    </div>
    
    <nav class="flex-1 p-2">
      <div class="space-y-0.5">
        <button 
          class="w-full flex items-center gap-2 px-2.5 py-2 rounded-lg text-[12px] font-medium transition-all
            {activeTab === 'dashboard' ? 'bg-gray-900 text-white' : 'text-gray-600 hover:bg-gray-50'}"
          on:click={() => activeTab = 'dashboard'}
        >
          <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
            <path stroke-linecap="round" stroke-linejoin="round" d="M3.75 6A2.25 2.25 0 016 3.75h2.25A2.25 2.25 0 0110.5 6v2.25a2.25 2.25 0 01-2.25 2.25H6a2.25 2.25 0 01-2.25-2.25V6zM3.75 15.75A2.25 2.25 0 016 13.5h2.25a2.25 2.25 0 012.25 2.25V18a2.25 2.25 0 01-2.25 2.25H6A2.25 2.25 0 013.75 18v-2.25zM13.5 6a2.25 2.25 0 012.25-2.25H18A2.25 2.25 0 0120.25 6v2.25A2.25 2.25 0 0118 10.5h-2.25a2.25 2.25 0 01-2.25-2.25V6zM13.5 15.75a2.25 2.25 0 012.25-2.25H18a2.25 2.25 0 012.25 2.25V18A2.25 2.25 0 0118 20.25h-2.25A2.25 2.25 0 0113.5 18v-2.25z" />
          </svg>
          {t.dashboard}
        </button>
        <button 
          class="w-full flex items-center gap-2 px-2.5 py-2 rounded-lg text-[12px] font-medium transition-all
            {activeTab === 'console' ? 'bg-gray-900 text-white' : 'text-gray-600 hover:bg-gray-50'}"
          on:click={() => activeTab = 'console'}
        >
          <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
            <path stroke-linecap="round" stroke-linejoin="round" d="M6.75 7.5l3 2.25-3 2.25m4.5 0h3m-9 8.25h13.5A2.25 2.25 0 0021 18V6a2.25 2.25 0 00-2.25-2.25H5.25A2.25 2.25 0 003 6v12a2.25 2.25 0 002.25 2.25z" />
          </svg>
          {t.console}
        </button>
        <button 
          class="w-full flex items-center gap-2 px-2.5 py-2 rounded-lg text-[12px] font-medium transition-all
            {activeTab === 'resources' ? 'bg-gray-900 text-white' : 'text-gray-600 hover:bg-gray-50'}"
          on:click={() => { activeTab = 'resources'; loadResourceSummary(); }}
        >
          <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
            <path stroke-linecap="round" stroke-linejoin="round" d="M3 7.5l9 4.5 9-4.5M3 12l9 4.5 9-4.5M3 16.5l9 4.5 9-4.5" />
          </svg>
          {t.resources}
        </button>
        <button 
          class="w-full flex items-center gap-2 px-2.5 py-2 rounded-lg text-[12px] font-medium transition-all
            {activeTab === 'compose' ? 'bg-gray-900 text-white' : 'text-gray-600 hover:bg-gray-50'}"
          on:click={() => activeTab = 'compose'}
        >
          <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
            <path stroke-linecap="round" stroke-linejoin="round" d="M3.75 6A2.25 2.25 0 016 3.75h12A2.25 2.25 0 0120.25 6v12A2.25 2.25 0 0118 20.25H6A2.25 2.25 0 013.75 18V6z" />
            <path stroke-linecap="round" stroke-linejoin="round" d="M8 8h8M8 12h8M8 16h5" />
          </svg>
          {t.compose}
        </button>
        <button 
          class="w-full flex items-center gap-2 px-2.5 py-2 rounded-lg text-[12px] font-medium transition-all
            {activeTab === 'credentials' ? 'bg-gray-900 text-white' : 'text-gray-600 hover:bg-gray-50'}"
          on:click={() => { activeTab = 'credentials'; }}
        >
          <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
            <path stroke-linecap="round" stroke-linejoin="round" d="M15.75 5.25a3 3 0 013 3m3 0a6 6 0 01-7.029 5.912c-.563-.097-1.159.026-1.563.43L10.5 17.25H8.25v2.25H6v2.25H2.25v-2.818c0-.597.237-1.17.659-1.591l6.499-6.499c.404-.404.527-1 .43-1.563A6 6 0 1121.75 8.25z" />
          </svg>
          {t.credentials}
        </button>
        <button 
          class="w-full flex items-center gap-2 px-2.5 py-2 rounded-lg text-[12px] font-medium transition-all
            {activeTab === 'registry' ? 'bg-gray-900 text-white' : 'text-gray-600 hover:bg-gray-50'}"
          on:click={() => { activeTab = 'registry'; }}
        >
          <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
            <path stroke-linecap="round" stroke-linejoin="round" d="M20.25 7.5l-.625 10.632a2.25 2.25 0 01-2.247 2.118H6.622a2.25 2.25 0 01-2.247-2.118L3.75 7.5M10 11.25h4M3.375 7.5h17.25c.621 0 1.125-.504 1.125-1.125v-1.5c0-.621-.504-1.125-1.125-1.125H3.375c-.621 0-1.125.504-1.125 1.125v1.5c0 .621.504 1.125 1.125 1.125z" />
          </svg>
          {t.registry}
        </button>
        <button 
          class="w-full flex items-center gap-2 px-2.5 py-2 rounded-lg text-[12px] font-medium transition-all
            {activeTab === 'localTemplates' ? 'bg-gray-900 text-white' : 'text-gray-600 hover:bg-gray-50'}"
          on:click={() => { activeTab = 'localTemplates'; }}
        >
          <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
            <path stroke-linecap="round" stroke-linejoin="round" d="M2.25 12.75V12A2.25 2.25 0 014.5 9.75h15A2.25 2.25 0 0121.75 12v.75m-8.69-6.44l-2.12-2.12a1.5 1.5 0 00-1.061-.44H4.5A2.25 2.25 0 002.25 6v12a2.25 2.25 0 002.25 2.25h15A2.25 2.25 0 0021.75 18V9a2.25 2.25 0 00-2.25-2.25h-5.379a1.5 1.5 0 01-1.06-.44z" />
          </svg>
          {t.localTemplates}
        </button>
        <button
          class="w-full flex items-center gap-2 px-2.5 py-2 rounded-lg text-[12px] font-medium transition-all
            {activeTab === 'specialModules' ? 'bg-gray-900 text-white' : 'text-gray-600 hover:bg-gray-50'}"
          on:click={() => { activeTab = 'specialModules'; }}
        >
          <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
            <path stroke-linecap="round" stroke-linejoin="round" d="M11.42 15.17L17.25 21A2.25 2.25 0 0020 18.75V8.25A2.25 2.25 0 0017.75 6H11.42M6.75 6h.008v.008H6.75V6zm2.25 0h.008v.008H9V6zm2.25 0h.008v.008h-.008V6zm2.25 0h.008v.008h-.008V6zm2.25 0h.008v.008h-.008V6zm2.25 0h.008v.008h-.008V6zm2.25 0h.008v.008h-.008V6zm2.25 0h.008v.008h-.008V6zm2.25 0h.008v.008h-.008V6zm2.25 0h.008v.008h-.008V6zm2.25 0h.0088v.008h-.008V6zm2.25 0h.008v.008h-.008V6zM6.75 8.25h.008v.008H6.75v-.008zm2.25 0h.008v.008H9v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zM6.75 10.5h.008v.008H6.75v-.008zm2.25 0h.008v.008H9v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.0088v.008h-.008v-.008zM6.75 12.75h.008v.008H6.75v-.008zm2.25 0h.008v.008H9v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v`-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zM6.75 15h.008v.008H6.75v-.008zm2.25 0h.008v.008H9v-.008zm2.25 0h.008v.008h-.`008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.`008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zM6.75 17.25h.008v.008H6.75v-.008zm2.25 0h.008v.008H9v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zM6.75 19.5h.008v.008H6.75v-.008zm2.25 0h.008v.008H9v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008zm2.25 0h.008v.008h-.008v-.008z" />
          </svg>
          {t.specialModules}
        </button>
        <button
          class="w-full flex items-center gap-2 px-2.5 py-2 rounded-lg text-[12px] font-medium transition-all
            {activeTab === 'ai' ? 'bg-gray-900 text-white' : 'text-gray-600 hover:bg-gray-50'}"
          on:click={() => { activeTab = 'ai'; loadMCPStatus(); }}
        >
          <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
            <path stroke-linecap="round" stroke-linejoin="round" d="M9.813 15.904L9 18.75l-.813-2.846a4.5 4.5 0 00-3.09-3.09L2.25 12l2.846-.813a4.5 4.5 0 003.09-3.09L9 5.25l.813 2.846a4.5 4.5 0 003.09 3.09L15.75 12l-2.846.813a4.5 4.5 0 00-3.09 3.09zM18.259 8.715L18 9.75l-.259-1.035a3.375 3.375 0 00-2.455-2.456L14.25 6l1.036-.`259a3.375 3.375 0 002.455-2.456L18 2.25l.259 1.035a3.375 3.375 0 002.456 2.456L21.75 6l-1.035.259a3.375 3.375 0 00-2.456 2.456zM16.894 20.567L16.5 21.75l-.394-1.183a2.25 2.25 0 00-1.423-1.423L13.5 18.75l1.183-.394a2.25 2.25 0 001.423-1.423l.394-1.183.394 1.183a2.25 2.25 0 001.423 1.423l1.183.394-1.183.394a2.25 2.25 0 00-1.423 1.423z" />
          </svg>
          {t.ai}
        </button>
        <button
          class="w-full flex items-center gap-2 px-2.5 py-2 rounded-lg text-[12px] font-medium transition-all
            {activeTab === 'settings' ? 'bg-gray-900 text-white' : 'text-gray-600 hover:bg-gray-50'}"
          on:click={() => activeTab = 'settings'}
        >
          <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
            <path stroke-linecap="round" stroke-linejoin="round" d="M9.594 3.94c.09-.542.56-.94 1.11-.94h2.593c.55 0 1.02.398 1.11.94l.213 1.281c.063.374.313.686.645.87.074.04.147.083.22.127.324.196.72.257 1.075.124l1.217-.456a1.125 1.125 0 011.37.49l1.296 2.247a1.125 1.125 0 01-.26 1.431l-1.003.827c-.293.24-.438.613-.431.992a6.759 6.759 0 010 .255c-.007.378.138.75.43.99l1.005.828c.424.35.534.954.26 1.43l-1.298 2.247a1.125 1.125 0 01-1.369.491l-1.217-.456c-.355-.133-.75-.072-1.076.124a6.57 6.57 0 01-.22.128c-.331.183-.581.495-.644.869l-.213 1.28c-.09.543-.56.941-1.11.941h-2.594c-.55 0-1.02-.398-1.11-.94l-.213-1.281c-.062-.374-.312-.686-.644-.87a6.52 6.52 0 01-.22-.127c-.325-.196-.72-.257-1.076-.124l-1.217.456a1.125 1.125 0 01-1.369-.49l-1.297-2.247a1.125 1.125 0 01.26-1.431l1.004-.827c.292-.24.437-.613.43-.992a6.932 6.932 0 010-.255c.007-.378-.138-.75-.43-.99l-1.004-.828a1.125 1.125 0 01-.26-1.43l1.297-2.247a1.125 1.125 0 011.37-.491l1.216.456c.356.133.751.072 1.076-.124.072-.044.146-.087.22-.128.332-.183.582-.495.644-.869l.214-1.281z" />
            <path stroke-linecap="round" stroke-linejoin="round" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
          </svg>
          {t.settings}
        </button>
      </div>
    </nav>

    <div class="p-2 border-t border-gray-100">
      <div class="flex items-center justify-between px-2 py-2">
        <span class="text-[10px] text-gray-400">v2.3.0 by WgpSec</span>
        <div class="flex items-center gap-1">
          <button
            class="w-6 h-6 flex items-center justify-center rounded hover:bg-gray-100 text-gray-400 hover:text-gray-600 transition-colors text-[10px] font-medium"
            on:click={toggleLang}
            title={lang === 'zh' ? 'Switch to English' : '切换到中文'}
          >{lang === 'zh' ? 'EN' : '中'}</button>
          <button
            class="w-6 h-6 flex items-center justify-center rounded hover:bg-gray-100 text-gray-400 hover:text-gray-600 transition-colors"
            on:click={openGitHub}
            title="GitHub"
          >
            <svg class="w-4 h-4" fill="currentColor" viewBox="0 0 24 24">
              <path fill-rule="evenodd" clip-rule="evenodd" d="M12 2C6.477 2 2 6.477 2 12c0 4.42 2.865 8.167 6.839 9.49.5.092.682-.217.682-.482 0-.237-.009-.866-.013-1.7-2.782.604-3.369-1.34-3.369-1.34-.454-1.156-1.11-1.464-1.11-1.464-.908-.62.069-.608.069-.608 1.003.07 1.531 1.03 1.531 1.03.892 1.529 2.341 1.087 2.91.831.092-.646.35-1.086.636-1.336-2.22-.253-4.555-1.11-4.555-4.943 0-1.091.39-1.984 1.029-2.683-.103-.253-.446-1.27.098-2.647 0 0 .84-.269 2.75 1.025A9.578 9.578 0 0112 6.836c.85.004 1.705.114 2.504.336 1.909-1.294 2.747-1.025 2.747-1.025.546 1.377.203 2.394.1 2.647.64.699 1.028 1.592 1.028 2.683 0 3.842-2.339 4.687-4.566 4.935.359.309.678.919.678 1.852 0 1.336-.012 2.415-.012 2.743 0 .267.18.578.688.48C19.138 20.163 22 16.418 22 12c0-5.523-4.477-10-10-10z" />
            </svg>
          </button>
        </div>
      </div>
    </div>
  </aside>

  <!-- Main -->
  <div class="flex-1 flex flex-col min-w-0">
    <!-- Header -->
    <header class="h-14 bg-white border-b border-gray-100 flex items-center justify-between px-6">
      <h1 class="text-[15px] font-medium text-gray-900">
        {#if activeTab === 'dashboard'}{t.sceneManage}{:else if activeTab === 'console'}{t.console}{:else if activeTab === 'resources'}{t.resources}{:else if activeTab === 'compose'}{t.compose}{:else if activeTab === 'registry'}{t.templateRepo}{:else if activeTab === 'localTemplates'}{t.localTmplManage}{:else if activeTab === 'ai'}{t.aiIntegration}{:else if activeTab === 'credentials'}{t.credentials}{:else if activeTab === 'specialModules'}{t.specialModules}{:else}{t.settings}{/if}
      </h1>
      <button 
        class="w-8 h-8 flex items-center justify-center rounded-lg hover:bg-gray-50 text-gray-400 hover:text-gray-600 transition-colors"
        on:click={() => { refreshData(); if (activeTab === 'ai') loadMCPStatus(); if (activeTab === 'resources') loadResourceSummary(); }}
      >
        <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
          <path stroke-linecap="round" stroke-linejoin="round" d="M16.023 9.348h4.992v-.001M2.985 19.644v-4.992m0 0h4.992m-4.993 0l3.181 3.183a8.25 8.25 0 0013.803-3.7M4.031 9.865a8.25 8.25 0 0113.803-3.7l3.181 3.182m0-4.991v4.99" />
        </svg>
      </button>
    </header>

    <!-- Content -->
    <main class="flex-1 overflow-auto p-6">
      {#if error}
        <div class="mb-5 flex items-center gap-3 px-4 py-3 bg-red-50 border border-red-100 rounded-lg">
          <svg class="w-4 h-4 text-red-500 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z" />
          </svg>
          <span class="text-[13px] text-red-700 flex-1">{error}</span>
          <button class="text-red-400 hover:text-red-600" on:click={() => error = ''}>
            <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>
      {/if}

      {#if isLoading}
        <div class="flex items-center justify-center h-64">
          <div class="w-6 h-6 border-2 border-gray-200 border-t-gray-900 rounded-full animate-spin"></div>
        </div>
      {:else if activeTab === 'dashboard'}
        <Dashboard bind:this={dashboardComponent} {t} onTabChange={(tab) => activeTab = tab} />

      {:else if activeTab === 'console'}
        <Console {logs} {t} />

      {:else if activeTab === 'resources'}
        <CloudResources {t} />

      {:else if activeTab === 'compose'}
        <Compose {t} />

      {:else if activeTab === 'settings'}
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

      {:else if activeTab === 'registry'}
        <Registry {t} />

      {:else if activeTab === 'ai'}
        <AIIntegration {t} />

      {:else if activeTab === 'specialModules'}
        <SpecialModules {t} />

      {:else if activeTab === 'credentials'}
        <Credentials {t} />

      {:else if activeTab === 'localTemplates'}
        <LocalTemplates {t} />
      {/if}
    </main>
  </div>
</div>

<style>
  :global(body) {
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
    -webkit-font-smoothing: antialiased;
  }
  :global(select) {
    appearance: none;
    background-image: url("data:image/svg+xml,%3csvg xmlns='http://www.w3.org/2000/svg' fill='none' viewBox='0 0 20 20'%3e%3cpath stroke='%236b7280' stroke-linecap='round' stroke-linejoin='round' stroke-width='1.5' d='M6 8l4 4 4-4'/%3e%3c/svg%3e");
    background-position: right 0.5rem center;
    background-repeat: no-repeat;
    background-size: 1.5em 1.5em;
    padding-right: 2.5rem;
  }
</style>
