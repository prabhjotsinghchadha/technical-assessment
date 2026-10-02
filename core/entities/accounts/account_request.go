package entities

import (
	"github.com/go-playground/validator/v10"
)

type ChangeAccountStatusRequest struct {
	AccountNumber string        `json:"account_number" validate:"required"`
	AccountStatus AccountStatus `json:"account_status" validate:"required"`
}

func (req ChangeAccountStatusRequest) Validate() error {
	v := validator.New()
	return v.Struct(req)
}

type ChangeAccountTypeRequest struct {
	AccountNumber string      `json:"account_number" validate:"required"`
	AccountType   AccountType `json:"account_type" validate:"required"`
}

func (req ChangeAccountTypeRequest) Validate() error {
	v := validator.New()
	return v.Struct(req)
}

type CreateAccountRequest struct {
	CustomerId  int64       `json:"customer_id" validate:"required"`
	AccountType AccountType `json:"account_type" validate:"required"`
	Currency    string      `json:"currency" validate:"required,len=3"`
}

func (req CreateAccountRequest) Validate() error {
	v := validator.New()
	return v.Struct(req)
}

type CreateAccountWithCustomerRequest struct {
	CustomerName         string      `json:"customer_name" validate:"required,max=150"`
	IdentificationNumber string      `json:"identification_number" validate:"required,max=30"`
	Email                string      `json:"email" validate:"required,email,min=10,max=150"`
	Phone                string      `json:"phone" validate:"required,max=20"`
	Address              string      `json:"address" validate:"required,max=200"`
	AccountType          AccountType `json:"account_type_id" validate:"required"`
	Currency             string      `json:"currency" validate:"required,max=3"`
}

func (req CreateAccountWithCustomerRequest) Validate() error {
	v := validator.New()
	return v.Struct(req)
}
