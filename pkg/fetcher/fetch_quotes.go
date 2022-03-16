/*
 * Copyright 2021 M1K
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

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	finnhub "github.com/Finnhub-Stock-API/finnhub-go/v2"
	"github.com/jmoiron/jsonq"
	"github.com/m1k8/harpe/pkg/config"
	harpeTypes "github.com/m1k8/harpe/pkg/types"
	"github.com/superoo7/go-gecko/v3/types"

	coingecko "github.com/superoo7/go-gecko/v3"
)

type jsonPrice struct {
	Price float64
}

var initOnce = sync.Once{}
var fetcher *DefaultFetcher

type DefaultFetcher struct {
	coinCacheMap map[string]types.CoinsListItem

	once sync.Once

	Endpoint string
	Key      string

	finn string
}

func NewFetcher() *DefaultFetcher {

	initOnce.Do(func() {
		cfgFile, err := os.Open("config.json")

		if err != nil {
			panic("Unable to open config.json!")
		}

		defer cfgFile.Close()

		byteValue, err := ioutil.ReadAll(cfgFile)

		if err != nil {
			panic("Error reading config.json!")
		}

		var cfg config.Config
		json.Unmarshal(byteValue, &cfg)

		if cfg.StocksCFG.Key == "" {
			panic("TV not configured")
		}

		endpoint := cfg.StocksCFG.E
		key := cfg.StocksCFG.Key

		finn := cfg.StocksCFG.Finn_API

		if finn == "" {
			panic("finn nil")
		}

		fetcher = &DefaultFetcher{
			coinCacheMap: make(map[string]types.CoinsListItem),
			once:         sync.Once{},
			Endpoint:     endpoint,
			Key:          key,
			finn:         finn,
		}
	})

	if fetcher == nil {
		panic("fetcher not set")
	}

	return fetcher
}
func (d *DefaultFetcher) GetStock(stock string) (float32, error) {
	cfg := finnhub.NewConfiguration()
	cfg.AddDefaultHeader("X-Finnhub-Token", d.finn)
	finnhubClient := finnhub.NewAPIClient(cfg).DefaultApi

	lastBidAsk, req, err := finnhubClient.Quote(context.Background()).Symbol(strings.ToUpper(stock)).Execute()

	if err != nil {
		if req != nil {
			return float32(req.StatusCode), err
		} else {
			return -1, err
		}
	}

	if lastBidAsk.GetC() == 0 {
		return -1, errors.New("ticker not found")
	}

	return lastBidAsk.GetC(), nil
}

func (d *DefaultFetcher) resetCryptoCache() {
	// reset one Once object so the code to fetch the list os regathered
	d.once = sync.Once{}
}

func (d *DefaultFetcher) GetCrypto(coin string, isThisARetry bool) (float32, error) {
	// 50 calls / minute
	httpClient := &http.Client{
		Timeout: time.Second * 30,
	}
	p := coingecko.NewClient(httpClient)

	var listErr error

	d.once.Do(func() {
		d.coinCacheMap = make(map[string]types.CoinsListItem)
		tickers, listErr := p.CoinsList()

		if listErr != nil {
			d.resetCryptoCache()
			return
		}
		for _, v := range *tickers {
			// shitty shitcoin fucks things up >.<
			if strings.Contains(strings.ToLower(v.Name), "wormhole") {
				continue
			}

			d.coinCacheMap[strings.ToUpper(v.Symbol)] = v
		}
	})

	if listErr != nil {
		return -1, listErr
	}

	coin = strings.ToUpper(coin)
	if val, ok := d.coinCacheMap[coin]; ok {
		price, err := p.SimplePrice([]string{val.ID}, []string{"usd"})
		if err != nil {
			return -1, err
		}
		res := *price
		return res[val.ID]["usd"], nil
	} else {
		if isThisARetry {
			return -1, errors.New("coin not found")
		} else {
			// if we cant find a coin in the cache, it is possible that it is v new, and has therefore been added since
			// thus, we invalidate the cache and try again, passing a flag to denote that this is a retry, so we only check once
			d.resetCryptoCache()
			return d.GetCrypto(coin, true)
		}
	}
}

func (d *DefaultFetcher) GetOption(ticker, contractType, day, month, year string, price, last float32) (float32, string, error) {
	optionID := GetCode(ticker, contractType, day, month, year, price)
	url := fmt.Sprintf("https://%v/v2/last/trade/O:%v?apiKey=%v", d.Endpoint, optionID, d.Key)

	client := http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		log.Println(err)
		return -1, "", err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)

	var quoteResp harpeTypes.LastQuoteResponse
	err = json.Unmarshal(body, &quoteResp)
	if err != nil {
		return -1, "", err
	}
	if quoteResp.Status != "OK" || resp.StatusCode != 200 || quoteResp.Results.P == 0 {
		log.Printf("Error pinging Polygon, falling back to snapshot endpoint %v\n", resp.Status)

		snapshot, _, err := d.GetOptionAdvanced(ticker, contractType, day, month, year, price)

		if err != nil {
			log.Println(err)
			return last, optionID, nil
		}

		if snapshot.Results.LastQuote.Ask == 0 {
			log.Println("snapshot is also nil")
			return last, optionID, nil
		}
		// instead of fallback, try the retry lib https://github.com/cenkalti/backoff
		return float32(snapshot.Results.LastQuote.Ask), optionID, nil
	}
	return float32(quoteResp.Results.P), optionID, nil
}

func (d *DefaultFetcher) GetOptionAdvanced(ticker, contractType, day, month, year string, price float32) (*harpeTypes.Snapshot, string, error) {
	optionID := GetCode(ticker, contractType, day, month, year, price)

	url := fmt.Sprintf("https://%v/v3/snapshot/options/%v/O:%v?apiKey=%v", d.Endpoint, strings.ToUpper(ticker), optionID, d.Key)
	client := http.Client{
		Timeout: 10 * time.Second,
	}
	resp, err := client.Get(url)
	if err != nil {
		log.Println(err)
		return nil, "", err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)

	var quoteResp harpeTypes.Snapshot
	err = json.Unmarshal(body, &quoteResp)
	if err != nil {
		return nil, "", err
	}
	if quoteResp.Status != "OK" || resp.StatusCode != 200 {
		err = errors.New("error pinging Polygon - " + quoteResp.Status)
		// instead of fallback, try the retry lib https://github.com/cenkalti/backoff
		//return GetOptionYahoo(ticker, contractType, day, month, year, price)
		return nil, optionID, err
	}

	if quoteResp.Results.UnderlyingAsset.Price == 0 {
		underlying, err := d.GetStock(ticker)

		if err != nil {
			log.Println(fmt.Errorf("Unable to get underlying: %w", err))
		}
		quoteResp.Results.UnderlyingAsset.Price = float64(underlying)

	}
	return &quoteResp, optionID, nil
}

func GetOptionYahoo(ticker, contractType, day, month, year string, price float32) (float32, string, error) {
	dateStr, err := time.Parse(time.RFC3339, fmt.Sprintf("%v-%v-%vT00:00:00Z", year, month, day))
	if err != nil {
		return -1, "", err
	}
	unixTime := dateStr.Unix()
	query := fmt.Sprintf("https://query2.finance.yahoo.com/v7/finance/options/%v?date=%d", strings.ToUpper(ticker), unixTime)

	resp, err := http.Get(query)
	if err != nil {
		return -1, "", err
	}

	if resp.StatusCode != 200 {
		log.Println(resp.StatusCode)
		return -1, "", nil
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return -1, "", err
	}
	sb := string(body)

	data := make(map[string]interface{})
	decoder := json.NewDecoder(strings.NewReader(sb))
	decoder.Decode(&data)
	jq := jsonq.NewQuery(data)

	if len(contractType) == 1 {
		// to account for when C or P is passed
		contractType += "xx"
	}

	if strings.ToUpper(contractType[:1]) == "C" {
		calls, err := jq.ArrayOfObjects("optionChain", "result", "0", "options", "0", "calls")
		if err != nil {
			return -1, "", err
		}

		sortedCalls := map[string]jsonPrice{}
		for _, v := range calls {
			s := v["contractSymbol"].(string)
			sortedCalls[s] = jsonPrice{
				Price: v["lastPrice"].(float64),
			}
		}
		optionID := GetCode(ticker, contractType, day, month, year, price)
		if sortedCalls[optionID].Price == 0 {
			return -1, "", errors.New("price is 0.  Make sure the dates youve entered are valid for this stock")
		}

		return float32(sortedCalls[optionID].Price), optionID, nil

	} else if strings.ToUpper(contractType[:1]) == "P" {
		puts, err := jq.ArrayOfObjects("optionChain", "result", "0", "options", "0", "puts")
		if err != nil {
			return -1, "", err
		}

		sortedPuts := map[string]jsonPrice{}

		for _, v := range puts {
			s := v["contractSymbol"].(string)
			sortedPuts[s] = struct{ Price float64 }{
				Price: v["lastPrice"].(float64),
			}
		}
		optionID := GetCode(ticker, contractType, day, month, year, price)
		if sortedPuts[optionID].Price == 0 {
			return -1, "", errors.New("price is 0 - please try again")
		}
		return float32(sortedPuts[optionID].Price), optionID, nil
	}

	return -1, "", errors.New("invalid contract type")
}

func GetCode(ticker, contractType, day, month, year string, price float32) string {
	code := ""
	code += strings.ToUpper(ticker)
	code += year[2:]
	code += month
	code += day

	if len(contractType) > 1 {
		code += strings.ToUpper(contractType[:1])
	} else {
		code += strings.ToUpper(contractType)
	}
	price *= 1000

	code += fmt.Sprintf("%08d", int(price)) // problem?

	return code
}
