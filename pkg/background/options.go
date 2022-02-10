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
	"math/rand"
	"strings"
	"time"

	"github.com/m1k8/harpe/pkg/db"
	"github.com/m1k8/harpe/pkg/utils"
	"github.com/uniplaces/carbon"
)

func (b *Background) CheckOptionsPriceInBG(outChan chan<- Response, guildID, author, ticker, contractType, day, month, year string, price float32, exit chan bool) {
	tick := time.NewTicker(750 * time.Millisecond)
	var last float32 = 0.0
	poiHit := false

	log.Println("Starting BG Scan for Options " + utils.NiceStr(ticker, contractType, day, month, year, price))
	hasPingedOverPct := map[float32]bool{ //: 5, 10, 20, 25, 50, 100, 200, 500, 1000
		5:    false,
		10:   false,
		20:   false,
		25:   false,
		50:   false,
		100:  false,
		200:  false,
		500:  false,
		1000: false,
	}
	prettyStr := utils.NiceStr(ticker, contractType, day, month, year, price)
	oID := utils.GetCode(ticker, contractType, day, month, year, price)

	defer (func() {
		log.Println("closing channel for option " + prettyStr)
		final, _, _ := b.Fetcher.GetOption(ticker, contractType, day, month, year, price, last)
		outChan <- Response{
			Type:  Exit,
			Price: final,
		}
		//close(exit)
		//close(outChan)
	})()

	expiryDate, err := time.Parse(time.RFC3339, fmt.Sprintf("%v-%v-%vT20:59:59Z", year, month, day))
	if err != nil {
		log.Println(err.Error())
		outChan <- Response{
			Type:    Error,
			Message: err.Error(),
		}
		return
	}
	now := time.Now()

	optionDb, err := b.Repo.GetOption(oID)
	if err != nil {
		log.Println(fmt.Errorf("unable to get option from db %v: %w", prettyStr, err))
		outChan <- Response{
			Type:    Error,
			Message: err.Error(),
		}
		return
	}
	poiHit = optionDb.OptionUnderlyingPOIHit
	highest := optionDb.OptionHighest

	// remove alert if contract has expired
	if !expiryDate.IsZero() && now.After(expiryDate) {
		final, _, _ := b.Fetcher.GetOption(ticker, contractType, day, month, year, price, last)
		outChan <- Response{
			Type:  Expired,
			Price: final,
		}
		return
	}

	callTimeCarbon := carbon.NewCarbon(optionDb.OptionCallTime)
	nowCarbon := carbon.NewCarbon(now)
	nowCarbon.SetTimeZone("America/Detroit")
	expiryCarbon := carbon.NewCarbon(expiryDate)
	expiryCarbon.SetTimeZone("America/Detroit")
	// remove if weekly
	// should really function but, but too many variables to pass around D:
	if callTimeCarbon.DiffInWeeks(nowCarbon, true) == 0 &&
		callTimeCarbon.DiffInDays(nowCarbon, false) > 2 &&
		callTimeCarbon.DiffInWeeks(expiryCarbon, false) == 0 {
		outChan <- Response{
			Type:    Expired,
			Message: " weird exp rule",
		}
		log.Println("Removed options ticker " + prettyStr)
		return
	}

	if optionDb.OptionStarting <= highest {
		outChan <- Response{
			Type:    New_High,
			Price:   highest,
			PctGain: 0,
			Message: "",
		}
	} else {
		outChan <- Response{
			Type:    New_High,
			Price:   optionDb.OptionStarting,
			PctGain: 0,
			Message: "",
		}
	}

	for {
		now = time.Now()
		select {
		case <-exit:
			return
		case <-tick.C:
			if !db.IsTradingHours() {
				if optionDb.ChannelType == utils.DAY || !expiryDate.IsZero() && now.After(expiryDate) {
					outChan <- Response{
						Type:    Expired,
						Message: "closing channel for option due to day trade",
					}
					log.Println("Removed options ticker " + prettyStr)
					return
				}
				nowCarbon := carbon.NewCarbon(now)
				nowCarbon.SetTimeZone("America/Detroit")
				expiryCarbon := carbon.NewCarbon(expiryDate)
				expiryCarbon.SetTimeZone("America/Detroit")
				//repeated code ik, too lazy to method out
				if callTimeCarbon.DiffInWeeks(nowCarbon, true) == 0 &&
					callTimeCarbon.DiffInDays(nowCarbon, false) > 2 &&
					callTimeCarbon.DiffInWeeks(expiryCarbon, false) == 0 {

					log.Println("Option " + prettyStr + " hit weekly expire. Called: " + callTimeCarbon.DateTimeString())
					outChan <- Response{
						Type:    Expired,
						Message: "weird exp rule",
					}
					log.Println("Removed options ticker " + prettyStr)
					return
				}

				log.Println("Options alert for " + prettyStr + " is sleeping.")
				log.Printf("wait: %v \n", utils.GetTimeToOpen())
				time.Sleep(utils.GetTimeToOpen())
			}
			newPrice, _, err := b.Fetcher.GetOption(ticker, contractType, day, month, year, price, last)
			if newPrice == 0 {
				newPrice = last
			} else {
				last = newPrice
			}
			if err != nil {
				if strings.Contains(err.Error(), "Price is 0") {
					s1 := rand.NewSource(time.Now().UnixNano())
					r := rand.New(s1)
					log.Println("option " + prettyStr + " is zeroing out :(")
					time.Sleep(120*time.Second + (time.Duration(r.Intn(30)) * time.Second))
					continue
				}
				log.Println(fmt.Errorf("unable to get option for %v: %w", prettyStr, err))
				continue
			}

			if newPrice == -1 {
				log.Println("Rate limited - try again in a few minutes")
				continue
			}

			if newPrice > highest {
				highest = newPrice
				outChan <- Response{
					Type:  New_High,
					Price: newPrice,
				}
			}

			if optionDb.OptionUnderlyingPoI > 0 || optionDb.OptionUnderlyingStop > 0 {
				underlying, _, err := b.Fetcher.GetOptionAdvanced(ticker, contractType, day, month, year, price)

				if err != nil {
					log.Println(err)
				} else {
					if time.Since(time.Unix(underlying.Results.UnderlyingAsset.LastUpdated, 0)) <= (time.Second * 2) {
						// hopefully prevents shenanigans

						if optionDb.OptionUnderlyingPoI > 0 && !poiHit {
							underPctDiff := math.Abs((underlying.Results.UnderlyingAsset.Price - float64(optionDb.OptionUnderlyingStarting)) / float64(optionDb.OptionUnderlyingStarting) * 100)

							if underPctDiff <= 1 {
								log.Println("POI Hit for " + ticker)
								outChan <- Response{
									Type:  POI,
									Price: newPrice,
								}
								poiHit = true
							}
						}

						priceU := float32(underlying.Results.UnderlyingAsset.Price)
						if optionDb.OptionUnderlyingStop > 0 && priceU <= optionDb.OptionUnderlyingStop {
							log.Println(fmt.Sprintf("Stop Hit for %v - $%.2f", ticker, priceU))
							outChan <- Response{
								Type:  SL,
								Price: newPrice,
							}
							return
						}
					}
				}
			}
			priceDiff := newPrice - optionDb.OptionStarting
			if priceDiff == 0 {
				continue
			}

			pctDiff := (priceDiff / optionDb.OptionStarting) * 100
			if pctDiff >= 5 && pctDiff < 10 {
				if !hasPingedOverPct[5] {
					hasPingedOverPct[5] = true
					log.Println(fmt.Sprintf("%v reached %.2f | 5", ticker, pctDiff))
					outChan <- Response{
						Type:    Price,
						Price:   newPrice,
						PctGain: pctDiff,
					}

				}
			}
			if pctDiff >= 10 && pctDiff < 20 {
				if !hasPingedOverPct[10] {
					hasPingedOverPct[5] = true
					hasPingedOverPct[10] = true
					log.Println(fmt.Sprintf("%v reached %.2f | 10", ticker, pctDiff))
					outChan <- Response{
						Type:    Price,
						Price:   newPrice,
						PctGain: pctDiff,
					}
				}
			}
			if pctDiff >= 20 && pctDiff < 25 {
				if !hasPingedOverPct[20] {
					hasPingedOverPct[5] = true
					hasPingedOverPct[10] = true
					hasPingedOverPct[20] = true
					log.Println(fmt.Sprintf("%v reached %.2f | 20", ticker, pctDiff))
					outChan <- Response{
						Type:    Price,
						Price:   newPrice,
						PctGain: pctDiff,
					}
				}
			}
			if pctDiff >= 25 && pctDiff < 50 {
				if !hasPingedOverPct[25] {
					hasPingedOverPct[5] = true
					hasPingedOverPct[10] = true
					hasPingedOverPct[20] = true
					hasPingedOverPct[25] = true
					log.Println(fmt.Sprintf("%v reached %.2f | 25", ticker, pctDiff))
					outChan <- Response{
						Type:    Price,
						Price:   newPrice,
						PctGain: pctDiff,
					}
				}
			}
			if pctDiff >= 50 && pctDiff < 100 {
				if !hasPingedOverPct[50] {
					hasPingedOverPct[5] = true
					hasPingedOverPct[10] = true
					hasPingedOverPct[20] = true
					hasPingedOverPct[25] = true
					hasPingedOverPct[50] = true
					log.Println(fmt.Sprintf("%v reached %.2f | 50", ticker, pctDiff))
					outChan <- Response{
						Type:    Price,
						Price:   newPrice,
						PctGain: pctDiff,
					}
				}
			}
			if pctDiff >= 100 && pctDiff < 200 {
				if !hasPingedOverPct[100] {
					hasPingedOverPct[5] = true
					hasPingedOverPct[10] = true
					hasPingedOverPct[20] = true
					hasPingedOverPct[25] = true
					hasPingedOverPct[50] = true
					hasPingedOverPct[100] = true
					log.Println(fmt.Sprintf("%v reached %.2f | 100", ticker, pctDiff))
					outChan <- Response{
						Type:    Price,
						Price:   newPrice,
						PctGain: pctDiff,
					}
				}
			}
			if pctDiff >= 200 && pctDiff < 500 {
				if !hasPingedOverPct[200] {
					hasPingedOverPct[5] = true
					hasPingedOverPct[10] = true
					hasPingedOverPct[20] = true
					hasPingedOverPct[25] = true
					hasPingedOverPct[50] = true
					hasPingedOverPct[100] = true
					hasPingedOverPct[200] = true
					log.Println(fmt.Sprintf("%v reached %.2f | 200", ticker, pctDiff))
					outChan <- Response{
						Type:    Price,
						Price:   newPrice,
						PctGain: pctDiff,
					}
				}
			}
			if pctDiff >= 500 && pctDiff < 1000 {
				if !hasPingedOverPct[500] {
					hasPingedOverPct[5] = true
					hasPingedOverPct[10] = true
					hasPingedOverPct[20] = true
					hasPingedOverPct[25] = true
					hasPingedOverPct[50] = true
					hasPingedOverPct[100] = true
					hasPingedOverPct[200] = true
					hasPingedOverPct[500] = true
					log.Println(fmt.Sprintf("%v reached %.2f | 500", ticker, pctDiff))
					outChan <- Response{
						Type:    Price,
						Price:   newPrice,
						PctGain: pctDiff,
					}
				}
			}
			if pctDiff >= 1000 {
				if !hasPingedOverPct[1000] {
					hasPingedOverPct[5] = true
					hasPingedOverPct[10] = true
					hasPingedOverPct[20] = true
					hasPingedOverPct[25] = true
					hasPingedOverPct[50] = true
					hasPingedOverPct[100] = true
					hasPingedOverPct[200] = true
					hasPingedOverPct[500] = true
					hasPingedOverPct[1000] = true
					log.Println(fmt.Sprintf("%v reached %.2f | 1000", ticker, pctDiff))
					outChan <- Response{
						Type:    Price,
						Price:   newPrice,
						PctGain: pctDiff,
					}
				}
			}
		}
	}
}
