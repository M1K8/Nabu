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
)

func (b *Background) CheckCryptoPriceInBG(ticker string, manageChan chan ManageMsg, priceChan chan<- float32, exitChan chan<- bool) {
	tick := time.NewTicker(10000 * time.Millisecond)
	log.Println("Starting BG Scan for Crypto " + ticker)

	for {
		select {
		case <-tick.C:
			newPrice, err := b.Fetcher.GetCrypto(ticker, false)
			if err != nil {
				log.Println(fmt.Errorf("unable to get crypto %v: %w", ticker, err))
				continue
			}
			priceChan <- newPrice
		case m := <-manageChan:
			switch m {
			case Add:
				b.Add()
			case Remove:
				remaining := b.Remove()
				if remaining <= 0 {
					exitChan <- true
					return
				}
			}
		}
	}
}
