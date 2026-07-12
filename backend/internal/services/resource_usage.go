package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rcn/rcn/backend/internal/database"
)

type ResourceUsageRecord struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	ResourceType string    `json:"resource_type"`
	Amount       float64   `json:"amount"`
	Unit         string    `json:"unit"`
	CostEstimate float64   `json:"cost_estimate"`
	Currency     string    `json:"currency"`
	PeriodStart  time.Time `json:"period_start"`
	PeriodEnd    time.Time `json:"period_end"`
	CreatedAt    time.Time `json:"created_at"`
}

type CostRate struct {
	ID            string    `json:"id"`
	ResourceType  string    `json:"resource_type"`
	RatePerUnit   float64   `json:"rate_per_unit"`
	Unit          string    `json:"unit"`
	Currency      string    `json:"currency"`
	EffectiveFrom time.Time `json:"effective_from"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type ResourceUsageService struct {
	rates map[string]CostRate // cached
	mu    sync.RWMutex
}

// NewResourceUsageService initializes and returns a new ResourceUsageService
func NewResourceUsageService() *ResourceUsageService {
	svc := &ResourceUsageService{
		rates: make(map[string]CostRate),
	}
	// Warm up cache in background to handle database transient states gracefully
	go func() {
		_ = svc.loadRatesToCache()
	}()
	return svc
}

func (s *ResourceUsageService) loadRatesToCache() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	db := database.GetDB()
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}

	query := `SELECT id, resource_type, rate_per_unit, unit, currency, effective_from, updated_at FROM cost_rates`
	rows, err := db.Query(query)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var rate CostRate
		err := rows.Scan(
			&rate.ID,
			&rate.ResourceType,
			&rate.RatePerUnit,
			&rate.Unit,
			&rate.Currency,
			&rate.EffectiveFrom,
			&rate.UpdatedAt,
		)
		if err != nil {
			return err
		}
		s.rates[rate.ResourceType] = rate
	}
	return nil
}

// RecordUsage inserts a new resource usage record into the database
func (s *ResourceUsageService) RecordUsage(ctx context.Context, userID, resourceType string, amount float64, unit string, costEstimate float64) error {
	db := database.GetDB()
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}

	periodStart := time.Now().Truncate(time.Hour)
	periodEnd := periodStart.Add(time.Hour)
	currency := "VND"

	s.mu.RLock()
	if rate, exists := s.rates[resourceType]; exists {
		currency = rate.Currency
	}
	s.mu.RUnlock()

	query := `
		INSERT INTO resource_usage (user_id, resource_type, amount, unit, cost_estimate, currency, period_start, period_end)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := db.ExecContext(ctx, query, userID, resourceType, amount, unit, costEstimate, currency, periodStart, periodEnd)
	if err != nil {
		return fmt.Errorf("failed to record resource usage: %w", err)
	}

	return nil
}

// GetUserUsage retrieves resource usage records for a specific user with pagination
func (s *ResourceUsageService) GetUserUsage(ctx context.Context, userID string, from, to time.Time, limit, offset int) ([]ResourceUsageRecord, int, error) {
	db := database.GetDB()
	if db == nil {
		return nil, 0, fmt.Errorf("database connection is nil")
	}

	countQuery := `
		SELECT COUNT(*)
		FROM resource_usage
		WHERE user_id = $1 AND period_start >= $2 AND period_end <= $3
	`
	var total int
	err := db.QueryRowContext(ctx, countQuery, userID, from, to).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count user usage: %w", err)
	}

	selectQuery := `
		SELECT id, user_id, resource_type, amount, unit, cost_estimate, currency, period_start, period_end, created_at
		FROM resource_usage
		WHERE user_id = $1 AND period_start >= $2 AND period_end <= $3
		ORDER BY period_start DESC
		LIMIT $4 OFFSET $5
	`
	rows, err := db.QueryContext(ctx, selectQuery, userID, from, to, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query user usage: %w", err)
	}
	defer rows.Close()

	var records []ResourceUsageRecord
	for rows.Next() {
		var r ResourceUsageRecord
		err := rows.Scan(
			&r.ID,
			&r.UserID,
			&r.ResourceType,
			&r.Amount,
			&r.Unit,
			&r.CostEstimate,
			&r.Currency,
			&r.PeriodStart,
			&r.PeriodEnd,
			&r.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan user usage record: %w", err)
		}
		records = append(records, r)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, err
	}

	return records, total, nil
}

// GetAllUsage retrieves resource usage records for all users (or filtered by user_id if provided) with pagination
func (s *ResourceUsageService) GetAllUsage(ctx context.Context, from, to time.Time, userID string, limit, offset int) ([]ResourceUsageRecord, int, error) {
	db := database.GetDB()
	if db == nil {
		return nil, 0, fmt.Errorf("database connection is nil")
	}

	var countQuery string
	var selectQuery string
	var countArgs []interface{}
	var selectArgs []interface{}

	if userID != "" {
		countQuery = `
			SELECT COUNT(*)
			FROM resource_usage
			WHERE period_start >= $1 AND period_end <= $2 AND user_id = $3
		`
		countArgs = []interface{}{from, to, userID}

		selectQuery = `
			SELECT id, user_id, resource_type, amount, unit, cost_estimate, currency, period_start, period_end, created_at
			FROM resource_usage
			WHERE period_start >= $1 AND period_end <= $2 AND user_id = $3
			ORDER BY period_start DESC
			LIMIT $4 OFFSET $5
		`
		selectArgs = []interface{}{from, to, userID, limit, offset}
	} else {
		countQuery = `
			SELECT COUNT(*)
			FROM resource_usage
			WHERE period_start >= $1 AND period_end <= $2
		`
		countArgs = []interface{}{from, to}

		selectQuery = `
			SELECT id, user_id, resource_type, amount, unit, cost_estimate, currency, period_start, period_end, created_at
			FROM resource_usage
			WHERE period_start >= $1 AND period_end <= $2
			ORDER BY period_start DESC
			LIMIT $3 OFFSET $4
		`
		selectArgs = []interface{}{from, to, limit, offset}
	}

	var total int
	err := db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count all usage: %w", err)
	}

	rows, err := db.QueryContext(ctx, selectQuery, selectArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query all usage: %w", err)
	}
	defer rows.Close()

	var records []ResourceUsageRecord
	for rows.Next() {
		var r ResourceUsageRecord
		err := rows.Scan(
			&r.ID,
			&r.UserID,
			&r.ResourceType,
			&r.Amount,
			&r.Unit,
			&r.CostEstimate,
			&r.Currency,
			&r.PeriodStart,
			&r.PeriodEnd,
			&r.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan usage record: %w", err)
		}
		records = append(records, r)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, err
	}

	return records, total, nil
}

// GetSummary returns resource usage summaries aggregated by type and user, along with total cost
func (s *ResourceUsageService) GetSummary(ctx context.Context, from, to time.Time) (map[string]interface{}, error) {
	db := database.GetDB()
	if db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	typeQuery := `
		SELECT resource_type, COALESCE(SUM(amount), 0), COALESCE(SUM(cost_estimate), 0)
		FROM resource_usage
		WHERE period_start >= $1 AND period_end <= $2
		GROUP BY resource_type
	`
	typeRows, err := db.QueryContext(ctx, typeQuery, from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to query usage by type: %w", err)
	}
	defer typeRows.Close()

	totalCost := 0.0
	totalByType := make(map[string]interface{})

	for typeRows.Next() {
		var rType string
		var amount float64
		var cost float64
		if err := typeRows.Scan(&rType, &amount, &cost); err != nil {
			return nil, fmt.Errorf("failed to scan type summary: %w", err)
		}
		totalByType[rType] = map[string]interface{}{
			"amount": amount,
			"cost":   cost,
		}
		totalCost += cost
	}

	userQuery := `
		SELECT user_id, COALESCE(SUM(cost_estimate), 0)
		FROM resource_usage
		WHERE period_start >= $1 AND period_end <= $2
		GROUP BY user_id
	`
	userRows, err := db.QueryContext(ctx, userQuery, from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to query usage by user: %w", err)
	}
	defer userRows.Close()

	totalByUser := make(map[string]interface{})
	for userRows.Next() {
		var userID string
		var cost float64
		if err := userRows.Scan(&userID, &cost); err != nil {
			return nil, fmt.Errorf("failed to scan user summary: %w", err)
		}
		totalByUser[userID] = cost
	}

	summary := map[string]interface{}{
		"total_cost":    totalCost,
		"total_by_type": totalByType,
		"total_by_user": totalByUser,
	}

	return summary, nil
}

// GetCostRates returns all configured cost rates and updates the cache
func (s *ResourceUsageService) GetCostRates() ([]CostRate, error) {
	db := database.GetDB()
	if db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	query := `SELECT id, resource_type, rate_per_unit, unit, currency, effective_from, updated_at FROM cost_rates`
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query cost rates: %w", err)
	}
	defer rows.Close()

	var rates []CostRate
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rates = make(map[string]CostRate)

	for rows.Next() {
		var rate CostRate
		err := rows.Scan(
			&rate.ID,
			&rate.ResourceType,
			&rate.RatePerUnit,
			&rate.Unit,
			&rate.Currency,
			&rate.EffectiveFrom,
			&rate.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan cost rate: %w", err)
		}
		rates = append(rates, rate)
		s.rates[rate.ResourceType] = rate
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return rates, nil
}

// UpsertCostRate inserts a cost rate or updates it on conflict, then updates the cache
func (s *ResourceUsageService) UpsertCostRate(ctx context.Context, resourceType string, rate float64, unit, currency string) error {
	db := database.GetDB()
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}

	query := `
		INSERT INTO cost_rates (resource_type, rate_per_unit, unit, currency)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (resource_type)
		DO UPDATE SET rate_per_unit = $2, unit = $3, currency = $4, updated_at = NOW()
	`

	_, err := db.ExecContext(ctx, query, resourceType, rate, unit, currency)
	if err != nil {
		return fmt.Errorf("failed to upsert cost rate: %w", err)
	}

	selectQuery := `SELECT id, resource_type, rate_per_unit, unit, currency, effective_from, updated_at FROM cost_rates WHERE resource_type = $1`
	var cr CostRate
	err = db.QueryRowContext(ctx, selectQuery, resourceType).Scan(
		&cr.ID,
		&cr.ResourceType,
		&cr.RatePerUnit,
		&cr.Unit,
		&cr.Currency,
		&cr.EffectiveFrom,
		&cr.UpdatedAt,
	)
	if err == nil {
		s.mu.Lock()
		s.rates[resourceType] = cr
		s.mu.Unlock()
	}

	return nil
}

// SeedDefaultRates seeds default rates with "VND" currency
func (s *ResourceUsageService) SeedDefaultRates(ctx context.Context, rates map[string]struct {
	Rate float64
	Unit string
}) error {
	for rType, val := range rates {
		err := s.UpsertCostRate(ctx, rType, val.Rate, val.Unit, "VND")
		if err != nil {
			return fmt.Errorf("failed to seed default rate for %s: %w", rType, err)
		}
	}
	return nil
}
