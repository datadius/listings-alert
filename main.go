package main

import (
	"encoding/json"
	"flag"
	"github.com/gorilla/websocket"
	"log"
	"net/url"
	"os"
	"os/signal"
	"regexp"
	"time"
)

type Tree_News struct {
	title string
}

type Listing string

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
		return `\([\p{Nd}]*([^()]+)\)`, BinanceListing, nil
	} else if match, _ := regexp.MatchString("마켓 디지털 자산 추가", message); match {
		return `[\( ](\w*)[,\)]`, UpbitListing, nil
	} else if match, _ := regexp.MatchString("Binance Futures Will Launch USDⓈ-M", message); match {
		return `Binance Futures Will Launch USDⓈ-M\s*(.*?)\s*Perpetual Contract`, BinanceFuturesListing, nil
	} else if match, _ := regexp.MatchString("원화 마켓 추가", message); match {
		return `\([\p{Nd}]*([^()]+)\)`, BithumbListing, nil
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
	matches := re.FindAllStringSubmatch(message, -1)
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

			symbols, listing, err := parse_title(tree_news.title)
			if err != nil {
				log.Println("parse_title:", err)
				return
			}

			for _, symbol := range symbols {
				log.Printf("recv: %s, listing: %s, symbol: %s", message, listing, symbol)
			}
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
