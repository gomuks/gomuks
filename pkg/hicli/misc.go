// Copyright (c) 2025 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package hicli

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"

	"maunium.net/go/mautrix/crypto/canonicaljson"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

type CreatePDU struct {
	AuthEvents     []id.EventID    `json:"auth_events"`
	PrevEvents     []id.EventID    `json:"prev_events"`
	Depth          int             `json:"depth"`
	Hashes         *Hashes         `json:"hashes,omitempty"`
	OriginServerTS int64           `json:"origin_server_ts"`
	Sender         id.UserID       `json:"sender"`
	StateKey       string          `json:"state_key"`
	Type           event.Type      `json:"type"`
	Content        json.RawMessage `json:"content"`
}

type Hashes struct {
	SHA256 string `json:"sha256"`
}

func (h *HiClient) CalculateRoomID(timestamp int64, content json.RawMessage) (id.RoomID, error) {
	pdu := &CreatePDU{
		AuthEvents:     []id.EventID{},
		PrevEvents:     []id.EventID{},
		Depth:          1,
		Hashes:         nil,
		OriginServerTS: timestamp,
		Sender:         h.Account.UserID,
		Type:           event.StateCreate,
		Content:        content,
	}
	pduJSON, err := json.Marshal(pdu)
	if err != nil {
		return "", err
	}
	pduJSON = canonicaljson.CanonicalJSONAssumeValid(pduJSON)
	pduHash := sha256.Sum256(pduJSON)
	pdu.Hashes = &Hashes{
		SHA256: base64.RawStdEncoding.EncodeToString(pduHash[:]),
	}
	pduJSON, err = json.Marshal(pdu)
	if err != nil {
		return "", err
	}
	pduJSON = canonicaljson.CanonicalJSONAssumeValid(pduJSON)
	pduHash = sha256.Sum256(pduJSON)
	return id.RoomID("!" + base64.RawURLEncoding.EncodeToString(pduHash[:])), nil
}
