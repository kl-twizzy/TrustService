package service

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
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
	ratingPattern       = regexp.MustCompile(`"ratingValue"\s*:\s*"?([0-9]+(?:\.[0-9]+)?)"?`)
	reviewCountPattern  = regexp.MustCompile(`"reviewCount"\s*:\s*"?([0-9]+)"?`)
	spacePattern        = regexp.MustCompile(`\s+`)
	suspiciousTextParts = []string{
		"лучший лучший",
		"best best",
		"рекомендую всем",
		"топ за свои деньги",
		"пришел быстро",
	}
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
	signals := domain.PageSignals{}

	if title := deriveTitleFromURL(rawURL); title != "" {
		signals.ProductTitle = title
		if len(product.Title) == 0 || isGenericTitle(product.Title) {
			product.Title = title
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		signals.FetchError = err.Error()
		return signals, product, seller, reviews
	}

	for key, value := range defaultFetchHeaders() {
		req.Header.Set(key, value)
	}

	resp, err := f.httpClient.Do(req)
	if err != nil {
		signals.FetchError = err.Error()
		return signals, product, seller, reviews
	}
	defer resp.Body.Close()

	signals.FetchStatusCode = resp.StatusCode
	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusUnavailableForLegalReasons || resp.StatusCode == http.StatusTooManyRequests {
		signals.FetchBlocked = true
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		signals.FetchError = "marketplace page fetch returned non-2xx status"
		return signals, product, seller, reviews
	}

	html, err := readResponseBody(resp)
	if err != nil {
		signals.FetchError = err.Error()
		return signals, product, seller, reviews
	}
	signals.FetchSuccessful = true

	if title := parseTitle(html); title != "" {
		signals.ProductTitle = title
		if isGenericTitle(product.Title) {
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

func defaultFetchHeaders() map[string]string {
	return map[string]string{
		"User-Agent":                "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36",
		"Accept":                    "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8",
		"Accept-Language":           "ru-RU,ru;q=0.9,en-US;q=0.8,en;q=0.7",
		"Cache-Control":             "no-cache",
		"Pragma":                    "no-cache",
		"Upgrade-Insecure-Requests": "1",
	}
}

func readResponseBody(resp *http.Response) (string, error) {
	var reader io.Reader = resp.Body
	if strings.Contains(strings.ToLower(resp.Header.Get("Content-Encoding")), "gzip") {
		gzipReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return "", err
		}
		defer gzipReader.Close()
		reader = gzipReader
	}

	body, err := io.ReadAll(io.LimitReader(reader, 1_200_000))
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func parseTitle(html string) string {
	match := titlePattern.FindStringSubmatch(html)
	if len(match) < 2 {
		return ""
	}
	title := strings.TrimSpace(stripWhitespace(match[1]))
	title = strings.TrimSuffix(title, " купить на OZON")
	title = strings.TrimSuffix(title, " — купить в интернет-магазине Wildberries")
	title = strings.TrimSuffix(title, " — Яндекс Маркет")
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

func deriveTitleFromURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	path := strings.Trim(parsed.Path, "/")
	if path == "" {
		return ""
	}

	segments := strings.Split(path, "/")
	last := segments[len(segments)-1]
	last = strings.TrimSuffix(last, ".aspx")
	if last == "" {
		return ""
	}

	if strings.HasPrefix(last, "product--") {
		last = strings.TrimPrefix(last, "product--")
	}
	if strings.HasPrefix(last, "catalog") && len(segments) >= 2 {
		last = segments[len(segments)-2]
	}

	last = stripTrailingID(last)
	last = strings.ReplaceAll(last, "-", " ")
	last = strings.ReplaceAll(last, "_", " ")
	last = stripWhitespace(last)
	if last == "" {
		return ""
	}
	return strings.Title(last)
}

func stripTrailingID(value string) string {
	value = strings.Trim(value, "/")
	idx := strings.LastIndex(value, "-")
	if idx == -1 {
		return value
	}
	suffix := value[idx+1:]
	if _, err := strconv.Atoi(suffix); err == nil {
		return value[:idx]
	}
	return value
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
	return spacePattern.ReplaceAllString(strings.TrimSpace(strings.ReplaceAll(value, "\n", " ")), " ")
}

func isGenericTitle(title string) bool {
	lower := strings.ToLower(strings.TrimSpace(title))
	return lower == "" || strings.Contains(lower, "product ")
}
