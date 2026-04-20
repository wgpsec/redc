<script>
  // Props
  let { 
    t, 
    value = '', 
    onChange = () => {},
    disabled = false
  } = $props();

  // Supported cloud providers
  const providers = $derived([
    { code: 'alicloud', name: t.providerAlicloudShort || '阿里云', nameEn: 'Alibaba Cloud' },
    { code: 'tencentcloud', name: t.providerTencentcloudShort || '腾讯云', nameEn: 'Tencent Cloud' },
    { code: 'aws', name: 'AWS', nameEn: 'Amazon Web Services' },
    { code: 'volcengine', name: t.providerVolcengineShort || '火山引擎', nameEn: 'Volcengine' },
    { code: 'huaweicloud', name: t.providerHuaweicloudShort || '华为云', nameEn: 'Huawei Cloud' }
  ]);

  function handleChange(event) {
    const newValue = event.currentTarget.value;
    onChange(newValue);
  }

  // Get display name for provider
  function getProviderDisplay(provider) {
    return `${provider.name} (${provider.nameEn})`;
  }
</script>

<div class="provider-selector">
  <label for="provider-select" class="block text-[12px] font-medium text-gray-700 mb-1.5">
    {t.provider || '云厂商'}
    <span class="text-red-500 ml-1">*</span>
  </label>
  
  <select
    id="provider-select"
    class="w-full h-10 px-3 text-[13px] bg-gray-50 border-0 rounded-lg text-gray-900 focus:ring-2 focus:ring-gray-900 focus:ring-offset-1 transition-shadow appearance-none cursor-pointer"
    {disabled}
    {value}
    onchange={handleChange}
  >
    <option value="" disabled>
      {t.selectProvider || '请选择云厂商...'}
    </option>
    {#each providers as provider}
      <option value={provider.code}>
        {getProviderDisplay(provider)}
      </option>
    {/each}
  </select>

  <!-- Custom dropdown arrow -->
  <div class="pointer-events-none absolute inset-y-0 right-0 flex items-center px-3 text-gray-400" style="margin-top: 26px;">
    <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
      <path stroke-linecap="round" stroke-linejoin="round" d="M19 9l-7 7-7-7" />
    </svg>
  </div>
</div>

<style>
  .provider-selector {
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
