package controllers

import (
	"github.com/ortizdavid/go-bank-core-api/core/repositories/reports"
	"gorm.io/gorm"
	"net/http"
)

type TransactionReportController struct {
	repositoryRepository *repositories.TransactionReportRepository
}

func NewTransactionReportController(db *gorm.DB) *TransactionReportController {
	return &TransactionReportController{
		repositoryRepository: repositories.NewTransactionReportRepository(db),
	}
}

func (*TransactionReportController) RegisterRoutes(router *http.ServeMux) {

}
