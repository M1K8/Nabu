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

func (b *Background) CheckShortPriceInBG(outChan chan<- Response, ticker, author, expiry, guildID string, exit chan bool) {
	log.Println("Starting BG Scan for Short " + ticker)
	tick := time.NewTicker(45 * time.Second) // slow because api is slow
	var expiryDate time.Time
	var err error = nil
	hasAlertedSPT := false
	poiHit := false
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

	defer (func() {
		log.Println("closing channel for short " + ticker)
		final, _ := b.Fetcher.GetStock(ticker)
		outChan <- Response{
			Type:  Exit,
			Price: final,
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
	dbShort, err := b.Repo.GetShort(ticker)
	if err != nil {
		log.Println(fmt.Errorf("unable to get short from db %v: %w", ticker, err))
		return
	}
	poiHit = dbShort.ShortPOIHit
	lowest := dbShort.ShortLowest

	if !expiryDate.IsZero() && time.Now().After(expiryDate) {
		final, _ := b.Fetcher.GetStock(ticker)
		outChan <- Response{
			Type:  Expired,
			Price: final,
		}
		return
	}

	for {
		select {
		case <-exit:
			return
		case <-tick.C:
			if !db.IsTradingHours() {
				if dbShort.ChannelType == utils.DAY || !expiryDate.IsZero() && time.Now().After(expiryDate) {
					outChan <- Response{
						Type: Expired,
					}
					return
				}
				log.Println("Short alert for " + ticker + " is sleeping.")
				log.Printf("now: %v, wait: %v \n", time.Now(), utils.GetTimeToOpen())
				time.Sleep(utils.GetTimeToOpen())
			}
			newPrice, err := b.Fetcher.GetStock(ticker)
			if err != nil {
				log.Println(fmt.Errorf("error for ticker %v: %w", ticker, err))
				continue
			}

			priceDiff := newPrice - dbShort.ShortStarting
			if priceDiff == 0 {
				continue
			}

			if newPrice < lowest {
				lowest = newPrice
				outChan <- Response{
					Type:  New_High,
					Price: newPrice,
				}
			}

			if dbShort.ShortSPt > 0 && !hasAlertedSPT {
				if newPrice <= dbShort.ShortSPt {
					outChan <- Response{
						Type:  PT1,
						Price: newPrice,
					}
				}
				hasAlertedSPT = true
			}

			if dbShort.ShortEPt > 0 {
				if newPrice <= dbShort.ShortEPt {
					outChan <- Response{
						Type:  PT2,
						Price: newPrice,
					}
					return
				}
			}

			if dbShort.ShortStop != 0 && newPrice >= dbShort.ShortStop {
				outChan <- Response{
					Type:  SL,
					Price: newPrice,
				}
				return
			}

			pctDiff := (priceDiff / dbShort.ShortStarting) * 100

			if dbShort.ShortPoI > 0.0 && !poiHit {

				if math.Abs(float64(pctDiff)) <= 0.5 {
					poiHit = true
					outChan <- Response{
						Type:  POI,
						Price: newPrice,
					}
				}
			}

			if pctDiff <= -3 && pctDiff > -5 {
				if !hasPingedOverPct[3] {
					hasPingedOverPct[3] = true
					log.Println(fmt.Sprintf("%v reached %.2f | 3", ticker, pctDiff))
					outChan <- Response{
						Type:    Price,
						Price:   newPrice,
						PctGain: float32(math.Abs(float64(pctDiff))),
					}
				}
			}
			if pctDiff <= -5 && pctDiff > -10 {
				if !hasPingedOverPct[5] {
					hasPingedOverPct[3] = true
					hasPingedOverPct[5] = true
					log.Println(fmt.Sprintf("%v reached %.2f | 5", ticker, pctDiff))
					outChan <- Response{
						Type:    Price,
						Price:   newPrice,
						PctGain: float32(math.Abs(float64(pctDiff))),
					}
				}
			}
			if pctDiff <= -10 && pctDiff > -15 {
				if !hasPingedOverPct[10] {
					hasPingedOverPct[3] = true
					hasPingedOverPct[5] = true
					hasPingedOverPct[10] = true
					log.Println(fmt.Sprintf("%v reached %.2f | 10", ticker, pctDiff))
					outChan <- Response{
						Type:    Price,
						Price:   newPrice,
						PctGain: float32(math.Abs(float64(pctDiff))),
					}
				}
			}
			if pctDiff <= -15 && pctDiff > -20 {
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
					}
				}
			}
			if pctDiff <= -20 && pctDiff > -25 {
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
					}
				}
			}

			if pctDiff <= -25 && pctDiff > -50 {
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
					}
				}
			}
			if pctDiff <= -50 && pctDiff > -100 {
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
					}
				}
			}
			if pctDiff <= -100 && pctDiff > -200 {
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
					}
				}
			}
			if pctDiff <= -200 {
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
					}
				}
			}
		}
	}
}
