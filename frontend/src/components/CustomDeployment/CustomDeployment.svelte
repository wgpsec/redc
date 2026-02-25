<script lang="ts">
  import { onMount } from 'svelte';
  import { GetBaseTemplates, GetTemplateMetadata, EstimateDeploymentCost, CreateCustomDeployment } from '../../../wailsjs/go/main/App.js';
  import TemplateSelector from './TemplateSelector.svelte';
  import ConfigEditor from './ConfigEditor.svelte';
  import DeploymentPreview from './DeploymentPreview.svelte';
  import ConfigTemplateManager from './ConfigTemplateManager.svelte';
  import SaveTemplateDialog from './SaveTemplateDialog.svelte';
  import ImportExportDialog from './ImportExportDialog.svelte';
  import DeploymentManagement from './DeploymentManagement.svelte';

  let { t } = $props();
  
  // Tab state
  let activeTab = $state('create'); // 'create' or 'manage'
  
  // State management
  let baseTemplates = $state([]);
  let selectedTemplate = $state(null);
  let templateMetadata = $state(null);
  let config = $state({
    name: '',
    provider: '',
    region: '',
    instanceType: '',
    userdata: '',
    isSpotInstance: false,
    variables: {}
  });
  let validation = $state({
    valid: true,
    errors: [],
    warnings: []
  });
  let preview = $state(null);
  let costEstimate = $state(null);
  let isLoading = $state(false);
  let isDeploying = $state(false);
  let deploymentResult = $state(null);
  let deploymentError = $state('');
  let isEstimatingCost = $state(false);
  let error = $state('');
  let costEstimateTimeout = null;
  
  // Template management state
  let showSaveDialog = $state(false);
  let showImportDialog = $state(false);
  let showExportDialog = $state(false);
  let exportTemplateName = $state('');
  let showTemplateManager = $state(false);

  onMount(async () => {
    await loadBaseTemplates();
  });

  async function loadBaseTemplates() {
    isLoading = true;
    error = '';
    try {
      baseTemplates = await GetBaseTemplates();
    } catch (e) {
      error = e.message || String(e);
      baseTemplates = [];
    } finally {
      isLoading = false;
    }
  }

  async function handleTemplateSelect(template) {
    selectedTemplate = template;
    error = '';
    
    if (!template) {
      templateMetadata = null;
      resetConfig();
      return;
    }

    try {
      templateMetadata = await GetTemplateMetadata(template.name);
      resetConfig();
      // 自动设置 provider（从模板元数据中读取）
      if (templateMetadata && templateMetadata.provider) {
        config.provider = templateMetadata.provider;
      }
    } catch (e) {
      error = e.message || String(e);
      templateMetadata = null;
    }
  }

  function resetConfig() {
    config = {
      name: '',
      provider: '',
      region: '',
      instanceType: '',
      userdata: '',
      isSpotInstance: false,
      variables: {}
    };
    validation = {
      valid: true,
      errors: [],
      warnings: []
    };
    preview = null;
  }

  function handleConfigUpdate(newConfig) {
    config = newConfig;
    // Schedule cost estimation when config changes
    scheduleCostEstimation();
  }

  function handleValidationUpdate(newValidation) {
    validation = newValidation;
  }

  // Schedule cost estimation with debounce
  function scheduleCostEstimation() {
    if (costEstimateTimeout) {
      clearTimeout(costEstimateTimeout);
    }
    
    costEstimateTimeout = setTimeout(() => {
      estimateCost();
    }, 1000); // Debounce for 1 second
  }

  // Estimate deployment cost
  async function estimateCost() {
    // Only estimate if we have the required fields
    if (!config.name || !config.provider || !config.region || !config.instanceType) {
      costEstimate = null;
      return;
    }

    // Only estimate if validation passed
    if (!validation.valid) {
      costEstimate = null;
      return;
    }

    isEstimatingCost = true;
    
    try {
      const configToEstimate: any = {
        name: config.name,
        template_name: templateMetadata?.name || '',
        provider: config.provider,
        region: config.region,
        instance_type: config.instanceType,
        userdata: config.userdata,
        variables: config.variables,
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString()
      };

      const estimate = await EstimateDeploymentCost(configToEstimate);
      
      if (estimate) {
        costEstimate = estimate;
      }
    } catch (error: any) {
      console.error('Cost estimation error:', error);
      // Don't show error to user, just clear the estimate
      costEstimate = null;
    } finally {
      isEstimatingCost = false;
    }
  }

  async function handleDeploy() {
    isDeploying = true;
    deploymentError = '';
    deploymentResult = null;
    
    try {
      // Prepare deployment config
      const deploymentConfig: any = {
        name: config.name || `deployment-${Date.now()}`,
        template_name: templateMetadata?.name || '',
        provider: config.provider,
        region: config.region,
        instance_type: config.instanceType,
        userdata: config.userdata,
        is_spot_instance: config.isSpotInstance,
        variables: config.variables,
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString()
      };

      console.log('Creating deployment with config:', deploymentConfig);
      
      // Call backend API to create deployment
      const result = await CreateCustomDeployment(deploymentConfig);
      
      console.log('Deployment created:', result);
      deploymentResult = result;
      
      // Show success message
      alert(`${t.deploymentCreated || '部署创建成功'}!\n\n${t.deploymentId || '部署 ID'}: ${result.id}\n\n${t.checkDeploymentManagement || '您可以在"部署管理"页面查看部署详情和日志。'}`);
      
      // Optionally navigate to deployment management page
      // TODO: Add navigation when deployment management page is ready
      
    } catch (err: any) {
      console.error('Deployment failed:', err);
      deploymentError = err.message || String(err);
      
      // Show error message
      alert(`部署失败：${deploymentError}\n\n请检查配置并重试。`);
    } finally {
      isDeploying = false;
    }
  }

  // Template management functions
  function handleSaveTemplate() {
    showSaveDialog = true;
  }

  function handleTemplateSaved(templateName: string) {
    console.log('Template saved:', templateName);
    // Optionally show a success message
  }

  function handleLoadTemplate(loadedConfig: any, templateName: string) {
    // Load the configuration into the current state
    config = {
      name: loadedConfig.name || '',
      provider: loadedConfig.provider || '',
      region: loadedConfig.region || '',
      instanceType: loadedConfig.instance_type || '',
      userdata: loadedConfig.userdata || '',
      isSpotInstance: loadedConfig.is_spot_instance || false,
      variables: loadedConfig.variables || {}
    };
    
    // If we have a template name in the config, try to load its metadata
    if (loadedConfig.template_name) {
      GetTemplateMetadata(loadedConfig.template_name).then(metadata => {
        templateMetadata = metadata;
        // Find and select the template
        const template = baseTemplates.find(t => t.name === loadedConfig.template_name);
        if (template) {
          selectedTemplate = template;
        }
      }).catch(e => {
        console.error('Failed to load template metadata:', e);
      });
    }
    
    showTemplateManager = false;
  }

  function handleImportTemplate() {
    showImportDialog = true;
  }

  function handleTemplateImported(templateName: string) {
    console.log('Template imported:', templateName);
    // Optionally reload the template list or show a success message
  }

  function toggleTemplateManager() {
    showTemplateManager = !showTemplateManager;
  }
</script>

<div class="space-y-5">
  <!-- Experimental Feature Notice -->
  <div class="bg-amber-50 border border-amber-200 rounded-lg p-4">
    <div class="flex items-start gap-3">
      <svg class="w-5 h-5 text-amber-600 flex-shrink-0 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
        <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126zM12 15.75h.007v.008H12v-.008z" />
      </svg>
      <div class="flex-1">
        <h3 class="text-[13px] font-semibold text-amber-900">{t.experimentalFeature || '实验性功能'}</h3>
        <p class="text-[12px] text-amber-700 mt-1">
          {t.experimentalFeatureDesc || '自定义部署功能目前处于实验阶段，可能存在不稳定情况。请谨慎使用于生产环境。'}
        </p>
      </div>
    </div>
  </div>

  <!-- Tabs -->
  <div class="flex gap-2 border-b border-gray-100">
    <button
      class="px-4 py-2 text-[13px] font-medium transition-colors {activeTab === 'create' ? 'text-blue-600 border-b-2 border-blue-600' : 'text-gray-600 hover:text-gray-900'}"
      onclick={() => activeTab = 'create'}
    >
      {t.createDeployment || '创建部署'}
    </button>
    <button
      class="px-4 py-2 text-[13px] font-medium transition-colors {activeTab === 'manage' ? 'text-blue-600 border-b-2 border-blue-600' : 'text-gray-600 hover:text-gray-900'}"
      onclick={() => activeTab = 'manage'}
    >
      {t.deploymentManagement || '部署管理'}
    </button>
  </div>

  {#if activeTab === 'create'}

  {#if error}
    <div class="flex items-center gap-3 px-4 py-3 bg-red-50 border border-red-100 rounded-lg">
      <svg class="w-4 h-4 text-red-500 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
        <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z" />
      </svg>
      <span class="text-[13px] text-red-700 flex-1">{error}</span>
      <button 
        class="text-red-400 hover:text-red-600" 
        onclick={() => error = ''}
        aria-label="Close error message"
      >
        <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
        </svg>
      </button>
    </div>
  {/if}

  <!-- Deployment Success Message -->
  {#if deploymentResult}
    <div class="flex items-start gap-3 px-4 py-3 bg-green-50 border border-green-200 rounded-lg">
      <svg class="w-5 h-5 text-green-600 flex-shrink-0 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
        <path stroke-linecap="round" stroke-linejoin="round" d="M9 12.75L11.25 15 15 9.75M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
      </svg>
      <div class="flex-1">
        <h4 class="text-[13px] font-semibold text-green-900">{t.deploymentCreated || '部署创建成功'}</h4>
        <p class="text-[12px] text-green-700 mt-1">
          {t.deploymentId || '部署 ID'}: <span class="font-mono font-medium">{deploymentResult.id}</span>
        </p>
        <p class="text-[12px] text-green-700">
          {t.status || '状态'}: <span class="font-medium">{deploymentResult.status}</span>
        </p>
        <p class="text-[11px] text-green-600 mt-2">
          {t.checkDeploymentManagement || '您可以在"部署管理"页面查看部署详情和日志。'}
        </p>
      </div>
      <button 
        class="text-green-400 hover:text-green-600" 
        onclick={() => deploymentResult = null}
        aria-label="Close success message"
      >
        <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
        </svg>
      </button>
    </div>
  {/if}

  <!-- Deployment Error Message -->
  {#if deploymentError}
    <div class="flex items-start gap-3 px-4 py-3 bg-red-50 border border-red-200 rounded-lg">
      <svg class="w-5 h-5 text-red-600 flex-shrink-0 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
        <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126zM12 15.75h.007v.008H12v-.008z" />
      </svg>
      <div class="flex-1">
        <h4 class="text-[13px] font-semibold text-red-900">{t.deploymentFailed || '部署失败'}</h4>
        <p class="text-[12px] text-red-700 mt-1">{deploymentError}</p>
        <p class="text-[11px] text-red-600 mt-2">
          {t.checkConfigAndRetry || '请检查配置并重试。'}
        </p>
      </div>
      <button 
        class="text-red-400 hover:text-red-600" 
        onclick={() => deploymentError = ''}
        aria-label="Close error message"
      >
        <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
        </svg>
      </button>
    </div>
  {/if}

  {#if isLoading}
    <div class="flex items-center justify-center h-64">
      <div class="w-6 h-6 border-2 border-gray-100 border-t-gray-900 rounded-full animate-spin"></div>
    </div>
  {:else}
    <!-- Template Selector -->
    <TemplateSelector 
      {t}
      templates={baseTemplates}
      selectedTemplate={selectedTemplate}
      onSelect={handleTemplateSelect}
    />

    <!-- Template Management Actions -->
    <div class="flex items-center gap-3">
      <button
        class="flex items-center gap-2 px-4 py-2 text-[13px] font-medium text-gray-700 bg-white border border-gray-300 hover:bg-gray-50 rounded-lg transition-colors"
        onclick={toggleTemplateManager}
      >
        <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M9 12h3.75M9 15h3.75M9 18h3.75m3 .75H18a2.25 2.25 0 002.25-2.25V6.108c0-1.135-.845-2.098-1.976-2.192a48.424 48.424 0 00-1.123-.08m-5.801 0c-.065.21-.1.433-.1.664 0 .414.336.75.75.75h4.5a.75.75 0 00.75-.75 2.25 2.25 0 00-.1-.664m-5.8 0A2.251 2.251 0 0113.5 2.25H15c1.012 0 1.867.668 2.15 1.586m-5.8 0c-.376.023-.75.05-1.124.08C9.095 4.01 8.25 4.973 8.25 6.108V8.25m0 0H4.875c-.621 0-1.125.504-1.125 1.125v11.25c0 .621.504 1.125 1.125 1.125h9.75c.621 0 1.125-.504 1.125-1.125V9.375c0-.621-.504-1.125-1.125-1.125H8.25zM6.75 12h.008v.008H6.75V12zm0 3h.008v.008H6.75V15zm0 3h.008v.008H6.75V18z" />
        </svg>
        {showTemplateManager ? (t.hideTemplates || '隐藏模板') : (t.manageTemplates || '管理模板')}
      </button>

      {#if selectedTemplate && templateMetadata && config.provider}
        <button
          class="flex items-center gap-2 px-4 py-2 text-[13px] font-medium text-white bg-blue-600 hover:bg-blue-700 rounded-lg transition-colors"
          onclick={handleSaveTemplate}
        >
          <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M17 16v2a2 2 0 01-2 2H5a2 2 0 01-2-2v-7a2 2 0 012-2h2m3-4H9a2 2 0 00-2 2v7a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-1m-1 4l-3 3m0 0l-3-3m3 3V3" />
          </svg>
          {t.saveAsTemplate || '保存为模板'}
        </button>
      {/if}

      <button
        class="flex items-center gap-2 px-4 py-2 text-[13px] font-medium text-gray-700 bg-white border border-gray-300 hover:bg-gray-50 rounded-lg transition-colors"
        onclick={handleImportTemplate}
      >
        <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M3 16.5v2.25A2.25 2.25 0 005.25 21h13.5A2.25 2.25 0 0021 18.75V16.5m-13.5-9L12 3m0 0l4.5 4.5M12 3v13.5" />
        </svg>
        {t.importTemplate || '导入模板'}
      </button>
    </div>

    <!-- Config Template Manager -->
    {#if showTemplateManager}
      <ConfigTemplateManager
        {t}
        onLoadTemplate={handleLoadTemplate}
      />
    {/if}

    <!-- Config Editor -->
    {#if selectedTemplate && templateMetadata}
      <ConfigEditor 
        {t}
        template={templateMetadata}
        {config}
        {validation}
        onConfigUpdate={handleConfigUpdate}
        onValidationUpdate={handleValidationUpdate}
      />

      <!-- Deployment Preview -->
      <DeploymentPreview
        {t}
        {config}
        template={templateMetadata}
        {costEstimate}
        {validation}
        {isDeploying}
        onDeploy={handleDeploy}
      />
    {/if}
  {/if}

  {:else if activeTab === 'manage'}
    <!-- Deployment Management Tab -->
    <DeploymentManagement {t} />
  {/if}
</div>

<!-- Save Template Dialog -->
<SaveTemplateDialog
  {t}
  show={showSaveDialog}
  {config}
  onClose={() => showSaveDialog = false}
  onSaved={handleTemplateSaved}
/>

<!-- Import Template Dialog -->
<ImportExportDialog
  {t}
  show={showImportDialog}
  mode="import"
  onClose={() => showImportDialog = false}
  onImported={handleTemplateImported}
/>

<style>
  /* Component-specific styles if needed */
</style>
