<script>
  import { onMount } from 'svelte';
  import { GetMCPStatus, StartMCPServer, StopMCPServer, GetActiveProfile } from '../../../wailsjs/go/main/App.js';

  let { t, onTabChange = () => {} } = $props();
  let mcpStatus = $state({ running: false, mode: '', address: '', protocolVersion: '' });
  let mcpForm = $state({ mode: 'sse', address: 'localhost:8080' });
  let mcpLoading = $state(false);
  let error = $state('');
  let successMessage = $state('');

  // AI Configuration state (loaded from Profile)
  let aiConfig = $state({
    provider: 'openai',
    apiKey: '',
    baseUrl: '',
    model: ''
  });
  let aiConfigLoading = $state(false);
  let hasAIConfig = $state(false);

  // Provider presets
  const providerPresets = {
    openai: {
      name: 'OpenAI API 兼容',
      nameEn: 'OpenAI API Compatible',
      baseUrl: 'https://api.openai.com/v1',
      defaultModel: 'gpt-4o'
    },
    anthropic: {
      name: 'Anthropic API 兼容',
      nameEn: 'Anthropic API Compatible',
      baseUrl: 'https://api.anthropic.com',
      defaultModel: 'claude-sonnet-4-20250514'
    }
  };

  const toolCategories = [
    { key: 'templateTools', label: '模板管理', labelEn: 'Template', tools: ['list_templates', 'search_templates', 'pull_template', 'get_template_info', 'get_template_files', 'delete_template', 'save_template_files'] },
    { key: 'deployTools', label: '场景部署', labelEn: 'Deploy', tools: ['plan_case', 'start_case', 'stop_case', 'kill_case', 'list_cases', 'get_case_status', 'get_case_outputs'] },
    { key: 'remoteTools', label: '远程操作', labelEn: 'Remote', tools: ['exec_command', 'get_ssh_info', 'upload_file', 'download_file'] },
    { key: 'userdataTools', label: 'Userdata', labelEn: 'Userdata', tools: ['list_userdata_templates', 'exec_userdata'] },
    { key: 'systemTools', label: '系统/其他', labelEn: 'System', tools: ['get_config', 'validate_config', 'ask_user'] },
  ];

  onMount(() => {
    loadMCPStatus();
    loadAIConfig();
  });

  async function loadMCPStatus() {
    try {
      mcpStatus = await GetMCPStatus();
    } catch (e) {
      console.error('Failed to load MCP status:', e);
    }
  }

  async function loadAIConfig() {
    aiConfigLoading = true;
    try {
      const profile = await GetActiveProfile();
      if (profile && profile.aiConfig) {
        aiConfig = {
          provider: profile.aiConfig.provider || 'openai',
          apiKey: profile.aiConfig.apiKey || '',
          baseUrl: profile.aiConfig.baseUrl || providerPresets[profile.aiConfig.provider || 'openai']?.baseUrl || '',
          model: profile.aiConfig.model || providerPresets[profile.aiConfig.provider || 'openai']?.defaultModel || ''
        };
        hasAIConfig = !!(aiConfig.apiKey && aiConfig.baseUrl && aiConfig.model);
      } else {
        const preset = providerPresets['openai'];
        aiConfig = {
          provider: 'openai',
          apiKey: '',
          baseUrl: preset.baseUrl,
          model: preset.defaultModel
        };
        hasAIConfig = false;
      }
    } catch (e) {
      console.error('Failed to load AI config:', e);
      hasAIConfig = false;
    } finally {
      aiConfigLoading = false;
    }
  }

  function isAIConfigured() {
    return hasAIConfig && aiConfig.apiKey && aiConfig.baseUrl && aiConfig.model;
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

  function getProviderDisplayName(provider) {
    const preset = providerPresets[provider];
    if (!preset) return provider;
    if (t && (t.openaiCompatible || '').includes('OpenAI')) {
      return preset.nameEn || preset.name;
    }
    return preset.name;
  }

</script>

<div class="space-y-4 sm:space-y-5">
  <!-- Error display -->
  {#if error}
    <div class="flex items-center gap-3 px-3 sm:px-4 py-2.5 sm:py-3 bg-red-50 border border-red-100 rounded-lg">
      <svg class="w-4 h-4 text-red-500 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
        <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z" />
      </svg>
      <span class="text-[12px] sm:text-[13px] text-red-700 flex-1">{error}</span>
      <button class="text-red-400 hover:text-red-600 cursor-pointer" onclick={() => error = ''} aria-label={t.close}>
        <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" /></svg>
      </button>
    </div>
  {/if}
  
  {#if successMessage}
    <div class="flex items-center gap-3 px-3 sm:px-4 py-2.5 sm:py-3 bg-green-50 border border-green-100 rounded-lg">
      <svg class="w-4 h-4 text-green-500 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
        <path stroke-linecap="round" stroke-linejoin="round" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
      </svg>
      <span class="text-[12px] sm:text-[13px] text-green-700 flex-1">{successMessage}</span>
      <button class="text-green-400 hover:text-green-600 cursor-pointer" onclick={() => successMessage = ''} aria-label={t.close}>
        <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" /></svg>
      </button>
    </div>
  {/if}

  <!-- Top row: AI Config status + AI Chat entry -->
  <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
    <!-- AI Configuration Status (compact) -->
    <div class="bg-white rounded-xl border border-gray-100 p-4 sm:p-5">
      <div class="flex items-center justify-between mb-3">
        <div class="flex items-center gap-2">
          <svg class="w-4 h-4 text-red-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M9.75 3.104v5.714a2.25 2.25 0 01-.659 1.591L5 14.5M9.75 3.104c-.251.023-.501.05-.75.082m.75-.082a24.301 24.301 0 014.5 0m0 0v5.714c0 .597.237 1.17.659 1.591L19.8 15.3M14.25 3.104c.251.023.501.05.75.082M19.8 15.3l-1.57.393A9.065 9.065 0 0112 15a9.065 9.065 0 00-6.23-.693L5 14.5m14.8.8l1.402 1.402c1.232 1.232.65 3.318-1.067 3.611A48.309 48.309 0 0112 21c-2.773 0-5.491-.235-8.135-.687-1.718-.293-2.3-2.379-1.067-3.61L5 14.5" />
          </svg>
          <h3 class="text-[13px] font-semibold text-gray-900">{t.aiConfig || 'AI 配置'}</h3>
        </div>
        {#if aiConfigLoading}
          <svg class="w-4 h-4 animate-spin text-gray-400" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path></svg>
        {:else if isAIConfigured()}
          <span class="inline-flex items-center gap-1.5 px-2 py-0.5 bg-emerald-50 text-emerald-600 text-[11px] font-medium rounded-full">
            <span class="w-1.5 h-1.5 rounded-full bg-emerald-500"></span>
            {t.aiConfigured || '已配置'}
          </span>
        {:else}
          <span class="inline-flex items-center gap-1.5 px-2 py-0.5 bg-amber-50 text-amber-600 text-[11px] font-medium rounded-full">
            <span class="w-1.5 h-1.5 rounded-full bg-amber-400"></span>
            {t.aiNotConfigured || '未配置'}
          </span>
        {/if}
      </div>

      {#if isAIConfigured()}
        <div class="flex items-center gap-4 text-[12px] text-gray-600 mb-3">
          <span>{getProviderDisplayName(aiConfig.provider)}</span>
          <span class="text-gray-300">·</span>
          <span class="font-mono text-gray-800">{aiConfig.model}</span>
        </div>
      {:else}
        <p class="text-[12px] text-gray-500 mb-3">{t.aiNotConfiguredHint || '请先在凭据管理页面配置 AI API Key'}</p>
      {/if}

      <button 
        onclick={() => onTabChange('credentials')}
        class="inline-flex items-center gap-1.5 text-[12px] text-gray-500 hover:text-gray-700 font-medium cursor-pointer transition-colors"
      >
        <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M10.5 6h9.75M10.5 6a1.5 1.5 0 11-3 0m3 0a1.5 1.5 0 10-3 0M3.75 6H7.5m3 12h9.75m-9.75 0a1.5 1.5 0 01-3 0m3 0a1.5 1.5 0 00-3 0m-3.75 0H7.5m9-6h3.75m-3.75 0a1.5 1.5 0 01-3 0m3 0a1.5 1.5 0 00-3 0m-9.75 0h9.75" /></svg>
        {isAIConfigured() ? (t.goToCredentials || '前往凭据管理') : (t.configureAI || '配置 AI')} →
      </button>
    </div>

    <!-- AI Chat entry card -->
    <div class="bg-white rounded-xl border border-gray-100 p-4 sm:p-5">
      <div class="flex items-center gap-2 mb-3">
        <svg class="w-4 h-4 text-blue-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M20.25 8.511c.884.284 1.5 1.128 1.5 2.097v4.286c0 1.136-.847 2.1-1.98 2.193-.34.027-.68.052-1.02.072v3.091l-3-3c-1.354 0-2.694-.055-4.02-.163a2.115 2.115 0 01-.825-.242m9.345-8.334a2.126 2.126 0 00-.476-.095 48.64 48.64 0 00-8.048 0c-1.131.094-1.976 1.057-1.976 2.192v4.286c0 .837.46 1.58 1.155 1.951m9.345-8.334V6.637c0-1.621-1.152-3.026-2.76-3.235A48.455 48.455 0 0011.25 3c-2.115 0-4.198.137-6.24.402-1.608.209-2.76 1.614-2.76 3.235v6.226c0 1.621 1.152 3.026 2.76 3.235.577.075 1.157.14 1.74.194V21l4.155-4.155" />
        </svg>
        <h3 class="text-[13px] font-semibold text-gray-900">{t.aiChat || 'AI 对话'}</h3>
      </div>
      <p class="text-[12px] text-gray-500 mb-3">{t.aiChatRedirectHint || 'AI 模板生成、场景推荐、成本优化等功能已迁移至 AI 对话页面'}</p>
      <button 
        onclick={() => onTabChange('aiChat')}
        class="inline-flex items-center gap-1.5 text-[12px] text-gray-500 hover:text-gray-700 font-medium cursor-pointer transition-colors"
      >
        <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M20.25 8.511c.884.284 1.5 1.128 1.5 2.097v4.286c0 1.136-.847 2.1-1.98 2.193-.34.027-.68.052-1.02.072v3.091l-3-3c-1.354 0-2.694-.055-4.02-.163a2.115 2.115 0 01-.825-.242m9.345-8.334a2.126 2.126 0 00-.476-.095 48.64 48.64 0 00-8.048 0c-1.131.094-1.976 1.057-1.976 2.192v4.286c0 .837.46 1.58 1.155 1.951m9.345-8.334V6.637c0-1.621-1.152-3.026-2.76-3.235A48.455 48.455 0 0011.25 3c-2.115 0-4.198.137-6.24.402-1.608.209-2.76 1.614-2.76 3.235v6.226c0 1.621 1.152 3.026 2.76 3.235.577.075 1.157.14 1.74.194V21l4.155-4.155" /></svg>
        {t.goToAIChat || '前往 AI 对话'} →
      </button>
    </div>
  </div>

  <!-- MCP Server Card -->
  <div class="bg-white rounded-xl border border-gray-100 p-4 sm:p-5">
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 sm:gap-0 mb-4">
      <div class="flex items-center gap-2">
        <svg class="w-4 h-4 text-gray-700" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M9.813 15.904L9 18.75l-.813-2.846a4.5 4.5 0 00-3.09-3.09L2.25 12l2.846-.813a4.5 4.5 0 003.09-3.09L9 5.25l.813 2.846a4.5 4.5 0 003.09 3.09L15.75 12l-2.846.813a4.5 4.5 0 00-3.09 3.09z" />
        </svg>
        <h3 class="text-[13px] sm:text-[14px] font-semibold text-gray-900">{t.mcpServer}</h3>
        <span class="text-[11px] text-gray-400">{t.mcpDesc}</span>
      </div>
      <div class="flex items-center gap-2">
        {#if mcpStatus.running}
          <span class="inline-flex items-center gap-1.5 px-2.5 py-1 bg-emerald-50 text-emerald-600 text-[11px] font-medium rounded-full">
            <span class="w-1.5 h-1.5 rounded-full bg-emerald-500 animate-pulse"></span>
            {t.running}
          </span>
          <button 
            class="h-8 px-3 bg-red-500 text-white text-[12px] font-medium rounded-lg hover:bg-red-600 transition-colors disabled:opacity-50 cursor-pointer"
            onclick={handleStopMCP}
            disabled={mcpLoading}
          >
            {mcpLoading ? t.stoppingServer : t.stopServer}
          </button>
        {:else}
          <span class="inline-flex items-center gap-1.5 px-2.5 py-1 bg-gray-50 text-gray-500 text-[11px] font-medium rounded-full">
            <span class="w-1.5 h-1.5 rounded-full bg-gray-400"></span>
            {t.stopped}
          </span>
          <button 
            class="h-8 px-3 bg-gray-900 text-white text-[12px] font-medium rounded-lg hover:bg-gray-800 transition-colors disabled:opacity-50 cursor-pointer inline-flex items-center gap-1.5"
            onclick={handleStartMCP}
            disabled={mcpLoading}
          >
            {#if mcpLoading}
              <svg class="animate-spin h-3.5 w-3.5" viewBox="0 0 24 24" fill="none"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path></svg>
            {/if}
            {mcpLoading ? t.startingServer : t.startServer}
          </button>
        {/if}
      </div>
    </div>

    {#if mcpStatus.running}
      <div class="bg-gray-50 rounded-lg p-3 sm:p-4">
        <div class="grid grid-cols-2 sm:grid-cols-4 gap-3 text-[11px] sm:text-[12px]">
          <div>
            <span class="text-gray-500">{t.transportMode}</span>
            <p class="font-medium text-gray-900 mt-0.5">Streamable HTTP</p>
          </div>
          <div>
            <span class="text-gray-500">{t.listenAddr}</span>
            <p class="font-mono font-medium text-gray-900 mt-0.5 break-all">{mcpStatus.address || '-'}</p>
          </div>
          <div>
            <span class="text-gray-500">{t.protocolVersion}</span>
            <p class="font-medium text-gray-900 mt-0.5">{mcpStatus.protocolVersion}</p>
          </div>
          <div>
            <span class="text-gray-500">{t.msgEndpoint}</span>
            <p class="font-mono font-medium text-gray-900 mt-0.5 text-[10px] break-all">http://{mcpStatus.address}/mcp</p>
          </div>
        </div>
      </div>
    {:else}
      <div class="flex items-center gap-3">
        <div class="flex items-center gap-2 text-[12px] text-gray-500">
          <span class="px-2 py-0.5 bg-gray-100 rounded text-[11px] font-medium text-gray-600">Streamable HTTP</span>
        </div>
        <div class="flex-1">
          <input 
            type="text" 
            placeholder="localhost:8080" 
            class="w-full h-9 px-3 text-[12px] bg-gray-50 border-0 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow font-mono"
            bind:value={mcpForm.address} 
          />
        </div>
      </div>
    {/if}
  </div>

  <!-- MCP Tools (categorized) -->
  <div class="bg-white rounded-xl border border-gray-100 p-4 sm:p-5">
    <div class="flex items-center justify-between mb-4">
      <h3 class="text-[13px] font-semibold text-gray-900">{t.availableTools || '可用工具'}</h3>
      <span class="text-[11px] text-gray-400">{toolCategories.reduce((sum, c) => sum + c.tools.length, 0)} tools</span>
    </div>
    <p class="text-[12px] text-gray-500 mb-4">{t.mcpInfo}</p>
    <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
      {#each toolCategories as cat}
        <div class="bg-gray-50 rounded-lg p-3">
          <div class="text-[11px] font-medium text-gray-500 uppercase tracking-wide mb-2">{t[cat.key] || cat.label}</div>
          <div class="space-y-1.5">
            {#each cat.tools as tool}
              <div class="flex items-center gap-2 text-[12px] text-gray-700">
                <span class="w-1 h-1 rounded-full bg-gray-400 flex-shrink-0"></span>
                <span class="font-mono text-[11px]">{tool}</span>
              </div>
            {/each}
          </div>
        </div>
      {/each}
    </div>
  </div>
</div>
