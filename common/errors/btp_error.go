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

package errors

import (
	"errors"
	"fmt"
)

var (
	ErrConnectFail            = errors.New("fail to connect")
	ErrSendFailByExpired      = errors.New("reject by expired")
	ErrSendFailByFuture       = errors.New("reject by future")
	ErrSendFailByOverflow     = errors.New("reject by overflow")
	ErrGetResultFailByPending = errors.New("fail to getresult by pending")
)

const (
	SUCCESS      Code = -1
	CodeBTP      Code = 0
	CodeBMC      Code = 10
	CodeBMV      Code = 25
	CodeBSH      Code = 40
	CodeReserved Code = 55
)

const (
	BMCRevert = CodeBMC + iota
	BMCRevertUnauthorized
	BMCRevertInvalidSN
	BMCRevertAlreadyExistsBMV
	BMCRevertNotExistsBMV
	BMCRevertAlreadyExistsBSH
	BMCRevertNotExistsBSH
	BMCRevertAlreadyExistsLink
	BMCRevertNotExistsLink
	BMCRevertUnreachable
	BMCRevertNotExistsPermission
)

var (
	BMCRevertCodeNames = map[Code]string{
		BMCRevert:                    "BMCRevert",
		BMCRevertUnauthorized:        "BMCRevertUnauthorized",
		BMCRevertInvalidSN:           "BMCRevertInvalidSN",
		BMCRevertAlreadyExistsBMV:    "BMCRevertAlreadyExistsBMV",
		BMCRevertNotExistsBMV:        "BMCRevertNotExistsBMV",
		BMCRevertAlreadyExistsBSH:    "BMCRevertAlreadyExistsBSH",
		BMCRevertNotExistsBSH:        "BMCRevertNotExistsBSH",
		BMCRevertAlreadyExistsLink:   "BMCRevertAlreadyExistsLink",
		BMCRevertNotExistsLink:       "BMCRevertNotExistsLink",
		BMCRevertUnreachable:         "BMCRevertUnreachable",
		BMCRevertNotExistsPermission: "BMCRevertNotExistsPermission",
	}
)

const (
	BMVUnknown = CodeBMV + iota
	BMVNotVerifiable
	BMVAlreadyVerified
	BMVRevertInvalidBlockWitnessOld
)

var (
	BMVRevertCodeNames = map[Code]string{
		BMVUnknown:                      "BMVRevert",
		BMVNotVerifiable:                "BMVNotVerifiable",
		BMVAlreadyVerified:              "BMVAlreadyVerified",
		BMVRevertInvalidBlockWitnessOld: "BMVRevertInvalidBlockWitnessOld",
	}
)

func NewRevertError(code int) error {
	c := Code(code)
	if c >= CodeBTP {
		var msg string
		var ok bool
		if c < CodeBMC {
			msg = fmt.Sprintf("BTPRevert[%d]", c)
		} else if c < CodeBMV {
			if msg, ok = BMCRevertCodeNames[c]; !ok {
				msg = fmt.Sprintf("BMCRevert[%d]", c)
			}
		} else if c < CodeBSH {
			if msg, ok = BMVRevertCodeNames[c]; !ok {
				msg = fmt.Sprintf("BMVRevert[%d]", c)
			}
		} else if c < CodeReserved {
			msg = fmt.Sprintf("BSHRevert[%d]", c)
		} else {
			msg = fmt.Sprintf("ReservedRevert[%d]", c)
		}
		return NewBase(c, msg)
	}
	return nil
}
