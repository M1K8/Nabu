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
	"github.com/uniplaces/carbon"
)

func (b *Background) KeepTrack(outChan chan<- bool, gid string) {
	hasPosted := false
	for {
		log.Println("Spinning tracker - " + gid)
		now, _ := carbon.NowInLocation("America/Detroit")
		if !db.IsTradingHours() && !hasPosted {
			log.Println("AH - Tracker being submitted - " + gid)
			if now.Hour() == 16 && !hasPosted { // if it isnt end of trading day - say bot is started outside of trading hours
				outChan <- true
				hasPosted = true
			} else {
				log.Println(fmt.Sprintf("Tracker sleeping until market open - %v - ", utils.GetTimeToOpen()) + gid)
				hasPosted = false
				tts := utils.GetTimeToOpen()

				if tts <= 0 { //idk
					log.Println("tts less than 0")
					tts = 12 * time.Hour
				}
				time.Sleep(tts)
			}
		} else {
			sleepTime := now.StartOfDay().Add(16*time.Hour + 1*time.Minute).Sub(now.Time)
			log.Println(fmt.Sprintf("Tracker sleeping until market close - %v - ", sleepTime) + gid)
			hasPosted = false

			if sleepTime <= 0 { //idk lol
				log.Println("tts less than 0")
				sleepTime = time.Hour
			}
			time.Sleep(sleepTime)
		}
	}
}
