package service

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"seller-trust-map/backend-go/internal/domain"
)

type PageFetcher struct {
	httpClient *http.Client
}

type productStructuredData struct {
	Name            string `json:"name"`
	Brand           any    `json:"brand"`
	Description     string `json:"description"`
	AggregateRating struct {
		RatingValue string `json:"ratingValue"`
		ReviewCount string `json:"reviewCount"`
		RatingCount string `json:"ratingCount"`
	} `json:"aggregateRating"`
}

var (
	jsonLDPattern       = regexp.MustCompile(`(?is)<script[^>]*type=["']application/ld\+json["'][^>]*>(.*?)</script>`)
	titlePattern        = regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)
	suspiciousTextParts = []string{"лучший лучший", "best best", "рекомендую всем", "топ за свои деньги", "пришел быстро"}
	ratingPattern       = regexp.MustCompile(`"ratingValue"\s*:\s*"?([0-9]+(?:\.[0-9]+)?)"?`)
	reviewCountPattern  = regexp.MustCompile(`"reviewCount"\s*:\s*"?([0-9]+)"?`)
)

func NewPageFetcher() *PageFetcher {
	timeout := 6 * time.Second
	if value := os.Getenv("PAGE_FETCH_TIMEOUT_MS"); value != "" {
		if ms, err := strconv.Atoi(value); err == nil && ms > 0 {
			timeout = time.Duration(ms) * time.Millisecond
		}
	}

	return &PageFetcher{
		httpClient: &http.Client{Timeout: timeout},
	}
}

func (f *PageFetcher) Enrich(ctx context.Context, rawURL string, product domain.Product, seller domain.Seller, reviews []domain.Review) (domain.PageSignals, domain.Product, domain.Seller, []domain.Review) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return domain.PageSignals{}, product, seller, reviews
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 SellerTrustMapDiploma/1.0")

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return domain.PageSignals{}, product, seller, reviews
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return domain.PageSignals{}, product, seller, reviews
	}

	bodyBytes := make([]byte, 0)
	buf := make([]byte, 8192)
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			bodyBytes = append(bodyBytes, buf[:n]...)
			if len(bodyBytes) > 1_200_000 {
				break
			}
		}
		if readErr != nil {
			break
		}
	}
	html := string(bodyBytes)
	signals := domain.PageSignals{FetchSuccessful: true}

	if title := parseTitle(html); title != "" {
		signals.ProductTitle = title
		if len(product.Title) == 0 || strings.Contains(strings.ToLower(product.Title), "product") {
			product.Title = title
		}
	}

	if data, ok := extractStructuredData(html); ok {
		signals.StructuredDataFound = true
		if data.Name != "" {
			signals.ProductTitle = data.Name
			product.Title = data.Name
		}
		if rating := parseFloat(data.AggregateRating.RatingValue); rating > 0 {
			signals.ParsedRating = rating
			product.Rating = rating
		}
		if count := parseInt(data.AggregateRating.ReviewCount); count > 0 {
			signals.ParsedReviewCount = count
			product.ReviewCount = count
		} else if count := parseInt(data.AggregateRating.RatingCount); count > 0 {
			signals.ParsedReviewCount = count
			product.ReviewCount = count
		}
	}

	if signals.ParsedRating == 0 {
		signals.ParsedRating = parseFirstFloat(html, ratingPattern)
		if signals.ParsedRating > 0 {
			product.Rating = signals.ParsedRating
		}
	}
	if signals.ParsedReviewCount == 0 {
		signals.ParsedReviewCount = parseFirstInt(html, reviewCountPattern)
		if signals.ParsedReviewCount > 0 {
			product.ReviewCount = signals.ParsedReviewCount
		}
	}

	signals.SuspiciousTextBlocks = countSuspiciousBlocks(html)
	if signals.SuspiciousTextBlocks >= 2 {
		reviews = append(reviews, domain.Review{
			ExternalID: "page-signal-suspicious",
			Rating:     5,
			Text:       "Repeated suspicious marketing language detected on page",
			IsVerified: false,
			CreatedAt:  time.Now().Format(time.RFC3339),
		})
	}

	return signals, product, seller, reviews
}

func parseTitle(html string) string {
	match := titlePattern.FindStringSubmatch(html)
	if len(match) < 2 {
		return ""
	}
	title := strings.TrimSpace(stripWhitespace(match[1]))
	title = strings.TrimSuffix(title, " купить на OZON")
	return title
}

func extractStructuredData(html string) (productStructuredData, bool) {
	matches := jsonLDPattern.FindAllStringSubmatch(html, -1)
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		var data productStructuredData
		if json.Unmarshal([]byte(strings.TrimSpace(match[1])), &data) == nil {
			if data.Name != "" || data.AggregateRating.RatingValue != "" {
				return data, true
			}
		}
	}
	return productStructuredData{}, false
}

func parseFloat(value string) float64 {
	number, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil {
		return 0
	}
	return number
}

func parseInt(value string) int {
	number, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0
	}
	return number
}

func parseFirstFloat(text string, pattern *regexp.Regexp) float64 {
	match := pattern.FindStringSubmatch(text)
	if len(match) < 2 {
		return 0
	}
	return parseFloat(match[1])
}

func parseFirstInt(text string, pattern *regexp.Regexp) int {
	match := pattern.FindStringSubmatch(text)
	if len(match) < 2 {
		return 0
	}
	return parseInt(match[1])
}

func countSuspiciousBlocks(html string) int {
	lower := strings.ToLower(html)
	count := 0
	for _, part := range suspiciousTextParts {
		if strings.Contains(lower, part) {
			count++
		}
	}
	return count
}

func stripWhitespace(value string) string {
	return strings.Join(strings.Fields(strings.ReplaceAll(value, "\n", " ")), " ")
}
