package main

import (
	"testing"
)

func TestTitle(t *testing.T) {
	tests := []struct {
		input        string
		want_symbols [][]string
		want_listing Listing
	}{ // Empty title
		{"", [][]string{{""}}, NoListing},
		{
			"Binance Will List BRC-20 Sats (1000SATS) with Seed Tag Applied",
			[][]string{{"(1000SATS)", "SATS"}},
			BinanceListing,
		},
		{
			"KRW, BTC 마켓 디지털 자산 추가 (ALT, PYTH)",
			[][]string{{"(ALT,", "ALT"}, {" PYTH)", "PYTH"}},
			UpbitListing,
		},
		{
			"Binance Futures Will Launch USDⓈ-M TON Perpetual Contract With Up to 50x Leverage",
			[][]string{{"Binance Futures Will Launch USDⓈ-M TON Perpetual Contract", "TON"}},
			BinanceFuturesListing,
		},
		{
			"피스 네트워크(PYTH) 원화 마켓 추가 안내",
			[][]string{{"(PYTH)", "PYTH"}},
			BithumbListing,
		},
	}

	for _, test := range tests {
		symbols, listing, err := parse_title(test.input)

		if (listing != test.want_listing) &&
			(err != nil) {
			t.Errorf(
				"Listing %q not equal to expected %q",
				listing,
				test.want_listing,
			)
		}

		if len(symbols) != len(test.want_symbols) {
			t.Errorf(
				"symbols length %d not equal to expected %d",
				len(symbols),
				len(test.want_symbols),
			)
			t.Errorf(
				"symbols %q not equal to expected %q",
				symbols,
				test.want_symbols)
		}

		if (len(symbols) > 0) && (len(symbols) == len(test.want_symbols)) &&
			(listing != NoListing) {
			for index, symbol := range symbols {

				if (symbol[0] != test.want_symbols[index][0]) ||
					(symbol[1] != test.want_symbols[index][1]) {
					t.Errorf("symbol %q not equal to expected %q", symbol, test.want_symbols[index])
				}
			}
		}
	}
}
