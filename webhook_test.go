package main

import (
	"testing"
)

func TestWebhook(t *testing.T) {
	tests := []struct {
		input_listing Listing
		input_symbols [][]string
		want          string
	}{
		{BinanceListing, [][]string{{"(1000SATS)", "SATS"}}, ""},
		{UpbitListing, [][]string{{"(ALT,", "ALT"}, {" PYTH)", "PYTH"}}, ""},
		{
			BinanceFuturesListing,
			[][]string{{"Binance Futures Will Launch USDⓈ-M TON Perpetual Contract", "TON"}},
			"",
		},
		{BithumbListing, [][]string{{"(PYTH)", "PYTH"}}, ""},
	}

	for _, test := range tests {
		got := send_discord_message(test.input_listing, test.input_symbols)
		if got != test.want {
			t.Errorf(
				"send_webhook(%q, %q) = %q; want %q",
				test.input_listing,
				test.input_symbols,
				got,
				test.want,
			)
		}
	}
}
