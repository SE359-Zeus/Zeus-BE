package service

import (
	"context"
	"fmt"
	"strings"
	"zeus-mrp-service/internal/models"

	"github.com/google/uuid"
)

func (s *ProductionService) GetAssemblies(ctx context.Context) ([]models.AssemblyResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var (
		boms []models.BomEntry
		err  error
	)
	if s.cache != nil {
		boms, err = s.cache.GetAllBOMs(ctx, s.repo.GetAllBOMs)
	} else {
		boms, err = s.repo.GetAllBOMs(ctx)
	}
	if err != nil {
		return nil, err
	}

	assemblies, err := s.groupAssemblies(ctx, boms)
	if err != nil {
		return nil, err
	}
	return s.hydrateAssemblyNames(ctx, assemblies), nil
}

func (s *ProductionService) GetAssembliesPage(ctx context.Context, page, per int) ([]models.AssemblyResponse, int, error) {
	if err := ctx.Err(); err != nil {
		return nil, 0, err
	}
	boms, total, err := s.repo.GetPagedBOMsByAssembly(ctx, page, per)
	if err != nil {
		return nil, 0, err
	}
	assemblies, err := s.groupAssemblies(ctx, boms)
	if err != nil {
		return nil, 0, err
	}
	return s.hydrateAssemblyNames(ctx, assemblies), total, nil
}

func (s *ProductionService) GetAssemblyByModelCode(ctx context.Context, modelCode string) (*models.AssemblyResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if modelCode == "" {
		return nil, fmt.Errorf("model code is required")
	}
	var (
		boms []models.BomEntry
		err  error
	)
	if s.cache != nil {
		boms, err = s.cache.GetBOMByModelCode(ctx, modelCode, s.repo.GetBOMByModelCode)
	} else {
		boms, err = s.repo.GetBOMByModelCode(ctx, modelCode)
	}
	if err != nil {
		return nil, err
	}
	if len(boms) == 0 {
		return nil, nil
	}
	comps := make([]models.ComponentReference, 0, len(boms))
	for _, e := range boms {
		sku, err := s.resolveComponentSKU(ctx, e.ComponentPartID)
		if err != nil {
			return nil, err
		}
		comps = append(comps, models.ComponentReference{
			SKU:      sku,
			Quantity: e.RequiredQuantityPerUnit,
		})
	}
	return &models.AssemblyResponse{
		ModelCode:  modelCode,
		Name:       s.resolveAssemblyName(ctx, modelCode),
		TotalParts: len(comps),
		Components: comps,
	}, nil
}

func (s *ProductionService) CreateAssembly(ctx context.Context, req models.CreateAssemblyRequest) (any, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if req.Name == "" {
		return nil, fmt.Errorf("assembly name is required")
	}

	// Reject duplicate component SKUs
	seen := map[string]struct{}{}
	entries := make([]models.BomEntry, 0, len(req.Components))
	for i, c := range req.Components {
		if c.Quantity <= 0 {
			return nil, fmt.Errorf("component[%d] quantity must be > 0", i)
		}
		if _, dup := seen[c.SKU]; dup {
			return nil, fmt.Errorf("component[%d] sku %s is duplicated in this request", i, c.SKU)
		}
		seen[c.SKU] = struct{}{}
		pid, err := s.resolveComponentPartID(ctx, c.SKU)
		if err != nil {
			return nil, fmt.Errorf("component[%d] %w", i, err)
		}
		entries = append(entries, models.BomEntry{
			ParentModelCode:         req.Name,
			ComponentPartID:         pid,
			RequiredQuantityPerUnit: c.Quantity,
		})
	}

	// replace existing BOM for this model
	if err := s.repo.DeleteBOMEntriesByModelCode(ctx, req.Name); err != nil {
		return nil, err
	}
	if err := s.repo.CreateBOMEntries(ctx, entries); err != nil {
		return nil, err
	}
	if s.cache != nil {
		_ = s.cache.InvalidateBOM(ctx, req.Name, uniquePartIDs(entries)...)
	}
	s.publishAudit(ctx, "CREATE", "mrp/assemblies/"+req.Name, "Created assembly "+req.Name)

	return req, nil
}

func (s *ProductionService) UpdateAssembly(ctx context.Context, id uuid.UUID, req models.UpdateAssemblyRequest) (any, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if id == uuid.Nil {
		return nil, fmt.Errorf("id must be a valid uuid")
	}

	// For this simple implementation we treat id as the model code UUID string
	modelCode := req.Name
	if modelCode == "" {
		return nil, fmt.Errorf("assembly name is required")
	}

	// rebuild entries
	entries := make([]models.BomEntry, 0, len(req.Components))
	for i, c := range req.Components {
		if c.Quantity <= 0 {
			return nil, fmt.Errorf("component[%d] quantity must be > 0", i)
		}
		pid, err := s.resolveComponentPartID(ctx, c.SKU)
		if err != nil {
			return nil, fmt.Errorf("component[%d] %w", i, err)
		}
		entries = append(entries, models.BomEntry{
			ParentModelCode:         modelCode,
			ComponentPartID:         pid,
			RequiredQuantityPerUnit: c.Quantity,
		})
	}

	if err := s.repo.DeleteBOMEntriesByModelCode(ctx, modelCode); err != nil {
		return nil, err
	}
	if err := s.repo.CreateBOMEntries(ctx, entries); err != nil {
		return nil, err
	}
	if s.cache != nil {
		_ = s.cache.InvalidateBOM(ctx, modelCode, uniquePartIDs(entries)...)
	}
	s.publishAudit(ctx, "UPDATE", "mrp/assemblies/"+modelCode, "Updated assembly "+modelCode)

	return req, nil
}

func (s *ProductionService) DeleteAssembly(ctx context.Context, id uuid.UUID) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if id == uuid.Nil {
		return fmt.Errorf("id must be a valid uuid")
	}
	// treat id as string model code
	modelCode := id.String()
	if err := s.repo.DeleteBOMEntriesByModelCode(ctx, modelCode); err != nil {
		return err
	}
	if s.cache != nil {
		_ = s.cache.InvalidateBOM(ctx, modelCode)
	}
	s.publishAudit(ctx, "DELETE", "mrp/assemblies/"+modelCode, "Deleted assembly "+modelCode)
	return nil
}

func (s *ProductionService) GetCatalog(ctx context.Context) ([]any, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var (
		boms []models.BomEntry
		err  error
	)
	if s.cache != nil {
		boms, err = s.cache.GetAllBOMs(ctx, s.repo.GetAllBOMs)
	} else {
		boms, err = s.repo.GetAllBOMs(ctx)
	}
	if err != nil {
		return nil, err
	}

	return catalogFromBOMs(boms), nil
}

func (s *ProductionService) GetWhereUsed(ctx context.Context, sku string) ([]any, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	pid, err := s.resolveComponentPartID(ctx, sku)
	if err != nil {
		return nil, err
	}
	var entries []models.BomEntry
	if s.cache != nil {
		entries, err = s.cache.GetWhereUsedByPartID(ctx, pid, s.repo.GetWhereUsedByPartID)
	} else {
		entries, err = s.repo.GetWhereUsedByPartID(ctx, pid)
	}
	if err != nil {
		return nil, err
	}
	return whereUsedFromBOMs(entries), nil
}

func (s *ProductionService) CreateCatalogPart(ctx context.Context, sku, description string, price float64) (*models.Part, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	sku = strings.TrimSpace(sku)
	if sku == "" {
		return nil, fmt.Errorf("sku cannot be empty")
	}
	if _, err := uuid.Parse(sku); err == nil {
		return nil, fmt.Errorf("sku must be a part number, not a UUID")
	}

	// Check if already exists
	existing, err := s.scmClient.GetPartCatalogBySKU(ctx, sku)
	if err == nil && existing != nil {
		return nil, fmt.Errorf("component with SKU %s already exists", sku)
	}
	part, err := s.scmClient.CreateCatalogPart(ctx, sku, description, price)
	if err != nil {
		return nil, err
	}
	s.publishAudit(ctx, "CREATE", "mrp/catalog/"+sku, "Created catalog component "+sku)
	return part, nil
}

func (s *ProductionService) UpdateCatalogPart(ctx context.Context, sku, description string, price float64) (*models.Part, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	sku = strings.TrimSpace(sku)
	if sku == "" {
		return nil, fmt.Errorf("sku cannot be empty")
	}
	if _, err := uuid.Parse(sku); err == nil {
		return nil, fmt.Errorf("sku must be a part number, not a UUID")
	}

	part, err := s.scmClient.UpdateCatalogPart(ctx, sku, description, price)
	if err != nil {
		return nil, err
	}
	s.publishAudit(ctx, "UPDATE", "mrp/catalog/"+sku, "Updated catalog component "+sku)
	return part, nil
}

func (s *ProductionService) DeleteCatalogPart(ctx context.Context, sku string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	sku = strings.TrimSpace(sku)
	if sku == "" {
		return fmt.Errorf("sku cannot be empty")
	}
	if _, err := uuid.Parse(sku); err == nil {
		return fmt.Errorf("sku must be a part number, not a UUID")
	}

	if err := s.scmClient.DeleteCatalogPart(ctx, sku); err != nil {
		return err
	}
	s.publishAudit(ctx, "DELETE", "mrp/catalog/"+sku, "Deleted catalog component "+sku)
	return nil
}

func (s *ProductionService) groupAssemblies(ctx context.Context, boms []models.BomEntry) ([]models.AssemblyResponse, error) {
	order := []string{}
	grouped := map[string][]models.ComponentReference{}
	for _, e := range boms {
		if _, seen := grouped[e.ParentModelCode]; !seen {
			order = append(order, e.ParentModelCode)
		}
		sku, err := s.resolveComponentSKU(ctx, e.ComponentPartID)
		if err != nil {
			return nil, err
		}
		grouped[e.ParentModelCode] = append(grouped[e.ParentModelCode], models.ComponentReference{
			SKU:      sku,
			Quantity: e.RequiredQuantityPerUnit,
		})
	}

	result := make([]models.AssemblyResponse, 0, len(order))
	for _, model := range order {
		comps := grouped[model]
		result = append(result, models.AssemblyResponse{
			ModelCode:  model,
			Name:       model,
			TotalParts: len(comps),
			Components: comps,
		})
	}
	return result, nil
}

func (s *ProductionService) hydrateAssemblyNames(ctx context.Context, assemblies []models.AssemblyResponse) []models.AssemblyResponse {
	if s == nil || s.scmClient == nil {
		return assemblies
	}
	resolved := make(map[string]string, len(assemblies))
	for i := range assemblies {
		code := assemblies[i].ModelCode
		if name, ok := resolved[code]; ok {
			assemblies[i].Name = name
			continue
		}
		name := s.resolveAssemblyName(ctx, code)
		resolved[code] = name
		assemblies[i].Name = name
	}
	return assemblies
}

func (s *ProductionService) resolveAssemblyName(ctx context.Context, modelCode string) string {
	if s == nil || s.scmClient == nil {
		return modelCode
	}
	model, err := s.scmClient.GetProductModelByCode(ctx, modelCode)
	if err != nil || model == nil {
		return modelCode
	}
	if name := strings.TrimSpace(model.ModelName); name != "" {
		return name
	}
	return modelCode
}

func (s *ProductionService) resolveComponentPartID(ctx context.Context, sku string) (uuid.UUID, error) {
	sku = strings.TrimSpace(sku)
	if sku == "" {
		return uuid.Nil, fmt.Errorf("component sku is required")
	}
	if _, err := uuid.Parse(sku); err == nil {
		return uuid.Nil, fmt.Errorf("sku must be a part number, not a UUID")
	}
	if s == nil || s.scmClient == nil {
		return uuid.Nil, fmt.Errorf("component sku %s could not be resolved without SCM client", sku)
	}
	part, err := s.scmClient.GetPartCatalogBySKU(ctx, sku)
	if err != nil {
		return uuid.Nil, err
	}
	if part == nil || part.ID == uuid.Nil {
		return uuid.Nil, fmt.Errorf("component sku %s not found", sku)
	}
	return part.ID, nil
}

func (s *ProductionService) resolveComponentSKU(ctx context.Context, partID uuid.UUID) (string, error) {
	if partID == uuid.Nil {
		return "", fmt.Errorf("component part id is required")
	}
	if s == nil || s.scmClient == nil {
		return partID.String(), nil
	}
	part, err := s.scmClient.GetPartCatalogByID(ctx, partID)
	if err != nil {
		return "", err
	}
	if part == nil || strings.TrimSpace(part.SKU) == "" {
		return "", fmt.Errorf("component part %s not found", partID.String())
	}
	return part.SKU, nil
}

func catalogFromBOMs(boms []models.BomEntry) []any {
	parentSet := map[string]struct{}{}
	for _, e := range boms {
		parentSet[e.ParentModelCode] = struct{}{}
	}
	res := make([]any, 0, len(parentSet))
	for p := range parentSet {
		res = append(res, map[string]any{"model_code": p})
	}
	return res
}

func whereUsedFromBOMs(entries []models.BomEntry) []any {
	res := make([]any, 0, len(entries))
	for _, e := range entries {
		res = append(res, map[string]any{"parent_model": e.ParentModelCode, "required_qty": e.RequiredQuantityPerUnit})
	}
	return res
}

func uniquePartIDs(entries []models.BomEntry) []uuid.UUID {
	seen := make(map[uuid.UUID]struct{}, len(entries))
	result := make([]uuid.UUID, 0, len(entries))
	for _, entry := range entries {
		if entry.ComponentPartID == uuid.Nil {
			continue
		}
		if _, ok := seen[entry.ComponentPartID]; ok {
			continue
		}
		seen[entry.ComponentPartID] = struct{}{}
		result = append(result, entry.ComponentPartID)
	}
	return result
}
