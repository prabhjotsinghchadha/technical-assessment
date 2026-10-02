package controllers

import (
	"github.com/ortizdavid/go-bank-core-api/core/repositories/reports"
	"gorm.io/gorm"
	"net/http"
)

type CustomerReportController struct {
	repository *repositories.CustomerReportRepository
}

func NewCustomerReportController(db *gorm.DB) *CustomerReportController {
	return &CustomerReportController{
		repository: repositories.NewCustomerReportRepository(db),
	}
}

func (*CustomerReportController) RegisterRoutes(router *http.ServeMux) {

}
