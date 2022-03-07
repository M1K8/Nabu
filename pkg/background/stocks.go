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
package background

import (
	"fmt"
	"log"
	"math"
	"strconv"
	"time"

	"github.com/m1k8/harpe/pkg/db"
	"github.com/m1k8/harpe/pkg/utils"
)

func (b *Background) CheckStockPriceInBG(outChan chan<- Response, ticker, author, expiry string, guildID string, exit chan bool, inChan <-chan Response) {
	log.Println("Starting BG Scan for Stock " + ticker)
	tick := time.NewTicker(45 * time.Second) // slow because api is slow
	var expiryDate time.Time
	var err error = nil
	hasAlertedSPT := false
	hasPingedOverPct := map[float32]bool{ // 3, 5, 10, 15, 20, 25, 50, 100, 200
		3:   false,
		5:   false,
		10:  false,
		15:  false,
		25:  false,
		50:  false,
		100: false,
		200: false,
	}
	dbStock, err := b.Repo.GetStock(ticker)
	if err != nil {
		log.Println(fmt.Errorf("unable to get Stock from db %v: %w", ticker, err))
		return
	}
	poiHit := dbStock.StockPOIHit
	highest := dbStock.StockHighest
	trailingPct := dbStock.StockTrailingStop / 100

	defer (func() {
		log.Println("closing channel for Stock " + ticker)
		outChan <- Response{
			Type:  Exit,
			Price: highest,
		}
		//close(exit)
		//close(outChan)
	})()

	if expiry != "" {
		expiryNum, _ := strconv.ParseInt(expiry, 10, 32)
		if expiryNum != 0 {
			expiryDate = time.Now().AddDate(0, 0, int(expiryNum))
		}

	}

	if !expiryDate.IsZero() && time.Now().After(expiryDate) {
		final, _ := b.Fetcher.GetStock(ticker)
		outChan <- Response{
			Type:    Expired,
			Price:   final,
			Message: dbStock.Caller,
		}
		return
	}

	if dbStock.StockStarting <= highest {
		outChan <- Response{
			Type:    New_High,
			Price:   highest,
			PctGain: 0,
			Message: dbStock.Caller,
		}
	} else {
		outChan <- Response{
			Type:    New_High,
			Price:   dbStock.StockStarting,
			PctGain: 0,
			Message: dbStock.Caller,
		}
	}

	for {
		select {
		case <-exit:
			return
		case m := <-inChan:
			switch m.Type {
			case New_Avg:
				log.Println("Getting new Avg for stock " + ticker)
				dbStock, err = b.Repo.GetStock(ticker)
				if err != nil {
					log.Println(fmt.Errorf("unable to get Stock from db %v: %w", ticker, err))
					return
				}
				highest = dbStock.StockStarting
				hasPingedOverPct[3] = false
				hasPingedOverPct[5] = false
				hasPingedOverPct[10] = false
				hasPingedOverPct[15] = false
				hasPingedOverPct[20] = false
				hasPingedOverPct[25] = false
				hasPingedOverPct[50] = false
				hasPingedOverPct[100] = false
				hasPingedOverPct[200] = false
				outChan <- Response{
					Type:    New_Avg,
					Price:   highest,
					Message: dbStock.Caller,
				}
			}
		case <-tick.C:
			if !db.IsTradingHours() {
				if dbStock.ChannelType == utils.DAY || !expiryDate.IsZero() && time.Now().After(expiryDate) {
					outChan <- Response{
						Type:    Expired,
						Message: dbStock.Caller,
					}
					return
				}
				log.Println("Stock alert for " + ticker + " is sleeping.")
				log.Printf("now: %v, wait: %v \n", time.Now(), utils.GetTimeToOpen())
				time.Sleep(utils.GetTimeToOpen())
			}
			newPrice, err := b.Fetcher.GetStock(ticker)
			if err != nil {
				log.Println(fmt.Errorf("error for ticker %v: %w", ticker, err))
				continue
			}

			priceDiff := newPrice - dbStock.StockStarting
			if priceDiff == 0 {
				continue
			}

			if newPrice > highest {
				highest = newPrice
				outChan <- Response{
					Type:    New_High,
					Price:   newPrice,
					Message: dbStock.Caller,
				}
			}

			if dbStock.StockSPt > 0 && !hasAlertedSPT {
				if newPrice >= dbStock.StockSPt {
					outChan <- Response{
						Type:    PT1,
						Price:   newPrice,
						Message: dbStock.Caller,
					}
				}
				hasAlertedSPT = true
			}

			if dbStock.StockEPt > 0 {
				if newPrice >= dbStock.StockEPt {
					outChan <- Response{
						Type:    PT2,
						Price:   newPrice,
						Message: dbStock.Caller,
					}
					return
				}
			}

			if trailingPct != 0 && newPrice < (1-trailingPct)*highest {
				outChan <- Response{
					Type:    TSL,
					Price:   newPrice,
					Message: dbStock.Caller,
				}
				return
			}

			if dbStock.StockStop != 0 && newPrice <= dbStock.StockStop {
				outChan <- Response{
					Type:    SL,
					Price:   newPrice,
					Message: dbStock.Caller,
				}
				return
			}

			pctDiff := (priceDiff / dbStock.StockStarting) * 100

			if dbStock.StockPoI > 0.0 && !poiHit {

				if math.Abs(float64(pctDiff)) <= 0.5 {
					poiHit = true
					outChan <- Response{
						Type:    POI,
						Price:   newPrice,
						Message: dbStock.Caller,
					}
				}
			}

			if pctDiff >= 3 && pctDiff < 5 {
				if !hasPingedOverPct[3] {
					hasPingedOverPct[3] = true
					log.Println(fmt.Sprintf("%v reached %.2f | 3", ticker, pctDiff))
					outChan <- Response{
						Type:    Price,
						Price:   newPrice,
						PctGain: float32(math.Abs(float64(pctDiff))),
						Message: dbStock.Caller,
					}
				}
			}
			if pctDiff >= 5 && pctDiff < 10 {
				if !hasPingedOverPct[5] {
					hasPingedOverPct[3] = true
					hasPingedOverPct[5] = true
					log.Println(fmt.Sprintf("%v reached %.2f | 5", ticker, pctDiff))
					outChan <- Response{
						Type:    Price,
						Price:   newPrice,
						PctGain: float32(math.Abs(float64(pctDiff))),
						Message: dbStock.Caller,
					}
				}
			}
			if pctDiff >= 10 && pctDiff < 15 {
				if !hasPingedOverPct[10] {
					hasPingedOverPct[3] = true
					hasPingedOverPct[5] = true
					hasPingedOverPct[10] = true
					log.Println(fmt.Sprintf("%v reached %.2f | 10", ticker, pctDiff))
					outChan <- Response{
						Type:    Price,
						Price:   newPrice,
						PctGain: float32(math.Abs(float64(pctDiff))),
						Message: dbStock.Caller,
					}
				}
			}
			if pctDiff >= 15 && pctDiff < 20 {
				if !hasPingedOverPct[15] {
					hasPingedOverPct[3] = true
					hasPingedOverPct[5] = true
					hasPingedOverPct[10] = true
					hasPingedOverPct[15] = true
					log.Println(fmt.Sprintf("%v reached %.2f | 15", ticker, pctDiff))
					outChan <- Response{
						Type:    Price,
						Price:   newPrice,
						PctGain: float32(math.Abs(float64(pctDiff))),
						Message: dbStock.Caller,
					}
				}
			}
			if pctDiff >= 20 && pctDiff < 25 {
				if !hasPingedOverPct[20] {
					hasPingedOverPct[3] = true
					hasPingedOverPct[5] = true
					hasPingedOverPct[10] = true
					hasPingedOverPct[15] = true
					hasPingedOverPct[20] = true
					log.Println(fmt.Sprintf("%v reached %.2f | 20", ticker, pctDiff))
					outChan <- Response{
						Type:    Price,
						Price:   newPrice,
						PctGain: float32(math.Abs(float64(pctDiff))),
						Message: dbStock.Caller,
					}
				}
			}

			if pctDiff >= 25 && pctDiff < 50 {
				if !hasPingedOverPct[25] {
					hasPingedOverPct[3] = true
					hasPingedOverPct[5] = true
					hasPingedOverPct[10] = true
					hasPingedOverPct[15] = true
					hasPingedOverPct[20] = true
					hasPingedOverPct[25] = true
					log.Println(fmt.Sprintf("%v reached %.2f | 25", ticker, pctDiff))
					outChan <- Response{
						Type:    Price,
						Price:   newPrice,
						PctGain: float32(math.Abs(float64(pctDiff))),
						Message: dbStock.Caller,
					}
				}
			}
			if pctDiff >= 50 && pctDiff < 100 {
				if !hasPingedOverPct[50] {
					hasPingedOverPct[3] = true
					hasPingedOverPct[5] = true
					hasPingedOverPct[10] = true
					hasPingedOverPct[15] = true
					hasPingedOverPct[20] = true
					hasPingedOverPct[25] = true
					hasPingedOverPct[50] = true
					log.Println(fmt.Sprintf("%v reached %.2f | 50", ticker, pctDiff))
					outChan <- Response{
						Type:    Price,
						Price:   newPrice,
						PctGain: float32(math.Abs(float64(pctDiff))),
						Message: dbStock.Caller,
					}
				}
			}
			if pctDiff >= 100 && pctDiff < 200 {
				if !hasPingedOverPct[100] {
					hasPingedOverPct[3] = true
					hasPingedOverPct[5] = true
					hasPingedOverPct[10] = true
					hasPingedOverPct[15] = true
					hasPingedOverPct[20] = true
					hasPingedOverPct[25] = true
					hasPingedOverPct[50] = true
					hasPingedOverPct[100] = true
					log.Println(fmt.Sprintf("%v reached %.2f | 100", ticker, pctDiff))
					outChan <- Response{
						Type:    Price,
						Price:   newPrice,
						PctGain: float32(math.Abs(float64(pctDiff))),
						Message: dbStock.Caller,
					}
				}
			}
			if pctDiff >= 200 {
				if !hasPingedOverPct[200] {
					hasPingedOverPct[3] = true
					hasPingedOverPct[5] = true
					hasPingedOverPct[10] = true
					hasPingedOverPct[15] = true
					hasPingedOverPct[20] = true
					hasPingedOverPct[25] = true
					hasPingedOverPct[50] = true
					hasPingedOverPct[100] = true
					hasPingedOverPct[200] = true
					log.Println(fmt.Sprintf("%v reached %.2f | 200", ticker, pctDiff))
					outChan <- Response{
						Type:    Price,
						Price:   newPrice,
						PctGain: float32(math.Abs(float64(pctDiff))),
						Message: dbStock.Caller,
					}
				}
			}
		}
	}
}
