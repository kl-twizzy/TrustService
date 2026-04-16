package domain

type AnalyzeRequest struct {
	Marketplace string `json:"marketplace"`
	SellerID    string `json:"seller_id"`
	ProductID   string `json:"product_id"`
}

type AnalyzeURLRequest struct {
	ProductURL string `json:"product_url"`
}

type AnalyzeContext struct {
	ClientID string
}

type Seller struct {
	ExternalID         string  `json:"external_id"`
	Name               string  `json:"name"`
	Marketplace        string  `json:"marketplace"`
	Rating             float64 `json:"rating"`
	TotalOrders        int     `json:"total_orders"`
	SuccessfulOrders   int     `json:"successful_orders"`
	ComplaintsCount    int     `json:"complaints_count"`
	ReturnsCount       int     `json:"returns_count"`
	AverageReviewScore float64 `json:"average_review_score"`
}

type Product struct {
	ExternalID    string  `json:"external_id"`
	SellerID      string  `json:"seller_id"`
	Title         string  `json:"title"`
	Category      string  `json:"category"`
	Rating        float64 `json:"rating"`
	ReviewCount   int     `json:"review_count"`
	OneStarCount  int     `json:"one_star_count"`
	FiveStarCount int     `json:"five_star_count"`
}

type Review struct {
	ExternalID string `json:"external_id"`
	Rating     int    `json:"rating"`
	Text       string `json:"text"`
	IsVerified bool   `json:"is_verified"`
	CreatedAt  string `json:"created_at"`
}

type AnalysisFactors struct {
	FakeReviewRisk     float64  `json:"fake_review_risk"`
	RatingManipulation float64  `json:"rating_manipulation_risk"`
	TextSimilarityRisk float64  `json:"text_similarity_risk"`
	ReviewBurstRisk    float64  `json:"review_burst_risk"`
	Warnings           []string `json:"warnings"`
	TrustPenalty       float64  `json:"trust_penalty"`
	AuthenticityScore  float64  `json:"authenticity_score"`
}

type TrustResponse struct {
	Marketplace        string          `json:"marketplace"`
	CheckedURL         string          `json:"checked_url,omitempty"`
	ClientID           string          `json:"client_id,omitempty"`
	Seller             SellerSummary   `json:"seller"`
	Product            ProductSummary  `json:"product"`
	TrustScore         int             `json:"trust_score"`
	TrustLevel         string          `json:"trust_level"`
	RatingAuthenticity int             `json:"rating_authenticity"`
	Suspicious         bool            `json:"suspicious"`
	Reasons            []string        `json:"reasons"`
	Metrics            TrustMetrics    `json:"metrics"`
	Analysis           AnalysisFactors `json:"analysis"`
	PageSignals        PageSignals     `json:"page_signals"`
	Recommendation     string          `json:"recommendation"`
}

type RecentCheck struct {
	ID                 string   `json:"id"`
	Marketplace        string   `json:"marketplace"`
	CheckedURL         string   `json:"checked_url"`
	ProductTitle       string   `json:"product_title"`
	SellerName         string   `json:"seller_name"`
	TrustScore         int      `json:"trust_score"`
	TrustLevel         string   `json:"trust_level"`
	RatingAuthenticity int      `json:"rating_authenticity"`
	Suspicious         bool     `json:"suspicious"`
	Reasons            []string `json:"reasons"`
	CheckedAt          string   `json:"checked_at"`
}

type PageSignals struct {
	FetchSuccessful      bool    `json:"fetch_successful"`
	ProductTitle         string  `json:"product_title,omitempty"`
	SellerName           string  `json:"seller_name,omitempty"`
	ParsedRating         float64 `json:"parsed_rating,omitempty"`
	ParsedReviewCount    int     `json:"parsed_review_count,omitempty"`
	StructuredDataFound  bool    `json:"structured_data_found"`
	SuspiciousTextBlocks int     `json:"suspicious_text_blocks"`
}

type OverviewResponse struct {
	TotalChecks         int     `json:"total_checks"`
	HighTrustCount      int     `json:"high_trust_count"`
	MediumTrustCount    int     `json:"medium_trust_count"`
	LowTrustCount       int     `json:"low_trust_count"`
	SuspiciousCount     int     `json:"suspicious_count"`
	AverageTrustScore   float64 `json:"average_trust_score"`
	AverageAuthenticity float64 `json:"average_authenticity"`
	LastCheckedAt       string  `json:"last_checked_at,omitempty"`
}

type SellerSummary struct {
	ID               string  `json:"id"`
	Name             string  `json:"name"`
	Rating           float64 `json:"rating"`
	SuccessfulOrders int     `json:"successful_orders"`
	ComplaintsCount  int     `json:"complaints_count"`
	ReturnsCount     int     `json:"returns_count"`
}

type ProductSummary struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Rating      float64 `json:"rating"`
	ReviewCount int     `json:"review_count"`
}

type TrustMetrics struct {
	OrderSuccessRate   float64 `json:"order_success_rate"`
	ComplaintRate      float64 `json:"complaint_rate"`
	ReturnRate         float64 `json:"return_rate"`
	FiveStarRatio      float64 `json:"five_star_ratio"`
	OneStarRatio       float64 `json:"one_star_ratio"`
	SellerBaseScore    int     `json:"seller_base_score"`
	ProductSignalScore int     `json:"product_signal_score"`
}
