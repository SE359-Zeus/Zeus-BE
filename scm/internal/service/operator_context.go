package service

import "context"

func operatorNameFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v := ctx.Value("full_name"); v != nil {
		if name, ok := v.(string); ok {
			return name
		}
	}
	return ""
}
