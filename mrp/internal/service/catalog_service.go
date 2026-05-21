package service

import (
	"context"
	"fmt"
	"zeus-mrp-service/internal/models"

	"github.com/google/uuid"
)

func (s *ProductionService) GetAssemblies(ctx context.Context) ([]any, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	boms, err := s.repo.GetAllBOMs(ctx)
	if err != nil {
		return nil, err
	}

	// group by parent model code
	grouped := map[string][]models.ComponentReference{}
	for _, e := range boms {
		grouped[e.ParentModelCode] = append(grouped[e.ParentModelCode], models.ComponentReference{
			SKU:      e.ComponentPartID.String(),
			Quantity: e.RequiredQuantityPerUnit,
		})
	}

	result := make([]any, 0, len(grouped))
	for model, comps := range grouped {
		result = append(result, models.CreateAssemblyRequest{
			Name:       model,
			Components: comps,
		})
	}
	return result, nil
}

func (s *ProductionService) CreateAssembly(ctx context.Context, req models.CreateAssemblyRequest) (any, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if req.Name == "" {
		return nil, fmt.Errorf("assembly name is required")
	}

	// convert components to BomEntry
	entries := make([]models.BomEntry, 0, len(req.Components))
	for i, c := range req.Components {
		if c.Quantity <= 0 {
			return nil, fmt.Errorf("component[%d] quantity must be > 0", i)
		}
		pid, err := uuid.Parse(c.SKU)
		if err != nil {
			return nil, fmt.Errorf("component[%d] sku must be a UUID", i)
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
		pid, err := uuid.Parse(c.SKU)
		if err != nil {
			return nil, fmt.Errorf("component[%d] sku must be a UUID", i)
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
	return s.repo.DeleteBOMEntriesByModelCode(ctx, modelCode)
}

func (s *ProductionService) GetCatalog(ctx context.Context) ([]any, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	boms, err := s.repo.GetAllBOMs(ctx)
	if err != nil {
		return nil, err
	}

	// produce simple catalog: list unique parent models
	parentSet := map[string]struct{}{}
	for _, e := range boms {
		parentSet[e.ParentModelCode] = struct{}{}
	}
	res := make([]any, 0, len(parentSet))
	for p := range parentSet {
		res = append(res, map[string]any{"model_code": p})
	}
	return res, nil
}

func (s *ProductionService) GetWhereUsed(ctx context.Context, sku string) ([]any, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	pid, err := uuid.Parse(sku)
	if err != nil {
		return nil, fmt.Errorf("sku must be a UUID representing part id")
	}
	entries, err := s.repo.GetWhereUsedByPartID(ctx, pid)
	if err != nil {
		return nil, err
	}
	res := make([]any, 0, len(entries))
	for _, e := range entries {
		res = append(res, map[string]any{"parent_model": e.ParentModelCode, "required_qty": e.RequiredQuantityPerUnit})
	}
	return res, nil
}
