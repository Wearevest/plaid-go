package plaid

import (
	"bytes"
	"encoding/json"
)

// Balance (POST /balance) retrieves real-time balance for a given access token.
//
// See https://plaid.com/docs/api/#balance.
func (c *Client) Transactions(accessToken string, startDate string, endDate string, options TransactionOptionsJson) (postRes *postResponse, err error) {
	jsonText, err := json.Marshal(transactionJson{
		ClientID:    c.clientID,
		Secret:      c.secret,
		AccessToken: accessToken,
		StartDate:   startDate,
		EndDate:     endDate,
		Options:     options,
	})
	if err != nil {
		return nil, err
	}
	postRes, _, err = c.postAndUnmarshal("/transactions/get", bytes.NewReader(jsonText))
	return postRes, err
}

type transactionJson struct {
	ClientID    string                 `json:"client_id"`
	Secret      string                 `json:"secret"`
	AccessToken string                 `json:"access_token"`
	StartDate   string                 `json:"start_date"`
	EndDate     string                 `json:"end_date"`
	Options     TransactionOptionsJson `json:"options"`
}

type TransactionOptionsJson struct {
	Count  int `json:"count"`
	Offset int `json:"offset"`
}
