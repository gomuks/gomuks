// gomuks - A Matrix client written in Go.
// Copyright (C) 2025 Tulir Asokan
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
import React, { JSX, use, useCallback, useRef, useState } from "react"
import { getAvatarThumbnailURL, getMediaURL, getUserColorIndex } from "@/api/media.ts"
import {
	RoomStateStore,
	applyPerMessageSender,
	maybeRedactMemberEvent,
	useRoomMember,
} from "@/api/statestore"
import { MemDBEvent, URLPreview as URLPreviewType, UnreadType } from "@/api/types"
import { formatDate, formatFullTime, formatShortTime, newSafeDate } from "@/util/datetime.ts"
import { displayAsRedacted } from "@/util/displayAsRedacted.ts"
import { isMobileDevice } from "@/util/ismobile.ts"
import { getDisplayname, getRelatesTo, getThreadRoot, isEventID, isThread } from "@/util/validation.ts"
import ClientContext from "../ClientContext.ts"
import MainScreenContext from "../MainScreenContext.ts"
import { EventFullMenu, EventHoverMenu, MenuPositioner } from "../menu"
import { ModalContext, NestableModalContext, modals } from "../modal"
import { useRoomContext } from "../roomview/roomcontext.ts"
import URLPreview from "../urlpreview/URLPreview.tsx"
import { jumpToEventInView } from "../util/jumpToEvent.tsx"
import { useHorizontalSwipe } from "../util/swipe.ts"
import ReadReceipts from "./ReadReceipts.tsx"
import { ReplyBody, ReplyIDBody } from "./ReplyBody.tsx"
import { ContentErrorBoundary, HiddenEvent, getBodyType, getPerMessageProfile, isSmallEvent } from "./content"
import ErrorIcon from "@/icons/error.svg?react"
import PendingIcon from "@/icons/pending.svg?react"
import SentIcon from "@/icons/sent.svg?react"
import "./TimelineEvent.css"

export type TimelineEventViewType =
	"timeline" | "thread" | "context" | "pinned" | "edit-history" | "confirm" | "notifications"

export interface TimelineEventProps {
	evt: MemDBEvent
	prevEvt: MemDBEvent | null
	disableMenu?: boolean
	smallReplies?: boolean
	smallThreads?: boolean
	viewType: TimelineEventViewType
}

interface EventReactionsProps {
	reactions: Record<string, number>
	onRereact: (mouseEvt: React.MouseEvent) => void
}

const EventReactions = ({ reactions, onRereact }: EventReactionsProps) => {
	const reactionEntries = Object.entries(reactions).filter(([, count]) => count > 0).sort((a, b) => b[1] - a[1])
	if (reactionEntries.length === 0) {
		return null
	}
	return <div className="event-reactions">
		{reactionEntries.map(([reaction, count]) =>
			<div key={reaction} className="reaction" title={reaction} onClick={onRereact}>
				{reaction.startsWith("mxc://")
					? <img className="reaction-emoji" src={getMediaURL(reaction)} alt=""/>
					: <span className="reaction-emoji">{reaction}</span>}
				<span className="reaction-count">{count}</span>
			</div>)}
	</div>
}

const EventSendStatus = ({ evt }: { evt: MemDBEvent }) => {
	if (evt.send_error && evt.send_error !== "not sent") {
		return <div className="event-send-status error" title={evt.send_error}><ErrorIcon/></div>
	} else if (evt.event_id.startsWith("~")) {
		return <div title="Waiting for /send to return" className="event-send-status sending"><PendingIcon/></div>
	} else if (evt.pending) {
		return <div title="Waiting to receive event in /sync" className="event-send-status sent"><SentIcon/></div>
	} else {
		return <div title="Event sent and remote echo received" className="event-send-status sent"><SentIcon/></div>
	}
}

const EventURLPreviews = ({ event, room }: {
	room: RoomStateStore
	event: MemDBEvent
}) => {
	const previews = (event.content["com.beeper.linkpreviews"] ?? event.content["m.url_previews"]) as URLPreviewType[]
	if (!previews) {
		return null
	}
	return <div className="url-previews">
		{previews.map((p, i) => <URLPreview key={i} room={room} preview={p}/>)}
	</div>
}

const TimelineEvent = ({
	evt, prevEvt, disableMenu, smallReplies, smallThreads, viewType,
}: TimelineEventProps) => {
	const roomCtx = useRoomContext()
	const client = use(ClientContext)!
	const mainScreen = use(MainScreenContext)
	const openModal = use(ModalContext)
	const openNestableModal = use(NestableModalContext)
	const [forceContextMenuOpen, setForceContextMenuOpen] = useState(false)
	const isFocused = useRef(false)
	const eventRef = useRef<HTMLDivElement>(null)
	const onContextMenu = (mouseEvt: React.MouseEvent) => {
		if (mouseEvt.shiftKey || disableMenu) {
			return
		}
		const targetElem = mouseEvt.target as HTMLElement
		if (
			!roomCtx.store.preferences.message_context_menu
			|| targetElem.tagName === "A"
			|| targetElem.tagName === "IMG"
			|| targetElem.tagName === "VIDEO"
			|| window.getSelection()?.type === "Range"
		) {
			return
		}
		mouseEvt.preventDefault()
		openModal({
			content: <MenuPositioner
				x={mouseEvt.clientX}
				y={mouseEvt.clientY}
				anchor="click"
				Child={EventFullMenu}
				evt={evt}
				roomCtx={roomCtx}
			/>,
		})
	}
	const onRereact = useCallback((mouseEvt: React.MouseEvent) => {
		client.sendEvent(evt.room_id, "m.reaction", {
			"m.relates_to": {
				rel_type: "m.annotation",
				event_id: evt.event_id,
				key: mouseEvt.currentTarget.getAttribute("title"),
			},
		}).catch(err => {
			console.error("Failed to send reaction", err)
			window.alert(`Failed to send reaction: ${err}`)
		})
		mouseEvt.stopPropagation()
	}, [client, evt])
	const onClickTimestamp = (mouseEvt: React.MouseEvent) => {
		if (viewType === "pinned" || (viewType === "notifications" && evt.room_id === roomCtx.store.roomID)) {
			mouseEvt.stopPropagation()
			jumpToEventInView(roomCtx, evt.event_id, document.querySelector("div.room-view"))
		} else if (viewType === "notifications") {
			mouseEvt.stopPropagation()
			mainScreen.setActiveRoom(evt.room_id, { openEventID: evt.event_id })
		}
	}
	const openEditHistory = (mouseEvt: React.MouseEvent) => {
		mouseEvt.stopPropagation()
		openNestableModal(modals.eventEditHistory(roomCtx, evt))
	}
	const perMessageSender = getPerMessageProfile(evt)
	const prevPerMessageSender = getPerMessageProfile(prevEvt)
	const memberEvt = useRoomMember(client, roomCtx.store, evt.sender)
	const memberEvtContent = maybeRedactMemberEvent(memberEvt)
	const renderMemberEvtContent = applyPerMessageSender(memberEvtContent, perMessageSender)

	const eventTS = newSafeDate(evt.timestamp)
	const editEventTS = evt.last_edit ? newSafeDate(evt.last_edit.timestamp) : null

	const relatesTo = getRelatesTo(evt)
	const replyTo = relatesTo?.["m.in_reply_to"]?.event_id
	const isFallbackReply = relatesTo?.is_falling_back
	const threadRoot = getThreadRoot(relatesTo)
	const isSmallThreadMessage = Boolean(threadRoot && smallThreads)

	const [onTouchStart, onTouchMove, onTouchEnd, onTouchCancel] = useHorizontalSwipe({
		onTrigger: () => {
			roomCtx.setReplyTo(evt.event_id)
		},
		startThreshold: 20,
		verticalLimit: 10,
		minTriggerDistance: 60,
		maxDistance: 90,
		left: true,
		enabled: (viewType === "timeline" || viewType === "thread") && !isSmallThreadMessage,
	})
	const enableTouchMenu = isMobileDevice && !disableMenu
	const onClick = (mouseEvt: React.MouseEvent<HTMLDivElement>) => {
		const targetElem = mouseEvt.target as HTMLElement
		if (targetElem.closest("a, img, video, summary, iframe, div.media-container, div.location-container.leaflet")) {
			return
		}

		mouseEvt.preventDefault()
		mouseEvt.stopPropagation()
		isFocused.current = true
		eventRef.current?.classList.add("focused-event")
		openModal({
			content: <MenuPositioner
				x={mouseEvt.clientX}
				y={mouseEvt.clientY}
				anchor="touch"
				Child={EventFullMenu}
				evt={evt}
				roomCtx={roomCtx}
			/>,
			onClose: () => {
				isFocused.current = false
				eventRef.current?.classList.remove("focused-event")
			},
		})
	}

	const wrapperClassNames = ["timeline-event"]
	const isRedacted = displayAsRedacted(evt, memberEvt, roomCtx.store)
	const BodyType = getBodyType(evt, isRedacted, isSmallThreadMessage)
	if (isRedacted) {
		wrapperClassNames.push("redacted-event")
	}
	if (evt.unread_type & UnreadType.Highlight) {
		wrapperClassNames.push("highlight")
	}
	if (evt.type === "m.room.member") {
		wrapperClassNames.push("membership-event")
	}
	if (BodyType === HiddenEvent) {
		wrapperClassNames.push("hidden-event")
	}
	if (evt.sender === client.userID) {
		wrapperClassNames.push("own-event")
	}
	if (enableTouchMenu) {
		wrapperClassNames.push("mobile-menu")
	}
	if (isMobileDevice || disableMenu) {
		wrapperClassNames.push("no-hover")
	}
	if (isFocused.current) {
		wrapperClassNames.push("focused-event")
	}
	if (evt.unsigned["io.element.synapse.soft_failed"]) {
		wrapperClassNames.push("soft-failed")
	}
	if (evt.unsigned["io.element.synapse.policy_server_spammy"]) {
		wrapperClassNames.push("policy-server-spammy")
	}
	let dateSeparator = null
	const showInitialDateSeparator = viewType === "timeline" || viewType === "thread"
		|| viewType === "context" || viewType === "notifications"
	const prevEvtDate = prevEvt ? newSafeDate(prevEvt.timestamp) : null
	if (
		(showInitialDateSeparator && !prevEvt)
		|| (prevEvtDate && (
			eventTS.getDate() !== prevEvtDate.getDate() ||
			eventTS.getMonth() !== prevEvtDate.getMonth() ||
			eventTS.getFullYear() !== prevEvtDate.getFullYear()
		))
	) {
		dateSeparator = <div className="date-separator">
			<hr role="none"/>
			{formatDate(eventTS)}
			<hr role="none"/>
		</div>
	}
	const isSmallBodyType = isSmallEvent(BodyType)
	let replyAboveMessage: JSX.Element | null = null
	let replyInMessage: JSX.Element | null = null
	if (
		isEventID(replyTo)
		&& BodyType !== HiddenEvent
		&& !isSmallThreadMessage
		&& !isRedacted
		&& viewType !== "edit-history"
		&& (!isFallbackReply || viewType !== "thread")
	) {
		const replyElem = <ReplyIDBody
			roomCtx={roomCtx}
			eventID={replyTo}
			isThread={viewType !== "thread" && isThread(relatesTo)}
			threadRoot={threadRoot}
			small={!!smallReplies}
		/>
		if (smallReplies && !isSmallBodyType) {
			replyAboveMessage = replyElem
			wrapperClassNames.push("reply-above")
		} else {
			replyInMessage = replyElem
		}
	}

	let smallAvatar = false
	let renderAvatar = true
	let eventTimeOnly = false
	if (isSmallBodyType) {
		wrapperClassNames.push("small-event")
		smallAvatar = true
		eventTimeOnly = true
	} else if (
		prevEvt?.sender === evt.sender
		&& prevEvt.timestamp + 15 * 60 * 1000 > evt.timestamp
		&& dateSeparator === null
		&& !replyAboveMessage
		&& !isSmallEvent(getBodyType(prevEvt, displayAsRedacted(prevEvt, memberEvt, roomCtx.store)))
		&& prevPerMessageSender?.id === perMessageSender?.id
	) {
		wrapperClassNames.push("same-sender")
		eventTimeOnly = true
		renderAvatar = false
	}
	if (viewType === "edit-history") {
		wrapperClassNames.push("edit-history-event")
	}
	if (isSmallThreadMessage) {
		const prevRelatesTo = getRelatesTo(prevEvt)
		if (dateSeparator === null && getThreadRoot(prevRelatesTo) === threadRoot) {
			wrapperClassNames.push("same-thread")
		}
		wrapperClassNames.push("small-thread-message")
		eventTimeOnly = true
		renderAvatar = !smallAvatar
		smallAvatar = false
	}

	const fullTime = formatFullTime(eventTS)
	const shortTime = formatShortTime(eventTS)
	const renderMemberEventDisplayname = getDisplayname(evt.sender, renderMemberEvtContent)
	const mainMemberEventDisplayname = getDisplayname(evt.sender, memberEvtContent)
	const mainEvent = <div
		data-event-id={evt.event_id}
		className={wrapperClassNames.join(" ")}
		onContextMenu={!isMobileDevice ? onContextMenu : undefined}
		onTouchStart={onTouchStart}
		onTouchMove={onTouchMove}
		onTouchEnd={onTouchEnd}
		onTouchCancel={onTouchCancel}
		onClick={enableTouchMenu ? onClick : undefined}
		ref={eventRef}
	>
		{!disableMenu && !isMobileDevice ? <div
			className={`context-menu-container ${forceContextMenuOpen ? "force-open" : ""}`}
		>
			<EventHoverMenu evt={evt} roomCtx={roomCtx} setForceOpen={setForceContextMenuOpen}/>
		</div> : null}
		{replyAboveMessage}
		{renderAvatar && <div
			className="sender-avatar"
			title={perMessageSender ? `${perMessageSender.id} via ${evt.sender}` : evt.sender}
			data-target-panel="user"
			data-target-user={evt.sender}
			onClick={mainScreen.clickRightPanelOpener}
		>
			<img
				className={`${smallAvatar ? "small" : ""} avatar`}
				loading="lazy"
				src={getAvatarThumbnailURL(perMessageSender?.id ?? evt.sender, renderMemberEvtContent)}
				alt=""
			/>
		</div>}
		{!eventTimeOnly ? <div className="event-sender-and-time">
			<span
				className={`event-sender sender-color-${getUserColorIndex(perMessageSender?.id ?? evt.sender)}`}
				data-target-user={evt.sender}
				onClick={perMessageSender ? undefined : roomCtx.appendMentionToComposer}
				title={perMessageSender ? perMessageSender.id : evt.sender}
			>
				{renderMemberEventDisplayname}
			</span>
			{renderMemberEventDisplayname !== mainMemberEventDisplayname && <>
				<span className="via">via</span>
				<span
					className={`event-sender original sender-color-${getUserColorIndex(evt.sender)}`}
					data-target-user={evt.sender}
					onClick={roomCtx.appendMentionToComposer}
					title={evt.sender}
				>
					{mainMemberEventDisplayname}
				</span>
			</>}
			<span className="event-time" title={fullTime} onClick={onClickTimestamp}>{shortTime}</span>
		</div> : <div className="event-time-only" onClick={onClickTimestamp}>
			<span className="event-time" title={fullTime}>{shortTime}</span>
		</div>}
		{isSmallThreadMessage ? <ReplyBody
			roomCtx={roomCtx}
			event={evt}
			isThread={true}
			threadRoot={threadRoot}
			timelineThreadMsg={true}
			reactions={evt.reactions ? <EventReactions reactions={evt.reactions} onRereact={onRereact} /> : null}
		/> : <div className="event-content">
			{replyInMessage}
			<ContentErrorBoundary>
				<BodyType room={roomCtx.store} sender={memberEvt} event={evt}/>
				{!isSmallBodyType && !isRedacted && <EventURLPreviews room={roomCtx.store} event={evt}/>}
			</ContentErrorBoundary>
			{evt.send_error && evt.send_error !== "not sent" ? <div className="event-send-error">
				{evt.send_error}
			</div> : null}
			{(viewType !== "edit-history" && editEventTS) ? <div
				className="event-edited"
				title={`Edited at ${formatFullTime(editEventTS)}`}
				onClick={openEditHistory}
			>
				(edited at {formatShortTime(editEventTS)})
			</div> : null}
			{evt.reactions ? <EventReactions reactions={evt.reactions} onRereact={onRereact} /> : null}
		</div>}
		{!evt.event_id.startsWith("~")
			&& roomCtx.store.preferences.display_read_receipts
			&& viewType === "timeline"
			? <ReadReceipts room={roomCtx.store} eventID={evt.event_id} extraEvents={evt.receipt_flattening} /> : null}
		{evt.sender === client.userID && evt.transaction_id && viewType !== "edit-history"
			? <EventSendStatus evt={evt}/> : null}
	</div>
	return <>
		{dateSeparator}
		{mainEvent}
	</>
}

export default React.memo(TimelineEvent)
