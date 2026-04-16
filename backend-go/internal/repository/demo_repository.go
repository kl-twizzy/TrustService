package repository

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"seller-trust-map/backend-go/internal/domain"
)

type DemoRepository struct{}

type URLResolution struct {
	Marketplace string
	SellerID    string
	ProductID   string
}

func NewDemoRepository() *DemoRepository {
	return &DemoRepository{}
}

func (r *DemoRepository) ResolveByURL(_ context.Context, marketplace string, productID string) (URLResolution, error) {
	resolved := map[string]URLResolution{
		"ozon:100100": {
			Marketplace: "ozon",
			SellerID:    "seller-ozon-1001",
			ProductID:   "product-ozon-100100",
		},
		"wildberries:200200": {
			Marketplace: "wildberries",
			SellerID:    "seller-wb-2002",
			ProductID:   "product-wb-200200",
		},
		"yandex_market:300300": {
			Marketplace: "yandex_market",
			SellerID:    "seller-ym-3003",
			ProductID:   "product-ym-300300",
		},
		"demo-market:product-777": {
			Marketplace: "demo-market",
			SellerID:    "seller-202",
			ProductID:   "product-777",
		},
	}

	if result, ok := resolved[marketplace+":"+productID]; ok {
		return result, nil
	}

	if marketplace == "demo-market" {
		return URLResolution{}, fmt.Errorf("unsupported demo-market url")
	}

	return URLResolution{
		Marketplace: marketplace,
		SellerID:    fmt.Sprintf("seller-%s-%s", marketplaceAlias(marketplace), productID),
		ProductID:   fmt.Sprintf("product-%s-%s", marketplaceAlias(marketplace), productID),
	}, nil
}

func (r *DemoRepository) GetSeller(_ context.Context, sellerID string) (domain.Seller, error) {
	sellers := map[string]domain.Seller{
		"seller-101": {
			ExternalID:         "seller-101",
			Name:               "TechNova",
			Marketplace:        "demo-market",
			Rating:             4.8,
			TotalOrders:        5200,
			SuccessfulOrders:   4990,
			ComplaintsCount:    37,
			ReturnsCount:       82,
			AverageReviewScore: 4.7,
		},
		"seller-202": {
			ExternalID:         "seller-202",
			Name:               "BestElectro",
			Marketplace:        "demo-market",
			Rating:             4.1,
			TotalOrders:        1400,
			SuccessfulOrders:   1180,
			ComplaintsCount:    96,
			ReturnsCount:       133,
			AverageReviewScore: 4.0,
		},
		"seller-ozon-1001": {
			ExternalID:         "seller-ozon-1001",
			Name:               "TechNova Ozon",
			Marketplace:        "ozon",
			Rating:             4.8,
			TotalOrders:        5200,
			SuccessfulOrders:   4990,
			ComplaintsCount:    37,
			ReturnsCount:       82,
			AverageReviewScore: 4.7,
		},
		"seller-wb-2002": {
			ExternalID:         "seller-wb-2002",
			Name:               "BestElectro WB",
			Marketplace:        "wildberries",
			Rating:             4.1,
			TotalOrders:        1400,
			SuccessfulOrders:   1180,
			ComplaintsCount:    96,
			ReturnsCount:       133,
			AverageReviewScore: 4.0,
		},
		"seller-ym-3003": {
			ExternalID:         "seller-ym-3003",
			Name:               "Market Gadget Hub",
			Marketplace:        "yandex_market",
			Rating:             4.6,
			TotalOrders:        2100,
			SuccessfulOrders:   1988,
			ComplaintsCount:    28,
			ReturnsCount:       49,
			AverageReviewScore: 4.5,
		},
	}

	if seller, ok := sellers[sellerID]; ok {
		return seller, nil
	}

	if generated, ok := generateSeller(sellerID); ok {
		return generated, nil
	}

	return domain.Seller{}, fmt.Errorf("seller not found")
}

func (r *DemoRepository) GetProduct(_ context.Context, productID string) (domain.Product, error) {
	products := map[string]domain.Product{
		"product-501": {
			ExternalID:    "product-501",
			SellerID:      "seller-101",
			Title:         "Wireless Headphones X",
			Category:      "electronics",
			Rating:        4.9,
			ReviewCount:   1830,
			OneStarCount:  21,
			FiveStarCount: 1650,
		},
		"product-777": {
			ExternalID:    "product-777",
			SellerID:      "seller-202",
			Title:         "Smart Watch Lite",
			Category:      "electronics",
			Rating:        4.8,
			ReviewCount:   910,
			OneStarCount:  74,
			FiveStarCount: 801,
		},
		"product-ozon-100100": {
			ExternalID:    "product-ozon-100100",
			SellerID:      "seller-ozon-1001",
			Title:         "TechNova X Wireless Headphones",
			Category:      "electronics",
			Rating:        4.9,
			ReviewCount:   1830,
			OneStarCount:  21,
			FiveStarCount: 1650,
		},
		"product-wb-200200": {
			ExternalID:    "product-wb-200200",
			SellerID:      "seller-wb-2002",
			Title:         "Smart Watch Lite",
			Category:      "electronics",
			Rating:        4.8,
			ReviewCount:   910,
			OneStarCount:  74,
			FiveStarCount: 801,
		},
		"product-ym-300300": {
			ExternalID:    "product-ym-300300",
			SellerID:      "seller-ym-3003",
			Title:         "Market Tab 11",
			Category:      "electronics",
			Rating:        4.7,
			ReviewCount:   640,
			OneStarCount:  19,
			FiveStarCount: 512,
		},
	}

	if product, ok := products[productID]; ok {
		return product, nil
	}

	if generated, ok := generateProduct(productID); ok {
		return generated, nil
	}

	return domain.Product{}, fmt.Errorf("product not found")
}

func (r *DemoRepository) GetReviews(_ context.Context, productID string) ([]domain.Review, error) {
	now := time.Now()
	reviews := map[string][]domain.Review{
		"product-501": {
			{ExternalID: "r-1", Rating: 5, Text: "Excellent item, fast shipping, recommended", IsVerified: true, CreatedAt: now.AddDate(0, 0, -10).Format(time.RFC3339)},
			{ExternalID: "r-2", Rating: 5, Text: "Very good purchase, everything works", IsVerified: true, CreatedAt: now.AddDate(0, 0, -9).Format(time.RFC3339)},
			{ExternalID: "r-3", Rating: 4, Text: "Good sound, but the case gets dirty quickly", IsVerified: true, CreatedAt: now.AddDate(0, 0, -8).Format(time.RFC3339)},
			{ExternalID: "r-4", Rating: 5, Text: "Battery life is strong and pairing is quick", IsVerified: true, CreatedAt: now.AddDate(0, 0, -5).Format(time.RFC3339)},
		},
		"product-777": {
			{ExternalID: "r-5", Rating: 5, Text: "Best product ever best best best", IsVerified: false, CreatedAt: now.AddDate(0, 0, -2).Format(time.RFC3339)},
			{ExternalID: "r-6", Rating: 5, Text: "Best product ever best best best", IsVerified: false, CreatedAt: now.AddDate(0, 0, -2).Format(time.RFC3339)},
			{ExternalID: "r-7", Rating: 5, Text: "Best product ever best best best", IsVerified: false, CreatedAt: now.AddDate(0, 0, -2).Format(time.RFC3339)},
			{ExternalID: "r-8", Rating: 1, Text: "Stopped working after one week", IsVerified: true, CreatedAt: now.AddDate(0, 0, -1).Format(time.RFC3339)},
		},
		"product-ozon-100100": {
			{ExternalID: "oz-r-1", Rating: 5, Text: "Original item and fast delivery", IsVerified: true, CreatedAt: now.AddDate(0, 0, -10).Format(time.RFC3339)},
			{ExternalID: "oz-r-2", Rating: 4, Text: "Sound quality is good, case is easy to scratch", IsVerified: true, CreatedAt: now.AddDate(0, 0, -8).Format(time.RFC3339)},
			{ExternalID: "oz-r-3", Rating: 5, Text: "Seller looks reliable, packaging was intact", IsVerified: true, CreatedAt: now.AddDate(0, 0, -7).Format(time.RFC3339)},
		},
		"product-wb-200200": {
			{ExternalID: "wb-r-1", Rating: 5, Text: "Best product ever best best best", IsVerified: false, CreatedAt: now.AddDate(0, 0, -2).Format(time.RFC3339)},
			{ExternalID: "wb-r-2", Rating: 5, Text: "Best product ever best best best", IsVerified: false, CreatedAt: now.AddDate(0, 0, -2).Format(time.RFC3339)},
			{ExternalID: "wb-r-3", Rating: 1, Text: "Stopped working after one week", IsVerified: true, CreatedAt: now.AddDate(0, 0, -1).Format(time.RFC3339)},
		},
		"product-ym-300300": {
			{ExternalID: "ym-r-1", Rating: 5, Text: "Stable performance and fast delivery", IsVerified: true, CreatedAt: now.AddDate(0, 0, -7).Format(time.RFC3339)},
			{ExternalID: "ym-r-2", Rating: 4, Text: "Good screen and decent battery", IsVerified: true, CreatedAt: now.AddDate(0, 0, -5).Format(time.RFC3339)},
			{ExternalID: "ym-r-3", Rating: 5, Text: "Seller responded quickly, item matches description", IsVerified: true, CreatedAt: now.AddDate(0, 0, -4).Format(time.RFC3339)},
		},
	}

	if productReviews, ok := reviews[productID]; ok {
		return productReviews, nil
	}

	if generated, ok := generateReviews(productID); ok {
		return generated, nil
	}

	return nil, fmt.Errorf("reviews not found")
}

func (r *DemoRepository) SaveCheckResult(_ context.Context, _ domain.TrustResponse, _ string) error {
	return nil
}

func (r *DemoRepository) ListRecentChecks(_ context.Context, _ int, _ string) ([]domain.RecentCheck, error) {
	return []domain.RecentCheck{}, nil
}

func generateSeller(sellerID string) (domain.Seller, bool) {
	parts := strings.Split(sellerID, "-")
	if len(parts) < 3 || parts[0] != "seller" {
		return domain.Seller{}, false
	}

	marketplace := marketplaceFromAlias(parts[1])
	if marketplace == "" {
		return domain.Seller{}, false
	}

	productNumericID := parts[len(parts)-1]
	seed := numericSeed(productNumericID)
	totalOrders := 700 + seed%4300
	successfulOrders := totalOrders - (20 + seed%180)
	complaints := 8 + seed%90
	returns := 15 + seed%120
	rating := 4.1 + float64(seed%9)/10.0
	if rating > 4.9 {
		rating = 4.9
	}

	return domain.Seller{
		ExternalID:         sellerID,
		Name:               generatedSellerName(marketplace),
		Marketplace:        marketplace,
		Rating:             rating,
		TotalOrders:        totalOrders,
		SuccessfulOrders:   successfulOrders,
		ComplaintsCount:    complaints,
		ReturnsCount:       returns,
		AverageReviewScore: rating - 0.1,
	}, true
}

func generateProduct(productID string) (domain.Product, bool) {
	parts := strings.Split(productID, "-")
	if len(parts) < 3 || parts[0] != "product" {
		return domain.Product{}, false
	}

	alias := parts[1]
	marketplace := marketplaceFromAlias(alias)
	if marketplace == "" {
		return domain.Product{}, false
	}

	rawID := parts[len(parts)-1]
	seed := numericSeed(rawID)
	reviewCount := 120 + seed%2600
	oneStarCount := 5 + seed%120
	fiveStarCount := reviewCount - oneStarCount - 20 - seed%70
	if fiveStarCount < 0 {
		fiveStarCount = reviewCount / 2
	}

	return domain.Product{
		ExternalID:    productID,
		SellerID:      fmt.Sprintf("seller-%s-%s", alias, rawID),
		Title:         fmt.Sprintf("%s product %s", generatedMarketplaceLabel(marketplace), rawID),
		Category:      generatedCategory(seed),
		Rating:        4.0 + float64(seed%10)/10.0,
		ReviewCount:   reviewCount,
		OneStarCount:  oneStarCount,
		FiveStarCount: fiveStarCount,
	}, true
}

func generateReviews(productID string) ([]domain.Review, bool) {
	parts := strings.Split(productID, "-")
	if len(parts) < 3 {
		return nil, false
	}

	rawID := parts[len(parts)-1]
	seed := numericSeed(rawID)
	now := time.Now()
	repetitive := seed%3 == 0

	text1 := "Delivery was on time and the packaging looked fine"
	text2 := "Quality matches the description, no major issues so far"
	text3 := "There are some doubts after the first days of use"

	if repetitive {
		text1 = "Best product ever best best best"
		text2 = "Best product ever best best best"
		text3 = "Stopped working after one week"
	}

	return []domain.Review{
		{ExternalID: rawID + "-r1", Rating: 5, Text: text1, IsVerified: seed%4 != 0, CreatedAt: now.AddDate(0, 0, -8).Format(time.RFC3339)},
		{ExternalID: rawID + "-r2", Rating: 5, Text: text2, IsVerified: seed%5 != 0, CreatedAt: now.AddDate(0, 0, -7).Format(time.RFC3339)},
		{ExternalID: rawID + "-r3", Rating: 3 + seed%3, Text: text3, IsVerified: true, CreatedAt: now.AddDate(0, 0, -1-seed%2).Format(time.RFC3339)},
	}, true
}

func numericSeed(value string) int {
	n, err := strconv.Atoi(value)
	if err == nil {
		if n < 0 {
			return -n
		}
		return n
	}

	sum := 0
	for _, r := range value {
		sum += int(r)
	}
	if sum == 0 {
		return 1
	}
	return sum
}

func marketplaceAlias(marketplace string) string {
	switch marketplace {
	case "ozon":
		return "ozon"
	case "wildberries":
		return "wb"
	case "yandex_market":
		return "ym"
	default:
		return marketplace
	}
}

func marketplaceFromAlias(alias string) string {
	switch alias {
	case "ozon":
		return "ozon"
	case "wb":
		return "wildberries"
	case "ym":
		return "yandex_market"
	default:
		return ""
	}
}

func generatedSellerName(marketplace string) string {
	switch marketplace {
	case "ozon":
		return "Ozon Marketplace Seller"
	case "wildberries":
		return "Wildberries Marketplace Seller"
	case "yandex_market":
		return "Yandex Market Seller"
	default:
		return "Marketplace Seller"
	}
}

func generatedMarketplaceLabel(marketplace string) string {
	switch marketplace {
	case "ozon":
		return "Ozon"
	case "wildberries":
		return "Wildberries"
	case "yandex_market":
		return "Yandex Market"
	default:
		return "Marketplace"
	}
}

func generatedCategory(seed int) string {
	categories := []string{"electronics", "auto", "home", "accessories"}
	return categories[seed%len(categories)]
}
