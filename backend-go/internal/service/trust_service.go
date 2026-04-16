package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/redis/go-redis/v9"

	"seller-trust-map/backend-go/internal/domain"
	"seller-trust-map/backend-go/internal/repository"
)

type TrustService struct {
	repo        repository.Repository
	mlClient    *MLClient
	pageFetcher *PageFetcher
	cache       *redis.Client
}

func NewTrustService(repo repository.Repository, mlClient *MLClient, pageFetcher *PageFetcher, cache *redis.Client) *TrustService {
	return &TrustService{
		repo:        repo,
		mlClient:    mlClient,
		pageFetcher: pageFetcher,
		cache:       cache,
	}
}

func (s *TrustService) Analyze(ctx context.Context, req domain.AnalyzeRequest) (domain.TrustResponse, error) {
	return s.AnalyzeWithContext(ctx, req, domain.AnalyzeContext{})
}

func (s *TrustService) AnalyzeWithContext(ctx context.Context, req domain.AnalyzeRequest, analyzeCtx domain.AnalyzeContext) (domain.TrustResponse, error) {
	cacheKey := fmt.Sprintf("trust:%s:%s:%s", req.Marketplace, req.SellerID, req.ProductID)
	if s.cache != nil {
		if cached, err := s.cache.Get(ctx, cacheKey).Result(); err == nil {
			var result domain.TrustResponse
			if json.Unmarshal([]byte(cached), &result) == nil {
				result.ClientID = analyzeCtx.ClientID
				if saveErr := s.repo.SaveCheckResult(ctx, result, analyzeCtx.ClientID); saveErr != nil && !errors.Is(saveErr, repository.ErrNotFound) {
					return domain.TrustResponse{}, saveErr
				}
				return result, nil
			}
		}
	}

	seller, err := s.repo.GetSeller(ctx, req.SellerID)
	if err != nil {
		return domain.TrustResponse{}, err
	}

	product, err := s.repo.GetProduct(ctx, req.ProductID)
	if err != nil {
		return domain.TrustResponse{}, err
	}

	reviews, err := s.repo.GetReviews(ctx, req.ProductID)
	if err != nil {
		return domain.TrustResponse{}, err
	}

	analysis, err := s.mlClient.Analyze(ctx, seller, product, reviews)
	if err != nil {
		analysis = fallbackAnalysis()
	}

	result := buildTrustResponse(req.Marketplace, seller, product, analysis)
	result.ClientID = analyzeCtx.ClientID
	if result.CheckedURL == "" {
		result.CheckedURL = fmt.Sprintf("%s:%s", req.Marketplace, req.ProductID)
	}

	if s.cache != nil {
		if payload, marshalErr := json.Marshal(result); marshalErr == nil {
			s.cache.Set(ctx, cacheKey, payload, time.Hour)
		}
	}

	if err := s.repo.SaveCheckResult(ctx, result, analyzeCtx.ClientID); err != nil && !errors.Is(err, repository.ErrNotFound) {
		return domain.TrustResponse{}, err
	}

	return result, nil
}

func (s *TrustService) AnalyzeURL(ctx context.Context, rawURL string) (domain.TrustResponse, error) {
	return s.AnalyzeURLWithContext(ctx, rawURL, domain.AnalyzeContext{})
}

func (s *TrustService) AnalyzeURLWithContext(ctx context.Context, rawURL string, analyzeCtx domain.AnalyzeContext) (domain.TrustResponse, error) {
	parsed, err := ParseMarketplaceProductURL(rawURL)
	if err != nil {
		return domain.TrustResponse{}, err
	}

	resolution, err := s.repo.ResolveByURL(ctx, parsed.Marketplace, parsed.ProductID)
	if err != nil {
		return domain.TrustResponse{}, err
	}

	seller, err := s.repo.GetSeller(ctx, resolution.SellerID)
	if err != nil {
		return domain.TrustResponse{}, err
	}

	product, err := s.repo.GetProduct(ctx, resolution.ProductID)
	if err != nil {
		return domain.TrustResponse{}, err
	}

	reviews, err := s.repo.GetReviews(ctx, resolution.ProductID)
	if err != nil {
		return domain.TrustResponse{}, err
	}

	pageSignals := domain.PageSignals{}
	if s.pageFetcher != nil {
		pageSignals, product, seller, reviews = s.pageFetcher.Enrich(ctx, parsed.Normalized, product, seller, reviews)
	}

	analysis, err := s.mlClient.Analyze(ctx, seller, product, reviews)
	if err != nil {
		analysis = fallbackAnalysis()
	}

	if pageSignals.SuspiciousTextBlocks >= 2 {
		analysis.Warnings = append(analysis.Warnings, "Suspicious repeated review-like phrases found on the product page")
		analysis.TrustPenalty += 6
		if analysis.AuthenticityScore > 6 {
			analysis.AuthenticityScore -= 6
		}
	}

	result := buildTrustResponse(parsed.Marketplace, seller, product, analysis)
	result.ClientID = analyzeCtx.ClientID
	result.PageSignals = pageSignals
	result.CheckedURL = parsed.Normalized

	cacheKey := fmt.Sprintf("trust:%s:%s:%s", parsed.Marketplace, resolution.SellerID, resolution.ProductID)
	if s.cache != nil {
		if payload, marshalErr := json.Marshal(result); marshalErr == nil {
			s.cache.Set(ctx, cacheKey, payload, time.Hour)
		}
	}

	if err := s.repo.SaveCheckResult(ctx, result, analyzeCtx.ClientID); err != nil && !errors.Is(err, repository.ErrNotFound) {
		return domain.TrustResponse{}, err
	}

	return result, nil
}

func (s *TrustService) ListRecentChecks(ctx context.Context, limit int) ([]domain.RecentCheck, error) {
	return s.ListRecentChecksForClient(ctx, limit, "")
}

func (s *TrustService) ListRecentChecksForClient(ctx context.Context, limit int, clientID string) ([]domain.RecentCheck, error) {
	if limit <= 0 {
		limit = 10
	}
	return s.repo.ListRecentChecks(ctx, limit, clientID)
}

func (s *TrustService) GetOverview(ctx context.Context) (domain.OverviewResponse, error) {
	return s.GetOverviewForClient(ctx, "")
}

func (s *TrustService) GetOverviewForClient(ctx context.Context, clientID string) (domain.OverviewResponse, error) {
	checks, err := s.repo.ListRecentChecks(ctx, 50, clientID)
	if err != nil {
		return domain.OverviewResponse{}, nil
	}

	overview := domain.OverviewResponse{
		TotalChecks: len(checks),
	}
	if len(checks) == 0 {
		return overview, nil
	}

	totalTrust := 0
	totalAuthenticity := 0
	for index, check := range checks {
		totalTrust += check.TrustScore
		totalAuthenticity += check.RatingAuthenticity
		if check.Suspicious {
			overview.SuspiciousCount++
		}
		switch check.TrustLevel {
		case "high":
			overview.HighTrustCount++
		case "medium":
			overview.MediumTrustCount++
		default:
			overview.LowTrustCount++
		}
		if index == 0 {
			overview.LastCheckedAt = check.CheckedAt
		}
	}

	overview.AverageTrustScore = round(float64(totalTrust) / float64(len(checks)))
	overview.AverageAuthenticity = round(float64(totalAuthenticity) / float64(len(checks)))
	return overview, nil
}

func buildTrustResponse(marketplace string, seller domain.Seller, product domain.Product, analysis domain.AnalysisFactors) domain.TrustResponse {
	orderSuccessRate := safeRatio(seller.SuccessfulOrders, seller.TotalOrders)
	complaintRate := safeRatio(seller.ComplaintsCount, seller.TotalOrders)
	returnRate := safeRatio(seller.ReturnsCount, seller.TotalOrders)
	fiveStarRatio := safeRatio(product.FiveStarCount, product.ReviewCount)
	oneStarRatio := safeRatio(product.OneStarCount, product.ReviewCount)

	sellerBase := int(math.Round(
		(seller.Rating/5.0)*35 +
			orderSuccessRate*35 +
			(1-complaintRate)*15 +
			(1-returnRate)*15,
	))

	productSignal := int(math.Round(
		(product.Rating/5.0)*45 +
			(1-math.Max(0, fiveStarRatio-0.75))*25 +
			(1-analysis.RatingManipulation)*15 +
			(1-analysis.FakeReviewRisk)*15,
	))

	trustScore := int(math.Round(float64(sellerBase)*0.55 + float64(productSignal)*0.45 - analysis.TrustPenalty))
	if trustScore < 0 {
		trustScore = 0
	}
	if trustScore > 100 {
		trustScore = 100
	}

	reasons := make([]string, 0, 6)
	if orderSuccessRate > 0.95 {
		reasons = append(reasons, "High share of successfully completed orders")
	}
	if complaintRate > 0.05 {
		reasons = append(reasons, "Elevated complaint rate for the seller")
	}
	if fiveStarRatio > 0.85 {
		reasons = append(reasons, "Unusually high share of five-star reviews")
	}
	if oneStarRatio > 0.08 && product.Rating > 4.6 {
		reasons = append(reasons, "Product rating looks too high compared with one-star feedback share")
	}
	if analysis.TextSimilarityRisk > 0.65 {
		reasons = append(reasons, "Review texts are too similar to each other")
	}
	if analysis.ReviewBurstRisk > 0.60 {
		reasons = append(reasons, "Suspicious burst of review activity detected")
	}
	reasons = append(reasons, analysis.Warnings...)

	level := trustLevel(trustScore)
	return domain.TrustResponse{
		Marketplace: marketplace,
		Seller: domain.SellerSummary{
			ID:               seller.ExternalID,
			Name:             seller.Name,
			Rating:           seller.Rating,
			SuccessfulOrders: seller.SuccessfulOrders,
			ComplaintsCount:  seller.ComplaintsCount,
			ReturnsCount:     seller.ReturnsCount,
		},
		Product: domain.ProductSummary{
			ID:          product.ExternalID,
			Title:       product.Title,
			Rating:      product.Rating,
			ReviewCount: product.ReviewCount,
		},
		TrustScore:         trustScore,
		TrustLevel:         level,
		RatingAuthenticity: int(math.Round(analysis.AuthenticityScore)),
		Suspicious:         trustScore < 60 || analysis.FakeReviewRisk > 0.65 || analysis.TextSimilarityRisk > 0.7,
		Reasons:            reasons,
		Metrics: domain.TrustMetrics{
			OrderSuccessRate:   round(orderSuccessRate),
			ComplaintRate:      round(complaintRate),
			ReturnRate:         round(returnRate),
			FiveStarRatio:      round(fiveStarRatio),
			OneStarRatio:       round(oneStarRatio),
			SellerBaseScore:    sellerBase,
			ProductSignalScore: productSignal,
		},
		Analysis:       analysis,
		Recommendation: recommendation(level),
	}
}

func fallbackAnalysis() domain.AnalysisFactors {
	return domain.AnalysisFactors{
		FakeReviewRisk:     0.35,
		RatingManipulation: 0.30,
		TextSimilarityRisk: 0.25,
		ReviewBurstRisk:    0.20,
		Warnings:           []string{"ML service unavailable, fallback heuristic applied"},
		TrustPenalty:       12,
		AuthenticityScore:  68,
	}
}

func safeRatio(part int, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(part) / float64(total)
}

func round(value float64) float64 {
	return math.Round(value*1000) / 1000
}

func trustLevel(score int) string {
	switch {
	case score >= 80:
		return "high"
	case score >= 60:
		return "medium"
	default:
		return "low"
	}
}

func recommendation(level string) string {
	switch level {
	case "high":
		return "Seller looks reliable. No strong signs of rating manipulation were detected."
	case "medium":
		return "Purchase may be acceptable, but it is worth double-checking reviews and seller history."
	default:
		return "Serious risk signals detected. Consider choosing another seller."
	}
}
