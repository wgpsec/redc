package mod

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/global"
	bss "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/bss/v2"
	bssmodel "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/bss/v2/model"
	bssregion "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/bss/v2/region"
	bssintl "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/bssintl/v2"
	bssintlmodel "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/bssintl/v2/model"
	bssintlregion "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/bssintl/v2/region"
)

// QueryHuaweiBalance queries Huawei Cloud account balance via BSS API.
func QueryHuaweiBalance(accessKey string, secretKey string, region string) (string, string, error) {
	if accessKey == "" || secretKey == "" {
		return "", "", fmt.Errorf("missing huaweicloud access key or secret")
	}
	if region == "" {
		region = "cn-north-1"
	}
	if strings.HasPrefix(region, "cn-") && region != "cn-north-1" {
		region = "cn-north-1"
	}

	if strings.HasPrefix(region, "cn-") {
		cred := global.NewCredentialsBuilder().WithAk(accessKey).WithSk(secretKey).Build()
		client := bss.NewBssClient(bss.BssClientBuilder().WithCredential(cred).WithRegion(bssregion.ValueOf(region)).Build())

		request := &bssmodel.ShowCustomerAccountBalancesRequest{}
		response, err := client.ShowCustomerAccountBalances(request)
		if err != nil {
			return "", "", err
		}
		if response == nil || response.AccountBalances == nil || len(*response.AccountBalances) == 0 {
			return "", "", fmt.Errorf("empty huaweicloud balance response")
		}

		balances := *response.AccountBalances
		balance := balances[0]
		for _, item := range balances {
			if item.AccountType == 1 {
				balance = item
				break
			}
		}
		amount := formatHuaweiAmount(balance.Amount)
		if amount == "0" || amount == "0.00" {
			for _, item := range balances {
				candidate := formatHuaweiAmount(item.Amount)
				if candidate != "" && candidate != "0" && candidate != "0.00" {
					balance = item
					amount = candidate
					break
				}
			}
		}
		currency := balance.Currency
		if currency == "" {
			if response.Currency != nil {
				currency = *response.Currency
			}
		}
		if currency == "" {
			currency = "CNY"
		}
		return amount, currency, nil
	}

	return queryHuaweiBalanceIntl(accessKey, secretKey, region)
}

func queryHuaweiBalanceIntl(accessKey string, secretKey string, region string) (string, string, error) {
	cred := global.NewCredentialsBuilder().WithAk(accessKey).WithSk(secretKey).Build()
	if region == "" || strings.HasPrefix(region, "cn-") {
		region = "ap-southeast-1"
	}
	client := bssintl.NewBssintlClient(bssintl.BssintlClientBuilder().WithCredential(cred).WithRegion(bssintlregion.ValueOf(region)).Build())

	request := &bssintlmodel.ShowCustomerAccountBalancesRequest{}
	response, err := client.ShowCustomerAccountBalances(request)
	if err != nil {
		return "", "", err
	}
	if response == nil || response.AccountBalances == nil || len(*response.AccountBalances) == 0 {
		return "", "", fmt.Errorf("empty huaweicloud balance response")
	}

	balances := *response.AccountBalances
	balance := balances[0]
	for _, item := range balances {
		if item.AccountType == 1 {
			balance = item
			break
		}
	}
	amount := formatHuaweiAmount(balance.Amount)
	if amount == "0" || amount == "0.00" {
		for _, item := range balances {
			candidate := formatHuaweiAmount(item.Amount)
			if candidate != "" && candidate != "0" && candidate != "0.00" {
				balance = item
				amount = candidate
				break
			}
		}
	}
	currency := balance.Currency
	if currency == "" {
		if response.Currency != nil {
			currency = *response.Currency
		}
	}
	if currency == "" {
		currency = "USD"
	}
	return amount, currency, nil
}

func formatHuaweiAmount(value interface{}) string {
	if value == nil {
		return ""
	}
	switch v := value.(type) {
	case *big.Int:
		if v == nil {
			return ""
		}
		return v.String()
	case big.Int:
		return v.String()
	case *big.Float:
		if v == nil {
			return ""
		}
		return v.Text('f', 2)
	case big.Float:
		return v.Text('f', 2)
	case *big.Rat:
		if v == nil {
			return ""
		}
		return v.FloatString(2)
	case big.Rat:
		return v.FloatString(2)
	case float64:
		return fmt.Sprintf("%.2f", v)
	case *float64:
		if v == nil {
			return ""
		}
		return fmt.Sprintf("%.2f", *v)
	case int, int32, int64, uint, uint32, uint64:
		return fmt.Sprintf("%v", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}
