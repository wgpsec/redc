<script lang="ts">
  import { onMount } from 'svelte';
  import { ListCustomDeployments, StartCustomDeployment, StopCustomDeployment, DeleteCustomDeployment, BatchStartCustomDeployments, BatchStopCustomDeployments, BatchDeleteCustomDeployments, AnalyzeDeploymentError, GetActiveProfile } from '../../../wailsjs/go/main/App';
  import { EventsOn } from '../../../wailsjs/runtime/runtime.js';
  import SSHModal from '../Cases/SSHModal.svelte';
  import ScheduleDialog from '../Cases/ScheduleDialog.svelte';

  let { t, onSelectDeployment = () => {}, onRefresh = () => {} } = $props();

  interface CustomDeployment {
    id: string;
    name: string;
    template_name: string;
    state: string;
    created_at: string;
    updated_at: string;
    outputs?: Record<string, any>;
    config?: {
      provider: string;
      region: string;
      instance_type: string;
    };
  }

  let deployments = $state<CustomDeployment[]>([]);
  let loading = $state(false);
  let error = $state('');
  let expandedDeploymentId = $state('');
  let deploymentOutputs = $state<Record<string, any>>({});
  let selectedDeploymentIds = $state<Set<string>>(new Set());
  let batchMode = $state(false);
  let batchOperating = $state(false);
  let batchDeleteConfirm = $state({ show: false, count: 0 });
  let batchStopConfirm = $state({ show: false, count: 0 });
  let copiedKey = $state<string | null>(null);
  let pollInterval: number | null = null;
  
  // SSH Modal state
  let sshModal = $state<{ show: boolean; deploymentId: string; deploymentName: string }>({ 
    show: false, 
    deploymentId: '', 
    deploymentName: '' 
  });
  
  // Schedule Dialog state
  let scheduleDialog = $state<{ show: boolean; deploymentId: string; deploymentName: string; action: string }>({ 
    show: false, 
    deploymentId: '', 
    deploymentName: '', 
    action: '' 
  });

  // AI Error Analysis state
  let aiAnalyzing = $state<Record<string, boolean>>({});
  let aiAnalysisResult = $state<Record<string, string>>({});
  let aiAnalysisCompleted = $state<Record<string, boolean>>({});

  // 状态颜色配置（与创建部署页面一致）
  const stateConfig: Record<string, { label: string; color: string; bg: string; dot: string }> = {
    'pending': { label: '待部署', color: 'text-amber-600', bg: 'bg-amber-50', dot: 'bg-amber-500' },
    'starting': { label: '启动中', color: 'text-blue-600', bg: 'bg-blue-50', dot: 'bg-blue-500 animate-pulse' },
    'running': { label: '运行中', color: 'text-emerald-600', bg: 'bg-emerald-50', dot: 'bg-emerald-500' },
    'stopping': { label: '停止中', color: 'text-orange-600', bg: 'bg-orange-50', dot: 'bg-orange-500 animate-pulse' },
    'stopped': { label: '已停止', color: 'text-slate-500', bg: 'bg-slate-50', dot: 'bg-slate-400' },
    'removing': { label: '删除中', color: 'text-red-600', bg: 'bg-red-50', dot: 'bg-red-500 animate-pulse' },
    'error': { label: '错误', color: 'text-red-600', bg: 'bg-red-50', dot: 'bg-red-500' }
  };

  // 云厂商映射
  const providerLabels: Record<string, string> = {
    'alicloud': '阿里云',
    'tencentcloud': '腾讯云',
    'aws': 'AWS',
    'volcengine': '火山引擎',
    'huaweicloud': '华为云'
  };

  // 截断 ID 显示（参考预定义场景页面）
  function getShortId(id: string): string {
    return id && id.length > 8 ? id.substring(0, 8) : id;
  }

  async function loadDeployments() {
    loading = true;
    error = '';
    try {
      const result = await ListCustomDeployments();
      deployments = (result || []) as any;
      
      // 检查是否有中间态的部署，如果有则启动轮询
      checkAndStartPolling();
    } catch (err) {
      error = `加载部署列表失败: ${err}`;
      console.error('Failed to load deployments:', err);
    } finally {
      loading = false;
    }
  }

  // 检查是否有中间态的部署
  function hasTransitioningDeployments(): boolean {
    return deployments.some(d => 
      d.state === 'starting' || 
      d.state === 'stopping' || 
      d.state === 'removing'
    );
  }

  // 检查并启动/停止轮询
  function checkAndStartPolling() {
    if (hasTransitioningDeployments()) {
      // 如果有中间态的部署且轮询未启动，则启动轮询
      if (!pollInterval) {
        console.log('启动部署状态轮询');
        pollInterval = window.setInterval(() => {
          loadDeploymentsQuietly();
        }, 3000); // 每3秒轮询一次
      }
    } else {
      // 如果没有中间态的部署，停止轮询
      if (pollInterval) {
        console.log('停止部署状态轮询');
        clearInterval(pollInterval);
        pollInterval = null;
      }
    }
  }

  // 静默加载（不显示 loading 状态）
  async function loadDeploymentsQuietly() {
    try {
      const result = await ListCustomDeployments();
      deployments = (result || []) as any;
      
      // 检查是否还需要继续轮询
      checkAndStartPolling();
    } catch (err) {
      console.error('轮询加载部署列表失败:', err);
      // 轮询失败时停止轮询
      if (pollInterval) {
        clearInterval(pollInterval);
        pollInterval = null;
      }
    }
  }

  function handleSelectDeployment(deployment: CustomDeployment) {
    if (batchMode) {
      toggleDeploymentSelection(deployment.id);
    } else {
      toggleDeploymentExpand(deployment.id);
    }
  }

  async function toggleDeploymentExpand(deploymentId: string) {
    if (expandedDeploymentId === deploymentId) {
      expandedDeploymentId = '';
    } else {
      expandedDeploymentId = deploymentId;
      // 查找部署对象
      const deployment = deployments.find(d => d.id === deploymentId);
      if (deployment && deployment.outputs) {
        deploymentOutputs[deploymentId] = deployment.outputs;
      }
    }
  }

  function copyToClipboard(value: string, key: string) {
    navigator.clipboard.writeText(value).then(() => {
      copiedKey = key;
      setTimeout(() => {
        copiedKey = null;
      }, 2000);
    }).catch(err => {
      console.error('Failed to copy:', err);
    });
  }

  async function handleStart(deploymentId: string) {
    // 立即更新本地状态为"启动中"
    deployments = deployments.map(d => 
      d.id === deploymentId ? { ...d, state: 'starting' } : d
    );
    
    try {
      await StartCustomDeployment(deploymentId);
      await loadDeployments();
      onRefresh();
    } catch (err: any) {
      alert(`启动失败: ${err.message || err}`);
      // 失败后重新加载以恢复正确状态
      await loadDeployments();
    }
  }

  async function handleStop(deploymentId: string) {
    // 立即更新本地状态为"停止中"
    deployments = deployments.map(d => 
      d.id === deploymentId ? { ...d, state: 'stopping' } : d
    );
    
    try {
      await StopCustomDeployment(deploymentId);
      await loadDeployments();
      onRefresh();
    } catch (err: any) {
      alert(`停止失败: ${err.message || err}`);
      // 失败后重新加载以恢复正确状态
      await loadDeployments();
    }
  }

  let deleteConfirm = $state({ show: false, deploymentId: '', deploymentName: '' });

  async function handleDelete(deploymentId: string, deploymentName: string) {
    deleteConfirm = { show: true, deploymentId, deploymentName };
  }

  function cancelDelete() {
    deleteConfirm = { show: false, deploymentId: '', deploymentName: '' };
  }

  async function confirmDelete() {
    const { deploymentId, deploymentName } = deleteConfirm;
    deleteConfirm = { show: false, deploymentId: '', deploymentName: '' };
    
    // 立即更新本地状态为"删除中"
    deployments = deployments.map(d => 
      d.id === deploymentId ? { ...d, state: 'removing' } : d
    );
    
    try {
      await DeleteCustomDeployment(deploymentId);
      await loadDeployments();
      onRefresh();
    } catch (err: any) {
      alert(`删除失败: ${err.message || err}`);
      // 失败后重新加载以恢复正确状态
      await loadDeployments();
    }
  }

  async function handleAIAnalysis(deploymentId: string, errorMessage: string, provider: string, templateName: string) {
    if (!errorMessage) {
      alert('没有错误信息可以分析');
      return;
    }
    
    // 先检查 AI 配置
    try {
      const profile = await GetActiveProfile();
      if (!profile || !profile.aiConfig || !profile.aiConfig.apiKey) {
        alert('请先在设置中配置 AI 服务');
        return;
      }
    } catch (err: any) {
      alert(`检查 AI 配置失败: ${err.message || err}`);
      return;
    }
    
    // 开始 AI 分析
    aiAnalyzing[deploymentId] = true;
    aiAnalyzing = { ...aiAnalyzing };
    aiAnalysisResult[deploymentId] = '';
    aiAnalysisResult = { ...aiAnalysisResult };
    
    try {
      await AnalyzeDeploymentError(deploymentId, errorMessage, provider, templateName);
    } catch (err: any) {
      alert(`AI 分析失败: ${err.message || err}`);
      aiAnalyzing[deploymentId] = false;
      aiAnalyzing = { ...aiAnalyzing };
    }
  }

  function showSSHModal(deploymentId: string, deploymentName: string) {
    sshModal = { show: true, deploymentId, deploymentName };
  }

  function showScheduleDialog(deploymentId: string, deploymentName: string, action: string) {
    scheduleDialog = { show: true, deploymentId, deploymentName, action };
  }

  function handleScheduleSuccess() {
    scheduleDialog = { show: false, deploymentId: '', deploymentName: '', action: '' };
    // 可以添加成功提示
  }

  function toggleDeploymentSelection(id: string) {
    if (selectedDeploymentIds.has(id)) {
      const newSet = new Set(selectedDeploymentIds);
      newSet.delete(id);
      selectedDeploymentIds = newSet;
    } else {
      const newSet = new Set(selectedDeploymentIds);
      newSet.add(id);
      selectedDeploymentIds = newSet;
    }
  }

  function toggleBatchMode() {
    batchMode = !batchMode;
    if (!batchMode) {
      selectedDeploymentIds = new Set();
    }
  }

  function selectAll() {
    selectedDeploymentIds = new Set(deployments.map(d => d.id));
  }

  function deselectAll() {
    selectedDeploymentIds = new Set();
  }

  function getSelectedDeployments(): CustomDeployment[] {
    return deployments.filter(d => selectedDeploymentIds.has(d.id));
  }

  function showBatchDeleteConfirm() {
    batchDeleteConfirm = { show: true, count: selectedDeploymentIds.size };
  }

  function cancelBatchDelete() {
    batchDeleteConfirm = { show: false, count: 0 };
  }

  async function confirmBatchDelete() {
    batchDeleteConfirm = { show: false, count: 0 };
    batchOperating = true;
    const deploymentIds = Array.from(selectedDeploymentIds);
    try {
      await BatchDeleteCustomDeployments(deploymentIds);
      selectedDeploymentIds = new Set();
      await loadDeployments();
    } catch (e) {
      error = e.message || String(e);
    } finally {
      batchOperating = false;
    }
  }

  function showBatchStopConfirm() {
    batchStopConfirm = { show: true, count: selectedDeploymentIds.size };
  }

  function cancelBatchStop() {
    batchStopConfirm = { show: false, count: 0 };
  }

  async function confirmBatchStop() {
    batchStopConfirm = { show: false, count: 0 };
    batchOperating = true;
    const deploymentIds = Array.from(selectedDeploymentIds);
    try {
      await BatchStopCustomDeployments(deploymentIds);
      selectedDeploymentIds = new Set();
      await loadDeployments();
    } catch (e) {
      error = e.message || String(e);
    } finally {
      batchOperating = false;
    }
  }

  async function handleBatchStart() {
    batchOperating = true;
    const deploymentIds = Array.from(selectedDeploymentIds);
    try {
      await BatchStartCustomDeployments(deploymentIds);
      selectedDeploymentIds = new Set();
      await loadDeployments();
    } catch (e) {
      error = e.message || String(e);
    } finally {
      batchOperating = false;
    }
  }

  // Export batch selection info for parent component
  export function getBatchSelection() {
    return {
      mode: batchMode,
      selectedIds: Array.from(selectedDeploymentIds),
      selectedDeployments: getSelectedDeployments()
    };
  }

  function formatDate(dateStr: string): string {
    if (!dateStr) return '-';
    try {
      const date = new Date(dateStr);
      return date.toLocaleString('zh-CN', {
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit'
      });
    } catch {
      return dateStr;
    }
  }

  function handleRefresh() {
    loadDeployments();
    onRefresh();
  }

  onMount(() => {
    loadDeployments();
    
    // 设置 AI 分析事件监听
    EventsOn('ai-deployment-error-chunk', (data: any) => {
      console.log('AI chunk received:', data);
      const { deploymentId, chunk } = data;
      if (deploymentId && chunk) {
        aiAnalysisResult[deploymentId] = (aiAnalysisResult[deploymentId] || '') + chunk;
        aiAnalysisResult = { ...aiAnalysisResult };
      }
    });
    
    EventsOn('ai-deployment-error-complete', (data: any) => {
      console.log('AI complete received:', data);
      const { deploymentId, success } = data;
      if (deploymentId) {
        aiAnalyzing[deploymentId] = false;
        aiAnalyzing = { ...aiAnalyzing };
        aiAnalysisCompleted[deploymentId] = true;
        aiAnalysisCompleted = { ...aiAnalysisCompleted };
        if (!success) {
          console.error('AI analysis failed for deployment:', deploymentId);
        }
      }
    });
    
    // 组件卸载时清理轮询
    return () => {
      if (pollInterval) {
        clearInterval(pollInterval);
        pollInterval = null;
      }
    };
  });

  // 导出刷新方法供父组件调用
  export function refresh() {
    loadDeployments();
  }
</script>

<div class="deployment-list">
  <div class="list-header">
    <div class="header-left">
      <h3>{t.customDeploymentList || '自定义部署列表'}</h3>
      {#if batchMode && selectedDeploymentIds.size > 0}
        <span class="selection-count">已选择 {selectedDeploymentIds.size} 项</span>
      {/if}
    </div>
    <div class="header-actions">
      <button 
        class="btn-batch-mode" 
        class:active={batchMode}
        onclick={toggleBatchMode}
      >
        <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" 
                d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4" />
        </svg>
        {batchMode ? (t.exitBatch || '退出批量') : (t.batchOperation || '批量操作')}
      </button>
      {#if batchMode}
        <button class="btn-select-all" onclick={selectAll}>全选</button>
        <button class="btn-deselect-all" onclick={deselectAll}>取消全选</button>
        <button
          class="px-4 py-2 text-[13px] font-medium text-emerald-700 bg-emerald-50 border border-emerald-200 rounded-md hover:bg-emerald-100 transition-colors disabled:opacity-50"
          onclick={handleBatchStart}
          disabled={batchOperating || selectedDeploymentIds.size === 0}
        >
          {t.batchStart || '批量启动'}
        </button>
        <button
          class="px-4 py-2 text-[13px] font-medium text-amber-700 bg-amber-50 border border-amber-200 rounded-md hover:bg-amber-100 transition-colors disabled:opacity-50"
          onclick={showBatchStopConfirm}
          disabled={batchOperating || selectedDeploymentIds.size === 0}
        >
          {t.batchStop || '批量停止'}
        </button>
        <button
          class="px-4 py-2 text-[13px] font-medium text-red-700 bg-red-50 border border-red-200 rounded-md hover:bg-red-100 transition-colors disabled:opacity-50"
          onclick={showBatchDeleteConfirm}
          disabled={batchOperating || selectedDeploymentIds.size === 0}
        >
          {t.batchDelete || '批量删除'}
        </button>
      {/if}
      <button class="btn-refresh" onclick={handleRefresh} disabled={loading}>
        <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" 
                d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
        </svg>
        {t.refresh || '刷新'}
      </button>
    </div>
  </div>

  {#if loading}
    <div class="loading">
      <div class="spinner"></div>
      <p>加载中...</p>
    </div>
  {:else if error}
    <div class="error-message">
      <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" 
              d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
      </svg>
      <p>{error}</p>
      <button class="btn-retry" onclick={loadDeployments}>{t.retry || '重试'}</button>
    </div>
  {:else if deployments.length === 0}
    <div class="empty-state">
      <svg class="icon-large" viewBox="0 0 24 24" fill="none" stroke="currentColor">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" 
              d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4" />
      </svg>
      <p>暂无自定义部署</p>
      <p class="hint">创建一个新的自定义部署开始使用</p>
    </div>
  {:else}
    <div class="table-container">
      <table class="deployment-table">
        <thead>
          <tr>
            {#if batchMode}
              <th class="checkbox-col"></th>
            {/if}
            <th>名称</th>
            <th>模板</th>
            <th>云厂商</th>
            <th>地域</th>
            <th>状态</th>
            <th>创建时间</th>
            <th class="text-right">操作</th>
          </tr>
        </thead>
        <tbody>
          {#each deployments as deployment (deployment.id)}
            <tr 
              class:selected={!batchMode && expandedDeploymentId === deployment.id}
              class:batch-selected={batchMode && selectedDeploymentIds.has(deployment.id)}
              onclick={() => handleSelectDeployment(deployment)}
            >
              {#if batchMode}
                <td class="checkbox-col" onclick={(e) => e.stopPropagation()}>
                  <input 
                    type="checkbox" 
                    checked={selectedDeploymentIds.has(deployment.id)}
                    onchange={() => toggleDeploymentSelection(deployment.id)}
                  />
                </td>
              {/if}
              <td class="name-cell">
                <div class="name-content">
                  <div class="flex items-center gap-2">
                    <svg class="w-4 h-4 text-gray-400 transition-transform {expandedDeploymentId === deployment.id ? 'rotate-90' : ''}" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
                    </svg>
                    <span class="name">{deployment.name}</span>
                  </div>
                  <code class="id">{getShortId(deployment.id)}</code>
                </div>
              </td>
              <td>{deployment.template_name || '-'}</td>
              <td>
                {deployment.config?.provider ? providerLabels[deployment.config.provider] || deployment.config.provider : '-'}
              </td>
              <td>{deployment.config?.region || '-'}</td>
              <td>
                <span class="inline-flex items-center gap-1.5 text-[12px] font-medium {stateConfig[deployment.state]?.color || 'text-gray-600'}">
                  <span class="w-1.5 h-1.5 rounded-full {stateConfig[deployment.state]?.dot || 'bg-gray-400'}"></span>
                  {stateConfig[deployment.state]?.label || deployment.state}
                </span>
                {#if deployment.config?.is_spot_instance}
                  <span class="ml-1.5 px-1.5 py-0.5 text-[10px] font-medium text-amber-700 bg-amber-50 border border-amber-200 rounded">
                    抢占式
                  </span>
                {/if}
              </td>
              <td class="date-cell">{formatDate(deployment.created_at)}</td>
              <td class="px-5 py-3.5 text-right" onclick={(e) => e.stopPropagation()}>
                <div class="inline-flex items-center gap-1">
                  {#if deployment.state === 'starting' || deployment.state === 'stopping' || deployment.state === 'removing'}
                    <span class="px-2.5 py-1 text-[12px] font-medium text-amber-600">
                      {stateConfig[deployment.state]?.label || '处理中'}...
                    </span>
                  {:else if deployment.state === 'error'}
                    <span class="px-2.5 py-1 text-[12px] font-medium text-red-600">
                      {stateConfig[deployment.state]?.label || '错误'}
                    </span>
                    <button 
                      class="px-2.5 py-1 text-[12px] font-medium text-amber-700 bg-amber-50 rounded-md hover:bg-amber-100 transition-colors"
                      onclick={() => handleStart(deployment.id)}
                    >重试</button>
                  {:else if deployment.state !== 'running'}
                    <!-- 定时启动按钮 -->
                    <button 
                      class="p-1.5 text-gray-400 hover:text-blue-600 hover:bg-blue-50 rounded transition-colors"
                      onclick={() => showScheduleDialog(deployment.id, deployment.name, 'start')}
                      title="定时启动"
                    >
                      <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                        <path stroke-linecap="round" stroke-linejoin="round" d="M12 6v6h4.5m4.5 0a9 9 0 11-18 0 9 9 0 0118 0z" />
                      </svg>
                    </button>
                    <button 
                      class="px-2.5 py-1 text-[12px] font-medium text-emerald-700 bg-emerald-50 rounded-md hover:bg-emerald-100 transition-colors"
                      onclick={() => handleStart(deployment.id)}
                    >启动</button>
                  {:else}
                    <button 
                      class="px-2.5 py-1 text-[12px] font-medium text-blue-700 bg-blue-50 rounded-md hover:bg-blue-100 transition-colors"
                      onclick={() => showSSHModal(deployment.id, deployment.name)}
                      title="SSH 运维"
                    >
                      <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                        <path stroke-linecap="round" stroke-linejoin="round" d="M6.75 7.5l3 2.25-3 2.25m4.5 0h3m-9 8.25h13.5A2.25 2.25 0 0021 18V6a2.25 2.25 0 00-2.25-2.25H5.25A2.25 2.25 0 003 6v12a2.25 2.25 0 002.25 2.25z" />
                      </svg>
                    </button>
                    <!-- 定时停止按钮 -->
                    <button 
                      class="p-1.5 text-gray-400 hover:text-amber-600 hover:bg-amber-50 rounded transition-colors"
                      onclick={() => showScheduleDialog(deployment.id, deployment.name, 'stop')}
                      title="定时停止"
                    >
                      <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                        <path stroke-linecap="round" stroke-linejoin="round" d="M12 6v6h4.5m4.5 0a9 9 0 11-18 0 9 9 0 0118 0z" />
                      </svg>
                    </button>
                    <button 
                      class="px-2.5 py-1 text-[12px] font-medium text-amber-700 bg-amber-50 rounded-md hover:bg-amber-100 transition-colors"
                      onclick={() => handleStop(deployment.id)}
                    >停止</button>
                  {/if}
                  {#if deployment.state !== 'starting' && deployment.state !== 'stopping' && deployment.state !== 'removing'}
                    <button 
                      class="px-2.5 py-1 text-[12px] font-medium text-red-700 bg-red-50 rounded-md hover:bg-red-100 transition-colors"
                      onclick={() => handleDelete(deployment.id, deployment.name)}
                    >删除</button>
                  {/if}
                </div>
              </td>
            </tr>
            <!-- Expanded row for outputs or error -->
            {#if expandedDeploymentId === deployment.id}
              <tr class="bg-slate-50">
                <td colspan="7" class="px-5 py-4">
                  <div class="pl-6">
                    {#if deployment.state === 'error'}
                      <div class="bg-red-50 border border-red-200 rounded-lg p-4">
                        <div class="flex items-start gap-3">
                          <svg class="w-5 h-5 text-red-600 flex-shrink-0 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                          </svg>
                          <div class="flex-1">
                            <h4 class="text-[13px] font-semibold text-red-900">部署失败</h4>
                            <p class="text-[12px] text-red-700 mt-1">请检查配置后重试。错误详情：</p>
                            <pre class="mt-2 p-3 bg-white rounded border border-red-200 text-[11px] text-red-800 overflow-x-auto whitespace-pre-wrap">{deployment.outputs?.error_message || '未知错误'}</pre>
                            
                            {#if aiAnalyzing[deployment.id] || aiAnalysisCompleted[deployment.id]}
                              <div class="mt-3 p-3 bg-blue-50 rounded border border-blue-200">
                                {#if aiAnalyzing[deployment.id]}
                                  <div class="flex items-center gap-2 mb-2">
                                    <svg class="animate-spin h-4 w-4 text-blue-600" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                                      <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                                      <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                                    </svg>
                                    <span class="text-[12px] text-blue-700">AI 正在分析错误...</span>
                                  </div>
                                {/if}
                                {#if aiAnalysisResult[deployment.id]}
                                  <pre class="text-[11px] text-blue-800 whitespace-pre-wrap">{aiAnalysisResult[deployment.id]}</pre>
                                {/if}
                              </div>
                            {:else}
                              <button 
                                class="mt-3 px-3 py-1.5 text-[12px] font-medium text-blue-700 bg-blue-50 border border-blue-200 rounded-md hover:bg-blue-100 transition-colors flex items-center gap-1.5"
                                onclick={() => handleAIAnalysis(deployment.id, deployment.outputs?.error_message || '', deployment.config?.provider || '', deployment.template_name || '')}
                              >
                                <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z" />
                                </svg>
                                AI 分析错误原因
                              </button>
                            {/if}
                          </div>
                        </div>
                      </div>
                    {:else if deployment.state === 'running'}
                      {#if deploymentOutputs[deployment.id] && Object.keys(deploymentOutputs[deployment.id]).length > 0}
                        <div class="grid grid-cols-2 gap-3">
                          {#each Object.entries(deploymentOutputs[deployment.id]) as [key, value]}
                            <div class="bg-white rounded-lg p-3 border border-gray-100 group relative">
                              <div class="flex items-center justify-between mb-1">
                                <div class="text-[11px] text-gray-500 uppercase tracking-wide">{key}</div>
                                <button 
                                  class="opacity-0 group-hover:opacity-100 transition-opacity p-1 hover:bg-gray-100 rounded flex items-center gap-1"
                                  onclick={(e) => { e.stopPropagation(); copyToClipboard(String(value), key); }}
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
                        <div class="text-[13px] text-gray-500">暂无输出信息</div>
                      {/if}
                    {:else}
                      <div class="text-[13px] text-gray-500">部署未运行，无输出信息</div>
                    {/if}
                  </div>
                </td>
              </tr>
            {/if}
          {/each}
        </tbody>
      </table>
    </div>
  {/if}
</div>

<!-- SSH Modal -->
{#if sshModal.show}
  <SSHModal
    t={t}
    caseId={sshModal.deploymentId}
    caseName={sshModal.deploymentName}
    onClose={() => { sshModal = { show: false, deploymentId: '', deploymentName: '' }; }}
  />
{/if}

<!-- Schedule Dialog -->
{#if scheduleDialog.show}
  <ScheduleDialog
    t={t}
    caseId={scheduleDialog.deploymentId}
    caseName={scheduleDialog.deploymentName}
    action={scheduleDialog.action}
    onClose={() => { scheduleDialog = { show: false, deploymentId: '', deploymentName: '', action: '' }; }}
    onSuccess={handleScheduleSuccess}
  />
{/if}

<style>
  .deployment-list {
    display: flex;
    flex-direction: column;
    height: 100%;
    background: white;
    border-radius: 8px;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
  }

  .list-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 16px 20px;
    border-bottom: 1px solid #e5e7eb;
  }

  .header-left {
    display: flex;
    align-items: center;
    gap: 16px;
  }

  .header-actions {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .list-header h3 {
    margin: 0;
    font-size: 18px;
    font-weight: 600;
    color: #111827;
  }

  .selection-count {
    padding: 4px 12px;
    background: #dbeafe;
    color: #1e40af;
    border-radius: 12px;
    font-size: 13px;
    font-weight: 500;
  }

  .btn-batch-mode {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 8px 16px;
    background: white;
    border: 1px solid #d1d5db;
    border-radius: 6px;
    font-size: 14px;
    color: #374151;
    cursor: pointer;
    transition: all 0.2s;
  }

  .btn-batch-mode:hover {
    background: #f3f4f6;
  }

  .btn-batch-mode.active {
    background: #3b82f6;
    border-color: #3b82f6;
    color: white;
  }

  .btn-select-all,
  .btn-deselect-all {
    padding: 8px 12px;
    background: white;
    border: 1px solid #d1d5db;
    border-radius: 6px;
    font-size: 13px;
    color: #374151;
    cursor: pointer;
    transition: all 0.2s;
  }

  .btn-select-all:hover,
  .btn-deselect-all:hover {
    background: #f3f4f6;
  }

  .btn-refresh {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 8px 16px;
    background: #f3f4f6;
    border: 1px solid #d1d5db;
    border-radius: 6px;
    font-size: 14px;
    color: #374151;
    cursor: pointer;
    transition: all 0.2s;
  }

  .btn-refresh:hover:not(:disabled) {
    background: #e5e7eb;
  }

  .btn-refresh:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .icon {
    width: 16px;
    height: 16px;
  }

  .loading {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 60px 20px;
    color: #6b7280;
  }

  .spinner {
    width: 40px;
    height: 40px;
    border: 3px solid #e5e7eb;
    border-top-color: #3b82f6;
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
    margin-bottom: 16px;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }

  .error-message {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 60px 20px;
    color: #dc2626;
  }

  .error-message .icon {
    width: 48px;
    height: 48px;
    margin-bottom: 12px;
  }

  .error-message p {
    margin: 0 0 16px 0;
    text-align: center;
  }

  .btn-retry {
    padding: 8px 20px;
    background: #dc2626;
    color: white;
    border: none;
    border-radius: 6px;
    font-size: 14px;
    cursor: pointer;
    transition: background 0.2s;
  }

  .btn-retry:hover {
    background: #b91c1c;
  }

  .empty-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 80px 20px;
    color: #6b7280;
  }

  .icon-large {
    width: 64px;
    height: 64px;
    margin-bottom: 16px;
    opacity: 0.5;
  }

  .empty-state p {
    margin: 0;
    font-size: 16px;
  }

  .empty-state .hint {
    margin-top: 8px;
    font-size: 14px;
    color: #9ca3af;
  }

  .table-container {
    flex: 1;
    overflow: auto;
  }

  .deployment-table {
    width: 100%;
    border-collapse: collapse;
  }

  .deployment-table thead {
    position: sticky;
    top: 0;
    background: #f9fafb;
    z-index: 1;
  }

  .deployment-table th {
    padding: 12px 16px;
    text-align: left;
    font-size: 12px;
    font-weight: 600;
    color: #6b7280;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    border-bottom: 1px solid #e5e7eb;
  }

  .deployment-table tbody tr {
    cursor: pointer;
    transition: background 0.15s;
  }

  .deployment-table tbody tr:hover {
    background: #f9fafb;
  }

  .deployment-table tbody tr.selected {
    background: #eff6ff;
  }

  .deployment-table tbody tr.batch-selected {
    background: #dbeafe;
  }

  .checkbox-col {
    width: 40px;
    text-align: center;
  }

  .checkbox-col input[type="checkbox"] {
    width: 16px;
    height: 16px;
    cursor: pointer;
  }

  .deployment-table td {
    padding: 12px 16px;
    font-size: 14px;
    color: #374151;
    border-bottom: 1px solid #f3f4f6;
  }

  .name-cell {
    max-width: 250px;
  }

  .name-content {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .name {
    font-weight: 500;
    color: #111827;
  }

  .id {
    font-size: 12px;
    color: #9ca3af;
    font-family: monospace;
    background: #f3f4f6;
    padding: 2px 6px;
    border-radius: 4px;
  }

  .state-badge {
    display: inline-block;
    padding: 4px 10px;
    border-radius: 12px;
    font-size: 12px;
    font-weight: 500;
    background: currentColor;
    color: white;
    opacity: 0.9;
  }

  .date-cell {
    color: #6b7280;
    font-size: 13px;
  }
</style>

<!-- Batch Delete Confirmation Modal -->
{#if batchDeleteConfirm.show}
  <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
  <div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onclick={cancelBatchDelete}>
    <div class="bg-white rounded-xl shadow-xl max-w-sm w-full mx-4 overflow-hidden" onclick={(e) => e.stopPropagation()}>
      <div class="px-6 py-5">
        <div class="flex items-center gap-3 mb-3">
          <div class="w-10 h-10 rounded-full bg-red-100 flex items-center justify-center">
            <svg class="w-5 h-5 text-red-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
            </svg>
          </div>
          <div>
            <h3 class="text-[15px] font-semibold text-gray-900">{t.confirmBatchDelete}</h3>
            <p class="text-[13px] text-gray-500">{t.cannotUndo}</p>
          </div>
        </div>
        <p class="text-[14px] text-gray-600 mb-4">
          {t.confirmBatchDeleteMessage || '确定要删除选中的'} {batchDeleteConfirm.count} {t.scenes || '个场景'}?
        </p>
        <div class="flex justify-end gap-2">
          <button 
            class="px-4 py-2 text-[13px] font-medium text-gray-700 bg-white border border-gray-100 rounded-lg hover:bg-gray-50 transition-colors"
            onclick={cancelBatchDelete}
          >{t.cancel}</button>
          <button 
            class="px-4 py-2 text-[13px] font-medium text-white bg-red-600 rounded-lg hover:bg-red-700 transition-colors"
            onclick={confirmBatchDelete}
          >{t.delete}</button>
        </div>
      </div>
    </div>
  </div>
{/if}

<!-- Batch Stop Confirmation Modal -->
{#if batchStopConfirm.show}
  <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
  <div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onclick={cancelBatchStop}>
    <div class="bg-white rounded-xl shadow-xl max-w-sm w-full mx-4 overflow-hidden" onclick={(e) => e.stopPropagation()}>
      <div class="px-6 py-5">
        <div class="flex items-center gap-3 mb-3">
          <div class="w-10 h-10 rounded-full bg-amber-100 flex items-center justify-center">
            <svg class="w-5 h-5 text-amber-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
            </svg>
          </div>
          <div>
            <h3 class="text-[15px] font-semibold text-gray-900">{t.confirmBatchStop}</h3>
            <p class="text-[13px] text-gray-500">{t.cannotUndo}</p>
          </div>
        </div>
        <p class="text-[14px] text-gray-600 mb-4">
          {t.confirmBatchStopMessage || '确定要停止选中的'} {batchStopConfirm.count} {t.scenes || '个场景'}?
        </p>
        <div class="flex justify-end gap-2">
          <button 
            class="px-4 py-2 text-[13px] font-medium text-gray-700 bg-white border border-gray-100 rounded-lg hover:bg-gray-50 transition-colors"
            onclick={cancelBatchStop}
          >{t.cancel}</button>
          <button 
            class="px-4 py-2 text-[13px] font-medium text-white bg-amber-600 rounded-lg hover:bg-amber-700 transition-colors"
            onclick={confirmBatchStop}
          >{t.stop}</button>
        </div>
      </div>
    </div>
  </div>
{/if}

<!-- Single Delete Confirmation Modal -->
{#if deleteConfirm.show}
  <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
  <div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onclick={cancelDelete}>
    <div class="bg-white rounded-xl shadow-xl max-w-sm w-full mx-4 overflow-hidden" onclick={(e) => e.stopPropagation()}>
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
        <p class="text-[14px] text-gray-600 mb-4">
          确定要删除部署 "{deleteConfirm.deploymentName}" 吗？
        </p>
        <div class="flex justify-end gap-2">
          <button 
            class="px-4 py-2 text-[13px] font-medium text-gray-700 bg-white border border-gray-100 rounded-lg hover:bg-gray-50 transition-colors"
            onclick={cancelDelete}
          >取消</button>
          <button 
            class="px-4 py-2 text-[13px] font-medium text-white bg-red-600 rounded-lg hover:bg-red-700 transition-colors"
            onclick={confirmDelete}
          >删除</button>
        </div>
      </div>
    </div>
  </div>
{/if}
