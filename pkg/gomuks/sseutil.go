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
	"cmp"
	"encoding/json"
	"io"
	"iter"
	"net/http"
	"slices"
	"strings"

	"github.com/klauspost/compress/flate"
	"github.com/klauspost/compress/zstd"
	"github.com/rs/zerolog"
	"maunium.net/go/mautrix"
)

type sseWriter struct {
	rc *http.ResponseController
	w  io.Writer
	j  *json.Encoder
	c  compressor

	simplified bool
}

type compressor interface {
	io.WriteCloser
	Flush() error
}

func newSSEWriter(w http.ResponseWriter, r *http.Request) *sseWriter {
	_, ok := w.(http.Flusher)
	if !ok {
		zerolog.Ctx(r.Context()).Error().Type("writer_type", w).Msg("ResponseWriter does not support flushing")
		mautrix.MUnknown.WithMessage("ResponseWriter does not support flushing").Write(w)
		return nil
	}
	rc := http.NewResponseController(w)
	var simplified bool
	if strings.Contains(r.Header.Get("Accept"), "application/jsonl") {
		w.Header().Set("Content-Type", "application/jsonl")
		simplified = true
	} else {
		w.Header().Set("Content-Type", "text/event-stream")
	}
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	var c compressor
	var err error
	if strings.Contains(r.Header.Get("Accept-Encoding"), "zstd") {
		w.Header().Set("Content-Encoding", "zstd")
		c, err = zstd.NewWriter(w, zstd.WithEncoderConcurrency(1), zstd.WithEncoderLevel(zstd.SpeedDefault))
	} else if strings.Contains(r.Header.Get("Accept-Encoding"), "deflate") {
		w.Header().Set("Content-Encoding", "deflate")
		c, err = flate.NewWriter(w, flate.DefaultCompression)
	}
	if err != nil {
		zerolog.Ctx(r.Context()).Err(err).Msg("Failed to create compression writer")
		return nil
	}
	rawWriter := cmp.Or[io.Writer](c, w)
	w.WriteHeader(http.StatusOK)
	return &sseWriter{
		rc: rc,
		w:  rawWriter,
		j:  json.NewEncoder(rawWriter),
		c:  c,

		simplified: simplified,
	}
}

var ssePingBytes = []byte(":\n\n")
var jsonlPingBytes = []byte("null\n")
var sseDataBytes = []byte("data:")

func (w *sseWriter) write(cmd *BufferedEvent) (err error) {
	if !w.simplified {
		_, err = w.w.Write(sseDataBytes)
		if err != nil {
			return
		}
	}
	err = w.j.Encode(cmd)
	if err != nil {
		return
	}
	if !w.simplified {
		// Double newline for SSE. Encode() emits one newline for jsonl on its own.
		_, err = w.w.Write(newlineBytes)
	}
	return
}

func (w *sseWriter) flush() error {
	if w.c != nil {
		err := w.c.Flush()
		if err != nil {
			return err
		}
	}
	return w.rc.Flush()
}

func (w *sseWriter) ping() (err error) {
	if w.simplified {
		_, err = w.w.Write(jsonlPingBytes)
	} else {
		_, err = w.w.Write(ssePingBytes)
	}
	if err != nil {
		return err
	}
	return w.flush()
}

func (w *sseWriter) writeAndFlush(cmd *BufferedEvent, extra iter.Seq[*BufferedEvent]) (err error) {
	if cmd != nil {
		err = w.write(cmd)
		if err != nil {
			return
		}
	}
	if extra != nil {
		err = w.writeSeq(extra)
		if err != nil {
			return
		}
	}
	return w.flush()
}

func (w *sseWriter) writeSeq(cmds iter.Seq[*BufferedEvent]) (err error) {
	if cmds == nil {
		return
	}
	for item := range cmds {
		err = w.write(item)
		if err != nil {
			return
		}
	}
	return
}

func (w *sseWriter) writeMany(cmds ...*BufferedEvent) (err error) {
	return w.writeSeq(slices.Values(cmds))
}
