<script>

  import { ComposePreview, ComposeUp, ComposeDown, SelectComposeFile } from '../../../wailsjs/go/main/App.js';

  let { t } = $props(); // i18n translations

  // State
  let composeFilePath = $state('');
  let composeProfiles = $state('');
  let composeSummary = $state(null);
  let composeLoading = $state(false);
  let composeActionLoading = $state(false);
  let composeError = $state('');
  let hasManuallyPreviewed = $state(false);

  // Functions
  async function handleBrowseFile() {
    try {
      const selectedPath = await SelectComposeFile();
      if (selectedPath) {
        composeFilePath = selectedPath;
      }
    } catch (e) {
      console.error('Failed to select file:', e);
    }
  }

  function parseComposeProfiles(value) {
    if (!value) return [];
    return value
      .split(',')
      .map(v => v.trim())
      .filter(Boolean);
  }

  // Auto-preview when file path or profiles change (only after first manual preview)
  let timer = null;
  
  async function previewCompose() {
    if (!composeFilePath) {
      composeError = '';
      composeSummary = null;
      return;
    }
    
    hasManuallyPreviewed = true;
    composeLoading = true;
    composeError = '';
    try {
      composeSummary = await ComposePreview(composeFilePath, parseComposeProfiles(composeProfiles));
    } catch (e) {
      composeError = e.message || String(e);
      composeSummary = null;
    } finally {
      composeLoading = false;
    }
  }

  $effect(() => {
    if (hasManuallyPreviewed && composeFilePath) {
      if (timer) clearTimeout(timer);
      timer = setTimeout(() => {
        previewCompose();
      }, 500);
    }
  });

  export async function handleComposeUp() {
    if (!composeFilePath) {
      composeError = t.composeFile + ' ' + t.paramRequired;
      return;
    }
    
    composeActionLoading = true;
    composeError = '';
    try {
      await ComposeUp(composeFilePath, parseComposeProfiles(composeProfiles));
    } catch (e) {
      composeError = e.message || String(e);
    } finally {
      composeActionLoading = false;
    }
  }

  export async function handleComposeDown() {
    if (!composeFilePath) {
      composeError = t.composeFile + ' ' + t.paramRequired;
      return;
    }
    
    composeActionLoading = true;
    composeError = '';
    try {
      await ComposeDown(composeFilePath, parseComposeProfiles(composeProfiles));
    } catch (e) {
      composeError = e.message || String(e);
    } finally {
      composeActionLoading = false;
    }
  }

</script>

<div class="max-w-3xl lg:max-w-5xl xl:max-w-full space-y-5">
  <div class="bg-white rounded-xl border border-gray-100 p-5">
    <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
      <div>
        <label class="block text-[12px] font-medium text-gray-500 mb-1.5">{t.composeFile}</label>
        <div class="flex gap-2">
          <input
            type="text"
            placeholder="redc-compose.yaml"
            class="flex-1 h-10 px-3 text-[13px] bg-gray-50 border-0 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow font-mono"
            bind:value={composeFilePath}
          />
          <button
            class="h-10 px-4 bg-gray-100 text-gray-700 text-[12px] font-medium rounded-lg hover:bg-gray-200 transition-colors"
            onclick={handleBrowseFile}
          >
            {t.browseFile}
          </button>
        </div>
      </div>
      <div>
        <label class="block text-[12px] font-medium text-gray-500 mb-1.5">{t.composeProfiles}</label>
        <input
          type="text"
          placeholder="prod,dev"
          class="w-full h-10 px-3 text-[13px] bg-gray-50 border-0 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow"
          bind:value={composeProfiles}
        />
      </div>
    </div>
    <div class="mt-4 flex flex-wrap gap-2">
      <button
        class="h-9 px-4 bg-gray-900 text-white text-[12px] font-medium rounded-lg hover:bg-gray-800 transition-colors disabled:opacity-50"
        onclick={previewCompose}
        disabled={composeLoading}
      >
        {composeLoading ? t.loading : t.previewCompose}
      </button>
      <button
        class="h-9 px-4 bg-emerald-500 text-white text-[12px] font-medium rounded-lg hover:bg-emerald-600 transition-colors disabled:opacity-50"
        onclick={handleComposeUp}
        disabled={composeActionLoading}
      >
        {composeActionLoading ? t.processing : t.composeUp}
      </button>
      <button
        class="h-9 px-4 bg-red-500 text-white text-[12px] font-medium rounded-lg hover:bg-red-600 transition-colors disabled:opacity-50"
        onclick={handleComposeDown}
        disabled={composeActionLoading}
      >
        {composeActionLoading ? t.processing : t.composeDown}
      </button>
    </div>
    {#if composeError}
      <div class="mt-3 text-[12px] text-red-500">{composeError}</div>
    {/if}
  </div>

  <div class="bg-white rounded-xl border border-gray-100 p-5">
    <div class="text-[14px] font-semibold text-gray-900 mb-4">{t.composePreview}</div>
    {#if composeLoading}
      <div class="flex items-center justify-center h-24">
        <div class="w-6 h-6 border-2 border-gray-200 border-t-gray-900 rounded-full animate-spin"></div>
      </div>
    {:else if composeSummary && composeSummary.services && composeSummary.services.length > 0}
      <div class="border border-gray-100 rounded-lg overflow-hidden">
        <table class="w-full text-[12px]">
          <thead>
            <tr class="bg-gray-50 border-b border-gray-100">
              <th class="text-left px-4 py-2.5 font-semibold text-gray-600">{t.serviceName}</th>
              <th class="text-left px-4 py-2.5 font-semibold text-gray-600">{t.serviceTemplate}</th>
              <th class="text-left px-4 py-2.5 font-semibold text-gray-600">{t.serviceProvider}</th>
              <th class="text-left px-4 py-2.5 font-semibold text-gray-600">{t.serviceDepends}</th>
              <th class="text-right px-4 py-2.5 font-semibold text-gray-600">{t.serviceReplicas}</th>
            </tr>
          </thead>
          <tbody>
            {#each composeSummary.services as svc}
              <tr class="border-b border-gray-50">
                <td class="px-4 py-3 text-gray-700">{svc.name}</td>
                <td class="px-4 py-3 text-gray-700">{svc.template}</td>
                <td class="px-4 py-3 text-gray-700">{svc.provider || '-'}</td>
                <td class="px-4 py-3 text-gray-700">{(svc.dependsOn && svc.dependsOn.length > 0) ? svc.dependsOn.join(', ') : '-'}</td>
                <td class="px-4 py-3 text-right text-gray-700">{svc.replicas || 1}</td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {:else}
      <div class="py-12 text-center text-[12px] text-gray-400">{t.noScene}</div>
    {/if}
  </div>
</div>
