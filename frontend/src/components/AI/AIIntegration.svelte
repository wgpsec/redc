<script>
  import { onMount } from 'svelte';
  import { GetMCPStatus, StartMCPServer, StopMCPServer, GetActiveProfile, ListSkills, GetSkill, SaveCustomSkill, DeleteCustomSkill, FetchSkillsRegistry, InstallSkill, UpdateSkill, InstallAllSkills, UpdateAllSkills, GetSkillsDir } from '../../../wailsjs/go/main/App.js';
  import { toast } from '../../lib/toast.js';

  let { t, onTabChange = () => {} } = $props();
  let mcpStatus = $state({ running: false, mode: '', address: '', protocolVersion: '' });
  let mcpForm = $state({ mode: 'sse', address: 'localhost:8080' });
  let mcpLoading = $state(false);
  let error = $state('');
  let successMessage = $state('');
  let subTab = $state('overview');

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
    loadSkills();
    GetSkillsDir().then(d => skillsDir = d).catch(() => {});
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

  // Skills Knowledge Base state
  let skills = $state([]);
  let skillsLoading = $state(false);
  let skillsSearch = $state('');
  let skillsDir = $state('');
  let selectedSkill = $state(null);
  let selectedSkillLoading = $state(false);
  let showNewSkillForm = $state(false);
  let newSkillId = $state('');
  let newSkillContent = $state('');
  let savingSkill = $state(false);

  // Skills market state
  let registrySkills = $state([]);
  let registryLoading = $state(false);
  let registryError = $state('');
  let installingSkillId = $state('');

  async function loadSkills() {
    skillsLoading = true;
    try {
      skills = await ListSkills(skillsSearch) || [];
    } catch (e) {
      console.error('Failed to load skills:', e);
      skills = [];
    } finally {
      skillsLoading = false;
    }
  }

  async function viewSkill(id) {
    selectedSkillLoading = true;
    try {
      selectedSkill = await GetSkill(id);
    } catch (e) {
      toast.error(String(e));
    } finally {
      selectedSkillLoading = false;
    }
  }

  async function handleSaveCustomSkill() {
    if (!newSkillId.trim() || !newSkillContent.trim()) return;
    savingSkill = true;
    try {
      await SaveCustomSkill(newSkillId.trim(), newSkillContent);
      toast.success(t.skillSaved || 'Skill saved');
      showNewSkillForm = false;
      newSkillId = '';
      newSkillContent = '';
      await loadSkills();
    } catch (e) {
      toast.error(String(e));
    } finally {
      savingSkill = false;
    }
  }

  async function handleDeleteSkill(id) {
    try {
      await DeleteCustomSkill(id);
      toast.success(t.skillDeleted || 'Skill deleted');
      if (selectedSkill && selectedSkill.id === id) selectedSkill = null;
      await loadSkills();
    } catch (e) {
      toast.error(String(e));
    }
  }

  async function loadRegistry() {
    registryLoading = true;
    registryError = '';
    try {
      registrySkills = await FetchSkillsRegistry() || [];
    } catch (e) {
      registryError = String(e);
      registrySkills = [];
    } finally {
      registryLoading = false;
    }
  }

  async function handleInstallSkill(skill) {
    installingSkillId = skill.id;
    try {
      await InstallSkill(skill.id, skill.url);
      toast.success((t.skillInstalled || 'Skill installed: ') + skill.name);
      await Promise.all([loadSkills(), loadRegistry()]);
    } catch (e) {
      toast.error(String(e));
    } finally {
      installingSkillId = '';
    }
  }

  async function handleUpdateSkill(skill) {
    installingSkillId = skill.id;
    try {
      await UpdateSkill(skill.id, skill.url, skill.sha256 || '');
      toast.success((t.skillUpdated || 'Skill updated: ') + skill.name);
      await Promise.all([loadSkills(), loadRegistry()]);
    } catch (e) {
      toast.error(String(e));
    } finally {
      installingSkillId = '';
    }
  }

  let batchInstalling = $state(false);
  let batchUpdating = $state(false);

  async function handleInstallAll() {
    const uninstalled = registrySkills.filter(s => !s.installed);
    if (uninstalled.length === 0) {
      toast.info(t.noSkillsToInstall || 'All skills already installed');
      return;
    }
    batchInstalling = true;
    try {
      const count = await InstallAllSkills();
      toast.success((t.installAllSuccess || 'Installed %d skills').replace('%d', count));
      await Promise.all([loadSkills(), loadRegistry()]);
    } catch (e) {
      toast.error(String(e));
      await Promise.all([loadSkills(), loadRegistry()]);
    } finally {
      batchInstalling = false;
    }
  }

  async function handleUpdateAll() {
    const updatable = registrySkills.filter(s => s.installed && s.hasUpdate);
    if (updatable.length === 0) {
      toast.info(t.noSkillsToUpdate || 'No skills to update');
      return;
    }
    batchUpdating = true;
    try {
      const count = await UpdateAllSkills();
      toast.success((t.updateAllSuccess || 'Updated %d skills').replace('%d', count));
      await Promise.all([loadSkills(), loadRegistry()]);
    } catch (e) {
      toast.error(String(e));
      await Promise.all([loadSkills(), loadRegistry()]);
    } finally {
      batchUpdating = false;
    }
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

  <!-- Segmented Control -->
  <div class="flex items-center gap-3">
    <div class="flex gap-1 bg-gray-100 rounded-lg p-1">
      <button
        onclick={() => subTab = 'overview'}
        class="px-3 py-1 text-[12px] rounded-md transition-colors cursor-pointer {subTab === 'overview' ? 'bg-white text-gray-900 shadow-sm font-medium' : 'text-gray-500 hover:text-gray-700'}"
      >{t.aiOverview || '概览'}</button>
      <button
        onclick={() => { subTab = 'skills'; loadSkills(); }}
        class="px-3 py-1 text-[12px] rounded-md transition-colors cursor-pointer {subTab === 'skills' ? 'bg-white text-gray-900 shadow-sm font-medium' : 'text-gray-500 hover:text-gray-700'}"
      >{t.skillsKnowledgeBase || 'Skills 技能库'}</button>
      <button
        onclick={() => { subTab = 'market'; loadRegistry(); }}
        class="px-3 py-1 text-[12px] rounded-md transition-colors cursor-pointer {subTab === 'market' ? 'bg-white text-gray-900 shadow-sm font-medium' : 'text-gray-500 hover:text-gray-700'}"
      >{t.skillsMarket || 'Skills 市场'}</button>
    </div>
  </div>

  {#if subTab === 'overview'}
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
  {/if}

  {#if subTab === 'skills'}
  <!-- Skills Knowledge Base -->
  <div class="bg-white rounded-xl border border-gray-100 p-4 sm:p-5">
    <div class="flex items-center justify-between mb-4">
      <div class="flex items-center gap-2">
        <svg class="w-4 h-4 text-purple-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M12 6.042A8.967 8.967 0 006 3.75c-1.052 0-2.062.18-3 .512v14.25A8.987 8.987 0 016 18c2.305 0 4.408.867 6 2.292m0-14.25a8.966 8.966 0 016-2.292c1.052 0 2.062.18 3 .512v14.25A8.987 8.987 0 0018 18a8.967 8.967 0 00-6 2.292m0-14.25v14.25" />
        </svg>
        <h3 class="text-[13px] font-semibold text-gray-900">{t.skillsKnowledgeBase || 'Skills 技能库'}</h3>
        <span class="text-[11px] text-gray-400">{skills.length} {t.skillItems || 'items'}</span>
      </div>
      <div class="flex items-center gap-2">
        <button
          onclick={() => { showNewSkillForm = !showNewSkillForm; selectedSkill = null; }}
          class="h-7 px-2.5 bg-gray-900 text-white text-[11px] font-medium rounded-lg hover:bg-gray-800 transition-colors cursor-pointer inline-flex items-center gap-1"
        >
          <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M12 4.5v15m7.5-7.5h-15" /></svg>
          {t.skillAdd || '添加'}
        </button>
      </div>
    </div>

    <p class="text-[12px] text-gray-500 mb-3">{t.skillsDesc || 'Skills 为 AI Agent 提供领域知识，Agent 模式下会根据对话内容自动匹配相关 Skill 注入上下文。'}</p>

    {#if skillsDir}
      <div class="flex items-center gap-1.5 text-[10px] text-gray-400 mb-3">
        <svg class="w-3 h-3 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M2.25 12.75V12A2.25 2.25 0 014.5 9.75h15A2.25 2.25 0 0121.75 12v.75m-8.69-6.44l-2.12-2.12a1.5 1.5 0 00-1.061-.44H4.5A2.25 2.25 0 002.25 6v12a2.25 2.25 0 002.25 2.25h15A2.25 2.25 0 0021.75 18V9a2.25 2.25 0 00-2.25-2.25h-5.379a1.5 1.5 0 01-1.06-.44z" /></svg>
        <span class="font-mono select-all">{skillsDir}</span>
      </div>
    {/if}

    <!-- Search -->
    <div class="mb-3">
      <input
        type="text"
        placeholder={t.skillSearch || '搜索 Skills...'}
        class="w-full h-8 px-3 text-[12px] bg-gray-50 border-0 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow"
        bind:value={skillsSearch}
        oninput={() => loadSkills()}
      />
    </div>

    <!-- Skills list -->
    {#if skillsLoading}
      <div class="flex items-center justify-center py-6">
        <svg class="w-5 h-5 animate-spin text-gray-400" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path></svg>
      </div>
    {:else if skills.length === 0}
      <p class="text-center text-[12px] text-gray-400 py-4">{t.skillsEmpty || '暂无 Skills'}</p>
    {:else}
      <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-2 mb-3">
        {#each skills as skill}
          <div
            role="button" tabindex="0"
            onclick={() => viewSkill(skill.id)}
            onkeydown={(e) => { if (e.key === 'Enter') viewSkill(skill.id); }}
            class="text-left bg-gray-50 hover:bg-gray-100 rounded-lg p-3 transition-colors cursor-pointer border {selectedSkill && selectedSkill.id === skill.id ? 'border-gray-900' : 'border-transparent'}"
          >
            <div class="flex items-center justify-between mb-1">
              <span class="text-[12px] font-medium text-gray-900 truncate">{skill.name}</span>
              <button
                onclick={(e) => { e.stopPropagation(); handleDeleteSkill(skill.id); }}
                class="text-gray-400 hover:text-red-500 cursor-pointer p-0.5"
                title={t.delete || '删除'}
              >
                <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M14.74 9l-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 01-2.244 2.077H8.084a2.25 2.25 0 01-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 00-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 013.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 00-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 00-7.5 0" /></svg>
              </button>
            </div>
            <p class="text-[11px] text-gray-500 line-clamp-2">{skill.description}</p>
            {#if skill.tags && skill.tags.length > 0}
              <div class="flex flex-wrap gap-1 mt-1.5">
                {#each skill.tags.slice(0, 4) as tag}
                  <span class="px-1.5 py-0.5 bg-gray-200 text-gray-600 text-[10px] rounded">{tag}</span>
                {/each}
                {#if skill.tags.length > 4}
                  <span class="text-[10px] text-gray-400">+{skill.tags.length - 4}</span>
                {/if}
              </div>
            {/if}
          </div>
        {/each}
      </div>
    {/if}

    <!-- Skill detail view -->
    {#if selectedSkillLoading}
      <div class="flex items-center justify-center py-4">
        <svg class="w-4 h-4 animate-spin text-gray-400" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path></svg>
      </div>
    {:else if selectedSkill}
      <div class="bg-gray-50 rounded-lg p-3 sm:p-4">
        <div class="flex items-center justify-between mb-2">
          <h4 class="text-[12px] font-semibold text-gray-900">{selectedSkill.name}</h4>
          <button onclick={() => selectedSkill = null} class="text-gray-400 hover:text-gray-600 cursor-pointer">
            <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" /></svg>
          </button>
        </div>
        <pre class="text-[11px] text-gray-700 whitespace-pre-wrap font-mono bg-white rounded p-3 max-h-64 overflow-y-auto border border-gray-100">{selectedSkill.content}</pre>
      </div>
    {/if}

    <!-- New skill form -->
    {#if showNewSkillForm}
      <div class="bg-gray-50 rounded-lg p-3 sm:p-4 mt-3">
        <h4 class="text-[12px] font-semibold text-gray-900 mb-3">{t.skillAddNew || '添加自定义 Skill'}</h4>
        <div class="space-y-3">
          <div>
            <label class="text-[11px] text-gray-500 block mb-1">ID ({t.skillIdHint || '英文标识符，如 my-skill'})</label>
            <input
              type="text"
              placeholder="my-custom-skill"
              class="w-full h-8 px-3 text-[12px] bg-white border border-gray-200 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow font-mono"
              bind:value={newSkillId}
            />
          </div>
          <div>
            <label class="text-[11px] text-gray-500 block mb-1">{t.skillContent || '内容'} (Markdown, {t.skillFrontmatterHint || '首行可用 YAML frontmatter 定义 name/description/tags'})</label>
            <textarea
              rows="10"
              placeholder={"---\nname: My Skill\ndescription: Description of the skill\ntags: tag1, tag2\n---\n# Skill Content\n\nYour knowledge base content here..."}
              class="w-full px-3 py-2 text-[12px] bg-white border border-gray-200 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow font-mono resize-y"
              bind:value={newSkillContent}
            ></textarea>
          </div>
          <div class="flex items-center gap-2">
            <button
              onclick={handleSaveCustomSkill}
              disabled={savingSkill || !newSkillId.trim() || !newSkillContent.trim()}
              class="h-8 px-4 bg-gray-900 text-white text-[12px] font-medium rounded-lg hover:bg-gray-800 transition-colors disabled:opacity-50 cursor-pointer"
            >
              {savingSkill ? '...' : (t.save || '保存')}
            </button>
            <button
              onclick={() => { showNewSkillForm = false; newSkillId = ''; newSkillContent = ''; }}
              class="h-8 px-4 bg-gray-100 text-gray-600 text-[12px] font-medium rounded-lg hover:bg-gray-200 transition-colors cursor-pointer"
            >
              {t.cancel || '取消'}
            </button>
          </div>
        </div>
      </div>
    {/if}
  </div>
  {/if}

  {#if subTab === 'market'}
  <!-- Skills Market -->
  <div class="bg-white rounded-xl border border-gray-100 p-4 sm:p-5">
    <div class="flex items-center justify-between mb-4">
      <div class="flex items-center gap-2">
        <svg class="w-4 h-4 text-blue-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M13.5 21v-7.5a.75.75 0 01.75-.75h3a.75.75 0 01.75.75V21m-4.5 0H2.36m11.14 0H18m0 0h3.64m-1.39 0V9.349m-16.5 11.65V9.35m0 0a3.001 3.001 0 003.75-.615A2.993 2.993 0 009.75 9.75c.896 0 1.7-.393 2.25-1.016a2.993 2.993 0 002.25 1.016c.896 0 1.7-.393 2.25-1.016a3.001 3.001 0 003.75.614m-16.5 0a3.004 3.004 0 01-.621-4.72L4.318 3.44A1.5 1.5 0 015.378 3h13.243a1.5 1.5 0 011.06.44l1.19 1.189a3 3 0 01-.621 4.72m-13.5 8.65h3.75a.75.75 0 00.75-.75V13.5a.75.75 0 00-.75-.75H6.75a.75.75 0 00-.75.75v3.15c0 .415.336.75.75.75z" />
        </svg>
        <h3 class="text-[13px] font-semibold text-gray-900">{t.skillsMarket || 'Skills 市场'}</h3>
      </div>
      <div class="flex items-center gap-2">
        <button
          onclick={handleInstallAll}
          disabled={batchInstalling || batchUpdating || registryLoading}
          class="h-7 px-3 bg-gray-900 text-white text-[11px] font-medium rounded-lg hover:bg-gray-800 transition-colors disabled:opacity-50 cursor-pointer inline-flex items-center gap-1.5"
        >
          {#if batchInstalling}
            <svg class="w-3.5 h-3.5 animate-spin" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path></svg>
          {:else}
            <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M3 16.5v2.25A2.25 2.25 0 005.25 21h13.5A2.25 2.25 0 0021 18.75V16.5M16.5 12L12 16.5m0 0L7.5 12m4.5 4.5V3" /></svg>
          {/if}
          {t.installAll || '一键安装'}
        </button>
        <button
          onclick={handleUpdateAll}
          disabled={batchInstalling || batchUpdating || registryLoading}
          class="h-7 px-3 bg-blue-600 text-white text-[11px] font-medium rounded-lg hover:bg-blue-700 transition-colors disabled:opacity-50 cursor-pointer inline-flex items-center gap-1.5"
        >
          {#if batchUpdating}
            <svg class="w-3.5 h-3.5 animate-spin" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path></svg>
          {:else}
            <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M16.023 9.348h4.992v-.001M2.985 19.644v-4.992m0 0h4.992m-4.993 0l3.181 3.183a8.25 8.25 0 0013.803-3.7M4.031 9.865a8.25 8.25 0 0113.803-3.7l3.181 3.182" /></svg>
          {/if}
          {t.updateAll || '一键更新'}
        </button>
        <button
          onclick={loadRegistry}
          class="text-gray-400 hover:text-gray-600 cursor-pointer p-1"
          title={t.refresh || '刷新'}
        >
          <svg class="w-4 h-4 {registryLoading ? 'animate-spin' : ''}" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M16.023 9.348h4.992v-.001M2.985 19.644v-4.992m0 0h4.992m-4.993 0l3.181 3.183a8.25 8.25 0 0013.803-3.7M4.031 9.865a8.25 8.25 0 0113.803-3.7l3.181 3.182" /></svg>
        </button>
      </div>
    </div>

    <p class="text-[12px] text-gray-500 mb-4">{t.skillsMarketDesc || '从远程仓库下载社区和官方维护的 Skills 技能库，下载后可在 AI Agent 模式中使用。'}</p>

    {#if registryError}
      <div class="flex items-center gap-2 px-3 py-2 bg-red-50 border border-red-100 rounded-lg mb-3">
        <svg class="w-4 h-4 text-red-500 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z" /></svg>
        <span class="text-[12px] text-red-700 flex-1">{registryError}</span>
      </div>
    {/if}

    {#if registryLoading}
      <div class="flex items-center justify-center py-8">
        <svg class="w-5 h-5 animate-spin text-gray-400" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path></svg>
      </div>
    {:else if registrySkills.length === 0 && !registryError}
      <p class="text-center text-[12px] text-gray-400 py-6">{t.skillsMarketEmpty || '暂无可用 Skills'}</p>
    {:else}
      <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
        {#each registrySkills as skill}
          <div class="bg-gray-50 rounded-lg p-3 border border-gray-100">
            <div class="flex items-center justify-between mb-1.5">
              <span class="text-[12px] font-medium text-gray-900 truncate">{skill.name}</span>
              {#if skill.installed}
                <div class="flex items-center gap-1.5">
                  <span class="inline-flex items-center gap-1 px-2 py-0.5 bg-emerald-50 text-emerald-600 text-[10px] font-medium rounded-full">
                    <svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M4.5 12.75l6 6 9-13.5" /></svg>
                    {t.installed || '已安装'}
                  </span>
                  {#if skill.hasUpdate}
                    <button
                      onclick={() => handleUpdateSkill(skill)}
                      disabled={installingSkillId === skill.id}
                      class="h-5 px-1.5 text-blue-600 bg-blue-50 text-[10px] font-medium rounded hover:bg-blue-100 transition-colors disabled:opacity-50 cursor-pointer inline-flex items-center gap-0.5"
                    >
                      {#if installingSkillId === skill.id}
                        <svg class="w-3 h-3 animate-spin" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path></svg>
                      {:else}
                        <svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M16.023 9.348h4.992v-.001M2.985 19.644v-4.992m0 0h4.992m-4.993 0l3.181 3.183a8.25 8.25 0 0013.803-3.7M4.031 9.865a8.25 8.25 0 0113.803-3.7l3.181 3.182" /></svg>
                      {/if}
                      {t.update || '更新'}
                    </button>
                  {/if}
                </div>
              {:else}
                <button
                  onclick={() => handleInstallSkill(skill)}
                  disabled={installingSkillId === skill.id}
                  class="h-6 px-2 bg-gray-900 text-white text-[10px] font-medium rounded-md hover:bg-gray-800 transition-colors disabled:opacity-50 cursor-pointer inline-flex items-center gap-1"
                >
                  {#if installingSkillId === skill.id}
                    <svg class="w-3 h-3 animate-spin" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path></svg>
                  {:else}
                    <svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M3 16.5v2.25A2.25 2.25 0 005.25 21h13.5A2.25 2.25 0 0021 18.75V16.5M16.5 12L12 16.5m0 0L7.5 12m4.5 4.5V3" /></svg>
                  {/if}
                  {installingSkillId === skill.id ? '...' : (t.install || '安装')}
                </button>
              {/if}
            </div>
            <p class="text-[11px] text-gray-500 line-clamp-2 mb-1.5">{skill.description}</p>
            {#if skill.tags && skill.tags.length > 0}
              <div class="flex flex-wrap gap-1">
                {#each skill.tags.slice(0, 4) as tag}
                  <span class="px-1.5 py-0.5 bg-gray-200 text-gray-600 text-[10px] rounded">{tag}</span>
                {/each}
                {#if skill.tags.length > 4}
                  <span class="text-[10px] text-gray-400">+{skill.tags.length - 4}</span>
                {/if}
              </div>
            {/if}
          </div>
        {/each}
      </div>
    {/if}
  </div>
  {/if}
</div>
