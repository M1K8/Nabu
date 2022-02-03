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
	harpe "github.com/m1k8/harpe/pkg/db"
	fetcher "github.com/m1k8/nabu/pkg/price_fetcher"
)

type Background struct {
	Fetcher fetcher.Fetcher
	Repo    Repo
	GuildID string
}

type Response struct {
	Type    ResponseType // 0 price, 1 expired, 2 PT1, 3 PT2, 4 SL, 5 POI, 6 New Hig, 7 EoD 9 other
	Price   float32
	PctGain float32
	Message string
}

type ResponseType int

const (
	Price ResponseType = iota
	Expired
	PT1
	PT2
	SL
	POI
	New_High
	EoD
	Error
)

func NewBG(guildID string, repo Repo) *Background {
	fetch := fetcher.NewFetcher()

	return &Background{
		Fetcher: fetch,
		Repo:    repo,
		GuildID: guildID,
	}
}

type Repo interface {
	RmAll() error
	GetAll() ([]*harpe.Stock, []*harpe.Short, []*harpe.Crypto, []*harpe.Option, error)
	GetExitChan(string) chan bool
	SetAndReturnNewExitChan(string, chan bool) chan bool
	RefreshFromDB() ([]*harpe.Stock, []*harpe.Short, []*harpe.Crypto, []*harpe.Option, error)

	GetOption(string) (*harpe.Option, error)
	CreateOption(string, string, int, string, string, string, string, string, float32, float32, float32, float32, float32, float32) (chan bool, string, bool, error)
	RemoveOption(string, string, string, string, string, float32) error

	CreateShort(string, string, int, float32, float32, float32, float32, int64, float32) (chan bool, bool, error)
	RemoveShort(string) error
	GetShort(string) (*harpe.Short, error)

	CreateStock(string, string, int, float32, float32, float32, float32, int64, float32) (chan bool, bool, error)
	RemoveStock(string) error
	GetStock(string) (*harpe.Stock, error)

	CreateCrypto(string, string, float32, float32, float32, float32, int, float32) (chan bool, bool, error)
	RemoveCrypto(string) error
	GetCrypto(string) (*harpe.Crypto, error)
}
