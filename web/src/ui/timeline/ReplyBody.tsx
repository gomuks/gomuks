// gomuks - A Matrix client written in Go.
// Copyright (C) 2024 Tulir Asokan
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
import { use } from "react"
import { getAvatarThumbnailURL, getUserColorIndex } from "@/api/media.ts"
import {
	applyPerMessageSender,
	maybeRedactMemberEvent,
	useRoomEvent,
	useRoomMember,
} from "@/api/statestore"
import { EventID, MemDBEvent } from "@/api/types"
import { displayAsRedacted } from "@/util/displayAsRedacted.ts"
import { getDisplayname } from "@/util/validation.ts"
import ClientContext from "../ClientContext.ts"
import MainScreenContext from "../MainScreenContext.ts"
import { RoomContextData } from "../roomview/roomcontext.ts"
import TooltipButton from "../util/TooltipButton.tsx"
import { jumpToEvent } from "../util/jumpToEvent.tsx"
import { ContentErrorBoundary, getBodyType, getPerMessageProfile } from "./content"
import CloseIcon from "@/icons/close.svg?react"
import EditIcon from "@/icons/edit.svg?react"
import NotificationsOffIcon from "@/icons/notifications-off.svg?react"
import NotificationsIcon from "@/icons/notifications.svg?react"
import ReplyIcon from "@/icons/reply.svg?react"
import ThreadIcon from "@/icons/thread.svg?react"
import "./ReplyBody.css"

interface ReplyBodyProps {
	roomCtx: RoomContextData
	event: MemDBEvent
	isThread: boolean
	threadRoot?: EventID
	small?: boolean
	isEditing?: boolean
	onClose?: (evt: React.MouseEvent) => void
	isSilent?: boolean
	onSetSilent?: (evt: React.MouseEvent) => void
	isExplicitInThread?: boolean
	onSetExplicitInThread?: (evt: React.MouseEvent) => void
	startNewThread?: boolean
	onSetStartNewThread?: (evt: React.MouseEvent) => void
}

interface ReplyIDBodyProps {
	roomCtx: RoomContextData
	eventID: EventID
	isThread: boolean
	threadRoot?: EventID
	small: boolean
}

export const ReplyIDBody = ({ roomCtx, eventID, isThread, threadRoot, small }: ReplyIDBodyProps) => {
	const event = useRoomEvent(roomCtx.store, eventID)
	if (!event) {
		// This caches whether the event is requested or not, so it doesn't need to be wrapped in an effect.
		use(ClientContext)!.requestEvent(roomCtx.store, eventID)
		return <blockquote className={`reply-body sender-color-null ${small ? "small" : ""}`}>
			{small && <div className="reply-spine"/>}
			Reply to unknown event
			{!small && <br/>}
			<code>{eventID}</code>
		</blockquote>
	}
	return <ReplyBody roomCtx={roomCtx} event={event} isThread={isThread} threadRoot={threadRoot} small={small}/>
}

export const ReplyBody = ({
	roomCtx, event, onClose, isThread, threadRoot, isEditing, small,
	isSilent, onSetSilent,
	isExplicitInThread, onSetExplicitInThread,
	startNewThread, onSetStartNewThread,
}: ReplyBodyProps) => {
	const room = roomCtx.store
	const client = use(ClientContext)
	const mainScreen = use(MainScreenContext)
	const memberEvt = useRoomMember(client, room, event.sender)
	const memberEvtContent = maybeRedactMemberEvent(memberEvt)
	const BodyType = getBodyType(
		event, displayAsRedacted(event, memberEvt, room), true,
	)
	const classNames = ["reply-body"]
	if (onClose) {
		classNames.push("composer")
	}
	if (isThread) {
		classNames.push("thread")
	}
	if (isEditing) {
		classNames.push("editing")
	}
	if (small) {
		classNames.push("small")
	}
	const perMessageSender = getPerMessageProfile(event)
	const renderMemberEvtContent = applyPerMessageSender(memberEvtContent, perMessageSender)
	const userColorIndex = getUserColorIndex(perMessageSender?.id ?? event.sender)
	classNames.push(`sender-color-${userColorIndex}`)
	const onClick = () => {
		if (isThread && threadRoot) {
			mainScreen.setRightPanel({
				type: "thread",
				threadRoot,
			})
		} else {
			jumpToEvent(roomCtx, event.event_id)
		}
	}
	return <blockquote className={classNames.join(" ")} onClick={onClick}>
		{small && <div className="reply-spine"/>}
		<div className="reply-sender">
			<div
				className="sender-avatar"
				title={perMessageSender ? `${perMessageSender.id} via ${event.sender}` : event.sender}
			>
				<img
					className="small avatar"
					loading="lazy"
					src={getAvatarThumbnailURL(perMessageSender?.id ?? event.sender, renderMemberEvtContent)}
					alt=""
				/>
			</div>
			<span
				className={`event-sender sender-color-${userColorIndex}`}
				title={perMessageSender ? perMessageSender.id : event.sender}
			>
				{getDisplayname(event.sender, renderMemberEvtContent)}
			</span>
			{isThread ? <span className="sender-extra thread-logo">
				(<ThreadIcon width=".75rem" height=".75rem" /> thread)
			</span> : null}
			{isEditing ? <span className="sender-extra editing-logo">
				(<EditIcon width="1rem" height="1rem"/> editing)
			</span> : null}
			{perMessageSender && <div className="per-message-event-sender">
				<span className="via">via</span>
				<span
					className={`event-sender sender-color-${getUserColorIndex(event.sender)}`}
					title={event.sender}
				>
					{getDisplayname(event.sender, memberEvtContent)}
				</span>
			</div>}
			{onClose && <div className="buttons">
				{onSetSilent && (isExplicitInThread || !isThread) && <TooltipButton
					tooltipText={isSilent
						? "Click to enable pinging the original author"
						: "Click to disable pinging the original author"}
					tooltipDirection="left"
					className="silent-reply"
					onClick={onSetSilent}
				>
					{isSilent ? <NotificationsOffIcon /> : <NotificationsIcon />}
				</TooltipButton>}
				{isThread && onSetExplicitInThread && <TooltipButton
					tooltipText={isExplicitInThread
						? "Click to respond in thread without replying to a specific message"
						: "Click to reply explicitly in thread"}
					tooltipDirection="left"
					className="thread-explicit-reply"
					onClick={onSetExplicitInThread}
				>
					{isExplicitInThread ? <ReplyIcon /> : <ThreadIcon />}
				</TooltipButton>}
				{!isThread && onSetStartNewThread && <TooltipButton
					tooltipText={startNewThread
						? "Click to reply in main timeline instead of starting a new thread"
						: "Click to start a new thread instead of replying"}
					tooltipDirection="left"
					className="thread-explicit-reply"
					onClick={onSetStartNewThread}
				>
					{startNewThread ? <ThreadIcon /> : <ReplyIcon />}
				</TooltipButton>}
				{onClose && <button className="close-reply" onClick={onClose}><CloseIcon/></button>}
			</div>}
		</div>
		<ContentErrorBoundary>
			<BodyType room={room} event={event} sender={memberEvt}/>
		</ContentErrorBoundary>
	</blockquote>
}
