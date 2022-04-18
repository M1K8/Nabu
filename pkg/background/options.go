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

	"github.com/m1k8/harpe/pkg/db"
	"github.com/m1k8/harpe/pkg/utils"
)

func (b *Background) CheckOptionPriceInBG(ticker, contractType, day, month, year string, price float32, uid string, manageChan chan ManageMsg, priceChan chan<- chan float32) {
	tick := time.NewTicker(333 * time.Millisecond)
	prettyStr := utils.NiceStr(ticker, contractType, day, month, year, price)
	log.Println("Starting BG Scan for Option " + prettyStr)

	for {
		select {
		case <-tick.C:
			if !db.IsTradingHours() {
				for _, v := range b.priceChans {
					v <- -8008.135
				}
				time.Sleep(utils.GetTimeToOpen())
			}
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
					log.Println("Background for " + prettyStr + " done!")
					return
				}
			}
		}
	}
}
