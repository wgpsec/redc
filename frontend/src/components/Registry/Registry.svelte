<script>
  import { onMount, onDestroy } from 'svelte';
  import { FetchRegistryTemplates, PullTemplate, ListTemplates } from '../../../wailsjs/go/main/App.js';
  import { normalizeVersion, compareVersions, hasUpdate } from '../../utils/version.js';

  export let t;

  // Registry state
  let registryTemplates = [];
  let registryLoading = false;
  let registryError = '';
  let registrySearch = '';
  let pullingTemplates = {};
  let registryNotice = { type: '', message: '' };
  let registryNoticeTimer = null;
  let templates = [];

  // Batch operation state
  let selectedTemplates = new Set();
  let batchOperating = false;
  let batchPullConfirm = { show: false, count: 0 };
  let batchUpdateConfirm = { show: false, count: 0 };

  $: allSelected = filteredRegistryTemplates.length > 0 && selectedTemplates.size === filteredRegistryTemplates.length;
  $: someSelected = selectedTemplates.size > 0 && selectedTemplates.size < filteredRegistryTemplates.length;
  $: hasSelection = selectedTemplates.size > 0;

  // Get templates that can be pulled (not installed)
  $: canPullTemplates = Array.from(selectedTemplates).filter(name => {
    const tmpl = registryTemplates.find(t => t.name === name);
    return tmpl && !tmpl.installed;
  });

  // Get templates that can be updated (installed and has update)
  $: canUpdateTemplates = Array.from(selectedTemplates).filter(name => {
    const tmpl = registryTemplates.find(t => t.name === name);
    return tmpl && tmpl.installed && hasUpdate(tmpl);
  });

  function setRegistryNotice(type, message, autoClear = true) {
    registryNotice = { type, message };
    if (registryNoticeTimer) {
      clearTimeout(registryNoticeTimer);
      registryNoticeTimer = null;
    }
    if (autoClear && message) {
      registryNoticeTimer = setTimeout(() => {
        registryNotice = { type: '', message: '' };
      }, 3000);
    }
  }

  async function loadRegistryTemplates() {
    registryLoading = true;
    registryError = '';
    try {
      registryTemplates = await FetchRegistryTemplates('');
    } catch (e) {
      registryError = e.message || String(e);
      registryTemplates = [];
    } finally {
      registryLoading = false;
    }
  }

  async function syncLocalTemplates() {
    try {
      const list = await ListTemplates();
      templates = list || [];
    } catch (e) {
      console.error('Failed to sync local templates:', e);
    }
  }

  async function handlePullTemplate(templateName, force = false) {
    pullingTemplates[templateName] = true;
    pullingTemplates = pullingTemplates;
    setRegistryNotice('info', `${t.pulling} ${templateName}`, false);
    try {
      await PullTemplate(templateName, force);
      // Refresh registry and local templates after successful pull
      await loadRegistryTemplates();
      await syncLocalTemplates();
      registryTemplates = (registryTemplates || []).map((tmpl) => {
        if (tmpl.name !== templateName) return tmpl;
        const latest = tmpl.latest || tmpl.localVersion;
        return {
          ...tmpl,
          installed: true,
          localVersion: latest || tmpl.localVersion
        };
      });
      setRegistryNotice('success', `${t.pullSuccess}: ${templateName}`);
    } catch (e) {
      setRegistryNotice('error', `${t.pullFailed}: ${templateName}`);
    } finally {
      pullingTemplates[templateName] = false;
      pullingTemplates = pullingTemplates;
    }
  }

  $: filteredRegistryTemplates = registryTemplates
    .filter(t => 
      !registrySearch || 
      t.name.toLowerCase().includes(registrySearch.toLowerCase()) ||
      (t.author && t.author.toLowerCase().includes(registrySearch.toLowerCase())) ||
      (t.description && t.description.toLowerCase().includes(registrySearch.toLowerCase())) ||
      (t.tags && t.tags.some(tag => tag.toLowerCase().includes(registrySearch.toLowerCase())))
    )
    .sort((a, b) => {
      // Installed templates first
      if (a.installed && !b.installed) return -1;
      if (!a.installed && b.installed) return 1;
      // Then sort by name alphabetically
      return a.name.localeCompare(b.name);
    });

  // Listen for refresh events to update pulling status
  $: if (registryTemplates.length > 0) {
    // Reset pulling status when templates are refreshed
    for (const t of registryTemplates) {
      if (t.installed && pullingTemplates[t.name]) {
        pullingTemplates[t.name] = false;
      }
    }
  }

  // ============================================================================
  // Batch Operation Functions
  // ============================================================================

  function toggleSelectAll() {
    if (allSelected) {
      selectedTemplates.clear();
    } else {
      filteredRegistryTemplates.forEach(t => selectedTemplates.add(t.name));
    }
    selectedTemplates = selectedTemplates;
  }

  function toggleSelectTemplate(templateName) {
    if (selectedTemplates.has(templateName)) {
      selectedTemplates.delete(templateName);
    } else {
      selectedTemplates.add(templateName);
    }
    selectedTemplates = selectedTemplates;
  }

  function showBatchPullConfirm() {
    batchPullConfirm = { show: true, count: canPullTemplates.length };
  }

  function cancelBatchPull() {
    batchPullConfirm = { show: false, count: 0 };
  }

  async function confirmBatchPull() {
    batchPullConfirm = { show: false, count: 0 };
    batchOperating = true;

    try {
      await Promise.all(canPullTemplates.map(name => handlePullTemplate(name, false)));
      selectedTemplates.clear();
      selectedTemplates = selectedTemplates;
    } catch (e) {
      setRegistryNotice('error', e.message || String(e));
    } finally {
      batchOperating = false;
      await loadRegistryTemplates();
    }
  }

  function showBatchUpdateConfirm() {
    batchUpdateConfirm = { show: true, count: canUpdateTemplates.length };
  }

  function cancelBatchUpdate() {
    batchUpdateConfirm = { show: false, count: 0 };
  }

  async function confirmBatchUpdate() {
    batchUpdateConfirm = { show: false, count: 0 };
    batchOperating = true;

    try {
      await Promise.all(canUpdateTemplates.map(name => handlePullTemplate(name, true)));
      selectedTemplates.clear();
      selectedTemplates = selectedTemplates;
    } catch (e) {
      setRegistryNotice('error', e.message || String(e));
    } finally {
      batchOperating = false;
      await loadRegistryTemplates();
    }
  }

  onMount(() => {
    loadRegistryTemplates();
  });

  onDestroy(() => {
    if (registryNoticeTimer) {
      clearTimeout(registryNoticeTimer);
      registryNoticeTimer = null;
    }
  });

  // Export refresh function for parent component
  export function refresh() {
    loadRegistryTemplates();
  }
</script>

<div class="space-y-5">
  <!-- Search -->
  <div class="bg-white rounded-xl border border-gray-100 p-5">
    <div class="flex items-center gap-4">
      <div class="flex-1 relative">
        <svg class="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
        </svg>
        <input 
          type="text" 
          placeholder={t.search}
          class="w-full h-10 pl-10 pr-4 text-[13px] bg-gray-50 border-0 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow"
          bind:value={registrySearch} 
        />
      </div>
      <button 
        class="h-10 px-5 bg-blue-600 text-white text-[13px] font-medium rounded-lg hover:bg-blue-700 transition-colors disabled:opacity-50"
        on:click={toggleSelectAll}
        disabled={registryLoading || filteredRegistryTemplates.length === 0}
      >
        {allSelected ? t.clearSelection : t.selectAll || '全选'}
      </button>
      <button 
        class="h-10 px-5 bg-gray-900 text-white text-[13px] font-medium rounded-lg hover:bg-gray-800 transition-colors disabled:opacity-50"
        on:click={loadRegistryTemplates}
        disabled={registryLoading}
      >
        {registryLoading ? t.loading : t.refreshRepo}
      </button>
    </div>
    {#if registryNotice.message}
      <div class="mt-3 flex items-center gap-2 rounded-lg border px-3 py-2 text-[12px]
        {registryNotice.type === 'success' ? 'bg-emerald-50 border-emerald-100 text-emerald-700' : registryNotice.type === 'error' ? 'bg-red-50 border-red-100 text-red-700' : 'bg-amber-50 border-amber-100 text-amber-700'}">
        {#if registryNotice.type === 'info'}
          <div class="w-3.5 h-3.5 border-2 border-amber-200 border-t-amber-600 rounded-full animate-spin"></div>
        {/if}
        <span class="flex-1 truncate">{registryNotice.message}</span>
        <button class="text-gray-400 hover:text-gray-600" on:click={() => setRegistryNotice('', '')}>
          <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>
    {/if}
  </div>

  {#if registryError}
    <div class="flex items-center gap-3 px-4 py-3 bg-red-50 border border-red-100 rounded-lg">
      <svg class="w-4 h-4 text-red-500 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
        <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z" />
      </svg>
      <span class="text-[13px] text-red-700 flex-1">{registryError}</span>
      <button class="text-red-400 hover:text-red-600" on:click={() => registryError = ''}>
        <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
        </svg>
      </button>
    </div>
  {/if}

  {#if registryLoading}
    <div class="flex items-center justify-center h-64">
      <div class="w-6 h-6 border-2 border-gray-200 border-t-gray-900 rounded-full animate-spin"></div>
    </div>
  {:else}
    <!-- Batch Operations Bar -->
    {#if hasSelection}
      <div class="bg-white rounded-xl border border-gray-100 p-5 mb-4">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-3">
            <span class="text-[13px] font-medium text-blue-900">
              {t.selected} {selectedTemplates.size} {t.items}
            </span>
            <button
              class="text-[12px] text-blue-600 hover:text-blue-800 underline"
              on:click={() => { selectedTemplates.clear(); selectedTemplates = selectedTemplates; }}
            >
              {t.clearSelection}
            </button>
          </div>
          <div class="flex items-center gap-2">
            {#if canPullTemplates.length > 0}
              <button
                class="px-3 py-1.5 text-[12px] font-medium text-white bg-gray-900 rounded-md hover:bg-gray-800 transition-colors disabled:opacity-50"
                on:click={showBatchPullConfirm}
                disabled={batchOperating}
              >
                {t.batchPull} ({canPullTemplates.length})
              </button>
            {/if}
            {#if canUpdateTemplates.length > 0}
              <button
                class="px-3 py-1.5 text-[12px] font-medium text-blue-700 bg-blue-50 rounded-md hover:bg-blue-100 transition-colors disabled:opacity-50"
                on:click={showBatchUpdateConfirm}
                disabled={batchOperating}
              >
                {t.batchUpdate} ({canUpdateTemplates.length})
              </button>
            {/if}
          </div>
        </div>
      </div>
    {/if}

    <!-- Template Grid -->
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      {#each filteredRegistryTemplates as tmpl}
        <div class="bg-white rounded-xl border border-gray-100 p-5 hover:shadow-md transition-shadow relative">
          <!-- Checkbox -->
          <div class="absolute top-4 left-4">
            <input
              type="checkbox"
              class="w-4 h-4 rounded border-gray-300 text-gray-900 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 cursor-pointer"
              checked={selectedTemplates.has(tmpl.name)}
              on:change={() => toggleSelectTemplate(tmpl.name)}
              on:click|stopPropagation
            />
          </div>
          
          <div class="pl-6">
            <div class="flex items-start justify-between mb-3">
              <div class="flex-1 min-w-0">
                <h3 class="text-[14px] font-semibold text-gray-900 truncate">{tmpl.name}</h3>
                <p class="text-[12px] text-gray-500 mt-0.5">v{tmpl.latest}</p>
              </div>
              {#if tmpl.installed}
                <span class="inline-flex items-center gap-1 px-2 py-0.5 bg-emerald-50 text-emerald-600 text-[11px] font-medium rounded-full">
                  <svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
                  </svg>
                  {t.installed}
                </span>
              {/if}
            </div>
            
            {#if tmpl.description}
              <p class="text-[12px] text-gray-600 mb-3 line-clamp-2">{tmpl.description}</p>
            {/if}
            
            {#if tmpl.tags && tmpl.tags.length > 0}
              <div class="flex flex-wrap gap-1 mb-3">
                {#each tmpl.tags.slice(0, 3) as tag}
                  <span class="px-2 py-0.5 bg-gray-100 text-gray-600 text-[10px] rounded-full">{tag}</span>
                {/each}
                {#if tmpl.tags.length > 3}
                  <span class="px-2 py-0.5 bg-gray-100 text-gray-400 text-[10px] rounded-full">+{tmpl.tags.length - 3}</span>
                {/if}
              </div>
            {/if}
            
            <div class="flex items-center justify-between pt-3 border-t border-gray-100">
              <div class="text-[11px] text-gray-400">
                {#if tmpl.author}by {tmpl.author}{/if}
              </div>
              {#if pullingTemplates[tmpl.name]}
                <span class="inline-flex items-center gap-2 px-3 py-1.5 text-[12px] font-medium text-amber-600">
                  <span class="w-3 h-3 border-2 border-amber-200 border-t-amber-600 rounded-full animate-spin"></span>
                  {t.pulling}
                </span>
              {:else if tmpl.installed && hasUpdate(tmpl)}
                <button 
                  class="px-3 py-1.5 text-[12px] font-medium text-blue-600 bg-blue-50 rounded-lg hover:bg-blue-100 transition-colors"
                  on:click={() => handlePullTemplate(tmpl.name, true)}
                >{t.update}</button>
              {:else if !tmpl.installed}
                <button 
                  class="px-3 py-1.5 text-[12px] font-medium text-white bg-gray-900 rounded-lg hover:bg-gray-800 transition-colors"
                  on:click={() => handlePullTemplate(tmpl.name, false)}
                >{t.pull}</button>
              {/if}
            </div>
          </div>
        </div>
      {:else}
        <div class="col-span-full py-16 text-center">
          <svg class="w-10 h-10 mx-auto mb-3 text-gray-300" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1">
            <path stroke-linecap="round" stroke-linejoin="round" d="M20.25 7.5l-.625 10.632a2.25 2.25 0 01-2.247 2.118H6.622a2.25 2.25 0 01-2.247-2.118L3.75 7.5M10 11.25h4M3.375 7.5h17.25c.621 0 1.125-.504 1.125-1.125v-1.5c0-.621-.504-1.125-1.125-1.125H3.375c-.621 0-1.125.504-1.125 1.125v1.5c0 .621.504 1.125 1.125 1.125z" />
          </svg>
          <p class="text-[13px] text-gray-400">
            {#if registrySearch}
              {t.noMatch}
            {:else}
              {t.clickRefresh}
            {/if}
          </p>
        </div>
      {/each}
    </div>
  {/if}
</div>

<!-- Batch Pull Confirmation Modal -->
{#if batchPullConfirm.show}
  <div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" on:click={cancelBatchPull}>
    <div class="bg-white rounded-xl shadow-xl max-w-sm w-full mx-4 overflow-hidden" on:click|stopPropagation>
      <div class="px-6 py-5">
        <div class="flex items-center gap-3 mb-3">
          <div class="w-10 h-10 rounded-full bg-gray-100 flex items-center justify-center">
            <svg class="w-5 h-5 text-gray-900" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
            </svg>
          </div>
          <div>
            <h3 class="text-[15px] font-semibold text-gray-900">{t.confirmBatchPull}</h3>
            <p class="text-[13px] text-gray-500">{t.pulling}</p>
          </div>
        </div>
        <p class="text-[13px] text-gray-600">
          {t.confirmBatchPullMessage} <span class="font-medium text-gray-900">{batchPullConfirm.count}</span> {t.templates}?
        </p>
      </div>
      <div class="px-6 py-4 bg-gray-50 flex justify-end gap-2">
        <button 
          class="px-4 py-2 text-[13px] font-medium text-gray-700 bg-white border border-gray-200 rounded-lg hover:bg-gray-50 transition-colors"
          on:click={cancelBatchPull}
        >{t.cancel}</button>
        <button 
          class="px-4 py-2 text-[13px] font-medium text-white bg-gray-900 rounded-lg hover:bg-gray-800 transition-colors"
          on:click={confirmBatchPull}
        >{t.pull}</button>
      </div>
    </div>
  </div>
{/if}

<!-- Batch Update Confirmation Modal -->
{#if batchUpdateConfirm.show}
  <div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" on:click={cancelBatchUpdate}>
    <div class="bg-white rounded-xl shadow-xl max-w-sm w-full mx-4 overflow-hidden" on:click|stopPropagation>
      <div class="px-6 py-5">
        <div class="flex items-center gap-3 mb-3">
          <div class="w-10 h-10 rounded-full bg-blue-100 flex items-center justify-center">
            <svg class="w-5 h-5 text-blue-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
            </svg>
          </div>
          <div>
            <h3 class="text-[15px] font-semibold text-gray-900">{t.confirmBatchUpdate}</h3>
            <p class="text-[13px] text-gray-500">{t.update}</p>
          </div>
        </div>
        <p class="text-[13px] text-gray-600">
          {t.confirmBatchUpdateMessage} <span class="font-medium text-gray-900">{batchUpdateConfirm.count}</span> {t.templates}?
        </p>
      </div>
      <div class="px-6 py-4 bg-gray-50 flex justify-end gap-2">
        <button 
          class="px-4 py-2 text-[13px] font-medium text-gray-700 bg-white border border-gray-200 rounded-lg hover:bg-gray-50 transition-colors"
          on:click={cancelBatchUpdate}
        >{t.cancel}</button>
        <button 
          class="px-4 py-2 text-[13px] font-medium text-white bg-blue-600 rounded-lg hover:bg-blue-700 transition-colors"
          on:click={confirmBatchUpdate}
        >{t.update}</button>
      </div>
    </div>
  </div>
{/if}
