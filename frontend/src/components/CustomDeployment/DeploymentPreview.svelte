<script lang="ts">
  let { 
    t, 
    config = {},
    template = null,
    costEstimate = null,
    validation = { valid: true, errors: [], warnings: [] },
    onDeploy = () => {},
    isDeploying = false
  } = $props();

  // Currency state
  let selectedCurrency = $state('CNY');
  const supportedCurrencies = ['CNY', 'USD', 'EUR'];

  // Exchange rates (simplified - in production, these should come from an API)
  const exchangeRates = {
    'CNY': 1.0,
    'USD': 0.14,  // 1 CNY ≈ 0.14 USD
    'EUR': 0.13   // 1 CNY ≈ 0.13 EUR
  };

  // Get provider display name
  function getProviderName(code) {
    const providers = {
      'alicloud': t.providerAlicloud || '阿里云 (Alibaba Cloud)',
      'tencentcloud': t.providerTencentcloud || '腾讯云 (Tencent Cloud)',
      'aws': t.providerAWS || 'AWS (Amazon Web Services)',
      'volcengine': t.providerVolcengine || '火山引擎 (Volcengine)',
      'huaweicloud': t.providerHuaweicloud || '华为云 (Huawei Cloud)'
    };
    return providers[code] || code;
  }

  // Convert cost to selected currency
  function convertCost(amount, fromCurrency, toCurrency) {
    if (!amount || fromCurrency === toCurrency) return amount;
    
    // Convert to CNY first (base currency)
    const amountInCNY = amount / (exchangeRates[fromCurrency] || 1);
    
    // Then convert to target currency
    return amountInCNY * (exchangeRates[toCurrency] || 1);
  }

  // Format cost
  function formatCost(amount, currency = 'CNY') {
    if (amount === null || amount === undefined) return '-';
    
    const symbols = {
      'CNY': '¥',
      'USD': '$',
      'EUR': '€'
    };
    
    const symbol = symbols[currency] || currency;
    return `${symbol}${amount.toFixed(2)}`;
  }

  // Get displayed cost in selected currency
  let displayedCost = $derived(() => {
    if (!costEstimate) return null;
    
    const originalCurrency = costEstimate.currency || 'CNY';
    const convertedAmount = convertCost(
      costEstimate.monthly_cost, 
      originalCurrency, 
      selectedCurrency
    );
    
    return {
      amount: convertedAmount,
      currency: selectedCurrency
    };
  });

  // Get displayed cost details in selected currency
  let displayedDetails = $derived(() => {
    if (!costEstimate || !costEstimate.details) return {};
    
    const originalCurrency = costEstimate.currency || 'CNY';
    const converted = {};
    
    for (const [key, value] of Object.entries(costEstimate.details)) {
      converted[key] = convertCost(value, originalCurrency, selectedCurrency);
    }
    
    return converted;
  });

  // Check if config is complete
  let isConfigComplete = $derived(() => {
    return config.name && 
           config.provider && 
           config.region && 
           config.instanceType &&
           validation.valid;
  });
</script>

<div class="bg-white rounded-xl border border-gray-100 p-5">
  <h2 class="text-[15px] font-semibold text-gray-900 mb-4">
    {t.deploymentPreview || '部署预览'}
  </h2>

  {#if !isConfigComplete()}
    <div class="flex flex-col items-center justify-center py-12 text-center">
      <svg class="w-12 h-12 text-gray-300 mb-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
        <path stroke-linecap="round" stroke-linejoin="round" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
      </svg>
      <p class="text-[13px] text-gray-500">
        {t.completeConfigToPreview || '请完成配置以查看预览'}
      </p>
    </div>
  {:else}
    <div class="space-y-4">
      <!-- Configuration Summary -->
      <div class="bg-gray-50 rounded-lg p-4">
        <h3 class="text-[12px] font-medium text-gray-700 mb-3">
          {t.configurationSummary || '配置摘要'}
        </h3>
        
        <div class="grid grid-cols-2 gap-3">
          <!-- Deployment Name -->
          <div>
            <p class="text-[10px] text-gray-500 uppercase tracking-wide mb-1">
              {t.deploymentName || '部署名称'}
            </p>
            <p class="text-[12px] text-gray-900 font-medium">
              {config.name || '-'}
            </p>
          </div>

          <!-- Template -->
          {#if template}
            <div>
              <p class="text-[10px] text-gray-500 uppercase tracking-wide mb-1">
                {t.template || '模板'}
              </p>
              <p class="text-[12px] text-gray-900 font-medium">
                {template.name || '-'}
              </p>
            </div>
          {/if}

          <!-- Provider -->
          <div>
            <p class="text-[10px] text-gray-500 uppercase tracking-wide mb-1">
              {t.provider || '云厂商'}
            </p>
            <p class="text-[12px] text-gray-900 font-medium">
              {getProviderName(config.provider)}
            </p>
          </div>

          <!-- Region -->
          <div>
            <p class="text-[10px] text-gray-500 uppercase tracking-wide mb-1">
              {t.region || '地域'}
            </p>
            <p class="text-[12px] text-gray-900 font-medium">
              {config.region || '-'}
            </p>
          </div>

          <!-- Instance Type -->
          <div>
            <p class="text-[10px] text-gray-500 uppercase tracking-wide mb-1">
              {t.instanceType || '实例规格'}
            </p>
            <p class="text-[12px] text-gray-900 font-medium">
              {config.instanceType || '-'}
            </p>
          </div>

          <!-- Userdata -->
          {#if config.userdata}
            <div class="col-span-2">
              <p class="text-[10px] text-gray-500 uppercase tracking-wide mb-1">
                {t.userdata || 'Userdata'}
              </p>
              <div class="bg-white rounded border border-gray-100 p-2 max-h-24 overflow-y-auto">
                <pre class="text-[10px] text-gray-700 font-mono whitespace-pre-wrap">{config.userdata}</pre>
              </div>
            </div>
          {/if}
        </div>

        <!-- Template Variables -->
        {#if config.variables && Object.keys(config.variables).length > 0}
          <div class="mt-4 pt-4 border-t border-gray-100">
            <p class="text-[10px] text-gray-500 uppercase tracking-wide mb-2">
              {t.templateParams || '模板参数'}
            </p>
            <div class="grid grid-cols-2 gap-2">
              {#each Object.entries(config.variables) as [key, value]}
                {#if value}
                  <div class="flex items-start gap-2">
                    <span class="text-[11px] text-gray-600">{key}:</span>
                    <span class="text-[11px] text-gray-900 font-medium flex-1 break-all">{value}</span>
                  </div>
                {/if}
              {/each}
            </div>
          </div>
        {/if}
      </div>

      <!-- Cost Estimate -->
      {#if costEstimate}
        <div class="bg-blue-50 rounded-lg p-4 border border-blue-100">
          <div class="flex items-center justify-between mb-3">
            <h3 class="text-[12px] font-medium text-blue-900">
              {t.costEstimate || '成本估算'}
            </h3>
            
            <!-- Currency Selector -->
            <div class="flex items-center gap-1 bg-white rounded-md border border-blue-200 p-0.5">
              {#each supportedCurrencies as currency}
                <button
                  class="px-2 py-1 text-[10px] font-medium rounded transition-colors {selectedCurrency === currency ? 'bg-blue-600 text-white' : 'text-blue-700 hover:bg-blue-100'}"
                  onclick={() => selectedCurrency = currency}
                >
                  {currency}
                </button>
              {/each}
            </div>
          </div>

          <div class="text-right mb-3">
            <p class="text-[10px] text-blue-600 uppercase tracking-wide">
              {t.estimatedMonthlyCost || '预估月度成本'}
            </p>
            <p class="text-[18px] font-bold text-blue-900">
              {formatCost(displayedCost()?.amount, displayedCost()?.currency)}
            </p>
          </div>

          <!-- Cost Details -->
          {#if displayedDetails() && Object.keys(displayedDetails()).length > 0}
            <div class="space-y-1.5">
              <p class="text-[10px] text-blue-700 uppercase tracking-wide mb-1">
                {t.costBreakdown || '成本明细'}
              </p>
              {#each Object.entries(displayedDetails()) as [resource, cost]}
                <div class="flex items-center justify-between text-[11px]">
                  <span class="text-blue-700">{resource}</span>
                  <span class="text-blue-900 font-medium">
                    {formatCost(cost, selectedCurrency)}
                  </span>
                </div>
              {/each}
            </div>
          {/if}

          <!-- Cost Disclaimer -->
          <div class="mt-3 pt-3 border-t border-blue-200">
            <p class="text-[10px] text-blue-600 italic">
              {t.costDisclaimer || '* 成本估算仅供参考，实际费用可能有所不同'}
            </p>
          </div>
        </div>
      {/if}

      <!-- Validation Status -->
      {#if !validation.valid && validation.errors.length > 0}
        <div class="bg-red-50 rounded-lg p-4 border border-red-100">
          <div class="flex items-start gap-2">
            <svg class="w-4 h-4 text-red-500 flex-shrink-0 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z" />
            </svg>
            <div class="flex-1">
              <p class="text-[12px] font-medium text-red-900 mb-1">
                {t.cannotDeploy || '无法部署'}
              </p>
              <p class="text-[11px] text-red-700">
                {t.fixErrorsBeforeDeploy || '请修复以下错误后再部署'}
              </p>
            </div>
          </div>
        </div>
      {/if}

      <!-- Deploy Button -->
      <div class="pt-2">
        <button
          class="w-full h-11 px-4 bg-gray-900 hover:bg-gray-800 disabled:bg-gray-300 disabled:cursor-not-allowed text-white text-[13px] font-medium rounded-lg transition-colors flex items-center justify-center gap-2"
          disabled={!validation.valid || isDeploying}
          onclick={() => onDeploy()}
        >
          {#if isDeploying}
            <div class="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin"></div>
            <span>{t.deploying || '部署中...'}</span>
          {:else}
            <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
            </svg>
            <span>{t.confirmDeploy || '确认部署'}</span>
          {/if}
        </button>
      </div>
    </div>
  {/if}
</div>

<style>
  /* Component-specific styles if needed */
</style>
