/*
 * Copyright 2021 ICON Foundation
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

package chain

import (
	"encoding/json"

	"github.com/icon-project/btp2/common/types"
)

type BaseConfig struct {
	Address           types.BtpAddress       `json:"address"`
	Endpoint          string                 `json:"endpoint"`
	KeyStoreData      json.RawMessage        `json:"key_store"`
	KeyStorePass      string                 `json:"key_password,omitempty"`
	KeySecret         string                 `json:"key_secret,omitempty"`
	RelayMode         string                 `json:"relay_mode"` //trustless, bridge
	LatestResult      bool                   `json:"latest_result"`
	FilledBlockUpdate bool                   `json:"filled_block_update"`
	Options           map[string]interface{} `json:"options,omitempty"`
}
