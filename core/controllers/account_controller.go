package controllers

import (
	"fmt"
	"net/http"

	"github.com/ortizdavid/go-bank-core-api/common/config"
	"github.com/ortizdavid/go-bank-core-api/common/helpers"
	entities "github.com/ortizdavid/go-bank-core-api/core/entities/accounts"
	"github.com/ortizdavid/go-bank-core-api/core/services/accounts"
	"github.com/ortizdavid/go-nopain/httputils"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type AccountController struct {
	service     *services.AccountService
	infoLogger  *zap.Logger
	errorLogger *zap.Logger
}

func NewAccountController(db *gorm.DB) *AccountController {
	return &AccountController{
		service:     services.NewAccountService(db),
		infoLogger:  config.NewLogger("accounts-info.log"),
		errorLogger: config.NewLogger("accounts-error.log"),
	}
}

func (ctrl *AccountController) RegisterRoutes(router *http.ServeMux) {
	router.HandleFunc("GET /api/accounts", ctrl.getAllAccounts)
	router.HandleFunc("GET /api/accounts/{id}", ctrl.getAccountById)
	router.HandleFunc("GET /api/accounts/by-number/{account_number}", ctrl.getAccountByNumber)
	router.HandleFunc("POST /api/accounts", ctrl.createAccount)
	router.HandleFunc("PUT /api/accounts/change-status", ctrl.changeAccountStatus)
}

func (ctrl *AccountController) getAllAccounts(w http.ResponseWriter, r *http.Request) {
	params, err := helpers.GetPaginationParams(r)
	if err != nil {
		helpers.HandleHttpErrors(w, err)
		return
	}
	accounts, err := ctrl.service.GetAllAccounts(r, r.Context(), params)
	if err != nil {
		helpers.HandleHttpErrors(w, err)
		return
	}
	httputils.WriteJsonSimple(w, http.StatusOK, accounts)
}

func (ctrl *AccountController) getAccountById(w http.ResponseWriter, r *http.Request) {
	id, err := helpers.PathInt64(r, "id")
	if err != nil {
		helpers.HandleHttpErrors(w, err)
		return
	}
	account, err := ctrl.service.GetAccountById(r.Context(), id)
	if err != nil {
		helpers.HandleHttpErrors(w, err)
		return
	}
	httputils.WriteJson(w, http.StatusOK, account)
}

func (ctrl *AccountController) getAccountByNumber(w http.ResponseWriter, r *http.Request) {
	accountNumber := r.PathValue("account_number")
	account, err := ctrl.service.GetAccountByNumber(r.Context(), accountNumber)
	if err != nil {
		helpers.HandleHttpErrors(w, err)
		return
	}
	httputils.WriteJson(w, http.StatusOK, account)
}

func (ctrl *AccountController) createAccount(w http.ResponseWriter, r *http.Request) {
	var request entities.CreateAccountRequest
	account, err := ctrl.service.CreateAccount(r, r.Context(), request)
	if err != nil {
		helpers.HandleHttpErrors(w, err)
		ctrl.errorLogger.Error(err.Error())
		return
	}
	msg := fmt.Sprintf("Account '%s' created", account.AccountNumber)
	ctrl.infoLogger.Info(msg)
	httputils.WriteJson(w, http.StatusCreated, account)
}

func (ctrl *AccountController) changeAccountStatus(w http.ResponseWriter, r *http.Request) {
	var request entities.ChangeAccountStatusRequest
	err := ctrl.service.ChangeAccountStatus(r, r.Context(), request)
	if err != nil {
		helpers.HandleHttpErrors(w, err)
		ctrl.errorLogger.Error(err.Error())
		return
	}
	msg := fmt.Sprintf("Account '%s' status changed", request.AccountNumber)
	ctrl.infoLogger.Info(msg)
	httputils.WriteJson(w, http.StatusOK, msg)
}
