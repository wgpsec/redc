<script>
  import { onMount } from 'svelte';
  import { ListTemplates, GetTemplateVariables, RemoveTemplate, CopyTemplate, GetTemplateFiles, SaveTemplateFiles } from '../../../wailsjs/go/main/App.js';

  export let t;

  // Local templates state
  let localTemplates = [];
  let localTemplatesLoading = false;
  let localTemplatesSearch = '';
  let localTemplateDetail = null;
  let localTemplateVars = [];
  let localTemplateVarsLoading = false;
  let deleteTemplateConfirm = { show: false, name: '' };
  let deletingTemplate = {};
  let cloneTemplateModal = { show: false, source: '', target: '' };
  let templateEditor = { show: false, name: '', files: {}, active: '', saving: false, error: '' };
  let error = '';

  async function loadLocalTemplates() {
    localTemplatesLoading = true;
    try {
      localTemplates = await ListTemplates() || [];
    } catch (e) {
      error = e.message || String(e);
      localTemplates = [];
    } finally {
      localTemplatesLoading = false;
    }
  }

  async function showTemplateDetail(tmpl) {
    localTemplateDetail = tmpl;
    localTemplateVars = [];
    localTemplateVarsLoading = true;
    try {
      const vars = await GetTemplateVariables(tmpl.name);
      localTemplateVars = vars || [];
    } catch (e) {
      console.error('Failed to load template variables:', e);
      localTemplateVars = [];
    } finally {
      localTemplateVarsLoading = false;
    }
  }

  function closeTemplateDetail() {
    localTemplateDetail = null;
    localTemplateVars = [];
  }

  function showDeleteTemplateConfirm(name) {
    deleteTemplateConfirm = { show: true, name };
  }

  function cancelDeleteTemplate() {
    deleteTemplateConfirm = { show: false, name: '' };
  }

  async function confirmDeleteTemplate() {
    const name = deleteTemplateConfirm.name;
    deleteTemplateConfirm = { show: false, name: '' };
    deletingTemplate[name] = true;
    deletingTemplate = deletingTemplate;
    try {
      await RemoveTemplate(name);
      await loadLocalTemplates();
    } catch (e) {
      error = e.message || String(e);
    } finally {
      deletingTemplate[name] = false;
      deletingTemplate = deletingTemplate;
    }
  }

  async function handleCloneTemplate(tmpl) {
    cloneTemplateModal = { show: true, source: tmpl.name, target: `${tmpl.name}-copy` };
  }

  function cancelCloneTemplate() {
    cloneTemplateModal = { show: false, source: '', target: '' };
  }

  async function confirmCloneTemplate() {
    const targetName = cloneTemplateModal.target.trim();
    const sourceName = cloneTemplateModal.source;
    cloneTemplateModal = { show: false, source: '', target: '' };
    if (!targetName) return;
    try {
      await CopyTemplate(sourceName, targetName);
      await loadLocalTemplates();
    } catch (e) {
      error = e.message || String(e);
    }
  }

  async function openTemplateEditor(tmpl) {
    templateEditor = { show: true, name: tmpl.name, files: {}, active: '', saving: false, error: '' };
    try {
      const files = await GetTemplateFiles(tmpl.name);
      const names = Object.keys(files || {});
      templateEditor = {
        ...templateEditor,
        files: files || {},
        active: names.length > 0 ? names[0] : '',
      };
    } catch (e) {
      templateEditor = { ...templateEditor, error: e.message || String(e) };
    }
  }

  function closeTemplateEditor() {
    templateEditor = { show: false, name: '', files: {}, active: '', saving: false, error: '' };
  }

  async function saveTemplateEditor() {
    if (!templateEditor.name) return;
    templateEditor = { ...templateEditor, saving: true, error: '' };
    try {
      await SaveTemplateFiles(templateEditor.name, templateEditor.files);
      templateEditor = { ...templateEditor, saving: false };
    } catch (e) {
      templateEditor = { ...templateEditor, saving: false, error: e.message || String(e) };
    }
  }

  $: filteredLocalTemplates = localTemplates
    .filter(t => 
      !localTemplatesSearch || 
      t.name.toLowerCase().includes(localTemplatesSearch.toLowerCase()) ||
      (t.description && t.description.toLowerCase().includes(localTemplatesSearch.toLowerCase())) ||
      (t.module && t.module.toLowerCase().includes(localTemplatesSearch.toLowerCase()))
    )
    .sort((a, b) => a.name.localeCompare(b.name));

  onMount(() => {
    loadLocalTemplates();
  });

  // Export refresh function for parent component
  export function refresh() {
    loadLocalTemplates();
  }
</script>

<div class="space-y-5">
  <!-- Search and Actions -->
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
          bind:value={localTemplatesSearch} 
        />
      </div>
      <button 
        class="h-10 px-5 bg-gray-900 text-white text-[13px] font-medium rounded-lg hover:bg-gray-800 transition-colors disabled:opacity-50"
        on:click={loadLocalTemplates}
        disabled={localTemplatesLoading}
      >
        {localTemplatesLoading ? t.loading : t.refresh}
      </button>
    </div>
  </div>

  {#if localTemplatesLoading}
    <div class="flex items-center justify-center h-64">
      <div class="w-6 h-6 border-2 border-gray-200 border-t-gray-900 rounded-full animate-spin"></div>
    </div>
  {:else}
    <!-- Template Table -->
    <div class="bg-white rounded-xl border border-gray-100 overflow-hidden">
      <table class="w-full table-fixed">
        <thead>
          <tr class="border-b border-gray-100">
            <th class="text-left px-4 py-3 text-[11px] font-semibold text-gray-500 uppercase tracking-wide w-[140px]">{t.name}</th>
            <th class="text-left px-3 py-3 text-[11px] font-semibold text-gray-500 uppercase tracking-wide w-[60px]">{t.version}</th>
            <th class="text-left px-3 py-3 text-[11px] font-semibold text-gray-500 uppercase tracking-wide w-[140px]">{t.author}</th>
            <th class="text-left px-3 py-3 text-[11px] font-semibold text-gray-500 uppercase tracking-wide w-[180px]">{t.module}</th>
            <th class="text-left px-3 py-3 text-[11px] font-semibold text-gray-500 uppercase tracking-wide w-[320px]">{t.description}</th>
            <th class="text-right pl-4 pr-6 py-3 text-[11px] font-semibold text-gray-500 uppercase tracking-wide w-[220px]">{t.actions}</th>
          </tr>
        </thead>
        <tbody>
          {#each filteredLocalTemplates as tmpl}
            <tr class="border-b border-gray-50 hover:bg-gray-50/50 transition-colors">
              <td class="px-4 py-3.5">
                <span class="text-[13px] font-medium text-gray-900 break-all">{tmpl.name}</span>
              </td>
              <td class="px-3 py-3.5">
                <span class="text-[13px] text-gray-600">{tmpl.version || '-'}</span>
              </td>
              <td class="px-3 py-3.5">
                <span class="text-[13px] text-gray-600 break-words whitespace-normal block" title={tmpl.user || '-'}>{tmpl.user || '-'}</span>
              </td>
              <td class="px-3 py-3.5">
                {#if tmpl.module}
                  <span class="px-2 py-0.5 bg-blue-50 text-blue-600 text-[11px] font-medium rounded-full inline-block break-words whitespace-normal max-w-full" title={tmpl.module}>{tmpl.module}</span>
                {:else}
                  <span class="text-[13px] text-gray-400">-</span>
                {/if}
              </td>
              <td class="px-3 py-3.5 w-[320px]">
                <span class="text-[12px] text-gray-500 break-words whitespace-normal" title={tmpl.description}>{tmpl.description || '-'}</span>
              </td>
              <td class="pl-4 pr-6 py-3.5 text-right w-[240px]">
                <div class="flex flex-col gap-2 items-end">
                  <div class="flex items-center gap-2">
                    <button 
                      class="min-w-[100px] px-2.5 py-1 text-[12px] font-medium text-gray-700 bg-gray-100 rounded-md hover:bg-gray-200 transition-colors whitespace-nowrap"
                      on:click={() => handleCloneTemplate(tmpl)}
                    >{t.cloneTemplate}</button>
                    <button 
                      class="min-w-[100px] px-2.5 py-1 text-[12px] font-medium text-indigo-700 bg-indigo-50 rounded-md hover:bg-indigo-100 transition-colors whitespace-nowrap"
                      on:click={() => openTemplateEditor(tmpl)}
                    >{t.editTemplate}</button>
                  </div>
                  <div class="flex items-center gap-2">
                    <button 
                      class="min-w-[100px] px-2.5 py-1 text-[12px] font-medium text-blue-700 bg-blue-50 rounded-md hover:bg-blue-100 transition-colors whitespace-nowrap"
                      on:click={() => showTemplateDetail(tmpl)}
                    >{t.viewParams}</button>
                    {#if deletingTemplate[tmpl.name]}
                      <span class="min-w-[100px] px-2.5 py-1 text-[12px] font-medium text-amber-600 text-center">{t.deleting}</span>
                    {:else}
                      <button 
                        class="min-w-[100px] px-2.5 py-1 text-[12px] font-medium text-red-700 bg-red-50 rounded-md hover:bg-red-100 transition-colors whitespace-nowrap"
                        on:click={() => showDeleteTemplateConfirm(tmpl.name)}
                      >{t.delete}</button>
                    {/if}
                  </div>
                </div>
              </td>
            </tr>
          {:else}
            <tr>
              <td colspan="6" class="py-16">
                <div class="flex flex-col items-center text-gray-400">
                  <svg class="w-10 h-10 mb-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M2.25 12.75V12A2.25 2.25 0 014.5 9.75h15A2.25 2.25 0 0121.75 12v.75m-8.69-6.44l-2.12-2.12a1.5 1.5 0 00-1.061-.44H4.5A2.25 2.25 0 002.25 6v12a2.25 2.25 0 002.25 2.25h15A2.25 2.25 0 0021.75 18V9a2.25 2.25 0 00-2.25-2.25h-5.379a1.5 1.5 0 01-1.06-.44z" />
                  </svg>
                  <p class="text-[13px]">{t.noLocalTemplates}</p>
                  <button 
                    class="mt-2 text-[12px] text-blue-600 hover:underline"
                    on:click={() => { window.dispatchEvent(new CustomEvent('switchTab', { detail: 'registry' })); }}
                  >{t.goToRegistry}</button>
                </div>
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/if}

  {#if error}
    <div class="flex items-center gap-3 px-4 py-3 bg-red-50 border border-red-100 rounded-lg">
      <svg class="w-4 h-4 text-red-500 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
        <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z" />
      </svg>
      <span class="text-[13px] text-red-700 flex-1">{error}</span>
      <button class="text-red-400 hover:text-red-600" on:click={() => error = ''}>
        <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
        </svg>
      </button>
    </div>
  {/if}
</div>

<!-- Delete Template Confirmation Modal -->
{#if deleteTemplateConfirm.show}
  <div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" on:click={cancelDeleteTemplate}>
    <div class="bg-white rounded-xl shadow-xl max-w-sm w-full mx-4 overflow-hidden" on:click|stopPropagation>
      <div class="px-6 py-5">
        <div class="flex items-center gap-3 mb-3">
          <div class="w-10 h-10 rounded-full bg-red-100 flex items-center justify-center">
            <svg class="w-5 h-5 text-red-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
            </svg>
          </div>
          <div>
            <h3 class="text-[15px] font-semibold text-gray-900">{t.confirmDelete}</h3>
            <p class="text-[13px] text-gray-500">{t.deleteWarning}</p>
          </div>
        </div>
        <p class="text-[13px] text-gray-600">
          {t.confirmDeleteTemplate} <span class="font-medium text-gray-900">"{deleteTemplateConfirm.name}"</span>?
        </p>
      </div>
      <div class="px-6 py-4 bg-gray-50 flex justify-end gap-2">
        <button 
          class="px-4 py-2 text-[13px] font-medium text-gray-700 bg-white border border-gray-200 rounded-lg hover:bg-gray-50 transition-colors"
          on:click={cancelDeleteTemplate}
        >{t.cancel}</button>
        <button 
          class="px-4 py-2 text-[13px] font-medium text-white bg-red-600 rounded-lg hover:bg-red-700 transition-colors"
          on:click={confirmDeleteTemplate}
        >{t.delete}</button>
      </div>
    </div>
  </div>
{/if}

<!-- Clone Template Modal -->
{#if cloneTemplateModal.show}
  <div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" on:click={cancelCloneTemplate}>
    <div class="bg-white rounded-xl shadow-xl max-w-sm w-full mx-4 overflow-hidden" on:click|stopPropagation>
      <div class="px-6 py-5">
        <div class="flex items-center gap-3 mb-3">
          <div class="w-10 h-10 rounded-full bg-indigo-100 flex items-center justify-center">
            <svg class="w-5 h-5 text-indigo-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16h8M8 12h8m-6 8h6a2 2 0 002-2V8a2 2 0 00-2-2h-2l-2-2H8a2 2 0 00-2 2v12a2 2 0 002 2z" />
            </svg>
          </div>
          <div>
            <h3 class="text-[15px] font-semibold text-gray-900">{t.cloneTitle}</h3>
            <p class="text-[13px] text-gray-500">{t.cloneHint}</p>
          </div>
        </div>
        <label class="block text-[12px] font-medium text-gray-500 mb-1.5">{t.cloneName}</label>
        <input
          type="text"
          class="w-full h-10 px-3 text-[13px] bg-gray-50 border-0 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow"
          bind:value={cloneTemplateModal.target}
        />
      </div>
      <div class="px-6 py-4 bg-gray-50 flex justify-end gap-2">
        <button 
          class="px-4 py-2 text-[13px] font-medium text-gray-700 bg-white border border-gray-200 rounded-lg hover:bg-gray-50 transition-colors"
          on:click={cancelCloneTemplate}
        >{t.cancel}</button>
        <button 
          class="px-4 py-2 text-[13px] font-medium text-white bg-indigo-600 rounded-lg hover:bg-indigo-700 transition-colors"
          on:click={confirmCloneTemplate}
        >{t.cloneTemplate}</button>
      </div>
    </div>
  </div>
{/if}

<!-- Template Detail Drawer -->
{#if localTemplateDetail}
  <div class="fixed inset-0 bg-black/50 flex justify-end z-50" on:click={closeTemplateDetail}>
    <div class="w-full max-w-2xl bg-white h-full overflow-auto shadow-xl" on:click|stopPropagation>
      <div class="sticky top-0 bg-white border-b border-gray-100 px-6 py-4 flex items-center justify-between">
        <div>
          <h2 class="text-[16px] font-semibold text-gray-900">{localTemplateDetail.name}</h2>
          <p class="text-[12px] text-gray-500 mt-0.5">v{localTemplateDetail.version || '-'}</p>
        </div>
        <button 
          class="w-8 h-8 flex items-center justify-center rounded-lg hover:bg-gray-100 text-gray-400 hover:text-gray-600 transition-colors"
          on:click={closeTemplateDetail}
        >
          <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>
      
      <div class="p-6 space-y-6">
        <!-- Template Info -->
        <div class="space-y-3">
          {#if localTemplateDetail.description}
            <div>
              <div class="text-[11px] font-medium text-gray-500 uppercase tracking-wide mb-1">{t.description}</div>
              <p class="text-[13px] text-gray-700">{localTemplateDetail.description}</p>
            </div>
          {/if}
          <div class="grid grid-cols-2 gap-4">
            <div>
              <div class="text-[11px] font-medium text-gray-500 uppercase tracking-wide mb-1">{t.author}</div>
              <p class="text-[13px] text-gray-900">{localTemplateDetail.user || '-'}</p>
            </div>
            <div>
              <div class="text-[11px] font-medium text-gray-500 uppercase tracking-wide mb-1">{t.module}</div>
              {#if localTemplateDetail.module}
                <span class="px-2 py-0.5 bg-blue-50 text-blue-600 text-[12px] font-medium rounded-full">{localTemplateDetail.module}</span>
              {:else}
                <p class="text-[13px] text-gray-400">-</p>
              {/if}
            </div>
          </div>
        </div>

        <!-- Template Parameters -->
        <div>
          <div class="text-[14px] font-semibold text-gray-900 mb-3">{t.templateParams}</div>
          {#if localTemplateVarsLoading}
            <div class="flex items-center justify-center py-8">
              <div class="w-5 h-5 border-2 border-gray-200 border-t-gray-900 rounded-full animate-spin"></div>
              <span class="ml-2 text-[13px] text-gray-500">{t.loadingParams}</span>
            </div>
          {:else if localTemplateVars.length === 0}
            <div class="py-8 text-center text-[13px] text-gray-400">
              {t.noParams}
            </div>
          {:else}
            <div class="border border-gray-100 rounded-lg overflow-x-auto">
              <table class="w-full text-[12px] min-w-[520px]">
                <thead>
                  <tr class="bg-gray-50 border-b border-gray-100">
                    <th class="text-left px-4 py-2.5 font-semibold text-gray-600">{t.paramName}</th>
                    <th class="text-left px-4 py-2.5 font-semibold text-gray-600">{t.paramType}</th>
                    <th class="text-left px-4 py-2.5 font-semibold text-gray-600">{t.paramDefault}</th>
                    <th class="text-center px-4 py-2.5 font-semibold text-gray-600">{t.paramRequired}</th>
                  </tr>
                </thead>
                <tbody>
                  {#each localTemplateVars as v}
                    <tr class="border-b border-gray-50 hover:bg-gray-50/50">
                      <td class="px-4 py-3">
                        <div class="font-medium text-gray-900">{v.name}</div>
                        {#if v.description}
                          <div class="text-[11px] text-gray-500 mt-0.5">{v.description}</div>
                        {/if}
                      </td>
                      <td class="px-4 py-3">
                        <code class="px-1.5 py-0.5 bg-gray-100 text-gray-700 rounded text-[11px]">{v.type}</code>
                      </td>
                      <td class="px-4 py-3">
                        {#if v.defaultValue}
                          <code class="text-gray-600">{v.defaultValue}</code>
                        {:else}
                          <span class="text-gray-400">-</span>
                        {/if}
                      </td>
                      <td class="px-4 py-3 text-center">
                        {#if v.required}
                          <span class="inline-flex items-center justify-center w-5 h-5 bg-emerald-100 text-emerald-600 rounded-full">
                            <svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
                            </svg>
                          </span>
                        {:else}
                          <span class="inline-flex items-center justify-center w-5 h-5 bg-gray-100 text-gray-400 rounded-full">
                            <svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 12h14" />
                            </svg>
                          </span>
                        {/if}
                      </td>
                    </tr>
                  {/each}
                </tbody>
              </table>
            </div>
          {/if}
        </div>
      </div>
    </div>
  </div>
{/if}

<!-- Template Editor Modal -->
{#if templateEditor.show}
  <div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" on:click={closeTemplateEditor}>
    <div class="bg-white rounded-xl shadow-xl max-w-4xl w-full mx-4 overflow-hidden" on:click|stopPropagation>
      <div class="px-6 py-4 border-b border-gray-100 flex items-center justify-between">
        <div>
          <h3 class="text-[15px] font-semibold text-gray-900">{t.editTemplate}</h3>
          <p class="text-[12px] text-gray-500">{templateEditor.name}</p>
        </div>
        <div class="flex items-center gap-2">
          <button
            class="px-3 py-1.5 text-[12px] font-medium text-gray-600 bg-gray-100 rounded-md hover:bg-gray-200 transition-colors"
            on:click={closeTemplateEditor}
          >{t.close}</button>
          <button
            class="px-3 py-1.5 text-[12px] font-medium text-white bg-emerald-500 rounded-md hover:bg-emerald-600 transition-colors disabled:opacity-50"
            on:click={saveTemplateEditor}
            disabled={templateEditor.saving}
          >{templateEditor.saving ? t.saving : t.saveTemplate}</button>
        </div>
      </div>
      <div class="flex h-[520px]">
        <div class="w-52 border-r border-gray-100 overflow-auto">
          <div class="px-4 py-3 text-[12px] font-semibold text-gray-600">{t.templateFiles}</div>
          {#each Object.keys(templateEditor.files) as fname}
            <button
              class="w-full text-left px-4 py-2 text-[12px] transition-colors {templateEditor.active === fname ? 'bg-gray-900 text-white' : 'text-gray-600 hover:bg-gray-50'}"
              on:click={() => templateEditor = { ...templateEditor, active: fname }}
            >{fname}</button>
          {/each}
        </div>
        <div class="flex-1 p-4">
          {#if templateEditor.error}
            <div class="text-[12px] text-red-500 mb-2">{templateEditor.error}</div>
          {/if}
          {#if templateEditor.active}
            <textarea
              class="w-full h-full text-[12px] font-mono bg-gray-50 border border-gray-100 rounded-lg p-3 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow"
              bind:value={templateEditor.files[templateEditor.active]}
            ></textarea>
          {:else}
            <div class="text-[12px] text-gray-400">{t.noParams}</div>
          {/if}
        </div>
      </div>
    </div>
  </div>
{/if}
