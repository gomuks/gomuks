// Copyright (c) 2025 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package store

import (
	"cmp"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"
	"sync"
	"sync/atomic"

	badGlobalLog "github.com/rs/zerolog/log"
	"github.com/tidwall/gjson"
	"go.mau.fi/util/exmaps"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/event/cmdschema"
	"maunium.net/go/mautrix/id"

	"go.mau.fi/gomuks/pkg/hicli/cmdspec"
	"go.mau.fi/gomuks/pkg/hicli/database"
	"go.mau.fi/gomuks/pkg/hicli/jsoncmd"
)

type AutocompleteMemberEntry struct {
	UserID       id.UserID
	Displayname  string
	AvatarURL    id.ContentURI
	SearchString string
	Membership   event.Membership
	Event        *database.Event
}

func StateKeySub(evtType event.Type, stateKey string) string {
	return fmt.Sprintf("%s\x00%s", evtType.Type, stateKey)
}

type RoomStore struct {
	parent     *GomuksStore
	lock       sync.RWMutex
	ID         id.RoomID
	Meta       EventDispatcher[*database.Room]
	Hidden     bool
	Paginating atomic.Bool

	TimelineCache     EventDispatcher[*[]*database.Event]
	accountData       map[event.Type]*database.AccountData
	timeline          []database.TimelineRowTuple
	hasMoreHistory    bool
	editTargets       []database.EventRowID
	eventsByRowID     map[database.EventRowID]*database.Event
	eventsByID        map[id.EventID]*database.Event
	requestedEvents   exmaps.Set[database.EventRowID]
	state             map[event.Type]map[string]database.EventRowID
	StateSubs         MultiNotifier[string]
	AccountDataSubs   MultiNotifier[event.Type]
	EventSubs         MultiNotifier[id.EventID]
	StateLoadLock     sync.Mutex
	StateLoaded       atomic.Bool
	FullMembersLoaded atomic.Bool
	requestedMembers  exmaps.Set[id.UserID]
	pendingEvents     []database.EventRowID
	membersCache      []*AutocompleteMemberEntry
	botCommandCache   []*WrappedCommand
	Typing            EventDispatcher[[]id.UserID]
	PreferenceCache   EventDispatcher[*Preferences]
	lastMarkedRead    database.EventRowID
}

type WrappedCommand struct {
	*cmdschema.EventContent
	Source id.UserID
}

func NewRoomStore(parent *GomuksStore, meta *database.Room) *RoomStore {
	return &RoomStore{
		ID:               meta.ID,
		parent:           parent,
		Meta:             *NewEventDispatcherWithValue(meta),
		accountData:      make(map[event.Type]*database.AccountData),
		state:            make(map[event.Type]map[string]database.EventRowID),
		hasMoreHistory:   true,
		eventsByRowID:    make(map[database.EventRowID]*database.Event),
		eventsByID:       make(map[id.EventID]*database.Event),
		requestedEvents:  make(exmaps.Set[database.EventRowID]),
		requestedMembers: make(exmaps.Set[id.UserID]),
	}
}

func (rs *RoomStore) GetPaginationParams() (oldestRowID database.TimelineRowID, count int) {
	rs.lock.RLock()
	defer rs.lock.RUnlock()
	if len(rs.timeline) > 0 {
		oldestRowID = rs.timeline[0].Timeline
	}
	if len(rs.timeline) < 100 {
		count = 50
	} else {
		count = 100
	}
	return
}

func (rs *RoomStore) notifyTimelineWatchers() {
	var ownMessages []database.EventRowID
	timelineCache := make([]*database.Event, 0, len(rs.timeline)+len(rs.pendingEvents))
	for _, tuple := range rs.timeline {
		evt, ok := rs.eventsByRowID[tuple.Event]
		if !ok {
			badGlobalLog.Debug().Any("tuple", tuple).Msg("MEOW??")
			continue
		}
		evt.TimelineRowID = tuple.Timeline
		timelineCache = append(timelineCache, evt)
		if evt.Sender == rs.parent.UserID && evt.GetType() == event.EventMessage && evt.RelationType != event.RelReplace {
			ownMessages = append(ownMessages, evt.RowID)
		}
	}
	for _, rowID := range rs.pendingEvents {
		evt, ok := rs.eventsByRowID[rowID]
		if !ok {
			continue
		}
		timelineCache = append(timelineCache, evt)
	}
	rs.TimelineCache.Emit(&timelineCache)
	rs.editTargets = ownMessages
}

func (rs *RoomStore) ApplySync(sync *jsoncmd.SyncRoom) {
	rs.lock.Lock()
	defer rs.lock.Unlock()
	if !rs.Meta.Current().VisibleMetaIsEqual(sync.Meta) {
		rs.Meta.Emit(sync.Meta)
	} else {
		rs.Meta.SetCurrent(sync.Meta)
	}
	for _, evt := range sync.Events {
		rs.applyEvent(evt, false)
	}
	for evtType, ad := range sync.AccountData {
		evtType.Class = event.AccountDataEventType
		if evtType == AccountDataGomuksPreferences {
			parsedPreferences := DefaultPreferences
			_ = json.Unmarshal(ad.Content, &parsedPreferences)
			rs.PreferenceCache.Emit(&parsedPreferences)
		}
		rs.accountData[evtType] = ad
		rs.AccountDataSubs.Notify(evtType)
	}
	for evtType, stateMap := range sync.State {
		evtType.Class = event.StateEventType
		cacheMap, ok := rs.state[evtType]
		if !ok {
			cacheMap = make(map[string]database.EventRowID)
			rs.state[evtType] = cacheMap
		}
		maps.Copy(cacheMap, stateMap)
		rs.invalidateStateCaches(evtType, slices.Collect(maps.Keys(stateMap))...)
	}
	if sync.Reset {
		rs.timeline = sync.Timeline
		rs.pendingEvents = rs.pendingEvents[:0]
	} else {
		rs.timeline = append(rs.timeline, sync.Timeline...)
	}
	if sync.Reset || len(sync.Timeline) > 0 {
		rs.notifyTimelineWatchers()
	}
}

func (rs *RoomStore) ApplyTyping(typing []id.UserID) {
	rs.Typing.Emit(typing)
}

func (rs *RoomStore) ApplyPending(evt *database.Event) {
	rs.lock.Lock()
	defer rs.lock.Unlock()
	if _, hasEvt := rs.eventsByRowID[evt.RowID]; hasEvt {
		return
	}
	if !slices.Contains(rs.pendingEvents, evt.RowID) {
		rs.pendingEvents = append(rs.pendingEvents, evt.RowID)
	}
	isFake := evt.Sender == cmdspec.FakeGomuksSender
	rs.applyEvent(evt, !isFake)
	if isFake {
		content := evt.GetMautrixContent().AsMessage()
		if content.FormattedBody == "" && evt.LocalContent.SanitizedHTML != "" && !evt.LocalContent.WasPlaintext {
			content.FormattedBody = evt.LocalContent.SanitizedHTML
			content.Format = event.FormatHTML
		}
		rs.timeline = append(rs.timeline, database.TimelineRowTuple{
			Timeline: evt.TimelineRowID,
			Event:    evt.RowID,
		})
	}
	rs.notifyTimelineWatchers()
}

func (rs *RoomStore) ApplySendComplete(evt *database.Event) {
	rs.lock.Lock()
	defer rs.lock.Unlock()
	if existing, hasEvt := rs.eventsByRowID[evt.RowID]; hasEvt && !existing.Pending {
		return
	}
	rs.applyEvent(evt, true)
	rs.notifyTimelineWatchers()
}

func (rs *RoomStore) ApplyPagination(resp *jsoncmd.PaginationResponse) {
	rs.lock.Lock()
	defer rs.lock.Unlock()
	rs.hasMoreHistory = resp.HasMore
	newTimeline := make([]database.TimelineRowTuple, 0, len(resp.Events))
	for _, evt := range slices.Backward(resp.Events) {
		rs.applyEvent(evt, false)
		newTimeline = append(newTimeline, database.TimelineRowTuple{
			Timeline: evt.TimelineRowID,
			Event:    evt.RowID,
		})
	}
	for _, evt := range resp.RelatedEvents {
		if _, hasEvt := rs.eventsByRowID[evt.RowID]; !hasEvt {
			rs.applyEvent(evt, false)
		}
	}
	rs.timeline = append(newTimeline, rs.timeline...)
	rs.notifyTimelineWatchers()
}

func (rs *RoomStore) ApplyDecrypted(resp *jsoncmd.EventsDecrypted) {
	rs.lock.Lock()
	defer rs.lock.Unlock()
	timelineChanged := false
	for _, evt := range resp.Events {
		if !timelineChanged {
			timelineChanged = slices.ContainsFunc(rs.timeline, func(tuple database.TimelineRowTuple) bool {
				return tuple.Event == evt.RowID
			})
		}
		rs.applyEvent(evt, false)
	}
	if timelineChanged {
		rs.notifyTimelineWatchers()
	}
	if resp.PreviewEventRowID != 0 {
		meta := rs.Meta.Current()
		meta.PreviewEventRowID = resp.PreviewEventRowID
		rs.Meta.Emit(meta)
	}
}

func (rs *RoomStore) ApplyState(evt *database.Event) {
	evtType := event.Type{Type: evt.Type, Class: event.StateEventType}
	rs.applyEvent(evt, false)
	stateMap, ok := rs.state[evtType]
	if !ok {
		stateMap = make(map[string]database.EventRowID)
		rs.state[evtType] = stateMap
	}
	stateMap[*evt.StateKey] = evt.RowID
	rs.invalidateStateCaches(evtType, *evt.StateKey)
	rs.StateSubs.Notify(evtType.Type)
	rs.StateSubs.Notify(StateKeySub(evtType, *evt.StateKey))
}

func (rs *RoomStore) invalidateStateCaches(evtType event.Type, stateKeys ...string) {
	switch evtType {
	case event.StateMember:
		for _, key := range stateKeys {
			rs.requestedMembers.Remove(id.UserID(key))
		}
		fallthrough
	case event.StatePowerLevels:
		rs.membersCache = nil
	case event.StateMSC4391BotCommand:
		rs.botCommandCache = nil
	}
	rs.StateSubs.Notify(evtType.Type)
	for _, stateKey := range stateKeys {
		rs.StateSubs.Notify(StateKeySub(evtType, stateKey))
	}
}

func (rs *RoomStore) ApplyFullState(events []*database.Event, omitMembers bool) {
	newStateMap := make(map[event.Type]map[string]database.EventRowID)
	for _, evt := range events {
		rs.applyEvent(evt, false)
		evtType := event.Type{Type: evt.Type, Class: event.StateEventType}
		stateMap, ok := newStateMap[evtType]
		if !ok {
			stateMap = make(map[string]database.EventRowID)
			newStateMap[evtType] = stateMap
		}
		stateMap[*evt.StateKey] = evt.RowID
	}
	if omitMembers {
		newStateMap[event.StateMember] = rs.state[event.StateMember]
	} else {
		rs.membersCache = nil
	}
	rs.botCommandCache = nil
	rs.state = newStateMap
	rs.StateLoaded.Store(true)
	if !omitMembers {
		rs.FullMembersLoaded.Store(true)
	}
	for evtType, stateMap := range newStateMap {
		rs.StateSubs.Notify(evtType.Type)
		for stateKey := range stateMap {
			rs.StateSubs.Notify(StateKeySub(evtType, stateKey))
		}
	}
}

const UnsentTimelineRowIDBase database.TimelineRowID = 1000000000000000

func (rs *RoomStore) applyEvent(evt *database.Event, pending bool) {
	if pending {
		evt.TimelineRowID = UnsentTimelineRowIDBase + database.TimelineRowID(evt.Timestamp.UnixMilli())
		evt.Pending = true
	}
	if evt.LastEditRowID != nil && *evt.LastEditRowID != 0 {
		evt.LastEditRef = rs.eventsByRowID[*evt.LastEditRowID]
	} else if evt.RelationType == event.RelReplace && evt.RelatesTo != "" {
		editTarget, ok := rs.eventsByID[evt.RelatesTo]
		if ok && editTarget.LastEditRowID != nil && *editTarget.LastEditRowID != 0 && *editTarget.LastEditRowID == evt.RowID {
			editTarget.LastEditRef = editTarget
			rs.EventSubs.Notify(editTarget.ID)
		}
	}
	rs.eventsByRowID[evt.RowID] = evt
	rs.eventsByID[evt.ID] = evt
	rs.requestedEvents.Remove(evt.RowID)
	if !pending {
		if pendingIdx := slices.Index(rs.pendingEvents, evt.RowID); pendingIdx != -1 {
			rs.pendingEvents = slices.Delete(rs.pendingEvents, pendingIdx, pendingIdx+1)
		}
	}
	rs.EventSubs.Notify(evt.ID)
}

func toSearchableString(s string) string {
	// TODO
	return s
}

func (rs *RoomStore) fillMembersCache() {
	memberEvtIDs, ok := rs.state[event.StateMember]
	if !ok {
		return
	}
	entries := make([]*AutocompleteMemberEntry, 0, len(memberEvtIDs))
	for stateKey, evtRowID := range memberEvtIDs {
		evt, ok := rs.eventsByRowID[evtRowID]
		if !ok {
			continue
		}
		membership := event.Membership(gjson.GetBytes(evt.Content, "membership").Str)
		if membership != event.MembershipJoin && membership != event.MembershipInvite {
			continue
		}
		displayName := gjson.GetBytes(evt.Content, "displayname").Str
		avatarURL, _ := id.ParseContentURI(gjson.GetBytes(evt.Content, "avatar_url").Str)
		entries = append(entries, &AutocompleteMemberEntry{
			UserID:       id.UserID(stateKey),
			Displayname:  cmp.Or(displayName, id.UserID(stateKey).Localpart()),
			AvatarURL:    avatarURL,
			Event:        evt,
			Membership:   membership,
			SearchString: toSearchableString(displayName + stateKey[1:]),
		})
	}
	rs.membersCache = entries
}

func (rs *RoomStore) GetPowerLevels() *event.PowerLevelsEventContent {
	evt := rs.GetStateEvent(event.StatePowerLevels, "")
	if evt == nil {
		return &event.PowerLevelsEventContent{}
	}
	createEvt := rs.GetStateEvent(event.StateCreate, "")
	if createEvt == nil {
		return &event.PowerLevelsEventContent{}
	}
	pls := evt.GetMautrixContent().AsPowerLevels()
	pls.CreateEvent = createEvt.AsMautrix()
	return pls
}

func (rs *RoomStore) GetMembers() []*AutocompleteMemberEntry {
	rs.lock.RLock()
	cache := rs.membersCache
	rs.lock.RUnlock()
	if cache == nil {
		rs.lock.Lock()
		defer rs.lock.Unlock()
		if rs.membersCache == nil {
			rs.fillMembersCache()
		}
		cache = rs.membersCache
	}
	return cache
}

func (rs *RoomStore) fillBotCommandCache() {
	botCommandEvtIDs, ok := rs.state[event.StateMSC4391BotCommand]
	if !ok {
		return
	}
	commands := make([]*WrappedCommand, 0, len(botCommandEvtIDs))
	for _, evtRowID := range botCommandEvtIDs {
		evt, ok := rs.eventsByRowID[evtRowID]
		if !ok || evt.RedactedBy != "" {
			continue
		}
		// TODO check sender membership
		cmdContent, ok := evt.GetMautrixContent().Parsed.(*cmdschema.EventContent)
		if !ok || !cmdContent.IsValid() {
			continue
		}
		commands = append(commands, &WrappedCommand{
			EventContent: cmdContent,
			Source:       evt.Sender,
		})
	}
	rs.botCommandCache = commands
}

func (rs *RoomStore) GetBotCommands() []*WrappedCommand {
	rs.lock.RLock()
	cache := rs.botCommandCache
	rs.lock.RUnlock()
	if cache == nil {
		rs.lock.Lock()
		defer rs.lock.Unlock()
		if rs.botCommandCache == nil {
			rs.fillBotCommandCache()
		}
		cache = rs.botCommandCache
	}
	return cache
}

func (rs *RoomStore) GetEventByRowID(rowID database.EventRowID) *database.Event {
	rs.lock.RLock()
	defer rs.lock.RUnlock()
	return rs.eventsByRowID[rowID]
}

func (rs *RoomStore) GetEventByID(evtID id.EventID) *database.Event {
	if evtID == "" {
		return nil
	}
	rs.lock.RLock()
	defer rs.lock.RUnlock()
	return rs.eventsByID[evtID]
}

func (rs *RoomStore) GetStateEvent(evtType event.Type, stateKey string) *database.Event {
	rs.lock.RLock()
	defer rs.lock.RUnlock()
	stateMap, ok := rs.state[evtType]
	if !ok {
		return nil
	}
	rowID, ok := stateMap[stateKey]
	if !ok {
		return nil
	}
	evt, ok := rs.eventsByRowID[rowID]
	if !ok {
		return nil
	}
	return evt
}

func (rs *RoomStore) GetMember(userID id.UserID) *event.MemberEventContent {
	evt := rs.GetStateEvent(event.StateMember, userID.String())
	if evt == nil {
		return nil
	}
	return evt.GetMautrixContent().AsMember()
}

func (rs *RoomStore) GetDisplayname(userID id.UserID) string {
	memberEvt := rs.GetMember(userID)
	if memberEvt == nil || memberEvt.Displayname == "" {
		return userID.Localpart()
	}
	return memberEvt.Displayname
}

func (rs *RoomStore) GetMarkAsReadParams() *jsoncmd.MarkReadParams {
	rs.lock.RLock()
	defer rs.lock.RUnlock()
	if len(rs.timeline) == 0 {
		return nil
	}
	var readEvt *database.Event
	for i := len(rs.timeline) - 1; i >= 0; i-- {
		tuple := rs.timeline[i]
		if tuple.Event == rs.lastMarkedRead {
			break
		}
		evt, ok := rs.eventsByRowID[tuple.Event]
		if ok && strings.HasPrefix(evt.ID.String(), "$") && evt.Sender != cmdspec.FakeGomuksSender {
			readEvt = evt
			rs.lastMarkedRead = tuple.Event
			break
		}
	}
	if readEvt == nil {
		return nil
	}
	// TODO get receipt type from preferences
	receiptType := event.ReceiptTypeReadPrivate
	return &jsoncmd.MarkReadParams{
		RoomID:      rs.ID,
		EventID:     readEvt.ID,
		ReceiptType: receiptType,
	}
}
