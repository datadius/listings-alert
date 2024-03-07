package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"github.com/gorilla/websocket"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"regexp"
	"time"
)

type Tree_News struct {
	Title string `json:"title"`
}

type Listing string

type ListingDiscordMessage struct {
	Content string `json:"content"`
}

const (
	BinanceListing        Listing = "BinanceListing"
	UpbitListing                  = "UpbitListing"
	BinanceFuturesListing         = "BinanceFuturesListing"
	BithumbListing                = "BithumbListing"
	NoListing                     = "NoListing"
)

var addr = flag.String("addr", "news.treeofalpha.com", "http service address")

func title_regex(message string) (string, Listing, error) {
	if match, _ := regexp.MatchString("Binance Will List", message); match {
		return `[\( ]\d*(\w*)[,\)]`, BinanceListing, nil
	} else if match, _ := regexp.MatchString("마켓 디지털 자산 추가", message); match {
		return `[\( ](\w*)[,\)]`, UpbitListing, nil
	} else if match, _ := regexp.MatchString("Binance Futures Will Launch USDⓈ-M", message); match {
		return `Binance Futures Will Launch USDⓈ-M\s*(.*?)\s*Perpetual Contract`, BinanceFuturesListing, nil
	} else if match, _ := regexp.MatchString("원화 마켓 추가", message); match {
		return `[\( ](\w*)[,\)]`, BithumbListing, nil
	} else {
		return "", NoListing, nil
	}
}

func parse_title(message string) ([][]string, Listing, error) {
	regex, listing, err := title_regex(message)
	if err != nil {
		//I hate this ngl
		return nil, NoListing, err
	}

	re := regexp.MustCompile(regex)
	//find all captures group
	matches := re.FindAllStringSubmatch(message, 2)

	return matches, listing, nil
}

func main() {
	flag.Parse()
	log.SetFlags(0)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	u := url.URL{Scheme: "wss", Host: *addr, Path: "/ws"}
	log.Printf("connecting to %s", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)

	if err != nil {
		log.Fatal("dial:", err)
	}

	defer c.Close()

	done := make(chan struct{})

	discordWeebhookUrl := os.Getenv("personal_test_webhook")

	go func() {
		defer close(done)
		for {
			_, message, err := c.ReadMessage()

			if err != nil {
				log.Println("read:", err)
				return
			}

			var tree_news Tree_News
			err = json.Unmarshal(message, &tree_news)
			if err != nil {
				log.Println("Unmarshal:", err)
				return
			}

			symbols, listing, err := parse_title(tree_news.Title)
			if err != nil {
				log.Println("parse_title:", err)
				return
			}

			log.Printf("recv: %s, listing: %s, symbol: %s", tree_news.Title, listing, symbols)

			body := ListingDiscordMessage{
				Content: tree_news.Title,
			}

			bodyBytes, err := json.Marshal(&body)
			if err != nil {
				log.Println("Marshall post message: ", err)
			}

			reader := bytes.NewReader(bodyBytes)

			var req *http.Response
			req, err = http.Post(discordWeebhookUrl, "application/json", reader)
			if err != nil {
				log.Println("Post:", err)
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

			log.Println("Response: ", string(responseBody))
		}
	}()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case t := <-ticker.C:
			err := c.WriteMessage(websocket.TextMessage, []byte(t.String()))
			if err != nil {
				log.Println("write:", err)
				return
			}
		case <-interrupt:
			log.Println("interrupt")

			err := c.WriteMessage(
				websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
			)
			if err != nil {
				log.Println("write close:", err)
				return
			}
			select {
			case <-done:
			case <-time.After(time.Second):
			}
			return
		}
	}
}
