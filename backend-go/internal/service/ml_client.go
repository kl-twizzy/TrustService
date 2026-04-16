package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"seller-trust-map/backend-go/internal/domain"
)

type MLClient struct {
	baseURL    string
	httpClient *http.Client
}

type mlAnalyzeRequest struct {
	Seller  domain.Seller   `json:"seller"`
	Product domain.Product  `json:"product"`
	Reviews []domain.Review `json:"reviews"`
}

type mlAnalyzeResponse struct {
	FakeReviewRisk     float64  `json:"fake_review_risk"`
	RatingManipulation float64  `json:"rating_manipulation_risk"`
	TextSimilarityRisk float64  `json:"text_similarity_risk"`
	ReviewBurstRisk    float64  `json:"review_burst_risk"`
	Warnings           []string `json:"warnings"`
	TrustPenalty       float64  `json:"trust_penalty"`
	AuthenticityScore  float64  `json:"authenticity_score"`
}

func NewMLClient() *MLClient {
	baseURL := os.Getenv("ML_SERVICE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8001"
	}

	return &MLClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (c *MLClient) Analyze(ctx context.Context, seller domain.Seller, product domain.Product, reviews []domain.Review) (domain.AnalysisFactors, error) {
	payload := mlAnalyzeRequest{
		Seller:  seller,
		Product: product,
		Reviews: reviews,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return domain.AnalysisFactors{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/analyze", bytes.NewBuffer(body))
	if err != nil {
		return domain.AnalysisFactors{}, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return domain.AnalysisFactors{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return domain.AnalysisFactors{}, fmt.Errorf("ml service returned status %d", resp.StatusCode)
	}

	var result mlAnalyzeResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return domain.AnalysisFactors{}, err
	}

	return domain.AnalysisFactors{
		FakeReviewRisk:     result.FakeReviewRisk,
		RatingManipulation: result.RatingManipulation,
		TextSimilarityRisk: result.TextSimilarityRisk,
		ReviewBurstRisk:    result.ReviewBurstRisk,
		Warnings:           result.Warnings,
		TrustPenalty:       result.TrustPenalty,
		AuthenticityScore:  result.AuthenticityScore,
	}, nil
}
