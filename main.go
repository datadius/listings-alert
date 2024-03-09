package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
)

type TradePairsResponse struct {
	Result struct {
		List []struct {
			Symbol string `json:"symbol"`
		} `json:"list"`
	} `json:"result"`
}

func getTradePairs(url url.URL) []string {
	var req *http.Response
	req, err := http.Get(url.String())
	if err != nil {
		log.Fatalln(err)
	}
	defer func() {
		err := req.Body.Close()
		if err != nil {
			log.Println("Body close: ", err)
		}
	}()

	responseBody, err := io.ReadAll(req.Body)
	if err != nil {
		log.Println("Read response body: ", err)
	}

	if req.StatusCode >= 400 && req.StatusCode <= 500 {
		log.Println("Error response. Status Code: ", req.StatusCode)
	}

	var tradePairs TradePairsResponse
	err = json.Unmarshal(responseBody, &tradePairs)
	if err != nil {
		log.Println("Unmarshal:", err)
	}

	tradePairsList := []string{}
	for _, pair := range tradePairs.Result.List {
		// Filter out pairs with USDT
		filterPairs := strings.Contains(pair.Symbol, "USDT") &&
			!strings.Contains(pair.Symbol, "2LUSDT") &&
			!strings.Contains(pair.Symbol, "2SUSDT") &&
			!strings.Contains(pair.Symbol, "3LUSDT") &&
			!strings.Contains(pair.Symbol, "3SUSDT")

		if filterPairs {
			tradePairsList = append(tradePairsList, pair.Symbol)
		}
	}

	return tradePairsList
}

func Difference(a, b []string) (diff []string) {
	m := make(map[string]bool)

	for _, item := range b {
		m[item] = true
	}

	for _, item := range a {
		if _, ok := m[item]; !ok {
			diff = append(diff, item)
		}
	}
	return
}

func SaveToFile(data []string, filename string) {
	file, err := os.Create(filename)
	if err != nil {
		log.Println("Create file: ", err)
	}
	defer func() {
		err := file.Close()
		if err != nil {
			log.Println("File close: ", err)
		}
	}()

	json_file, err := json.MarshalIndent(data, "", "\t")
	if err != nil {
		log.Fatalf("Error marshaling file %s", err)
	}
	_, err = io.Copy(file, bytes.NewReader(json_file))
	if err != nil {
		log.Println("Write to file: ", err)
	}

	log.Printf("File %s created", filename)
}

func ReadFromFile(filename string) []string {
	file, err := os.Open(filename)
	if err != nil {
		log.Println("Open file: ", err)
		os.Exit(1)
	}
	defer func() {
		err := file.Close()
		if err != nil {
			log.Println("File close: ", err)
		}
	}()

	fileData, err := io.ReadAll(file)
	if err != nil {
		log.Println("Read file: ", err)
	}

	var tradePairs []string
	err = json.Unmarshal(fileData, &tradePairs)
	if err != nil {
		log.Println("Unmarshal:", err)
	}

	log.Printf("File %s read", filename)
	log.Printf("Slice size %s", len(tradePairs))

	return tradePairs
}

type ListingDiscordMessage struct {
	Content string `json:"content"`
}

func SendDiscordMessage(tradePairs []string, category string) {
	for _, pair := range tradePairs {
		discordContent := pair + " opened for trading on Bybit " + category
		body := ListingDiscordMessage{
			Content: discordContent,
		}

		bodyBytes, err := json.Marshal(&body)
		if err != nil {
			log.Println("Marshall post message: ", err)
		}

		reader := bytes.NewReader(bodyBytes)

		var req *http.Response
		req, err = http.Post(os.Getenv("personal_test_webhook"), "application/json", reader)
		if err != nil {
			log.Println("Post:", err)
		}

		defer func() {
			err := req.Body.Close()
			if err != nil {
				log.Println("Body close: ", err)
			}
		}()

		requestBody, err := io.ReadAll(req.Body)
		if err != nil {
			log.Println("Read response body: ", err)
		}

		if req.StatusCode >= 400 && req.StatusCode <= 500 {
			log.Println("Error response. Status Code: ", req.StatusCode)
		}

		log.Printf("Discord Request Body: %s", requestBody)
	}

}

func main() {

	urlSpot := url.URL{
		Scheme:   "https",
		Host:     "api.bybit.com",
		Path:     "/v5/market/tickers",
		RawQuery: "category=spot",
	}
	urlFutures := url.URL{
		Scheme:   "https",
		Host:     "api.bybit.com",
		Path:     "/v5/market/tickers",
		RawQuery: "category=linear",
	}

	spotPairs := getTradePairs(urlSpot)
	futuresPairs := getTradePairs(urlFutures)

	if _, err := os.Stat("spot_pairs.json"); err == nil {
		oldSpotPairs := ReadFromFile("/data/spot_pairs.json")

		diffSpot := Difference(spotPairs, oldSpotPairs)

		SendDiscordMessage(diffSpot, "Spot")

	}
	if _, err := os.Stat("futures_pairs.json"); err == nil {
		oldFuturesPairs := ReadFromFile("/data/futures_pairs.json")

		diffFutures := Difference(futuresPairs, oldFuturesPairs)

		SendDiscordMessage(diffFutures, "Futures")
	}

	SaveToFile(spotPairs, "/data/spot_pairs.json")
	SaveToFile(futuresPairs, "/data/futures_pairs.json")
}
