<script>
  import { onMount, onDestroy } from 'svelte';
  import { EventsOn, EventsOff } from '../wailsjs/runtime/runtime.js';
  import { ListCases, ListTemplates, StartCase, StopCase, RemoveCase, CreateCase, CreateAndRunCase, GetConfig, GetCaseOutputs, GetTemplateVariables, SaveProxyConfig, FetchRegistryTemplates, PullTemplate, GetMCPStatus, StartMCPServer, StopMCPServer, GetProvidersConfig, SaveProvidersConfig } from '../wailsjs/go/main/App.js';

  let cases = [];
  let templates = [];
  let logs = [];
  let config = { redcPath: '', projectPath: '', logPath: '', httpProxy: '', httpsProxy: '', noProxy: '' };
  let activeTab = 'dashboard';
  let selectedTemplate = '';
  let newCaseName = '';
  let isLoading = false;
  let error = '';
  let expandedCase = null;
  let caseOutputs = {};
  let deleteConfirm = { show: false, caseId: null, caseName: '' };
  let templateVariables = [];
  let variableValues = {};
  let proxyForm = { httpProxy: '', httpsProxy: '', noProxy: '' };
  let proxySaving = false;
  
  // Registry state
  let registryTemplates = [];
  let registryLoading = false;
  let registryError = '';
  let registrySearch = '';
  let pullingTemplates = {};

  // MCP state
  let mcpStatus = { running: false, mode: '', address: '', protocolVersion: '' };
  let mcpForm = { mode: 'sse', address: 'localhost:8080' };
  let mcpLoading = false;

  // Credentials state
  let providersConfig = { configPath: '', providers: [] };
  let credentialsLoading = false;
  let credentialsSaving = {};
  let editingProvider = null;
  let editFields = {};
  let customConfigPath = '';

  const stateConfig = {
    'running': { label: '运行中', color: 'text-emerald-600', bg: 'bg-emerald-50', dot: 'bg-emerald-500' },
    'stopped': { label: '已停止', color: 'text-slate-500', bg: 'bg-slate-50', dot: 'bg-slate-400' },
    'error': { label: '异常', color: 'text-red-600', bg: 'bg-red-50', dot: 'bg-red-500' },
    'created': { label: '已创建', color: 'text-blue-600', bg: 'bg-blue-50', dot: 'bg-blue-500' },
    'pending': { label: '等待中', color: 'text-amber-600', bg: 'bg-amber-50', dot: 'bg-amber-500' },
    'starting': { label: '启动中', color: 'text-amber-600', bg: 'bg-amber-50', dot: 'bg-amber-500 animate-pulse' },
    'stopping': { label: '停止中', color: 'text-amber-600', bg: 'bg-amber-50', dot: 'bg-amber-500 animate-pulse' },
    'removing': { label: '删除中', color: 'text-amber-600', bg: 'bg-amber-50', dot: 'bg-amber-500 animate-pulse' }
  };

  onMount(async () => {
    EventsOn('log', (message) => {
      logs = [...logs, { time: new Date().toLocaleTimeString(), message }];
    });
    EventsOn('refresh', async () => {
      await refreshData();
    });
    await refreshData();
  });

  onDestroy(() => {
    EventsOff('log');
    EventsOff('refresh');
  });

  async function refreshData() {
    isLoading = true;
    error = '';
    try {
      [cases, templates, config] = await Promise.all([
        ListCases(),
        ListTemplates(),
        GetConfig()
      ]);
      // Initialize proxy form with current config
      proxyForm = {
        httpProxy: config.httpProxy || '',
        httpsProxy: config.httpsProxy || '',
        noProxy: config.noProxy || ''
      };
    } catch (e) {
      error = e.message || String(e);
      cases = [];
      templates = [];
    } finally {
      isLoading = false;
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

  async function handleStart(caseId) {
    // 立即更新本地状态为"启动中"
    cases = cases.map(c => c.id === caseId ? { ...c, state: 'starting' } : c);
    try {
      await StartCase(caseId);
    } catch (e) {
      error = e.message || String(e);
      await refreshData(); // 出错时刷新恢复真实状态
    }
  }

  async function handleStop(caseId) {
    // 立即更新本地状态为"停止中"
    cases = cases.map(c => c.id === caseId ? { ...c, state: 'stopping' } : c);
    try {
      await StopCase(caseId);
    } catch (e) {
      error = e.message || String(e);
      await refreshData();
    }
  }

  function showDeleteConfirm(caseId, caseName) {
    deleteConfirm = { show: true, caseId, caseName };
  }

  function cancelDelete() {
    deleteConfirm = { show: false, caseId: null, caseName: '' };
  }

  async function confirmDelete() {
    const caseId = deleteConfirm.caseId;
    deleteConfirm = { show: false, caseId: null, caseName: '' };
    // 立即更新本地状态为"删除中"
    cases = cases.map(c => c.id === caseId ? { ...c, state: 'removing' } : c);
    try {
      await RemoveCase(caseId);
    } catch (e) {
      error = e.message || String(e);
      await refreshData();
    }
  }

  async function loadTemplateVariables(templateName) {
    if (!templateName) {
      templateVariables = [];
      variableValues = {};
      return;
    }
    try {
      const vars = await GetTemplateVariables(templateName);
      templateVariables = vars || [];
      // Initialize values with defaults
      variableValues = {};
      for (const v of templateVariables) {
        variableValues[v.name] = v.defaultValue || '';
      }
    } catch (e) {
      console.error('Failed to load template variables:', e);
      templateVariables = [];
      variableValues = {};
    }
  }

  async function handleCreate() {
    if (!selectedTemplate) {
      error = '请选择一个模板';
      return;
    }
    try {
      // Build vars object from variableValues, only include non-empty values
      const vars = {};
      for (const [key, value] of Object.entries(variableValues)) {
        if (value !== '') {
          vars[key] = value;
        }
      }
      await CreateCase(selectedTemplate, newCaseName, vars);
      selectedTemplate = '';
      newCaseName = '';
      templateVariables = [];
      variableValues = {};
      // 不需要立即刷新，后端完成后会发送 refresh 事件
    } catch (e) {
      error = e.message || String(e);
    }
  }

  async function handleCreateAndRun() {
    if (!selectedTemplate) {
      error = '请选择一个模板';
      return;
    }
    try {
      // Build vars object from variableValues, only include non-empty values
      const vars = {};
      for (const [key, value] of Object.entries(variableValues)) {
        if (value !== '') {
          vars[key] = value;
        }
      }
      await CreateAndRunCase(selectedTemplate, newCaseName, vars);
      selectedTemplate = '';
      newCaseName = '';
      templateVariables = [];
      variableValues = {};
      // 不需要立即刷新，后端完成后会发送 refresh 事件
    } catch (e) {
      error = e.message || String(e);
    }
  }

  function clearLogs() {
    logs = [];
  }

  function getShortId(id) {
    return id && id.length > 8 ? id.substring(0, 8) : id;
  }

  function getStateConfig(state) {
    return stateConfig[state] || stateConfig['pending'];
  }

  async function toggleCaseExpand(caseId, state) {
    if (expandedCase === caseId) {
      expandedCase = null;
      return;
    }
    expandedCase = caseId;
    // Only load outputs for running cases
    if (state === 'running' && !caseOutputs[caseId]) {
      try {
        const outputs = await GetCaseOutputs(caseId);
        if (outputs) {
          caseOutputs[caseId] = outputs;
          caseOutputs = caseOutputs; // trigger reactivity
        }
      } catch (e) {
        console.error('Failed to load outputs:', e);
      }
    }
  }

  let copiedKey = null;
  async function copyToClipboard(value, key) {
    try {
      await navigator.clipboard.writeText(value);
      copiedKey = key;
      setTimeout(() => { copiedKey = null; }, 2000);
    } catch (e) {
      console.error('Failed to copy:', e);
    }
  }

  // Registry functions
  async function loadRegistryTemplates() {
    registryLoading = true;
    registryError = '';
    try {
      registryTemplates = await FetchRegistryTemplates('');
    } catch (e) {
      registryError = e.message || String(e);
      registryTemplates = [];
    } finally {
      registryLoading = false;
    }
  }

  async function handlePullTemplate(templateName, force = false) {
    pullingTemplates[templateName] = true;
    pullingTemplates = pullingTemplates;
    try {
      await PullTemplate(templateName, force);
    } catch (e) {
      error = e.message || String(e);
    }
  }

  $: filteredRegistryTemplates = registryTemplates.filter(t => 
    !registrySearch || 
    t.name.toLowerCase().includes(registrySearch.toLowerCase()) ||
    (t.description && t.description.toLowerCase().includes(registrySearch.toLowerCase())) ||
    (t.tags && t.tags.some(tag => tag.toLowerCase().includes(registrySearch.toLowerCase())))
  );

  // Listen for refresh events to update pulling status
  $: if (registryTemplates.length > 0) {
    // Reset pulling status when templates are refreshed
    for (const t of registryTemplates) {
      if (t.installed && pullingTemplates[t.name]) {
        pullingTemplates[t.name] = false;
      }
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

  // Credentials functions
  async function loadProvidersConfig() {
    credentialsLoading = true;
    try {
      providersConfig = await GetProvidersConfig(customConfigPath);
    } catch (e) {
      error = e.message || String(e);
    } finally {
      credentialsLoading = false;
    }
  }

  function startEditProvider(provider) {
    editingProvider = provider.name;
    editFields = {};
    // Initialize edit fields with empty values (user must re-enter secrets)
    for (const key of Object.keys(provider.fields)) {
      // For non-secret fields (like region), pre-fill with current value
      if (!provider.hasSecrets || !provider.hasSecrets[key]) {
        editFields[key] = provider.fields[key] || '';
      } else {
        editFields[key] = '';
      }
    }
  }

  function cancelEditProvider() {
    editingProvider = null;
    editFields = {};
  }

  async function saveProviderCredentials(providerName) {
    credentialsSaving[providerName] = true;
    credentialsSaving = credentialsSaving;
    try {
      await SaveProvidersConfig(providerName, editFields, customConfigPath);
      editingProvider = null;
      editFields = {};
      await loadProvidersConfig();
    } catch (e) {
      error = e.message || String(e);
    } finally {
      credentialsSaving[providerName] = false;
      credentialsSaving = credentialsSaving;
    }
  }

  function getFieldLabel(key) {
    const labels = {
      accessKey: 'Access Key',
      secretKey: 'Secret Key',
      secretId: 'Secret ID',
      region: '区域',
      credentials: '凭据 JSON',
      project: '项目 ID',
      clientId: 'Client ID',
      clientSecret: 'Client Secret',
      subscriptionId: 'Subscription ID',
      tenantId: 'Tenant ID',
      user: '用户 OCID',
      tenancy: 'Tenancy OCID',
      fingerprint: '指纹',
      keyFile: '私钥文件路径',
      email: '邮箱',
      apiKey: 'API Key',
    };
    return labels[key] || key;
  }

  function isSecretField(key) {
    const secrets = ['accessKey', 'secretKey', 'secretId', 'credentials', 'clientId', 'clientSecret', 'subscriptionId', 'tenantId', 'user', 'tenancy', 'fingerprint', 'apiKey'];
    return secrets.includes(key);
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
          仪表盘
        </button>
        <button 
          class="w-full flex items-center gap-2 px-2.5 py-2 rounded-lg text-[12px] font-medium transition-all
            {activeTab === 'console' ? 'bg-gray-900 text-white' : 'text-gray-600 hover:bg-gray-50'}"
          on:click={() => activeTab = 'console'}
        >
          <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
            <path stroke-linecap="round" stroke-linejoin="round" d="M6.75 7.5l3 2.25-3 2.25m4.5 0h3m-9 8.25h13.5A2.25 2.25 0 0021 18V6a2.25 2.25 0 00-2.25-2.25H5.25A2.25 2.25 0 003 6v12a2.25 2.25 0 002.25 2.25z" />
          </svg>
          控制台
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
          设置
        </button>
        <button 
          class="w-full flex items-center gap-2 px-2.5 py-2 rounded-lg text-[12px] font-medium transition-all
            {activeTab === 'credentials' ? 'bg-gray-900 text-white' : 'text-gray-600 hover:bg-gray-50'}"
          on:click={() => { activeTab = 'credentials'; loadProvidersConfig(); }}
        >
          <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
            <path stroke-linecap="round" stroke-linejoin="round" d="M15.75 5.25a3 3 0 013 3m3 0a6 6 0 01-7.029 5.912c-.563-.097-1.159.026-1.563.43L10.5 17.25H8.25v2.25H6v2.25H2.25v-2.818c0-.597.237-1.17.659-1.591l6.499-6.499c.404-.404.527-1 .43-1.563A6 6 0 1121.75 8.25z" />
          </svg>
          凭据管理
        </button>
        <button 
          class="w-full flex items-center gap-2 px-2.5 py-2 rounded-lg text-[12px] font-medium transition-all
            {activeTab === 'registry' ? 'bg-gray-900 text-white' : 'text-gray-600 hover:bg-gray-50'}"
          on:click={() => { activeTab = 'registry'; loadRegistryTemplates(); }}
        >
          <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
            <path stroke-linecap="round" stroke-linejoin="round" d="M20.25 7.5l-.625 10.632a2.25 2.25 0 01-2.247 2.118H6.622a2.25 2.25 0 01-2.247-2.118L3.75 7.5M10 11.25h4M3.375 7.5h17.25c.621 0 1.125-.504 1.125-1.125v-1.5c0-.621-.504-1.125-1.125-1.125H3.375c-.621 0-1.125.504-1.125 1.125v1.5c0 .621.504 1.125 1.125 1.125z" />
          </svg>
          仓库
        </button>
        <button 
          class="w-full flex items-center gap-2 px-2.5 py-2 rounded-lg text-[12px] font-medium transition-all
            {activeTab === 'ai' ? 'bg-gray-900 text-white' : 'text-gray-600 hover:bg-gray-50'}"
          on:click={() => { activeTab = 'ai'; loadMCPStatus(); }}
        >
          <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
            <path stroke-linecap="round" stroke-linejoin="round" d="M9.813 15.904L9 18.75l-.813-2.846a4.5 4.5 0 00-3.09-3.09L2.25 12l2.846-.813a4.5 4.5 0 003.09-3.09L9 5.25l.813 2.846a4.5 4.5 0 003.09 3.09L15.75 12l-2.846.813a4.5 4.5 0 00-3.09 3.09zM18.259 8.715L18 9.75l-.259-1.035a3.375 3.375 0 00-2.455-2.456L14.25 6l1.036-.259a3.375 3.375 0 002.455-2.456L18 2.25l.259 1.035a3.375 3.375 0 002.456 2.456L21.75 6l-1.035.259a3.375 3.375 0 00-2.456 2.456zM16.894 20.567L16.5 21.75l-.394-1.183a2.25 2.25 0 00-1.423-1.423L13.5 18.75l1.183-.394a2.25 2.25 0 001.423-1.423l.394-1.183.394 1.183a2.25 2.25 0 001.423 1.423l1.183.394-1.183.394a2.25 2.25 0 00-1.423 1.423z" />
          </svg>
          AI 集成
        </button>
      </div>
    </nav>

    <div class="p-2 border-t border-gray-100">
      <div class="px-2 py-2 text-[10px] text-gray-400">v1.0.0</div>
    </div>
  </aside>

  <!-- Main -->
  <div class="flex-1 flex flex-col min-w-0">
    <!-- Header -->
    <header class="h-14 bg-white border-b border-gray-100 flex items-center justify-between px-6">
      <h1 class="text-[15px] font-medium text-gray-900">
        {#if activeTab === 'dashboard'}场景管理{:else if activeTab === 'console'}控制台{:else if activeTab === 'registry'}模板仓库{:else if activeTab === 'ai'}AI 集成{:else if activeTab === 'credentials'}凭据管理{:else}设置{/if}
      </h1>
      <button 
        class="w-8 h-8 flex items-center justify-center rounded-lg hover:bg-gray-50 text-gray-400 hover:text-gray-600 transition-colors"
        on:click={() => { refreshData(); if (activeTab === 'registry') loadRegistryTemplates(); if (activeTab === 'ai') loadMCPStatus(); if (activeTab === 'credentials') loadProvidersConfig(); }}
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
        <div class="space-y-5">
          <!-- Quick Create -->
          <div class="bg-white rounded-xl border border-gray-100 p-5">
            <div class="flex items-end gap-4 mb-4">
              <div class="flex-1">
                <label class="block text-[12px] font-medium text-gray-500 mb-1.5">模板</label>
                <select 
                  class="w-full h-10 px-3 text-[13px] bg-gray-50 border-0 rounded-lg text-gray-900 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow"
                  bind:value={selectedTemplate}
                  on:change={() => loadTemplateVariables(selectedTemplate)}
                >
                  <option value="">选择模板...</option>
                  {#each templates || [] as tmpl}
                    <option value={tmpl.name}>{tmpl.name}</option>
                  {/each}
                </select>
              </div>
              <div class="w-48">
                <label class="block text-[12px] font-medium text-gray-500 mb-1.5">名称</label>
                <input 
                  type="text" 
                  placeholder="可选" 
                  class="w-full h-10 px-3 text-[13px] bg-gray-50 border-0 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow"
                  bind:value={newCaseName} 
                />
              </div>
              <button 
                class="h-10 px-5 bg-gray-900 text-white text-[13px] font-medium rounded-lg hover:bg-gray-800 transition-colors"
                on:click={handleCreate}
              >
                创建
              </button>
              <button 
                class="h-10 px-5 bg-emerald-500 text-white text-[13px] font-medium rounded-lg hover:bg-emerald-600 transition-colors"
                on:click={handleCreateAndRun}
              >
                创建并运行
              </button>
            </div>
            
            <!-- Template Variables -->
            {#if templateVariables.length > 0}
              <div class="border-t border-gray-100 pt-4 mt-4">
                <div class="text-[12px] font-medium text-gray-500 mb-3">模板参数</div>
                <div class="grid grid-cols-2 gap-3">
                  {#each templateVariables as variable}
                    <div class="flex flex-col">
                      <label class="text-[11px] text-gray-500 mb-1">
                        {variable.name}
                        {#if variable.required}
                          <span class="text-red-500">*</span>
                        {/if}
                        {#if variable.description}
                          <span class="text-gray-400 ml-1">({variable.description})</span>
                        {/if}
                      </label>
                      <input 
                        type="text"
                        class="h-9 px-3 text-[12px] bg-gray-50 border-0 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow"
                        placeholder={variable.defaultValue || ''}
                        bind:value={variableValues[variable.name]}
                      />
                    </div>
                  {/each}
                </div>
              </div>
            {/if}
          </div>

          <!-- Table -->
          <div class="bg-white rounded-xl border border-gray-100 overflow-hidden">
            <table class="w-full">
              <thead>
                <tr class="border-b border-gray-100">
                  <th class="text-left px-5 py-3 text-[11px] font-semibold text-gray-500 uppercase tracking-wide">ID</th>
                  <th class="text-left px-5 py-3 text-[11px] font-semibold text-gray-500 uppercase tracking-wide">名称</th>
                  <th class="text-left px-5 py-3 text-[11px] font-semibold text-gray-500 uppercase tracking-wide">类型</th>
                  <th class="text-left px-5 py-3 text-[11px] font-semibold text-gray-500 uppercase tracking-wide">状态</th>
                  <th class="text-left px-5 py-3 text-[11px] font-semibold text-gray-500 uppercase tracking-wide">时间</th>
                  <th class="text-right px-5 py-3 text-[11px] font-semibold text-gray-500 uppercase tracking-wide">操作</th>
                </tr>
              </thead>
              <tbody>
                {#each cases || [] as c, i}
                  <tr 
                    class="border-b border-gray-50 hover:bg-gray-50/50 transition-colors cursor-pointer"
                    on:click={() => toggleCaseExpand(c.id, c.state)}
                  >
                    <td class="px-5 py-3.5">
                      <div class="flex items-center gap-2">
                        <svg class="w-4 h-4 text-gray-400 transition-transform {expandedCase === c.id ? 'rotate-90' : ''}" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
                        </svg>
                        <code class="text-[12px] text-gray-500 bg-gray-100 px-1.5 py-0.5 rounded">{getShortId(c.id)}</code>
                      </div>
                    </td>
                    <td class="px-5 py-3.5">
                      <span class="text-[13px] font-medium text-gray-900">{c.name}</span>
                    </td>
                    <td class="px-5 py-3.5">
                      <span class="text-[13px] text-gray-600">{c.type}</span>
                    </td>
                    <td class="px-5 py-3.5">
                      <span class="inline-flex items-center gap-1.5 text-[12px] font-medium {getStateConfig(c.state).color}">
                        <span class="w-1.5 h-1.5 rounded-full {getStateConfig(c.state).dot}"></span>
                        {getStateConfig(c.state).label}
                      </span>
                    </td>
                    <td class="px-5 py-3.5">
                      <span class="text-[12px] text-gray-500">{c.createTime}</span>
                    </td>
                    <td class="px-5 py-3.5 text-right" on:click|stopPropagation>
                      <div class="inline-flex items-center gap-1">
                        {#if c.state === 'starting' || c.state === 'stopping' || c.state === 'removing'}
                          <span class="px-2.5 py-1 text-[12px] font-medium text-amber-600">
                            {stateConfig[c.state]?.label || '处理中'}...
                          </span>
                        {:else if c.state !== 'running'}
                          <button 
                            class="px-2.5 py-1 text-[12px] font-medium text-emerald-700 bg-emerald-50 rounded-md hover:bg-emerald-100 transition-colors"
                            on:click={() => handleStart(c.id)}
                          >启动</button>
                        {:else}
                          <button 
                            class="px-2.5 py-1 text-[12px] font-medium text-amber-700 bg-amber-50 rounded-md hover:bg-amber-100 transition-colors"
                            on:click={() => handleStop(c.id)}
                          >停止</button>
                        {/if}
                        {#if c.state !== 'starting' && c.state !== 'stopping' && c.state !== 'removing'}
                          <button 
                            class="px-2.5 py-1 text-[12px] font-medium text-gray-600 bg-gray-50 rounded-md hover:bg-gray-100 transition-colors"
                            on:click={() => showDeleteConfirm(c.id, c.name)}
                          >删除</button>
                        {/if}
                      </div>
                    </td>
                  </tr>
                  <!-- Expanded row for outputs -->
                  {#if expandedCase === c.id}
                    <tr class="bg-slate-50">
                      <td colspan="6" class="px-5 py-4">
                        <div class="pl-6">
                          {#if c.state === 'running'}
                            {#if caseOutputs[c.id]}
                              <div class="grid grid-cols-2 gap-3">
                                {#each Object.entries(caseOutputs[c.id]) as [key, value]}
                                  <div class="bg-white rounded-lg p-3 border border-gray-100 group relative">
                                    <div class="flex items-center justify-between mb-1">
                                      <div class="text-[11px] text-gray-500 uppercase tracking-wide">{key}</div>
                                      <button 
                                        class="opacity-0 group-hover:opacity-100 transition-opacity p-1 hover:bg-gray-100 rounded flex items-center gap-1"
                                        on:click|stopPropagation={() => copyToClipboard(value, key)}
                                        title="复制"
                                      >
                                        {#if copiedKey === key}
                                          <svg class="w-4 h-4 text-emerald-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
                                          </svg>
                                          <span class="text-[10px] text-emerald-500">已复制</span>
                                        {:else}
                                          <svg class="w-4 h-4 text-gray-400 hover:text-gray-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
                                          </svg>
                                        {/if}
                                      </button>
                                    </div>
                                    <div class="text-[13px] font-mono text-gray-900 break-all">{value}</div>
                                  </div>
                                {/each}
                              </div>
                            {:else}
                              <div class="text-[13px] text-gray-500">正在加载输出信息...</div>
                            {/if}
                          {:else}
                            <div class="text-[13px] text-gray-500">场景未运行，无输出信息</div>
                          {/if}
                        </div>
                      </td>
                    </tr>
                  {/if}
                {:else}
                  <tr>
                    <td colspan="6" class="py-16">
                      <div class="flex flex-col items-center text-gray-400">
                        <svg class="w-10 h-10 mb-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1">
                          <path stroke-linecap="round" stroke-linejoin="round" d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4" />
                        </svg>
                        <p class="text-[13px]">暂无场景</p>
                      </div>
                    </td>
                  </tr>
                {/each}
              </tbody>
            </table>
          </div>
        </div>

      {:else if activeTab === 'console'}
        <div class="h-full flex flex-col bg-[#1e1e1e] rounded-xl overflow-hidden">
          <div class="flex items-center justify-between px-4 py-2.5 bg-[#252526] border-b border-[#3c3c3c]">
            <div class="flex items-center gap-2">
              <div class="flex gap-1.5">
                <span class="w-3 h-3 rounded-full bg-[#ff5f56]"></span>
                <span class="w-3 h-3 rounded-full bg-[#ffbd2e]"></span>
                <span class="w-3 h-3 rounded-full bg-[#27ca40]"></span>
              </div>
              <span class="text-[12px] text-gray-500 ml-2">Terminal</span>
            </div>
            <button 
              class="text-[11px] text-gray-500 hover:text-gray-300 transition-colors"
              on:click={clearLogs}
            >清空</button>
          </div>
          <div class="flex-1 p-4 overflow-auto font-mono text-[12px] leading-5">
            {#each logs as log}
              <div class="flex">
                <span class="text-gray-600 select-none">[{log.time}]</span>
                <span class="text-gray-300 ml-2">{log.message}</span>
              </div>
            {:else}
              <div class="text-gray-600">$ 等待输出...</div>
            {/each}
          </div>
        </div>

      {:else if activeTab === 'settings'}
        <div class="max-w-xl space-y-4">
          <!-- 基本信息 -->
          <div class="bg-white rounded-xl border border-gray-100 divide-y divide-gray-100">
            <div class="px-5 py-4">
              <div class="text-[12px] font-medium text-gray-500 mb-1">RedC 路径</div>
              <div class="text-[13px] text-gray-900 font-mono">{config.redcPath || '-'}</div>
            </div>
            <div class="px-5 py-4">
              <div class="text-[12px] font-medium text-gray-500 mb-1">项目路径</div>
              <div class="text-[13px] text-gray-900 font-mono">{config.projectPath || '-'}</div>
            </div>
            <div class="px-5 py-4">
              <div class="text-[12px] font-medium text-gray-500 mb-1">日志路径</div>
              <div class="text-[13px] text-gray-900 font-mono">{config.logPath || '-'}</div>
            </div>
          </div>

          <!-- 代理配置 -->
          <div class="bg-white rounded-xl border border-gray-100 p-5">
            <div class="text-[14px] font-medium text-gray-900 mb-4">代理配置</div>
            <div class="space-y-4">
              <div>
                <label class="block text-[12px] font-medium text-gray-500 mb-1.5">HTTP 代理</label>
                <input 
                  type="text" 
                  placeholder="例如: http://127.0.0.1:7890 或 socks5://127.0.0.1:1080" 
                  class="w-full h-10 px-3 text-[13px] bg-gray-50 border-0 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow font-mono"
                  bind:value={proxyForm.httpProxy} 
                />
              </div>
              <div>
                <label class="block text-[12px] font-medium text-gray-500 mb-1.5">HTTPS 代理</label>
                <input 
                  type="text" 
                  placeholder="例如: http://127.0.0.1:7890 或 socks5://127.0.0.1:1080" 
                  class="w-full h-10 px-3 text-[13px] bg-gray-50 border-0 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow font-mono"
                  bind:value={proxyForm.httpsProxy} 
                />
              </div>
              <div>
                <label class="block text-[12px] font-medium text-gray-500 mb-1.5">不使用代理的地址 (NO_PROXY)</label>
                <input 
                  type="text" 
                  placeholder="例如: localhost,127.0.0.1,.local" 
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
                  {proxySaving ? '保存中...' : '保存代理配置'}
                </button>
                <span class="ml-3 text-[12px] text-gray-500">配置后将用于 Terraform 的网络请求</span>
              </div>
            </div>
          </div>
        </div>

      {:else if activeTab === 'registry'}
        <div class="space-y-5">
          <!-- Search -->
          <div class="bg-white rounded-xl border border-gray-100 p-5">
            <div class="flex items-center gap-4">
              <div class="flex-1 relative">
                <svg class="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
                </svg>
                <input 
                  type="text" 
                  placeholder="搜索模板..." 
                  class="w-full h-10 pl-10 pr-4 text-[13px] bg-gray-50 border-0 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow"
                  bind:value={registrySearch} 
                />
              </div>
              <button 
                class="h-10 px-5 bg-gray-900 text-white text-[13px] font-medium rounded-lg hover:bg-gray-800 transition-colors disabled:opacity-50"
                on:click={loadRegistryTemplates}
                disabled={registryLoading}
              >
                {registryLoading ? '加载中...' : '刷新仓库'}
              </button>
            </div>
          </div>

          {#if registryError}
            <div class="flex items-center gap-3 px-4 py-3 bg-red-50 border border-red-100 rounded-lg">
              <svg class="w-4 h-4 text-red-500 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z" />
              </svg>
              <span class="text-[13px] text-red-700 flex-1">{registryError}</span>
              <button class="text-red-400 hover:text-red-600" on:click={() => registryError = ''}>
                <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>
          {/if}

          {#if registryLoading}
            <div class="flex items-center justify-center h-64">
              <div class="w-6 h-6 border-2 border-gray-200 border-t-gray-900 rounded-full animate-spin"></div>
            </div>
          {:else}
            <!-- Template Grid -->
            <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
              {#each filteredRegistryTemplates as tmpl}
                <div class="bg-white rounded-xl border border-gray-100 p-5 hover:shadow-md transition-shadow">
                  <div class="flex items-start justify-between mb-3">
                    <div class="flex-1 min-w-0">
                      <h3 class="text-[14px] font-semibold text-gray-900 truncate">{tmpl.name}</h3>
                      <p class="text-[12px] text-gray-500 mt-0.5">v{tmpl.latest}</p>
                    </div>
                    {#if tmpl.installed}
                      <span class="inline-flex items-center gap-1 px-2 py-0.5 bg-emerald-50 text-emerald-600 text-[11px] font-medium rounded-full">
                        <svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
                        </svg>
                        已安装
                      </span>
                    {/if}
                  </div>
                  
                  {#if tmpl.description}
                    <p class="text-[12px] text-gray-600 mb-3 line-clamp-2">{tmpl.description}</p>
                  {/if}
                  
                  {#if tmpl.tags && tmpl.tags.length > 0}
                    <div class="flex flex-wrap gap-1 mb-3">
                      {#each tmpl.tags.slice(0, 3) as tag}
                        <span class="px-2 py-0.5 bg-gray-100 text-gray-600 text-[10px] rounded-full">{tag}</span>
                      {/each}
                      {#if tmpl.tags.length > 3}
                        <span class="px-2 py-0.5 bg-gray-100 text-gray-400 text-[10px] rounded-full">+{tmpl.tags.length - 3}</span>
                      {/if}
                    </div>
                  {/if}
                  
                  <div class="flex items-center justify-between pt-3 border-t border-gray-100">
                    <div class="text-[11px] text-gray-400">
                      {#if tmpl.author}by {tmpl.author}{/if}
                    </div>
                    {#if pullingTemplates[tmpl.name]}
                      <span class="px-3 py-1.5 text-[12px] font-medium text-amber-600">
                        拉取中...
                      </span>
                    {:else if tmpl.installed}
                      <button 
                        class="px-3 py-1.5 text-[12px] font-medium text-blue-600 bg-blue-50 rounded-lg hover:bg-blue-100 transition-colors"
                        on:click={() => handlePullTemplate(tmpl.name, true)}
                      >更新</button>
                    {:else}
                      <button 
                        class="px-3 py-1.5 text-[12px] font-medium text-white bg-gray-900 rounded-lg hover:bg-gray-800 transition-colors"
                        on:click={() => handlePullTemplate(tmpl.name, false)}
                      >拉取</button>
                    {/if}
                  </div>
                </div>
              {:else}
                <div class="col-span-full py-16 text-center">
                  <svg class="w-10 h-10 mx-auto mb-3 text-gray-300" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M20.25 7.5l-.625 10.632a2.25 2.25 0 01-2.247 2.118H6.622a2.25 2.25 0 01-2.247-2.118L3.75 7.5M10 11.25h4M3.375 7.5h17.25c.621 0 1.125-.504 1.125-1.125v-1.5c0-.621-.504-1.125-1.125-1.125H3.375c-.621 0-1.125.504-1.125 1.125v1.5c0 .621.504 1.125 1.125 1.125z" />
                  </svg>
                  <p class="text-[13px] text-gray-400">
                    {#if registrySearch}
                      未找到匹配的模板
                    {:else}
                      点击"刷新仓库"加载模板列表
                    {/if}
                  </p>
                </div>
              {/each}
            </div>
          {/if}
        </div>

      {:else if activeTab === 'ai'}
        <div class="max-w-2xl space-y-5">
          <!-- MCP Status Card -->
          <div class="bg-white rounded-xl border border-gray-100 p-5">
            <div class="flex items-center justify-between mb-4">
              <div class="flex items-center gap-3">
                <div class="w-10 h-10 rounded-lg bg-gradient-to-br from-purple-500 to-indigo-600 flex items-center justify-center">
                  <svg class="w-5 h-5 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.813 15.904L9 18.75l-.813-2.846a4.5 4.5 0 00-3.09-3.09L2.25 12l2.846-.813a4.5 4.5 0 003.09-3.09L9 5.25l.813 2.846a4.5 4.5 0 003.09 3.09L15.75 12l-2.846.813a4.5 4.5 0 00-3.09 3.09z" />
                  </svg>
                </div>
                <div>
                  <h2 class="text-[14px] font-semibold text-gray-900">MCP 服务器</h2>
                  <p class="text-[12px] text-gray-500">Model Context Protocol 服务</p>
                </div>
              </div>
              <div class="flex items-center gap-2">
                {#if mcpStatus.running}
                  <span class="inline-flex items-center gap-1.5 px-2.5 py-1 bg-emerald-50 text-emerald-600 text-[12px] font-medium rounded-full">
                    <span class="w-1.5 h-1.5 rounded-full bg-emerald-500 animate-pulse"></span>
                    运行中
                  </span>
                {:else}
                  <span class="inline-flex items-center gap-1.5 px-2.5 py-1 bg-gray-50 text-gray-500 text-[12px] font-medium rounded-full">
                    <span class="w-1.5 h-1.5 rounded-full bg-gray-400"></span>
                    已停止
                  </span>
                {/if}
              </div>
            </div>

            {#if mcpStatus.running}
              <!-- Running status info -->
              <div class="bg-gray-50 rounded-lg p-4 mb-4">
                <div class="grid grid-cols-2 gap-4 text-[12px]">
                  <div>
                    <span class="text-gray-500">传输模式</span>
                    <p class="font-medium text-gray-900 mt-0.5">{mcpStatus.mode === 'sse' ? 'SSE (HTTP)' : 'STDIO'}</p>
                  </div>
                  <div>
                    <span class="text-gray-500">监听地址</span>
                    <p class="font-mono font-medium text-gray-900 mt-0.5">{mcpStatus.address || '-'}</p>
                  </div>
                  <div>
                    <span class="text-gray-500">协议版本</span>
                    <p class="font-medium text-gray-900 mt-0.5">{mcpStatus.protocolVersion}</p>
                  </div>
                  <div>
                    <span class="text-gray-500">消息端点</span>
                    <p class="font-mono font-medium text-gray-900 mt-0.5 text-[11px]">http://{mcpStatus.address}/message</p>
                  </div>
                </div>
              </div>
              <button 
                class="w-full h-10 bg-red-500 text-white text-[13px] font-medium rounded-lg hover:bg-red-600 transition-colors disabled:opacity-50"
                on:click={handleStopMCP}
                disabled={mcpLoading}
              >
                {mcpLoading ? '停止中...' : '停止服务器'}
              </button>
            {:else}
              <!-- Configuration form -->
              <div class="space-y-4 mb-4">
                <div>
                  <label class="block text-[12px] font-medium text-gray-500 mb-1.5">传输模式</label>
                  <div class="flex gap-2">
                    <button 
                      class="flex-1 h-10 px-4 text-[13px] font-medium rounded-lg border transition-colors
                        {mcpForm.mode === 'sse' ? 'bg-gray-900 text-white border-gray-900' : 'bg-white text-gray-700 border-gray-200 hover:bg-gray-50'}"
                      on:click={() => mcpForm.mode = 'sse'}
                    >
                      SSE (HTTP)
                    </button>
                    <button 
                      class="flex-1 h-10 px-4 text-[13px] font-medium rounded-lg border transition-colors
                        {mcpForm.mode === 'stdio' ? 'bg-gray-900 text-white border-gray-900' : 'bg-white text-gray-700 border-gray-200 hover:bg-gray-50'}"
                      on:click={() => mcpForm.mode = 'stdio'}
                    >
                      STDIO
                    </button>
                  </div>
                </div>
                {#if mcpForm.mode === 'sse'}
                  <div>
                    <label class="block text-[12px] font-medium text-gray-500 mb-1.5">监听地址</label>
                    <input 
                      type="text" 
                      placeholder="localhost:8080" 
                      class="w-full h-10 px-3 text-[13px] bg-gray-50 border-0 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow font-mono"
                      bind:value={mcpForm.address} 
                    />
                  </div>
                {/if}
              </div>
              <button 
                class="w-full h-10 bg-gray-900 text-white text-[13px] font-medium rounded-lg hover:bg-gray-800 transition-colors disabled:opacity-50"
                on:click={handleStartMCP}
                disabled={mcpLoading}
              >
                {mcpLoading ? '启动中...' : '启动服务器'}
              </button>
            {/if}
          </div>

          <!-- MCP Info Card -->
          <div class="bg-white rounded-xl border border-gray-100 p-5">
            <h3 class="text-[14px] font-semibold text-gray-900 mb-3">关于 MCP</h3>
            <p class="text-[12px] text-gray-600 leading-relaxed mb-4">
              Model Context Protocol (MCP) 是一种开放协议，允许 AI 助手与外部工具和数据源进行交互。
              启用 MCP 服务器后，您可以通过 Claude、Cursor 等支持 MCP 的 AI 工具直接管理 RedC 基础设施。
            </p>
            <div class="bg-gray-50 rounded-lg p-4">
              <div class="text-[11px] font-medium text-gray-500 uppercase tracking-wide mb-2">可用工具</div>
              <div class="grid grid-cols-2 gap-2 text-[12px]">
                <div class="flex items-center gap-2 text-gray-700">
                  <span class="w-1 h-1 rounded-full bg-gray-400"></span>
                  list_templates
                </div>
                <div class="flex items-center gap-2 text-gray-700">
                  <span class="w-1 h-1 rounded-full bg-gray-400"></span>
                  list_cases
                </div>
                <div class="flex items-center gap-2 text-gray-700">
                  <span class="w-1 h-1 rounded-full bg-gray-400"></span>
                  plan_case
                </div>
                <div class="flex items-center gap-2 text-gray-700">
                  <span class="w-1 h-1 rounded-full bg-gray-400"></span>
                  start_case
                </div>
                <div class="flex items-center gap-2 text-gray-700">
                  <span class="w-1 h-1 rounded-full bg-gray-400"></span>
                  stop_case
                </div>
                <div class="flex items-center gap-2 text-gray-700">
                  <span class="w-1 h-1 rounded-full bg-gray-400"></span>
                  kill_case
                </div>
                <div class="flex items-center gap-2 text-gray-700">
                  <span class="w-1 h-1 rounded-full bg-gray-400"></span>
                  get_case_status
                </div>
                <div class="flex items-center gap-2 text-gray-700">
                  <span class="w-1 h-1 rounded-full bg-gray-400"></span>
                  exec_command
                </div>
              </div>
            </div>
          </div>
        </div>

      {:else if activeTab === 'credentials'}
        <div class="max-w-3xl space-y-5">
          <!-- Config Path -->
          <div class="bg-white rounded-xl border border-gray-100 p-5">
            <div class="flex items-center gap-4">
              <div class="flex-1">
                <label class="block text-[12px] font-medium text-gray-500 mb-1.5">配置文件路径</label>
                <input 
                  type="text" 
                  placeholder="留空使用默认路径 ~/redc/config.yaml" 
                  class="w-full h-10 px-3 text-[13px] bg-gray-50 border-0 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow font-mono"
                  bind:value={customConfigPath} 
                />
              </div>
              <button 
                class="h-10 px-5 bg-gray-900 text-white text-[13px] font-medium rounded-lg hover:bg-gray-800 transition-colors disabled:opacity-50 mt-5"
                on:click={loadProvidersConfig}
                disabled={credentialsLoading}
              >
                {credentialsLoading ? '加载中...' : '加载配置'}
              </button>
            </div>
            {#if providersConfig.configPath}
              <div class="mt-3 text-[12px] text-gray-500">
                当前配置: <span class="font-mono">{providersConfig.configPath}</span>
              </div>
            {/if}
          </div>

          <!-- Security Notice -->
          <div class="flex items-start gap-3 px-4 py-3 bg-amber-50 border border-amber-100 rounded-lg">
            <svg class="w-4 h-4 text-amber-500 flex-shrink-0 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126zM12 15.75h.007v.008H12v-.008z" />
            </svg>
            <div class="text-[12px] text-amber-800">
              <strong>安全提示：</strong>凭据以脱敏形式显示，编辑时需重新输入完整值。空字段不会覆盖已有配置。
            </div>
          </div>

          {#if credentialsLoading}
            <div class="flex items-center justify-center h-32">
              <div class="w-6 h-6 border-2 border-gray-200 border-t-gray-900 rounded-full animate-spin"></div>
            </div>
          {:else}
            <!-- Provider Cards -->
            <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
              {#each providersConfig.providers || [] as provider}
                <div class="bg-white rounded-xl border border-gray-100 overflow-hidden">
                  <div class="px-5 py-4 border-b border-gray-100 flex items-center justify-between">
                    <h3 class="text-[14px] font-semibold text-gray-900">{provider.name}</h3>
                    {#if editingProvider === provider.name}
                      <div class="flex gap-2">
                        <button 
                          class="px-3 py-1 text-[12px] font-medium text-gray-600 bg-gray-100 rounded-md hover:bg-gray-200 transition-colors"
                          on:click={cancelEditProvider}
                        >取消</button>
                        <button 
                          class="px-3 py-1 text-[12px] font-medium text-white bg-emerald-500 rounded-md hover:bg-emerald-600 transition-colors disabled:opacity-50"
                          on:click={() => saveProviderCredentials(provider.name)}
                          disabled={credentialsSaving[provider.name]}
                        >
                          {credentialsSaving[provider.name] ? '保存中...' : '保存'}
                        </button>
                      </div>
                    {:else}
                      <button 
                        class="px-3 py-1 text-[12px] font-medium text-blue-600 bg-blue-50 rounded-md hover:bg-blue-100 transition-colors"
                        on:click={() => startEditProvider(provider)}
                      >编辑</button>
                    {/if}
                  </div>
                  <div class="p-5 space-y-3">
                    {#each Object.entries(provider.fields) as [key, value]}
                      <div>
                        <label class="block text-[11px] font-medium text-gray-500 mb-1">
                          {getFieldLabel(key)}
                          {#if provider.hasSecrets && provider.hasSecrets[key]}
                            <span class="ml-1 text-amber-500">🔒</span>
                          {/if}
                        </label>
                        {#if editingProvider === provider.name}
                          {#if isSecretField(key)}
                            <input 
                              type="password"
                              placeholder="输入新值覆盖"
                              class="w-full h-9 px-3 text-[12px] bg-gray-50 border-0 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow font-mono"
                              bind:value={editFields[key]}
                            />
                          {:else}
                            <input 
                              type="text"
                              placeholder={value || '未设置'}
                              class="w-full h-9 px-3 text-[12px] bg-gray-50 border-0 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow font-mono"
                              bind:value={editFields[key]}
                            />
                          {/if}
                        {:else}
                          <div class="h-9 px-3 flex items-center text-[12px] bg-gray-50 rounded-lg font-mono {value ? 'text-gray-900' : 'text-gray-400'}">
                            {value || '未设置'}
                          </div>
                        {/if}
                      </div>
                    {/each}
                  </div>
                </div>
              {:else}
                <div class="col-span-full py-16 text-center">
                  <svg class="w-10 h-10 mx-auto mb-3 text-gray-300" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M15.75 5.25a3 3 0 013 3m3 0a6 6 0 01-7.029 5.912c-.563-.097-1.159.026-1.563.43L10.5 17.25H8.25v2.25H6v2.25H2.25v-2.818c0-.597.237-1.17.659-1.591l6.499-6.499c.404-.404.527-1 .43-1.563A6 6 0 1121.75 8.25z" />
                  </svg>
                  <p class="text-[13px] text-gray-400">点击"加载配置"查看凭据</p>
                </div>
              {/each}
            </div>
          {/if}
        </div>
      {/if}
    </main>
  </div>
</div>

<!-- Delete Confirmation Modal -->
{#if deleteConfirm.show}
  <div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" on:click={cancelDelete}>
    <div class="bg-white rounded-xl shadow-xl max-w-sm w-full mx-4 overflow-hidden" on:click|stopPropagation>
      <div class="px-6 py-5">
        <div class="flex items-center gap-3 mb-3">
          <div class="w-10 h-10 rounded-full bg-red-100 flex items-center justify-center">
            <svg class="w-5 h-5 text-red-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
            </svg>
          </div>
          <div>
            <h3 class="text-[15px] font-semibold text-gray-900">确认删除</h3>
            <p class="text-[13px] text-gray-500">此操作不可撤销</p>
          </div>
        </div>
        <p class="text-[13px] text-gray-600">
          确定要删除场景 <span class="font-medium text-gray-900">"{deleteConfirm.caseName}"</span> 吗？
        </p>
      </div>
      <div class="px-6 py-4 bg-gray-50 flex justify-end gap-2">
        <button 
          class="px-4 py-2 text-[13px] font-medium text-gray-700 bg-white border border-gray-200 rounded-lg hover:bg-gray-50 transition-colors"
          on:click={cancelDelete}
        >取消</button>
        <button 
          class="px-4 py-2 text-[13px] font-medium text-white bg-red-600 rounded-lg hover:bg-red-700 transition-colors"
          on:click={confirmDelete}
        >删除</button>
      </div>
    </div>
  </div>
{/if}

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
