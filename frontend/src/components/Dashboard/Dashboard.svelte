<script>
  import { onMount } from 'svelte';
  import { ListCases, GetResourceSummary, GetBalances, ListTemplates, ListProjects, TestTerraformEndpoints, GetTotalRuntime, ListScheduledTasks, ListAllScheduledTasks, GetMCPStatus, CheckAllUpdates, StartCase, StopCase } from '../../../wailsjs/go/main/App.js';
  import { toast } from '../../lib/toast.js';

  let { t, onTabChange = () => {} } = $props();

  // 翻译凭据错误信息
  function translateError(error) {
    if (!error) return '';
    // 中文厂商名映射到英文键名
    const providerMap = {
      '阿里云': 'Aliyun',
      '腾讯云': 'Tencent',
      '火山引擎': 'Volcengine',
      '华为云': 'Huawei',
      'UCloud': 'UCloud',
      'Vultr': 'Vultr',
      'AWS': 'AWS',
      'GCP': 'GCP',
      'Azure': 'Azure'
    };
    let provider = error.replace('未配置', '').replace('凭据', '').trim();
    const key = 'noCredentials' + (providerMap[provider] || provider);
    return t[key] || error;
  }
  
  // Dashboard state
  let stats = $state({
    totalCases: 0,
    runningCases: 0,
    stoppedCases: 0,
    errorCases: 0
  });
  
  let resourceSummary = $state([]);
  let balances = $state([]);
  let balancesLoading = $state(false);
  let recentCases = $state([]);
  let loading = $state(true);
  let stopConfirm = $state({ show: false, caseId: null, caseName: '' });
  
  // Version check
  let updateResult = $state(null);
  let updateLoading = $state(false);
  
  // Real data for templates and projects
  let templateCount = $state(0);
  let projectCount = $state(0);
  
  // Network diagnostics
  let networkChecks = $state([]);
  let networkCheckLoading = $state(false);
  
  // Real data for runtime and scheduled tasks
  let totalRuntime = $state('0h');
  let scheduledTaskCount = $state(0);
  
  // Recent AI conversations (from localStorage)
  let recentConversations = $state([]);
  
  // Recent tasks (all tasks for bottom section)
  let recentTasks = $state([]);
  
  // MCP status
  let mcpStatus = $state({ running: false, mode: '', address: '', protocolVersion: '' });

  // Quick stats with real data — clickable navigation targets
  let quickStats = $derived([
    { label: t.scheduledTasks || '定时任务', value: String(scheduledTaskCount), icon: 'clock', tab: 'taskCenter' },
    { label: t.runtime, value: totalRuntime, icon: 'timer', tab: null },
    { label: t.templateCount, value: String(templateCount), icon: 'template', tab: 'localTemplates' },
    { label: t.projectCount, value: String(projectCount), icon: 'project', tab: 'credentials' }
  ]);
  
  onMount(async () => {
    await loadDashboardData();
    await runNetworkCheck();
    await loadRuntime();
    loadRecentConversations();
    await loadRecentTasks();
    await loadMCPStatus();
  });
  
  async function loadDashboardData() {
    loading = true;
    try {
      // Load cases
      const cases = await ListCases();
      stats.totalCases = cases.length;
      stats.runningCases = cases.filter(c => c.state === 'running').length;
      stats.stoppedCases = cases.filter(c => c.state === 'stopped').length;
      stats.errorCases = cases.filter(c => c.state === 'error').length;
      
      // Get recent cases (last 5)
      recentCases = cases.slice(0, 5);
      
      // Load templates count
      try {
        const templates = await ListTemplates();
        templateCount = templates.length;
      } catch (e) {
        console.error('Failed to load templates:', e);
        templateCount = 0;
      }
      
      // Load projects count
      try {
        const projects = await ListProjects();
        projectCount = projects.length;
      } catch (e) {
        console.error('Failed to load projects:', e);
        projectCount = 0;
      }
      
      // Load resource summary
      try {
        resourceSummary = await GetResourceSummary();
      } catch (e) {
        console.error('Failed to load resource summary:', e);
      }
    } catch (e) {
      console.error('Failed to load dashboard data:', e);
    } finally {
      loading = false;
    }
  }
  
  async function queryBalances() {
    balancesLoading = true;
    balances = [];
    try {
      balances = await GetBalances(['aliyun', 'tencentcloud', 'volcengine', 'huaweicloud', 'ucloud', 'vultr']);
    } catch (e) {
      console.error('Failed to load balances:', e);
      balances = [];
    } finally {
      balancesLoading = false;
    }
  }
  
  async function checkUpdates() {
    updateLoading = true;
    try {
      updateResult = await CheckAllUpdates();
    } catch (e) {
      console.error('Failed to check updates:', e);
      updateResult = null;
    } finally {
      updateLoading = false;
    }
  }
  
  function getStateColor(state) {
    const colors = {
      'running': 'text-emerald-600 bg-emerald-50',
      'stopped': 'text-slate-500 bg-slate-50',
      'error': 'text-red-600 bg-red-50',
      'created': 'text-blue-600 bg-blue-50'
    };
    return colors[state] || 'text-gray-600 bg-gray-50';
  }
  
  function navigateToCases() {
    onTabChange('cases');
  }
  
  async function runNetworkCheck() {
    networkCheckLoading = true;
    try {
      networkChecks = await TestTerraformEndpoints();
    } catch (e) {
      console.error('Failed to run network check:', e);
      networkChecks = [];
    } finally {
      networkCheckLoading = false;
    }
  }
  
  async function loadRuntime() {
    try {
      totalRuntime = await GetTotalRuntime();
    } catch (e) {
      console.error('Failed to load runtime:', e);
      totalRuntime = '0h';
    }
    
    try {
      const tasks = await ListScheduledTasks();
      scheduledTaskCount = tasks ? tasks.length : 0;
    } catch (e) {
      console.error('Failed to load scheduled tasks:', e);
      scheduledTaskCount = 0;
    }
  }

  function loadRecentConversations() {
    try {
      const saved = localStorage.getItem('redc-ai-chat-conversations');
      if (saved) {
        const all = JSON.parse(saved);
        recentConversations = all
          .sort((a, b) => new Date(b.updatedAt).getTime() - new Date(a.updatedAt).getTime())
          .slice(0, 5);
      }
    } catch (e) {
      console.error('Failed to load AI conversations:', e);
    }
  }

  async function loadRecentTasks() {
    try {
      const tasks = await ListAllScheduledTasks();
      recentTasks = (tasks || [])
        .sort((a, b) => Number(new Date(String(b.createdAt))) - Number(new Date(String(a.createdAt))))
        .slice(0, 5);
    } catch (e) {
      console.error('Failed to load recent tasks:', e);
      recentTasks = [];
    }
  }

  async function loadMCPStatus() {
    try {
      mcpStatus = await GetMCPStatus();
    } catch (e) {
      console.error('Failed to load MCP status:', e);
      mcpStatus = { running: false, mode: '', address: '', protocolVersion: '' };
    }
  }

  function formatTimeAgo(dateStr) {
    if (!dateStr) return '';
    const diff = Date.now() - new Date(dateStr).getTime();
    const mins = Math.floor(diff / 60000);
    if (mins < 1) return t.justNow || '刚刚';
    if (mins < 60) return `${mins}${t.minutesAgo || '分钟前'}`;
    const hours = Math.floor(mins / 60);
    if (hours < 24) return `${hours}${t.hoursAgo || '小时前'}`;
    const days = Math.floor(hours / 24);
    return `${days}${t.daysAgo || '天前'}`;
  }

  function getModeLabel(modeId) {
    const labels = {
      'free': t.aiChatFreeChat || '自由对话',
      'agent': t.aiChatAgent || 'Agent',
      'deploy': t.aiChatDeploy || '部署',
      'errorAnalysis': t.aiChatErrorAnalysis || '错误分析',
      'generate': t.aiChatGenTemplate || '模板生成',
      'recommend': t.aiChatRecommend || '推荐',
      'cost': t.aiChatCostOpt || '成本优化'
    };
    return labels[modeId] || modeId;
  }

  function getTaskStatusStyle(status) {
    const styles = {
      'pending': 'text-blue-600 bg-blue-50',
      'completed': 'text-emerald-600 bg-emerald-50',
      'failed': 'text-red-600 bg-red-50',
      'cancelled': 'text-gray-500 bg-gray-50'
    };
    return styles[status] || 'text-gray-600 bg-gray-50';
  }

  function getTaskActionLabel(action) {
    const labels = {
      'start': t.start || '启动',
      'stop': t.stop || '停止',
      'ssh_command': 'SSH',
      'auto_stop': t.autoStop || '自动停止'
    };
    return labels[action] || action;
  }

  // Case action loading states
  let actionLoading = $state({});

  async function handleStartCase(e, caseId) {
    e.stopPropagation();
    actionLoading[caseId] = 'start';
    actionLoading = actionLoading;
    try {
      await StartCase(caseId);
      toast.success(t.caseStarted || '场景已启动');
      await loadDashboardData();
    } catch (err) {
      toast.error(`${t.startFailed || '启动失败'}: ${err.message || err}`);
    } finally {
      delete actionLoading[caseId];
      actionLoading = actionLoading;
    }
  }

  async function handleStopCase(e, caseId) {
    e.stopPropagation();
    const c = recentCases.find(c => c.id === caseId);
    stopConfirm = { show: true, caseId, caseName: c?.name || caseId };
  }

  async function confirmStopCase() {
    const caseId = stopConfirm.caseId;
    stopConfirm = { show: false, caseId: null, caseName: '' };
    actionLoading[caseId] = 'stop';
    actionLoading = actionLoading;
    try {
      await StopCase(caseId);
      toast.success(t.caseStopped || '场景已停止');
      await loadDashboardData();
    } catch (err) {
      toast.error(`${t.stopFailed || '停止失败'}: ${err.message || err}`);
    } finally {
      delete actionLoading[caseId];
      actionLoading = actionLoading;
    }
  }

  function handleSSH(e, c) {
    e.stopPropagation();
    onTabChange('cases');
  }

  function navigateToCreate() {
    onTabChange('cases');
  }

  function navigateToCredentials() {
    onTabChange('credentials');
  }
</script>

<div class="space-y-3">
  <!-- Stats Row: 4 scene stats + 4 quick stats in one row -->
  <div class="grid grid-cols-4 sm:grid-cols-8 gap-3">
    <div class="bg-white rounded-xl border border-gray-100 p-4">
      <div class="text-[11px] text-gray-500 mb-1">{t.totalScenes || '总场景数'}</div>
      <div class="text-[22px] font-bold text-gray-900">{stats.totalCases}</div>
    </div>
    <div class="bg-white rounded-xl border border-gray-100 p-4">
      <div class="text-[11px] text-gray-500 mb-1">{t.runningScenes || '运行中'}</div>
      <div class="text-[22px] font-bold text-emerald-600">{stats.runningCases}</div>
    </div>
    <div class="bg-white rounded-xl border border-gray-100 p-4">
      <div class="text-[11px] text-gray-500 mb-1">{t.stoppedScenes || '已停止'}</div>
      <div class="text-[22px] font-bold text-gray-400">{stats.stoppedCases}</div>
    </div>
    <div class="bg-white rounded-xl border border-gray-100 p-4">
      <div class="text-[11px] text-gray-500 mb-1">{t.errorScenes || '异常'}</div>
      <div class="text-[22px] font-bold text-red-500">{stats.errorCases}</div>
    </div>
    {#each quickStats as stat}
      <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
      <div 
        class="bg-white rounded-xl border border-gray-100 p-4 {stat.tab ? 'cursor-pointer hover:border-gray-200 hover:shadow-sm transition-all' : ''}"
        onclick={() => stat.tab && onTabChange(stat.tab)}
      >
        <div class="text-[11px] text-gray-500 mb-1">{stat.label}</div>
        <div class="text-[22px] font-bold text-gray-900">{stat.value}</div>
      </div>
    {/each}
  </div>
  
  <!-- Main Content Grid (unified single grid) -->
  <div class="grid grid-cols-1 lg:grid-cols-3 gap-3">
    <!-- Left Column: Recent Cases + Balance/Bill -->
    <div class="lg:col-span-2 space-y-3">
    <!-- Recent Cases -->
    <div class="bg-white rounded-xl border border-gray-100 overflow-hidden">
      <div class="px-4 py-3 border-b border-gray-100 flex items-center justify-between">
        <h3 class="text-[13px] font-semibold text-gray-900">{t.recentScenes || '最近场景'}</h3>
        <div class="flex items-center gap-2">
          <button
            class="inline-flex items-center gap-1 h-7 px-2.5 text-white bg-gray-900 hover:bg-gray-800 text-[11px] font-medium rounded-lg transition-colors cursor-pointer"
            onclick={navigateToCreate}
          >
            <svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M12 4v16m8-8H4" /></svg>
            {t.quickCreate || '快速创建'}
          </button>
          <button 
            class="text-[11px] text-gray-500 hover:text-gray-700 font-medium cursor-pointer"
            onclick={navigateToCases}
          >
            {t.viewAll || '查看全部'} →
          </button>
        </div>
      </div>
      <div class="divide-y divide-gray-50">
        {#if loading}
          <div class="px-5 py-8 text-center text-[13px] text-gray-400">
            {t.loading || '加载中...'}
          </div>
        {:else if recentCases.length === 0}
          <div class="px-5 py-10 text-center">
            <svg class="w-8 h-8 mx-auto mb-2 text-gray-300" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1">
              <path stroke-linecap="round" stroke-linejoin="round" d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4" />
            </svg>
            <div class="text-[12px] text-gray-400 mb-3">{t.noRecentScenes || '暂无场景'}</div>
            <button
              class="inline-flex items-center gap-1.5 h-7 px-3 text-[11px] text-white bg-gray-900 hover:bg-gray-800 rounded-lg transition-colors font-medium cursor-pointer"
              onclick={navigateToCreate}
            >
              <svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M12 4v16m8-8H4" /></svg>
              {t.createFirstScene || '创建第一个场景'}
            </button>
          </div>
        {:else}
          {#each recentCases as c}
            <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
            <div class="px-4 py-2.5 hover:bg-gray-50/50 transition-colors cursor-pointer" onclick={navigateToCases}>
              <div class="flex items-center justify-between">
                <div class="flex-1 min-w-0">
                  <div class="text-[12px] font-medium text-gray-900 truncate">{c.name}</div>
                  <div class="text-[10px] text-gray-400 mt-0.5">{c.type} · {c.stateTime}</div>
                </div>
                <div class="flex items-center gap-1.5">
                  {#if c.state === 'running'}
                    <button
                      class="w-6 h-6 flex items-center justify-center text-red-400 hover:text-red-600 hover:bg-red-50 rounded transition-colors disabled:opacity-50 cursor-pointer"
                      onclick={(e) => handleStopCase(e, c.id)}
                      disabled={actionLoading[c.id]}
                      title={t.stop || '停止'}
                    >
                      {#if actionLoading[c.id] === 'stop'}
                        <div class="w-3 h-3 border-2 border-red-200 border-t-red-500 rounded-full animate-spin"></div>
                      {:else}
                        <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M5.25 7.5A2.25 2.25 0 017.5 5.25h9a2.25 2.25 0 012.25 2.25v9a2.25 2.25 0 01-2.25 2.25h-9a2.25 2.25 0 01-2.25-2.25v-9z" /></svg>
                      {/if}
                    </button>
                    <button
                      class="w-6 h-6 flex items-center justify-center text-gray-400 hover:text-gray-700 hover:bg-gray-100 rounded transition-colors cursor-pointer"
                      onclick={(e) => handleSSH(e, c)}
                      title="SSH"
                    >
                      <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M6.75 7.5l3 2.25-3 2.25m4.5 0h3m-9 8.25h13.5A2.25 2.25 0 0021 18V6a2.25 2.25 0 00-2.25-2.25H5.25A2.25 2.25 0 003 6v12a2.25 2.25 0 002.25 2.25z" /></svg>
                    </button>
                  {:else if c.state === 'stopped' || c.state === 'created'}
                    <button
                      class="w-6 h-6 flex items-center justify-center text-emerald-400 hover:text-emerald-600 hover:bg-emerald-50 rounded transition-colors disabled:opacity-50 cursor-pointer"
                      onclick={(e) => handleStartCase(e, c.id)}
                      disabled={actionLoading[c.id]}
                      title={t.start || '启动'}
                    >
                      {#if actionLoading[c.id] === 'start'}
                        <div class="w-3 h-3 border-2 border-emerald-200 border-t-emerald-500 rounded-full animate-spin"></div>
                      {:else}
                        <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M5.25 5.653c0-.856.917-1.398 1.667-.986l11.54 6.348a1.125 1.125 0 010 1.971l-11.54 6.347a1.125 1.125 0 01-1.667-.985V5.653z" /></svg>
                      {/if}
                    </button>
                  {/if}
                  <span class="inline-flex items-center gap-1 px-2 py-0.5 text-[10px] font-medium rounded-full {getStateColor(c.state)}">
                    <span class="w-1.5 h-1.5 rounded-full bg-current"></span>
                    {c.state}
                  </span>
                </div>
              </div>
            </div>
          {/each}
        {/if}
      </div>
    </div>

    <!-- Balance + Version Check sub-grid under Recent Cases -->
    <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
      <!-- Account Balances -->
      <div class="bg-white rounded-xl border border-gray-100 overflow-hidden">
        <div class="px-4 py-3 border-b border-gray-100 flex items-center justify-between">
          <h3 class="text-[13px] font-semibold text-gray-900">{t.accountBalance || '账户余额'}</h3>
          <button 
            onclick={queryBalances}
            disabled={balancesLoading}
            class="h-6 px-2 text-gray-500 hover:text-gray-700 hover:bg-gray-50 text-[10px] font-medium rounded transition-colors disabled:opacity-50 cursor-pointer inline-flex items-center gap-1"
          >
            <svg class="w-3 h-3 {balancesLoading ? 'animate-spin' : ''}" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M16.023 9.348h4.992v-.001M2.985 19.644v-4.992m0 0h4.992m-4.993 0l3.181 3.183a8.25 8.25 0 0013.803-3.7M4.031 9.865a8.25 8.25 0 0113.803-3.7l3.181 3.182" /></svg>
            {balancesLoading ? (t.loading || '...') : (t.queryBalance || '查询')}
          </button>
        </div>
        <div class="px-4 py-3">
          {#if balancesLoading}
            <div class="text-center py-4 text-[12px] text-gray-400">
              {t.loading || '加载中...'}
            </div>
          {:else if balances.length === 0}
            <div class="text-center py-4 text-[12px] text-gray-400">
              {t.clickToQueryBalance || '点击上方按钮查询账户余额'}
            </div>
          {:else}
            <div class="divide-y divide-gray-50">
              {#each balances as balance}
                <div class="flex items-center justify-between py-1.5">
                  <span class="text-[11px] text-gray-500">{balance.provider}</span>
                  {#if balance.error}
                    <span class="text-[10px] text-gray-400">{translateError(balance.error)}</span>
                  {:else}
                    <span class="text-[12px] font-medium text-gray-900 tabular-nums">{balance.currency} {balance.amount}</span>
                  {/if}
                </div>
              {/each}
            </div>
          {/if}
        </div>
      </div>

      <!-- Version Check -->
      <div class="bg-white rounded-xl border border-gray-100 overflow-hidden">
        <div class="px-4 py-3 border-b border-gray-100 flex items-center justify-between">
          <h3 class="text-[13px] font-semibold text-gray-900">{t.versionCheck || '版本检查'}</h3>
          <button 
            onclick={checkUpdates}
            disabled={updateLoading}
            class="h-6 px-2 text-gray-500 hover:text-gray-700 hover:bg-gray-50 text-[10px] font-medium rounded transition-colors disabled:opacity-50 cursor-pointer inline-flex items-center gap-1"
          >
            <svg class="w-3 h-3 {updateLoading ? 'animate-spin' : ''}" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M16.023 9.348h4.992v-.001M2.985 19.644v-4.992m0 0h4.992m-4.993 0l3.181 3.183a8.25 8.25 0 0013.803-3.7M4.031 9.865a8.25 8.25 0 0113.803-3.7l3.181 3.182" /></svg>
            {updateLoading ? (t.loading || '...') : (t.checkUpdate || '检查')}
          </button>
        </div>
        <div class="px-4 py-3">
          {#if updateLoading}
            <div class="text-center py-4 text-[12px] text-gray-400">
              {t.loading || '加载中...'}
            </div>
          {:else if !updateResult}
            <div class="text-center py-4 text-[12px] text-gray-400">
              {t.clickToCheckUpdate || '点击上方按钮检查版本更新'}
            </div>
          {:else}
            <div class="divide-y divide-gray-50">
              <!-- RedC version -->
              <div class="flex items-center justify-between py-1.5">
                <span class="text-[11px] text-gray-500">RedC</span>
                {#if updateResult.redc.error}
                  <span class="text-[10px] text-gray-400">{updateResult.redc.error}</span>
                {:else if updateResult.redc.hasUpdate}
                  <span class="text-[11px]">
                    <span class="text-gray-400">{updateResult.redc.currentVersion}</span>
                    <span class="text-gray-300 mx-1">→</span>
                    <span class="text-emerald-600 font-medium">{updateResult.redc.latestVersion}</span>
                    <span class="ml-1 text-[10px] text-white bg-emerald-500 px-1 py-0.5 rounded">NEW</span>
                  </span>
                {:else}
                  <span class="text-[11px] text-emerald-600">✓ {updateResult.redc.currentVersion}</span>
                {/if}
              </div>
              <!-- Templates -->
              {#if updateResult.templates && updateResult.templates.length > 0}
                {@const updatable = updateResult.templates.filter(t => t.hasUpdate)}
                <div class="flex items-center justify-between py-1.5">
                  <span class="text-[11px] text-gray-500">{t.templates || '模板'}</span>
                  {#if updatable.length > 0}
                    <span class="text-[11px]">
                      <span class="text-amber-600 font-medium">{updatable.length}</span>
                      <span class="text-gray-400 ml-0.5">{t.updatesAvailable || '个可更新'}</span>
                    </span>
                  {:else}
                    <span class="text-[11px] text-emerald-600">✓ {t.allUpToDate || '全部最新'}</span>
                  {/if}
                </div>
                {#if updatable.length > 0}
                  {#each updatable as tmpl}
                    <div class="flex items-center justify-between py-1 pl-4">
                      <span class="text-[10px] text-gray-400">{tmpl.name}</span>
                      <span class="text-[10px]">
                        <span class="text-gray-400">{tmpl.localVersion}</span>
                        <span class="text-gray-300 mx-0.5">→</span>
                        <span class="text-emerald-600">{tmpl.latestVersion}</span>
                      </span>
                    </div>
                  {/each}
                {/if}
              {:else}
                <div class="flex items-center justify-between py-1.5">
                  <span class="text-[11px] text-gray-500">{t.templates || '模板'}</span>
                  <span class="text-[11px] text-gray-400">{t.noLocalTemplates || '暂无本地模板'}</span>
                </div>
              {/if}
              <!-- Plugins -->
              {#if updateResult.plugins && updateResult.plugins.length > 0}
                <div class="flex items-center justify-between py-1.5">
                  <span class="text-[11px] text-gray-500">{t.plugins || '插件'}</span>
                  <span class="text-[11px] text-emerald-600">✓ {t.allUpToDate || '全部最新'}</span>
                </div>
              {:else}
                <div class="flex items-center justify-between py-1.5">
                  <span class="text-[11px] text-gray-500">{t.plugins || '插件'}</span>
                  <span class="text-[11px] text-gray-400">{t.noPlugins || '暂无插件'}</span>
                </div>
              {/if}
            </div>
          {/if}
        </div>
      </div>
    </div>
    </div> <!-- close left column wrapper -->
    
    <!-- Right column: Network + Quick Links -->
    <div class="flex flex-col gap-3">
      <!-- Network Diagnostics -->
      <div class="bg-white rounded-xl border border-gray-100 overflow-hidden">
        <div class="px-4 py-3 border-b border-gray-100 flex items-center justify-between">
          <h3 class="text-[13px] font-semibold text-gray-900">{t.networkCheck || '网络诊断'}</h3>
          <button
            class="h-6 px-2 text-gray-500 hover:text-gray-700 hover:bg-gray-50 text-[10px] font-medium rounded transition-colors disabled:opacity-50 cursor-pointer inline-flex items-center gap-1"
            onclick={runNetworkCheck}
            disabled={networkCheckLoading}
          >
            <svg class="w-3 h-3 {networkCheckLoading ? 'animate-spin' : ''}" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M16.023 9.348h4.992v-.001M2.985 19.644v-4.992m0 0h4.992m-4.993 0l3.181 3.183a8.25 8.25 0 0013.803-3.7M4.031 9.865a8.25 8.25 0 0113.803-3.7l3.181 3.182" /></svg>
            {networkCheckLoading ? t.networkChecking : t.retest}
          </button>
        </div>
        <div class="px-4 py-3">
          {#if networkCheckLoading}
            <div class="text-center py-6 text-[12px] text-gray-400">
              {t.loading || '加载中...'}
            </div>
          {:else if networkChecks.length === 0}
            <div class="text-center py-6 text-[12px] text-gray-400">
              {t.noNetworkData || '暂无网络诊断数据'}
            </div>
          {:else}
            <div class="divide-y divide-gray-50">
              {#each networkChecks as item}
                <div class="flex items-center justify-between py-1.5">
                  <div class="flex items-center gap-2 flex-1 min-w-0">
                    <div class="w-1.5 h-1.5 rounded-full flex-shrink-0 {item.ok ? 'bg-emerald-500' : 'bg-red-500'}"></div>
                    <span class="text-[11px] text-gray-600 truncate">{item.name}</span>
                  </div>
                  <div class="flex items-center gap-2 flex-shrink-0">
                    <span class="text-[10px] text-gray-400 tabular-nums">{item.latencyMs}ms</span>
                    <span class="text-[10px] font-medium {item.ok ? 'text-emerald-500' : 'text-red-500'} w-8 text-right">
                      {item.ok ? 'OK' : 'FAIL'}
                    </span>
                  </div>
                </div>
              {/each}
            </div>
            {#if networkChecks.some(item => !item.ok)}
              <div class="mt-2 pt-2 border-t border-gray-100">
                <button
                  class="text-[10px] text-gray-500 hover:text-gray-700 font-medium cursor-pointer"
                  onclick={navigateToCredentials}
                >
                  {t.goToCredentials || '前往凭据管理'} →
                </button>
              </div>
            {/if}
          {/if}
        </div>
      </div>

      <!-- Quick Links -->
      <div class="bg-white rounded-xl border border-gray-100 p-4 flex-1">
        <h3 class="text-[13px] font-semibold text-gray-900 mb-3">{t.quickLinks || '快捷入口'}</h3>
        <div class="grid grid-cols-2 gap-2">
          <button class="flex items-center gap-2 px-3 py-2 text-[11px] text-gray-600 bg-gray-50 rounded-lg hover:bg-gray-100 transition-colors cursor-pointer" onclick={() => onTabChange('aiChat')}>
            <svg class="w-3.5 h-3.5 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M20.25 8.511c.884.284 1.5 1.128 1.5 2.097v4.286c0 1.136-.847 2.1-1.98 2.193-.34.027-.68.052-1.02.072v3.091l-3-3c-1.354 0-2.694-.055-4.02-.163a2.115 2.115 0 01-.825-.242m9.345-8.334a2.126 2.126 0 00-.476-.095 48.64 48.64 0 00-8.048 0c-1.131.094-1.976 1.057-1.976 2.192v4.286c0 .837.46 1.58 1.155 1.951m9.345-8.334V6.637c0-1.621-1.152-3.026-2.76-3.235A48.455 48.455 0 0011.25 3c-2.115 0-4.198.137-6.24.402-1.608.209-2.76 1.614-2.76 3.235v6.226c0 1.621 1.152 3.026 2.76 3.235.577.075 1.157.14 1.74.194V21l4.155-4.155" /></svg>
            {t.aiChat || 'AI 对话'}
          </button>
          <button class="flex items-center gap-2 px-3 py-2 text-[11px] text-gray-600 bg-gray-50 rounded-lg hover:bg-gray-100 transition-colors cursor-pointer" onclick={() => onTabChange('registry')}>
            <svg class="w-3.5 h-3.5 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M20.25 7.5l-.625 10.632a2.25 2.25 0 01-2.247 2.118H6.622a2.25 2.25 0 01-2.247-2.118L3.75 7.5M10 11.25h4M3.375 7.5h17.25c.621 0 1.125-.504 1.125-1.125v-1.5c0-.621-.504-1.125-1.125-1.125H3.375c-.621 0-1.125.504-1.125 1.125v1.5c0 .621.504 1.125 1.125 1.125z" /></svg>
            {t.registry || '模板仓库'}
          </button>
          <button class="flex items-center gap-2 px-3 py-2 text-[11px] text-gray-600 bg-gray-50 rounded-lg hover:bg-gray-100 transition-colors cursor-pointer" onclick={() => onTabChange('sshManager')}>
            <svg class="w-3.5 h-3.5 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M6.75 7.5l3 2.25-3 2.25m4.5 0h3m-9 8.25h13.5A2.25 2.25 0 0021 18V6a2.25 2.25 0 00-2.25-2.25H5.25A2.25 2.25 0 003 6v12a2.25 2.25 0 002.25 2.25z" /></svg>
            {t.ssh || 'SSH 管理'}
          </button>
          <button class="flex items-center gap-2 px-3 py-2 text-[11px] text-gray-600 bg-gray-50 rounded-lg hover:bg-gray-100 transition-colors cursor-pointer" onclick={() => onTabChange('credentials')}>
            <svg class="w-3.5 h-3.5 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M15.75 5.25a3 3 0 013 3m3 0a6 6 0 01-7.029 5.912c-.563-.097-1.159.026-1.563.43L10.5 17.25H8.25v2.25H6v2.25H2.25v-2.818c0-.597.237-1.17.659-1.591l6.499-6.499c.404-.404.527-1 .43-1.563A6 6 0 1121.75 8.25z" /></svg>
            {t.credentials || '凭据管理'}
          </button>
        </div>
      </div>
    </div>
  </div>

  <!-- Bottom Section: Recent AI Chat + Recent Tasks + MCP Status -->
  <div class="grid grid-cols-1 lg:grid-cols-3 gap-3">
    <!-- Recent AI Conversations -->
    <div class="bg-white rounded-xl border border-gray-100 overflow-hidden">
      <div class="px-4 py-3 border-b border-gray-100 flex items-center justify-between">
        <h3 class="text-[13px] font-semibold text-gray-900">{t.recentAIChat || '最近 AI 对话'}</h3>
        <button class="text-[11px] text-gray-500 hover:text-gray-700 font-medium cursor-pointer" onclick={() => onTabChange('aiChat')}>
          {t.viewAll || '查看全部'} →
        </button>
      </div>
      <div class="divide-y divide-gray-50">
        {#if recentConversations.length === 0}
          <div class="px-4 py-6 text-center text-[12px] text-gray-400">
            {t.noAIChat || '暂无对话记录'}
          </div>
        {:else}
          {#each recentConversations as conv}
            <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
            <div class="px-4 py-2.5 hover:bg-gray-50/50 transition-colors cursor-pointer" onclick={() => onTabChange('aiChat')}>
              <div class="flex items-center justify-between">
                <div class="flex-1 min-w-0">
                  <div class="text-[12px] font-medium text-gray-900 truncate">{conv.title || t.untitledConversation || '未命名对话'}</div>
                  <div class="text-[10px] text-gray-400 mt-0.5">
                    <span class="inline-flex items-center px-1.5 py-0.5 rounded text-[9px] font-medium bg-gray-100 text-gray-500 mr-1">{getModeLabel(conv.mode)}</span>
                    {conv.messages?.length || 0} {t.messages || '条消息'}
                  </div>
                </div>
                <span class="text-[10px] text-gray-400 flex-shrink-0 ml-2">{formatTimeAgo(conv.updatedAt)}</span>
              </div>
            </div>
          {/each}
        {/if}
      </div>
    </div>

    <!-- Recent Tasks -->
    <div class="bg-white rounded-xl border border-gray-100 overflow-hidden">
      <div class="px-4 py-3 border-b border-gray-100 flex items-center justify-between">
        <h3 class="text-[13px] font-semibold text-gray-900">{t.recentTasks || '最近任务'}</h3>
        <button class="text-[11px] text-gray-500 hover:text-gray-700 font-medium cursor-pointer" onclick={() => onTabChange('taskCenter')}>
          {t.viewAll || '查看全部'} →
        </button>
      </div>
      <div class="divide-y divide-gray-50">
        {#if recentTasks.length === 0}
          <div class="px-4 py-6 text-center text-[12px] text-gray-400">
            {t.noTasks || '暂无任务'}
          </div>
        {:else}
          {#each recentTasks as task}
            <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
            <div class="px-4 py-2.5 hover:bg-gray-50/50 transition-colors cursor-pointer" onclick={() => onTabChange('taskCenter')}>
              <div class="flex items-center justify-between">
                <div class="flex-1 min-w-0">
                  <div class="text-[12px] font-medium text-gray-900 truncate">{task.caseName || task.caseId}</div>
                  <div class="text-[10px] text-gray-400 mt-0.5">{getTaskActionLabel(task.action)} · {formatTimeAgo(task.createdAt)}</div>
                </div>
                <span class="inline-flex items-center px-1.5 py-0.5 rounded-full text-[9px] font-medium {getTaskStatusStyle(task.status)}">
                  {task.status}
                </span>
              </div>
            </div>
          {/each}
        {/if}
      </div>
    </div>

    <!-- MCP Status -->
    <div class="bg-white rounded-xl border border-gray-100 overflow-hidden">
      <div class="px-4 py-3 border-b border-gray-100 flex items-center justify-between">
        <h3 class="text-[13px] font-semibold text-gray-900">{t.mcpStatus || 'MCP 状态'}</h3>
        <button class="text-[11px] text-gray-500 hover:text-gray-700 font-medium cursor-pointer" onclick={() => onTabChange('ai')}>
          {t.manage || '管理'} →
        </button>
      </div>
      <div class="px-4 py-4">
        <div class="flex items-center gap-3 mb-4">
          <div class="w-8 h-8 rounded-full flex items-center justify-center {mcpStatus.running ? 'bg-emerald-100' : 'bg-gray-100'}">
            <div class="w-2.5 h-2.5 rounded-full {mcpStatus.running ? 'bg-emerald-500 animate-pulse' : 'bg-gray-400'}"></div>
          </div>
          <div>
            <div class="text-[13px] font-medium {mcpStatus.running ? 'text-emerald-700' : 'text-gray-500'}">
              {mcpStatus.running ? (t.mcpRunning || '运行中') : (t.mcpStopped || '未启动')}
            </div>
            {#if mcpStatus.running}
              <div class="text-[10px] text-gray-400">{mcpStatus.mode?.toUpperCase()} · {mcpStatus.address}</div>
            {/if}
          </div>
        </div>
        {#if mcpStatus.running}
          <div class="space-y-2">
            <div class="flex items-center justify-between">
              <span class="text-[11px] text-gray-500">{t.mcpMode || '模式'}</span>
              <span class="text-[11px] font-medium text-gray-900">{mcpStatus.mode?.toUpperCase()}</span>
            </div>
            <div class="flex items-center justify-between">
              <span class="text-[11px] text-gray-500">{t.mcpAddress || '地址'}</span>
              <span class="text-[11px] font-medium text-gray-900 font-mono">{mcpStatus.address}</span>
            </div>
            <div class="flex items-center justify-between">
              <span class="text-[11px] text-gray-500">{t.mcpProtocol || '协议版本'}</span>
              <span class="text-[11px] font-medium text-gray-900">{mcpStatus.protocolVersion}</span>
            </div>
          </div>
        {:else}
          <div class="text-center text-[11px] text-gray-400">
            {t.mcpNotStartedHint || '前往 AI 集成页面启动 MCP 服务'}
          </div>
        {/if}
      </div>
    </div>
  </div>
</div>

{#if stopConfirm.show}
  <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
  <div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onclick={() => stopConfirm = { show: false, caseId: null, caseName: '' }}>
    <div class="bg-white rounded-xl border border-gray-200 shadow-xl max-w-sm w-full mx-4 overflow-hidden" onclick={(e) => e.stopPropagation()}>
      <div class="px-6 py-5">
        <div class="flex items-center gap-3 mb-3">
          <div class="w-10 h-10 rounded-full bg-amber-100 flex items-center justify-center">
            <svg class="w-5 h-5 text-amber-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
            </svg>
          </div>
          <div>
            <h3 class="text-[15px] font-semibold text-gray-900">{t.confirmStop || '确认停止'}</h3>
            <p class="text-[13px] text-gray-500">{t.stopWarning || '此操作将销毁相关资源'}</p>
          </div>
        </div>
        <p class="text-[13px] text-gray-600">
          {t.confirmStopScene || '确定要停止场景'} <span class="font-medium text-gray-900">"{stopConfirm.caseName}"</span>?
        </p>
      </div>
      <div class="px-6 py-4 bg-gray-50 flex justify-end gap-2">
        <button class="px-4 py-2 text-[13px] font-medium text-gray-700 bg-white border border-gray-100 rounded-lg hover:bg-gray-50 transition-colors"
          onclick={() => stopConfirm = { show: false, caseId: null, caseName: '' }}
        >{t.cancel || '取消'}</button>
        <button class="px-4 py-2 text-[13px] font-medium text-white bg-amber-600 rounded-lg hover:bg-amber-700 transition-colors"
          onclick={confirmStopCase}
        >{t.stop || '停止'}</button>
      </div>
    </div>
  </div>
{/if}
