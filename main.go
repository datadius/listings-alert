package main

import (
	"bytes"
	"encoding/json"
	"github.com/go-co-op/gocron/v2"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
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
		log.Println("Response Body: ", string(responseBody))
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
	log.Printf("Slice size %d", len(tradePairs))

	return tradePairs
}

type ListingDiscordMessage struct {
	Content string `json:"content"`
}

func SendDiscordMessage(tradePairs []string, category string) {
	// Insurance to avoid sending too many messages to discord
	// There is a issue with bybit API where I get an empty response and then it thinks there are more coins
	if len(tradePairs) < 5 {
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
}

func main() {
	pathSpot := "/data/spot_pairs.json"
	pathFutures := "/data/futures_pairs.json"

	if runtime.GOOS == "windows" {
		log.Println("Running on Windows")
		pathSpot = "./data/spot_pairs.json"
		pathFutures = "./data/futures_pairs.json"
	}

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

	scheduler, err := gocron.NewScheduler()
	if err != nil {
		log.Println("Error creating scheduler: ", err)
	}

	j, err := scheduler.NewJob(
		gocron.CronJob("* * * * *", false),
		gocron.NewTask(func(pathSpot, pathFutures string) {
			spotPairs := getTradePairs(urlSpot)
			futuresPairs := getTradePairs(urlFutures)

			if _, err := os.Stat(pathSpot); err == nil {
				oldSpotPairs := ReadFromFile(pathSpot)

				if len(spotPairs) != 0 {
					diffSpot := Difference(spotPairs, oldSpotPairs)

					SendDiscordMessage(diffSpot, "Spot")
				}

			}
			if _, err := os.Stat(pathFutures); err == nil {
				oldFuturesPairs := ReadFromFile(pathFutures)

				if len(futuresPairs) != 0 {
					diffFutures := Difference(futuresPairs, oldFuturesPairs)

					SendDiscordMessage(diffFutures, "Futures")
				}
			}

			SaveToFile(spotPairs, pathSpot)
			SaveToFile(futuresPairs, pathFutures)
		}, pathSpot, pathFutures),
	)

	if err != nil {
		log.Println("Issue creating cron job: ", err)
	}

	log.Println("Starting cron job ", j.ID())

	scheduler.Start()

	// block until you are ready to shut down
	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)
	log.Println("Blocking, press ctrl+c to exit...")
	select {
	case <-done:
	}

	// when you're done, shut it down
	err = scheduler.Shutdown()
	if err != nil {
		// handle error
	}
}
