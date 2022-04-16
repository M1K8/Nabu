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
	harpe "github.com/m1k8/harpe/pkg/db"
	"github.com/m1k8/nabu/pkg/fetcher"
)

type Background struct {
	Fetcher    fetcher.Fetcher
	Repo       Repo
	References int
	priceChans map[string]chan float32
}

type ManageMsg int

type ResponseType int

const (
	Add ManageMsg = iota
	Remove
	Exit
)

func NewBG(guildID string, repo Repo) *Background {
	fetch := fetcher.NewFetcher()

	return &Background{
		Fetcher:    fetch,
		Repo:       repo,
		priceChans: make(map[string]chan float32),
	}
}

type Repo interface {
	RmAll() error
	GetAll() ([]*harpe.Stock, []*harpe.Short, []*harpe.Crypto, []*harpe.Option, error)
	GetExitChan(string) chan bool
	SetAndReturnNewExitChan(string, chan bool) chan bool
	RefreshFromDB() ([]*harpe.Stock, []*harpe.Short, []*harpe.Crypto, []*harpe.Option, error)

	GetOption(string) (*harpe.Option, error)
	CreateOption(string, string, string, int, string, string, string, string, string, float32, float32, float32, float32, float32, float32, float32) (chan bool, string, bool, error)
	RemoveOption(string, string, string, string, string, float32) error

	CreateShort(string, string, string, int, float32, float32, float32, float32, float32, int64, float32) (chan bool, bool, error)
	RemoveShort(string) error
	GetShort(string) (*harpe.Short, error)

	CreateStock(string, string, string, int, float32, float32, float32, float32, float32, int64, float32) (chan bool, bool, error)
	RemoveStock(string) error
	GetStock(string) (*harpe.Stock, error)

	CreateCrypto(string, string, string, float32, float32, float32, float32, float32, int, float32) (chan bool, bool, error)
	RemoveCrypto(string) error
	GetCrypto(string) (*harpe.Crypto, error)
}

func (b *Background) addChan(uid string) chan float32 {
	b.References += 1
	newChan := make(chan float32)
	b.priceChans[uid] = newChan
	return newChan
}

func (b *Background) removeChan(uid string) int {
	close(b.priceChans[uid])
	delete(b.priceChans, uid)
	b.References -= 1
	return b.References
}

func (b *Background) pushPrice(p float32) {
	for _, v := range b.priceChans {
		v <- p
	}
}
