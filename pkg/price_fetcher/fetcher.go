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
package fetcher

type Fetcher interface {
	GetStock(string) (float32, error)
	GetCrypto(string, bool) (float32, error)
	GetOption(string, string, string, string, string, float32, float32) (float32, string, error)
	// This one we'll do locally
	//GetOptionAdvanced(string, string, string, string, string, float32, float32) (*Snapshot, string, error)
}
