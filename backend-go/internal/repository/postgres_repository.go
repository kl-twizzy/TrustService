package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"

	"seller-trust-map/backend-go/internal/domain"
)

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) ResolveByURL(ctx context.Context, marketplace string, productID string) (URLResolution, error) {
	const query = `
		SELECT s.external_id, p.external_id
		FROM products p
		JOIN marketplaces m ON m.id = p.marketplace_id
		LEFT JOIN sellers s ON s.id = p.seller_id
		WHERE m.code::text = $1
		  AND (
		    p.external_id = $2
		    OR p.external_id LIKE $3
		    OR p.sku = $2
		  )
		ORDER BY p.updated_at DESC, p.created_at DESC
		LIMIT 1
	`

	var sellerID sql.NullString
	var productExternalID string
	err := r.db.QueryRowContext(ctx, query, marketplace, productID, "%"+productID).Scan(&sellerID, &productExternalID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return URLResolution{}, ErrNotFound
		}
		return URLResolution{}, err
	}

	if !sellerID.Valid {
		return URLResolution{}, ErrNotFound
	}

	return URLResolution{
		Marketplace: marketplace,
		SellerID:    sellerID.String,
		ProductID:   productExternalID,
	}, nil
}

func (r *PostgresRepository) GetSeller(ctx context.Context, sellerID string) (domain.Seller, error) {
	const query = `
		SELECT s.external_id, s.display_name, m.code::text, s.rating, s.total_orders,
		       s.successful_orders, s.complaints_count, s.returns_count, s.average_review_score
		FROM sellers s
		JOIN marketplaces m ON m.id = s.marketplace_id
		WHERE s.external_id = $1
		LIMIT 1
	`

	var seller domain.Seller
	err := r.db.QueryRowContext(ctx, query, sellerID).Scan(
		&seller.ExternalID,
		&seller.Name,
		&seller.Marketplace,
		&seller.Rating,
		&seller.TotalOrders,
		&seller.SuccessfulOrders,
		&seller.ComplaintsCount,
		&seller.ReturnsCount,
		&seller.AverageReviewScore,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Seller{}, ErrNotFound
		}
		return domain.Seller{}, err
	}

	return seller, nil
}

func (r *PostgresRepository) GetProduct(ctx context.Context, productID string) (domain.Product, error) {
	const query = `
		SELECT p.external_id, s.external_id, p.title, COALESCE(c.slug, 'uncategorized'),
		       p.rating, p.review_count, p.one_star_count, p.five_star_count
		FROM products p
		LEFT JOIN sellers s ON s.id = p.seller_id
		LEFT JOIN product_categories c ON c.id = p.category_id
		WHERE p.external_id = $1
		LIMIT 1
	`

	var product domain.Product
	var sellerID sql.NullString
	err := r.db.QueryRowContext(ctx, query, productID).Scan(
		&product.ExternalID,
		&sellerID,
		&product.Title,
		&product.Category,
		&product.Rating,
		&product.ReviewCount,
		&product.OneStarCount,
		&product.FiveStarCount,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Product{}, ErrNotFound
		}
		return domain.Product{}, err
	}

	product.SellerID = sellerID.String
	return product, nil
}

func (r *PostgresRepository) GetReviews(ctx context.Context, productID string) ([]domain.Review, error) {
	const query = `
		SELECT rv.external_id, rv.rating, rv.review_text, rv.is_verified_purchase, rv.source_created_at
		FROM reviews rv
		JOIN products p ON p.id = rv.product_id
		WHERE p.external_id = $1
		ORDER BY rv.source_created_at DESC
		LIMIT 50
	`

	rows, err := r.db.QueryContext(ctx, query, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	reviews := make([]domain.Review, 0, 16)
	for rows.Next() {
		var review domain.Review
		var createdAt time.Time
		if err := rows.Scan(&review.ExternalID, &review.Rating, &review.Text, &review.IsVerified, &createdAt); err != nil {
			return nil, err
		}
		review.CreatedAt = createdAt.Format(time.RFC3339)
		reviews = append(reviews, review)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(reviews) == 0 {
		return nil, ErrNotFound
	}

	return reviews, nil
}

func (r *PostgresRepository) SaveCheckResult(ctx context.Context, result domain.TrustResponse, clientID string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	marketplaceID, _ := r.lookupMarketplaceID(ctx, tx, result.Marketplace)
	productDBID, _ := r.lookupEntityID(ctx, tx, "products", result.Product.ID)
	sellerDBID, _ := r.lookupEntityID(ctx, tx, "sellers", result.Seller.ID)

	const analysisRunQuery = `
		INSERT INTO analysis_runs (product_id, seller_id, marketplace_id, source_url, parser_version, model_version, status, finished_at)
		VALUES ($1, $2, $3, $4, 'v1', 'v1', 'completed', NOW())
		RETURNING id
	`

	var analysisRunID string
	if err := tx.QueryRowContext(ctx, analysisRunQuery,
		nullableString(productDBID),
		nullableString(sellerDBID),
		nullableString(marketplaceID),
		result.CheckedURL,
	).Scan(&analysisRunID); err != nil {
		return err
	}

	const snapshotQuery = `
		INSERT INTO trust_snapshots (
			analysis_run_id, product_id, seller_id, trust_score, trust_level, rating_authenticity,
			fake_review_risk, rating_manipulation_risk, text_similarity_risk, review_burst_risk,
			suspicious, recommendation
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id
	`

	var snapshotID string
	if err := tx.QueryRowContext(ctx, snapshotQuery,
		analysisRunID,
		nullableString(productDBID),
		nullableString(sellerDBID),
		result.TrustScore,
		result.TrustLevel,
		result.RatingAuthenticity,
		result.Analysis.FakeReviewRisk,
		result.Analysis.RatingManipulation,
		result.Analysis.TextSimilarityRisk,
		result.Analysis.ReviewBurstRisk,
		result.Suspicious,
		result.Recommendation,
	).Scan(&snapshotID); err != nil {
		return err
	}

	const reasonQuery = `
		INSERT INTO trust_reasons (snapshot_id, reason_code, reason_text, severity)
		VALUES ($1, $2, $3, $4)
	`
	for index, reason := range result.Reasons {
		if _, err := tx.ExecContext(ctx, reasonQuery, snapshotID, fmt.Sprintf("reason_%d", index+1), reason, 2); err != nil {
			return err
		}
	}

	const browserCheckQuery = `
		INSERT INTO browser_checks (source_url, marketplace_id, product_id, seller_id, client_type, client_version, client_id, success)
		VALUES ($1, $2, $3, $4, 'browser_extension', '0.2.0', $5, TRUE)
	`
	if _, err := tx.ExecContext(ctx, browserCheckQuery,
		result.CheckedURL,
		nullableString(marketplaceID),
		nullableString(productDBID),
		nullableString(sellerDBID),
		nullableString(clientID),
	); err != nil {
		return err
	}

	return tx.Commit()
}

func (r *PostgresRepository) ListRecentChecks(ctx context.Context, limit int, clientID string) ([]domain.RecentCheck, error) {
	baseQuery := `
		SELECT
			ts.id,
			COALESCE(m.code::text, ar.source_url, 'unknown'),
			COALESCE(ar.source_url, ''),
			COALESCE(p.title, ''),
			COALESCE(s.display_name, ''),
			ts.trust_score,
			ts.trust_level,
			ts.rating_authenticity,
			ts.suspicious,
			COALESCE(array_remove(array_agg(tr.reason_text), NULL), '{}')::text[],
			ts.created_at
		FROM trust_snapshots ts
		LEFT JOIN analysis_runs ar ON ar.id = ts.analysis_run_id
		LEFT JOIN browser_checks bc ON bc.source_url = ar.source_url
		LEFT JOIN products p ON p.id = ts.product_id
		LEFT JOIN sellers s ON s.id = ts.seller_id
		LEFT JOIN marketplaces m ON m.id = ar.marketplace_id
		LEFT JOIN trust_reasons tr ON tr.snapshot_id = ts.id
	`
	whereClause := ""
	args := []any{}
	if strings.TrimSpace(clientID) != "" {
		whereClause = ` WHERE bc.client_id = $1 `
		args = append(args, clientID)
	}
	query := baseQuery + whereClause + `
		GROUP BY ts.id, m.code, ar.source_url, p.title, s.display_name, ts.trust_score, ts.trust_level,
		         ts.rating_authenticity, ts.suspicious, ts.created_at
		ORDER BY ts.created_at DESC
		LIMIT $` + fmt.Sprintf("%d", len(args)+1)
	args = append(args, limit)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]domain.RecentCheck, 0, limit)
	for rows.Next() {
		var item domain.RecentCheck
		var checkedAt time.Time
		var reasons pq.StringArray
		if err := rows.Scan(
			&item.ID,
			&item.Marketplace,
			&item.CheckedURL,
			&item.ProductTitle,
			&item.SellerName,
			&item.TrustScore,
			&item.TrustLevel,
			&item.RatingAuthenticity,
			&item.Suspicious,
			&reasons,
			&checkedAt,
		); err != nil {
			return nil, err
		}
		item.Reasons = []string(reasons)
		item.Marketplace = normalizeMarketplaceLabel(item.Marketplace)
		item.CheckedAt = checkedAt.Format(time.RFC3339)
		result = append(result, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return nil, ErrNotFound
	}

	return result, nil
}

func (r *PostgresRepository) lookupMarketplaceID(ctx context.Context, tx *sql.Tx, marketplace string) (string, error) {
	const query = `SELECT id FROM marketplaces WHERE code::text = $1 LIMIT 1`
	var id string
	if err := tx.QueryRowContext(ctx, query, marketplace).Scan(&id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", err
	}
	return id, nil
}

func (r *PostgresRepository) lookupEntityID(ctx context.Context, tx *sql.Tx, table string, externalID string) (string, error) {
	if externalID == "" {
		return "", ErrNotFound
	}
	query := fmt.Sprintf(`SELECT id FROM %s WHERE external_id = $1 LIMIT 1`, table)
	var id string
	if err := tx.QueryRowContext(ctx, query, externalID).Scan(&id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", err
	}
	return id, nil
}

func nullableString(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func normalizeMarketplaceLabel(value string) string {
	switch value {
	case "ozon":
		return "ozon"
	case "wildberries":
		return "wildberries"
	case "yandex_market":
		return "yandex_market"
	default:
		return value
	}
}
