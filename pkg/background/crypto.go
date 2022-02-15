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
package background

import (
	"fmt"
	"log"
	"math"
	"strconv"
	"time"
)

func (b *Background) CheckCryptoPriceInBG(outChan chan<- Response, ticker, expiry, guildID, author string, exit chan bool) {
	tick := time.NewTicker(10000 * time.Millisecond)
	log.Println("Starting BG Scan for Crypto " + ticker)
	var expiryDate time.Time
	var err error = nil
	hasAlertedSPT := false
	poiHit := false

	hasPingedOverPct := map[float32]bool{ //500, 1000, 2000
		10:  false,
		30:  false,
		50:  false,
		100: false,
		200: false,
	}
	if expiry != "" {
		expiryNum, _ := strconv.ParseInt(expiry, 10, 32)
		if expiryNum != 0 {
			expiryDate = time.Now().AddDate(0, 0, int(expiryNum))
		}
	}

	defer (func() {
		log.Println("closing channel for crypto " + ticker)
		final, _ := b.Fetcher.GetCrypto(ticker, false)
		outChan <- Response{
			Type:  Exit,
			Price: final,
		}
		//close(exit)
		//close(outChan)
	})()
	coinDb, err := b.Repo.GetCrypto(ticker)
	if err != nil {
		log.Println(fmt.Errorf("unable to get crypto from db %v: %w", ticker, err))
		return
	}
	poiHit = coinDb.CryptoPOIHit
	highest := coinDb.CryptoHighest

	if !expiryDate.IsZero() && time.Now().After(expiryDate) {
		outChan <- Response{
			Type:    Expired,
			Price:   highest,
			PctGain: 0,
			Message: coinDb.Caller,
		}
		return
	}

	if coinDb.CryptoStarting <= highest {
		outChan <- Response{
			Type:    New_High,
			Price:   highest,
			PctGain: 0,
			Message: coinDb.Caller,
		}
	} else {
		outChan <- Response{
			Type:    New_High,
			Price:   coinDb.CryptoStarting,
			PctGain: 0,
			Message: coinDb.Caller,
		}
	}

	for {
		select {
		case <-exit:
			return
		case <-tick.C:
			newPrice, err := b.Fetcher.GetCrypto(ticker, false)
			if err != nil {
				log.Println(fmt.Errorf("unable to get crypto %v: %w", ticker, err))
				continue
			}

			priceDiff := newPrice - coinDb.CryptoStarting
			if priceDiff == 0 {
				continue
			}

			if newPrice > highest {
				highest = newPrice
				outChan <- Response{
					Type:    New_High,
					Price:   newPrice,
					PctGain: 0,
					Message: coinDb.Caller,
				}
			}

			if coinDb.CryptoSPt > 0 && !hasAlertedSPT {
				if newPrice >= coinDb.CryptoSPt {
					outChan <- Response{
						Type:    PT1,
						Price:   newPrice,
						PctGain: 0,
						Message: coinDb.Caller,
					}
				}
				hasAlertedSPT = true
			}

			if coinDb.CryptoEPt > 0 {
				if newPrice >= coinDb.CryptoEPt {
					outChan <- Response{
						Type:    PT2,
						Price:   newPrice,
						PctGain: 0,
						Message: coinDb.Caller,
					}
					return
				}
			}

			if coinDb.CryptoStop != 0 && newPrice <= coinDb.CryptoStop {
				outChan <- Response{
					Type:    SL,
					Price:   newPrice,
					PctGain: 0,
					Message: coinDb.Caller,
				}
				return
			}

			pctDiff := (priceDiff / coinDb.CryptoStarting) * 100

			if coinDb.CryptoPoI > 0.0 && !poiHit {
				if math.Abs(float64(pctDiff)) <= 0.5 {
					outChan <- Response{
						Type:    POI,
						Price:   newPrice,
						PctGain: pctDiff,
						Message: coinDb.Caller,
					}
					poiHit = true
				} else {
					continue
				}
			}

			if pctDiff >= 10 && pctDiff < 30 {
				if !hasPingedOverPct[10] {
					hasPingedOverPct[10] = true
					outChan <- Response{
						Type:    Price,
						Price:   newPrice,
						PctGain: pctDiff,
						Message: coinDb.Caller,
					}
					log.Println(fmt.Sprintf("%v reached %.2f | 10", ticker, pctDiff))
				}
			}
			if pctDiff >= 30 && pctDiff < 50 {
				if !hasPingedOverPct[30] {
					hasPingedOverPct[10] = true
					hasPingedOverPct[30] = true
					outChan <- Response{
						Type:    Price,
						Price:   newPrice,
						PctGain: pctDiff,
						Message: coinDb.Caller,
					}
					log.Println(fmt.Sprintf("%v reached %.2f | 30", ticker, pctDiff))
				}
			}
			if pctDiff >= 50 && pctDiff < 100 {
				if !hasPingedOverPct[50] {
					hasPingedOverPct[10] = true
					hasPingedOverPct[30] = true
					hasPingedOverPct[50] = true
					outChan <- Response{
						Type:    Price,
						Price:   newPrice,
						PctGain: pctDiff,
						Message: coinDb.Caller,
					}
					log.Println(fmt.Sprintf("%v reached %.2f | 50", ticker, pctDiff))
				}
			}
			if pctDiff >= 100 && pctDiff < 200 {
				if !hasPingedOverPct[100] {
					hasPingedOverPct[10] = true
					hasPingedOverPct[30] = true
					hasPingedOverPct[50] = true
					hasPingedOverPct[100] = true
					outChan <- Response{
						Type:    Price,
						Price:   newPrice,
						PctGain: pctDiff,
						Message: coinDb.Caller,
					}
					log.Println(fmt.Sprintf("%v reached %.2f | 100", ticker, pctDiff))
				}
			}
			if pctDiff >= 200 && pctDiff < 500 {
				if !hasPingedOverPct[200] {
					hasPingedOverPct[10] = true
					hasPingedOverPct[30] = true
					hasPingedOverPct[50] = true
					hasPingedOverPct[100] = true
					hasPingedOverPct[200] = true
					outChan <- Response{
						Type:    Price,
						Price:   newPrice,
						PctGain: pctDiff,
						Message: coinDb.Caller,
					}
					log.Println(fmt.Sprintf("%v reached %.2f | 200", ticker, pctDiff))
				}
			}
			if pctDiff >= 500 && pctDiff < 1000 {
				if !hasPingedOverPct[500] {
					hasPingedOverPct[10] = true
					hasPingedOverPct[30] = true
					hasPingedOverPct[50] = true
					hasPingedOverPct[100] = true
					hasPingedOverPct[200] = true
					hasPingedOverPct[500] = true
					outChan <- Response{
						Type:    Price,
						Price:   newPrice,
						PctGain: pctDiff,
						Message: coinDb.Caller,
					}
					log.Println(fmt.Sprintf("%v reached %.2f | 500", ticker, pctDiff))
				}
			}
			if pctDiff >= 1000 && pctDiff < 2000 {
				if !hasPingedOverPct[1000] {
					hasPingedOverPct[10] = true
					hasPingedOverPct[30] = true
					hasPingedOverPct[50] = true
					hasPingedOverPct[100] = true
					hasPingedOverPct[200] = true
					hasPingedOverPct[500] = true
					hasPingedOverPct[1000] = true
					outChan <- Response{
						Type:    Price,
						Price:   newPrice,
						PctGain: pctDiff,
						Message: coinDb.Caller,
					}
					log.Println(fmt.Sprintf("%v reached %.2f | 1000", ticker, pctDiff))
				}
			}
			if pctDiff >= 2000 {
				if !hasPingedOverPct[2000] {
					hasPingedOverPct[10] = true
					hasPingedOverPct[30] = true
					hasPingedOverPct[50] = true
					hasPingedOverPct[100] = true
					hasPingedOverPct[200] = true
					hasPingedOverPct[500] = true
					hasPingedOverPct[1000] = true
					hasPingedOverPct[2000] = true
					outChan <- Response{
						Type:    Price,
						Price:   newPrice,
						PctGain: pctDiff,
						Message: coinDb.Caller,
					}
					log.Println(fmt.Sprintf("%v reached %.2f | 2000", ticker, pctDiff))
				}
			}
		}
	}
}
