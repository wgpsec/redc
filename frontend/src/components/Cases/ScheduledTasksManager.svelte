<script>
  import { onMount, onDestroy } from 'svelte';
  import { ListScheduledTasks, CancelScheduledTask } from '../../../wailsjs/go/main/App.js';

  let { t, refresh: externalRefresh = null } = $props();
  
  let tasks = $state([]);
  let loading = $state(false);
  let error = $state('');
  let expanded = $state(false);
  let cancelConfirm = $state({ show: false, taskId: null, taskName: '' });
  let refreshInterval = null;

  onMount(async () => {
    await loadTasks();
    // 每30秒刷新一次任务列表
    refreshInterval = setInterval(loadTasks, 30000);
  });

  onDestroy(() => {
    if (refreshInterval) {
      clearInterval(refreshInterval);
    }
  });

  // 暴露 refresh 方法给父组件
  // svelte-ignore state_referenced_locally
  if (externalRefresh) {
    externalRefresh.current = loadTasks;
  }

  async function loadTasks() {
    try {
      loading = true;
      const allTasks = await ListScheduledTasks();
      // 只显示待执行的任务
      tasks = (allTasks || []).filter(t => t.status === 'pending');
    } catch (e) {
      console.error('Failed to load scheduled tasks:', e);
      error = e.message || String(e);
    } finally {
      loading = false;
    }
  }

  function showCancelConfirm(taskId, caseName, action) {
    cancelConfirm = { show: true, taskId, taskName: `${caseName} (${action === 'start' ? t.start : t.stop})` };
  }

  function closeCancelConfirm() {
    cancelConfirm = { show: false, taskId: null, taskName: '' };
  }

  async function confirmCancel() {
    const taskId = cancelConfirm.taskId;
    closeCancelConfirm();
    
    try {
      await CancelScheduledTask(taskId);
      await loadTasks();
    } catch (e) {
      error = e.message || String(e);
    }
  }

  function formatTime(timeStr) {
    try {
      const date = new Date(timeStr);
      return date.toLocaleString('zh-CN', {
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit',
        hour12: false
      });
    } catch (e) {
      return timeStr;
    }
  }

  function getTimeRemaining(scheduledAt) {
    try {
      const now = new Date();
      const scheduled = new Date(scheduledAt);
      const diff = Number(scheduled) - Number(now);
      
      if (diff <= 0) {
        return t.executing || '执行中';
      }
      
      const hours = Math.floor(Number(diff) / (1000 * 60 * 60));
      const minutes = Math.floor(Number(diff) % (1000 * 60 * 60) / (1000 * 60));
      const seconds = Math.floor(Number(diff) % (1000 * 60) / 1000);
      
      if (hours > 0) {
        return `${hours}${t.hour || '小时'}${minutes}${t.minute || '分钟'}`;
      } else if (minutes > 0) {
        return `${minutes}${t.minute || '分钟'}${seconds}${t.second || '秒'}`;
      } else {
        return `${seconds}${t.second || '秒'}`;
      }
    } catch (e) {
      return '-';
    }
  }

  function getActionLabel(action) {
    return action === 'start' ? t.start : t.stop;
  }

  function getActionColor(action) {
    return action === 'start' 
      ? 'text-emerald-700 bg-emerald-50' 
      : 'text-amber-700 bg-amber-50';
  }
</script>

{#if error}
  <div class="flex items-center gap-3 px-4 py-3 bg-red-50 border border-red-100 rounded-lg mb-5">
    <svg class="w-4 h-4 text-red-500 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
      <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z" />
    </svg>
    <span class="text-[13px] text-red-700 flex-1">{error}</span>
    <button class="text-red-400 hover:text-red-600" onclick={() => error = ''} aria-label="关闭错误">
      <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
        <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
      </svg>
    </button>
  </div>
{/if}

<div class="bg-white rounded-xl border border-gray-100 overflow-hidden mb-5">
  <!-- Header -->
  <button 
    class="w-full px-5 py-4 flex items-center justify-between hover:bg-gray-50 transition-colors"
    onclick={() => expanded = !expanded}
  >
    <div class="flex items-center gap-3">
      <svg class="w-5 h-5 text-blue-600" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
        <path stroke-linecap="round" stroke-linejoin="round" d="M12 6v6h4.5m4.5 0a9 9 0 11-18 0 9 9 0 0118 0z" />
      </svg>
      <div class="text-left">
        <h3 class="text-[14px] font-semibold text-gray-900">{t.scheduledTasks || '定时任务'}</h3>
        <p class="text-[12px] text-gray-500">{t.scheduledTasksDesc || '查看和管理所有待执行的定时任务'}</p>
      </div>
    </div>
    <div class="flex items-center gap-3">
      {#if tasks.length > 0}
        <span class="px-2.5 py-1 text-[12px] font-medium text-blue-700 bg-blue-50 rounded-full">
          {tasks.length} {t.pending || '待执行'}
        </span>
      {/if}
      <svg 
        class="w-5 h-5 text-gray-400 transition-transform {expanded ? 'rotate-180' : ''}" 
        fill="none" 
        viewBox="0 0 24 24" 
        stroke="currentColor" 
        stroke-width="2"
      >
        <path stroke-linecap="round" stroke-linejoin="round" d="M19 9l-7 7-7-7" />
      </svg>
    </div>
  </button>

  <!-- Content -->
  {#if expanded}
    <div class="border-t border-gray-100">
      {#if loading}
        <div class="px-5 py-8 flex items-center justify-center">
          <div class="w-6 h-6 border-2 border-gray-100 border-t-gray-900 rounded-full animate-spin"></div>
        </div>
      {:else if tasks.length === 0}
        <div class="px-5 py-8 text-center">
          <svg class="w-12 h-12 mx-auto text-gray-300 mb-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1">
            <path stroke-linecap="round" stroke-linejoin="round" d="M12 6v6h4.5m4.5 0a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
          <p class="text-[13px] text-gray-500">{t.noScheduledTasks || '暂无待执行的定时任务'}</p>
        </div>
      {:else}
        <div class="divide-y divide-gray-100">
          {#each tasks as task}
            <div class="px-5 py-4 hover:bg-gray-50 transition-colors">
              <div class="flex items-center justify-between">
                <div class="flex-1">
                  <div class="flex items-center gap-2 mb-2">
                    <span class="text-[14px] font-medium text-gray-900">{task.caseName}</span>
                    <span class="px-2 py-0.5 text-[11px] font-medium rounded {getActionColor(task.action)}">
                      {getActionLabel(task.action)}
                    </span>
                  </div>
                  <div class="flex items-center gap-4 text-[12px] text-gray-500">
                    <div class="flex items-center gap-1.5">
                      <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                        <path stroke-linecap="round" stroke-linejoin="round" d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
                      </svg>
                      <span>{formatTime(task.scheduledAt)}</span>
                    </div>
                    <div class="flex items-center gap-1.5">
                      <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                        <path stroke-linecap="round" stroke-linejoin="round" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                      </svg>
                      <span>{t.remaining || '剩余'}: {getTimeRemaining(task.scheduledAt)}</span>
                    </div>
                  </div>
                </div>
                <button 
                  class="px-3 py-1.5 text-[12px] font-medium text-red-700 bg-red-50 rounded-md hover:bg-red-100 transition-colors"
                  onclick={() => showCancelConfirm(task.id, task.caseName, task.action)}
                >
                  {t.cancel || '取消'}
                </button>
              </div>
            </div>
          {/each}
        </div>
      {/if}
    </div>
  {/if}
</div>

<!-- Cancel Confirmation Modal -->
{#if cancelConfirm.show}
  <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
  <div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onclick={closeCancelConfirm}>
    <div class="bg-white rounded-xl shadow-xl max-w-sm w-full mx-4 overflow-hidden" onclick={(e) => e.stopPropagation()}>
      <div class="px-6 py-5">
        <div class="flex items-center gap-3 mb-3">
          <div class="w-10 h-10 rounded-full bg-amber-100 flex items-center justify-center">
            <svg class="w-5 h-5 text-amber-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
            </svg>
          </div>
          <div>
            <h3 class="text-[15px] font-semibold text-gray-900">{t.confirmCancelTask || '确认取消任务'}</h3>
            <p class="text-[13px] text-gray-500">{t.cancelTaskWarning || '此操作不可撤销'}</p>
          </div>
        </div>
        <p class="text-[13px] text-gray-600">
          {t.confirmCancelTaskMessage || '确定要取消定时任务'} <span class="font-medium text-gray-900">"{cancelConfirm.taskName}"</span>?
        </p>
      </div>
      <div class="px-6 py-4 bg-gray-50 flex justify-end gap-2">
        <button 
          class="px-4 py-2 text-[13px] font-medium text-gray-700 bg-white border border-gray-100 rounded-lg hover:bg-gray-50 transition-colors"
          onclick={closeCancelConfirm}
        >{t.cancel || '取消'}</button>
        <button 
          class="px-4 py-2 text-[13px] font-medium text-white bg-amber-600 rounded-lg hover:bg-amber-700 transition-colors"
          onclick={confirmCancel}
        >{t.confirm || '确认'}</button>
      </div>
    </div>
  </div>
{/if}
