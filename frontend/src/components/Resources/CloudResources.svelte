<script>

  import { GetResourceSummary, GetBalances } from '../../../wailsjs/go/main/App.js';

let { t } = $props();
  
  let resourceSummary = $state([]);
  let resourcesLoading = $state(false);
  let resourcesError = $state('');
  let balanceResults = $state([]);
  let balanceLoading = $state(false);
  let balanceError = $state('');
  let balanceCooldown = $state(0);
  let balanceCooldownTimer = $state(null);

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
    return GetBalances(['aliyun', 'tencentcloud', 'volcengine', 'huaweicloud'])
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

</script>

<div class="max-w-3xl lg:max-w-5xl xl:max-w-full space-y-5">
  <div class="bg-white rounded-xl border border-gray-100 p-5">
    <div class="flex items-center justify-between mb-4">
      <div>
        <h3 class="text-[14px] font-semibold text-gray-900">{t.resourceSummary}</h3>
        <p class="text-[12px] text-gray-500"></p>
      </div>
      <button
        class="h-9 px-4 bg-gray-900 text-white text-[12px] font-medium rounded-lg hover:bg-gray-800 transition-colors disabled:opacity-50"
        onclick={loadResourceSummary}
        disabled={resourcesLoading}
      >
        {resourcesLoading ? t.loading : t.refresh}
      </button>
    </div>

    {#if resourcesError}
      <div class="text-[12px] text-red-500">{resourcesError}</div>
    {:else if resourcesLoading}
      <div class="flex items-center justify-center h-24">
        <div class="w-6 h-6 border-2 border-gray-200 border-t-gray-900 rounded-full animate-spin"></div>
      </div>
    {:else}
      <div class="border border-gray-100 rounded-lg overflow-hidden">
        <table class="w-full text-[12px]">
          <thead>
            <tr class="bg-gray-50 border-b border-gray-100">
              <th class="text-left px-4 py-2.5 font-semibold text-gray-600">{t.resourceType}</th>
              <th class="text-right px-4 py-2.5 font-semibold text-gray-600">{t.resourceCount}</th>
            </tr>
          </thead>
          <tbody>
            {#each resourceSummary as r}
              <tr class="border-b border-gray-50">
                <td class="px-4 py-3 text-gray-700">{r.type}</td>
                <td class="px-4 py-3 text-right text-gray-700">{r.count}</td>
              </tr>
            {:else}
              <tr>
                <td colspan="2" class="py-12 text-center text-[12px] text-gray-400">{t.noScene}</td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/if}
  </div>

  <div class="bg-white rounded-xl border border-gray-100 p-5">
    <div class="flex items-center justify-between mb-4">
      <div>
        <h3 class="text-[14px] font-semibold text-gray-900">{t.balanceQuery}</h3>
        <p class="text-[12px] text-gray-500">{t.profileSwitchHint}</p>
      </div>
      <button
        class="h-9 px-4 bg-blue-600 text-white text-[12px] font-medium rounded-lg hover:bg-blue-700 transition-colors disabled:opacity-50"
        onclick={queryBalances}
        disabled={balanceLoading || balanceCooldown > 0}
      >
        {balanceLoading ? t.loading : balanceCooldown > 0 ? `${t.balanceCooldown} ${balanceCooldown}s` : t.balanceQuery}
      </button>
    </div>

    {#if balanceError}
      <div class="text-[12px] text-red-500">{balanceError}</div>
    {:else}
      <div class="border border-gray-100 rounded-lg overflow-hidden">
        <table class="w-full text-[12px]">
          <thead>
            <tr class="bg-gray-50 border-b border-gray-100">
              <th class="text-left px-4 py-2.5 font-semibold text-gray-600">{t.balanceProvider}</th>
              <th class="text-right px-4 py-2.5 font-semibold text-gray-600">{t.balanceAmount}</th>
              <th class="text-left px-4 py-2.5 font-semibold text-gray-600">{t.balanceCurrency}</th>
              <th class="text-left px-4 py-2.5 font-semibold text-gray-600">{t.balanceUpdatedAt}</th>
            </tr>
          </thead>
          <tbody>
            {#each balanceResults as b}
              <tr class="border-b border-gray-50">
                <td class="px-4 py-3 text-gray-700">{b.provider}</td>
                <td class="px-4 py-3 text-right text-gray-700">{b.amount}</td>
                <td class="px-4 py-3 text-gray-700">{b.currency}</td>
                <td class="px-4 py-3 text-gray-500">{b.updatedAt}</td>
              </tr>
              {#if b.error}
                <tr class="border-b border-gray-50">
                  <td colspan="4" class="px-4 pb-3 text-[11px] text-amber-600">{b.error}</td>
                </tr>
              {/if}
            {:else}
              <tr>
                <td colspan="4" class="py-12 text-center text-[12px] text-gray-400">{t.balancePlaceholder}</td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/if}
  </div>
</div>
