<script>
  import { onMount } from 'svelte';
  import { GetInstanceTypes } from '../../../wailsjs/go/main/App.js';

  // Props
  let { 
    t, 
    provider = '', 
    region = '', 
    value = '', 
    onChange = () => {},
    disabled = false
  } = $props();

  // State
  let instanceTypes = $state([]);
  let isLoading = $state(false);
  let error = $state('');
  let searchQuery = $state('');
  let selectedFamily = $state('all'); // Filter by instance family
  
  // Track previous values to avoid unnecessary reloads
  let prevProvider = $state('');
  let prevRegion = $state('');

  // Load instance types when provider or region changes
  $effect(() => {
    // Only reload if provider or region actually changed
    if (provider && region && (provider !== prevProvider || region !== prevRegion)) {
      prevProvider = provider;
      prevRegion = region;
      loadInstanceTypes(provider, region);
    } else if (!provider || !region) {
      instanceTypes = [];
      error = '';
      prevProvider = '';
      prevRegion = '';
    }
  });

  async function loadInstanceTypes(providerCode, regionCode) {
    if (!providerCode || !regionCode) {
      instanceTypes = [];
      return;
    }

    isLoading = true;
    error = '';
    
    try {
      const result = await GetInstanceTypes(providerCode, regionCode);
      instanceTypes = result || [];
    } catch (e) {
      error = e.message || String(e);
      instanceTypes = [];
    } finally {
      isLoading = false;
    }
  }

  function handleChange(event) {
    const newValue = event.currentTarget.value;
    onChange(newValue);
  }

  // Extract instance family from code (e.g., "ecs.t6" from "ecs.t6-c1m1.large")
  function getInstanceFamily(code) {
    const parts = code.split('.');
    if (parts.length >= 2) {
      return parts.slice(0, 2).join('.');
    }
    return code.split('-')[0] || 'other';
  }

  // Get unique instance families
  let instanceFamilies = $derived(() => {
    const families = new Set();
    instanceTypes.forEach(type => {
      families.add(getInstanceFamily(type.code));
    });
    return Array.from(families).sort();
  });

  // Filtered instance types based on search query and family filter
  let filteredInstanceTypes = $derived(() => {
    let result = instanceTypes;

    // Filter by family
    if (selectedFamily !== 'all') {
      result = result.filter(type => getInstanceFamily(type.code) === selectedFamily);
    }

    // Filter by search query
    if (searchQuery.trim()) {
      const query = searchQuery.toLowerCase();
      result = result.filter(type => 
        type.code.toLowerCase().includes(query) ||
        (type.name && type.name.toLowerCase().includes(query)) ||
        (type.description && type.description.toLowerCase().includes(query))
      );
    }

    return result;
  });

  // Format memory size (MB to GB)
  function formatMemory(memory) {
    if (memory >= 1024) {
      return `${(memory / 1024).toFixed(0)} GB`;
    }
    return `${memory} MB`;
  }

  // Format price
  function formatPrice(price) {
    if (!price || price === 0) {
      return '';
    }
    return `¥${price.toFixed(4)}${t.perHour || '/小时'}`;
  }

  // Get display text for instance type
  function getInstanceTypeDisplay(type) {
    let display = type.code;
    
    if (type.cpu || type.memory) {
      const specs = [];
      if (type.cpu) specs.push(`${type.cpu} vCPU`);
      if (type.memory) specs.push(formatMemory(type.memory));
      display += ` (${specs.join(', ')})`;
    }
    
    if (type.price && type.price > 0) {
      display += ` - ${formatPrice(type.price)}`;
    }
    
    return display;
  }
</script>

<div class="instance-type-selector">
  <label for="instance-type-select" class="block text-[12px] font-medium text-gray-700 mb-1.5">
    {t.instanceType || '实例规格'}
    <span class="text-red-500 ml-1">*</span>
  </label>

  {#if !provider || !region}
    <div class="w-full h-10 px-3 text-[13px] bg-gray-100 border-0 rounded-lg text-gray-400 flex items-center">
      {t.selectProviderAndRegionFirst || '请先选择云厂商和地域'}
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
    <!-- Family filter (if more than one family) -->
    {#if instanceFamilies().length > 1}
      <div class="flex gap-2 mb-2 flex-wrap">
        <button
          class="px-3 py-1.5 text-[11px] font-medium rounded-lg transition-colors {selectedFamily === 'all' ? 'bg-gray-900 text-white' : 'bg-gray-100 text-gray-700 hover:bg-gray-200'}"
          onclick={() => selectedFamily = 'all'}
        >
          {t.allFamilies || '全部规格族'}
        </button>
        {#each instanceFamilies() as family}
          <button
            class="px-3 py-1.5 text-[11px] font-medium rounded-lg transition-colors {selectedFamily === family ? 'bg-gray-900 text-white' : 'bg-gray-100 text-gray-700 hover:bg-gray-200'}"
            onclick={() => selectedFamily = family}
          >
            {family}
          </button>
        {/each}
      </div>
    {/if}

    <!-- Search input for filtering -->
    {#if instanceTypes.length > 10}
      <input
        type="text"
        placeholder={t.searchInstanceType || '搜索实例规格...'}
        class="w-full h-9 px-3 mb-2 text-[12px] bg-gray-50 border border-gray-100 rounded-lg text-gray-900 placeholder-gray-400 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow"
        bind:value={searchQuery}
      />
    {/if}

    <select
      id="instance-type-select"
      class="w-full h-10 px-3 text-[13px] bg-gray-50 border-0 rounded-lg text-gray-900 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow appearance-none cursor-pointer"
      disabled={disabled || instanceTypes.length === 0}
      {value}
      onchange={handleChange}
    >
      <option value="" disabled>
        {instanceTypes.length === 0 
          ? (t.noInstanceTypesAvailable || '暂无可用实例规格') 
          : (t.selectInstanceType || '请选择实例规格...')}
      </option>
      <!-- Show the current value even if not in the loaded list yet (for loaded configs) -->
      {#if value && !instanceTypes.find(t => t.code === value)}
        <option value={value} selected>
          {value}
        </option>
      {/if}
      {#each filteredInstanceTypes() as type}
        <option value={type.code}>
          {getInstanceTypeDisplay(type)}
        </option>
      {/each}
    </select>

    <!-- Custom dropdown arrow -->
    <div class="pointer-events-none absolute right-0 flex items-center px-3 text-gray-400" style="margin-top: -34px;">
      <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
        <path stroke-linecap="round" stroke-linejoin="round" d="M19 9l-7 7-7-7" />
      </svg>
    </div>

    {#if filteredInstanceTypes().length === 0 && (searchQuery || selectedFamily !== 'all')}
      <p class="text-[11px] text-gray-500 mt-1">
        {t.noMatchingInstanceTypes || '未找到匹配的实例规格'}
      </p>
    {/if}

    {#if instanceTypes.length > 0}
      <p class="text-[11px] text-gray-500 mt-1">
        {t.instanceTypesAvailable || '可用规格'}: {filteredInstanceTypes().length} / {instanceTypes.length}
      </p>
    {/if}

    <!-- Selected instance type details -->
    {#if value}
      {@const selectedType = instanceTypes.find(t => t.code === value)}
      {#if selectedType}
        <div class="mt-3 p-3 bg-blue-50 border border-blue-100 rounded-lg">
          <h4 class="text-[12px] font-medium text-blue-900 mb-2">
            {t.selectedInstanceType || '已选实例规格'}
          </h4>
          <div class="grid grid-cols-2 gap-2 text-[11px]">
            <div>
              <span class="text-blue-700">{t.cpu || 'CPU'}:</span>
              <span class="text-blue-900 ml-1">{selectedType.cpu} vCPU</span>
            </div>
            <div>
              <span class="text-blue-700">{t.memory || '内存'}:</span>
              <span class="text-blue-900 ml-1">{formatMemory(selectedType.memory)}</span>
            </div>
            {#if selectedType.price && selectedType.price > 0}
              <div class="col-span-2">
                <span class="text-blue-700">{t.price || '价格'}:</span>
                <span class="text-blue-900 ml-1">{formatPrice(selectedType.price)}</span>
              </div>
            {/if}
            {#if selectedType.description}
              <div class="col-span-2">
                <span class="text-blue-700">{t.description || '描述'}:</span>
                <span class="text-blue-900 ml-1">{selectedType.description}</span>
              </div>
            {/if}
          </div>
        </div>
      {:else}
        <!-- Show basic info when instance type data is not loaded yet -->
        <div class="mt-3 p-3 bg-gray-50 border border-gray-100 rounded-lg">
          <h4 class="text-[12px] font-medium text-gray-700 mb-1">
            {t.selectedInstanceType || '已选实例规格'}
          </h4>
          <p class="text-[11px] text-gray-600">{value}</p>
          {#if isLoading}
            <p class="text-[10px] text-gray-500 mt-1">
              {t.loadingDetails || '正在加载详细信息...'}
            </p>
          {/if}
        </div>
      {/if}
    {/if}
  {/if}
</div>

<style>
  .instance-type-selector {
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
