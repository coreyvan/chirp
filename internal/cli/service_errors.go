package cli

import (
	"errors"

	appnode "github.com/coreyvan/chirp/internal/app/node"
)

func mapServiceError(err error) error {
	if err == nil {
		return nil
	}

	var validationErr *appnode.ValidationError
	if errors.As(err, &validationErr) {
		return newUserInputError(err)
	}
	return err
}
