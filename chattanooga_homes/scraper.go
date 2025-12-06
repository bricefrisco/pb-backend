package chattanooga_homes

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

const (
	// Base URL with filter parameters
	baseURL = "https://my.flexmls.com/greaterchattanooganew/search/idx_links/20240916004638701725000000/listings"

	// Default filter for Chattanooga area homes (no MlsStatus filter to see all statuses)
	defaultFilter = "MlsId+Eq+'20240417141107895724000000'+And+CountyOrParish+Eq+'Hamilton','Marion'+And+PropertyType+Eq+'A'+And+CurrentPrice+Bt+250000.0,800000.0+And+\"General+Property+Information\".\"Lot+Size+Acres\"+Ge+2.0"

	// Pagination settings
	pageLimit    = 10 // listings per page
	maxPages     = 1  // only fetch first page since we poll every minute
	pageLoadWait = 8 * time.Second

	// Realistic User Agent
	userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
)

type Scraper struct{}

func NewScraper() *Scraper {
	return &Scraper{}
}

// buildURL constructs the URL for a specific page
func buildURL(page int) string {
	return fmt.Sprintf("%s?_filter=%s&list_view=summary&page=%d&_limit=%d",
		baseURL, defaultFilter, page, pageLimit)
}

// ScrapeListings fetches all listings using a headless browser
func (s *Scraper) ScrapeListings() ([]Home, error) {
	log.Println("Starting headless browser scrape...")

	// Create browser options to appear more like a real browser
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.Flag("disable-infobars", true),
		chromedp.Flag("start-maximized", true),
		chromedp.Flag("disable-extensions", true),
		chromedp.UserAgent(userAgent),
		chromedp.WindowSize(1920, 1080),
	)

	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer allocCancel()

	// Create context with logging
	ctx, cancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(log.Printf))
	defer cancel()

	// Set overall timeout
	ctx, cancel = context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	// Set extra headers
	headers := map[string]interface{}{
		"Accept":                    "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8",
		"Accept-Language":           "en-US,en;q=0.9",
		"Accept-Encoding":           "gzip, deflate, br",
		"Connection":                "keep-alive",
		"Upgrade-Insecure-Requests": "1",
		"Sec-Fetch-Dest":            "document",
		"Sec-Fetch-Mode":            "navigate",
		"Sec-Fetch-Site":            "none",
		"Sec-Fetch-User":            "?1",
		"Cache-Control":             "max-age=0",
	}

	// Enable network and set headers
	if err := chromedp.Run(ctx, network.Enable(), network.SetExtraHTTPHeaders(network.Headers(headers))); err != nil {
		log.Printf("Warning: Could not set extra headers: %v", err)
	}

	var allHomes []Home

	// Fetch all pages
	for page := 1; page <= maxPages; page++ {
		url := buildURL(page)
		log.Printf("Fetching page %d: %s", page, url)

		homes, err := s.scrapePage(ctx, url)
		if err != nil {
			log.Printf("Error scraping page %d: %v", page, err)
			continue
		}

		log.Printf("Page %d: Found %d listings", page, len(homes))

		if len(homes) == 0 {
			log.Printf("No listings on page %d, stopping pagination", page)
			break
		}

		allHomes = append(allHomes, homes...)

		// If we got fewer than the limit, we've reached the end
		if len(homes) < pageLimit {
			log.Printf("Fewer than %d listings on page %d, stopping pagination", pageLimit, page)
			break
		}
	}

	log.Printf("Total listings scraped: %d", len(allHomes))
	return allHomes, nil
}

// scrapePage scrapes a single page of listings
func (s *Scraper) scrapePage(ctx context.Context, url string) ([]Home, error) {
	var html string

	// Navigate and wait for content with longer wait times
	err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		// Wait for DOM ready and at least one listing card; avoids fixed sleeps
		chromedp.WaitReady("body", chromedp.ByQuery),
		chromedp.WaitVisible(`div.summary-card`, chromedp.ByQuery),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to navigate: %w", err)
	}

	// Try waiting for real content to appear (not challenge page)
	// We'll check multiple times if we're still getting challenge page
	maxAttempts := 5
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		err = chromedp.Run(ctx, chromedp.OuterHTML("html", &html))
		if err != nil {
			return nil, fmt.Errorf("failed to get HTML: %w", err)
		}

		// Check if we got past the challenge (real content has more HTML)
		if len(html) > 50000 || strings.Contains(html, "listing") || strings.Contains(html, "ListingId") || strings.Contains(html, "property") {
			log.Printf("Got real content on attempt %d (HTML length: %d bytes)", attempt, len(html))
			break
		}

		// Still on challenge page, wait and retry
		log.Printf("Attempt %d: Still on challenge page (HTML length: %d bytes), waiting...", attempt, len(html))
		if err := chromedp.Run(ctx, chromedp.Sleep(3*time.Second)); err != nil {
			return nil, err
		}
	}

	// Check for bot challenge page
	if len(html) < 20000 {
		if strings.Contains(html, "_fs-ch") || strings.Contains(html, "challenge") {
			log.Printf("WARNING: Detected bot challenge page")
		}
	}

	// Extract listings from HTML
	homes := extractListingsFromHTML(html)

	return homes, nil
}

// extractListingsFromHTML parses the HTML to extract listing data
func extractListingsFromHTML(html string) []Home {
	var homes []Home

	// Find all listing cards using the summary-card pattern
	// Each listing starts with: <div id="LISTING_KEY" data-standard-status="Active" ... class="summary-card listingListItem">
	listingPattern := regexp.MustCompile(`<div\s+id="(\d{26})"\s+data-standard-status="([^"]*)"\s+[^>]*data-current-price="([^"]*)"[^>]*class="summary-card[^"]*"`)
	matches := listingPattern.FindAllStringSubmatch(html, -1)

	log.Printf("Found %d listing card matches", len(matches))

	for _, match := range matches {
		if len(match) < 3 {
			continue
		}

		listingKey := match[1]
		status := match[2]
		priceStr := match[3]

		home := Home{
			ListingID: listingKey,
			Status:    strings.TrimSpace(status),
		}

		// Parse price
		if price, err := strconv.ParseFloat(priceStr, 64); err == nil {
			home.Price = int(price)
		}

		// Find the section of HTML for this listing to extract other details
		listingStart := strings.Index(html, `id="`+listingKey+`"`)
		if listingStart == -1 {
			continue
		}

		// Find next listing by looking for next <div id="26-digit-number"
		// We need to skip past the current listing's opening tag (at least 500 chars)
		nextListingPattern := regexp.MustCompile(`<div\s+id="\d{26}"`)
		searchStart := listingStart + 500 // Skip past current listing's opening tag
		if searchStart > len(html) {
			searchStart = len(html)
		}

		nextMatch := nextListingPattern.FindStringIndex(html[searchStart:])
		var listingHTML string
		if nextMatch != nil {
			listingHTML = html[listingStart : searchStart+nextMatch[0]]
		} else {
			// Take rest of content or a large chunk
			endIdx := listingStart + 20000
			if endIdx > len(html) {
				endIdx = len(html)
			}
			listingHTML = html[listingStart:endIdx]
		}

		// Debug: log the chunk size
		if len(homes) == 0 {
			log.Printf("First listing HTML chunk size: %d bytes", len(listingHTML))
		}

		// Extract street address (line-one)
		streetPattern := regexp.MustCompile(`<div class="line-one">([^<]+)</div>`)
		if streetMatch := streetPattern.FindStringSubmatch(listingHTML); len(streetMatch) > 1 {
			home.Street = strings.TrimSpace(streetMatch[1])
		}

		// Extract city, state, zip (line-two)
		lineTwo := regexp.MustCompile(`<div class="line-two">([^<]+)</div>`)
		if lineTwoMatch := lineTwo.FindStringSubmatch(listingHTML); len(lineTwoMatch) > 1 {
			parts := strings.Split(strings.TrimSpace(lineTwoMatch[1]), ", ")
			if len(parts) >= 1 {
				home.City = parts[0]
			}
			if len(parts) >= 2 {
				// State and zip are in "TN 37308" format
				stateZip := strings.Split(parts[1], " ")
				if len(stateZip) >= 1 {
					home.State = stateZip[0]
				}
				if len(stateZip) >= 2 {
					home.Zip = stateZip[1]
				}
			}
		}

		// Extract data fields from title/value pairs
		home.SubType = extractDataValue(listingHTML, "Sub Type")
		home.County = extractDataValue(listingHTML, "County")
		home.Area = extractDataValue(listingHTML, "Area")
		home.Subdivision = extractDataValue(listingHTML, "Subdivision")

		// Living Area (numeric)
		if livingArea := extractDataValue(listingHTML, "Living Area"); livingArea != "" {
			cleanedLA := strings.ReplaceAll(livingArea, ",", "")
			if la, err := strconv.Atoi(cleanedLA); err == nil {
				home.LivingArea = la
			}
		}

		// Beds Total (numeric)
		if beds := extractDataValue(listingHTML, "Beds Total"); beds != "" {
			if b, err := strconv.Atoi(beds); err == nil {
				home.BedsTotal = b
			}
		}

		// Baths Total (numeric, can be float)
		if baths := extractDataValue(listingHTML, "Baths Total"); baths != "" {
			if b, err := strconv.ParseFloat(baths, 64); err == nil {
				home.BathsTotal = b
			}
		}

		// Acres (float)
		if acres := extractDataValue(listingHTML, "Acres"); acres != "" {
			if a, err := strconv.ParseFloat(acres, 64); err == nil {
				home.Acres = a
			}
		}

		// Year Built (numeric)
		if yearBuilt := extractDataValue(listingHTML, "Year Built"); yearBuilt != "" {
			if yb, err := strconv.Atoi(yearBuilt); err == nil {
				home.YearBuilt = yb
			}
		}

		// Extract first image URL (main listing image)
		// FlexMLS often uses data-src/src with sparkplatform CDN; support both and protocol-relative URLs.
		imgPattern := regexp.MustCompile(`<img[^>]+(?:data-src|src)="([^"]*sparkplatform[^"]+)"`)
		if imgMatch := imgPattern.FindStringSubmatch(listingHTML); len(imgMatch) > 1 {
			imgURL := imgMatch[1]
			if strings.HasPrefix(imgURL, "//") {
				imgURL = "https:" + imgURL
			}
			home.ImageURL = imgURL
		}

		// Build URL for this listing
		home.URL = fmt.Sprintf("https://my.flexmls.com/greaterchattanooganew/search/idx_links/20240916004638701725000000/listings/%s", listingKey)

		homes = append(homes, home)
	}

	return homes
}

// extractDataValue extracts the value for a given title from the data-row pattern
func extractDataValue(html, title string) string {
	// Pattern: <div class="title" title="Title">Title</div>\n<div class="value" title="Value">Value</div>
	pattern := regexp.MustCompile(`<div class="title"[^>]*>` + regexp.QuoteMeta(title) + `</div>\s*<div class="value"[^>]*title="([^"]*)"`)
	if match := pattern.FindStringSubmatch(html); len(match) > 1 {
		return strings.TrimSpace(match[1])
	}
	return ""
}
