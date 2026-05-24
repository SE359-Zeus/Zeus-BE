package exception

import "net/http"

var (
	ErrUnauthorized      = New("AUTH_001", "Unauthorized user", http.StatusUnauthorized)
	ErrInvalidToken      = New("AUTH_002", "Invalid or expired token", http.StatusUnauthorized)
	ErrInactiveAccount   = New("AUTH_003", "Account is inactive", http.StatusUnauthorized)
	ErrAPIKeyExpired     = New("AUTH_004", "API key expired", http.StatusUnauthorized)
	ErrAPIKeyInvalid     = New("AUTH_005", "Invalid API key", http.StatusUnauthorized)
	ErrInvalidAuthHeader = New("AUTH_006", "Invalid authorization header format", http.StatusUnauthorized)
	ErrMissingAuth       = New("AUTH_007", "Missing authentication", http.StatusUnauthorized)
	ErrLoginFailed       = New("AUTH_008", "Invalid email or password", http.StatusUnauthorized)
	ErrMissingAuthHeader = New("AUTH_009", "Missing authorization header", http.StatusUnauthorized)

	ErrMissingRole = New("RBAC_001", "Missing role in context", http.StatusUnauthorized)
	ErrForbidden   = New("RBAC_002", "Forbidden: insufficient role level", http.StatusForbidden)
	ErrAccessCheck = New("RBAC_003", "Access check failed", http.StatusInternalServerError)

	ErrInvalidBody       = New("VAL_001", "Invalid request body", http.StatusBadRequest)
	ErrInvalidInput      = New("VAL_002", "Invalid input", http.StatusBadRequest)
	ErrInvalidResourceID = New("VAL_003", "Invalid resource ID", http.StatusBadRequest)

	ErrInternal = New("SYS_001", "Internal server error", http.StatusInternalServerError)
	ErrPanic    = New("SYS_002", "Unexpected server error", http.StatusInternalServerError)

	ErrDatabase = New("DB_001", "Database error", http.StatusInternalServerError)

	ErrDuplicateEmail = New("USER_001", "Email already exists", http.StatusConflict)
	ErrNotFound       = New("USER_002", "Resource not found", http.StatusNotFound)
	ErrInvalidRole    = New("USER_003", "Invalid role", http.StatusBadRequest)
	ErrInvalidEmail   = New("USER_004", "Invalid email format", http.StatusBadRequest)
	ErrEmptyEmail     = New("USER_005", "Email cannot be empty", http.StatusBadRequest)
	ErrEmptyPassword  = New("USER_006", "Password cannot be empty", http.StatusBadRequest)
	ErrShortPassword  = New("USER_007", "Password must be at least 8 characters", http.StatusBadRequest)
	ErrEmptyName      = New("USER_008", "Full name cannot be empty", http.StatusBadRequest)
	ErrNilID          = New("USER_009", "ID is required", http.StatusBadRequest)

	ErrMonoVendorViolation = New("PO_001", "Purchase order must involve a single vendor", http.StatusBadRequest)
	ErrInvalidTransition   = New("PO_002", "Invalid state transition", http.StatusBadRequest)

	ErrAlreadyLocked  = New("GR_001", "Resource already locked by another operator", http.StatusConflict)
	ErrLockExpired    = New("GR_002", "Lock has expired", http.StatusConflict)
	ErrNotImplemented = New("GR_003", "Not implemented", http.StatusNotImplemented)

	ErrShipmentLockConflict = New("SHIP_001", "Shipment already locked by another operator", http.StatusConflict)

	ErrAgingQuarantine     = New("INV_001", "Component exceeds aging threshold", http.StatusBadRequest)
	ErrInsufficientDeficit = New("INV_002", "Insufficient deficit in pool for this SKU", http.StatusBadRequest)
	ErrNoOptimalSupplier   = New("VEN_001", "No optimal supplier found for the given SKU", http.StatusNotFound)
	ErrStateRegression     = New("PO_003", "State regression is not allowed", http.StatusBadRequest)
)
