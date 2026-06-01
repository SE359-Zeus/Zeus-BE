package exception

import "net/http"

var (
	// ── Authentication ────────────────────────────────────────────────────────
	ErrUnauthorized      = New("AUTH_001", "Please log in or provide a valid API key", http.StatusUnauthorized)
	ErrInvalidToken      = New("AUTH_002", "Your session has expired, please log in again", http.StatusUnauthorized)
	ErrInactiveAccount   = New("AUTH_003", "Your account has been deactivated, please contact an administrator", http.StatusUnauthorized)
	ErrAPIKeyExpired     = New("AUTH_004", "Your API key has expired, please generate a new one", http.StatusUnauthorized)
	ErrAPIKeyInvalid     = New("AUTH_005", "The API key provided is not valid", http.StatusUnauthorized)
	ErrInvalidAuthHeader = New("AUTH_006", "Authorization header format is invalid, expected: Bearer <token>", http.StatusUnauthorized)
	ErrMissingAuth       = New("AUTH_007", "Please include an Authorization header with your request", http.StatusUnauthorized)
	ErrLoginFailed       = New("AUTH_008", "The email or password you entered is incorrect", http.StatusUnauthorized)
	ErrMissingAuthHeader = New("AUTH_009", "Please include an Authorization header with your request", http.StatusUnauthorized)

	// ── Authorization ─────────────────────────────────────────────────────────
	ErrMissingRole = New("RBAC_001", "Your account does not have a role assigned, please contact an administrator", http.StatusUnauthorized)
	ErrForbidden   = New("RBAC_002", "You do not have permission to perform this action", http.StatusForbidden)
	ErrAccessCheck = New("RBAC_003", "Unable to verify your access level, please try again", http.StatusInternalServerError)

	// ── Validation ────────────────────────────────────────────────────────────
	ErrInvalidBody       = New("VAL_001", "The request body is not valid JSON or is missing required fields", http.StatusBadRequest)
	ErrInvalidInput      = New("VAL_002", "One or more input values are invalid", http.StatusBadRequest)
	ErrInvalidResourceID = New("VAL_003", "The resource identifier provided is not a valid format", http.StatusBadRequest)

	// ── System ────────────────────────────────────────────────────────────────
	ErrInternal = New("SYS_001", "Something went wrong on our end, please try again later", http.StatusInternalServerError)
	ErrPanic    = New("SYS_002", "An unexpected error occurred, please try again later", http.StatusInternalServerError)

	// ── Database ──────────────────────────────────────────────────────────────
	ErrDatabase = New("DB_001", "A database error occurred, please try again later", http.StatusInternalServerError)

	// ── User ──────────────────────────────────────────────────────────────────
	ErrDuplicateEmail    = New("USER_001", "An account with this email already exists", http.StatusConflict)
	ErrNotFound          = New("USER_002", "The requested resource could not be found", http.StatusNotFound)
	ErrInvalidRole       = New("USER_003", "The role specified is not valid", http.StatusBadRequest)
	ErrInvalidEmail      = New("USER_004", "Please enter a valid email address", http.StatusBadRequest)
	ErrEmptyEmail        = New("USER_005", "Email address is required", http.StatusBadRequest)
	ErrEmptyPassword     = New("USER_006", "Password is required", http.StatusBadRequest)
	ErrShortPassword     = New("USER_007", "Password must be at least 8 characters long", http.StatusBadRequest)
	ErrEmptyName         = New("USER_008", "Full name is required", http.StatusBadRequest)
	ErrNilID             = New("USER_009", "A resource ID is required", http.StatusBadRequest)

	// ── Purchase Orders ───────────────────────────────────────────────────────
	ErrMonoVendorViolation = New("PO_001", "A purchase order can only contain items from a single supplier, please create separate orders for each supplier", http.StatusBadRequest)
	ErrInvalidTransition   = New("PO_002", "This order cannot be moved to the requested status from its current state", http.StatusBadRequest)
	ErrStateRegression     = New("PO_003", "Orders cannot be moved back to a previous status", http.StatusBadRequest)
	ErrIncompleteGRs       = New("PO_004", "All goods receipts for this order must be completed before it can advance", http.StatusConflict)

	// ── Goods Receipts ────────────────────────────────────────────────────────
	ErrAlreadyLocked  = New("GR_001", "This resource is currently being processed by another operator, please wait or ask them to release it", http.StatusConflict)
	ErrLockExpired    = New("GR_002", "Your session has expired, please re-acquire the lock to continue", http.StatusConflict)
	ErrNotImplemented = New("GR_003", "This feature is not available yet", http.StatusNotImplemented)

	// ── Shipments ─────────────────────────────────────────────────────────────
	ErrShipmentLockConflict     = New("SHIP_001", "This shipment is currently being packed by another operator, please wait or ask them to release the lock", http.StatusConflict)
	ErrShipmentLockExpired      = New("SHIP_002", "Your packing session has expired, please re-acquire the lock to continue", http.StatusConflict)
	ErrShipmentNotLocked        = New("SHIP_003", "You must start packing this shipment before it can be dispatched", http.StatusPreconditionRequired)
	ErrShipmentAlreadyDispatched = New("SHIP_004", "This shipment has already been dispatched and cannot be modified", http.StatusConflict)

	// ── Inventory ─────────────────────────────────────────────────────────────
	ErrAgingQuarantine     = New("INV_001", "This component has been flagged for quarantine due to exceeding the aging threshold", http.StatusBadRequest)
	ErrInsufficientDeficit = New("INV_002", "There is not enough deficit capacity available for this SKU and quantity", http.StatusBadRequest)
	ErrNoOptimalSupplier   = New("VEN_001", "No supplier mapping exists for this SKU, please create a vendor-SKU mapping first", http.StatusNotFound)
)
