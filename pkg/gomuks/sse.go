// gomuks - A Matrix client written in Go.
// Copyright (C) 2026 Tulir Asokan
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package gomuks

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/coder/websocket"
	"github.com/rs/zerolog"
	"maunium.net/go/mautrix"

	"go.mau.fi/gomuks/pkg/hicli/jsoncmd"
)

func (gmx *Gomuks) HandleSSE(w http.ResponseWriter, r *http.Request) {
	log := zerolog.Ctx(r.Context())
	sw := newSSEWriter(w, r)
	if sw == nil {
		return
	}

	resumeFrom, lastServerTS, resumeRunID, prevListenerID := gmx.parseSocketParams(r.URL.Query())
	log.Info().
		Int64("resume_from", resumeFrom).
		Int64("resume_run_id", resumeRunID).
		Int64("resume_server_ts", lastServerTS).
		Uint64("prev_listener_id", prevListenerID).
		Int64("current_run_id", runID).
		Bool("compressed", sw.c != nil).
		Msg("Accepting new SSE connection")
	ctx, cancel := context.WithCancelCause(r.Context())
	defer cancel(fmt.Errorf("defer cancel"))
	evts := make(chan *BufferedEvent, 512)
	listenerID, resumeData := gmx.EventBuffer.Subscribe(resumeFrom, func(statusCode websocket.StatusCode, reason string) {
		cancel(fmt.Errorf("closed by buffer: %s", reason))
	}, func(evt *BufferedEvent) {
		if ctx.Err() != nil {
			return
		}
		select {
		case evts <- evt:
		default:
			log.Warn().Msg("Event queue full, closing connection")
			cancel(fmt.Errorf("event queue full"))
		}
	})
	defer gmx.EventBuffer.Unsubscribe(listenerID)

	initErr := sw.writeMany(
		jsoncmd.SpecRunID.Format(&jsoncmd.RunData{
			RunID:      strconv.FormatInt(runID, 10),
			ETag:       gmx.frontendETag,
			VAPIDKey:   gmx.Config.Push.VAPIDPublicKey,
			ListenerID: listenerID,
		}).AsAny(),
		jsoncmd.SpecClientState.Format(gmx.Client.State()).AsAny(),
		jsoncmd.SpecSyncStatus.Format(gmx.Client.SyncStatus.Load()).AsAny(),
	)
	if initErr != nil {
		log.Err(initErr).Msg("Failed to write init client state message")
		return
	}
	sendImageAuthToken := func() {
		err := sw.write(jsoncmd.SpecImageAuthToken.Format(gmx.GenerateImageToken(1 * time.Hour)).AsAny())
		if err != nil {
			cancel(fmt.Errorf("failed to write image auth token: %w", err))
		}
	}
	sendImageAuthToken()
	err := sw.rc.Flush()
	if err != nil {
		log.Err(err).Msg("Failed to flush initial messages to client")
		return
	}
	var inited bool
	if resumeData != nil {
		err = sw.writeMany(resumeData...)
		if err != nil {
			log.Err(err).Msg("Failed to write resume data to client")
			return
		}
		resumeData = nil
		inited = true
	} else {
		err = gmx.Client.Initialized.Wait(ctx)
		if err != nil {
			return
		}
		if gmx.Client.IsLoggedInAndVerified() {
			var roomCount int
			for payload := range gmx.Client.GetInitialSync(ctx, 100, lastServerTS) {
				roomCount += len(payload.Rooms)
				err = sw.writeAndFlush(jsoncmd.SpecSyncComplete.Format(payload).AsAny(), nil)
				if err != nil {
					log.Err(err).Msg("Failed to send initial rooms to client")
					return
				}
			}
			log.Info().
				Int("room_count", roomCount).
				Int64("catchup_since", lastServerTS).
				Msg("Sent initial rooms to client")
			inited = true
		}
	}
	if inited {
		err = sw.writeAndFlush(jsoncmd.SpecInitComplete.Format(jsoncmd.InitComplete{}).AsAny(), nil)
		if err != nil {
			log.Err(err).Msg("Failed to write init complete message")
			return
		}
	}
	log.Debug().Bool("did_resume", resumeData != nil).Msg("Connection initialization complete")
	imageAuthTicker := time.NewTicker(30 * time.Minute)
	pingTimer := time.NewTimer(15 * time.Second)
	for {
		select {
		case evt := <-evts:
			err = sw.writeAndFlush(evt, chanToSeq(evts))
			if err != nil {
				log.Err(err).Msg("Failed to write outgoing event to client")
				return
			}
			log.Trace().Int64("req_id", evt.RequestID).Msg("Sent outgoing event")
		case <-imageAuthTicker.C:
			sendImageAuthToken()
		case <-pingTimer.C:
			err = sw.ping()
			if err != nil {
				log.Err(err).Msg("Failed to write ping to client")
				return
			}
		case <-ctx.Done():
			log.Debug().Err(context.Cause(ctx)).Msg("SSE connection done")
			return
		}
	}
}

func (gmx *Gomuks) HandleSSEPing(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	pingRunID, _ := strconv.ParseInt(r.URL.Query().Get("run_id"), 10, 64)
	listenerID, _ := strconv.ParseUint(q.Get("listener_id"), 10, 64)
	lastReceivedEvent, _ := strconv.ParseInt(q.Get("last_received_event"), 10, 64)
	if pingRunID != runID {
		mautrix.MBadState.WithMessage("Run ID mismatch").Write(w)
	} else if listenerID == 0 {
		mautrix.MInvalidParam.WithMessage("Invalid listener ID").Write(w)
	} else if lastReceivedEvent == 0 {
		mautrix.MInvalidParam.WithMessage("Invalid last event ID").Write(w)
	} else {
		gmx.EventBuffer.SetLastAckedID(listenerID, lastReceivedEvent)
	}
}
