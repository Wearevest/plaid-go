package plaid

import (
	"fmt"
)

type plaidError struct {
	// List of all errors: https://github.com/plaid/support/blob/master/errors.md
	HttpErrorCode int    `json:"http_code"`
	ErrorCode     string `json:"error_type"`
	Message       string `json:"error_message"`

	// StatusCode needs to manually set from the http response
	StatusCode int
}

func (e plaidError) Error() string {
	return fmt.Sprintf("Plaid Error - http status: %d, code: %d, message: %s",
		e.HttpErrorCode, e.ErrorCode, e.Message)
}
