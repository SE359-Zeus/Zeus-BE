package handler

import (
	"encoding/csv"
	"net/http"
	"strconv"
	"strings"
	"zeus-scm-service/internal/exception"
	"zeus-scm-service/internal/models"
	"zeus-scm-service/internal/pagination"
	"zeus-scm-service/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type VendorHandler struct {
	svc service.IVendorService
}

func NewVendorHandler(svc service.IVendorService) *VendorHandler {
	return &VendorHandler{svc: svc}
}

func (h *VendorHandler) GetOptimalSupplier(c *gin.Context) {
	sku := c.Query("sku")
	if sku == "" {
		exception.WriteError(c, exception.ErrInvalidInput.WithMessage("sku query param required"))
		return
	}
	supplier, mapping, err := h.svc.GetOptimalSupplier(c.Request.Context(), sku)
	if err != nil {
		if appErr := exception.Resolve(err); appErr != nil {
			exception.WriteError(c, appErr)
			return
		}
		exception.WriteError(c, exception.ErrInternal)
		return
	}
	writeJSON(c, 200, gin.H{
		"supplier": supplier,
		"mapping":  mapping,
	})
}

func (h *VendorHandler) UpdateSupplierMetrics(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		exception.WriteError(c, exception.ErrInvalidResourceID.WithMessage("invalid supplier id"))
		return
	}
	if err := h.svc.UpdateSupplierMetrics(c.Request.Context(), id); err != nil {
		exception.WriteError(c, exception.ErrInternal.WithError(err))
		return
	}
	writeJSON(c, 200, gin.H{"message": "metrics updated"})
}

func (h *VendorHandler) ListSuppliers(c *gin.Context) {
	params := parsePaginationParams(c)
	tier := c.Query("tier")
	q := c.Query("q")

	suppliers, meta, err := h.svc.ListSuppliers(c.Request.Context(), tier, params, q)
	if err != nil {
		exception.WriteError(c, exception.Resolve(err))
		return
	}
	writeJSON(c, 200, pagination.Response{
		Data:       suppliers,
		Pagination: *meta,
	})
}

func (h *VendorHandler) GetSupplierMetrics(c *gin.Context) {
	count, rate, err := h.svc.GetSupplierMetrics(c.Request.Context())
	if err != nil {
		exception.WriteError(c, exception.Resolve(err))
		return
	}
	writeJSON(c, 200, gin.H{
		"total_active_suppliers": count,
		"on_time_delivery_rate":  rate,
	})
}

type createSupplierRequest struct {
	Name         string `json:"name" binding:"required"`
	Contact      string `json:"contact" binding:"required"`
	Tier         string `json:"tier" binding:"required"`
	LeadTimeDays int    `json:"lead_time_days" binding:"required,min=1"`
}

func (h *VendorHandler) CreateSupplier(c *gin.Context) {
	var req createSupplierRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		exception.WriteError(c, exception.ErrInvalidBody)
		return
	}

	tierVal := models.SupplierTier(req.Tier)
	switch strings.ToLower(req.Tier) {
	case "tier 1", "tier1":
		tierVal = models.SupplierTier1
	case "tier 2", "tier2":
		tierVal = models.SupplierTier2
	case "tier 3", "tier3":
		tierVal = models.SupplierTier3
	}

	supplier := &models.Supplier{
		ID:           uuid.New(),
		Name:         req.Name,
		Contact:      req.Contact,
		Tier:         tierVal,
		LeadTimeDays: req.LeadTimeDays,
		QualityScore: 100.0,
		OnTimeRate:   100.0,
	}

	if err := h.svc.CreateSupplier(c.Request.Context(), supplier); err != nil {
		exception.WriteError(c, exception.Resolve(err))
		return
	}
	writeJSON(c, 201, supplier)
}

type createSkuMappingRequest struct {
	SKU          string  `json:"sku" binding:"required"`
	Name         string  `json:"name" binding:"required"`
	UnitPrice    float64 `json:"unit_price" binding:"required,gt=0"`
	LeadTimeDays int     `json:"lead_time_days" binding:"required,min=1"`
	MinOrderQty  int     `json:"min_order_qty" binding:"required,min=1"`
}

func (h *VendorHandler) CreateSkuMapping(c *gin.Context) {
	idStr := c.Param("id")
	supplierID, err := uuid.Parse(idStr)
	if err != nil {
		exception.WriteError(c, exception.ErrInvalidResourceID.WithMessage("invalid supplier id"))
		return
	}

	var req createSkuMappingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		exception.WriteError(c, exception.ErrInvalidBody)
		return
	}

	mapping := &models.SkuMapping{
		ID:           uuid.New(),
		SupplierID:   supplierID,
		SKU:          req.SKU,
		Name:         req.Name,
		UnitPrice:    req.UnitPrice,
		LeadTimeDays: req.LeadTimeDays,
		MinOrderQty:  req.MinOrderQty,
	}

	if err := h.svc.CreateSkuMapping(c.Request.Context(), mapping); err != nil {
		exception.WriteError(c, exception.Resolve(err))
		return
	}
	writeJSON(c, 201, mapping)
}

func (h *VendorHandler) ExportSuppliersReport(c *gin.Context) {
	ctx := c.Request.Context()
	suppliers, err := h.svc.FindAllSuppliersWithMappings(ctx)
	if err != nil {
		exception.WriteError(c, exception.Resolve(err))
		return
	}

	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Disposition", `attachment; filename="suppliers_report.csv"`)
	c.Header("Content-Type", "text/csv")
	c.Header("Transfer-Encoding", "chunked")

	c.Writer.WriteHeader(http.StatusOK)

	csvWriter := csv.NewWriter(c.Writer)
	header := []string{
		"Supplier ID", "Supplier Name", "Contact", "Tier", "Lead Time Days",
		"Quality Score", "On-Time Rate", "Mapped SKU", "Item Name",
		"Unit Price", "Mapping Lead Time Days", "Min Order Qty",
	}
	if err := csvWriter.Write(header); err != nil {
		return
	}
	csvWriter.Flush()
	c.Writer.Flush()

	for _, s := range suppliers {
		if len(s.SkuMappings) == 0 {
			row := []string{
				s.ID.String(), s.Name, s.Contact, string(s.Tier), strconv.Itoa(s.LeadTimeDays),
				strconv.FormatFloat(s.QualityScore, 'f', 2, 64), strconv.FormatFloat(s.OnTimeRate, 'f', 2, 64),
				"", "", "", "", "",
			}
			if err := csvWriter.Write(row); err != nil {
				return
			}
		} else {
			for _, m := range s.SkuMappings {
				row := []string{
					s.ID.String(), s.Name, s.Contact, string(s.Tier), strconv.Itoa(s.LeadTimeDays),
					strconv.FormatFloat(s.QualityScore, 'f', 2, 64), strconv.FormatFloat(s.OnTimeRate, 'f', 2, 64),
					m.SKU, m.Name, strconv.FormatFloat(m.UnitPrice, 'f', 2, 64),
					strconv.Itoa(m.LeadTimeDays), strconv.Itoa(m.MinOrderQty),
				}
				if err := csvWriter.Write(row); err != nil {
					return
				}
			}
		}
		csvWriter.Flush()
		c.Writer.Flush()
	}
}

func (h *VendorHandler) GetShortageSummary(c *gin.Context) {
	summaries, err := h.svc.GetShortageSummary(c.Request.Context())
	if err != nil {
		exception.WriteError(c, exception.Resolve(err))
		return
	}
	writeJSON(c, 200, summaries)
}
