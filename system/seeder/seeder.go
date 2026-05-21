package seeder

import (
	"log"
	"time"

	"zeus-system-service/internal/models"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type roleSeed struct {
	Name        string
	DisplayName string
	Level       string
	Module      string
	Description string
}

var roles = []roleSeed{
	{Name: "admin", DisplayName: "Administrator", Level: "Administrator", Module: "Global", Description: "Global system control, user management, and employee oversight."},
	{Name: "scm_operator", DisplayName: "SCM Operator", Level: "Operator", Module: "SCM", Description: "Resolve vendor selection, approve POs, orchestrate outbound dispatch."},
	{Name: "scm_worker", DisplayName: "SCM Worker", Level: "Worker", Module: "SCM", Description: "Physical validation of inbound shipments, inventory ledger updates."},
	{Name: "mrp_operator", DisplayName: "MRP Operator", Level: "Operator", Module: "MRP", Description: "Manage Component Catalog/BOM, trigger calculation runs."},
	{Name: "mrp_worker", DisplayName: "MRP Worker", Level: "Worker", Module: "MRP", Description: "Monitor material readiness and generate warehouse pick lists."},
	{Name: "sales_operator", DisplayName: "Sales Operator", Level: "Operator", Module: "Sales", Description: "Manage Client Registry, resolve fulfillment bottlenecks."},
	{Name: "sales_worker", DisplayName: "Sales Worker", Level: "Worker", Module: "Sales", Description: "API order validation, sales order creation."},
}

type actionTypeSeed struct {
	Name        string
	Description string
	IsSecurity  bool
}

var actionTypes = []actionTypeSeed{
	{Name: "LOGIN", Description: "User login event", IsSecurity: false},
	{Name: "CREATE", Description: "Resource creation event", IsSecurity: false},
	{Name: "UPDATE", Description: "Resource update event", IsSecurity: false},
	{Name: "DELETE", Description: "Resource deletion event", IsSecurity: false},
	{Name: "SECURITY", Description: "Security-related event", IsSecurity: true},
}

type endpointSeed struct {
	Method        string
	Path          string
	RequiredLevel string
}

var endpoints = []endpointSeed{
	{Method: "POST", Path: "/api/v1/users", RequiredLevel: "Administrator"},
	{Method: "GET", Path: "/api/v1/users", RequiredLevel: "Administrator"},
	{Method: "GET", Path: "/api/v1/users/:id", RequiredLevel: "Administrator"},
	{Method: "PUT", Path: "/api/v1/users/:id", RequiredLevel: "Administrator"},
	{Method: "PATCH", Path: "/api/v1/users/:id/status", RequiredLevel: "Administrator"},
	{Method: "POST", Path: "/api/v1/logs/ingest", RequiredLevel: "Administrator"},
	{Method: "GET", Path: "/api/v1/logs", RequiredLevel: "Administrator"},
	{Method: "GET", Path: "/api/v1/logs/metrics", RequiredLevel: "Administrator"},
}

func SeedAll(db *gorm.DB) error {
	log.Println("Seeding system service data...")

	seedRoles(db)
	seedActionTypes(db)
	seedEndpointRoles(db)
	seedUsers(db)
	seedAuditLogs(db)

	log.Println("Seeding complete.")
	return nil
}

func seedRoles(db *gorm.DB) {
	for _, rs := range roles {
		role := models.Role{
			Name:        rs.Name,
			DisplayName: rs.DisplayName,
			Level:       rs.Level,
			Module:      rs.Module,
			Description: rs.Description,
		}
		result := db.Where("name = ?", rs.Name).FirstOrCreate(&role)
		if result.Error != nil {
			log.Fatalf("failed to seed role %s: %v", rs.Name, result.Error)
		}
		if result.RowsAffected > 0 {
			log.Printf("Created role: %s (%s)", rs.Name, rs.DisplayName)
		}
	}
}

func seedActionTypes(db *gorm.DB) {
	for _, at := range actionTypes {
		entry := models.ActionTypeEntry{
			Name:        at.Name,
			Description: at.Description,
			IsSecurity:  at.IsSecurity,
		}
		result := db.Where("name = ?", at.Name).FirstOrCreate(&entry)
		if result.Error != nil {
			log.Fatalf("failed to seed action type %s: %v", at.Name, result.Error)
		}
		if result.RowsAffected > 0 {
			log.Printf("Created action type: %s", at.Name)
		}
	}
}

func seedEndpointRoles(db *gorm.DB) {
	for _, ep := range endpoints {
		entry := models.EndpointRole{
			Method:        ep.Method,
			Path:          ep.Path,
			RequiredLevel: ep.RequiredLevel,
		}
		result := db.Where("method = ? AND path = ?", ep.Method, ep.Path).FirstOrCreate(&entry)
		if result.Error != nil {
			log.Fatalf("failed to seed endpoint role %s %s: %v", ep.Method, ep.Path, result.Error)
		}
		if result.RowsAffected > 0 {
			log.Printf("Created endpoint role: %s %s → %s", ep.Method, ep.Path, ep.RequiredLevel)
		}
	}
}

func seedUsers(db *gorm.DB) {
	type seedUser struct {
		Email    string
		Password string
		FullName string
		Role     string
	}

	users := []seedUser{
		{Email: "admin@zeus.com", Password: "admin123", FullName: "System Administrator", Role: "admin"},
		{Email: "scm-operator@zeus.com", Password: "scm123", FullName: "SCM Operator", Role: "scm_operator"},
		{Email: "scm-worker@zeus.com", Password: "scm123", FullName: "SCM Worker", Role: "scm_worker"},
		{Email: "mrp-operator@zeus.com", Password: "mrp123", FullName: "MRP Operator", Role: "mrp_operator"},
		{Email: "mrp-worker@zeus.com", Password: "mrp123", FullName: "MRP Worker", Role: "mrp_worker"},
		{Email: "sales-operator@zeus.com", Password: "sales123", FullName: "Sales Operator", Role: "sales_operator"},
		{Email: "sales-worker@zeus.com", Password: "sales123", FullName: "Sales Worker", Role: "sales_worker"},
	}

	for _, u := range users {
		hash, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
		if err != nil {
			log.Fatalf("failed to hash password for %s: %v", u.Email, err)
		}

		user := &models.User{
			Email:        u.Email,
			PasswordHash: string(hash),
			FullName:     u.FullName,
			Role:         u.Role,
			Status:       models.AccountStatusActive,
		}

		result := db.Where("email = ?", u.Email).FirstOrCreate(user)
		if result.Error != nil {
			log.Fatalf("failed to seed user %s: %v", u.Email, result.Error)
		}
		if result.RowsAffected > 0 {
			log.Printf("Created user: %s (%s) — %s", u.Email, u.Role, u.FullName)
		} else {
			log.Printf("User already exists: %s", u.Email)
		}
	}
}

func seedAuditLogs(db *gorm.DB) {
	now := time.Now()

	var allUsers []models.User
	if err := db.Find(&allUsers).Error; err != nil {
		log.Printf("Warning: failed to fetch users for audit log seeding: %v", err)
		return
	}
	if len(allUsers) == 0 {
		return
	}
	admin := &allUsers[0]

	type seedEvent struct {
		user       *models.User
		action     string
		target     string
		details    string
		ip         string
		isSecurity bool
		hoursAgo   int
	}

	events := []seedEvent{
		{admin, "LOGIN", "auth/login", "Successful login", "10.0.0.1", false, 1},
		{admin, "LOGIN", "auth/login", "Successful login", "10.0.0.1", false, 2},
		{admin, "CREATE", "users/" + admin.ID.String(), "Created admin account", "10.0.0.1", false, 4},
		{admin, "UPDATE", "config/security", "Updated password policy", "10.0.0.1", false, 12},
		{admin, "SECURITY", "auth/login", "Failed login attempt from unknown IP", "203.0.113.1", true, 9},
		{admin, "SECURITY", "auth/login", "Brute force attempt detected", "198.51.100.1", true, 10},
		{admin, "SECURITY", "resources/confidential", "Unauthorized access attempt", "192.0.2.1", true, 16},
	}

	for _, e := range events {
		ts := now.Add(-time.Duration(e.hoursAgo) * time.Hour)
		logEntry := &models.AuditLog{
			UserID:          e.user.ID,
			UserEmail:       e.user.Email,
			ActionType:      models.ActionType(e.action),
			TargetResource:  e.target,
			Details:         e.details,
			IPAddress:       e.ip,
			IsSecurityEvent: e.isSecurity,
			Timestamp:       ts,
		}

		result := db.Create(logEntry)
		if result.Error != nil {
			log.Printf("Warning: failed to create audit log entry: %v", result.Error)
		}
	}

	log.Printf("Seeded %d audit log entries", len(events))
}

func init() {
	uuid.New()
}
