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

func (b *Background) CheckCryptoPriceInBG(ticker, uid string, manageChan <-chan ManageMsg, priceChan chan<- chan float32) {
	tick := time.NewTicker(10000 * time.Millisecond)
	log.Println("Starting BG Scan for Crypto " + ticker)

	// every .Add, add a new channel and pass it back over the mgmt channel

	for {
		select {
		case <-tick.C:
			newPrice, err := b.Fetcher.GetCrypto(ticker, false)
			if err != nil {
				log.Println(fmt.Errorf("unable to get crypto %v: %w", ticker, err))
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
					return
				}
			}
		}
	}
}
