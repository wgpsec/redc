<script>

  import { GetResourceSummary, GetBalances, GetBills } from '../../../wailsjs/go/main/App.js';
  import { BrowserOpenURL } from '../../../wailsjs/runtime/runtime.js';
  import { cloudDocs } from '../../lib/cloudDocs.js';

  function openLink(url) {
    BrowserOpenURL(url);
  }

  const docColorMap = {
    orange: { bg: 'bg-orange-100', text: 'text-orange-600' },
    blue:   { bg: 'bg-blue-100',   text: 'text-blue-600' },
    red:    { bg: 'bg-red-100',    text: 'text-red-600' },
    purple: { bg: 'bg-purple-100', text: 'text-purple-600' },
  };

  function getDocColor(color) {
    return docColorMap[color] || docColorMap.purple;
  }

  let { t } = $props();
  
  let resourceSummary = $state([]);
  let resourcesLoading = $state(false);
  let resourcesError = $state('');
  let balanceResults = $state([]);
  let balanceLoading = $state(false);
  let balanceError = $state('');
  let balanceCooldown = $state(0);
  let balanceCooldownTimer = $state(null);
  let billResults = $state([]);
  let billLoading = $state(false);
  let billError = $state('');

  export function loadResourceSummary() {
    resourcesLoading = true;
    resourcesError = '';
    return GetResourceSummary()
      .then(data => {
        resourceSummary = data || [];
        return data;
      })
      .catch(e => {
        resourcesError = e.message || String(e);
        resourceSummary = [];
        throw e;
      })
      .finally(() => {
        resourcesLoading = false;
      });
  }

  export function queryBalances() {
    if (balanceCooldown > 0) return Promise.resolve();
    balanceLoading = true;
    balanceError = '';
    return GetBalances(['aliyun', 'tencentcloud', 'volcengine', 'huaweicloud', 'ucloud', 'vultr'])
      .then(data => {
        balanceResults = data || [];
        balanceCooldown = 5;
        if (balanceCooldownTimer) {
          clearInterval(balanceCooldownTimer);
        }
        balanceCooldownTimer = setInterval(() => {
          balanceCooldown = Math.max(0, balanceCooldown - 1);
          if (balanceCooldown === 0 && balanceCooldownTimer) {
            clearInterval(balanceCooldownTimer);
            balanceCooldownTimer = null;
          }
        }, 1000);
        return data;
      })
      .catch(e => {
        balanceError = e.message || String(e);
        throw e;
      })
      .finally(() => {
        balanceLoading = false;
      });
  }
  export function queryBills() {
    billLoading = true;
    billError = '';
    return GetBills(['aws', 'vultr'])
      .then(data => {
        billResults = data || [];
        return data;
      })
      .catch(e => {
        billError = e.message || String(e);
        throw e;
      })
      .finally(() => {
        billLoading = false;
      });
  }

</script>

<div class="space-y-4">
  <!-- Resource Summary -->
  <div class="bg-white rounded-xl border border-gray-100 p-5">
    <div class="flex items-center justify-between mb-4">
      <h3 class="text-[13px] font-semibold text-gray-900">{t.resourceSummary}</h3>
      <button
        class="p-1.5 text-gray-400 hover:text-gray-600 hover:bg-gray-100 rounded-lg transition-colors cursor-pointer"
        onclick={loadResourceSummary}
        disabled={resourcesLoading}
        title={t.refresh || '刷新'}
      >
        <svg class="w-4 h-4 {resourcesLoading ? 'animate-spin' : ''}" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M16.023 9.348h4.992v-.001M2.985 19.644v-4.992m0 0h4.992m-4.993 0l3.181 3.183a8.25 8.25 0 0013.803-3.7M4.031 9.865a8.25 8.25 0 0113.803-3.7l3.181 3.182m0-4.991v4.99" /></svg>
      </button>
    </div>

    {#if resourcesError}
      <div class="flex items-center gap-2 px-4 py-2.5 bg-red-50 border border-red-100 rounded-xl mb-4">
        <svg class="w-3.5 h-3.5 text-red-500 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z" /></svg>
        <span class="text-[12px] text-red-700 flex-1">{resourcesError}</span>
        <button class="p-0.5 text-red-400 hover:text-red-600 cursor-pointer" onclick={() => resourcesError = ''}>
          <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" /></svg>
        </button>
      </div>
    {/if}

    {#if resourcesLoading && resourceSummary.length === 0}
      <div class="flex items-center justify-center gap-2 py-8">
        <svg class="w-4 h-4 animate-spin text-gray-400" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path></svg>
        <span class="text-[12px] text-gray-400">{t.loading || '加载中...'}</span>
      </div>
    {:else}
      <div class="border border-gray-100 rounded-lg overflow-hidden">
        <table class="w-full text-[12px]">
          <thead>
            <tr class="bg-gray-50 border-b border-gray-100">
              <th class="text-left px-4 py-2.5 font-medium text-gray-500">{t.resourceType}</th>
              <th class="text-right px-4 py-2.5 font-medium text-gray-500">{t.resourceCount}</th>
            </tr>
          </thead>
          <tbody>
            {#each resourceSummary as r}
              <tr class="border-b border-gray-50 hover:bg-gray-50/50 transition-colors">
                <td class="px-4 py-3 text-gray-700">{r.type}</td>
                <td class="px-4 py-3 text-right font-medium text-gray-900">{r.count}</td>
              </tr>
            {:else}
              <tr>
                <td colspan="2" class="py-10 text-center">
                  <svg class="w-8 h-8 mx-auto text-gray-200 mb-2" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M20.25 6.375c0 2.278-3.694 4.125-8.25 4.125S3.75 8.653 3.75 6.375m16.5 0c0-2.278-3.694-4.125-8.25-4.125S3.75 4.097 3.75 6.375m16.5 0v11.25c0 2.278-3.694 4.125-8.25 4.125s-8.25-1.847-8.25-4.125V6.375m16.5 0v3.75m-16.5-3.75v3.75m16.5 0v3.75C20.25 16.153 16.556 18 12 18s-8.25-1.847-8.25-4.125v-3.75m16.5 0c0 2.278-3.694 4.125-8.25 4.125s-8.25-1.847-8.25-4.125" />
                  </svg>
                  <p class="text-[12px] text-gray-400">{t.noScene}</p>
                  <p class="text-[11px] text-gray-300 mt-0.5">{t.clickRefreshToLoad || '点击刷新按钮加载数据'}</p>
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/if}
  </div>

  <!-- Balance Query -->
  <div class="bg-white rounded-xl border border-gray-100 p-5">
    <div class="flex items-center justify-between mb-4">
      <div>
        <h3 class="text-[13px] font-semibold text-gray-900">{t.balanceQuery}</h3>
        <p class="text-[11px] text-gray-400 mt-0.5">{t.profileSwitchHint}</p>
      </div>
      <button
        class="h-8 px-3.5 bg-gray-900 text-white text-[12px] font-medium rounded-lg hover:bg-gray-800 transition-colors disabled:opacity-50 inline-flex items-center gap-1.5 cursor-pointer"
        onclick={queryBalances}
        disabled={balanceLoading || balanceCooldown > 0}
      >
        {#if balanceLoading}
          <svg class="animate-spin h-3.5 w-3.5" viewBox="0 0 24 24" fill="none"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path></svg>
          {t.loading || '查询中...'}
        {:else if balanceCooldown > 0}
          <svg class="w-3.5 h-3.5 opacity-50" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M12 6v6h4.5m4.5 0a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
          {balanceCooldown}s
        {:else}
          <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M2.25 18.75a60.07 60.07 0 0115.797 2.101c.727.198 1.453-.342 1.453-1.096V18.75M3.75 4.5v.75A.75.75 0 013 6h-.75m0 0v-.375c0-.621.504-1.125 1.125-1.125H20.25M2.25 6v9m18-10.5v.75c0 .414.336.75.75.75h.75m-1.5-1.5h.375c.621 0 1.125.504 1.125 1.125v9.75c0 .621-.504 1.125-1.125 1.125h-.375m1.5-1.5H21a.75.75 0 00-.75.75v.75m0 0H3.75m0 0h-.375a1.125 1.125 0 01-1.125-1.125V15m1.5 1.5v-.75A.75.75 0 003 15h-.75M15 10.5a3 3 0 11-6 0 3 3 0 016 0zm3 0h.008v.008H18V10.5zm-12 0h.008v.008H6V10.5z" /></svg>
          {t.balanceQuery}
        {/if}
      </button>
    </div>

    {#if balanceError}
      <div class="flex items-center gap-2 px-4 py-2.5 bg-red-50 border border-red-100 rounded-xl mb-4">
        <svg class="w-3.5 h-3.5 text-red-500 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z" /></svg>
        <span class="text-[12px] text-red-700 flex-1">{balanceError}</span>
        <button class="p-0.5 text-red-400 hover:text-red-600 cursor-pointer" onclick={() => balanceError = ''}>
          <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" /></svg>
        </button>
      </div>
    {/if}

    <div class="border border-gray-100 rounded-lg overflow-hidden">
      <table class="w-full text-[12px]">
        <thead>
          <tr class="bg-gray-50 border-b border-gray-100">
            <th class="text-left px-4 py-2.5 font-medium text-gray-500">{t.balanceProvider}</th>
            <th class="text-right px-4 py-2.5 font-medium text-gray-500">{t.balanceAmount}</th>
            <th class="text-left px-4 py-2.5 font-medium text-gray-500">{t.balanceCurrency}</th>
            <th class="text-left px-4 py-2.5 font-medium text-gray-500">{t.balanceUpdatedAt}</th>
          </tr>
        </thead>
        <tbody>
          {#each balanceResults as b}
            <tr class="border-b border-gray-50 hover:bg-gray-50/50 transition-colors">
              <td class="px-4 py-3 text-gray-700">{b.provider}</td>
              <td class="px-4 py-3 text-right font-medium text-gray-900">{b.amount}</td>
              <td class="px-4 py-3 text-gray-500">{b.currency}</td>
              <td class="px-4 py-3 text-gray-400">{b.updatedAt}</td>
            </tr>
            {#if b.error}
              <tr class="border-b border-gray-50">
                <td colspan="4" class="px-4 pb-3">
                  <span class="inline-flex items-center gap-1 text-[11px] text-amber-600">
                    <svg class="w-3 h-3 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126zM12 15.75h.007v.008H12v-.008z" /></svg>
                    {b.error}
                  </span>
                </td>
              </tr>
            {/if}
          {:else}
            <tr>
              <td colspan="4" class="py-10 text-center">
                <svg class="w-8 h-8 mx-auto text-gray-200 mb-2" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M2.25 18.75a60.07 60.07 0 0115.797 2.101c.727.198 1.453-.342 1.453-1.096V18.75M3.75 4.5v.75A.75.75 0 013 6h-.75m0 0v-.375c0-.621.504-1.125 1.125-1.125H20.25M2.25 6v9m18-10.5v.75c0 .414.336.75.75.75h.75m-1.5-1.5h.375c.621 0 1.125.504 1.125 1.125v9.75c0 .621-.504 1.125-1.125 1.125h-.375m1.5-1.5H21a.75.75 0 00-.75.75v.75m0 0H3.75m0 0h-.375a1.125 1.125 0 01-1.125-1.125V15m1.5 1.5v-.75A.75.75 0 003 15h-.75M15 10.5a3 3 0 11-6 0 3 3 0 016 0zm3 0h.008v.008H18V10.5zm-12 0h.008v.008H6V10.5z" />
                </svg>
                <p class="text-[12px] text-gray-400">{t.balancePlaceholder}</p>
                <p class="text-[11px] text-gray-300 mt-0.5">{t.clickQueryToLoad || '点击查询按钮获取余额'}</p>
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  </div>

  <!-- Monthly Bill -->
  <div class="bg-white rounded-xl border border-gray-100 p-5">
    <div class="flex items-center justify-between mb-4">
      <div>
        <h3 class="text-[13px] font-semibold text-gray-900">{t.currentMonthBill || '当月账单'}</h3>
        <p class="text-[10px] text-amber-500 mt-0.5">{t.billCostWarning || 'AWS Cost Explorer 每次查询约 $0.01'}</p>
      </div>
      <button
        class="h-8 px-3.5 bg-gray-900 text-white text-[12px] font-medium rounded-lg hover:bg-gray-800 transition-colors disabled:opacity-50 inline-flex items-center gap-1.5 cursor-pointer"
        onclick={queryBills}
        disabled={billLoading}
      >
        {#if billLoading}
          <svg class="animate-spin h-3.5 w-3.5" viewBox="0 0 24 24" fill="none"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path></svg>
          {t.loading || '查询中...'}
        {:else}
          <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M2.25 8.25h19.5M2.25 9h19.5m-16.5 5.25h6m-6 2.25h3m-3.75 3h15a2.25 2.25 0 002.25-2.25V6.75A2.25 2.25 0 0019.5 4.5h-15a2.25 2.25 0 00-2.25 2.25v10.5A2.25 2.25 0 004.5 19.5z" /></svg>
          {t.queryBill || '查询账单'}
        {/if}
      </button>
    </div>

    {#if billError}
      <div class="flex items-center gap-2 px-4 py-2.5 bg-red-50 border border-red-100 rounded-xl mb-4">
        <svg class="w-3.5 h-3.5 text-red-500 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z" /></svg>
        <span class="text-[12px] text-red-700 flex-1">{billError}</span>
        <button class="p-0.5 text-red-400 hover:text-red-600 cursor-pointer" onclick={() => billError = ''}>
          <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" /></svg>
        </button>
      </div>
    {/if}

    {#if billResults.length > 0}
      <div class="border border-gray-100 rounded-lg overflow-hidden">
        <table class="w-full text-[12px]">
          <thead>
            <tr class="bg-gray-50 border-b border-gray-100">
              <th class="text-left px-4 py-2.5 font-medium text-gray-500">{t.provider || '云厂商'}</th>
              <th class="text-right px-4 py-2.5 font-medium text-gray-500">{t.amount || '金额'}</th>
              <th class="text-right px-4 py-2.5 font-medium text-gray-500">{t.period || '账期'}</th>
            </tr>
          </thead>
          <tbody>
            {#each billResults as bill}
              <tr class="border-b border-gray-50 last:border-0">
                <td class="px-4 py-3 font-medium text-gray-700 uppercase">{bill.provider}</td>
                <td class="px-4 py-3 text-right tabular-nums">
                  {#if bill.error}
                    <span class="text-[11px] text-amber-500">{bill.error}</span>
                  {:else}
                    <span class="font-medium text-gray-900">{bill.currency} {bill.totalAmount}</span>
                  {/if}
                </td>
                <td class="px-4 py-3 text-right text-gray-400">{bill.startDate || ''} ~ {bill.endDate || ''}</td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {:else if !billLoading}
      <div class="text-center py-6">
        <svg class="w-8 h-8 mx-auto text-gray-200 mb-2" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
          <path stroke-linecap="round" stroke-linejoin="round" d="M2.25 8.25h19.5M2.25 9h19.5m-16.5 5.25h6m-6 2.25h3m-3.75 3h15a2.25 2.25 0 002.25-2.25V6.75A2.25 2.25 0 0019.5 4.5h-15a2.25 2.25 0 00-2.25 2.25v10.5A2.25 2.25 0 004.5 19.5z" />
        </svg>
        <p class="text-[12px] text-gray-400">{t.clickToQueryBill || '点击查询按钮获取当月账单'}</p>
      </div>
    {/if}
  </div>

  <!-- Dev Docs -->
  <div class="bg-white rounded-xl border border-gray-100 p-5">
    <h3 class="text-[13px] font-semibold text-gray-900 mb-3">{t.devDocs || '开发文档'}</h3>

    <div class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-6 gap-2">
      {#each cloudDocs as doc}
        {@const c = getDocColor(doc.color)}
        <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
        <div
          class="flex items-center gap-2 px-3 py-2.5 bg-gray-50 hover:bg-gray-100 rounded-lg transition-colors group cursor-pointer"
          onclick={() => openLink(doc.url)}
        >
          <div class="w-7 h-7 rounded-md flex items-center justify-center flex-shrink-0 {c.bg}">
            <svg class="w-3.5 h-3.5 {c.text}" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M13.5 6H5.25A2.25 2.25 0 003 8.25v10.5A2.25 2.25 0 005.25 21h10.5A2.25 2.25 0 0018 18.75V10.5m-10.5 6L21 3m0 0h-5.25M21 3v5.25" />
            </svg>
          </div>
          <span class="text-[12px] font-medium text-gray-600 group-hover:text-gray-900 truncate">{t[doc.name] || doc.nameZh}</span>
        </div>
      {/each}
    </div>
  </div>
</div>
