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
      case 'start': return { cls: 'text-emerald-700 bg-emerald-50' };
      case 'stop': return { cls: 'text-amber-700 bg-amber-50' };
      case 'ssh_command': return { cls: 'text-gray-700 bg-gray-100' };
      case 'auto_stop': return { cls: 'text-gray-700 bg-gray-100' };
      default: return { cls: 'text-gray-600 bg-gray-100' };
    }
  }

  function getStatusBadge(status) {
    switch (status) {
      case 'pending': return { text: t.pending || '待执行', cls: 'text-gray-700 bg-gray-100' };
      case 'executing': return { text: t.executing || '执行中', cls: 'text-amber-700 bg-amber-50' };
      case 'completed': return { text: t.completed || '已完成', cls: 'text-emerald-700 bg-emerald-50' };
      case 'failed': return { text: t.failed || '失败', cls: 'text-red-700 bg-red-50' };
      case 'cancelled': return { text: t.cancelled || '已取消', cls: 'text-gray-600 bg-gray-100' };
      default: return { text: status, cls: 'text-gray-600 bg-gray-100' };
    }
  }
</script>

<div class="space-y-4">
  <!-- Error -->
  {#if error}
    <div class="flex items-center gap-2 px-4 py-2.5 bg-red-50 border border-red-100 rounded-xl">
      <svg class="w-3.5 h-3.5 text-red-500 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z" /></svg>
      <span class="text-[12px] text-red-700 flex-1">{error}</span>
      <button class="p-0.5 text-red-400 hover:text-red-600 cursor-pointer" onclick={() => error = ''}>
        <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" /></svg>
      </button>
    </div>
  {/if}

  <!-- Toolbar: stats + actions -->
  <div class="flex items-center gap-3 flex-wrap">
    <!-- Inline stats -->
    <div class="flex items-center gap-3 text-[11px] text-gray-400">
      <span class="flex items-center gap-1"><span class="w-1.5 h-1.5 rounded-full bg-gray-400"></span> {pendingTasks.length} {t.pending || '待执行'}</span>
      <span class="flex items-center gap-1"><span class="w-1.5 h-1.5 rounded-full bg-emerald-500"></span> {completedTasks.length} {t.completed || '已完成'}</span>
      {#if failedTasks.length > 0}
        <span class="flex items-center gap-1"><span class="w-1.5 h-1.5 rounded-full bg-red-500"></span> {failedTasks.length} {t.failed || '失败'}</span>
      {/if}
      {#if cancelledTasks.length > 0}
        <span class="flex items-center gap-1"><span class="w-1.5 h-1.5 rounded-full bg-gray-300"></span> {cancelledTasks.length} {t.cancelled || '已取消'}</span>
      {/if}
    </div>

    <div class="flex-1"></div>

    <!-- Refresh -->
    <button
      class="p-1.5 text-gray-400 hover:text-gray-600 hover:bg-gray-100 rounded-lg cursor-pointer transition-colors"
      onclick={loadTasks}
      title={t.refresh || '刷新'}
    >
      <svg class="w-4 h-4 {loading ? 'animate-spin' : ''}" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M16.023 9.348h4.992v-.001M2.985 19.644v-4.992m0 0h4.992m-4.993 0l3.181 3.183a8.25 8.25 0 0013.803-3.7M4.031 9.865a8.25 8.25 0 0113.803-3.7l3.181 3.182m0-4.991v4.99" /></svg>
    </button>

    <!-- Create -->
    <button
      class="px-3 py-1.5 text-[12px] font-medium text-white bg-gray-900 rounded-lg hover:bg-gray-800 transition-colors flex items-center gap-1.5 cursor-pointer"
      onclick={openCreateForm}
    >
      <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M12 4.5v15m7.5-7.5h-15" /></svg>
      {t.taskCreate || '创建任务'}
    </button>
  </div>

  <!-- Pending Tasks -->
  <div class="bg-white rounded-xl border border-gray-100 overflow-hidden">
    <div class="px-4 py-3 border-b border-gray-100 flex items-center justify-between">
      <div class="flex items-center gap-2">
        <h3 class="text-[13px] font-semibold text-gray-900">{t.pendingTasks || '待执行任务'}</h3>
        {#if pendingTasks.length > 0}
          <span class="px-2 py-0.5 text-[11px] font-medium text-gray-600 bg-gray-100 rounded-full">{pendingTasks.length}</span>
        {/if}
      </div>
    </div>

    {#if loading && tasks.length === 0}
      <div class="px-4 py-8 flex items-center justify-center gap-2">
        <svg class="w-4 h-4 animate-spin text-gray-400" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path></svg>
        <span class="text-[12px] text-gray-400">{t.loading || '加载中...'}</span>
      </div>
    {:else if pendingTasks.length === 0}
      <div class="px-4 py-8 text-center">
        <svg class="w-8 h-8 mx-auto text-gray-200 mb-2" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
          <path stroke-linecap="round" stroke-linejoin="round" d="M12 6v6h4.5m4.5 0a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
        <p class="text-[12px] text-gray-400">{t.noScheduledTasks || '暂无待执行的定时任务'}</p>
      </div>
    {:else}
      <div class="divide-y divide-gray-50">
        {#each pendingTasks as task (task.id)}
          <div class="px-4 py-3 hover:bg-gray-50/50 transition-colors">
            <div class="flex items-center justify-between">
              <div class="flex-1 min-w-0">
                <div class="flex items-center gap-2 mb-1">
                  <span class="text-[12px] font-medium text-gray-900 truncate">{task.caseName}</span>
                  <span class="px-1.5 py-0.5 text-[10px] font-medium rounded {getActionBadge(task.action).cls}">
                    {getActionLabel(task.action)}
                  </span>
                  {#if task.repeatType && task.repeatType !== 'once'}
                    <span class="px-1.5 py-0.5 text-[10px] font-medium text-gray-600 bg-gray-100 rounded flex items-center gap-0.5">
                      <svg class="w-2.5 h-2.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M16.023 9.348h4.992v-.001M2.985 19.644v-4.992m0 0h4.992m-4.993 0l3.181 3.183a8.25 8.25 0 0013.803-3.7M4.031 9.865a8.25 8.25 0 0113.803-3.7l3.181 3.182m0-4.991v4.99" /></svg>
                      {getRepeatLabel(task)}
                    </span>
                  {/if}
                  {#if task.notifyEnabled}
                    <span class="p-0.5 text-gray-400" title={t.notifyOnComplete || '完成通知'}>
                      <svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M14.857 17.082a23.848 23.848 0 005.454-1.31A8.967 8.967 0 0118 9.75v-.7V9A6 6 0 006 9v.75a8.967 8.967 0 01-2.312 6.022c1.733.64 3.56 1.085 5.455 1.31m5.714 0a24.255 24.255 0 01-5.714 0m5.714 0a3 3 0 11-5.714 0" /></svg>
                    </span>
                  {/if}
                  {#if task.status === 'executing'}
                    <span class="px-1.5 py-0.5 text-[10px] font-medium text-amber-700 bg-amber-50 rounded flex items-center gap-1">
                      <svg class="w-3 h-3 animate-spin" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/></svg>
                      {t.executing || '执行中'}
                    </span>
                  {/if}
                </div>
                <div class="flex items-center gap-3 text-[11px] text-gray-500">
                  <span class="flex items-center gap-1">
                    <svg class="w-3 h-3 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M6.75 3v2.25M17.25 3v2.25M3 18.75V7.5a2.25 2.25 0 012.25-2.25h13.5A2.25 2.25 0 0121 7.5v11.25m-18 0A2.25 2.25 0 005.25 21h13.5A2.25 2.25 0 0021 18.75m-18 0v-7.5A2.25 2.25 0 015.25 9h13.5A2.25 2.25 0 0121 11.25v7.5" /></svg>
                    {formatTime(task.scheduledAt)}
                  </span>
                  {#if task.status === 'pending'}
                    <span class="flex items-center gap-1">
                      <svg class="w-3 h-3 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M12 6v6h4.5m4.5 0a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
                      {t.remaining || '剩余'}: {getTimeRemaining(task.scheduledAt)}
                    </span>
                  {/if}
                  {#if task.action === 'ssh_command' && task.sshCommand}
                    <span class="text-gray-600 font-mono truncate max-w-[250px]" title={task.sshCommand}>$ {task.sshCommand}</span>
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
      class="w-full px-4 py-3 flex items-center justify-between hover:bg-gray-50/50 transition-colors cursor-pointer"
      onclick={() => showHistory = !showHistory}
    >
      <div class="flex items-center gap-2">
        <h3 class="text-[13px] font-semibold text-gray-900">{t.taskHistory || '历史记录'}</h3>
        {#if historyTasks.length > 0}
          <span class="px-2 py-0.5 text-[11px] font-medium text-gray-600 bg-gray-100 rounded-full">{historyTasks.length}</span>
        {/if}
      </div>
      <svg class="w-3.5 h-3.5 text-gray-400 transition-transform {showHistory ? 'rotate-180' : ''}" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
        <path stroke-linecap="round" stroke-linejoin="round" d="M19 9l-7 7-7-7" />
      </svg>
    </button>

    {#if showHistory}
      <div class="border-t border-gray-100">
        {#if historyTasks.length === 0}
          <div class="px-4 py-8 text-center">
            <svg class="w-8 h-8 mx-auto text-gray-200 mb-2" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
              <path stroke-linecap="round" stroke-linejoin="round" d="M19.5 14.25v-2.625a3.375 3.375 0 00-3.375-3.375h-1.5A1.125 1.125 0 0113.5 7.125v-1.5a3.375 3.375 0 00-3.375-3.375H8.25m0 12.75h7.5m-7.5 3H12M10.5 2.25H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 00-9-9z" />
            </svg>
            <p class="text-[12px] text-gray-400">{t.noTaskHistory || '暂无历史记录'}</p>
          </div>
        {:else}
          <div class="divide-y divide-gray-50">
            {#each historyTasks as task (task.id)}
              <div class="px-4 py-3 hover:bg-gray-50/50 transition-colors">
                <div class="flex items-center justify-between">
                  <div class="flex-1 min-w-0">
                    <div class="flex items-center gap-2 mb-1">
                      <span class="text-[12px] font-medium text-gray-900 truncate">{task.caseName}</span>
                      <span class="px-1.5 py-0.5 text-[10px] font-medium rounded {getActionBadge(task.action).cls}">
                        {getActionLabel(task.action)}
                      </span>
                      <span class="px-1.5 py-0.5 text-[10px] font-medium rounded {getStatusBadge(task.status).cls}">{getStatusBadge(task.status).text}</span>
                    </div>
                    <div class="flex items-center gap-3 text-[11px] text-gray-500">
                      <span class="flex items-center gap-1">
                        <svg class="w-3 h-3 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M6.75 3v2.25M17.25 3v2.25M3 18.75V7.5a2.25 2.25 0 012.25-2.25h13.5A2.25 2.25 0 0121 7.5v11.25m-18 0A2.25 2.25 0 005.25 21h13.5A2.25 2.25 0 0021 18.75m-18 0v-7.5A2.25 2.25 0 015.25 9h13.5A2.25 2.25 0 0121 11.25v7.5" /></svg>
                        {formatTime(task.scheduledAt)}
                      </span>
                      {#if task.error}
                        <span class="flex items-center gap-1 text-red-500 truncate max-w-[300px]" title={task.error}>
                          <svg class="w-3 h-3 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M9.75 9.75l4.5 4.5m0-4.5l-4.5 4.5M21 12a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
                          {task.error}
                        </span>
                      {/if}
                      {#if task.action === 'ssh_command' && task.sshCommand}
                        <span class="text-gray-600 font-mono truncate max-w-[200px]" title={task.sshCommand}>$ {task.sshCommand}</span>
                      {/if}
                    </div>
                  </div>
                  {#if task.taskResult}
                    <button
                      class="px-2 py-1 text-[10px] font-medium text-gray-700 bg-gray-100 rounded-md hover:bg-gray-200 transition-colors cursor-pointer"
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
            <button class="px-3 py-2 text-[12px] font-medium rounded-lg transition-colors cursor-pointer flex items-center justify-center gap-1.5 {formAction === 'start' ? 'bg-gray-900 text-white' : 'bg-gray-100 text-gray-700 hover:bg-gray-200'}" onclick={() => formAction = 'start'}>
              <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M5.25 5.653c0-.856.917-1.398 1.667-.986l11.54 6.348a1.125 1.125 0 010 1.971l-11.54 6.347a1.125 1.125 0 01-1.667-.985V5.653z" /></svg>
              {t.start || '启动'}
            </button>
            <button class="px-3 py-2 text-[12px] font-medium rounded-lg transition-colors cursor-pointer flex items-center justify-center gap-1.5 {formAction === 'stop' ? 'bg-gray-900 text-white' : 'bg-gray-100 text-gray-700 hover:bg-gray-200'}" onclick={() => formAction = 'stop'}>
              <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M5.25 7.5A2.25 2.25 0 017.5 5.25h9a2.25 2.25 0 012.25 2.25v9a2.25 2.25 0 01-2.25 2.25h-9a2.25 2.25 0 01-2.25-2.25v-9z" /></svg>
              {t.stop || '停止'}
            </button>
            <button class="px-3 py-2 text-[12px] font-medium rounded-lg transition-colors cursor-pointer flex items-center justify-center gap-1.5 {formAction === 'ssh_command' ? 'bg-gray-900 text-white' : 'bg-gray-100 text-gray-700 hover:bg-gray-200'}" onclick={() => formAction = 'ssh_command'}>
              <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="m6.75 7.5 3 2.25-3 2.25m4.5 0h3m-9 8.25h13.5A2.25 2.25 0 0 0 21 18V6a2.25 2.25 0 0 0-2.25-2.25H5.25A2.25 2.25 0 0 0 3 6v12a2.25 2.25 0 0 0 2.25 2.25Z" /></svg>
              {t.sshCommand || 'SSH'}
            </button>
            <button class="px-3 py-2 text-[12px] font-medium rounded-lg transition-colors cursor-pointer flex items-center justify-center gap-1.5 {formAction === 'auto_stop' ? 'bg-gray-900 text-white' : 'bg-gray-100 text-gray-700 hover:bg-gray-200'}" onclick={() => formAction = 'auto_stop'}>
              <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M12 6v6h4.5m4.5 0a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
              {t.autoStop || '自动停机'}
            </button>
          </div>
        </div>

        <!-- SSH Command input (only for ssh_command action) -->
        {#if formAction === 'ssh_command'}
          <div>
            <label class="block text-[12px] font-medium text-gray-700 mb-1.5">{t.sshCommandInput || 'SSH 命令'}</label>
            <textarea
              class="w-full px-3 py-2 text-[13px] font-mono border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-gray-900 bg-gray-50 resize-none"
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
              <input type="number" min="0.5" max="168" step="0.5" class="w-24 px-3 py-2 text-[13px] border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-gray-900" bind:value={formAutoStopHours} />
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
            <button class="flex-1 px-4 py-2 text-[13px] font-medium rounded-lg transition-colors cursor-pointer {formScheduleType === 'relative' ? 'bg-gray-900 text-white' : 'bg-gray-100 text-gray-700 hover:bg-gray-200'}" onclick={() => formScheduleType = 'relative'}>{t.relativeTime || '相对时间'}</button>
            <button class="flex-1 px-4 py-2 text-[13px] font-medium rounded-lg transition-colors cursor-pointer {formScheduleType === 'absolute' ? 'bg-gray-900 text-white' : 'bg-gray-100 text-gray-700 hover:bg-gray-200'}" onclick={() => formScheduleType = 'absolute'}>{t.absoluteTime || '绝对时间'}</button>
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
              <button class="px-3 py-1.5 text-[12px] font-medium rounded-lg transition-colors cursor-pointer {formRepeatType === val ? 'bg-gray-900 text-white' : 'bg-gray-100 text-gray-700 hover:bg-gray-200'}" onclick={() => formRepeatType = val}>{label}</button>
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
            class="relative w-10 h-5 rounded-full transition-colors cursor-pointer {formNotifyEnabled ? 'bg-emerald-500' : 'bg-gray-300'}"
            onclick={() => formNotifyEnabled = !formNotifyEnabled}
          >
            <span class="absolute top-0.5 left-0.5 w-4 h-4 bg-white rounded-full shadow transition-transform {formNotifyEnabled ? 'translate-x-5' : ''}"></span>
          </button>
        </div>
      </div>

      <div class="px-5 py-4 bg-gray-50 flex justify-end gap-2">
        <button class="px-4 py-2 text-[13px] font-medium text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors cursor-pointer" onclick={() => showCreateForm = false}>{t.cancel || '取消'}</button>
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
        <button class="px-4 py-2 text-[13px] font-medium text-white bg-red-600 rounded-lg hover:bg-red-700 cursor-pointer" onclick={confirmCancel}>{t.confirm || '确认'}</button>
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
