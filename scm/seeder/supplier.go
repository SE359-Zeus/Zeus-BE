package seeder

import (
	"zeus-scm-service/internal/models"

	"gorm.io/gorm"
)

func seedSuppliers(db *gorm.DB, count int) []models.Supplier {
	var existingCount int64
	if err := db.Model(&models.Supplier{}).Count(&existingCount).Error; err == nil && existingCount > 0 {
		var suppliers []models.Supplier
		db.Find(&suppliers)
		return suppliers
	}

	type supplierSpec struct {
		Name         string
		Contact      string
		LeadTimeDays int
		QualityScore float64
		OnTimeRate   float64
	}

	specs := []supplierSpec{
		{Name: "Northwind Component Supply", Contact: "orders@northwind.example", LeadTimeDays: 7, QualityScore: 97.5, OnTimeRate: 98.1},
		{Name: "Apex Industrial Parts", Contact: "sales@apex.example", LeadTimeDays: 10, QualityScore: 95.2, OnTimeRate: 96.4},
		{Name: "BluePeak Electronics", Contact: "procurement@bluepeak.example", LeadTimeDays: 12, QualityScore: 94.8, OnTimeRate: 95.0},
		{Name: "Orion Component Works", Contact: "contact@orion.example", LeadTimeDays: 15, QualityScore: 92.6, OnTimeRate: 93.3},
		{Name: "Vertex Supply Group", Contact: "orders@vertex.example", LeadTimeDays: 18, QualityScore: 90.4, OnTimeRate: 91.7},
	}

	var suppliers []models.Supplier
	limit := count
	if limit > len(specs) {
		limit = len(specs)
	}
	for i := 0; i < limit; i++ {
		spec := specs[i]
		s := models.Supplier{
			ID:           stableUUID("supplier:" + spec.Name),
			Name:         spec.Name,
			Contact:      spec.Contact,
			Tier:         models.SupplierTier2,
			LeadTimeDays: spec.LeadTimeDays,
			QualityScore: spec.QualityScore,
			OnTimeRate:   spec.OnTimeRate,
		}
		db.Where("id = ?", s.ID).Assign(s).FirstOrCreate(&s)
		suppliers = append(suppliers, s)
	}
	return suppliers
}
