package plaid

import (
	"fmt"
)

type plaidError struct {
	// List of all errors: https://github.com/plaid/support/blob/master/errors.md
	ErrorCode      string `json:"error_code"`
	ErrorType      string `json:"error_type"`
	ErrorMessage   string `json:"error_message"`
	DisplayMessage string `json:"display_message"`

	// StatusCode needs to manually set from the http response
	StatusCode int
}

func (e plaidError) Error() string {
	return fmt.Sprintf("Plaid Error - http status: %s, code: %s, message: %s, display: %s",
		e.ErrorCode, e.ErrorType, e.ErrorMessage, e.DisplayMessage)
}
