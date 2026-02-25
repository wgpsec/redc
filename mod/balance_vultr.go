package mod

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// QueryVultrBalance queries Vultr account balance via API.
func QueryVultrBalance(apiKey string) (string, string, error) {
	if apiKey == "" {
		return "", "", fmt.Errorf("missing vultr api key")
	}

	req, err := http.NewRequest("GET", "https://api.vultr.com/v2/account", nil)
	if err != nil {
		return "", "", err
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("vultr api error: %s", string(body))
	}

	var result struct {
		Account struct {
			Balance         float64 `json:"balance"`
			PendingCharges  float64 `json:"pending_charges"`
			LastPaymentDate string  `json:"last_payment_date"`
			LastPaymentAmount float64 `json:"last_payment_amount"`
		} `json:"account"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", "", fmt.Errorf("failed to parse vultr response: %v", err)
	}

	// Vultr 返回的余额是负数（表示欠费），pending_charges 是待结算金额
	// 账户实际余额 = balance + pending_charges (因为 pending 是预估的未结算费用)
	totalBalance := result.Account.Balance + result.Account.PendingCharges

	currency := "USD"
	return fmt.Sprintf("%.2f", totalBalance), currency, nil
}
