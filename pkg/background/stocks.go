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

func (b *Background) CheckStockPriceInBG(ticker string, manageChan chan MngMsg, priceChan chan<- chan float32) {
	tick := time.NewTicker(45000 * time.Millisecond)
	log.Println("Starting BG Scan for Stock " + ticker)

	for {
		select {
		case <-tick.C:
			if !db.IsTradingHours() {
				// magic secret special message
				log.Println("Sleeping " + ticker)
				/*for _, v := range b.priceChans {
					v <- -8008.135
				}*/
				time.Sleep(utils.GetTimeToOpen())
			}
			newPrice, err := b.Fetcher.GetStock(ticker)
			if err != nil {
				log.Println(fmt.Errorf("unable to get Stock %v: %w", ticker, err))
				continue
			}
			b.pushPrice(newPrice)
		case m := <-manageChan:
			switch m.Cmd {
			case Add:
				newChan := b.addChan(m.ChanID)
				priceChan <- newChan
			case Remove:
				remaining := b.removeChan(m.ChanID)
				if remaining <= 0 {
					log.Println("Background for " + ticker + " done!")
					return
				}
			}
		}
	}
}
