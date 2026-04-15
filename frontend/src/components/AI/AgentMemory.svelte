<script>
  import { onMount } from 'svelte';
  import Modal from '../UI/Modal.svelte';
  import { GetAgentMemories, DeleteAgentMemory, ClearAgentMemories } from '../../../wailsjs/go/main/App.js';

  let { t } = $props();

  let memories = $state([]);
  let loading = $state(true);
  let showClearConfirm = $state(false);

  const categoryColors = {
    lesson: { bg: 'bg-amber-50', text: 'text-amber-700', border: 'border-amber-200', label: () => t.memoryLesson || '经验教训' },
    preference: { bg: 'bg-blue-50', text: 'text-blue-700', border: 'border-blue-200', label: () => t.memoryPreference || '使用偏好' },
    note: { bg: 'bg-gray-50', text: 'text-gray-600', border: 'border-gray-200', label: () => t.memoryNote || '备注' },
  };

  async function loadMemories() {
    loading = true;
    try {
      memories = await GetAgentMemories() || [];
    } catch (e) {
      console.error('Failed to load memories:', e);
      memories = [];
    } finally {
      loading = false;
    }
  }

  async function handleDelete(id) {
    try {
      await DeleteAgentMemory(id);
      memories = memories.filter(m => m.id !== id);
    } catch (e) {
      console.error('Failed to delete memory:', e);
    }
  }

  async function handleClear() {
    try {
      await ClearAgentMemories();
      memories = [];
      showClearConfirm = false;
    } catch (e) {
      console.error('Failed to clear memories:', e);
    }
  }

  function formatTime(t) {
    if (!t) return '';
    try {
      const d = new Date(t.replace(' ', 'T'));
      return d.toLocaleDateString() + ' ' + d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
    } catch { return t; }
  }

  let lessonCount = $derived(memories.filter(m => m.category === 'lesson').length);
  let preferenceCount = $derived(memories.filter(m => m.category === 'preference').length);

  onMount(() => { loadMemories(); });
</script>

<div class="h-full flex flex-col">
  <!-- Stats cards -->
  <div class="grid grid-cols-3 gap-4 mb-6">
    <div class="bg-white rounded-xl border border-gray-100 p-4">
      <div class="text-[11px] text-gray-500 mb-1">{t.memoryTotal || '总记忆'}</div>
      <div class="text-2xl font-semibold text-gray-900">{memories.length}</div>
    </div>
    <div class="bg-white rounded-xl border border-gray-100 p-4">
      <div class="text-[11px] text-gray-500 mb-1">{t.memoryLesson || '经验教训'}</div>
      <div class="text-2xl font-semibold text-amber-600">{lessonCount}</div>
    </div>
    <div class="bg-white rounded-xl border border-gray-100 p-4">
      <div class="text-[11px] text-gray-500 mb-1">{t.memoryPreference || '使用偏好'}</div>
      <div class="text-2xl font-semibold text-blue-600">{preferenceCount}</div>
    </div>
  </div>

  <!-- Header with actions -->
  <div class="flex items-center justify-between mb-4">
    <h3 class="text-[13px] font-medium text-gray-700">{t.memoryList || '记忆列表'}</h3>
    <div class="flex items-center gap-2">
      <button
        class="text-[11px] px-3 py-1.5 rounded-lg text-gray-600 hover:bg-gray-100 transition-colors cursor-pointer"
        onclick={loadMemories}
      >↻ {t.refresh || '刷新'}</button>
      {#if memories.length > 0}
        <button
          class="text-[11px] px-3 py-1.5 rounded-lg text-red-600 hover:bg-red-50 transition-colors cursor-pointer"
          onclick={() => showClearConfirm = true}
        >{t.clearAll || '清空全部'}</button>
      {/if}
    </div>
  </div>

  <!-- Memory list -->
  <div class="flex-1 overflow-y-auto space-y-3">
    {#if loading}
      <div class="flex items-center justify-center h-32">
        <div class="w-6 h-6 border-2 border-gray-100 border-t-gray-900 rounded-full animate-spin"></div>
      </div>
    {:else if memories.length === 0}
      <div class="flex flex-col items-center justify-center h-64 text-center">
        <div class="text-4xl mb-3"><svg class="w-10 h-10 text-gray-300 mx-auto" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M9.813 15.904L9 18.75l-.813-2.846a4.5 4.5 0 00-3.09-3.09L2.25 12l2.846-.813a4.5 4.5 0 003.09-3.09L9 5.25l.813 2.846a4.5 4.5 0 003.09 3.09L15.75 12l-2.846.813a4.5 4.5 0 00-3.09 3.09zM18.259 8.715L18 9.75l-.259-1.035a3.375 3.375 0 00-2.455-2.456L14.25 6l1.036-.259a3.375 3.375 0 002.455-2.456L18 2.25l.259 1.035a3.375 3.375 0 002.455 2.456L21.75 6l-1.036.259a3.375 3.375 0 00-2.455 2.456z" /></svg></div>
        <p class="text-[13px] text-gray-500 mb-1">{t.noMemories || '暂无记忆'}</p>
        <p class="text-[11px] text-gray-400">{t.noMemoriesHint || 'Agent 在对话结束后会自动提取经验教训'}</p>
      </div>
    {:else}
      {#each memories as memory (memory.id)}
        {@const cat = categoryColors[memory.category] || categoryColors.note}
        <div class="bg-white rounded-xl border border-gray-100 p-4 hover:shadow-sm transition-shadow group">
          <div class="flex items-start justify-between gap-3">
            <div class="flex-1 min-w-0">
              <div class="flex items-center gap-2 mb-2">
                <span class="inline-flex items-center px-2 py-0.5 rounded-md text-[10px] font-medium {cat.bg} {cat.text} {cat.border} border">
                  {cat.label()}
                </span>
                <span class="text-[10px] text-gray-400">{formatTime(memory.createdAt)}</span>
                {#if memory.source && memory.source !== 'auto'}
                  <span class="text-[10px] text-gray-300">·</span>
                  <span class="text-[10px] text-gray-400">{memory.source}</span>
                {/if}
              </div>
              <p class="text-[12px] text-gray-800 leading-relaxed">{memory.content}</p>
            </div>
            <button
              class="flex-shrink-0 p-1 rounded-md text-gray-300 hover:text-red-500 hover:bg-red-50 opacity-0 group-hover:opacity-100 transition-all cursor-pointer"
              onclick={() => handleDelete(memory.id)}
              title={t.delete || '删除'}
            >
              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
              </svg>
            </button>
          </div>
        </div>
      {/each}
    {/if}
  </div>
</div>

<!-- Clear confirm modal -->
<Modal show={showClearConfirm} onclose={() => showClearConfirm = false}>
    <div class="bg-white rounded-2xl p-6 max-w-sm mx-4 shadow-xl">
      <h3 class="text-[14px] font-semibold text-gray-900 mb-2">{t.clearConfirmTitle || '确认清空'}</h3>
      <p class="text-[12px] text-gray-600 mb-4">{t.clearConfirmMessage || '确定要清空所有 Agent 记忆吗？此操作不可撤销。'}</p>
      <div class="flex justify-end gap-2">
        <button
          class="px-4 py-2 text-[12px] rounded-lg text-gray-600 hover:bg-gray-100 transition-colors cursor-pointer"
          onclick={() => showClearConfirm = false}
        >{t.cancel || '取消'}</button>
        <button
          class="px-4 py-2 text-[12px] rounded-lg bg-red-500 text-white hover:bg-red-600 transition-colors cursor-pointer"
          onclick={handleClear}
        >{t.confirmClear || '确认清空'}</button>
      </div>
    </div>
</Modal>
