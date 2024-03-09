package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
)

type TradePairsResponse struct {
	Result struct {
		List []struct {
			Symbol string `json:"symbol"`
		} `json:"list"`
	} `json:"result"`
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

	log.Println("Spot Pairs: ", strings.Join(spotPairs, ", "))
	log.Println("Futures Pairs: ", strings.Join(futuresPairs, ", "))
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
