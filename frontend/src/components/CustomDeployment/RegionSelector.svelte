<script>
  import { onMount } from 'svelte';
  import { GetProviderRegions } from '../../../wailsjs/go/main/App.js';

  // Props
  let { 
    t, 
    provider = '', 
    value = '', 
    onChange = () => {},
    disabled = false
  } = $props();

  // State
  let regions = $state([]);
  let isLoading = $state(false);
  let error = $state('');
  let searchQuery = $state('');
  
  // Track previous provider to avoid unnecessary reloads
  let prevProvider = $state('');

  // Load regions when provider changes
  $effect(() => {
    // Only reload if provider actually changed
    if (provider && provider !== prevProvider) {
      prevProvider = provider;
      loadRegions(provider);
    } else if (!provider) {
      regions = [];
      error = '';
      prevProvider = '';
    }
  });

  async function loadRegions(providerCode) {
    if (!providerCode) {
      regions = [];
      return;
    }

    isLoading = true;
    error = '';
    
    try {
      const result = await GetProviderRegions(providerCode);
      regions = result || [];
    } catch (e) {
      error = e.message || String(e);
      regions = [];
    } finally {
      isLoading = false;
    }
  }

  function handleChange(event) {
    const newValue = event.currentTarget.value;
    onChange(newValue);
  }

  // Filtered regions based on search query
  let filteredRegions = $derived(() => {
    if (!searchQuery.trim()) {
      return regions;
    }
    
    const query = searchQuery.toLowerCase();
    return regions.filter(region => 
      region.code.toLowerCase().includes(query) ||
      region.name.toLowerCase().includes(query)
    );
  });

  // Get display text for region
  function getRegionDisplay(region) {
    return `${region.name} (${region.code})`;
  }
</script>

<div class="region-selector">
  <label for="region-select" class="block text-[12px] font-medium text-gray-700 mb-1.5">
    {t.region || '地域'}
    <span class="text-red-500 ml-1">*</span>
  </label>

  {#if !provider}
    <div class="w-full h-10 px-3 text-[13px] bg-gray-100 border-0 rounded-lg text-gray-400 flex items-center">
      {t.selectProviderFirst || '请先选择云厂商'}
    </div>
  {:else if isLoading}
    <div class="w-full h-10 px-3 text-[13px] bg-gray-50 border-0 rounded-lg text-gray-500 flex items-center gap-2">
      <div class="w-4 h-4 border-2 border-gray-300 border-t-gray-600 rounded-full animate-spin"></div>
      {t.loading || '加载中...'}
    </div>
  {:else if error}
    <div class="w-full px-3 py-2 text-[12px] bg-red-50 border border-red-100 rounded-lg text-red-700">
      {error}
    </div>
  {:else}
    <!-- Search input for filtering -->
    {#if regions.length > 10}
      <input
        type="text"
        placeholder={t.searchRegion || '搜索地域...'}
        class="w-full h-9 px-3 mb-2 text-[12px] bg-gray-50 border border-gray-100 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow"
        bind:value={searchQuery}
      />
    {/if}

    <select
      id="region-select"
      class="w-full h-10 px-3 text-[13px] bg-gray-50 border-0 rounded-lg text-gray-900 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow appearance-none cursor-pointer"
      disabled={disabled || regions.length === 0}
      {value}
      onchange={handleChange}
    >
      <option value="" disabled>
        {regions.length === 0 
          ? (t.noRegionsAvailable || '暂无可用地域') 
          : (t.selectRegion || '请选择地域...')}
      </option>
      {#each filteredRegions() as region}
        <option value={region.code}>
          {getRegionDisplay(region)}
        </option>
      {/each}
    </select>

    <!-- Custom dropdown arrow -->
    <div class="pointer-events-none absolute right-0 flex items-center px-3 text-gray-400" style="margin-top: -34px;">
      <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
        <path stroke-linecap="round" stroke-linejoin="round" d="M19 9l-7 7-7-7" />
      </svg>
    </div>

    {#if filteredRegions().length === 0 && searchQuery}
      <p class="text-[11px] text-gray-500 mt-1">
        {t.noMatchingRegions || '未找到匹配的地域'}
      </p>
    {/if}

    {#if regions.length > 0}
      <p class="text-[11px] text-gray-500 mt-1">
        {t.regionsAvailable || '可用地域'}: {filteredRegions().length}
      </p>
    {/if}
  {/if}
</div>

<style>
  .region-selector {
    position: relative;
  }

  select {
    background-image: none;
    padding-right: 2.5rem;
  }

  select:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }
</style>
