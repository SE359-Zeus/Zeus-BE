package seeder

import (
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"zeus-scm-service/internal/models"
)

func buildProcurementBundles(suppliers []models.Supplier, mappingsBySupplier map[uuid.UUID][]models.SkuMapping) []procurementBundle {
	baseTime := time.Date(2026, 5, 19, 9, 0, 0, 0, time.UTC)
	carriers := []string{"TransGlobal Freight", "Northwind Logistics", "Skyline Couriers", "BlueRoute Cargo", "Apex Transit"}
	lineCount := 3
	if len(suppliers) < 5 {
		lineCount = 2
	}

	templates := []struct {
		poStatus       models.POStatus
		shipmentStatus models.ShipmentStatus
		grStatus       models.GRStatus
		includeDocs    bool
	}{
		{poStatus: models.POStatusDraft, includeDocs: false},
		{poStatus: models.POStatusApproved, shipmentStatus: models.ShipmentStatusScheduled, grStatus: models.GRStatusPending, includeDocs: true},
		{poStatus: models.POStatusInTransit, shipmentStatus: models.ShipmentStatusInTransit, grStatus: models.GRStatusPending, includeDocs: true},
		{poStatus: models.POStatusReceived, shipmentStatus: models.ShipmentStatusDelivered, grStatus: models.GRStatusComplete, includeDocs: true},
		{poStatus: models.POStatusPartial, shipmentStatus: models.ShipmentStatusDelayed, grStatus: models.GRStatusDiscrepancy, includeDocs: true},
	}

	bundles := make([]procurementBundle, 0, len(templates))
	for i, supplier := range suppliers {
		if i >= len(templates) {
			break
		}

		mappings := mappingsBySupplier[supplier.ID]
		if len(mappings) == 0 {
			continue
		}

		itemCount := lineCount
		if len(mappings) < itemCount {
			itemCount = len(mappings)
		}

		lineItems := make([]procurementLineSpec, 0, itemCount)
		for j := 0; j < itemCount; j++ {
			mapping := mappings[(i+j)%len(mappings)]
			orderedQty := 40 + (i * 15) + (j * 10)
			receivedQty := 0
			defectiveQty := 0
			agingSensitive := false
			var productionDate *time.Time
			if templates[i].poStatus == models.POStatusReceived {
				receivedQty = orderedQty
			}
			if templates[i].poStatus == models.POStatusPartial {
				receivedQty = orderedQty - maxInt(5, orderedQty/4)
				defectiveQty = maxInt(1, orderedQty/10)
				if j == 0 {
					agingSensitive = true
					aged := baseTime.AddDate(-4, 0, 0)
					productionDate = &aged
				}
			}

			lineItems = append(lineItems, procurementLineSpec{
				SKU:            mapping.SKU,
				Description:    mapping.Name,
				OrderedQty:     orderedQty,
				ReceivedQty:    receivedQty,
				DefectiveQty:   defectiveQty,
				UnitPrice:      mapping.UnitPrice,
				AgingSensitive: agingSensitive,
				ProductionDate: productionDate,
			})
		}

		bundle := procurementBundle{
			POID:         fmt.Sprintf("PO-2026-%03d", i+1),
			VendorID:     supplier.ID,
			POStatus:     templates[i].poStatus,
			PaymentTerms: "Net 30",
			LineItems:    lineItems,
		}

		if templates[i].includeDocs {
			bundle.ShipmentID = fmt.Sprintf("SHP-2026-%03d", i+1)
			bundle.ShipmentStatus = templates[i].shipmentStatus
			bundle.Carrier = carriers[i]
			bundle.TrackingNo = fmt.Sprintf("TRK-2026-%03d", i+1)
			bundle.Origin = "Shenzhen, CN"
			bundle.ShipDate = baseTime.AddDate(0, 0, 3+i*2)
			bundle.ETA = baseTime.AddDate(0, 0, 7+i*2)

			// Seed Goods Receipt (GR) only if the shipment is delivered/received
			if templates[i].shipmentStatus == models.ShipmentStatusDelivered || templates[i].grStatus == models.GRStatusComplete || templates[i].grStatus == models.GRStatusDiscrepancy {
				bundle.GRID = fmt.Sprintf("GR-2026-%03d", i+1)
				bundle.GRStatus = templates[i].grStatus
				bundle.ArrivalDate = baseTime.AddDate(0, 0, 8+i*2)
				bundle.OperatorID = stableUUID("user:scm-operator").String()
			}
		}

		bundles = append(bundles, bundle)
	}

	return bundles
}

func seedProcurementBundle(db *gorm.DB, bundle procurementBundle) {
	totalValue := 0.0
	for _, item := range bundle.LineItems {
		totalValue += float64(item.OrderedQty) * item.UnitPrice
	}

	po := models.PurchaseOrder{
		ID:           bundle.POID,
		VendorID:     bundle.VendorID,
		Status:       bundle.POStatus,
		TotalValue:   math.Round(totalValue*100) / 100,
		PaymentTerms: bundle.PaymentTerms,
	}
	db.FirstOrCreate(&po, models.PurchaseOrder{ID: po.ID})

	for idx, item := range bundle.LineItems {
		poLine := models.POLineItem{
			ID:          uuid.NewMD5(uuid.Nil, []byte(fmt.Sprintf("%s|po-line|%d", bundle.POID, idx))),
			POID:        bundle.POID,
			SKU:         item.SKU,
			Description: item.Description,
			OrderedQty:  item.OrderedQty,
			ReceivedQty: item.ReceivedQty,
			UnitPrice:   item.UnitPrice,
		}
		db.FirstOrCreate(&poLine, models.POLineItem{ID: poLine.ID})
	}

	if bundle.ShipmentID != "" {
		shipment := models.Shipment{
			ID:         bundle.ShipmentID,
			PORef:      bundle.POID,
			SupplierID: bundle.VendorID,
			Status:     bundle.ShipmentStatus,
			Carrier:    bundle.Carrier,
			TrackingNo: bundle.TrackingNo,
			Origin:     bundle.Origin,
			ShipDate:   bundle.ShipDate,
			ETA:        bundle.ETA,
		}
		db.FirstOrCreate(&shipment, models.Shipment{ID: shipment.ID})

		for idx, item := range bundle.LineItems {
			shipmentItem := models.ShipmentItem{
				ID:          uuid.NewMD5(uuid.Nil, []byte(fmt.Sprintf("%s|shipment-item|%d", bundle.ShipmentID, idx))),
				ShipmentID:  bundle.ShipmentID,
				SKU:         item.SKU,
				Description: item.Description,
				Qty:         item.OrderedQty,
			}
			db.FirstOrCreate(&shipmentItem, models.ShipmentItem{ID: shipmentItem.ID})
		}
	}

	if bundle.GRID != "" {
		gr := models.GoodsReceipt{
			ID:          bundle.GRID,
			PORef:       bundle.POID,
			VendorID:    bundle.VendorID,
			Status:      bundle.GRStatus,
			ArrivalDate: bundle.ArrivalDate,
			OperatorID:  bundle.OperatorID,
		}
		db.FirstOrCreate(&gr, models.GoodsReceipt{ID: gr.ID})

		for idx, item := range bundle.LineItems {
			line := models.GRLineItem{
				ID:             uuid.NewMD5(uuid.Nil, []byte(fmt.Sprintf("%s|gr-line|%d", bundle.GRID, idx))),
				GRID:           bundle.GRID,
				SKU:            item.SKU,
				Name:           item.Description,
				OrderedQty:     item.OrderedQty,
				AgingSensitive: item.AgingSensitive,
			}
			if item.ReceivedQty > 0 {
				received := item.ReceivedQty
				line.ReceivedQty = &received
			}
			if item.DefectiveQty > 0 {
				defective := item.DefectiveQty
				line.DefectiveQty = &defective
			}
			if item.AgingSensitive && item.ProductionDate != nil {
				line.ProductionDate = item.ProductionDate
				line.AgingLabel = agingLabelForProductionDate(*item.ProductionDate)
			}
			db.FirstOrCreate(&line, models.GRLineItem{ID: line.ID})
		}
	}
}

func agingLabelForProductionDate(productionDate time.Time) string {
	if time.Since(productionDate) > 3*365*24*time.Hour {
		return "Over-Age"
	}
	return "Within Threshold"
}

func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}
