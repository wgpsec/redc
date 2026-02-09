<script>

  import { onMount, onDestroy } from 'svelte';
  import { ListCases, ListTemplates, StartCase, StopCase, RemoveCase, CreateCase, CreateAndRunCase, GetCaseOutputs, GetTemplateVariables, GetCostEstimate } from '../../../wailsjs/go/main/App.js';

let { t, onTabChange = () => {} } = $props();
  let cases = $state([]);
  let templates = $state([]);
  let selectedTemplate = $state('');
  let newCaseName = $state('');
  let expandedCase = $state(null);
  let caseOutputs = $state({});
  let deleteConfirm = $state({ show: false, caseId: null, caseName: '' });
  let stopConfirm = $state({ show: false, caseId: null, caseName: '' });
  let templateVariables = $state([]);
  let variableValues = $state({});
  let error = $state('');
  
  // Cost estimation state
  let showCostEstimate = $state(false);
  let costEstimate = $state(null);
  let costEstimateLoading = $state(false);
  let costEstimateError = $state('');
  let costEstimateDebounceTimer = null;
  
  // Template list cost estimation state
  let templateCosts = $state({}); // Map of template name to cost estimate
  let templateCostsLoading = $state(new Set()); // Set of template names currently loading
  let allTemplateCostsLoading = $state(false); // Loading state for all templates
  
  // Batch operation state
  let selectedCases = $state(new Set());
  let batchOperating = $state(false);
  let batchDeleteConfirm = $state({ show: false, count: 0 });
  let batchStopConfirm = $state({ show: false, count: 0 });
  
  // Create status state
  let createStatus = $state('idle');
  let createStatusMessage = $state('');
  let createStatusDetail = $state('');
  let createStatusTimer = null;
  
  // Terraform init hint
  let terraformInitHint = $state({ show: false, message: '', detail: '' });
  let terraformInitHintDismissed = false;
  let terraformInitHintLastDetail = '';
  
  let copiedKey = $state(null);
  
  let createBusy = $derived(createStatus === 'creating' || createStatus === 'initializing');

  
  let allSelected = $derived(cases.length > 0 && selectedCases.size === cases.length);

  let someSelected = $derived(selectedCases.size > 0 && selectedCases.size < cases.length);

  let hasSelection = $derived(selectedCases.size > 0);

  
  let stateConfig = $derived({
    'running': { label: t.running, color: 'text-emerald-600', bg: 'bg-emerald-50', dot: 'bg-emerald-500' },
    'stopped': { label: t.stopped, color: 'text-slate-500', bg: 'bg-slate-50', dot: 'bg-slate-400' },
    'error': { label: t.error, color: 'text-red-600', bg: 'bg-red-50', dot: 'bg-red-500' },
    'created': { label: t.created, color: 'text-blue-600', bg: 'bg-blue-50', dot: 'bg-blue-500' },
    'pending': { label: t.pending, color: 'text-amber-600', bg: 'bg-amber-50', dot: 'bg-amber-500' },
    'starting': { label: t.starting, color: 'text-amber-600', bg: 'bg-amber-50', dot: 'bg-amber-500 animate-pulse' },
    'stopping': { label: t.stopping, color: 'text-amber-600', bg: 'bg-amber-50', dot: 'bg-amber-500 animate-pulse' },
    'removing': { label: t.removing, color: 'text-amber-600', bg: 'bg-amber-50', dot: 'bg-amber-500 animate-pulse' }
  });

  
  onMount(async () => {
    await refresh();
  });
  
  onDestroy(() => {
    if (createStatusTimer) {
      clearTimeout(createStatusTimer);
      createStatusTimer = null;
    }
    if (costEstimateDebounceTimer) {
      clearTimeout(costEstimateDebounceTimer);
      costEstimateDebounceTimer = null;
    }
  });
  
  function stripAnsi(value) {
    if (!value) return '';
    return value.replace(/\x1B\[[0-9;]*m/g, '');
  }
  
  function setCreateStatus(status, message, detail = '') {
    createStatus = status;
    createStatusMessage = message || '';
    createStatusDetail = detail || '';
    if (createStatusTimer) {
      clearTimeout(createStatusTimer);
      createStatusTimer = null;
    }
    if (status === 'success') {
      createStatusTimer = setTimeout(() => {
        createStatus = 'idle';
        createStatusMessage = '';
        createStatusDetail = '';
      }, 3000);
    }
  }
  
  function detectTerraformInitIssue(message) {
    const lower = message.toLowerCase();
    const hit = lower.includes('registry.terraform.io') || lower.includes('failed to query available provider packages') || lower.includes('x509') || lower.includes('tls') || lower.includes('context deadline') || lower.includes('client.timeout') || lower.includes('could not connect');
    if (hit) {
      if (terraformInitHintDismissed && terraformInitHintLastDetail === message) {
        return;
      }
      terraformInitHintDismissed = false;
      terraformInitHintLastDetail = message;
      terraformInitHint = { show: true, message: t.mirrorDetected, detail: message };
    }
  }
  
  function dismissTerraformInitHint() {
    terraformInitHint = { show: false, message: '', detail: '' };
    terraformInitHintDismissed = true;
  }
  
  export async function refresh() {
    try {
      [cases, templates] = await Promise.all([
        ListCases(),
        ListTemplates()
      ]);
      
      // Note: Template list cost preview is now manual (user must click button)
      // This prevents automatic loading of all template costs on page load
    } catch (e) {
      error = e.message || String(e);
      cases = [];
      templates = [];
    }
  }
  
  export function updateCreateStatusFromLog(message) {
    const cleanMessage = stripAnsi(message);
    if (cleanMessage.includes('正在创建场景:') || cleanMessage.includes('正在创建并运行场景:')) {
      setCreateStatus('creating', t.creating, message);
      return;
    }
    if (cleanMessage.includes('场景初始化中:')) {
      setCreateStatus('initializing', t.initializing, message);
      return;
    }
    if (cleanMessage.includes('场景创建成功')) {
      setCreateStatus('success', t.createSuccess, message);
      return;
    }
    if (cleanMessage.includes('场景创建失败') || cleanMessage.includes('创建场景时发生错误')) {
      setCreateStatus('error', t.createFailed, message);
      detectTerraformInitIssue(cleanMessage);
      return;
    }
  }
  
  async function loadTemplateVariables(templateName) {
    if (!templateName) {
      templateVariables = [];
      variableValues = {};
      return;
    }
    try {
      const vars = await GetTemplateVariables(templateName);
      templateVariables = vars || [];
      variableValues = {};
      for (const v of templateVariables) {
        variableValues[v.name] = v.defaultValue || '';
      }
    } catch (e) {
      console.error('Failed to load template variables:', e);
      templateVariables = [];
      variableValues = {};
    }
  }
  
  async function handleCreate() {
    if (!selectedTemplate) {
      error = t.selectTemplateErr;
      return;
    }
    setCreateStatus('creating', t.creating, '');
    try {
      const vars = {};
      for (const [key, value] of Object.entries(variableValues)) {
        if (value !== '') {
          vars[key] = value;
        }
      }
      await CreateCase(selectedTemplate, newCaseName, vars);
      selectedTemplate = '';
      newCaseName = '';
      templateVariables = [];
      variableValues = {};
    } catch (e) {
      error = e.message || String(e);
      setCreateStatus('error', t.createFailed, error);
    }
  }
  
  async function handleCreateAndRun() {
    if (!selectedTemplate) {
      error = t.selectTemplateErr;
      return;
    }
    setCreateStatus('creating', t.creating, '');
    try {
      const vars = {};
      for (const [key, value] of Object.entries(variableValues)) {
        if (value !== '') {
          vars[key] = value;
        }
      }
      await CreateAndRunCase(selectedTemplate, newCaseName, vars);
      selectedTemplate = '';
      newCaseName = '';
      templateVariables = [];
      variableValues = {};
    } catch (e) {
      error = e.message || String(e);
      setCreateStatus('error', t.createFailed, error);
    }
  }
  
  async function handleStart(caseId) {
    cases = cases.map(c => c.id === caseId ? { ...c, state: 'starting' } : c);
    try {
      await StartCase(caseId);
    } catch (e) {
      error = e.message || String(e);
      await refresh();
    }
  }
  
  async function handleStop(caseId) {
    cases = cases.map(c => c.id === caseId ? { ...c, state: 'stopping' } : c);
    try {
      await StopCase(caseId);
    } catch (e) {
      error = e.message || String(e);
      await refresh();
    }
  }
  
  function showStopConfirm(caseId, caseName) {
    stopConfirm = { show: true, caseId, caseName };
  }
  
  function cancelStop() {
    stopConfirm = { show: false, caseId: null, caseName: '' };
  }
  
  async function confirmStop() {
    const caseId = stopConfirm.caseId;
    stopConfirm = { show: false, caseId: null, caseName: '' };
    await handleStop(caseId);
  }
  
  function showDeleteConfirm(caseId, caseName) {
    deleteConfirm = { show: true, caseId, caseName };
  }
  
  function cancelDelete() {
    deleteConfirm = { show: false, caseId: null, caseName: '' };
  }
  
  async function confirmDelete() {
    const caseId = deleteConfirm.caseId;
    deleteConfirm = { show: false, caseId: null, caseName: '' };
    cases = cases.map(c => c.id === caseId ? { ...c, state: 'removing' } : c);
    try {
      await RemoveCase(caseId);
    } catch (e) {
      error = e.message || String(e);
      await refresh();
    }
  }
  
  function getShortId(id) {
    return id && id.length > 8 ? id.substring(0, 8) : id;
  }
  
  function getStateConfig(state) {
    return stateConfig[state] || stateConfig['pending'];
  }
  
  async function toggleCaseExpand(caseId, state) {
    if (expandedCase === caseId) {
      expandedCase = null;
      return;
    }
    expandedCase = caseId;
    if (state === 'running' && !caseOutputs[caseId]) {
      try {
        const outputs = await GetCaseOutputs(caseId);
        if (outputs) {
          caseOutputs[caseId] = outputs;
          caseOutputs = caseOutputs;
        }
      } catch (e) {
        console.error('Failed to load outputs:', e);
      }
    }
  }
  
  async function copyToClipboard(value, key) {
    try {
      await navigator.clipboard.writeText(value);
      copiedKey = key;
      setTimeout(() => { copiedKey = null; }, 2000);
    } catch (e) {
      console.error('Failed to copy:', e);
    }
  }

  // ============================================================================
  // Cost Estimation Functions
  // ============================================================================

  /**
   * Load base cost estimate for a template using default variable values
   * This is used for the template list preview
   * Failures are handled silently (no error messages shown to user)
   */
  async function loadTemplateCost(templateName) {
    if (!templateName || templateCostsLoading.has(templateName)) {
      return;
    }
    
    // Mark as loading
    templateCostsLoading.add(templateName);
    templateCostsLoading = templateCostsLoading;
    
    try {
      // Get template variables to extract defaults
      const vars = await GetTemplateVariables(templateName);
      
      // Build variables object with default values only
      const defaultVars = {};
      if (vars && vars.length > 0) {
        for (const v of vars) {
          if (v.defaultValue && v.defaultValue !== '') {
            defaultVars[v.name] = String(v.defaultValue);
          }
        }
      }
      
      // Call GetCostEstimate with default variables
      // Cast to Record<string, string> to satisfy TypeScript
      const estimate = await GetCostEstimate(templateName, defaultVars);
      
      // Store the estimate
      templateCosts[templateName] = estimate;
      templateCosts = templateCosts; // Trigger reactivity
    } catch (e) {
      // Silent failure - don't show error to user
      // Just don't add the cost to templateCosts
      console.debug(`Failed to load cost for template ${templateName}:`, e);
    } finally {
      // Remove from loading set
      templateCostsLoading.delete(templateName);
      templateCostsLoading = templateCostsLoading;
    }
  }

  /**
   * Load base cost estimates for all templates
   * Called manually by user clicking the "Load All Template Costs" button
   */
  async function loadAllTemplateCosts() {
    if (!templates || templates.length === 0) {
      return;
    }
    
    allTemplateCostsLoading = true;
    
    try {
      // Load costs for all templates in parallel
      // Each loadTemplateCost handles its own errors silently
      await Promise.all(templates.map(tmpl => loadTemplateCost(tmpl.name)));
    } finally {
      allTemplateCostsLoading = false;
    }
  }

  async function loadCostEstimate() {
    if (!selectedTemplate) return;
    
    // Set loading state and clear previous errors
    costEstimateLoading = true;
    costEstimateError = '';
    
    try {
      // Prepare variables object with non-empty values
      const vars = {};
      for (const [key, value] of Object.entries(variableValues)) {
        if (value !== '') {
          vars[key] = value;
        }
      }
      
      // Call GetCostEstimate API
      costEstimate = await GetCostEstimate(selectedTemplate, vars);
      
      // Show modal on success
      showCostEstimate = true;
    } catch (e) {
      // Set user-friendly error message
      costEstimateError = e.message || String(e);
    } finally {
      // Clear loading state
      costEstimateLoading = false;
    }
  }

  /**
   * Debounced cost estimation function
   * Waits 500ms after the last variable change before triggering cost estimation
   * Only triggers if the cost estimate modal is currently shown
   */
  function debouncedCostEstimate() {
    // Clear any existing timer
    if (costEstimateDebounceTimer) {
      clearTimeout(costEstimateDebounceTimer);
    }
    
    // Set new timer for 500ms delay
    costEstimateDebounceTimer = setTimeout(() => {
      // Only trigger if cost estimate modal is currently shown
      if (showCostEstimate) {
        loadCostEstimate();
      }
    }, 500);
  }

  // Watch for variable changes and trigger debounced cost estimation
  // This reactive statement runs whenever variableValues changes
  $effect(() => {
	if (selectedTemplate && Object.keys(variableValues).length > 0) {
    debouncedCostEstimate();
  }
});

  // ============================================================================
  // Batch Operation Functions
  // ============================================================================

  function toggleSelectAll() {
    if (allSelected) {
      selectedCases.clear();
    } else {
      cases.forEach(c => selectedCases.add(c.id));
    }
    selectedCases = selectedCases; // Trigger reactivity
  }

  function toggleSelectCase(caseId) {
    if (selectedCases.has(caseId)) {
      selectedCases.delete(caseId);
    } else {
      selectedCases.add(caseId);
    }
    selectedCases = selectedCases; // Trigger reactivity
  }

  function showBatchDeleteConfirm() {
    batchDeleteConfirm = { show: true, count: selectedCases.size };
  }

  function cancelBatchDelete() {
    batchDeleteConfirm = { show: false, count: 0 };
  }

  async function confirmBatchDelete() {
    batchDeleteConfirm = { show: false, count: 0 };
    batchOperating = true;
    
    const caseIds = Array.from(selectedCases);
    
    try {
      // Execute deletions in parallel
      await Promise.all(caseIds.map(caseId => RemoveCase(caseId)));
      selectedCases.clear();
      selectedCases = selectedCases;
    } catch (e) {
      error = e.message || String(e);
    } finally {
      batchOperating = false;
      await refresh();
    }
  }

  function showBatchStopConfirm() {
    batchStopConfirm = { show: true, count: selectedCases.size };
  }

  function cancelBatchStop() {
    batchStopConfirm = { show: false, count: 0 };
  }

  async function confirmBatchStop() {
    batchStopConfirm = { show: false, count: 0 };
    batchOperating = true;
    
    const caseIds = Array.from(selectedCases);
    
    try {
      // Execute stops in parallel
      await Promise.all(caseIds.map(caseId => StopCase(caseId)));
      selectedCases.clear();
      selectedCases = selectedCases;
    } catch (e) {
      error = e.message || String(e);
    } finally {
      batchOperating = false;
      await refresh();
    }
  }

  async function handleBatchStart() {
    batchOperating = true;
    
    const caseIds = Array.from(selectedCases);
    
    try {
      // Execute starts in parallel
      await Promise.all(caseIds.map(caseId => StartCase(caseId)));
      selectedCases.clear();
      selectedCases = selectedCases;
    } catch (e) {
      error = e.message || String(e);
    } finally {
      batchOperating = false;
      await refresh();
    }
  }


</script>

<div class="space-y-5">
  {#if error}
    <div class="flex items-center gap-3 px-4 py-3 bg-red-50 border border-red-100 rounded-lg">
      <svg class="w-4 h-4 text-red-500 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
        <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z" />
      </svg>
      <span class="text-[13px] text-red-700 flex-1">{error}</span>
      <button class="text-red-400 hover:text-red-600" onclick={() => error = ''}>
        <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
        </svg>
      </button>
    </div>
  {/if}

  <!-- Quick Create -->
  <div class="bg-white rounded-xl border border-gray-100 p-5">
    <div class="flex items-end gap-4 mb-4">
      <div class="flex-1">
        <label class="block text-[12px] font-medium text-gray-500 mb-1.5">{t.template}</label>
        <select 
          class="w-full h-10 px-3 text-[13px] bg-gray-50 border-0 rounded-lg text-gray-900 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow"
          bind:value={selectedTemplate}
          onchange={() => loadTemplateVariables(selectedTemplate)}
        >
          <option value="">{t.selectTemplate}</option>
          {#each templates || [] as tmpl}
            <option value={tmpl.name}>
              {tmpl.name}
              {#if templateCosts[tmpl.name]}
                · {templateCosts[tmpl.name].currency} {templateCosts[tmpl.name].total_monthly_cost.toFixed(2)}/mo
              {/if}
            </option>
          {/each}
        </select>
      </div>
      <div class="w-48">
        <label class="block text-[12px] font-medium text-gray-500 mb-1.5">{t.name}</label>
        <input 
          type="text" 
          placeholder={t.optional}
          class="w-full h-10 px-3 text-[13px] bg-gray-50 border-0 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow"
          bind:value={newCaseName} 
        />
      </div>
      <button 
        class="h-10 px-5 bg-gray-500 text-white text-[13px] font-medium rounded-lg hover:bg-gray-600 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
        onclick={handleCreate}
        disabled={createBusy}
      >
        {t.create}
      </button>
      <button 
        class="h-10 px-5 bg-emerald-500 text-white text-[13px] font-medium rounded-lg hover:bg-emerald-600 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
        onclick={handleCreateAndRun}
        disabled={createBusy}
      >
        {t.createAndRun}
      </button>
      {#if selectedTemplate}
        <button 
          class="h-10 px-5 bg-blue-500 text-white text-[13px] font-medium rounded-lg hover:bg-blue-600 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          onclick={loadCostEstimate}
          disabled={costEstimateLoading}
        >
          {#if costEstimateLoading}
            <span class="flex items-center gap-2">
              <div class="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin"></div>
              {t.calculating}
            </span>
          {:else}
            {t.costEstimate}
          {/if}
        </button>
      {/if}
      <button 
        class="h-10 px-5 bg-orange-500 text-white text-[13px] font-medium rounded-lg hover:bg-orange-600 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
        onclick={loadAllTemplateCosts}
        disabled={allTemplateCostsLoading || templates.length === 0}
      >
        {#if allTemplateCostsLoading}
          <span class="flex items-center gap-2">
            <div class="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin"></div>
            {t.loadingAllTemplateCosts}
          </span>
        {:else}
          {t.batchEstimate}
        {/if}
      </button>
    </div>

    <!-- Cost Estimation Error Display -->
    {#if costEstimateError}
      <div class="flex items-center gap-3 px-4 py-3 bg-amber-50 border border-amber-100 rounded-lg">
        <svg class="w-4 h-4 text-amber-500 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z" />
        </svg>
        <div class="flex-1">
          <div class="text-[13px] text-amber-800 font-medium">{t.costEstimateError}</div>
          <div class="text-[12px] text-amber-700 mt-0.5">{costEstimateError}</div>
          <div class="text-[11px] text-amber-600 mt-1">{t.costEstimateErrorHint}</div>
        </div>
        <button class="text-amber-400 hover:text-amber-600" onclick={() => costEstimateError = ''}>
          <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>
    {/if}

    {#if createStatus !== 'idle'}
      <div class="mt-3 flex items-center gap-2 rounded-lg border border-gray-100 bg-gray-50 px-3 py-2 text-[12px]">
        {#if createStatus === 'creating' || createStatus === 'initializing'}
          <div class="w-3.5 h-3.5 border-2 border-gray-200 border-t-gray-900 rounded-full animate-spin"></div>
          <span class="text-gray-700">{createStatusMessage}</span>
        {:else if createStatus === 'success'}
          <span class="text-emerald-600">{createStatusMessage}</span>
        {:else if createStatus === 'error'}
          <span class="text-red-600">{createStatusMessage}</span>
        {/if}
        {#if createStatusDetail}
          <span class="text-gray-400 truncate">{createStatusDetail}</span>
        {/if}
      </div>
    {:else if costEstimateError}
      <!-- Spacer to maintain layout when cost estimate error is shown but no create status -->
      <div class="mt-3"></div>
    {/if}

    {#if terraformInitHint.show}
      <div class="mt-3 rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-[12px] text-amber-700 relative">
        <button
          class="absolute right-2 top-2 text-amber-400 hover:text-amber-600"
          onclick={dismissTerraformInitHint}
          aria-label="close"
        >
          <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
        <div class="flex items-start gap-2">
          <svg class="w-4 h-4 mt-0.5 text-amber-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3m0 4h.01M10.29 3.86l-7.4 12.8A2 2 0 004.61 19h14.78a2 2 0 001.72-2.34l-7.4-12.8a2 2 0 00-3.42 0z" />
          </svg>
          <div class="flex-1">
            <div class="font-medium">{t.mirrorDetected}</div>
            <div class="text-amber-600 mt-1">{t.mirrorDetectedDesc}</div>
            {#if terraformInitHint.detail}
              <div class="text-amber-500 mt-1 truncate">{terraformInitHint.detail}</div>
            {/if}
            <div class="mt-2 text-amber-700">
              <div class="font-medium">{t.mirrorFixTitle}</div>
              <ul class="mt-1 list-disc list-inside text-amber-600 space-y-0.5">
                <li>{t.mirrorFixStep1}</li>
                <li>{t.mirrorFixStep2}</li>
                <li>{t.mirrorFixStep3}</li>
              </ul>
            </div>
            <div class="mt-2 flex flex-wrap gap-2">
              <button
                class="h-8 px-3 bg-white text-amber-700 text-[12px] font-medium rounded-md border border-amber-200 hover:bg-amber-100 transition-colors"
                onclick={() => onTabChange('settings')}
              >{t.mirrorGoSettings}</button>
            </div>
          </div>
        </div>
      </div>
    {:else if costEstimateError}
      <!-- Spacer to maintain layout when cost estimate error is shown but no terraform hint -->
      <div class="mt-3"></div>
    {/if}
    
    <!-- Template Variables -->
    {#if templateVariables.length > 0}
      <div class="border-t border-gray-100 pt-4 mt-4">
        <div class="text-[12px] font-medium text-gray-500 mb-3">{t.templateParams}</div>
        <div class="grid grid-cols-2 gap-3">
          {#each templateVariables as variable}
            <div class="flex flex-col">
              <label class="text-[11px] text-gray-500 mb-1">
                {variable.name}
                {#if variable.required}
                  <span class="text-red-500">*</span>
                {/if}
                {#if variable.description}
                  <span class="text-gray-400 ml-1">({variable.description})</span>
                {/if}
              </label>
              <input 
                type="text"
                class="h-9 px-3 text-[12px] bg-gray-50 border-0 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow"
                placeholder={variable.defaultValue || ''}
                bind:value={variableValues[variable.name]}
              />
            </div>
          {/each}
        </div>
      </div>
    {/if}
  </div>

  <!-- Table -->
  <div class="bg-white rounded-xl border border-gray-100 overflow-hidden">
    <!-- Batch Operations Bar -->
    {#if hasSelection}
      <div class="px-5 py-3 bg-blue-50 border-b border-blue-100 flex items-center justify-between">
        <div class="flex items-center gap-3">
          <span class="text-[13px] font-medium text-blue-900">
            {t.selected} {selectedCases.size} {t.items}
          </span>
          <button
            class="text-[12px] text-blue-600 hover:text-blue-800 underline"
            onclick={() => { selectedCases.clear(); selectedCases = selectedCases; }}
          >
            {t.clearSelection}
          </button>
        </div>
        <div class="flex items-center gap-2">
          <button
            class="px-3 py-1.5 text-[12px] font-medium text-emerald-700 bg-emerald-50 rounded-md hover:bg-emerald-100 transition-colors disabled:opacity-50"
            onclick={handleBatchStart}
            disabled={batchOperating}
          >
            {t.batchStart}
          </button>
          <button
            class="px-3 py-1.5 text-[12px] font-medium text-amber-700 bg-amber-50 rounded-md hover:bg-amber-100 transition-colors disabled:opacity-50"
            onclick={showBatchStopConfirm}
            disabled={batchOperating}
          >
            {t.batchStop}
          </button>
          <button
            class="px-3 py-1.5 text-[12px] font-medium text-red-700 bg-red-50 rounded-md hover:bg-red-100 transition-colors disabled:opacity-50"
            onclick={showBatchDeleteConfirm}
            disabled={batchOperating}
          >
            {t.batchDelete}
          </button>
        </div>
      </div>
    {/if}
    
    <table class="w-full">
      <thead>
        <tr class="border-b border-gray-100">
          <th class="text-left pl-4 pr-1 py-3 text-[11px] font-semibold text-gray-500 uppercase tracking-wide w-6">
            <input
              type="checkbox"
              class="w-4 h-4 rounded border-gray-300 text-gray-900 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 cursor-pointer"
              checked={allSelected}
              indeterminate={someSelected}
              onchange={toggleSelectAll}
            />
          </th>
          <th class="text-left px-5 py-3 text-[11px] font-semibold text-gray-500 uppercase tracking-wide">{t.id}</th>
          <th class="text-left px-5 py-3 text-[11px] font-semibold text-gray-500 uppercase tracking-wide">{t.name}</th>
          <th class="text-left px-5 py-3 text-[11px] font-semibold text-gray-500 uppercase tracking-wide">{t.type}</th>
          <th class="text-left px-5 py-3 text-[11px] font-semibold text-gray-500 uppercase tracking-wide">{t.state}</th>
          <th class="text-left px-5 py-3 text-[11px] font-semibold text-gray-500 uppercase tracking-wide">{t.time}</th>
          <th class="text-right px-5 py-3 text-[11px] font-semibold text-gray-500 uppercase tracking-wide">{t.actions}</th>
        </tr>
      </thead>
      <tbody>
        {#each cases || [] as c, i}
          <tr 
            class="border-b border-gray-50 hover:bg-gray-50/50 transition-colors cursor-pointer"
            onclick={() => toggleCaseExpand(c.id, c.state)}
          >
            <td class="pl-4 pr-1 py-3.5" onclick={(e) => e.stopPropagation()}>
              <input
                type="checkbox"
                class="w-4 h-4 rounded border-gray-300 text-gray-900 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 cursor-pointer"
                checked={selectedCases.has(c.id)}
                onchange={() => toggleSelectCase(c.id)}
              />
            </td>
            <td class="px-5 py-3.5">
              <div class="flex items-center gap-2">
                <svg class="w-4 h-4 text-gray-400 transition-transform {expandedCase === c.id ? 'rotate-90' : ''}" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
                </svg>
                <code class="text-[12px] text-gray-500 bg-gray-100 px-1.5 py-0.5 rounded">{getShortId(c.id)}</code>
              </div>
            </td>
            <td class="px-5 py-3.5">
              <span class="text-[13px] font-medium text-gray-900">{c.name}</span>
            </td>
            <td class="px-5 py-3.5">
              <span class="text-[13px] text-gray-600">{c.type}</span>
            </td>
            <td class="px-5 py-3.5">
              <span class="inline-flex items-center gap-1.5 text-[12px] font-medium {(stateConfig[c.state] || stateConfig['pending']).color}">
                <span class="w-1.5 h-1.5 rounded-full {(stateConfig[c.state] || stateConfig['pending']).dot}"></span>
                {(stateConfig[c.state] || stateConfig['pending']).label}
              </span>
            </td>
            <td class="px-5 py-3.5">
              <span class="text-[12px] text-gray-500">{c.createTime}</span>
            </td>
            <td class="px-5 py-3.5 text-right" onclick={(e) => e.stopPropagation()}>
              <div class="inline-flex items-center gap-1">
                {#if c.state === 'starting' || c.state === 'stopping' || c.state === 'removing'}
                  <span class="px-2.5 py-1 text-[12px] font-medium text-amber-600">
                    {stateConfig[c.state]?.label || t.processing}...
                  </span>
                {:else if c.state !== 'running'}
                  <button 
                    class="px-2.5 py-1 text-[12px] font-medium text-emerald-700 bg-emerald-50 rounded-md hover:bg-emerald-100 transition-colors"
                    onclick={() => handleStart(c.id)}
                  >{t.start}</button>
                {:else}
                  <button 
                    class="px-2.5 py-1 text-[12px] font-medium text-amber-700 bg-amber-50 rounded-md hover:bg-amber-100 transition-colors"
                    onclick={() => showStopConfirm(c.id, c.name)}
                  >{t.stop}</button>
                {/if}
                {#if c.state !== 'starting' && c.state !== 'stopping' && c.state !== 'removing'}
                  <button 
                    class="px-2.5 py-1 text-[12px] font-medium text-red-700 bg-red-50 rounded-md hover:bg-red-100 transition-colors"
                    onclick={() => showDeleteConfirm(c.id, c.name)}
                  >{t.delete}</button>
                {/if}
              </div>
            </td>
          </tr>
          <!-- Expanded row for outputs -->
          {#if expandedCase === c.id}
            <tr class="bg-slate-50">
              <td colspan="7" class="px-5 py-4">
                <div class="pl-6">
                  {#if c.state === 'running'}
                    {#if caseOutputs[c.id]}
                      <div class="grid grid-cols-2 gap-3">
                        {#each Object.entries(caseOutputs[c.id]) as [key, value]}
                          <div class="bg-white rounded-lg p-3 border border-gray-100 group relative">
                            <div class="flex items-center justify-between mb-1">
                              <div class="text-[11px] text-gray-500 uppercase tracking-wide">{key}</div>
                              <button 
                                class="opacity-0 group-hover:opacity-100 transition-opacity p-1 hover:bg-gray-100 rounded flex items-center gap-1"
                                onclick={(e) => { e.stopPropagation(); (() => copyToClipboard(value, key))(e); }}
                                title={t.copy}
                              >
                                {#if copiedKey === key}
                                  <svg class="w-4 h-4 text-emerald-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
                                  </svg>
                                  <span class="text-[10px] text-emerald-500">{t.copied}</span>
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
                      <div class="text-[13px] text-gray-500">{t.loadingOutputs}</div>
                    {/if}
                  {:else}
                    <div class="text-[13px] text-gray-500">{t.noOutput}</div>
                  {/if}
                </div>
              </td>
            </tr>
          {/if}
        {:else}
          <tr>
            <td colspan="7" class="py-16">
              <div class="flex flex-col items-center text-gray-400">
                <svg class="w-10 h-10 mb-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4" />
                </svg>
                <p class="text-[13px]">{t.noScene}</p>
              </div>
            </td>
          </tr>
        {/each}
      </tbody>
    </table>
  </div>
</div>

<!-- Delete Confirmation Modal -->
{#if deleteConfirm.show}
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
            <h3 class="text-[15px] font-semibold text-gray-900">{t.confirmDelete}</h3>
            <p class="text-[13px] text-gray-500">{t.cannotUndo}</p>
          </div>
        </div>
        <p class="text-[13px] text-gray-600">
          {t.confirmDeleteScene} <span class="font-medium text-gray-900">"{deleteConfirm.caseName}"</span>?
        </p>
      </div>
      <div class="px-6 py-4 bg-gray-50 flex justify-end gap-2">
        <button 
          class="px-4 py-2 text-[13px] font-medium text-gray-700 bg-white border border-gray-200 rounded-lg hover:bg-gray-50 transition-colors"
          onclick={cancelDelete}
        >{t.cancel}</button>
        <button 
          class="px-4 py-2 text-[13px] font-medium text-white bg-red-600 rounded-lg hover:bg-red-700 transition-colors"
          onclick={confirmDelete}
        >{t.delete}</button>
      </div>
    </div>
  </div>
{/if}

<!-- Batch Delete Confirmation Modal -->
{#if batchDeleteConfirm.show}
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
        <p class="text-[13px] text-gray-600">
          {t.confirmBatchDeleteMessage} <span class="font-medium text-gray-900">{batchDeleteConfirm.count}</span> {t.scenes}?
        </p>
      </div>
      <div class="px-6 py-4 bg-gray-50 flex justify-end gap-2">
        <button 
          class="px-4 py-2 text-[13px] font-medium text-gray-700 bg-white border border-gray-200 rounded-lg hover:bg-gray-50 transition-colors"
          onclick={cancelBatchDelete}
        >{t.cancel}</button>
        <button 
          class="px-4 py-2 text-[13px] font-medium text-white bg-red-600 rounded-lg hover:bg-red-700 transition-colors"
          onclick={confirmBatchDelete}
        >{t.delete}</button>
      </div>
    </div>
  </div>
{/if}

<!-- Batch Stop Confirmation Modal -->
{#if batchStopConfirm.show}
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
            <p class="text-[13px] text-gray-500">{t.stopWarning}</p>
          </div>
        </div>
        <p class="text-[13px] text-gray-600">
          {t.confirmBatchStopMessage} <span class="font-medium text-gray-900">{batchStopConfirm.count}</span> {t.scenes}?
        </p>
      </div>
      <div class="px-6 py-4 bg-gray-50 flex justify-end gap-2">
        <button 
          class="px-4 py-2 text-[13px] font-medium text-gray-700 bg-white border border-gray-200 rounded-lg hover:bg-gray-50 transition-colors"
          onclick={cancelBatchStop}
        >{t.cancel}</button>
        <button 
          class="px-4 py-2 text-[13px] font-medium text-white bg-amber-600 rounded-lg hover:bg-amber-700 transition-colors"
          onclick={confirmBatchStop}
        >{t.stop}</button>
      </div>
    </div>
  </div>
{/if}

<!-- Stop Confirmation Modal -->
{#if stopConfirm.show}
  <div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onclick={cancelStop}>
    <div class="bg-white rounded-xl shadow-xl max-w-sm w-full mx-4 overflow-hidden" onclick={(e) => e.stopPropagation()}>
      <div class="px-6 py-5">
        <div class="flex items-center gap-3 mb-3">
          <div class="w-10 h-10 rounded-full bg-amber-100 flex items-center justify-center">
            <svg class="w-5 h-5 text-amber-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
            </svg>
          </div>
          <div>
            <h3 class="text-[15px] font-semibold text-gray-900">{t.confirmStop}</h3>
            <p class="text-[13px] text-gray-500">{t.stopWarning}</p>
          </div>
        </div>
        <p class="text-[13px] text-gray-600">
          {t.confirmStopScene} <span class="font-medium text-gray-900">"{stopConfirm.caseName}"</span>?
        </p>
      </div>
      <div class="px-6 py-4 bg-gray-50 flex justify-end gap-2">
        <button 
          class="px-4 py-2 text-[13px] font-medium text-gray-700 bg-white border border-gray-200 rounded-lg hover:bg-gray-50 transition-colors"
          onclick={cancelStop}
        >{t.cancel}</button>
        <button 
          class="px-4 py-2 text-[13px] font-medium text-white bg-amber-600 rounded-lg hover:bg-amber-700 transition-colors"
          onclick={confirmStop}
        >{t.stop}</button>
      </div>
    </div>
  </div>
{/if}

<!-- Cost Estimate Modal -->
{#if showCostEstimate && costEstimate}
  <div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onclick={() => showCostEstimate = false}>
    <div class="bg-white rounded-xl shadow-xl max-w-2xl w-full mx-4 overflow-hidden" onclick={(e) => e.stopPropagation()}>
      <!-- Header -->
      <div class="px-6 py-5 border-b border-gray-100">
        <h3 class="text-[17px] font-semibold text-gray-900">{t.costEstimate}</h3>
        <p class="text-[13px] text-gray-500 mt-1">{costEstimate.disclaimer}</p>
      </div>
      
      <!-- Content -->
      <div class="px-6 py-5">
        <!-- Total Cost Summary -->
        <div class="grid grid-cols-2 gap-4 mb-6">
          <div class="bg-blue-50 rounded-lg p-4">
            <div class="text-[12px] text-blue-600 font-medium">{t.estimatedHourlyCost}</div>
            <div class="text-[24px] font-bold text-blue-900 mt-1">
              {costEstimate.currency} {costEstimate.total_hourly_cost.toFixed(4)}
            </div>
          </div>
          <div class="bg-emerald-50 rounded-lg p-4">
            <div class="text-[12px] text-emerald-600 font-medium">{t.estimatedMonthlyCost}</div>
            <div class="text-[24px] font-bold text-emerald-900 mt-1">
              {costEstimate.currency} {costEstimate.total_monthly_cost.toFixed(2)}
            </div>
          </div>
        </div>
        
        <!-- Cost Breakdown -->
        <div class="text-[13px] font-medium text-gray-700 mb-3">{t.costBreakdown}</div>
        <div class="space-y-2 max-h-64 overflow-y-auto">
          {#each costEstimate.breakdown as item}
            <div class="flex items-center justify-between p-3 bg-gray-50 rounded-lg">
              <div class="flex-1">
                <div class="text-[13px] font-medium text-gray-900">{item.resource_name}</div>
                <div class="text-[11px] text-gray-500">{item.resource_type} × {item.count}</div>
              </div>
              <div class="text-right">
                {#if item.available}
                  <div class="text-[13px] font-medium text-gray-900">
                    {item.currency} {item.total_monthly.toFixed(2)}/mo
                  </div>
                  <div class="text-[11px] text-gray-500">
                    {item.currency} {item.total_hourly.toFixed(4)}/hr
                  </div>
                {:else}
                  <div class="text-[12px] text-amber-600">{t.pricingUnavailable}</div>
                {/if}
              </div>
            </div>
          {/each}
        </div>
        
        <!-- Warnings -->
        {#if costEstimate.warnings && costEstimate.warnings.length > 0}
          <div class="mt-4 p-3 bg-amber-50 border border-amber-200 rounded-lg">
            <div class="text-[12px] font-medium text-amber-800 mb-1">{t.warnings}</div>
            <ul class="text-[11px] text-amber-700 space-y-1">
              {#each costEstimate.warnings as warning}
                <li>• {warning}</li>
              {/each}
            </ul>
          </div>
        {/if}
      </div>
      
      <!-- Footer -->
      <div class="px-6 py-4 bg-gray-50 flex justify-end gap-2">
        <button 
          class="px-4 py-2 text-[13px] font-medium text-gray-700 bg-white border border-gray-200 rounded-lg hover:bg-gray-50 transition-colors"
          onclick={() => showCostEstimate = false}
        >{t.close}</button>
      </div>
    </div>
  </div>
{/if}
