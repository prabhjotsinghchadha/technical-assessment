package controllers

import (
	"github.com/ortizdavid/go-bank-core-api/core/repositories/reports"
	"gorm.io/gorm"
	"net/http"
)

type AccountReportController struct {
	repository *repositories.AccountReportRepository
}

func NewAccountReportController(db *gorm.DB) *AccountReportController {
	return &AccountReportController{
		repository: repositories.NewAccountReportRepository(db),
	}
}

func (*AccountReportController) RegisterRoutes(router *http.ServeMux) {

}
