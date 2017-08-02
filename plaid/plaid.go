// Package plaid implements a Go client for the Plaid API (https://plaid.com/docs)
package plaid

import (
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
)

// NewClient instantiates a Client associated with a client id, secret and environment.
// See https://plaid.com/docs/api/#gaining-access.
func NewClient(clientID, secret string, environment environmentURL) *Client {
	return &Client{clientID, secret, environment, &http.Client{}}
}

// Same as above but with additional parameter to pass http.Client. This is required
// if you want to run the code on Google AppEngine which prohibits use of http.DefaultClient
func NewCustomClient(clientID, secret string, environment environmentURL, httpClient *http.Client) *Client {
	return &Client{clientID, secret, environment, httpClient}
}

// Note: Client is only exported for method documentation purposes.
// Instances should only be created through the 'NewClient' function.
//
// See https://github.com/golang/go/issues/7823.
type Client struct {
	clientID    string
	secret      string
	environment environmentURL
	httpClient  *http.Client
}

type environmentURL string

var Sandbox environmentURL = "https://sandbox.plaid.com"
var Production environmentURL = "https://production.plaid.com"

type Account struct {
	Transactions []Transaction `json:"transactions" bson:"transactions"`
	Type         string        `json:"type"`
	Mask         string        `json:"mask"`
	Name         string        `json:"name"`
	AccountID    string        `json:"account_id"`
	Balances     struct {
		Limit     float64 `json:"limit"`
		Available float64 `json:"available"`
		Current   float64 `json:"current"`
	} `json:"balances"`
	Subtype      string `json:"subtype"`
	OfficialName string `json:"official_name"`
}

type Transaction struct {
	PendingTransactionID string   `json:"pending_transaction_id"`
	Name                 string   `json:"name"`
	AccountOwner         string   `json:"account_owner"`
	Category             []string `json:"category"`
	TransactionType      string   `json:"transaction_type"`
	AccountID            string   `json:"account_id"`
	Amount               float32  `json:"amount"`
	Date                 string   `json:"date"`
	TransactionID        string   `json:"transaction_id"`
	Location             struct {
		Zip         string  `json:"zip"`
		State       string  `json:"state"`
		StoreNumber string  `json:"store_number"`
		Lon         float64 `json:"lon"`
		City        string  `json:"city"`
		Lat         float64 `json:"lat"`
		Address     string  `json:"address"`
	} `json:"location"`
	CategoryID  string `json:"category_id"`
	Pending     bool   `json:"pending"`
	PaymentMeta struct {
		Reason           string `json:"reason"`
		Payee            string `json:"payee"`
		PpdID            string `json:"ppd_id"`
		Payer            string `json:"payer"`
		ByOrderOf        string `json:"by_order_of"`
		ReferenceNumber  string `json:"reference_number"`
		PaymentProcessor string `json:"payment_processor"`
		PaymentMethod    string `json:"payment_method"`
	} `json:"payment_meta"`
}

type mfaIntermediate struct {
	AccessToken string      `json:"access_token"`
	MFA         interface{} `json:"mfa"`
	Type        string      `json:"type"`
}
type mfaDevice struct {
	Message string
}
type mfaList struct {
	Mask string
	Type string
}
type mfaQuestion struct {
	Question string
}
type mfaSelection struct {
	Answers  []string
	Question string
}

// 'mfa' contains the union of all possible mfa types
// Users should switch on the 'Type' field
type mfaResponse struct {
	AccessToken string
	Type        string

	Device     mfaDevice
	List       []mfaList
	Questions  []mfaQuestion
	Selections []mfaSelection
}

type postResponse struct {
	// Normal response fields
	AccessToken       string        `json:"access_token"`
	AccountId         string        `json:"account_id"`
	Accounts          []Account     `json:"accounts"`
	BankAccountToken  string        `json:"stripe_bank_account_token"`
	MFA               string        `json:"mfa"`
	Transactions      []Transaction `json:"transactions"`
	TotalTransactions int           `json:"total_transactions"`
	Item              Item          `json:"item"`
}
type Item struct {
	InstitutionId string `json:"institution_id"`
	ItemId        string `json:"item_id"`
	Webhook       string `json:"webhook"`
}

type deleteResponse struct {
	Message string `json:"message"`
}

// getAndUnmarshal is not a method because no client authentication is required
func getAndUnmarshal(environment environmentURL, endpoint string, structure interface{}) error {
	res, err := http.Get(string(environment) + endpoint)
	if err != nil {
		return err
	}
	raw, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	res.Body.Close()

	// Successful response
	if res.StatusCode == 200 {
		if err = json.Unmarshal(raw, structure); err != nil {
			return err
		}
		return nil
	}
	// Attempt to unmarshal into Plaid error format
	var plaidErr plaidError
	if err = json.Unmarshal(raw, &plaidErr); err != nil {
		return err
	}
	plaidErr.StatusCode = res.StatusCode
	return plaidErr
}

func (c *Client) postAndUnmarshal(endpoint string,
	body io.Reader) (*postResponse, *mfaResponse, error) {
	// Read response body
	req, err := http.NewRequest("POST", string(c.environment)+endpoint, body)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("User-Agent", "plaid-go")
	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, nil, err
	}
	raw, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, nil, err
	}
	res.Body.Close()

	return unmarshalPostMFA(res, raw)
}

func (c *Client) patchAndUnmarshal(endpoint string,
	body io.Reader) (*postResponse, *mfaResponse, error) {

	req, err := http.NewRequest("PATCH", string(c.environment)+endpoint, body)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("User-Agent", "plaid-go")
	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, nil, err
	}
	raw, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, nil, err
	}
	res.Body.Close()

	return unmarshalPostMFA(res, raw)
}

func (c *Client) deleteAndUnmarshal(endpoint string,
	body io.Reader) (*deleteResponse, error) {

	req, err := http.NewRequest("DELETE", string(c.environment)+endpoint, body)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("User-Agent", "plaid-go")
	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	raw, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	res.Body.Close()

	// Successful response
	var deleteRes deleteResponse
	if res.StatusCode == 200 {
		if err = json.Unmarshal(raw, &deleteRes); err != nil {
			return nil, err
		}
		return &deleteRes, nil
	}
	// Attempt to unmarshal into Plaid error format
	var plaidErr plaidError
	if err = json.Unmarshal(raw, &plaidErr); err != nil {
		return nil, err
	}
	plaidErr.StatusCode = res.StatusCode
	return nil, plaidErr
}

// Unmarshals response into postResponse, mfaResponse, or plaidError
func unmarshalPostMFA(res *http.Response, body []byte) (*postResponse, *mfaResponse, error) {
	// Different marshaling cases
	var mfaInter mfaIntermediate
	var postRes postResponse
	var err error
	switch {
	// Successful response
	case res.StatusCode == 200:
		if err = json.Unmarshal(body, &postRes); err != nil {

			return nil, nil, err
		}
		return &postRes, nil, nil

	// MFA case
	case res.StatusCode == 201:
		if err = json.Unmarshal(body, &mfaInter); err != nil {
			return nil, nil, err
		}
		mfaRes := mfaResponse{Type: mfaInter.Type, AccessToken: mfaInter.AccessToken}
		switch mfaInter.Type {
		case "device":
			temp, ok := mfaInter.MFA.(interface{})
			if !ok {
				return nil, nil, errors.New("Could not decode device mfa")
			}
			deviceStruct, ok := temp.(map[string]interface{})
			if !ok {
				return nil, nil, errors.New("Could not decode device mfa")
			}
			deviceText, ok := deviceStruct["message"].(string)
			if !ok {
				return nil, nil, errors.New("Could not decode device mfa")
			}
			mfaRes.Device.Message = deviceText

		case "list":
			temp, ok := mfaInter.MFA.([]interface{})
			if !ok {
				return nil, nil, errors.New("Could not decode list mfa")
			}
			for _, v := range temp {
				listArray, ok := v.(map[string]interface{})
				if !ok {
					return nil, nil, errors.New("Could not decode list mfa")
				}
				maskText, ok := listArray["mask"].(string)
				if !ok {
					return nil, nil, errors.New("Could not decode list mfa")
				}
				typeText, ok := listArray["type"].(string)
				if !ok {
					return nil, nil, errors.New("Could not decode list mfa")
				}
				mfaRes.List = append(mfaRes.List, mfaList{Mask: maskText, Type: typeText})
			}

		case "questions":
			questions, ok := mfaInter.MFA.([]interface{})
			if !ok {
				return nil, nil, errors.New("Could not decode questions mfa")
			}
			for _, v := range questions {
				q, ok := v.(map[string]interface{})
				if !ok {
					return nil, nil, errors.New("Could not decode questions mfa")
				}
				questionText, ok := q["question"].(string)
				if !ok {
					return nil, nil, errors.New("Could not decode questions mfa question")
				}
				mfaRes.Questions = append(mfaRes.Questions, mfaQuestion{Question: questionText})
			}

		case "selections":
			selections, ok := mfaInter.MFA.([]interface{})
			if !ok {
				return nil, nil, errors.New("Could not decode selections mfa")
			}
			for _, v := range selections {
				s, ok := v.(map[string]interface{})
				if !ok {
					return nil, nil, errors.New("Could not decode selections mfa")
				}
				tempAnswers, ok := s["answers"].([]interface{})
				if !ok {
					return nil, nil, errors.New("Could not decode selections answers")
				}
				answers := make([]string, len(tempAnswers))
				for i, a := range tempAnswers {
					answers[i], ok = a.(string)
				}
				if !ok {
					return nil, nil, errors.New("Could not decode selections answers")
				}
				question, ok := s["question"].(string)
				if !ok {
					return nil, nil, errors.New("Could not decode selections questions")
				}
				mfaRes.Selections = append(mfaRes.Selections, mfaSelection{Answers: answers, Question: question})
			}
		}
		return nil, &mfaRes, nil

	// Error case, attempt to unmarshal into Plaid error format
	case res.StatusCode >= 400:
		var plaidErr plaidError
		if err = json.Unmarshal(body, &plaidErr); err != nil {
			return nil, nil, err
		}
		plaidErr.StatusCode = res.StatusCode
		return nil, nil, plaidErr
	}
	return nil, nil, errors.New("Unknown Plaid Error - Status:" + string(res.StatusCode))
}
