package repository

import (
	"context"
	"errors"

	"seller-trust-map/backend-go/internal/domain"
)

type CombinedRepository struct {
	primary  Repository
	fallback Repository
}

func NewCombinedRepository(primary Repository, fallback Repository) *CombinedRepository {
	return &CombinedRepository{
		primary:  primary,
		fallback: fallback,
	}
}

func (r *CombinedRepository) ResolveByURL(ctx context.Context, marketplace string, productID string) (URLResolution, error) {
	if r.primary != nil {
		result, err := r.primary.ResolveByURL(ctx, marketplace, productID)
		if err == nil {
			return result, nil
		}
		if !errors.Is(err, ErrNotFound) {
			return URLResolution{}, err
		}
	}
	return r.fallback.ResolveByURL(ctx, marketplace, productID)
}

func (r *CombinedRepository) GetSeller(ctx context.Context, sellerID string) (domain.Seller, error) {
	if r.primary != nil {
		result, err := r.primary.GetSeller(ctx, sellerID)
		if err == nil {
			return result, nil
		}
		if !errors.Is(err, ErrNotFound) {
			return domain.Seller{}, err
		}
	}
	return r.fallback.GetSeller(ctx, sellerID)
}

func (r *CombinedRepository) GetProduct(ctx context.Context, productID string) (domain.Product, error) {
	if r.primary != nil {
		result, err := r.primary.GetProduct(ctx, productID)
		if err == nil {
			return result, nil
		}
		if !errors.Is(err, ErrNotFound) {
			return domain.Product{}, err
		}
	}
	return r.fallback.GetProduct(ctx, productID)
}

func (r *CombinedRepository) GetReviews(ctx context.Context, productID string) ([]domain.Review, error) {
	if r.primary != nil {
		result, err := r.primary.GetReviews(ctx, productID)
		if err == nil {
			return result, nil
		}
		if !errors.Is(err, ErrNotFound) {
			return nil, err
		}
	}
	return r.fallback.GetReviews(ctx, productID)
}

func (r *CombinedRepository) SaveCheckResult(ctx context.Context, result domain.TrustResponse, clientID string) error {
	if r.primary == nil {
		return nil
	}
	return r.primary.SaveCheckResult(ctx, result, clientID)
}

func (r *CombinedRepository) ListRecentChecks(ctx context.Context, limit int, clientID string) ([]domain.RecentCheck, error) {
	if r.primary != nil {
		result, err := r.primary.ListRecentChecks(ctx, limit, clientID)
		if err == nil && len(result) > 0 {
			return result, nil
		}
		if err != nil && !errors.Is(err, ErrNotFound) {
			return nil, err
		}
	}
	return []domain.RecentCheck{}, nil
}
