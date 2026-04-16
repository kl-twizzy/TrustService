package repository

import (
	"context"
	"errors"

	"seller-trust-map/backend-go/internal/domain"
)

var ErrNotFound = errors.New("not found")

type Repository interface {
	ResolveByURL(ctx context.Context, marketplace string, productID string) (URLResolution, error)
	GetSeller(ctx context.Context, sellerID string) (domain.Seller, error)
	GetProduct(ctx context.Context, productID string) (domain.Product, error)
	GetReviews(ctx context.Context, productID string) ([]domain.Review, error)
	SaveCheckResult(ctx context.Context, result domain.TrustResponse, clientID string) error
	ListRecentChecks(ctx context.Context, limit int, clientID string) ([]domain.RecentCheck, error)
}
