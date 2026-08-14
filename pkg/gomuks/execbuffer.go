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
	"sync"
	"time"

	"github.com/rs/zerolog"
)

const ExecutionBufferCleanupDelay = 5 * time.Minute

type execution[Result any, Error error] struct {
	ch   chan struct{}
	data Result
	err  Error
}

type ExecutionBuffer[Result any, Error error] struct {
	bgCtx    context.Context
	lock     sync.Mutex
	requests map[string]execution[Result, Error]
}

func NewExecutionBuffer[Result any, Error error](bgCtx context.Context) *ExecutionBuffer[Result, Error] {
	return &ExecutionBuffer[Result, Error]{
		bgCtx:    bgCtx,
		requests: make(map[string]execution[Result, Error]),
	}
}

func (eb *ExecutionBuffer[Result, Error]) Do(
	ctx context.Context,
	txnID string,
	fn func(context.Context) (Result, Error),
) (Result, Error) {
	if txnID == "" {
		return fn(ctx)
	}
	eb.lock.Lock()
	req, ok := eb.requests[txnID]
	if !ok {
		req = execution[Result, Error]{ch: make(chan struct{})}
		eb.requests[txnID] = req
	}
	eb.lock.Unlock()
	if ok {
		<-req.ch
		return req.data, req.err
	}
	req.data, req.err = fn(zerolog.Ctx(ctx).WithContext(eb.bgCtx))
	close(req.ch)
	go func() {
		time.Sleep(ExecutionBufferCleanupDelay)
		eb.lock.Lock()
		delete(eb.requests, txnID)
		eb.lock.Unlock()
	}()
	return req.data, req.err
}
