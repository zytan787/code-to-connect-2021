package internal

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/zytan787/code-to-connect-2021/api"
	"net/http"
)

type MainHandler struct {
	PortfolioLoader   *PortfolioLoader
	CompressionEngine *CompressionEngine
	EventGenerator    *EventGenerator
	DataChecker       *DataChecker
}

func NewMainHandler() *MainHandler {
	return &MainHandler{
		PortfolioLoader:   &PortfolioLoader{},
		CompressionEngine: &CompressionEngine{},
		EventGenerator:    &EventGenerator{},
		DataChecker:       &DataChecker{},
	}
}

func (handler *MainHandler) CompressTrades(c *gin.Context) {
	var req api.CompressTradesReq
	var resp api.CompressTradesResp

	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Error = err.Error()
		c.JSON(http.StatusBadRequest, resp)
		return
	}

	rawTrades, err := handler.DecodeInputFiles(req.InputFiles)
	if err != nil {
		resp.Error = fmt.Sprintf("Error in DecodeInputFiles due to: %s", err.Error())
		c.JSON(http.StatusBadRequest, resp)
		return
	}

	handler.LoadPortfolio(rawTrades)

	err = handler.GenerateCompressionResults()
	if err != nil {
		resp.Error = fmt.Sprintf("Error in GenerateCompressionResults due to: %s", err.Error())
		c.JSON(http.StatusInternalServerError, resp)
		return
	}

	err = handler.GenerateProposals()
	if err != nil {
		resp.Error = fmt.Sprintf("Error in GenerateProposals due to: %s", err.Error())
		c.JSON(http.StatusInternalServerError, resp)
		return
	}

	err = handler.CheckData()
	if err != nil {
		resp.Error = fmt.Sprintf("Error in CheckData due to: %s", err.Error())
		c.JSON(http.StatusInternalServerError, resp)
		return
	}

	err = handler.GenerateBookLevelCompressionResults()
	if err != nil {
		resp.Error = fmt.Sprintf("Error in GenerateBookLevelCompressionResults due to: %s", err.Error())
		c.JSON(http.StatusInternalServerError, resp)
		return
	}

	exclusionCSV, err := handler.GetExcludedTradesAsCSV()
	if err != nil {
		resp.Error = fmt.Sprintf("Error in GetExcludedTradesAsCSV due to: %s", err.Error())
		c.JSON(http.StatusInternalServerError, resp)
		return
	}
	resp.Exclusion = exclusionCSV

	compressionReport, err := handler.GetCompressionReportAsCSV()
	if err != nil {
		resp.Error = fmt.Sprintf("Error in GetCompressionReportAsCSV due to: %s", err.Error())
		c.JSON(http.StatusInternalServerError, resp)
		return
	}
	resp.CompressionReport = compressionReport

	compressionReportBookLevel, err := handler.GetCompressionReportBookLevelAsCSV()
	if err != nil {
		resp.Error = fmt.Sprintf("Error in GetCompressionReportBookLevelAsCSV due to: %s", err.Error())
		c.JSON(http.StatusInternalServerError, resp)
		return
	}
	resp.CompressionReportBookLevel = compressionReportBookLevel

	proposals, err := handler.GetProposalsAsCSV()
	if err != nil {
		resp.Error = fmt.Sprintf("Error in GetProposalsAsCSV due to: %s", err.Error())
		c.JSON(http.StatusInternalServerError, resp)
		return
	}
	resp.Proposals = proposals

	dataCheckResults, err := handler.GetDataCheckResultsAsCSV()
	if err != nil {
		resp.Error = fmt.Sprintf("Error in GetDataCheckResultsAsCSV due to: %s", err.Error())
		c.JSON(http.StatusInternalServerError, resp)
		return
	}
	resp.DataCheck = dataCheckResults

	c.JSON(http.StatusOK, resp)
}
