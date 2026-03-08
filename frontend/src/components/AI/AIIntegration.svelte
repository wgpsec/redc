<script>
  import { onMount } from 'svelte';
  import { GetMCPStatus, StartMCPServer, StopMCPServer, GetActiveProfile } from '../../../wailsjs/wailsjs/go/main/App.js';

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
  let aiConfigSaved = $state(false);
  let showApiKey = $state(false);
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
    // Check if current language is English
    if (t && (t.openaiCompatible || '').includes('OpenAI')) {
      return preset.nameEn || preset.name;
    }
    return preset.name;
  }

</script>

<div class="w-full max-w-2xl mx-auto space-y-4 sm:space-y-5">
  <!-- Error display -->
  {#if error}
    <div class="flex items-center gap-3 px-3 sm:px-4 py-2.5 sm:py-3 bg-red-50 border border-red-100 rounded-lg">
      <svg class="w-4 h-4 text-red-500 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
        <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z" />
      </svg>
      <span class="text-[12px] sm:text-[13px] text-red-700 flex-1">{error}</span>
      <button class="text-red-400 hover:text-red-600 cursor-pointer" onclick={() => error = ''} aria-label="关闭错误">
        <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
        </svg>
      </button>
    </div>
  {/if}
  
  <!-- Success display -->
  {#if successMessage}
    <div class="flex items-center gap-3 px-3 sm:px-4 py-2.5 sm:py-3 bg-green-50 border border-green-100 rounded-lg">
      <svg class="w-4 h-4 text-green-500 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
        <path stroke-linecap="round" stroke-linejoin="round" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
      </svg>
      <span class="text-[12px] sm:text-[13px] text-green-700 flex-1">{successMessage}</span>
      <button class="text-green-400 hover:text-green-600 cursor-pointer" onclick={() => successMessage = ''} aria-label="关闭提示">
        <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
        </svg>
      </button>
    </div>
  {/if}

  <!-- AI Configuration Status Card -->
  <div class="bg-white rounded-xl border border-gray-100 p-4 sm:p-5">
    <div class="flex items-center gap-3 mb-4">
      <div class="w-9 h-9 sm:w-10 sm:h-10 rounded-lg bg-rose-600 flex items-center justify-center">
        <svg class="w-4.5 h-4.5 sm:w-5 sm:h-5 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M9.75 3.104v5.714a2.25 2.25 0 01-.659 1.591L5 14.5M9.75 3.104c-.251.023-.501.05-.75.082m.75-.082a24.301 24.301 0 014.5 0m0 0v5.714c0 .597.237 1.17.659 1.591L19.8 15.3M14.25 3.104c.251.023.501.05.75.082M19.8 15.3l-1.57.393A9.065 9.065 0 0112 15a9.065 9.065 0 00-6.23-.693L5 14.5m14.8.8l1.402 1.402c1.232 1.232.65 3.318-1.067 3.611A48.309 48.309 0 0112 21c-2.773 0-5.491-.235-8.135-.687-1.718-.293-2.3-2.379-1.067-3.61L5 14.5" />
        </svg>
      </div>
      <div>
        <h2 class="text-[13px] sm:text-[14px] font-semibold text-gray-900">{t.aiConfig || 'AI 配置'}</h2>
        <p class="text-[11px] sm:text-[12px] text-gray-500">{t.aiConfigStatusDesc || '当前 AI 服务配置状态'}</p>
      </div>
    </div>

    {#if aiConfigLoading}
      <div class="flex items-center justify-center py-6">
        <svg class="w-5 h-5 animate-spin text-gray-400" fill="none" viewBox="0 0 24 24">
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
        </svg>
      </div>
    {:else if isAIConfigured()}
      <div class="bg-emerald-50 rounded-lg p-3 sm:p-4">
        <div class="flex items-center gap-2 mb-3">
          <svg class="w-4 h-4 text-emerald-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
          <span class="text-[12px] sm:text-[13px] font-medium text-emerald-700">{t.aiConfigured || 'AI 服务已配置'}</span>
        </div>
        <div class="grid grid-cols-2 gap-3 text-[11px] sm:text-[12px]">
          <div>
            <span class="text-gray-500">{t.aiProvider || '服务商'}</span>
            <p class="font-medium text-gray-900 mt-0.5">{getProviderDisplayName(aiConfig.provider)}</p>
          </div>
          <div>
            <span class="text-gray-500">{t.aiModel || '模型'}</span>
            <p class="font-medium text-gray-900 mt-0.5">{aiConfig.model}</p>
          </div>
        </div>
      </div>
      <div class="mt-3">
        <button 
          onclick={() => onTabChange('credentials')}
          class="text-[11px] sm:text-[12px] text-blue-600 hover:text-blue-700 font-medium flex items-center gap-1 cursor-pointer"
        >
          {t.goToCredentials || '前往凭据管理'} →
        </button>
      </div>
    {:else}
      <div class="bg-amber-50 rounded-lg p-3 sm:p-4">
        <div class="flex items-center gap-2 mb-2">
          <svg class="w-4 h-4 text-amber-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126zM12 15.75h.007v.008H12v-.008z" />
          </svg>
          <span class="text-[12px] sm:text-[13px] font-medium text-amber-700">{t.aiNotConfigured || 'AI 服务未配置'}</span>
        </div>
        <p class="text-[11px] sm:text-[12px] text-amber-600">{t.aiNotConfiguredHint || '请先在凭据管理页面配置 AI API Key'}</p>
      </div>
      <div class="mt-3">
        <button 
          class="inline-flex items-center gap-2 px-4 h-9 text-gray-700 bg-white border border-gray-300 hover:bg-gray-50 text-[12px] font-medium rounded-lg transition-colors cursor-pointer"
          onclick={() => onTabChange('credentials')}
        >
          <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M10.5 6h9.75M10.5 6a1.5 1.5 0 11-3 0m3 0a1.5 1.5 0 10-3 0M3.75 6H7.5m3 12h9.75m-9.75 0a1.5 1.5 0 01-3 0m3 0a1.5 1.5 0 00-3 0m-3.75 0H7.5m9-6h3.75m-3.75 0a1.5 1.5 0 01-3 0m3 0a1.5 1.5 0 00-3 0m-9.75 0h9.75" />
          </svg>
          {t.configureAI || '配置 AI'}
        </button>
      </div>
    {/if}
  </div>

  <!-- MCP Status Card -->
  <div class="bg-white rounded-xl border border-gray-100 p-4 sm:p-5">
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 sm:gap-0 mb-4">
      <div class="flex items-center gap-3">
        <div class="w-9 h-9 sm:w-10 sm:h-10 rounded-lg bg-rose-600 flex items-center justify-center">
          <svg class="w-4.5 h-4.5 sm:w-5 sm:h-5 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.813 15.904L9 18.75l-.813-2.846a4.5 4.5 0 00-3.09-3.09L2.25 12l2.846-.813a4.5 4.5 0 003.09-3.09L9 5.25l.813 2.846a4.5 4.5 0 003.09 3.09L15.75 12l-2.846.813a4.5 4.5 0 00-3.09 3.09z" />
          </svg>
        </div>
        <div>
          <h2 class="text-[13px] sm:text-[14px] font-semibold text-gray-900">{t.mcpServer}</h2>
          <p class="text-[11px] sm:text-[12px] text-gray-500">{t.mcpDesc}</p>
        </div>
      </div>
      <div class="flex items-center gap-2">
        {#if mcpStatus.running}
          <span class="inline-flex items-center gap-1.5 px-2.5 py-1 bg-emerald-50 text-emerald-600 text-[11px] sm:text-[12px] font-medium rounded-full">
            <span class="w-1.5 h-1.5 rounded-full bg-emerald-500 animate-pulse"></span>
            {t.running}
          </span>
        {:else}
          <span class="inline-flex items-center gap-1.5 px-2.5 py-1 bg-gray-50 text-gray-500 text-[11px] sm:text-[12px] font-medium rounded-full">
            <span class="w-1.5 h-1.5 rounded-full bg-gray-400"></span>
            {t.stopped}
          </span>
        {/if}
      </div>
    </div>

    {#if mcpStatus.running}
      <!-- Running status info -->
      <div class="bg-gray-50 rounded-lg p-3 sm:p-4 mb-4">
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-3 sm:gap-4 text-[11px] sm:text-[12px]">
          <div>
            <span class="text-gray-500">{t.transportMode}</span>
            <p class="font-medium text-gray-900 mt-0.5">SSE (HTTP)</p>
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
            <p class="font-mono font-medium text-gray-900 mt-0.5 text-[10px] sm:text-[11px] break-all">http://{mcpStatus.address}/message</p>
          </div>
        </div>
      </div>
      <button 
        class="w-full h-9 sm:h-10 bg-red-500 text-white text-[12px] sm:text-[13px] font-medium rounded-lg hover:bg-red-600 transition-colors disabled:opacity-50 cursor-pointer"
        onclick={handleStopMCP}
        disabled={mcpLoading}
      >
        {mcpLoading ? t.stoppingServer : t.stopServer}
      </button>
    {:else}
      <!-- Configuration form -->
      <div class="space-y-3 sm:space-y-4 mb-4">
        <div>
          <span class="block text-[11px] sm:text-[12px] font-medium text-gray-500 mb-1.5">{t.transportMode}</span>
          <div class="inline-flex items-center h-9 sm:h-10 px-3 sm:px-4 text-[12px] sm:text-[13px] font-medium rounded-lg border bg-white text-gray-700 border-gray-300">
            SSE (HTTP)
          </div>
        </div>
        <div>
          <label for="listenAddr" class="block text-[11px] sm:text-[12px] font-medium text-gray-500 mb-1.5">{t.listenAddr}</label>
          <input 
            id="listenAddr"
            type="text" 
            placeholder="localhost:8080" 
            class="w-full h-9 sm:h-10 px-3 text-[12px] sm:text-[13px] bg-gray-50 border-0 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow font-mono"
            bind:value={mcpForm.address} 
          />
        </div>
      </div>
      <button 
        class="w-full h-9 sm:h-10 text-gray-700 bg-white border border-gray-300 hover:bg-gray-50 text-[12px] sm:text-[13px] font-medium rounded-lg transition-colors disabled:opacity-50 cursor-pointer"
        onclick={handleStartMCP}
        disabled={mcpLoading}
      >
        {mcpLoading ? t.startingServer : t.startServer}
      </button>
    {/if}
  </div>

  <!-- MCP Info Card -->
  <div class="bg-white rounded-xl border border-gray-100 p-4 sm:p-5">
    <h3 class="text-[13px] sm:text-[14px] font-semibold text-gray-900 mb-3">{t.aboutMcp}</h3>
    <p class="text-[11px] sm:text-[12px] text-gray-600 leading-relaxed mb-4">
      {t.mcpInfo}
    </p>
    <div class="bg-gray-50 rounded-lg p-3 sm:p-4">
      <div class="text-[10px] sm:text-[11px] font-medium text-gray-500 uppercase tracking-wide mb-2">{t.availableTools}</div>
      <div class="grid grid-cols-2 sm:grid-cols-3 gap-2 text-[11px] sm:text-[12px]">
        <div class="flex items-center gap-2 text-gray-700">
          <span class="w-1 h-1 rounded-full bg-gray-400"></span>
          list_templates
        </div>
        <div class="flex items-center gap-2 text-gray-700">
          <span class="w-1 h-1 rounded-full bg-gray-400"></span>
          search_templates
        </div>
        <div class="flex items-center gap-2 text-gray-700">
          <span class="w-1 h-1 rounded-full bg-gray-400"></span>
          pull_template
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
        <div class="flex items-center gap-2 text-gray-700">
          <span class="w-1 h-1 rounded-full bg-gray-400"></span>
          get_ssh_info
        </div>
        <div class="flex items-center gap-2 text-gray-700">
          <span class="w-1 h-1 rounded-full bg-gray-400"></span>
          upload_file
        </div>
        <div class="flex items-center gap-2 text-gray-700">
          <span class="w-1 h-1 rounded-full bg-gray-400"></span>
          download_file
        </div>
        <div class="flex items-center gap-2 text-gray-700">
          <span class="w-1 h-1 rounded-full bg-gray-400"></span>
          get_template_info
        </div>
        <div class="flex items-center gap-2 text-gray-700">
          <span class="w-1 h-1 rounded-full bg-gray-400"></span>
          delete_template
        </div>
        <div class="flex items-center gap-2 text-gray-700">
          <span class="w-1 h-1 rounded-full bg-gray-400"></span>
          get_case_outputs
        </div>
        <div class="flex items-center gap-2 text-gray-700">
          <span class="w-1 h-1 rounded-full bg-gray-400"></span>
          get_config
        </div>
        <div class="flex items-center gap-2 text-gray-700">
          <span class="w-1 h-1 rounded-full bg-gray-400"></span>
          validate_config
        </div>
      </div>
    </div>
  </div>
</div>
