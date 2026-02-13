<script>

  import { onMount } from 'svelte';
  import { GetProvidersConfig, SaveProvidersConfig, ListProfiles, GetActiveProfile, SetActiveProfile, CreateProfile, UpdateProfile, DeleteProfile, UpdateProfileAIConfig } from '../../../wailsjs/go/main/App.js';

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
  let profileError = $state('');
  let error = $state('');
  let saveConfirm = $state({ show: false, providerName: '' });

  // AI Configuration state
  let aiConfig = $state({
    provider: 'openai',
    apiKey: '',
    baseUrl: '',
    model: ''
  });
  let aiConfigSaving = $state(false);
  let aiConfigSaved = $state(false);
  let showApiKey = $state(false);

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
            model: active.aiConfig.model || aiProviderPresets[active.aiConfig.provider || 'openai'].defaultModel
          };
        } else {
          const preset = aiProviderPresets['openai'];
          aiConfig = {
            provider: 'openai',
            apiKey: '',
            baseUrl: preset.baseUrl,
            model: preset.defaultModel
          };
        }
        customConfigPath = profileForm.configPath;
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
      await UpdateProfileAIConfig(activeProfileId, aiConfig.provider, aiConfig.apiKey, aiConfig.baseUrl, aiConfig.model);
      aiConfigSaved = true;
      setTimeout(() => { aiConfigSaved = false; }, 2000);
    } catch (e) {
      error = e.message || String(e);
    } finally {
      aiConfigSaving = false;
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
      region: 'Region',
      credentials: '凭据 JSON',
      project: '项目 ID',
      clientId: 'Client ID',
      clientSecret: 'Client Secret',
      subscriptionId: 'Subscription ID',
      tenantId: 'Tenant ID',
      user: '用户 OCID',
      tenancy: 'Tenancy OCID',
      fingerprint: '指纹',
      keyFile: '私钥文件路径',
      email: '邮箱',
      apiKey: 'API Key',
    };
    return labels[key] || key;
  }

  function isSecretField(key) {
    const secrets = ['accessKey', 'secretKey', 'secretId', 'credentials', 'clientId', 'clientSecret', 'subscriptionId', 'tenantId', 'user', 'tenancy', 'fingerprint', 'apiKey'];
    return secrets.includes(key);
  }

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
      <div class="text-[12px] text-gray-500">
        {t.activeProfile}: <span class="font-medium text-gray-700">{profiles.find(p => p.id === activeProfileId)?.name || '-'}</span>
      </div>
    </div>

    <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
      <div>
        <label class="block text-[12px] font-medium text-gray-500 mb-1.5">{t.profile}</label>
        <select
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
        <label class="block text-[12px] font-medium text-gray-500 mb-1.5">{t.profileName}</label>
        <input
          type="text"
          class="w-full h-10 px-3 text-[13px] bg-gray-50 border-0 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow"
          bind:value={profileForm.name}
        />
      </div>
      <div class="md:col-span-2">
        <label class="block text-[12px] font-medium text-gray-500 mb-1.5">{t.configPath}</label>
        <input
          type="text"
          placeholder={t.defaultPath}
          class="w-full h-10 px-3 text-[13px] bg-gray-50 border-0 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow font-mono"
          bind:value={profileForm.configPath}
          oninput={() => { customConfigPath = profileForm.configPath; }}
        />
      </div>
      <div class="md:col-span-2">
        <label class="block text-[12px] font-medium text-gray-500 mb-1.5">{t.templateDir}</label>
        <input
          type="text"
          placeholder={t.defaultPath}
          class="w-full h-10 px-3 text-[13px] bg-gray-50 border-0 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow font-mono"
          bind:value={profileForm.templateDir}
        />
      </div>
    </div>

    {#if profileError}
      <div class="mt-3 text-[12px] text-red-600">{profileError}</div>
    {/if}

    <div class="mt-4 flex flex-wrap gap-2">
      <button
        class="h-9 px-4 bg-gray-900 text-white text-[12px] font-medium rounded-lg hover:bg-gray-800 transition-colors disabled:opacity-50"
        onclick={handleCreateProfile}
        disabled={profileSaving}
      >
        {t.createProfile}
      </button>
      <button
        class="h-9 px-4 bg-emerald-500 text-white text-[12px] font-medium rounded-lg hover:bg-emerald-600 transition-colors disabled:opacity-50"
        onclick={handleSaveProfile}
        disabled={profileSaving || !activeProfileId}
      >
        {t.saveProfile}
      </button>
      <button
        class="h-9 px-4 bg-red-50 text-red-600 text-[12px] font-medium rounded-lg hover:bg-red-100 transition-colors disabled:opacity-50"
        onclick={handleDeleteProfile}
        disabled={profileSaving || !activeProfileId}
      >
        {t.deleteProfile}
      </button>
      <button
        class="h-9 px-4 bg-gray-100 text-gray-700 text-[12px] font-medium rounded-lg hover:bg-gray-200 transition-colors disabled:opacity-50"
        onclick={loadProvidersConfig}
        disabled={credentialsLoading}
      >
        {credentialsLoading ? t.loading : t.loadConfig}
      </button>
    </div>

    {#if providersConfig.configPath}
      <div class="mt-3 text-[12px] text-gray-500">
        {t.currentConfig}: <span class="font-mono">{providersConfig.configPath}</span>
      </div>
    {/if}
    <div class="mt-2 text-[12px] text-gray-500">
      {t.profileCredentialsFrom}: <span class="font-mono">{profileForm.configPath || providersConfig.configPath || '-'}</span>
    </div>
    <div class="text-[12px] text-gray-500">
      {t.profileTemplateFrom}: <span class="font-mono">{profileForm.templateDir || '-'}</span>
    </div>
    <div class="text-[11px] text-gray-400 mt-1">
      {t.profileSwitchHint}
    </div>
  </div>

  <!-- Security Notice -->
  <div class="flex items-start gap-3 px-4 py-3 bg-amber-50 border border-amber-100 rounded-lg">
    <svg class="w-4 h-4 text-amber-500 flex-shrink-0 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
      <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126zM12 15.75h.007v.008H12v-.008z" />
    </svg>
    <div class="text-[12px] text-amber-800">
      <strong>{t.securityTip}</strong>{t.securityInfo}
    </div>
  </div>

  <!-- AI Configuration Card -->
  <div class="bg-white rounded-xl border border-gray-100 overflow-hidden">
    <div class="px-5 py-4 border-b border-gray-100 flex items-center justify-between">
      <div class="flex items-center gap-2">
        <svg class="w-5 h-5 text-purple-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
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
          class="px-3 py-1 text-[12px] font-medium text-white bg-purple-500 rounded-md hover:bg-purple-600 transition-colors disabled:opacity-50"
          onclick={handleSaveAIConfig}
          disabled={aiConfigSaving || !activeProfileId}
        >
          {aiConfigSaving ? (t.saving || 'Saving...') : (t.save || 'Save')}
        </button>
      </div>
    </div>
    <div class="p-5">
      <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div>
          <label class="block text-[11px] font-medium text-gray-500 mb-1">{t.aiProvider || 'Provider'}</label>
          <select 
            class="w-full h-9 px-3 text-[12px] bg-gray-50 border-0 rounded-lg text-gray-900 focus:ring-2 focus:ring-purple-500 focus:ring-offset-1 transition-shadow"
            bind:value={aiConfig.provider}
            onchange={handleProviderChange}
          >
            {#each Object.entries(aiProviderPresets) as [key, preset]}
              <option value={key}>{preset.name}</option>
            {/each}
          </select>
        </div>
        <div>
          <label class="block text-[11px] font-medium text-gray-500 mb-1">{t.aiModel || 'Model'}</label>
          <input 
            type="text"
            placeholder={aiProviderPresets[aiConfig.provider]?.placeholder || 'Enter model name'}
            class="w-full h-9 px-3 text-[12px] bg-gray-50 border-0 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-purple-500 focus:ring-offset-1 transition-shadow font-mono"
            bind:value={aiConfig.model}
          />
          <p class="text-[10px] text-gray-400 mt-1">{t.aiModelHint || '支持任意兼容的模型名称'}</p>
        </div>
        <div class="md:col-span-2">
          <label class="block text-[11px] font-medium text-gray-500 mb-1">
            {t.aiApiKey || 'API Key'}
            <span class="ml-1 text-amber-500">🔒</span>
          </label>
          <div class="relative">
            <input 
              type={showApiKey ? 'text' : 'password'}
              placeholder={t.aiApiKeyPlaceholder || 'Enter your API key'}
              class="w-full h-9 px-3 pr-10 text-[12px] bg-gray-50 border-0 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-purple-500 focus:ring-offset-1 transition-shadow font-mono"
              bind:value={aiConfig.apiKey}
            />
            <button 
              type="button"
              class="absolute right-2 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600"
              onclick={() => showApiKey = !showApiKey}
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
          <label class="block text-[11px] font-medium text-gray-500 mb-1">{t.aiBaseUrl || 'Base URL'}</label>
          <input 
            type="text"
            placeholder={aiProviderPresets[aiConfig.provider]?.baseUrl || ''}
            class="w-full h-9 px-3 text-[12px] bg-gray-50 border-0 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-purple-500 focus:ring-offset-1 transition-shadow font-mono"
            bind:value={aiConfig.baseUrl}
          />
          <p class="text-[10px] text-gray-400 mt-1">{t.aiBaseUrlHint || 'Optional: Override the default API endpoint'}</p>
        </div>
      </div>
      {#if !activeProfileId}
        <p class="text-[11px] text-amber-600 mt-3">{t.aiConfigProfileHint || 'Please select a profile first to configure AI settings'}</p>
      {/if}
    </div>
  </div>

  {#if credentialsLoading}
    <div class="flex items-center justify-center h-32">
      <div class="w-6 h-6 border-2 border-gray-200 border-t-gray-900 rounded-full animate-spin"></div>
    </div>
  {:else}
    <!-- Provider Cards -->
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      {#each providersConfig.providers || [] as provider}
        <div class="bg-white rounded-xl border border-gray-100 overflow-hidden">
          <div class="px-5 py-4 border-b border-gray-100 flex items-center justify-between">
            <h3 class="text-[14px] font-semibold text-gray-900">{provider.name}</h3>
            {#if editingProvider === provider.name}
              <div class="flex gap-2">
                <button 
                  class="px-3 py-1 text-[12px] font-medium text-gray-600 bg-gray-100 rounded-md hover:bg-gray-200 transition-colors"
                  onclick={cancelEditProvider}
                >{t.cancel}</button>
                <button 
                  class="px-3 py-1 text-[12px] font-medium text-white bg-emerald-500 rounded-md hover:bg-emerald-600 transition-colors disabled:opacity-50"
                  onclick={() => showSaveConfirm(provider.name)}
                  disabled={credentialsSaving[provider.name]}
                >
                  {credentialsSaving[provider.name] ? t.saving : t.save}
                </button>
              </div>
            {:else}
              <button 
                class="px-3 py-1 text-[12px] font-medium text-blue-600 bg-blue-50 rounded-md hover:bg-blue-100 transition-colors"
                onclick={() => startEditProvider(provider)}
              >{t.edit}</button>
            {/if}
          </div>
          <div class="p-5 space-y-3">
            {#each Object.entries(provider.fields) as [key, value]}
              <div>
                <label class="block text-[11px] font-medium text-gray-500 mb-1">
                  {getFieldLabel(key)}
                  {#if provider.hasSecrets && provider.hasSecrets[key]}
                    <span class="ml-1 text-amber-500">🔒</span>
                  {/if}
                </label>
                {#if editingProvider === provider.name}
                  {#if isSecretField(key)}
                    <input 
                      type="password"
                      placeholder={t.enterNew}
                      class="w-full h-9 px-3 text-[12px] bg-gray-50 border-0 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow font-mono"
                      bind:value={editFields[key]}
                    />
                  {:else}
                    <input 
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
        <div class="col-span-full py-16 text-center">
          <svg class="w-10 h-10 mx-auto mb-3 text-gray-300" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1">
            <path stroke-linecap="round" stroke-linejoin="round" d="M15.75 5.25a3 3 0 013 3m3 0a6 6 0 01-7.029 5.912c-.563-.097-1.159.026-1.563.43L10.5 17.25H8.25v2.25H6v2.25H2.25v-2.818c0-.597.237-1.17.659-1.591l6.499-6.499c.404-.404.527-1 .43-1.563A6 6 0 1121.75 8.25z" />
          </svg>
          <p class="text-[13px] text-gray-400">{t.clickLoad}</p>
        </div>
      {/each}
    </div>
  {/if}

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
</div>

<!-- Save Credentials Confirmation Modal -->
{#if saveConfirm.show}
  <div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onclick={cancelSave}>
    <div class="bg-white rounded-xl shadow-xl max-w-sm w-full mx-4 overflow-hidden" onclick={(e) => e.stopPropagation()}>
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
          {t.confirmSaveCredentials || '确认保存'} <span class="font-medium text-gray-900">"{saveConfirm.providerName}"</span> {t.credentials || '的凭据'}?
        </p>
      </div>
      <div class="px-6 py-4 bg-gray-50 flex justify-end gap-2">
        <button 
          class="px-4 py-2 text-[13px] font-medium text-gray-700 bg-white border border-gray-200 rounded-lg hover:bg-gray-50 transition-colors"
          onclick={cancelSave}
        >{t.cancel}</button>
        <button 
          class="px-4 py-2 text-[13px] font-medium text-white bg-emerald-600 rounded-lg hover:bg-emerald-700 transition-colors"
          onclick={confirmSave}
        >{t.save}</button>
      </div>
    </div>
  </div>
{/if}
