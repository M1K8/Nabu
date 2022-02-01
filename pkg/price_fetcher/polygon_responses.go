/*
 * Copyright 2022 M1K
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package fetcher

type LastQuoteResponse struct {
	RequestID string `json:"request_id"`
	Results   struct {
		T  string  `json:"T"`
		C  []int   `json:"c"`
		F  int64   `json:"f"`
		I  string  `json:"i"`
		P  float64 `json:"p"`
		Q  int     `json:"q"`
		R  int     `json:"r"`
		S  int     `json:"s"`
		T_ int64   `json:"t"`
		X  int     `json:"x"`
	} `json:"results"`
	Status string `json:"status"`
}

type LastTradeOptionsResponse struct {
	RequestID string `json:"request_id"`
	Results   struct {
		BreakEvenPrice float64 `json:"break_even_price"`
		Day            struct {
			Close         float64 `json:"close"`
			High          float64 `json:"high"`
			LastUpdated   int64   `json:"last_updated"`
			Low           float64 `json:"low"`
			Open          float64 `json:"open"`
			PreviousClose float64 `json:"previous_close"`
			Volume        int     `json:"volume"`
			Vwap          float64 `json:"vwap"`
		} `json:"day"`
		Details struct {
			ContractType      string `json:"contract_type"`
			ExerciseStyle     string `json:"exercise_style"`
			ExpirationDate    string `json:"expiration_date"`
			SharesPerContract int    `json:"shares_per_contract"`
			StrikePrice       int    `json:"strike_price"`
			Ticker            string `json:"ticker"`
		} `json:"details"`
		Greeks struct {
			Delta float64 `json:"delta"`
			Gamma float64 `json:"gamma"`
			Theta float64 `json:"theta"`
			Vega  float64 `json:"vega"`
		} `json:"greeks"`
		ImpliedVolatility float64 `json:"implied_volatility"`
		LastQuote         struct {
			Ask         float64 `json:"ask"`
			AskSize     int     `json:"ask_size"`
			Bid         float64 `json:"bid"`
			BidSize     int     `json:"bid_size"`
			LastUpdated int64   `json:"last_updated"`
			Midpoint    float64 `json:"midpoint"`
			Timeframe   string  `json:"timeframe"`
		} `json:"last_quote"`
		OpenInterest    int `json:"open_interest"`
		UnderlyingAsset struct {
			ChangeToBreakEven float64 `json:"change_to_break_even"`
			LastUpdated       int64   `json:"last_updated"`
			Price             float64 `json:"price"`
			Ticker            string  `json:"ticker"`
			Timeframe         string  `json:"timeframe"`
		} `json:"underlying_asset"`
	} `json:"results"`
	Status string `json:"status"`
}

type Snapshot struct {
	RequestID string `json:"request_id"`
	Results   struct {
		BreakEvenPrice float64 `json:"break_even_price"`
		Day            struct {
			Close         float64 `json:"close"`
			High          float64 `json:"high"`
			LastUpdated   int64   `json:"last_updated"`
			Low           float64 `json:"low"`
			Open          float64 `json:"open"`
			PreviousClose float64 `json:"previous_close"`
			Volume        int     `json:"volume"`
			Vwap          float64 `json:"vwap"`
		} `json:"day"`
		Details struct {
			ContractType      string  `json:"contract_type"`
			ExerciseStyle     string  `json:"exercise_style"`
			ExpirationDate    string  `json:"expiration_date"`
			SharesPerContract int     `json:"shares_per_contract"`
			StrikePrice       float64 `json:"strike_price"`
			Ticker            string  `json:"ticker"`
		} `json:"details"`
		Greeks struct {
			Delta float64 `json:"delta"`
			Gamma float64 `json:"gamma"`
			Theta float64 `json:"theta"`
			Vega  float64 `json:"vega"`
		} `json:"greeks"`
		ImpliedVolatility float64 `json:"implied_volatility"`
		LastQuote         struct {
			Ask         float64 `json:"ask"`
			AskSize     int     `json:"ask_size"`
			Bid         float64 `json:"bid"`
			BidSize     int     `json:"bid_size"`
			LastUpdated int64   `json:"last_updated"`
			Midpoint    float64 `json:"midpoint"`
			Timeframe   string  `json:"timeframe"`
		} `json:"last_quote"`
		OpenInterest    int `json:"open_interest"`
		UnderlyingAsset struct {
			ChangeToBreakEven float64 `json:"change_to_break_even"`
			LastUpdated       int64   `json:"last_updated"`
			Price             float64 `json:"price"`
			Ticker            string  `json:"ticker"`
			Timeframe         string  `json:"timeframe"`
		} `json:"underlying_asset"`
	} `json:"results"`
	Status string `json:"status"`
}
