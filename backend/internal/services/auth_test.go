package services

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/rcn/rcn/backend/internal/config"
	"github.com/rcn/rcn/backend/internal/middleware"
)

// TestPasswordHash tests bcrypt password hashing and comparison.
func TestPasswordHash(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wrongPass string
		wantErr  bool
	}{
		{
			name:      "Valid password",
			password:  "SuperSecurePassword123!",
			wrongPass: "WrongSecurePassword123!",
			wantErr:   false,
		},
		{
			name:      "Empty password",
			password:  "",
			wrongPass: "somepass",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := bcrypt.GenerateFromPassword([]byte(tt.password), bcrypt.DefaultCost)
			if err != nil {
				t.Fatalf("Failed to generate hash: %v", err)
			}

			// Compare correct password
			err = bcrypt.CompareHashAndPassword(hash, []byte(tt.password))
			if err != nil {
				t.Errorf("Password comparison failed for correct password: %v", err)
			}

			// Compare wrong password
			err = bcrypt.CompareHashAndPassword(hash, []byte(tt.wrongPass))
			if err == nil {
				t.Error("Expected comparison failure for incorrect password, but got success")
			}
		})
	}
}

// TestTokenGeneration tests JWT creation and validation.
func TestTokenGeneration(t *testing.T) {
	secretKey := "test-jwt-secret-key-that-is-long-enough-12345"
	cfg := &config.Config{
		JWTSecretKey:     secretKey,
		JWTExpireMinutes: 10,
	}

	claims := jwt.MapClaims{
		"admin_id":       "user-123",
		"admin_role":     "admin",
		"admin_username": "admin_user",
		"typ":            "session",
		"exp":            time.Now().Add(10 * time.Minute).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(cfg.JWTSecretKey))
	if err != nil {
		t.Fatalf("Failed to sign token: %v", err)
	}

	// Validate token
	parsedToken, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(cfg.JWTSecretKey), nil
	})

	if err != nil || !parsedToken.Valid {
		t.Fatalf("Failed to validate token: %v", err)
	}

	parsedClaims, ok := parsedToken.Claims.(jwt.MapClaims)
	if !ok {
		t.Fatal("Failed to parse claims")
	}

	if parsedClaims["admin_id"] != "user-123" {
		t.Errorf("Expected admin_id 'user-123', got '%v'", parsedClaims["admin_id"])
	}
	if parsedClaims["admin_role"] != "admin" {
		t.Errorf("Expected admin_role 'admin', got '%v'", parsedClaims["admin_role"])
	}
}

// TestRoleHierarchy tests the role hierarchy and permission restrictions.
func TestRoleHierarchy(t *testing.T) {
	gin.SetMode(gin.TestMode)
	secretKey := "test-jwt-secret-key-that-is-long-enough-12345"
	cfg := &config.Config{
		JWTSecretKey:     secretKey,
		JWTExpireMinutes: 10,
	}

	// Helper to generate a token with specific claims
	genToken := func(role string, typ string) string {
		claims := jwt.MapClaims{
			"admin_id":   "admin-id-123",
			"admin_role": role,
			"typ":        typ,
			"exp":        time.Now().Add(10 * time.Minute).Unix(),
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		str, _ := token.SignedString([]byte(cfg.JWTSecretKey))
		return str
	}

	tests := []struct {
		name           string
		token          string
		setupRoutes    func(r *gin.Engine)
		expectedStatus int
	}{
		{
			name:  "Superadmin can access superadmin-only route",
			token: genToken("superadmin", "session"),
			setupRoutes: func(r *gin.Engine) {
				r.GET("/super", middleware.RequireAdmin(cfg), middleware.RequireSuperAdmin(), func(c *gin.Context) {
					c.Status(http.StatusOK)
				})
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:  "Regular admin cannot access superadmin-only route",
			token: genToken("admin", "session"),
			setupRoutes: func(r *gin.Engine) {
				r.GET("/super", middleware.RequireAdmin(cfg), middleware.RequireSuperAdmin(), func(c *gin.Context) {
					c.Status(http.StatusOK)
				})
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:  "Viewer cannot access editor-required route due to lack of role match",
			token: genToken("viewer", "session"),
			setupRoutes: func(r *gin.Engine) {
				r.GET("/editor", middleware.RequireAdmin(cfg), middleware.RequireRole("editor"), func(c *gin.Context) {
					c.Status(http.StatusOK)
				})
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:  "Editor can access editor-required route",
			token: genToken("editor", "session"),
			setupRoutes: func(r *gin.Engine) {
				r.GET("/editor", middleware.RequireAdmin(cfg), middleware.RequireRole("editor"), func(c *gin.Context) {
					c.Status(http.StatusOK)
				})
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:  "Admin token with wrong type rejected",
			token: genToken("admin", "kernel"),
			setupRoutes: func(r *gin.Engine) {
				r.GET("/admin", middleware.RequireAdmin(cfg), func(c *gin.Context) {
					c.Status(http.StatusOK)
				})
			},
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			tt.setupRoutes(r)

			req, _ := http.NewRequest("GET", "/super", nil)
			if tt.name == "Viewer cannot access editor-required route due to lack of role match" || tt.name == "Editor can access editor-required route" {
				req, _ = http.NewRequest("GET", "/editor", nil)
			} else if tt.name == "Admin token with wrong type rejected" {
				req, _ = http.NewRequest("GET", "/admin", nil)
			}
			req.Header.Set("Authorization", "Bearer "+tt.token)

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}
