<script>
  import { onMount } from 'svelte';
  import { ListCases, GetResourceSummary, GetBalances, ListTemplates, ListProjects, TestTerraformEndpoints, GetTotalRuntime, GetPredictedMonthlyCost } from '../../../wailsjs/go/main/App.js';

  let { t, onTabChange = () => {} } = $props();
  
  // Dashboard state
  let stats = $state({
    totalCases: 0,
    runningCases: 0,
    stoppedCases: 0,
    errorCases: 0
  });
  
  let resourceSummary = $state([]);
  let balances = $state([]);
  let recentCases = $state([]);
  let loading = $state(true);
  
  // Real data for templates and projects
  let templateCount = $state(0);
  let projectCount = $state(0);
  
  // Network diagnostics
  let networkChecks = $state([]);
  let networkCheckLoading = $state(false);
  
  // Real data for runtime and predicted cost
  let totalRuntime = $state('0h');
  let predictedMonthlyCost = $state('¥0.00');
  
  // Quick stats with real data
  let quickStats = $derived([
    { label: '预测本月费用', value: predictedMonthlyCost, change: '0', trend: 'neutral', mock: false },
    { label: '运行时长', value: totalRuntime, change: '0', trend: 'neutral', mock: false },
    { label: '模板数量', value: String(templateCount), change: '0', trend: 'neutral', mock: false },
    { label: '项目数量', value: String(projectCount), change: '0', trend: 'neutral', mock: false }
  ]);
  
  onMount(async () => {
    await loadDashboardData();
    await runNetworkCheck();
    await loadRuntime();
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
      
      // Load balances
      try {
        balances = await GetBalances(['aliyun', 'tencentcloud', 'volcengine', 'huaweicloud']);
      } catch (e) {
        console.error('Failed to load balances:', e);
      }
    } catch (e) {
      console.error('Failed to load dashboard data:', e);
    } finally {
      loading = false;
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
      // Load total runtime
      totalRuntime = await GetTotalRuntime();
    } catch (e) {
      console.error('Failed to load runtime:', e);
      totalRuntime = '0h';
    }
    
    try {
      // Load predicted monthly cost
      predictedMonthlyCost = await GetPredictedMonthlyCost();
    } catch (e) {
      console.error('Failed to load predicted cost:', e);
      predictedMonthlyCost = '¥0.00';
    }
  }
</script>

<div class="space-y-5">
  <!-- Stats Cards -->
  <div class="grid grid-cols-4 gap-4">
    <div class="bg-white rounded-xl border border-gray-100 p-5">
      <div class="flex items-center justify-between mb-3">
        <div class="w-10 h-10 rounded-lg bg-blue-50 flex items-center justify-center">
          <svg class="w-5 h-5 text-blue-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4" />
          </svg>
        </div>
      </div>
      <div class="text-[28px] font-bold text-gray-900">{stats.totalCases}</div>
      <div class="text-[13px] text-gray-500 mt-1">{t.totalScenes || '总场景数'}</div>
    </div>
    
    <div class="bg-white rounded-xl border border-gray-100 p-5">
      <div class="flex items-center justify-between mb-3">
        <div class="w-10 h-10 rounded-lg bg-emerald-50 flex items-center justify-center">
          <svg class="w-5 h-5 text-emerald-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
          </svg>
        </div>
      </div>
      <div class="text-[28px] font-bold text-emerald-600">{stats.runningCases}</div>
      <div class="text-[13px] text-gray-500 mt-1">{t.runningScenes || '运行中'}</div>
    </div>
    
    <div class="bg-white rounded-xl border border-gray-100 p-5">
      <div class="flex items-center justify-between mb-3">
        <div class="w-10 h-10 rounded-lg bg-slate-50 flex items-center justify-center">
          <svg class="w-5 h-5 text-slate-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 9v6m4-6v6m7-3a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
        </div>
      </div>
      <div class="text-[28px] font-bold text-slate-600">{stats.stoppedCases}</div>
      <div class="text-[13px] text-gray-500 mt-1">{t.stoppedScenes || '已停止'}</div>
    </div>
    
    <div class="bg-white rounded-xl border border-gray-100 p-5">
      <div class="flex items-center justify-between mb-3">
        <div class="w-10 h-10 rounded-lg bg-red-50 flex items-center justify-center">
          <svg class="w-5 h-5 text-red-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
          </svg>
        </div>
      </div>
      <div class="text-[28px] font-bold text-red-600">{stats.errorCases}</div>
      <div class="text-[13px] text-gray-500 mt-1">{t.errorScenes || '异常'}</div>
    </div>
  </div>
  
  <!-- Quick Stats -->
  <div class="grid grid-cols-4 gap-4">
    {#each quickStats as stat}
      <div class="bg-white rounded-xl border border-gray-100 p-4">
        <div class="text-[12px] text-gray-500 mb-2">{stat.label}</div>
        <div class="flex items-end justify-between">
          <div class="text-[20px] font-bold text-gray-900">{stat.value}</div>
          <div class="text-[11px] font-medium {stat.trend === 'up' ? 'text-emerald-600' : stat.trend === 'down' ? 'text-red-600' : 'text-gray-500'}">
            {stat.change}
          </div>
        </div>
      </div>
    {/each}
  </div>
  
  <!-- Main Content Grid -->
  <div class="grid grid-cols-3 gap-5">
    <!-- Recent Cases -->
    <div class="col-span-2 bg-white rounded-xl border border-gray-100 overflow-hidden">
      <div class="px-5 py-4 border-b border-gray-100 flex items-center justify-between">
        <h3 class="text-[15px] font-semibold text-gray-900">{t.recentScenes || '最近场景'}</h3>
        <button 
          class="text-[12px] text-blue-600 hover:text-blue-700 font-medium"
          onclick={navigateToCases}
        >
          {t.viewAll || '查看全部'} →
        </button>
      </div>
      <div class="divide-y divide-gray-50">
        {#if loading}
          <div class="px-5 py-8 text-center text-[13px] text-gray-400">
            {t.loading || '加载中...'}
          </div>
        {:else if recentCases.length === 0}
          <div class="px-5 py-8 text-center text-[13px] text-gray-400">
            {t.noRecentScenes || '暂无场景'}
          </div>
        {:else}
          {#each recentCases as c}
            <div class="px-5 py-3 hover:bg-gray-50 transition-colors cursor-pointer" onclick={navigateToCases}>
              <div class="flex items-center justify-between">
                <div class="flex-1">
                  <div class="text-[13px] font-medium text-gray-900">{c.name}</div>
                  <div class="text-[11px] text-gray-500 mt-0.5">{c.type} · {c.createTime}</div>
                </div>
                <span class="inline-flex items-center gap-1.5 px-2.5 py-1 text-[11px] font-medium rounded-full {getStateColor(c.state)}">
                  <span class="w-1.5 h-1.5 rounded-full bg-current"></span>
                  {c.state}
                </span>
              </div>
            </div>
          {/each}
        {/if}
      </div>
    </div>
    
    <!-- Network Diagnostics -->
    <div class="bg-white rounded-xl border border-gray-100 overflow-hidden">
      <div class="px-5 py-4 border-b border-gray-100 flex items-center justify-between">
        <h3 class="text-[15px] font-semibold text-gray-900">{t.networkCheck || '网络诊断'}</h3>
        <button
          class="h-7 px-2.5 bg-gray-900 text-white text-[10px] font-medium rounded-lg hover:bg-gray-800 transition-colors disabled:opacity-50"
          onclick={runNetworkCheck}
          disabled={networkCheckLoading}
        >
          {networkCheckLoading ? '检测中' : '重新检测'}
        </button>
      </div>
      <div class="p-4">
        {#if networkCheckLoading}
          <div class="text-center py-8 text-[13px] text-gray-400">
            {t.loading || '加载中...'}
          </div>
        {:else if networkChecks.length === 0}
          <div class="text-center py-8 text-[13px] text-gray-400">
            {t.noNetworkData || '暂无网络诊断数据'}
          </div>
        {:else}
          <div class="space-y-2.5">
            {#each networkChecks as item}
              <div class="flex items-center justify-between py-2">
                <div class="flex items-center gap-2 flex-1 min-w-0">
                  <div class="w-2 h-2 rounded-full flex-shrink-0 {item.ok ? 'bg-emerald-500' : 'bg-red-500'}"></div>
                  <span class="text-[11px] text-gray-700 truncate">{item.name}</span>
                </div>
                <div class="flex items-center gap-2 flex-shrink-0">
                  <span class="text-[10px] text-gray-500">{item.latencyMs}ms</span>
                  <span class="text-[10px] font-medium {item.ok ? 'text-emerald-600' : 'text-red-600'} w-10 text-right">
                    {item.ok ? 'OK' : 'FAIL'}
                  </span>
                </div>
              </div>
            {/each}
          </div>
        {/if}
      </div>
    </div>
  </div>
  
  <!-- Resource Summary & Balances -->
  <div class="grid grid-cols-2 gap-5">
    <!-- Resource Summary -->
    <div class="bg-white rounded-xl border border-gray-100 overflow-hidden">
      <div class="px-5 py-4 border-b border-gray-100">
        <h3 class="text-[15px] font-semibold text-gray-900">{t.resourceSummary || '资源概览'}</h3>
      </div>
      <div class="p-5">
        {#if loading}
          <div class="text-center py-8 text-[13px] text-gray-400">
            {t.loading || '加载中...'}
          </div>
        {:else if resourceSummary.length === 0}
          <div class="text-center py-8 text-[13px] text-gray-400">
            {t.noResources || '暂无资源'}
          </div>
        {:else}
          <div class="space-y-3">
            {#each resourceSummary.slice(0, 6) as resource}
              <div class="flex items-center justify-between">
                <span class="text-[12px] text-gray-600">{resource.type}</span>
                <span class="text-[13px] font-medium text-gray-900">{resource.count}</span>
              </div>
            {/each}
          </div>
        {/if}
      </div>
    </div>
    
    <!-- Account Balances -->
    <div class="bg-white rounded-xl border border-gray-100 overflow-hidden">
      <div class="px-5 py-4 border-b border-gray-100">
        <h3 class="text-[15px] font-semibold text-gray-900">{t.accountBalance || '账户余额'}</h3>
      </div>
      <div class="p-5">
        {#if loading}
          <div class="text-center py-8 text-[13px] text-gray-400">
            {t.loading || '加载中...'}
          </div>
        {:else if balances.length === 0}
          <div class="text-center py-8 text-[13px] text-gray-400">
            {t.noBalanceData || '暂无余额数据'}
          </div>
        {:else}
          <div class="space-y-3">
            {#each balances as balance}
              <div class="flex items-center justify-between">
                <span class="text-[12px] text-gray-600">{balance.provider}</span>
                {#if balance.error}
                  <span class="text-[11px] text-red-600">{t.loadFailed || '加载失败'}</span>
                {:else}
                  <span class="text-[13px] font-medium text-gray-900">{balance.currency} {balance.amount}</span>
                {/if}
              </div>
            {/each}
          </div>
        {/if}
      </div>
    </div>
  </div>
</div>
