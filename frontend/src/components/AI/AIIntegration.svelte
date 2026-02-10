<script>
  import { onMount } from 'svelte';
  import { GetMCPStatus, StartMCPServer, StopMCPServer, RecommendTemplates, AIRecommendTemplates, AICostOptimization, PullTemplate, GetActiveProfile } from '../../../wailsjs/go/main/App.js';
  import { EventsOn, EventsOff } from '../../../wailsjs/runtime/runtime.js';

  let { t, onTabChange = () => {} } = $props();
  let mcpStatus = $state({ running: false, mode: '', address: '', protocolVersion: '' });
  let mcpForm = $state({ mode: 'sse', address: 'localhost:8080' });
  let mcpLoading = $state(false);
  let error = $state('');

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

  // Smart recommendation state
  let recommendQuery = $state('');
  let recommendLoading = $state(false);
  let recommendResults = $state([]);
  let showRecommendResults = $state(false);
  let pullingTemplate = $state('');
  let aiRecommendText = $state('');
  let aiRecommending = $state(false);

  // Cost optimization state
  let costLoading = $state(false);
  let costSuggestions = $state([]);
  let showCostSuggestions = $state(false);
  let aiCostText = $state('');
  let aiCostAnalyzing = $state(false);

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

  // Mock data for cost optimization
  const mockCostSuggestions = [
    {
      caseId: 'case-001',
      caseName: '测试环境-阿里云',
      currentCost: 156.80,
      potentialSavings: 47.20,
      suggestions: [
        '建议将 ecs.g6.large 实例降级为 ecs.g6.medium，可节省约 30% 成本',
        '当前实例利用率仅 15%，建议启用自动伸缩策略'
      ],
      priority: 'high'
    },
    {
      caseId: 'case-002', 
      caseName: '开发环境-腾讯云',
      currentCost: 89.50,
      potentialSavings: 26.85,
      suggestions: [
        '开发环境建议使用竞价实例，可节省约 60% 成本',
        '建议配置定时关机策略，非工作时间自动停止'
      ],
      priority: 'medium'
    },
    {
      caseId: 'case-003',
      caseName: '生产环境-华为云',
      currentCost: 328.00,
      potentialSavings: 65.60,
      suggestions: [
        '建议购买预留实例券，长期使用可节省约 20%',
        '存储卷类型可由 SSD 降级为高 IO，节省存储成本'
      ],
      priority: 'low'
    }
  ];

  onMount(() => {
    loadMCPStatus();
    loadAIConfig();

    // Listen for AI recommendation events
    EventsOn('ai-recommend-chunk', (chunk) => {
      aiRecommendText += chunk;
      // Show results when first chunk arrives
      if (!showRecommendResults) {
        showRecommendResults = true;
      }
    });

    EventsOn('ai-recommend-complete', () => {
      aiRecommending = false;
      recommendLoading = false;
    });

    // Listen for AI cost optimization events
    EventsOn('ai-cost-chunk', (chunk) => {
      aiCostText += chunk;
      // Show results when first chunk arrives
      if (!showCostSuggestions) {
        showCostSuggestions = true;
      }
    });

    EventsOn('ai-cost-complete', () => {
      aiCostAnalyzing = false;
      costLoading = false;
    });

    return () => {
      EventsOff('ai-recommend-chunk');
      EventsOff('ai-recommend-complete');
      EventsOff('ai-cost-chunk');
      EventsOff('ai-cost-complete');
    };
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

  async function handleRecommend() {
    if (!recommendQuery.trim()) return;
    
    // Check if AI is configured
    if (!isAIConfigured()) {
      error = 'AI 服务未配置，请先配置 AI API Key';
      return;
    }

    recommendLoading = true;
    aiRecommending = true;
    showRecommendResults = false;
    aiRecommendText = '';
    error = '';
    
    try {
      await AIRecommendTemplates(recommendQuery);
      // Don't set showRecommendResults here - it will be set when first chunk arrives
    } catch (e) {
      error = e.message || String(e);
      aiRecommending = false;
      recommendLoading = false;
    }
  }

  async function handlePullTemplate(template) {
    pullingTemplate = template;
    try {
      await PullTemplate(template, false);
    } catch (e) {
      error = e.message || String(e);
    } finally {
      pullingTemplate = '';
    }
  }

  async function handleAnalyzeCost() {
    // Check if AI is configured
    if (!isAIConfigured()) {
      error = 'AI 服务未配置，请先配置 AI API Key';
      return;
    }

    costLoading = true;
    aiCostAnalyzing = true;
    showCostSuggestions = false;
    aiCostText = '';
    error = '';

    try {
      await AICostOptimization();
      // Don't set showCostSuggestions here - it will be set when first chunk arrives
    } catch (e) {
      error = e.message || String(e);
      aiCostAnalyzing = false;
      costLoading = false;
    }
  }

  function getPriorityColor(priority) {
    switch(priority) {
      case 'high': return 'text-red-600 bg-red-50';
      case 'medium': return 'text-amber-600 bg-amber-50';
      case 'low': return 'text-emerald-600 bg-emerald-50';
      default: return 'text-gray-600 bg-gray-50';
    }
  }

  function getPriorityLabel(priority) {
    switch(priority) {
      case 'high': return t.priorityHigh || '高';
      case 'medium': return t.priorityMedium || '中';
      case 'low': return t.priorityLow || '低';
      default: return priority;
    }
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
      <button class="text-red-400 hover:text-red-600" onclick={() => error = ''}>
        <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
        </svg>
      </button>
    </div>
  {/if}

  <!-- AI Configuration Status Card -->
  <div class="bg-white rounded-xl border border-gray-100 p-4 sm:p-5">
    <div class="flex items-center gap-3 mb-4">
      <div class="w-9 h-9 sm:w-10 sm:h-10 rounded-lg bg-gradient-to-br from-orange-500 to-amber-600 flex items-center justify-center">
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
            <p class="font-medium text-gray-900 mt-0.5">{providerPresets[aiConfig.provider]?.name || aiConfig.provider}</p>
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
          class="text-[11px] sm:text-[12px] text-blue-600 hover:text-blue-700 font-medium flex items-center gap-1"
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
          class="inline-flex items-center gap-2 px-4 h-9 bg-purple-500 text-white text-[12px] font-medium rounded-lg hover:bg-purple-600 transition-colors"
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

  <!-- Smart Template Recommendation Card -->
  <div class="bg-white rounded-xl border border-gray-100 p-4 sm:p-5">
    <div class="flex items-center gap-3 mb-4">
      <div class="w-9 h-9 sm:w-10 sm:h-10 rounded-lg bg-gradient-to-br from-blue-500 to-cyan-600 flex items-center justify-center">
        <svg class="w-4.5 h-4.5 sm:w-5 sm:h-5 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z" />
        </svg>
      </div>
      <div>
        <h2 class="text-[13px] sm:text-[14px] font-semibold text-gray-900">{t.smartRecommend || '智能场景推荐'}</h2>
        <p class="text-[11px] sm:text-[12px] text-gray-500">{t.smartRecommendDesc || '描述您的需求，AI 将为您推荐最合适的场景模板'}</p>
      </div>
    </div>

    <div class="space-y-3">
      <div class="flex gap-2">
        <input 
          type="text" 
          placeholder={t.recommendPlaceholder || '例如：我需要一个阿里云的测试环境...'}
          class="flex-1 h-9 sm:h-10 px-3 text-[12px] sm:text-[13px] bg-gray-50 border-0 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-blue-500 focus:ring-offset-1 transition-shadow"
          bind:value={recommendQuery}
          onkeydown={(e) => e.key === 'Enter' && handleRecommend()}
        />
        <button 
          class="px-4 h-9 sm:h-10 bg-blue-500 text-white text-[12px] sm:text-[13px] font-medium rounded-lg hover:bg-blue-600 transition-colors disabled:opacity-50 flex items-center gap-2"
          onclick={handleRecommend}
          disabled={recommendLoading || !recommendQuery.trim()}
        >
          {#if recommendLoading}
            <svg class="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
            </svg>
          {:else}
            <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M21 21l-5.197-5.197m0 0A7.5 7.5 0 105.196 5.196a7.5 7.5 0 0010.607 10.607z" />
            </svg>
          {/if}
          {t.recommend || '推荐'}
        </button>
      </div>

      {#if showRecommendResults && aiRecommendText}
        <div class="mt-4 bg-gray-50 rounded-lg p-4 border border-gray-200">
          <div class="flex items-start gap-2 mb-2">
            <svg class="w-4 h-4 text-blue-500 flex-shrink-0 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M9.813 15.904L9 18.75l-.813-2.846a4.5 4.5 0 00-3.09-3.09L2.25 12l2.846-.813a4.5 4.5 0 003.09-3.09L9 5.25l.813 2.846a4.5 4.5 0 003.09 3.09L15.75 12l-2.846.813a4.5 4.5 0 00-3.09 3.09z" />
            </svg>
            <div class="flex-1">
              <h3 class="text-[12px] font-medium text-gray-900 mb-2">AI 推荐结果</h3>
              <div class="text-[12px] text-gray-700 whitespace-pre-wrap leading-relaxed">
                {aiRecommendText}
                {#if aiRecommending}
                  <span class="inline-block w-1.5 h-4 bg-blue-500 animate-pulse ml-0.5"></span>
                {/if}
              </div>
            </div>
          </div>
        </div>
      {:else if showRecommendResults && !aiRecommendText && !aiRecommending}
        <div class="text-center py-6 text-gray-500 text-[12px]">
          {t.noMatch || '未找到匹配的模板'}
        </div>
      {/if}
    </div>
  </div>

  <!-- Cost Optimization Card -->
  <div class="bg-white rounded-xl border border-gray-100 p-4 sm:p-5">
    <div class="flex items-center gap-3 mb-4">
      <div class="w-9 h-9 sm:w-10 sm:h-10 rounded-lg bg-gradient-to-br from-emerald-500 to-teal-600 flex items-center justify-center">
        <svg class="w-4.5 h-4.5 sm:w-5 sm:h-5 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8c1.11 0 2.08.402 2.599 1M12 8V7m0 1v8m0 0v1m0-1c-1.11 0-2.08-.402-2.599-1M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
      </div>
      <div>
        <h2 class="text-[13px] sm:text-[14px] font-semibold text-gray-900">{t.costOptimization || '成本优化建议'}</h2>
        <p class="text-[11px] sm:text-[12px] text-gray-500">{t.costOptimizationDesc || '分析运行中的场景，提供成本优化建议'}</p>
      </div>
    </div>

    <div class="space-y-3">
      <button 
        class="w-full h-9 sm:h-10 bg-emerald-500 text-white text-[12px] sm:text-[13px] font-medium rounded-lg hover:bg-emerald-600 transition-colors disabled:opacity-50 flex items-center justify-center gap-2"
        onclick={handleAnalyzeCost}
        disabled={costLoading}
      >
        {#if costLoading}
          <svg class="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
          </svg>
          {t.analyzing || '分析中...'}
        {:else}
          <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4" />
          </svg>
          {t.startAnalysis || '开始分析'}
        {/if}
      </button>

      {#if showCostSuggestions && aiCostText}
        <div class="mt-4 bg-gray-50 rounded-lg p-4 border border-gray-200">
          <div class="flex items-start gap-2 mb-2">
            <svg class="w-4 h-4 text-emerald-500 flex-shrink-0 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8c1.11 0 2.08.402 2.599 1M12 8V7m0 1v8m0 0v1m0-1c-1.11 0-2.08-.402-2.599-1M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
            <div class="flex-1">
              <h3 class="text-[12px] font-medium text-gray-900 mb-2">成本优化分析</h3>
              <div class="text-[12px] text-gray-700 whitespace-pre-wrap leading-relaxed">
                {aiCostText}
                {#if aiCostAnalyzing}
                  <span class="inline-block w-1.5 h-4 bg-emerald-500 animate-pulse ml-0.5"></span>
                {/if}
              </div>
            </div>
          </div>
        </div>
      {:else if showCostSuggestions && !aiCostText && !aiCostAnalyzing}
        <div class="text-center py-6 text-gray-500 text-[12px]">
          {t.noRunningCases || '当前没有运行中的场景'}
        </div>
      {/if}
    </div>
  </div>

<!-- MCP Status Card -->
  <div class="bg-white rounded-xl border border-gray-100 p-4 sm:p-5">
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 sm:gap-0 mb-4">
      <div class="flex items-center gap-3">
        <div class="w-9 h-9 sm:w-10 sm:h-10 rounded-lg bg-gradient-to-br from-purple-500 to-indigo-600 flex items-center justify-center">
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
        class="w-full h-9 sm:h-10 bg-red-500 text-white text-[12px] sm:text-[13px] font-medium rounded-lg hover:bg-red-600 transition-colors disabled:opacity-50"
        onclick={handleStopMCP}
        disabled={mcpLoading}
      >
        {mcpLoading ? t.stoppingServer : t.stopServer}
      </button>
    {:else}
      <!-- Configuration form -->
      <div class="space-y-3 sm:space-y-4 mb-4">
        <div>
          <label class="block text-[11px] sm:text-[12px] font-medium text-gray-500 mb-1.5">{t.transportMode}</label>
          <div class="inline-flex items-center h-9 sm:h-10 px-3 sm:px-4 text-[12px] sm:text-[13px] font-medium rounded-lg border bg-gray-900 text-white border-gray-900">
            SSE (HTTP)
          </div>
        </div>
        <div>
          <label class="block text-[11px] sm:text-[12px] font-medium text-gray-500 mb-1.5">{t.listenAddr}</label>
          <input 
            type="text" 
            placeholder="localhost:8080" 
            class="w-full h-9 sm:h-10 px-3 text-[12px] sm:text-[13px] bg-gray-50 border-0 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow font-mono"
            bind:value={mcpForm.address} 
          />
        </div>
      </div>
      <button 
        class="w-full h-9 sm:h-10 bg-gray-900 text-white text-[12px] sm:text-[13px] font-medium rounded-lg hover:bg-gray-800 transition-colors disabled:opacity-50"
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
      <div class="grid grid-cols-1 sm:grid-cols-2 gap-2 text-[11px] sm:text-[12px]">
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
      </div>
    </div>
  </div>
</div>
