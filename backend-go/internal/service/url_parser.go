package service

import (
	"errors"
	"net/url"
	"regexp"
	"strings"
)

var digitsPattern = regexp.MustCompile(`\d+`)

type ParsedProductURL struct {
	Marketplace string
	ProductID   string
	Normalized  string
}

func ParseMarketplaceProductURL(rawURL string) (ParsedProductURL, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ParsedProductURL{}, err
	}

	host := strings.ToLower(parsed.Hostname())
	path := strings.Trim(parsed.EscapedPath(), "/")

	switch {
	case strings.Contains(host, "ozon.ru"):
		id := lastDigits(path)
		if id == "" {
			return ParsedProductURL{}, errors.New("cannot extract OZON product id from url")
		}
		return ParsedProductURL{
			Marketplace: "ozon",
			ProductID:   id,
			Normalized:  "https://www.ozon.ru/" + path,
		}, nil
	case strings.Contains(host, "wildberries.ru"):
		id := firstDigits(path)
		if id == "" {
			return ParsedProductURL{}, errors.New("cannot extract Wildberries product id from url")
		}
		return ParsedProductURL{
			Marketplace: "wildberries",
			ProductID:   id,
			Normalized:  "https://www.wildberries.ru/" + path,
		}, nil
	case strings.Contains(host, "market.yandex.ru"):
		id := lastDigits(path)
		if id == "" {
			return ParsedProductURL{}, errors.New("cannot extract Yandex Market product id from url")
		}
		return ParsedProductURL{
			Marketplace: "yandex_market",
			ProductID:   id,
			Normalized:  "https://market.yandex.ru/" + path,
		}, nil
	case strings.Contains(host, "demo-market"):
		return ParsedProductURL{
			Marketplace: "demo-market",
			ProductID:   "product-777",
			Normalized:  rawURL,
		}, nil
	default:
		return ParsedProductURL{}, errors.New("unsupported marketplace url")
	}
}

func firstDigits(value string) string {
	match := digitsPattern.FindString(value)
	return match
}

func lastDigits(value string) string {
	matches := digitsPattern.FindAllString(value, -1)
	if len(matches) == 0 {
		return ""
	}

	return matches[len(matches)-1]
}
