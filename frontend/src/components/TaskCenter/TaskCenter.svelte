<script>
  import { onMount, onDestroy } from 'svelte';
  import { ListAllScheduledTasks, ListCases, ScheduleTaskFull, CancelScheduledTask } from '../../../wailsjs/go/main/App.js';

  let { t } = $props();

  let tasks = $state([]);
  let cases = $state([]);
  let loading = $state(false);
  let error = $state('');
  let refreshInterval = null;
  let showHistory = $state(false);
  let showCreateForm = $state(false);
  let cancelConfirm = $state({ show: false, taskId: null, taskName: '' });
  let resultModal = $state({ show: false, result: '', title: '' });

  // Create form state
  let formCaseId = $state('');
  let formCaseName = $state('');
  let formAction = $state('start');
  let formScheduleType = $state('relative');
  let formRelativeHours = $state(1);
  let formRelativeMinutes = $state(0);
  let formAbsoluteDate = $state('');
  let formAbsoluteTime = $state('');
  let formRepeatType = $state('once');
  let formRepeatInterval = $state(60);
  let formSSHCommand = $state('');
  let formAutoStopHours = $state(2);
  let formNotifyEnabled = $state(false);
  let formLoading = $state(false);
  let formError = $state('');

  let pendingTasks = $derived(tasks.filter(t => t.status === 'pending' || t.status === 'executing'));
  let completedTasks = $derived(tasks.filter(t => t.status === 'completed'));
  let failedTasks = $derived(tasks.filter(t => t.status === 'failed'));
  let cancelledTasks = $derived(tasks.filter(t => t.status === 'cancelled'));
  let historyTasks = $derived(tasks.filter(t => t.status !== 'pending' && t.status !== 'executing'));

  onMount(async () => {
    await loadTasks();
    refreshInterval = setInterval(loadTasks, 15000);
    const now = new Date();
    now.setHours(now.getHours() + 1);
    formAbsoluteDate = now.toISOString().split('T')[0];
    formAbsoluteTime = now.toTimeString().slice(0, 5);
  });

  onDestroy(() => {
    if (refreshInterval) clearInterval(refreshInterval);
  });

  async function loadTasks() {
    try {
      loading = true;
      tasks = (await ListAllScheduledTasks()) || [];
    } catch (e) {
      error = e.message || String(e);
    } finally {
      loading = false;
    }
  }

  async function loadCases() {
    try {
      const result = await ListCases();
      cases = result || [];
    } catch (e) {
      console.error('Failed to load cases:', e);
    }
  }

  function openCreateForm() {
    showCreateForm = true;
    formError = '';
    formAction = 'start';
    formSSHCommand = '';
    formAutoStopHours = 2;
    formNotifyEnabled = false;
    loadCases();
  }

  function onCaseSelect(e) {
    const caseId = e.target.value;
    formCaseId = caseId;
    const c = cases.find(c => (c.id || c.Id) === caseId);
    formCaseName = c ? (c.name || c.Name || caseId) : caseId;
  }

  async function handleCreate() {
    if (!formCaseId) { formError = t.taskSelectCase || '请选择场景'; return; }
    if (formAction === 'ssh_command' && !formSSHCommand.trim()) {
      formError = t.sshCommandRequired || '请输入 SSH 命令';
      return;
    }

    formLoading = true;
    formError = '';
    try {
      let scheduledAt;
      let action = formAction;
      let sshCommand = '';
      let repeatType = formRepeatType;
      let repeatInterval = formRepeatType === 'interval' ? formRepeatInterval : 0;

      if (action === 'auto_stop') {
        // Auto-stop: schedule N hours from now, always once
        scheduledAt = new Date(Date.now() + formAutoStopHours * 60 * 60 * 1000);
        repeatType = 'once';
        repeatInterval = 0;
      } else if (formScheduleType === 'relative') {
        scheduledAt = new Date(Date.now() + (formRelativeHours * 60 + formRelativeMinutes) * 60 * 1000);
      } else {
        scheduledAt = new Date(`${formAbsoluteDate}T${formAbsoluteTime}:00`);
      }

      if (scheduledAt <= new Date()) {
        formError = t.scheduleTimeInvalid || '计划时间必须晚于当前时间';
        formLoading = false;
        return;
      }

      if (action === 'ssh_command') {
        sshCommand = formSSHCommand.trim();
      }

      await ScheduleTaskFull(formCaseId, formCaseName, action, scheduledAt, repeatType, repeatInterval, sshCommand, formNotifyEnabled);
      showCreateForm = false;
      await loadTasks();
    } catch (e) {
      formError = e.message || String(e);
    } finally {
      formLoading = false;
    }
  }

  function showCancelDialog(taskId, caseName, action) {
    cancelConfirm = { show: true, taskId, taskName: `${caseName} (${getActionLabel(action)})` };
  }

  async function confirmCancel() {
    const taskId = cancelConfirm.taskId;
    cancelConfirm = { show: false, taskId: null, taskName: '' };
    try {
      await CancelScheduledTask(taskId);
      await loadTasks();
    } catch (e) {
      error = e.message || String(e);
    }
  }

  function showResult(task) {
    resultModal = { show: true, result: task.taskResult || '', title: `${task.caseName} - ${getActionLabel(task.action)}` };
  }

  function formatTime(timeStr) {
    try {
      return new Date(timeStr).toLocaleString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: false });
    } catch { return timeStr; }
  }

  function getTimeRemaining(scheduledAt) {
    try {
      const diff = Number(new Date(scheduledAt)) - Number(new Date());
      if (diff <= 0) return t.executing || '执行中';
      const hours = Math.floor(diff / (1000 * 60 * 60));
      const minutes = Math.floor(diff % (1000 * 60 * 60) / (1000 * 60));
      if (hours > 0) return `${hours}${t.hour || '小时'}${minutes}${t.minute || '分钟'}`;
      return `${minutes}${t.minute || '分钟'}`;
    } catch { return '-'; }
  }

  function getRepeatLabel(task) {
    switch (task.repeatType) {
      case 'daily': return t.repeatDaily || '每天';
      case 'weekly': return t.repeatWeekly || '每周';
      case 'interval': return `${t.repeatEvery || '每'}${task.repeatInterval}${t.minute || '分钟'}`;
      default: return t.repeatOnce || '单次';
    }
  }

  function getActionLabel(action) {
    switch (action) {
      case 'start': return t.start || '启动';
      case 'stop': return t.stop || '停止';
      case 'ssh_command': return t.sshCommand || 'SSH 命令';
      case 'auto_stop': return t.autoStop || '自动停机';
      default: return action;
    }
  }

  function getActionBadge(action) {
    switch (action) {
      case 'start': return { cls: 'text-emerald-700 bg-emerald-50', icon: '▶' };
      case 'stop': return { cls: 'text-amber-700 bg-amber-50', icon: '⏹' };
      case 'ssh_command': return { cls: 'text-cyan-700 bg-cyan-50', icon: '⌨' };
      case 'auto_stop': return { cls: 'text-orange-700 bg-orange-50', icon: '⏱' };
      default: return { cls: 'text-gray-600 bg-gray-100', icon: '?' };
    }
  }

  function getStatusBadge(status) {
    switch (status) {
      case 'pending': return { text: t.pending || '待执行', cls: 'text-blue-700 bg-blue-50' };
      case 'executing': return { text: t.executing || '执行中', cls: 'text-amber-700 bg-amber-50' };
      case 'completed': return { text: t.completed || '已完成', cls: 'text-emerald-700 bg-emerald-50' };
      case 'failed': return { text: t.failed || '失败', cls: 'text-red-700 bg-red-50' };
      case 'cancelled': return { text: t.cancelled || '已取消', cls: 'text-gray-600 bg-gray-100' };
      default: return { text: status, cls: 'text-gray-600 bg-gray-100' };
    }
  }
</script>

<div class="max-w-5xl mx-auto">
  <!-- Header -->
  <div class="flex items-center justify-between mb-5">
    <div>
      <h2 class="text-lg font-semibold text-gray-900">{t.taskCenter || '任务中心'}</h2>
      <p class="text-[12px] text-gray-500 mt-0.5">{t.taskCenterDesc || '统一管理所有定时任务，支持单次和周期性任务'}</p>
    </div>
    <div class="flex items-center gap-2">
      <button
        class="px-3 py-1.5 text-[12px] font-medium text-gray-700 bg-white border border-gray-200 rounded-lg hover:bg-gray-50 transition-colors flex items-center gap-1.5 cursor-pointer"
        onclick={loadTasks}
      >
        <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
        </svg>
        {t.refresh || '刷新'}
      </button>
      <button
        class="px-3 py-1.5 text-[12px] font-medium text-white bg-gray-900 rounded-lg hover:bg-gray-800 transition-colors flex items-center gap-1.5 cursor-pointer"
        onclick={openCreateForm}
      >
        <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
        </svg>
        {t.taskCreate || '创建任务'}
      </button>
    </div>
  </div>

  <!-- Error -->
  {#if error}
    <div class="flex items-center gap-3 px-4 py-3 bg-red-50 border border-red-100 rounded-lg mb-4">
      <svg class="w-4 h-4 text-red-500 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
        <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z" />
      </svg>
      <span class="text-[13px] text-red-700 flex-1">{error}</span>
      <button class="text-red-400 hover:text-red-600 cursor-pointer" onclick={() => error = ''}>
        <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" /></svg>
      </button>
    </div>
  {/if}

  <!-- Stats Cards -->
  <div class="grid grid-cols-4 gap-3 mb-5">
    <div class="bg-white rounded-xl border border-gray-100 p-4">
      <div class="text-[11px] text-gray-500 mb-1">{t.pending || '待执行'}</div>
      <div class="text-2xl font-bold text-blue-600">{pendingTasks.length}</div>
    </div>
    <div class="bg-white rounded-xl border border-gray-100 p-4">
      <div class="text-[11px] text-gray-500 mb-1">{t.completed || '已完成'}</div>
      <div class="text-2xl font-bold text-emerald-600">{completedTasks.length}</div>
    </div>
    <div class="bg-white rounded-xl border border-gray-100 p-4">
      <div class="text-[11px] text-gray-500 mb-1">{t.failed || '失败'}</div>
      <div class="text-2xl font-bold text-red-600">{failedTasks.length}</div>
    </div>
    <div class="bg-white rounded-xl border border-gray-100 p-4">
      <div class="text-[11px] text-gray-500 mb-1">{t.cancelled || '已取消'}</div>
      <div class="text-2xl font-bold text-gray-500">{cancelledTasks.length}</div>
    </div>
  </div>

  <!-- Pending Tasks -->
  <div class="bg-white rounded-xl border border-gray-100 overflow-hidden mb-5">
    <div class="px-5 py-4 border-b border-gray-100 flex items-center justify-between">
      <div class="flex items-center gap-2">
        <svg class="w-5 h-5 text-blue-600" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M12 6v6h4.5m4.5 0a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
        <h3 class="text-[14px] font-semibold text-gray-900">{t.pendingTasks || '待执行任务'}</h3>
        {#if pendingTasks.length > 0}
          <span class="px-2 py-0.5 text-[11px] font-medium text-blue-700 bg-blue-50 rounded-full">{pendingTasks.length}</span>
        {/if}
      </div>
    </div>

    {#if loading && tasks.length === 0}
      <div class="px-5 py-8 flex items-center justify-center">
        <div class="w-6 h-6 border-2 border-gray-100 border-t-gray-900 rounded-full animate-spin"></div>
      </div>
    {:else if pendingTasks.length === 0}
      <div class="px-5 py-8 text-center">
        <svg class="w-10 h-10 mx-auto text-gray-300 mb-2" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1">
          <path stroke-linecap="round" stroke-linejoin="round" d="M12 6v6h4.5m4.5 0a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
        <p class="text-[13px] text-gray-500">{t.noScheduledTasks || '暂无待执行的定时任务'}</p>
      </div>
    {:else}
      <div class="divide-y divide-gray-50">
        {#each pendingTasks as task (task.id)}
          <div class="px-5 py-3.5 hover:bg-gray-50 transition-colors">
            <div class="flex items-center justify-between">
              <div class="flex-1 min-w-0">
                <div class="flex items-center gap-2 mb-1.5">
                  <span class="text-[13px] font-medium text-gray-900 truncate">{task.caseName}</span>
                  <span class="px-1.5 py-0.5 text-[10px] font-medium rounded {getActionBadge(task.action).cls}">
                    {getActionBadge(task.action).icon} {getActionLabel(task.action)}
                  </span>
                  {#if task.repeatType && task.repeatType !== 'once'}
                    <span class="px-1.5 py-0.5 text-[10px] font-medium text-purple-700 bg-purple-50 rounded">
                      🔄 {getRepeatLabel(task)}
                    </span>
                  {/if}
                  {#if task.notifyEnabled}
                    <span class="px-1.5 py-0.5 text-[10px] font-medium text-indigo-700 bg-indigo-50 rounded" title={t.notifyOnComplete || '完成通知'}>🔔</span>
                  {/if}
                  {#if task.status === 'executing'}
                    <span class="px-1.5 py-0.5 text-[10px] font-medium text-amber-700 bg-amber-50 rounded flex items-center gap-1">
                      <svg class="w-3 h-3 animate-spin" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/></svg>
                      {t.executing || '执行中'}
                    </span>
                  {/if}
                </div>
                <div class="flex items-center gap-4 text-[11px] text-gray-500">
                  <span>📅 {formatTime(task.scheduledAt)}</span>
                  {#if task.status === 'pending'}
                    <span>⏳ {t.remaining || '剩余'}: {getTimeRemaining(task.scheduledAt)}</span>
                  {/if}
                  {#if task.action === 'ssh_command' && task.sshCommand}
                    <span class="text-cyan-600 font-mono truncate max-w-[250px]" title={task.sshCommand}>$ {task.sshCommand}</span>
                  {/if}
                </div>
              </div>
              {#if task.status === 'pending'}
                <button
                  class="px-2.5 py-1 text-[11px] font-medium text-red-700 bg-red-50 rounded-md hover:bg-red-100 transition-colors cursor-pointer"
                  onclick={() => showCancelDialog(task.id, task.caseName, task.action)}
                >{t.cancel || '取消'}</button>
              {/if}
            </div>
          </div>
        {/each}
      </div>
    {/if}
  </div>

  <!-- History -->
  <div class="bg-white rounded-xl border border-gray-100 overflow-hidden">
    <button
      class="w-full px-5 py-4 flex items-center justify-between hover:bg-gray-50 transition-colors cursor-pointer"
      onclick={() => showHistory = !showHistory}
    >
      <div class="flex items-center gap-2">
        <svg class="w-5 h-5 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M19.5 14.25v-2.625a3.375 3.375 0 00-3.375-3.375h-1.5A1.125 1.125 0 0113.5 7.125v-1.5a3.375 3.375 0 00-3.375-3.375H8.25m0 12.75h7.5m-7.5 3H12M10.5 2.25H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 00-9-9z" />
        </svg>
        <h3 class="text-[14px] font-semibold text-gray-900">{t.taskHistory || '历史记录'}</h3>
        {#if historyTasks.length > 0}
          <span class="px-2 py-0.5 text-[11px] font-medium text-gray-600 bg-gray-100 rounded-full">{historyTasks.length}</span>
        {/if}
      </div>
      <svg class="w-4 h-4 text-gray-400 transition-transform {showHistory ? 'rotate-180' : ''}" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
        <path stroke-linecap="round" stroke-linejoin="round" d="M19 9l-7 7-7-7" />
      </svg>
    </button>

    {#if showHistory}
      <div class="border-t border-gray-100">
        {#if historyTasks.length === 0}
          <div class="px-5 py-6 text-center text-[13px] text-gray-500">{t.noTaskHistory || '暂无历史记录'}</div>
        {:else}
          <div class="divide-y divide-gray-50">
            {#each historyTasks as task (task.id)}
              <div class="px-5 py-3 hover:bg-gray-50 transition-colors">
                <div class="flex items-center justify-between">
                  <div class="flex-1 min-w-0">
                    <div class="flex items-center gap-2 mb-1">
                      <span class="text-[13px] font-medium text-gray-900 truncate">{task.caseName}</span>
                      <span class="px-1.5 py-0.5 text-[10px] font-medium rounded {getActionBadge(task.action).cls}">
                        {getActionBadge(task.action).icon} {getActionLabel(task.action)}
                      </span>
                      <span class="px-1.5 py-0.5 text-[10px] font-medium rounded {getStatusBadge(task.status).cls}">{getStatusBadge(task.status).text}</span>
                    </div>
                    <div class="flex items-center gap-3 text-[11px] text-gray-500">
                      <span>{formatTime(task.scheduledAt)}</span>
                      {#if task.error}
                        <span class="text-red-500 truncate max-w-[300px]" title={task.error}>❌ {task.error}</span>
                      {/if}
                      {#if task.action === 'ssh_command' && task.sshCommand}
                        <span class="text-cyan-600 font-mono truncate max-w-[200px]" title={task.sshCommand}>$ {task.sshCommand}</span>
                      {/if}
                    </div>
                  </div>
                  {#if task.taskResult}
                    <button
                      class="px-2 py-1 text-[10px] font-medium text-blue-700 bg-blue-50 rounded-md hover:bg-blue-100 transition-colors cursor-pointer"
                      onclick={() => showResult(task)}
                    >{t.viewResult || '查看结果'}</button>
                  {/if}
                </div>
              </div>
            {/each}
          </div>
        {/if}
      </div>
    {/if}
  </div>
</div>

<!-- Create Task Modal -->
{#if showCreateForm}
  <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
  <div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4" onclick={(e) => { if (e.target === e.currentTarget) showCreateForm = false; }}>
    <div class="bg-white rounded-xl shadow-xl w-full max-w-lg overflow-hidden" onclick={(e) => e.stopPropagation()}>
      <div class="px-5 py-4 border-b border-gray-100 flex items-center justify-between">
        <h3 class="text-[15px] font-semibold text-gray-900">{t.taskCreate || '创建定时任务'}</h3>
        <button class="w-8 h-8 flex items-center justify-center rounded-lg hover:bg-gray-100 text-gray-400 cursor-pointer" onclick={() => showCreateForm = false}>
          <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" /></svg>
        </button>
      </div>

      <div class="px-5 py-4 space-y-4">
        {#if formError}
          <div class="flex items-center gap-2 px-3 py-2 bg-red-50 border border-red-100 rounded-lg">
            <svg class="w-4 h-4 text-red-500 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" /></svg>
            <span class="text-[12px] text-red-700">{formError}</span>
          </div>
        {/if}

        <!-- Case Select -->
        <div>
          <label class="block text-[12px] font-medium text-gray-700 mb-1.5">{t.taskSelectCase || '选择场景'}</label>
          <select class="w-full px-3 py-2 text-[13px] border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-gray-900" onchange={onCaseSelect}>
            <option value="">{t.taskSelectCasePlaceholder || '-- 请选择 --'}</option>
            {#each cases as c}
              <option value={c.id || c.Id}>{c.name || c.Name} ({c.id || c.Id})</option>
            {/each}
          </select>
        </div>

        <!-- Action -->
        <div>
          <label class="block text-[12px] font-medium text-gray-700 mb-1.5">{t.action || '操作'}</label>
          <div class="grid grid-cols-4 gap-2">
            <button class="px-3 py-2 text-[12px] font-medium rounded-lg transition-colors cursor-pointer {formAction === 'start' ? 'bg-emerald-500 text-white' : 'bg-gray-100 text-gray-700'}" onclick={() => formAction = 'start'}>▶ {t.start || '启动'}</button>
            <button class="px-3 py-2 text-[12px] font-medium rounded-lg transition-colors cursor-pointer {formAction === 'stop' ? 'bg-amber-500 text-white' : 'bg-gray-100 text-gray-700'}" onclick={() => formAction = 'stop'}>⏹ {t.stop || '停止'}</button>
            <button class="px-3 py-2 text-[12px] font-medium rounded-lg transition-colors cursor-pointer {formAction === 'ssh_command' ? 'bg-cyan-500 text-white' : 'bg-gray-100 text-gray-700'}" onclick={() => formAction = 'ssh_command'}>⌨ {t.sshCommand || 'SSH'}</button>
            <button class="px-3 py-2 text-[12px] font-medium rounded-lg transition-colors cursor-pointer {formAction === 'auto_stop' ? 'bg-orange-500 text-white' : 'bg-gray-100 text-gray-700'}" onclick={() => formAction = 'auto_stop'}>⏱ {t.autoStop || '自动停机'}</button>
          </div>
        </div>

        <!-- SSH Command input (only for ssh_command action) -->
        {#if formAction === 'ssh_command'}
          <div>
            <label class="block text-[12px] font-medium text-gray-700 mb-1.5">{t.sshCommandInput || 'SSH 命令'}</label>
            <textarea
              class="w-full px-3 py-2 text-[13px] font-mono border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-cyan-400 bg-gray-50 resize-none"
              rows="3"
              placeholder={t.sshCommandPlaceholder || '例: apt update && apt upgrade -y'}
              bind:value={formSSHCommand}
            ></textarea>
            <p class="text-[11px] text-gray-400 mt-1">{t.sshCommandHint || '命令将通过 SSH 在场景实例上执行'}</p>
          </div>
        {/if}

        <!-- Auto-stop duration (only for auto_stop action) -->
        {#if formAction === 'auto_stop'}
          <div>
            <label class="block text-[12px] font-medium text-gray-700 mb-1.5">{t.autoStopAfter || '运行时长后自动停止'}</label>
            <div class="flex items-center gap-2">
              <input type="number" min="0.5" max="168" step="0.5" class="w-24 px-3 py-2 text-[13px] border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-orange-400" bind:value={formAutoStopHours} />
              <span class="text-[12px] text-gray-600">{t.hoursLater || '小时后自动停止'}</span>
            </div>
            <p class="text-[11px] text-gray-400 mt-1">{t.autoStopHint || '从现在起计算，到时自动执行停止操作'}</p>
          </div>
        {/if}

        <!-- Time (hidden for auto_stop which has its own input) -->
        {#if formAction !== 'auto_stop'}
        <div>
          <label class="block text-[12px] font-medium text-gray-700 mb-1.5">{t.scheduleType || '时间设置'}</label>
          <div class="flex gap-2 mb-3">
            <button class="flex-1 px-4 py-2 text-[13px] font-medium rounded-lg transition-colors cursor-pointer {formScheduleType === 'relative' ? 'bg-blue-500 text-white' : 'bg-gray-100 text-gray-700'}" onclick={() => formScheduleType = 'relative'}>{t.relativeTime || '相对时间'}</button>
            <button class="flex-1 px-4 py-2 text-[13px] font-medium rounded-lg transition-colors cursor-pointer {formScheduleType === 'absolute' ? 'bg-blue-500 text-white' : 'bg-gray-100 text-gray-700'}" onclick={() => formScheduleType = 'absolute'}>{t.absoluteTime || '绝对时间'}</button>
          </div>
          {#if formScheduleType === 'relative'}
            <div class="grid grid-cols-2 gap-3">
              <div>
                <label class="block text-[11px] text-gray-500 mb-1">{t.hour || '小时'}</label>
                <input type="number" min="0" max="72" class="w-full px-3 py-2 text-[13px] border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-gray-900" bind:value={formRelativeHours} />
              </div>
              <div>
                <label class="block text-[11px] text-gray-500 mb-1">{t.minute || '分钟'}</label>
                <input type="number" min="0" max="59" class="w-full px-3 py-2 text-[13px] border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-gray-900" bind:value={formRelativeMinutes} />
              </div>
            </div>
          {:else}
            <div class="grid grid-cols-2 gap-3">
              <div>
                <label class="block text-[11px] text-gray-500 mb-1">{t.date || '日期'}</label>
                <input type="date" class="w-full px-3 py-2 text-[13px] border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-gray-900" bind:value={formAbsoluteDate} />
              </div>
              <div>
                <label class="block text-[11px] text-gray-500 mb-1">{t.time || '时间'}</label>
                <input type="time" class="w-full px-3 py-2 text-[13px] border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-gray-900" bind:value={formAbsoluteTime} />
              </div>
            </div>
          {/if}
        </div>

        <!-- Repeat -->
        <div>
          <label class="block text-[12px] font-medium text-gray-700 mb-1.5">{t.taskRepeatMode || '重复模式'}</label>
          <div class="flex gap-2 flex-wrap">
            {#each [['once', t.repeatOnce || '单次'], ['daily', t.repeatDaily || '每天'], ['weekly', t.repeatWeekly || '每周'], ['interval', t.repeatInterval || '自定义间隔']] as [val, label]}
              <button class="px-3 py-1.5 text-[12px] font-medium rounded-lg transition-colors cursor-pointer {formRepeatType === val ? 'bg-purple-500 text-white' : 'bg-gray-100 text-gray-700'}" onclick={() => formRepeatType = val}>{label}</button>
            {/each}
          </div>
          {#if formRepeatType === 'interval'}
            <div class="mt-2 flex items-center gap-2">
              <span class="text-[12px] text-gray-600">{t.repeatEvery || '每'}</span>
              <input type="number" min="1" max="10080" class="w-20 px-2 py-1.5 text-[13px] border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-gray-900" bind:value={formRepeatInterval} />
              <span class="text-[12px] text-gray-600">{t.minuteRepeat || '分钟执行一次'}</span>
            </div>
          {/if}
        </div>
        {/if}

        <!-- Notification toggle -->
        <div class="flex items-center justify-between">
          <div>
            <label class="text-[12px] font-medium text-gray-700">{t.notifyOnComplete || '完成时通知'}</label>
            <p class="text-[11px] text-gray-400">{t.notifyOnCompleteHint || '任务完成/失败时发送系统通知和 Webhook'}</p>
          </div>
          <button
            class="relative w-10 h-5 rounded-full transition-colors cursor-pointer {formNotifyEnabled ? 'bg-indigo-500' : 'bg-gray-300'}"
            onclick={() => formNotifyEnabled = !formNotifyEnabled}
          >
            <span class="absolute top-0.5 left-0.5 w-4 h-4 bg-white rounded-full shadow transition-transform {formNotifyEnabled ? 'translate-x-5' : ''}"></span>
          </button>
        </div>
      </div>

      <div class="px-5 py-4 bg-gray-50 flex justify-end gap-2">
        <button class="px-4 py-2 text-[13px] font-medium text-gray-700 bg-white border border-gray-200 rounded-lg hover:bg-gray-50 transition-colors cursor-pointer" onclick={() => showCreateForm = false}>{t.cancel || '取消'}</button>
        <button class="px-4 py-2 text-[13px] font-medium text-white bg-gray-900 rounded-lg hover:bg-gray-800 transition-colors disabled:opacity-50 flex items-center gap-2 cursor-pointer" onclick={handleCreate} disabled={formLoading}>
          {#if formLoading}
            <svg class="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path></svg>
          {/if}
          {t.confirm || '确认'}
        </button>
      </div>
    </div>
  </div>
{/if}

<!-- Cancel Confirmation Modal -->
{#if cancelConfirm.show}
  <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
  <div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onclick={() => cancelConfirm = { show: false, taskId: null, taskName: '' }}>
    <div class="bg-white rounded-xl border border-gray-200 max-w-sm w-full mx-4" onclick={(e) => e.stopPropagation()}>
      <div class="px-6 py-5">
        <div class="flex items-center gap-3 mb-3">
          <div class="w-10 h-10 rounded-full bg-amber-100 flex items-center justify-center">
            <svg class="w-5 h-5 text-amber-600" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" /></svg>
          </div>
          <div>
            <h3 class="text-[15px] font-semibold text-gray-900">{t.confirmCancelTask || '确认取消任务'}</h3>
          </div>
        </div>
        <p class="text-[13px] text-gray-600">
          {t.confirmCancelTaskMessage || '确定要取消定时任务'} <span class="font-medium text-gray-900">"{cancelConfirm.taskName}"</span>?
        </p>
      </div>
      <div class="px-6 py-4 bg-gray-50 flex justify-end gap-2">
        <button class="px-4 py-2 text-[13px] font-medium text-gray-700 bg-white border border-gray-100 rounded-lg hover:bg-gray-50 cursor-pointer" onclick={() => cancelConfirm = { show: false, taskId: null, taskName: '' }}>{t.cancel || '取消'}</button>
        <button class="px-4 py-2 text-[13px] font-medium text-white bg-amber-600 rounded-lg hover:bg-amber-700 cursor-pointer" onclick={confirmCancel}>{t.confirm || '确认'}</button>
      </div>
    </div>
  </div>
{/if}

<!-- Result Modal -->
{#if resultModal.show}
  <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
  <div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4" onclick={() => resultModal = { show: false, result: '', title: '' }}>
    <div class="bg-white rounded-xl shadow-xl w-full max-w-2xl overflow-hidden" onclick={(e) => e.stopPropagation()}>
      <div class="px-5 py-4 border-b border-gray-100 flex items-center justify-between">
        <h3 class="text-[15px] font-semibold text-gray-900">{resultModal.title}</h3>
        <button class="w-8 h-8 flex items-center justify-center rounded-lg hover:bg-gray-100 text-gray-400 cursor-pointer" onclick={() => resultModal = { show: false, result: '', title: '' }}>
          <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" /></svg>
        </button>
      </div>
      <div class="px-5 py-4 max-h-[60vh] overflow-auto">
        <pre class="text-[12px] font-mono text-gray-800 bg-gray-50 rounded-lg p-4 whitespace-pre-wrap break-all">{resultModal.result || '(empty)'}</pre>
      </div>
    </div>
  </div>
{/if}
