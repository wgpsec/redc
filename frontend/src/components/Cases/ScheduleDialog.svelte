<script>
  import { ScheduleTask, ListCaseScheduledTasks, CancelScheduledTask } from '../../../wailsjs/go/main/App.js';
  import Modal from '../UI/Modal.svelte';

  let { t, caseId, caseName, action, onClose, onSuccess } = $props();

  // action: "start" or "stop"
  let scheduleType = $state('relative'); // 'relative' or 'absolute'
  let relativeTime = $state({ hours: 1, minutes: 0 });
  let absoluteDate = $state('');
  let absoluteTime = $state('');
  let loading = $state(false);
  let error = $state('');
  let existingTasks = $state([]);
  let loadingTasks = $state(false);

  // 初始化默认值
  $effect(() => {
    const now = new Date();
    now.setHours(now.getHours() + 1);
    absoluteDate = now.toISOString().split('T')[0];
    absoluteTime = now.toTimeString().slice(0, 5);
    loadExistingTasks();
  });

  async function loadExistingTasks() {
    loadingTasks = true;
    try {
      const tasks = await ListCaseScheduledTasks(caseId);
      // 只显示待执行的任务
      existingTasks = tasks.filter(t => t.status === 'pending' && t.action === action);
    } catch (e) {
      console.error('加载任务失败:', e);
    } finally {
      loadingTasks = false;
    }
  }

  async function handleSchedule() {
    loading = true;
    error = '';

    try {
      let scheduledAt;

      if (scheduleType === 'relative') {
        // 相对时间
        const now = new Date();
        scheduledAt = new Date(now.getTime() + (relativeTime.hours * 60 + relativeTime.minutes) * 60 * 1000);
      } else {
        // 绝对时间
        scheduledAt = new Date(`${absoluteDate}T${absoluteTime}:00`);
      }

      // 验证时间
      if (scheduledAt <= new Date()) {
        error = t.scheduleTimeInvalid || '计划时间必须晚于当前时间';
        loading = false;
        return;
      }

      // 调用 API
      await ScheduleTask(caseId, caseName, action, scheduledAt);

      // 成功
      if (onSuccess) onSuccess();
      onClose();
    } catch (e) {
      error = e.message || String(e);
    } finally {
      loading = false;
    }
  }

  async function handleCancelTask(taskId) {
    try {
      await CancelScheduledTask(taskId);
      await loadExistingTasks();
    } catch (e) {
      error = e.message || String(e);
    }
  }

  function formatScheduledTime(timeStr) {
    const date = new Date(timeStr);
    return date.toLocaleString();
  }

  function getTimeRemaining(timeStr) {
    const scheduled = new Date(timeStr);
    const now = new Date();
    const diff = Number(scheduled) - Number(now);

    if (diff <= 0) return t.executing || '执行中';

    const hours = Math.floor(Number(diff) / (1000 * 60 * 60));
    const minutes = Math.floor(Number(diff) % (1000 * 60 * 60) / (1000 * 60));

    if (hours > 0) {
      return `${hours} ${t.hour || '小时'} ${minutes} ${t.minute || '分钟'}`;
    }
    return `${minutes} ${t.minute || '分钟'}`;
  }

  function handleKeydown(e) {
    if (e.key === 'Escape') {
      onClose();
    }
  }
</script>

<svelte:window on:keydown={handleKeydown} />

<Modal show={true} onclose={onClose} class="p-4">
  <div class="bg-white rounded-xl shadow-xl w-full max-w-lg overflow-hidden" onclick={(e) => e.stopPropagation()}>
    <!-- Header -->
    <div class="px-5 py-4 border-b border-gray-100">
      <div class="flex items-center justify-between">
        <div>
          <h3 class="text-[15px] font-semibold text-gray-900">
            {action === 'start' ? (t.scheduleStart || '定时启动') : (t.scheduleStop || '定时停止')}
          </h3>
          <p class="text-[12px] text-gray-500 mt-0.5">{caseName}</p>
        </div>
        <button
          class="w-8 h-8 flex items-center justify-center rounded-lg hover:bg-gray-100 text-gray-400 hover:text-gray-600 transition-colors"
          onclick={onClose}
          aria-label={t.close}
        >
          <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>
    </div>

    <!-- Content -->
    <div class="px-5 py-4 space-y-4">
      <!-- Error Display -->
      {#if error}
        <div class="flex items-center gap-2 px-3 py-2 bg-red-50 border border-red-100 rounded-lg">
          <svg class="w-4 h-4 text-red-500 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
          </svg>
          <span class="text-[12px] text-red-700 flex-1">{error}</span>
          <button class="text-red-400 hover:text-red-600 cursor-pointer" onclick={() => error = ''} aria-label={t.closeError}>
            <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>
      {/if}

      <!-- Schedule Type Selection -->
      <div>
        <span class="block text-[12px] font-medium text-gray-700 mb-2">{t.scheduleType || '时间设置'}</span>
        <div class="flex gap-2">
          <button
            class="flex-1 px-4 py-2 text-[13px] font-medium rounded-lg transition-colors"
            class:bg-blue-500={scheduleType === 'relative'}
            class:text-white={scheduleType === 'relative'}
            class:bg-gray-100={scheduleType !== 'relative'}
            class:text-gray-700={scheduleType !== 'relative'}
            onclick={() => scheduleType = 'relative'}
          >
            {t.relativeTime || '相对时间'}
          </button>
          <button
            class="flex-1 px-4 py-2 text-[13px] font-medium rounded-lg transition-colors"
            class:bg-blue-500={scheduleType === 'absolute'}
            class:text-white={scheduleType === 'absolute'}
            class:bg-gray-100={scheduleType !== 'absolute'}
            class:text-gray-700={scheduleType !== 'absolute'}
            onclick={() => scheduleType = 'absolute'}
          >
            {t.absoluteTime || '绝对时间'}
          </button>
        </div>
      </div>

      <!-- Relative Time Input -->
      {#if scheduleType === 'relative'}
        <div class="grid grid-cols-2 gap-3">
          <div>
            <label for="relativeHours" class="block text-[12px] font-medium text-gray-700 mb-1.5">{t.hour || '小时'}</label>
            <input
              id="relativeHours"
              type="number"
              min="0"
              max="72"
              class="w-full px-3 py-2 text-[13px] border border-gray-100 rounded-lg focus:outline-none focus:ring-2 focus:ring-gray-900"
              bind:value={relativeTime.hours}
            />
          </div>
          <div>
            <label for="relativeMinutes" class="block text-[12px] font-medium text-gray-700 mb-1.5">{t.minute || '分钟'}</label>
            <input
              id="relativeMinutes"
              type="number"
              min="0"
              max="59"
              class="w-full px-3 py-2 text-[13px] border border-gray-100 rounded-lg focus:outline-none focus:ring-2 focus:ring-gray-900"
              bind:value={relativeTime.minutes}
            />
          </div>
        </div>
        <p class="text-[11px] text-gray-500">
          {t.willExecuteIn || '将在'} {relativeTime.hours} {t.hour || '小时'} {relativeTime.minutes} {t.minutesLater || '分钟后执行'}
        </p>
      {/if}

      <!-- Absolute Time Input -->
      {#if scheduleType === 'absolute'}
        <div class="grid grid-cols-2 gap-3">
          <div>
            <label for="scheduleDate" class="block text-[12px] font-medium text-gray-700 mb-1.5">{t.date || '日期'}</label>
            <input
              id="scheduleDate"
              type="date"
              class="w-full px-3 py-2 text-[13px] border border-gray-100 rounded-lg focus:outline-none focus:ring-2 focus:ring-gray-900"
              bind:value={absoluteDate}
            />
          </div>
          <div>
            <label for="scheduleTime" class="block text-[12px] font-medium text-gray-700 mb-1.5">{t.time || '时间'}</label>
            <input
              id="scheduleTime"
              type="time"
              class="w-full px-3 py-2 text-[13px] border border-gray-100 rounded-lg focus:outline-none focus:ring-2 focus:ring-gray-900"
              bind:value={absoluteTime}
            />
          </div>
        </div>
      {/if}

      <!-- Existing Tasks -->
      {#if existingTasks.length > 0}
        <div class="border-t border-gray-100 pt-4">
          <div class="text-[12px] font-medium text-gray-700 mb-2">{t.existingTasks || '已有任务'}</div>
          <div class="space-y-2">
            {#each existingTasks as task}
              <div class="flex items-center justify-between p-2 bg-gray-50 rounded-lg">
                <div class="flex-1">
                  <div class="text-[12px] text-gray-900">{formatScheduledTime(task.scheduledAt)}</div>
                  <div class="text-[11px] text-gray-500">{getTimeRemaining(task.scheduledAt)}</div>
                </div>
                <button
                  class="px-2 py-1 text-[11px] font-medium text-red-600 hover:bg-red-50 rounded transition-colors"
                  onclick={() => handleCancelTask(task.id)}
                >
                  {t.cancel || '取消'}
                </button>
              </div>
            {/each}
          </div>
        </div>
      {/if}
    </div>

    <!-- Footer -->
    <div class="px-5 py-4 bg-gray-50 flex justify-end gap-2">
      <button
        class="px-4 py-2 text-[13px] font-medium text-gray-700 bg-white border border-gray-100 rounded-lg hover:bg-gray-50 transition-colors"
        onclick={onClose}
      >
        {t.cancel || '取消'}
      </button>
      <button
        class="px-4 py-2 text-[13px] font-medium text-white bg-blue-500 rounded-lg hover:bg-blue-600 transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
        onclick={handleSchedule}
        disabled={loading}
      >
        {#if loading}
          <svg class="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
          </svg>
        {/if}
        {t.confirm || '确认'}
      </button>
    </div>
  </div>
</Modal>
