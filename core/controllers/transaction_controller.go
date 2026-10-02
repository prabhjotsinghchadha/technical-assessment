package controllers

import (
	"fmt"
	"net/http"

	"github.com/ortizdavid/go-bank-core-api/common/config"
	"github.com/ortizdavid/go-bank-core-api/common/helpers"
	entities "github.com/ortizdavid/go-bank-core-api/core/entities/transactions"
	"github.com/ortizdavid/go-bank-core-api/core/services/transactions"
	"github.com/ortizdavid/go-nopain/httputils"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type TransactionController struct {
	service     *services.TransactionService
	infoLogger  *zap.Logger
	errorLogger *zap.Logger
}

func NewTransactionController(db *gorm.DB) *TransactionController {
	return &TransactionController{
		service:     services.NewTransactionService(db),
		infoLogger:  config.NewLogger("transactions-info.log"),
		errorLogger: config.NewLogger("transactions-error.log"),
	}
}

func (ctrl *TransactionController) RegisterRoutes(router *http.ServeMux) {
	router.HandleFunc("GET /api/transactions", ctrl.getAllTransactions)
	router.HandleFunc("GET /api/transactions/account/{account_number}", ctrl.getAccountTransactions)
	router.HandleFunc("POST /api/transactions/deposit", ctrl.deposit)
	router.HandleFunc("POST /api/transactions/withdraw", ctrl.withdraw)
	router.HandleFunc("POST /api/transactions/transfer", ctrl.transfer)
}

func (ctrl *TransactionController) getAllTransactions(w http.ResponseWriter, r *http.Request) {
	params, err := helpers.GetPaginationParams(r)
	if err != nil {
		helpers.HandleHttpErrors(w, err)
		return
	}
	transactions, err := ctrl.service.GetAllTransactions(r, r.Context(), params)
	if err != nil {
		helpers.HandleHttpErrors(w, err)
		return
	}
	httputils.WriteJsonSimple(w, http.StatusOK, transactions)
}

func (ctrl *TransactionController) getAccountTransactions(w http.ResponseWriter, r *http.Request) {
	params, err := helpers.GetPaginationParams(r)
	if err != nil {
		helpers.HandleHttpErrors(w, err)
		return
	}
	accountNumber := r.PathValue("account_number")
	transactions, err := ctrl.service.GetAccountTransactions(r, r.Context(), accountNumber, params)
	if err != nil {
		helpers.HandleHttpErrors(w, err)
		return
	}
	httputils.WriteJsonSimple(w, http.StatusOK, transactions)
}

func (ctrl *TransactionController) deposit(w http.ResponseWriter, r *http.Request) {
	var request entities.DepositRequest
	transaction, err := ctrl.service.Deposit(r, r.Context(), request)
	if err != nil {
		helpers.HandleHttpErrors(w, err)
		ctrl.errorLogger.Error(err.Error())
		return
	}
	ctrl.infoLogger.Info(fmt.Sprintf("Deposit '%s' completed", transaction.Code))
	httputils.WriteJson(w, http.StatusCreated, transaction)
}

func (ctrl *TransactionController) withdraw(w http.ResponseWriter, r *http.Request) {
	var request entities.WithdrawRequest
	transaction, err := ctrl.service.Withdraw(r, r.Context(), request)
	if err != nil {
		helpers.HandleHttpErrors(w, err)
		ctrl.errorLogger.Error(err.Error())
		return
	}
	ctrl.infoLogger.Info(fmt.Sprintf("Withdrawal '%s' completed", transaction.Code))
	httputils.WriteJson(w, http.StatusCreated, transaction)
}

func (ctrl *TransactionController) transfer(w http.ResponseWriter, r *http.Request) {
	var request entities.TransferNumberRequest
	transaction, err := ctrl.service.Transfer(r, r.Context(), request)
	if err != nil {
		helpers.HandleHttpErrors(w, err)
		ctrl.errorLogger.Error(err.Error())
		return
	}
	ctrl.infoLogger.Info(fmt.Sprintf("Transfer '%s' completed", transaction.Code))
	httputils.WriteJson(w, http.StatusCreated, transaction)
}
