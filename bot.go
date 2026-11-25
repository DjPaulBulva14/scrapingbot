package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"

	"time"

	"github.com/Noooste/azuretls-client"
)

const (
	//your JA3 value from https://tls.peet.ws/api/all
	ja3Value       = "#"
	// your HTTP2 value from https://tls.peet.ws/api/all
	http2Value     = "#"
	//use mozzila because chrome changes its code to be more secure
	userAgent      = "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:159.0) Gecko/20100101 Firefox/159.0"
	acceptLanguage = "cs,sk;q=0.9,en-US;q=0.8,en;q=0.7"
)

const (
	baseURL = "https://www.smarty.cz/"
	apiURL  = "https://www.smarty.cz/Web/GoogleAnalytics/GaInfo?errorCode=null&errorPath=null&productId=286131&orderId=null&cartId=null&orderStep=null&isDetailReservation=null"
	referer = "https://www.smarty.cz/Pokemon-TCG-30th-Celebration-Booster-Box-Japanese--4p286131"

	proxyURL = ""

	MaxRetries = 3
	pauseMS    = 1500
)

type Product struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Price        float64 `json:"fullPrice"`
	Availability string  `json:"available"`
}

type ProductResponse struct {
	Product struct {
		Page struct {
			Detail struct {
				Products []Product `json:"products"`
			} `json:"detail"`
		} `json:"page"`
	} `json:"product"`
}

func newSession() (*azuretls.Session, error) {
	session := azuretls.NewSession()

	session.Browser = azuretls.Firefox
	session.SetTimeout(20 * time.Second)

	if err := session.ApplyJa3(ja3Value, azuretls.Firefox); err != nil {
		session.Close()
		return nil, fmt.Errorf("failed to apply JA3 fingerprint: %w", err)
	}

	if err := session.ApplyHTTP2(http2Value); err != nil {
		session.Close()
		return nil, fmt.Errorf("failed to apply HTTP/2 fingerprint: %w", err)
	}

	if proxyURL != "" {
		if err := session.SetProxy(proxyURL); err != nil {
			session.Close()
			return nil, fmt.Errorf("failed to set proxy: %w", err)
		}

	}

	headers := azuretls.OrderedHeaders{
		{"User-Agent", userAgent},
		{"Accept", ""},
		{"Accept-Language", acceptLanguage},
		{"Accept-Encoding", "gzip, deflate, br, zstd"},
		{"Referer", referer},
		{"Sec-Fetch-Dest", "empty"},
		{"Sec-Fetch-Mode", "cors"},
		{"Sec-Fetch-Site", "same-origin"},
		{"TE", "trailers"},
	}

	session.OrderedHeaders = headers
	return session, nil

	/*
	   GET /Web/GoogleAnalytics/GaInfo?errorCode=null&errorPath=null&productId=286131&orderId=null&cartId=null&orderStep=null&isDetailReservation=null HTTP/2
	   Host: www.smarty.cz
	   User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:159.0) Gecko/20100101 Firefox/159.0
	   Accept:
	   Accept-Language: cs,sk;q=0.9,en-US;q=0.8,en;q=0.7
	   Accept-Encoding: gzip, deflate, br, zstd
	   Referer: https://www.smarty.cz/Pokemon-TCG-30th-Celebration-Booster-Box-Japanese--4p286131
	   X-Requested-With: XMLHttpRequest
	   Connection: keep-alive
	   Cookie: #
	   Sec-Fetch-Dest: empty
	   Sec-Fetch-Mode: cors
	   Sec-Fetch-Site: same-origin
	   TE: trailers
	*/
}



func fetchPage(session *azuretls.Session, url string) (string, error) {
	var lastErr error

	for attempt := 1; attempt <= MaxRetries; attempt++ {
		resp, err := session.Get(url)
		if err != nil {
			lastErr = err
		} else {
			body := resp.String()

			switch resp.StatusCode {
			case 200:
				return body, nil
			case 401, 403:
				return "", fmt.Errorf("status: %d, body %s", resp.StatusCode, body)
			default:
				lastErr = fmt.Errorf("status: %d", resp.StatusCode)
			}
		}

		if attempt < MaxRetries {
			time.Sleep(time.Duration(attempt+1) * 5 * time.Second)
		}

	}
	return "", fmt.Errorf("failed to fetch page after %d attempts: %w", MaxRetries, lastErr)
}

func main() {

	session, err := newSession()
	if err != nil {
	}
	defer session.Close()


	if _, err := session.Get(baseURL); err != nil {
		log.Fatalf("úvodní stránka: %v", err)
	}
	if u, err := url.Parse(apiURL); err == nil {
		for _, c := range session.CookieJar.Cookies(u) {
			fmt.Printf("Cookie: %s=%s\n", c.Name, c.Value)
		}
	}

	time.Sleep(pauseMS * time.Millisecond)

	body, err := fetchPage(session, apiURL)
	if err != nil {
		log.Fatal(err)
	}

	var data ProductResponse
	err = json.Unmarshal([]byte(body), &data)
	if err != nil {
		fmt.Println("Chyba při dekódování JSON:", err)
		return
	}

	for _, product := range data.Product.Page.Detail.Products {
		fmt.Printf("ID: %s, Price: %2f, Skladem: %s, Name: %s\n", product.ID, product.Price, product.Availability, product.Name)
	}

}
