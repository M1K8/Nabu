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
	"time"

	"github.com/m1k8/harpe/pkg/utils"
)

func (b *Background) CheckOptionPriceInBG(ticker, contractType, day, month, year string, price float32, uid string, manageChan chan ManageMsg, priceChan chan<- chan float32, exitChan chan<- bool) {
	tick := time.NewTicker(333 * time.Millisecond)
	prettyStr := utils.NiceStr(ticker, contractType, day, month, year, price)
	log.Println("Starting BG Scan for Option " + prettyStr)

	for {
		select {
		case <-tick.C:
			newPrice, _, err := b.Fetcher.GetOption(ticker, contractType, day, month, year, price, 0)
			if err != nil {
				log.Println(fmt.Errorf("unable to get Option %v: %w", prettyStr, err))
				continue
			}
			b.pushPrice(newPrice)
		case m := <-manageChan:
			switch m {
			case Add:
				newChan := b.addChan(uid)
				priceChan <- newChan
			case Remove:
				remaining := b.removeChan(uid)
				if remaining <= 0 {
					exitChan <- true
					return
				}
			}
		}
	}
}

/*

func (b *Background) CheckOptionsPriceInBG(outChan chan<- Response, guildID, author, ticker, contractType, day, month, year string, price float32, exit chan bool, inChan <-chan Response) {
	tick := time.NewTicker(333 * time.Millisecond)
	var last float32 = 0.0

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

	optionDb, err := b.Repo.GetOption(oID)
	if err != nil {
		log.Println(fmt.Errorf("unable to get option from db %v: %w", prettyStr, err))
		outChan <- Response{
			Type:    Error,
			Message: err.Error(),
		}
		return
	}
	poiHit := optionDb.OptionUnderlyingPOIHit
	highest := optionDb.OptionHighest
	trailingPct := optionDb.OptionTrailingStop / 100

	defer (func() {
		log.Println("closing channel for option " + prettyStr)
		outChan <- Response{
			Type:  Exit,
			Price: highest,
		}
	})()

	expiryDate, err := time.Parse(time.RFC3339, fmt.Sprintf("%v-%v-%vT20:59:58Z", year, month, day))
	if err != nil {
		log.Println(err.Error())
		outChan <- Response{
			Type:    Error,
			Message: err.Error(),
		}
		return
	}
	now := time.Now()

	// remove alert if contract has expired
	if !expiryDate.IsZero() && now.After(expiryDate) {
		final, _, _ := b.Fetcher.GetOption(ticker, contractType, day, month, year, price, last)
		outChan <- Response{
			Type:    Expired,
			Price:   final,
			Message: fmt.Sprintf("%v", expiryDate),
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
		log.Println("Option " + prettyStr + " hit weekly expire. Called: " + callTimeCarbon.DateTimeString())
		outChan <- Response{
			Type:    Expired,
			Message: optionDb.Caller,
		}
		log.Println("Removed options ticker due to weird rule" + prettyStr)
		return
	}

	if optionDb.OptionStarting <= highest {
		outChan <- Response{
			Type:    New_High,
			Price:   highest,
			PctGain: 0,
			Message: optionDb.Caller,
		}
	} else {
		outChan <- Response{
			Type:    New_High,
			Price:   optionDb.OptionStarting,
			PctGain: 0,
			Message: optionDb.Caller,
		}
	}

	for {
		now = time.Now()
		select {
		case <-exit:
			return
		case m := <-inChan:
			switch m.Type {
			case New_Avg:
				log.Println("Getting new Avg for option " + oID)
				optionDb, err = b.Repo.GetOption(oID)
				if err != nil {
					log.Println(fmt.Errorf("unable to get Option from db %v: %w", oID, err))
					return
				}
				highest = optionDb.OptionStarting // as we're basically resetting the alert, reset the new highest
				hasPingedOverPct[5] = false
				hasPingedOverPct[10] = false
				hasPingedOverPct[20] = false
				hasPingedOverPct[25] = false
				hasPingedOverPct[50] = false
				hasPingedOverPct[100] = false
				hasPingedOverPct[200] = false
				hasPingedOverPct[500] = false
				hasPingedOverPct[1000] = false
				hasPingedOverPct[5000] = false
				outChan <- Response{
					Type:    New_Avg,
					Price:   highest,
					Message: optionDb.Caller,
				}
			}
		case <-tick.C:
			newPrice, _, err := b.Fetcher.GetOption(ticker, contractType, day, month, year, price, last)
			if !db.IsTradingHours() {
				if !expiryDate.IsZero() && now.After(expiryDate) || optionDb.ChannelType == utils.DAY {
					outChan <- Response{
						Type:    Expired,
						Message: optionDb.Caller,
					}
					log.Println("Removed options ticker " + prettyStr)
					return
				}
				log.Println(prettyStr + " expiry - " + day + "/" + month)
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
						Message: optionDb.Caller,
					}
					log.Println("Removed options ticker due to weird rule" + prettyStr)
					return
				}

				log.Println("Options alert for " + prettyStr + " is sleeping.")
				log.Printf("wait: %v \n", utils.GetTimeToOpen())
				time.Sleep(utils.GetTimeToOpen())
			}
			if newPrice == 0 {
				newPrice = last
				log.Println("Price is zero, waiting 2 mins for " + prettyStr)
				time.Sleep(2 * time.Minute)
			} else if newPrice == -1 {
				log.Println("Rate limited - try again in a few minutes")
				continue
			} else {
				last = newPrice
			}

			if err != nil {
				if strings.Contains(err.Error(), "Price is 0") {
					log.Println("option " + prettyStr + " is zeroing out :(")
					time.Sleep(120 * time.Second)
					continue
				}
				log.Println(fmt.Errorf("unable to get option for %v: %w", prettyStr, err))
				continue
			}

			go func() {
				if newPrice > highest {
					highest = newPrice
					outChan <- Response{
						Type:    New_High,
						Price:   newPrice,
						Message: optionDb.Caller,
					}
				}
			}()
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
						Message: optionDb.Caller,
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
						Message: optionDb.Caller,
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
						Message: optionDb.Caller,
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
						Message: optionDb.Caller,
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
						Message: optionDb.Caller,
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
						Message: optionDb.Caller,
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
						Message: optionDb.Caller,
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
						Message: optionDb.Caller,
					}
				}
			}
			if pctDiff >= 1000 && pctDiff < 5000 {
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
						Message: optionDb.Caller,
					}
				}
			}
			if pctDiff >= 5000 {
				if !hasPingedOverPct[5000] {
					hasPingedOverPct[5] = true
					hasPingedOverPct[10] = true
					hasPingedOverPct[20] = true
					hasPingedOverPct[25] = true
					hasPingedOverPct[50] = true
					hasPingedOverPct[100] = true
					hasPingedOverPct[200] = true
					hasPingedOverPct[500] = true
					hasPingedOverPct[1000] = true
					hasPingedOverPct[5000] = true
					log.Println(fmt.Sprintf("%v reached %.2f | 5000", ticker, pctDiff))
					outChan <- Response{
						Type:    Price,
						Price:   newPrice,
						PctGain: pctDiff,
						Message: optionDb.Caller,
					}
				}
			}

			if trailingPct != 0 && newPrice < (1-trailingPct)*highest {
				outChan <- Response{
					Type:    TSL,
					Price:   newPrice,
					Message: optionDb.Caller,
				}
				return
			}

			if optionDb.OptionUnderlyingPoI > 0 || optionDb.OptionUnderlyingStop > 0 {
				underlying, _, err := b.Fetcher.GetOptionAdvanced(ticker, contractType, day, month, year, price)

				if err != nil {
					log.Println(prettyStr + " poi / stop " + err.Error())
					log.Println(optionDb.OptionUnderlyingStop)
					log.Println(optionDb.OptionUnderlyingPoI)
					time.Sleep(5 * time.Second)
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

						if strings.ToLower(contractType) == "p" {
							if optionDb.OptionUnderlyingStop > 0 && priceU >= optionDb.OptionUnderlyingStop {
								log.Println(fmt.Sprintf("Stop Hit for %v - $%.2f", ticker, priceU))
								outChan <- Response{
									Type:  SL,
									Price: newPrice,
								}
								return
							}
						} else if strings.ToLower(contractType) == "c" {
							if optionDb.OptionUnderlyingStop > 0 && priceU <= optionDb.OptionUnderlyingStop {
								log.Println(fmt.Sprintf("Stop Hit for %v - $%.2f", ticker, priceU))
								outChan <- Response{
									Type:  SL,
									Price: newPrice,
								}
								return
							}
						} else {
							log.Println(contractType)
						}

					}
				}
			}
		}
	}
}
*/
