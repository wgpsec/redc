<script>

  import { onMount } from 'svelte';
  import Modal from '../UI/Modal.svelte';
  import { GetProvidersConfig, SaveProvidersConfig, ListProfiles, GetActiveProfile, SetActiveProfile, CreateProfile, UpdateProfile, DeleteProfile, UpdateProfileAIConfig, UpdateProfileFallbackProviders, GetProfileFallbackProviders } from '../../../wailsjs/go/main/App.js';

  // Credentials state
let { t } = $props();
  let providersConfig = $state({ configPath: '', providers: [] });
  let credentialsLoading = $state(false);
  let credentialsSaving = $state({});
  let editingProvider = $state(null);
  /** @type {Record<string, string>} */
  let editFields = $state({});
  let customConfigPath = $state('');
  let profiles = $state([]);
  let activeProfileId = $state('');
  let profileForm = $state({ name: '', configPath: '', templateDir: '' });
  let profileLoading = $state(false);
  let profileSaving = $state(false);
  let profileSaved = $state('');
  let profileError = $state('');
  let error = $state('');
  let saveConfirm = $state({ show: false, providerName: '' });
  let securityDismissed = $state(false);
  let providerSearch = $state('');
  let showProfileDetails = $state(false);

  // AI Configuration state
  let aiConfig = $state({
    provider: 'openai',
    apiKey: '',
    baseUrl: '',
    model: '',
    maxToolRounds: 0,
    enableAskUser: true,
    enableMemory: true,
    contextWindow: 0
  });
  let aiConfigSaving = $state(false);
  let aiConfigSaved = $state(false);
  let showApiKey = $state(false);

  // Fallback providers state
  let fallbackProviders = $state([]);
  let fallbackSaving = $state(false);
  let fallbackSaved = $state(false);
  let showFallbackApiKey = $state({});

  // Provider presets
  const aiProviderPresets = {
    openai: {
      name: 'OpenAI API 兼容',
      nameEn: 'OpenAI API Compatible',
      baseUrl: 'https://api.openai.com/v1',
      placeholder: 'gpt-4o, gpt-4o-mini, deepseek-chat, MiniMax-M2.1...',
      defaultModel: 'gpt-4o'
    },
    anthropic: {
      name: 'Anthropic API 兼容',
      nameEn: 'Anthropic API Compatible',
      baseUrl: 'https://api.anthropic.com',
      placeholder: 'claude-sonnet-4-20250514, claude-3-5-sonnet-20241022...',
      defaultModel: 'claude-sonnet-4-20250514'
    }
  };

  async function loadProvidersConfig() {
    credentialsLoading = true;
    try {
      providersConfig = await GetProvidersConfig(customConfigPath);
    } catch (e) {
      error = e.message || String(e);
    } finally {
      credentialsLoading = false;
    }
  }

  async function loadProfiles() {
    profileLoading = true;
    profileError = '';
    try {
      const [list, active] = await Promise.all([
        ListProfiles(),
        GetActiveProfile()
      ]);
      profiles = list || [];
      if (active && active.id) {
        activeProfileId = active.id;
        profileForm = {
          name: active.name || '',
          configPath: active.configPath || '',
          templateDir: active.templateDir || ''
        };
        if (active.aiConfig) {
          aiConfig = {
            provider: active.aiConfig.provider || 'openai',
            apiKey: active.aiConfig.apiKey || '',
            baseUrl: active.aiConfig.baseUrl || aiProviderPresets[active.aiConfig.provider || 'openai'].baseUrl,
            model: active.aiConfig.model || aiProviderPresets[active.aiConfig.provider || 'openai'].defaultModel,
            maxToolRounds: active.aiConfig.maxToolRounds || 0,
            enableAskUser: active.aiConfig.enableAskUser !== false,
            enableMemory: active.aiConfig.enableMemory !== false,
            contextWindow: active.aiConfig.contextWindow || 0
          };
        } else {
          const preset = aiProviderPresets['openai'];
          aiConfig = {
            provider: 'openai',
            apiKey: '',
            baseUrl: preset.baseUrl,
            model: preset.defaultModel,
            maxToolRounds: 0,
            enableAskUser: true,
            enableMemory: true,
            contextWindow: 0
          };
        }
        customConfigPath = profileForm.configPath;
        // Load fallback providers
        try {
          const fbs = await GetProfileFallbackProviders(activeProfileId);
          fallbackProviders = (fbs || []).map(fb => ({...fb}));
        } catch {
          fallbackProviders = [];
        }
      }
    } catch (e) {
      profileError = e.message || String(e);
    } finally {
      profileLoading = false;
    }
  }

  async function handleProfileChange(id) {
    if (!id) return;
    profileLoading = true;
    profileError = '';
    try {
      const active = await SetActiveProfile(id);
      activeProfileId = active.id;
      profileForm = {
        name: active.name || '',
        configPath: active.configPath || '',
        templateDir: active.templateDir || ''
      };
      customConfigPath = profileForm.configPath;
      await loadProvidersConfig();
    } catch (e) {
      profileError = e.message || String(e);
    } finally {
      profileLoading = false;
    }
  }

  async function handleCreateProfile() {
    if (!profileForm.name) {
      profileError = t.profileNameRequired;
      return;
    }
    profileSaving = true;
    profileError = '';
    try {
      const created = await CreateProfile(profileForm.name, profileForm.configPath, profileForm.templateDir);
      profiles = await ListProfiles();
      await handleProfileChange(created.id);
      profileSaved = 'create';
      setTimeout(() => { profileSaved = ''; }, 1500);
    } catch (e) {
      profileError = e.message || String(e);
    } finally {
      profileSaving = false;
    }
  }

  async function handleSaveProfile() {
    if (!activeProfileId) return;
    if (!profileForm.name) {
      profileError = t.profileNameRequired;
      return;
    }
    profileSaving = true;
    profileError = '';
    try {
      const updated = await UpdateProfile(activeProfileId, profileForm.name, profileForm.configPath, profileForm.templateDir);
      profiles = await ListProfiles();
      activeProfileId = updated.id;
      profileForm = {
        name: updated.name || '',
        configPath: updated.configPath || '',
        templateDir: updated.templateDir || ''
      };
      customConfigPath = profileForm.configPath;
      await SetActiveProfile(activeProfileId);
      await loadProvidersConfig();
      profileSaved = 'save';
      setTimeout(() => { profileSaved = ''; }, 1500);
    } catch (e) {
      profileError = e.message || String(e);
    } finally {
      profileSaving = false;
    }
  }

  async function handleDeleteProfile() {
    if (!activeProfileId) return;
    profileSaving = true;
    profileError = '';
    try {
      await DeleteProfile(activeProfileId);
      await loadProfiles();
      if (activeProfileId) {
        await handleProfileChange(activeProfileId);
      }
      profileSaved = 'delete';
      setTimeout(() => { profileSaved = ''; }, 1500);
    } catch (e) {
      profileError = e.message || String(e);
    } finally {
      profileSaving = false;
    }
  }

  async function handleSaveAIConfig() {
    if (!activeProfileId) return;
    aiConfigSaving = true;
    aiConfigSaved = false;
    try {
      await UpdateProfileAIConfig(activeProfileId, aiConfig.provider, aiConfig.apiKey, aiConfig.baseUrl, aiConfig.model, aiConfig.maxToolRounds || 0, aiConfig.enableAskUser !== false, aiConfig.enableMemory !== false, aiConfig.contextWindow || 0);
      aiConfigSaved = true;
      setTimeout(() => { aiConfigSaved = false; }, 2000);
    } catch (e) {
      error = e.message || String(e);
    } finally {
      aiConfigSaving = false;
    }
  }

  function addFallbackProvider() {
    fallbackProviders = [...fallbackProviders, { name: '', provider: 'openai', apiKey: '', baseUrl: '', model: '' }];
  }

  function removeFallbackProvider(index) {
    fallbackProviders = fallbackProviders.filter((_, i) => i !== index);
  }

  async function handleSaveFallbackProviders() {
    if (!activeProfileId) return;
    fallbackSaving = true;
    fallbackSaved = false;
    try {
      // Filter out empty entries
      const validFallbacks = fallbackProviders.filter(fb => fb.apiKey && fb.baseUrl && fb.model);
      await UpdateProfileFallbackProviders(activeProfileId, validFallbacks);
      fallbackSaved = true;
      setTimeout(() => { fallbackSaved = false; }, 2000);
    } catch (e) {
      error = e.message || String(e);
    } finally {
      fallbackSaving = false;
    }
  }

  function handleProviderChange() {
    const preset = aiProviderPresets[aiConfig.provider];
    if (preset) {
      aiConfig.baseUrl = preset.baseUrl;
      // Don't auto-fill model, let user input custom model name
      if (!aiConfig.model) {
        aiConfig.model = '';
      }
    }
  }

  function startEditProvider(provider) {
    editingProvider = provider.name;
    editFields = {};
    // Initialize edit fields with empty values (user must re-enter secrets)
    for (const key of Object.keys(provider.fields)) {
      // For non-secret fields (like region), pre-fill with current value
      if (!provider.hasSecrets || !provider.hasSecrets[key]) {
        editFields[key] = provider.fields[key] || '';
      } else {
        editFields[key] = '';
      }
    }
  }

  function cancelEditProvider() {
    editingProvider = null;
    editFields = {};
  }

  function showSaveConfirm(providerName) {
    saveConfirm = { show: true, providerName };
  }

  function cancelSave() {
    saveConfirm = { show: false, providerName: '' };
  }

  async function confirmSave() {
    const providerName = saveConfirm.providerName;
    saveConfirm = { show: false, providerName: '' };
    await saveProviderCredentials(providerName);
  }

  async function saveProviderCredentials(providerName) {
    credentialsSaving[providerName] = true;
    credentialsSaving = credentialsSaving;
    try {
      await SaveProvidersConfig(providerName, editFields, customConfigPath);
      editingProvider = null;
      editFields = {};
      await loadProvidersConfig();
    } catch (e) {
      error = e.message || String(e);
    } finally {
      credentialsSaving[providerName] = false;
      credentialsSaving = credentialsSaving;
    }
  }

  function getFieldLabel(key) {
    const labels = {
      accessKey: 'Access Key',
      secretKey: 'Secret Key',
      secretId: 'Secret ID',
      publicKey: 'Public Key',
      privateKey: 'Private Key',
      projectId: 'Project ID',
      region: 'Region',
      credentials: t.credentialsJson || '凭据 JSON 路径',
      project: t.project || '项目 ID',
      clientId: 'Client ID',
      clientSecret: 'Client Secret',
      subscriptionId: 'Subscription ID',
      tenantId: 'Tenant ID',
      user: t.user || '用户 OCID',
      tenancy: 'Tenancy OCID',
      fingerprint: t.fingerprint || '指纹',
      keyFile: t.keyFile || '私钥文件路径',
      email: t.email || '邮箱',
      apiKey: 'API Key',
    };
    return labels[key] || key;
  }
  
  // 定义字段显示顺序
  const fieldOrder = {
    'alibaba': ['accessKey', 'secretKey', 'region'],
    '阿里云': ['accessKey', 'secretKey', 'region'],
    'AWS': ['accessKey', 'secretKey', 'region'],
    '腾讯云': ['secretId', 'secretKey', 'region'],
    '火山引擎': ['accessKey', 'secretKey', 'region'],
    '华为云': ['accessKey', 'secretKey', 'region'],
    'UCloud': ['publicKey', 'privateKey', 'projectId', 'region'],
    'Ctyun': ['accessKey', 'secretKey', 'region'],
    'Vultr': ['apiKey'],
    'Google Cloud': ['credentials', 'project', 'region'],
    'Azure': ['clientId', 'clientSecret', 'subscriptionId', 'tenantId'],
    'Oracle Cloud': ['user', 'tenancy', 'fingerprint', 'keyFile', 'region'],
    'Cloudflare': ['email', 'apiKey'],
  };
  
  function getOrderedFields(providerName, fields) {
    const order = fieldOrder[providerName] || [];
    const ordered = [];
    const remaining = [];
    
    // 按顺序添加字段
    for (const key of order) {
      if (fields[key] !== undefined) {
        ordered.push([key, fields[key]]);
      }
    }
    
    // 添加不在顺序中的字段
    for (const [key, value] of Object.entries(fields)) {
      if (!order.includes(key)) {
        remaining.push([key, value]);
      }
    }
    
    return [...ordered, ...remaining];
  }

  function isSecretField(key) {
    const secrets = ['accessKey', 'secretKey', 'secretId', 'clientId', 'clientSecret', 'subscriptionId', 'tenantId', 'user', 'tenancy', 'fingerprint', 'apiKey'];
    return secrets.includes(key);
  }

  function isProviderConfigured(provider) {
    if (!provider || !provider.fields) return false;
    return Object.entries(provider.fields).some(([key, v]) => {
      if (!v || v === '') return false;
      if (key === 'region') return false; // region 为空不影响配置状态
      return true; // 包括 *** 遮罩值，说明已配置
    });
  }

  let filteredProviders = $derived(
    (providersConfig.providers || []).filter(p => {
      if (!providerSearch) return true;
      const q = providerSearch.toLowerCase();
      const name = (t[p.name] || p.name || '').toLowerCase();
      return name.includes(q) || p.name.toLowerCase().includes(q);
    })
  );

  let configuredCount = $derived(
    (providersConfig.providers || []).filter(isProviderConfigured).length
  );

  onMount(() => {
    loadProfiles();
    loadProvidersConfig();
  });

  // Export refresh function for parent component
  export function refresh() {
    loadProfiles();
    loadProvidersConfig();
  }

</script>

<div class="max-w-3xl lg:max-w-5xl xl:max-w-full space-y-5">
  <!-- Profile Management -->
  <div class="bg-white rounded-xl border border-gray-100 p-5">
    <div class="flex items-center justify-between mb-4">
      <div>
        <h3 class="text-[14px] font-semibold text-gray-900">{t.profileManage}</h3>
        <p class="text-[12px] text-gray-500">{t.profileHint}</p>
      </div>
      <div class="flex items-center gap-2">
        <button
          class="h-8 px-3 text-gray-500 hover:text-gray-700 bg-white border border-gray-200 hover:bg-gray-50 text-[11px] font-medium rounded-lg transition-colors disabled:opacity-50 inline-flex items-center gap-1 cursor-pointer"
          onclick={handleCreateProfile}
          disabled={profileSaving}
        >
          <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M12 4.5v15m7.5-7.5h-15" /></svg>
          {t.createProfile}
        </button>
      </div>
    </div>

    <div class="grid grid-cols-1 md:grid-cols-[1fr_1fr_auto] gap-4 items-end">
      <div>
        <label for="profile" class="block text-[12px] font-medium text-gray-500 mb-1.5">{t.profile}</label>
        <select
          id="profile"
          class="w-full h-10 px-3 text-[13px] bg-gray-50 border-0 rounded-lg text-gray-900 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow"
          bind:value={activeProfileId}
          onchange={() => handleProfileChange(activeProfileId)}
        >
          <option value="" disabled>{t.selectProfile}</option>
          {#each profiles as p}
            <option value={p.id}>{p.name}</option>
          {/each}
        </select>
      </div>
      <div>
        <label for="profileName" class="block text-[12px] font-medium text-gray-500 mb-1.5">{t.profileName}</label>
        <input
          id="profileName"
          type="text"
          class="w-full h-10 px-3 text-[13px] bg-gray-50 border-0 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow"
          bind:value={profileForm.name}
        />
      </div>
      <div class="flex gap-2">
        <button
          class="h-10 px-4 bg-emerald-500 text-white text-[12px] font-medium rounded-lg hover:bg-emerald-600 transition-colors disabled:opacity-50 inline-flex items-center justify-center gap-1.5 cursor-pointer"
          onclick={handleSaveProfile}
          disabled={profileSaving || !activeProfileId}
        >
          {#if profileSaving}
            <svg class="animate-spin h-3.5 w-3.5" viewBox="0 0 24 24" fill="none"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path></svg>
          {:else if profileSaved === 'save'}
            <svg class="h-3.5 w-3.5" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd" /></svg>
          {/if}
          {profileSaved === 'save' ? (t.saved || '已保存') : t.saveProfile}
        </button>
        <button
          class="h-10 px-3 text-red-500 hover:bg-red-50 text-[12px] font-medium rounded-lg transition-colors disabled:opacity-50 inline-flex items-center cursor-pointer"
          onclick={handleDeleteProfile}
          disabled={profileSaving || !activeProfileId}
          title={t.deleteProfile}
        >
          <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M14.74 9l-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 01-2.244 2.077H8.084a2.25 2.25 0 01-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 00-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 013.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 00-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 00-7.5 0" /></svg>
        </button>
      </div>
    </div>

    <!-- Profile details (collapsible) -->
    <button
      class="mt-3 text-[11px] text-gray-400 hover:text-gray-600 transition-colors inline-flex items-center gap-1 cursor-pointer"
      onclick={() => showProfileDetails = !showProfileDetails}
    >
      <svg class="w-3 h-3 transition-transform" class:rotate-90={showProfileDetails} fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M8.25 4.5l7.5 7.5-7.5 7.5" /></svg>
      {t.advancedSettings || '高级设置'}
    </button>
    {#if showProfileDetails}
      <div class="mt-3 grid grid-cols-1 gap-3 animate-in">
        <div>
          <label for="configPath" class="block text-[12px] font-medium text-gray-500 mb-1.5">{t.configPath}</label>
          <input
            id="configPath"
            type="text"
            placeholder={t.defaultPath}
            class="w-full h-10 px-3 text-[13px] bg-gray-50 border-0 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow font-mono"
            bind:value={profileForm.configPath}
            oninput={() => { customConfigPath = profileForm.configPath; }}
          />
        </div>
        <div>
          <label for="templateDir" class="block text-[12px] font-medium text-gray-500 mb-1.5">{t.templateDir}</label>
          <input
            id="templateDir"
            type="text"
            placeholder={t.defaultPath}
            class="w-full h-10 px-3 text-[13px] bg-gray-50 border-0 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow font-mono"
            bind:value={profileForm.templateDir}
          />
        </div>
        <div class="flex items-center gap-2">
          <button
            class="h-8 px-3 text-gray-500 hover:text-gray-700 bg-white border border-gray-200 hover:bg-gray-50 text-[11px] font-medium rounded-lg transition-colors disabled:opacity-50 inline-flex items-center gap-1 cursor-pointer"
            onclick={loadProvidersConfig}
            disabled={credentialsLoading}
          >
            {#if credentialsLoading}
              <svg class="animate-spin h-3 w-3" viewBox="0 0 24 24" fill="none"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path></svg>
            {/if}
            {credentialsLoading ? t.loading : (t.loadConfig || '重新加载配置')}
          </button>
          {#if providersConfig.configPath}
            <span class="text-[11px] text-gray-400 font-mono">{providersConfig.configPath}</span>
          {/if}
        </div>
        <p class="text-[11px] text-gray-400">{t.profileSwitchHint}</p>
      </div>
    {/if}

    {#if profileError}
      <div class="mt-3 text-[12px] text-red-600">{profileError}</div>
    {/if}
  </div>

  <!-- Security Notice (dismissible) -->
  {#if !securityDismissed}
  <div class="flex items-start gap-3 px-4 py-3 bg-amber-50 border border-amber-100 rounded-lg">
    <svg class="w-4 h-4 text-amber-500 flex-shrink-0 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
      <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126zM12 15.75h.007v.008H12v-.008z" />
    </svg>
    <div class="text-[12px] text-amber-800 flex-1">
      <strong>{t.securityTip}</strong>{t.securityInfo}
    </div>
    <button class="text-amber-400 hover:text-amber-600 flex-shrink-0 cursor-pointer" onclick={() => securityDismissed = true} aria-label={t.close}>
      <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" /></svg>
    </button>
  </div>
  {/if}

  <!-- AI Configuration Card -->
  <div class="bg-white rounded-xl border border-gray-100 overflow-hidden">
    <div class="px-5 py-4 border-b border-gray-100 flex items-center justify-between">
      <div class="flex items-center gap-2">
        <svg class="w-5 h-5 text-red-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M9.813 15.904L9 18.75l-.813-2.846a4.5 4.5 0 00-3.09-3.09L2.25 12l2.846-.813a4.5 4.5 0 003.09-3.09L9 5.25l.813 2.846a4.5 4.5 0 003.09 3.09L15.75 12l-2.846.813a4.5 4.5 0 00-3.09 3.09zM18.259 8.715L18 9.75l-.259-1.035a3.375 3.375 0 00-2.455-2.456L14.25 6l1.036-.259a3.375 3.375 0 002.455-2.456L18 2.25l.259 1.035a3.375 3.375 0 002.456 2.456L21.75 6l-1.035.259a3.375 3.375 0 00-2.456 2.456zM16.894 20.567L16.5 21.75l-.394-1.183a2.25 2.25 0 00-1.423-1.423L13.5 18.75l1.183-.394a2.25 2.25 0 001.423-1.423l.394-1.183.394 1.183a2.25 2.25 0 001.423 1.423l1.183.394-1.183.394a2.25 2.25 0 00-1.423 1.423z" />
        </svg>
        <h3 class="text-[14px] font-semibold text-gray-900">{t.aiConfig || 'AI Configuration'}</h3>
      </div>
      <div class="flex items-center gap-2">
        {#if aiConfigSaved}
          <span class="text-[12px] text-emerald-600 flex items-center gap-1">
            <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M4.5 12.75l6 6 9-13.5" />
            </svg>
            {t.saved || 'Saved'}
          </span>
        {/if}
        <button 
          class="px-3 py-1 text-[12px] font-medium text-white bg-emerald-500 rounded-md hover:bg-emerald-600 transition-colors disabled:opacity-50 inline-flex items-center gap-1.5"
          onclick={handleSaveAIConfig}
          disabled={aiConfigSaving || !activeProfileId}
        >
          {#if aiConfigSaving}
            <svg class="animate-spin h-3.5 w-3.5" viewBox="0 0 24 24" fill="none"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path></svg>
            {t.saving || 'Saving...'}
          {:else}
            {t.save || 'Save'}
          {/if}
        </button>
      </div>
    </div>
    <div class="p-5 space-y-5">
      <!-- Connection settings -->
      <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div>
          <label for="aiProvider" class="block text-[11px] font-medium text-gray-500 mb-1">{t.aiProvider || 'Provider'}</label>
          <select 
            id="aiProvider"
            class="w-full h-9 px-3 text-[12px] bg-gray-50 border-0 rounded-lg text-gray-900 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow"
            bind:value={aiConfig.provider}
            onchange={handleProviderChange}
          >
            {#each Object.entries(aiProviderPresets) as [key, preset]}
              <option value={key}>{t[key + 'Compatible'] || preset.name}</option>
            {/each}
          </select>
        </div>
        <div>
          <label for="aiModel" class="block text-[11px] font-medium text-gray-500 mb-1">{t.aiModel || 'Model'}</label>
          <input 
            id="aiModel"
            type="text"
            placeholder={aiProviderPresets[aiConfig.provider]?.placeholder || 'Enter model name'}
            class="w-full h-9 px-3 text-[12px] bg-gray-50 border-0 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow font-mono"
            bind:value={aiConfig.model}
          />
          <p class="text-[10px] text-gray-500 mt-1">{t.aiModelHint || '支持任意兼容的模型名称'}</p>
        </div>
        <div class="md:col-span-2">
          <label for="aiApiKey" class="block text-[11px] font-medium text-gray-500 mb-1">
            {t.aiApiKey || 'API Key'}
            <svg class="inline-block ml-1 w-3 h-3 text-amber-500 -mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M16.5 10.5V6.75a4.5 4.5 0 10-9 0v3.75m-.75 11.25h10.5a2.25 2.25 0 002.25-2.25v-6.75a2.25 2.25 0 00-2.25-2.25H6.75a2.25 2.25 0 00-2.25 2.25v6.75a2.25 2.25 0 002.25 2.25z" /></svg>
          </label>
          <div class="relative">
            <input 
              id="aiApiKey"
              type={showApiKey ? 'text' : 'password'}
              placeholder={t.aiApiKeyPlaceholder || 'Enter your API key'}
              class="w-full h-9 px-3 pr-10 text-[12px] bg-gray-50 border-0 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow font-mono"
              bind:value={aiConfig.apiKey}
            />
            <button 
              type="button"
              class="absolute right-2 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600 transition-colors cursor-pointer"
              onclick={() => showApiKey = !showApiKey}
              aria-label={showApiKey ? (t.hideApiKey || '隐藏API密钥') : (t.showApiKey || '显示API密钥')}
            >
              {#if showApiKey}
                <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M3.98 8.223A10.477 10.477 0 001.934 12C3.226 16.338 7.244 19.5 12 19.5c.993 0 1.953-.138 2.863-.395M6.228 6.228A10.45 10.45 0 0112 4.5c4.756 0 8.773 3.162 10.065 7.498a10.523 10.523 0 01-4.293 5.774M6.228 6.228L3 3m3.228 3.228l3.65 3.65m7.894 7.894L21 21m-3.228-3.228l-3.65-3.65m0 0a3 3 0 10-4.243-4.243m4.242 4.242L9.88 9.88" />
                </svg>
              {:else}
                <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M2.036 12.322a1.012 1.012 0 010-.639C3.423 7.51 7.36 4.5 12 4.5c4.638 0 8.573 3.007 9.963 7.178.07.207.07.431 0 .639C20.577 16.49 16.64 19.5 12 19.5c-4.638 0-8.573-3.007-9.963-7.178z" />
                  <path stroke-linecap="round" stroke-linejoin="round" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                </svg>
              {/if}
            </button>
          </div>
        </div>
        <div class="md:col-span-2">
          <label for="aiBaseUrl" class="block text-[11px] font-medium text-gray-500 mb-1">{t.aiBaseUrl || 'Base URL'}</label>
          <input 
            id="aiBaseUrl"
            type="text"
            placeholder={aiProviderPresets[aiConfig.provider]?.baseUrl || ''}
            class="w-full h-9 px-3 text-[12px] bg-gray-50 border-0 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow font-mono"
            bind:value={aiConfig.baseUrl}
          />
          <p class="text-[10px] text-gray-500 mt-1">{t.aiBaseUrlHint || 'Optional: Override the default API endpoint'}</p>
        </div>
      </div>

      <!-- Agent behavior settings -->
      <div class="border-t border-gray-100 pt-4">
        <h4 class="text-[12px] font-medium text-gray-700 mb-3">{t.agentBehavior || 'Agent 行为设置'}</h4>
        <div class="space-y-3">
          <div>
            <label for="aiMaxRounds" class="block text-[11px] font-medium text-gray-500 mb-1">{t.aiMaxToolRounds || 'Agent 最大工具调用轮次'}</label>
            <div class="flex items-center gap-3">
              <input 
                id="aiMaxRounds"
                type="range"
                min="0"
                max="200"
                step="10"
                class="flex-1 h-2 bg-gray-200 rounded-lg appearance-none cursor-pointer accent-emerald-500"
                bind:value={aiConfig.maxToolRounds}
                oninput={(e) => { aiConfig.maxToolRounds = parseInt(e.target.value); }}
              />
              <span class="text-[12px] font-mono text-gray-700 w-8 text-center">{aiConfig.maxToolRounds || 50}</span>
            </div>
            <p class="text-[10px] text-gray-500 mt-1">{t.aiMaxToolRoundsHint || 'Agent/开源部署模式下的最大工具调用轮次，0 为使用默认值'}</p>
          </div>
          <div>
            <label for="aiContextWindow" class="block text-[11px] font-medium text-gray-500 mb-1">{t.aiContextWindow || '模型上下文窗口 (tokens)'}</label>
            <div class="flex items-center gap-3">
              <input 
                id="aiContextWindow"
                type="range"
                min="0"
                max="200000"
                step="1000"
                class="flex-1 h-2 bg-gray-200 rounded-lg appearance-none cursor-pointer accent-emerald-500"
                bind:value={aiConfig.contextWindow}
                oninput={(e) => { aiConfig.contextWindow = parseInt(e.target.value); }}
              />
              <span class="text-[12px] font-mono text-gray-700 w-16 text-center">{aiConfig.contextWindow ? (aiConfig.contextWindow / 1000).toFixed(0) + 'K' : '120K'}</span>
            </div>
            <p class="text-[10px] text-gray-500 mt-1">{t.aiContextWindowHint || '模型支持的最大上下文长度，超出后自动压缩历史对话，0 为默认 120K'}</p>
          </div>
          <div class="flex items-center justify-between py-2">
            <div>
              <label class="block text-[11px] font-medium text-gray-500">{t.enableAskUser || 'Agent 人机协作决策'}</label>
              <p class="text-[10px] text-gray-500 mt-0.5">{t.enableAskUserHint || 'Agent 遇到需要决策的问题时暂停并向用户询问'}</p>
            </div>
            <button
              type="button"
              class="relative w-9 h-5 flex-shrink-0 rounded-full transition-colors cursor-pointer"
              class:bg-emerald-500={aiConfig.enableAskUser}
              class:bg-gray-300={!aiConfig.enableAskUser}
              onclick={() => { aiConfig.enableAskUser = !aiConfig.enableAskUser; }}
            >
              <span class="absolute top-0.5 left-0.5 w-4 h-4 bg-white rounded-full shadow transition-transform"
                class:translate-x-4={aiConfig.enableAskUser}
                class:translate-x-0={!aiConfig.enableAskUser}></span>
            </button>
          </div>
          <div class="flex items-center justify-between py-2">
            <div>
              <label class="block text-[11px] font-medium text-gray-500">{t.enableMemory || 'Agent 记忆系统'}</label>
              <p class="text-[10px] text-gray-500 mt-0.5">{t.enableMemoryHint || '自动记忆历史操作经验，避免重复踩坑'}</p>
            </div>
            <button
              type="button"
              class="relative w-9 h-5 flex-shrink-0 rounded-full transition-colors cursor-pointer"
              class:bg-emerald-500={aiConfig.enableMemory}
              class:bg-gray-300={!aiConfig.enableMemory}
              onclick={() => { aiConfig.enableMemory = !aiConfig.enableMemory; }}
            >
              <span class="absolute top-0.5 left-0.5 w-4 h-4 bg-white rounded-full shadow transition-transform"
                class:translate-x-4={aiConfig.enableMemory}
                class:translate-x-0={!aiConfig.enableMemory}></span>
            </button>
          </div>
        </div>
      </div>

      <!-- Fallback Providers Section -->
      <div class="border-t border-gray-100 pt-4">
        <div class="flex items-center justify-between mb-3">
          <div>
            <h4 class="text-[12px] font-medium text-gray-700">{t.fallbackProviders || '备用 Provider（自动故障转移）'}</h4>
            <p class="text-[10px] text-gray-500 mt-0.5">{t.fallbackProviderHint || '当主 Provider 出现错误时，自动切换到备用 Provider 继续工作'}</p>
          </div>
          <div class="flex items-center gap-2">
            {#if fallbackSaved}
              <span class="text-[11px] text-emerald-600 flex items-center gap-1">
                <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M4.5 12.75l6 6 9-13.5" /></svg>
                {t.saved || 'Saved'}
              </span>
            {/if}
            {#if fallbackProviders.length > 0}
              <button
                class="px-2.5 py-1 text-[11px] font-medium text-white bg-emerald-500 rounded-md hover:bg-emerald-600 transition-colors disabled:opacity-50"
                onclick={handleSaveFallbackProviders}
                disabled={fallbackSaving || !activeProfileId}
              >
                {fallbackSaving ? (t.saving || 'Saving...') : (t.saveFallbackProviders || '保存备用 Provider')}
              </button>
            {/if}
            <button
              class="px-2.5 py-1 text-[11px] font-medium text-gray-600 bg-gray-100 rounded-md hover:bg-gray-200 transition-colors disabled:opacity-50 inline-flex items-center gap-1"
              onclick={addFallbackProvider}
              disabled={!activeProfileId}
            >
              <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M12 4.5v15m7.5-7.5h-15" /></svg>
              {t.addFallbackProvider || '添加'}
            </button>
          </div>
        </div>

        {#if fallbackProviders.length === 0}
          <div class="text-center py-4">
            <svg class="w-8 h-8 text-gray-300 mx-auto mb-2" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
              <path stroke-linecap="round" stroke-linejoin="round" d="M7.5 21L3 16.5m0 0L7.5 12M3 16.5h13.5m0-13.5L21 7.5m0 0L16.5 12M21 7.5H7.5" />
            </svg>
            <p class="text-[11px] text-gray-400">{t.noFallbackProviders || '未配置备用 Provider，主 Provider 故障时将无法自动切换'}</p>
          </div>
        {:else}
          <div class="space-y-3">
            {#each fallbackProviders as fb, idx}
              <div class="bg-gray-50 rounded-lg p-3 relative group">
                <button
                  class="absolute top-2 right-2 text-gray-400 hover:text-red-500 transition-colors opacity-0 group-hover:opacity-100 cursor-pointer"
                  onclick={() => removeFallbackProvider(idx)}
                  title={t.removeFallbackProvider || '移除'}
                >
                  <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" /></svg>
                </button>
                <div class="flex items-center gap-2 mb-2">
                  <span class="text-[10px] font-medium text-gray-400 bg-gray-200 px-1.5 py-0.5 rounded">#{idx + 1}</span>
                  <span class="text-[11px] font-medium text-gray-600">{fb.name || `Fallback ${idx + 1}`}</span>
                </div>
                <div class="grid grid-cols-2 gap-2">
                  <div>
                    <label class="block text-[10px] text-gray-400 mb-0.5">{t.fallbackProviderName || '名称'}</label>
                    <input type="text" placeholder="e.g. backup-deepseek"
                      class="w-full h-7 px-2 text-[11px] bg-white border-0 rounded text-gray-900 placeholder-gray-400 focus:ring-1 focus:ring-gray-400 font-mono"
                      bind:value={fb.name} />
                  </div>
                  <div>
                    <label class="block text-[10px] text-gray-400 mb-0.5">{t.fallbackProviderType || 'Provider 类型'}</label>
                    <select class="w-full h-7 px-2 text-[11px] bg-white border-0 rounded text-gray-900 focus:ring-1 focus:ring-gray-400"
                      bind:value={fb.provider}>
                      {#each Object.entries(aiProviderPresets) as [key, preset]}
                        <option value={key}>{t[key + 'Compatible'] || preset.name}</option>
                      {/each}
                    </select>
                  </div>
                  <div>
                    <label class="block text-[10px] text-gray-400 mb-0.5">{t.fallbackProviderModel || '模型'}</label>
                    <input type="text" placeholder={aiProviderPresets[fb.provider]?.placeholder || 'model name'}
                      class="w-full h-7 px-2 text-[11px] bg-white border-0 rounded text-gray-900 placeholder-gray-400 focus:ring-1 focus:ring-gray-400 font-mono"
                      bind:value={fb.model} />
                  </div>
                  <div>
                    <label class="block text-[10px] text-gray-400 mb-0.5">{t.fallbackProviderBaseUrl || 'Base URL'}</label>
                    <input type="text" placeholder={aiProviderPresets[fb.provider]?.baseUrl || ''}
                      class="w-full h-7 px-2 text-[11px] bg-white border-0 rounded text-gray-900 placeholder-gray-400 focus:ring-1 focus:ring-gray-400 font-mono"
                      bind:value={fb.baseUrl} />
                  </div>
                  <div class="col-span-2">
                    <label class="block text-[10px] text-gray-400 mb-0.5">{t.fallbackProviderApiKey || 'API Key'}</label>
                    <div class="relative">
                      <input type={showFallbackApiKey[idx] ? 'text' : 'password'} placeholder="sk-..."
                        class="w-full h-7 px-2 pr-7 text-[11px] bg-white border-0 rounded text-gray-900 placeholder-gray-400 focus:ring-1 focus:ring-gray-400 font-mono"
                        bind:value={fb.apiKey} />
                      <button type="button"
                        class="absolute right-1.5 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600 cursor-pointer"
                        onclick={() => { showFallbackApiKey[idx] = !showFallbackApiKey[idx]; showFallbackApiKey = {...showFallbackApiKey}; }}>
                        <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                          {#if showFallbackApiKey[idx]}
                            <path stroke-linecap="round" stroke-linejoin="round" d="M3.98 8.223A10.477 10.477 0 001.934 12C3.226 16.338 7.244 19.5 12 19.5c.993 0 1.953-.138 2.863-.395M6.228 6.228A10.45 10.45 0 0112 4.5c4.756 0 8.773 3.162 10.065 7.498a10.523 10.523 0 01-4.293 5.774M6.228 6.228L3 3m3.228 3.228l3.65 3.65m7.894 7.894L21 21m-3.228-3.228l-3.65-3.65m0 0a3 3 0 10-4.243-4.243m4.242 4.242L9.88 9.88" />
                          {:else}
                            <path stroke-linecap="round" stroke-linejoin="round" d="M2.036 12.322a1.012 1.012 0 010-.639C3.423 7.51 7.36 4.5 12 4.5c4.638 0 8.573 3.007 9.963 7.178.07.207.07.431 0 .639C20.577 16.49 16.64 19.5 12 19.5c-4.638 0-8.573-3.007-9.963-7.178z" />
                            <path stroke-linecap="round" stroke-linejoin="round" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                          {/if}
                        </svg>
                      </button>
                    </div>
                  </div>
                </div>
              </div>
            {/each}
          </div>
        {/if}
      </div>

      {#if !activeProfileId}
        <p class="text-[11px] text-amber-600 mt-3">{t.aiConfigProfileHint || 'Please select a profile first to configure AI settings'}</p>
      {/if}
    </div>
  </div>

  {#if credentialsLoading}
    <div class="flex items-center justify-center h-32">
      <div class="w-6 h-6 border-2 border-gray-100 border-t-gray-900 rounded-full animate-spin"></div>
    </div>
  {:else if (providersConfig.providers || []).length > 0}
    <!-- Provider header with search and stats -->
    <div class="flex items-center justify-between">
      <div class="flex items-center gap-3">
        <h3 class="text-[14px] font-semibold text-gray-900">{t.cloudCredentials || '云厂商凭据'}</h3>
        <span class="text-[11px] text-gray-400">{configuredCount}/{(providersConfig.providers || []).length} {t.configured || '已配置'}</span>
      </div>
      <div class="relative">
        <svg class="w-3.5 h-3.5 absolute left-2.5 top-1/2 -translate-y-1/2 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M21 21l-5.197-5.197m0 0A7.5 7.5 0 105.196 5.196a7.5 7.5 0 0010.607 10.607z" /></svg>
        <input
          type="text"
          placeholder={t.searchProvider || '搜索厂商...'}
          class="h-8 pl-8 pr-3 text-[12px] bg-gray-50 border-0 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow w-48"
          bind:value={providerSearch}
        />
      </div>
    </div>

    <!-- Provider Cards -->
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      {#each filteredProviders as provider}
        {@const configured = isProviderConfigured(provider)}
        <div class="bg-white rounded-xl border overflow-hidden transition-colors {configured ? 'border-gray-200 border-l-2 border-l-emerald-500' : 'border-gray-100 opacity-75 hover:opacity-100'}">
          <div class="px-5 py-4 border-b border-gray-100 flex items-center justify-between">
            <div class="flex items-center gap-2">
              <span class="w-2 h-2 rounded-full flex-shrink-0 {configured ? 'bg-emerald-500' : 'bg-gray-300'}"></span>
              <h3 class="text-[14px] font-semibold text-gray-900">{t[provider.name] || provider.name}</h3>
            </div>
            {#if editingProvider === provider.name}
              <div class="flex gap-2">
                <button 
                  class="px-3 py-1 text-[12px] font-medium text-gray-600 bg-gray-100 rounded-md hover:bg-gray-200 transition-colors cursor-pointer"
                  onclick={cancelEditProvider}
                >{t.cancel}</button>
                <button 
                  class="px-3 py-1 text-[12px] font-medium text-white bg-emerald-500 rounded-md hover:bg-emerald-600 transition-colors disabled:opacity-50 inline-flex items-center gap-1.5 cursor-pointer"
                  onclick={() => showSaveConfirm(provider.name)}
                  disabled={credentialsSaving[provider.name]}
                >
                  {#if credentialsSaving[provider.name]}
                    <svg class="animate-spin h-3.5 w-3.5" viewBox="0 0 24 24" fill="none"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path></svg>
                    {t.saving}
                  {:else}
                    {t.save}
                  {/if}
                </button>
              </div>
            {:else}
              <button 
                class="px-3 py-1 text-[12px] font-medium text-gray-600 bg-gray-100 rounded-md hover:bg-gray-200 hover:text-gray-900 transition-colors cursor-pointer"
                onclick={() => startEditProvider(provider)}
              >{t.edit}</button>
            {/if}
          </div>
          <div class="p-5 space-y-3">
            {#each getOrderedFields(provider.name, provider.fields) as [key, value]}
              <div>
                <label for="field-{provider.name}-{key}" class="block text-[11px] font-medium text-gray-500 mb-1">
                  {getFieldLabel(key)}
                  {#if provider.hasSecrets && provider.hasSecrets[key]}
                    <svg class="inline-block ml-1 w-3 h-3 text-amber-500 -mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M16.5 10.5V6.75a4.5 4.5 0 10-9 0v3.75m-.75 11.25h10.5a2.25 2.25 0 002.25-2.25v-6.75a2.25 2.25 0 00-2.25-2.25H6.75a2.25 2.25 0 00-2.25 2.25v6.75a2.25 2.25 0 002.25 2.25z" /></svg>
                  {/if}
                </label>
                {#if editingProvider === provider.name}
                  {#if isSecretField(key)}
                    <input 
                      id="field-{provider.name}-{key}"
                      type="password"
                      placeholder={t.enterNew}
                      class="w-full h-9 px-3 text-[12px] bg-gray-50 border-0 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow font-mono"
                      bind:value={editFields[key]}
                    />
                  {:else}
                    <input 
                      id="field-{provider.name}-{key}"
                      type="text"
                      placeholder={value || t.notSet}
                      class="w-full h-9 px-3 text-[12px] bg-gray-50 border-0 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow font-mono"
                      bind:value={editFields[key]}
                    />
                  {/if}
                {:else}
                  <div class="h-9 px-3 flex items-center text-[12px] bg-gray-50 rounded-lg font-mono {value ? 'text-gray-900' : 'text-gray-400'}">
                    {value || t.notSet}
                  </div>
                {/if}
              </div>
            {/each}
          </div>
        </div>
      {:else}
        <div class="col-span-full py-12 text-center">
          <p class="text-[13px] text-gray-400">{t.noMatchProvider || '没有匹配的厂商'}</p>
        </div>
      {/each}
    </div>
  {:else}
    <!-- Empty state -->
    <div class="py-16 text-center">
      <svg class="w-10 h-10 mx-auto mb-3 text-gray-300" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1">
        <path stroke-linecap="round" stroke-linejoin="round" d="M15.75 5.25a3 3 0 013 3m3 0a6 6 0 01-7.029 5.912c-.563-.097-1.159.026-1.563.43L10.5 17.25H8.25v2.25H6v2.25H2.25v-2.818c0-.597.237-1.17.659-1.591l6.499-6.499c.404-.404.527-1 .43-1.563A6 6 0 1121.75 8.25z" />
      </svg>
      <p class="text-[13px] text-gray-500">{t.clickLoad}</p>
    </div>
  {/if}

  {#if error}
    <div class="flex items-center gap-3 px-4 py-3 bg-red-50 border border-red-100 rounded-lg">
      <svg class="w-4 h-4 text-red-500 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
        <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z" />
      </svg>
      <span class="text-[13px] text-red-700 flex-1">{error}</span>
      <button class="text-red-400 hover:text-red-600 cursor-pointer" onclick={() => error = ''} aria-label={t.closeError}>
        <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
        </svg>
      </button>
    </div>
  {/if}
</div>

<!-- Save Credentials Confirmation Modal -->
<Modal show={saveConfirm.show} onclose={cancelSave} class="overflow-visible">
    <div class="bg-white rounded-xl border border-gray-200 shadow-xl max-w-sm w-full mx-4 overflow-hidden" onclick={(e) => e.stopPropagation()}>
      <div class="px-6 py-5">
        <div class="flex items-center gap-3 mb-3">
          <div class="w-10 h-10 rounded-full bg-emerald-100 flex items-center justify-center">
            <svg class="w-5 h-5 text-emerald-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
            </svg>
          </div>
          <div>
            <h3 class="text-[15px] font-semibold text-gray-900">{t.confirmSave || '确认保存'}</h3>
            <p class="text-[13px] text-gray-500">{t.saveWarning || '凭据将被保存到配置文件'}</p>
          </div>
        </div>
        <p class="text-[13px] text-gray-600">
          {t.confirmSaveCredentials || '确认保存'} <span class="font-medium text-gray-900">"{t[saveConfirm.providerName] || saveConfirm.providerName}"</span> {t.credentials || '的凭据'}?
        </p>
      </div>
      <div class="px-6 py-4 bg-gray-50 flex justify-end gap-2">
        <button 
          class="px-4 py-2 text-[13px] font-medium text-gray-700 bg-white border border-gray-100 rounded-lg hover:bg-gray-50 transition-colors"
          onclick={cancelSave}
        >{t.cancel}</button>
        <button 
          class="px-4 py-2 text-[13px] font-medium text-white bg-emerald-600 rounded-lg hover:bg-emerald-700 transition-colors"
          onclick={confirmSave}
        >{t.save}</button>
      </div>
    </div>
</Modal>
