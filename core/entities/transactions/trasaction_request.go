package entities

import (
	"errors"
	"math"

	"github.com/go-playground/validator/v10"
)

// maxAmount is the upper bound of the DECIMAL(18, 2) amount and balance columns.
const maxAmount = 1e16

type DepositRequest struct {
	AccountNumber string  `json:"account_number" validate:"required,max=18"`
	Amount        float64 `json:"amount"`
	Currency      string  `json:"currency" validate:"required,len=3"`
}

func (req DepositRequest) Validate() error {
	if err := validator.New().Struct(req); err != nil {
		return errors.New("account_number and a 3-letter currency are required")
	}
	return validateAmount(req.Amount)
}

type TransferIbanRequest struct {
	SourceIban      string  `json:"source_iban" validate:"required,max=31"`
	DestinationIban string  `json:"destination_iban" validate:"required,max=31"`
	Amount          float64 `json:"amount"`
	Currency        string  `json:"currency" validate:"required,len=3"`
}

func (req TransferIbanRequest) Validate() error {
	v := validator.New()
	return v.Struct(req)
}

type TransferNumberRequest struct {
	SourceNumber      string  `json:"source_number" validate:"required,max=18"`
	DestinationNumber string  `json:"destination_number" validate:"required,max=18"`
	Amount            float64 `json:"amount"`
	Currency          string  `json:"currency" validate:"required,len=3"`
}

func (req TransferNumberRequest) Validate() error {
	if err := validator.New().Struct(req); err != nil {
		return errors.New("source_number, destination_number and a 3-letter currency are required")
	}
	if err := validateAmount(req.Amount); err != nil {
		return err
	}
	if req.SourceNumber == req.DestinationNumber {
		return errors.New("source and destination accounts must be different")
	}
	return nil
}

type WithdrawRequest struct {
	AccountNumber string  `json:"account_number" validate:"required,max=18"`
	Amount        float64 `json:"amount"`
	Currency      string  `json:"currency" validate:"required,len=3"`
}

func (req WithdrawRequest) Validate() error {
	if err := validator.New().Struct(req); err != nil {
		return errors.New("account_number and a 3-letter currency are required")
	}
	return validateAmount(req.Amount)
}

// validateAmount accepts positive amounts expressed in whole cents.
func validateAmount(amount float64) error {
	if amount <= 0 {
		return errors.New("amount must be greater than zero")
	}
	if amount >= maxAmount {
		return errors.New("amount is too large")
	}
	if math.Round(amount*100)/100 != amount {
		return errors.New("amount must have at most two decimal places")
	}
	return nil
}
