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
import React, { use, useCallback, useState } from "react"
import { StateStore, useAccountData } from "@/api/statestore"
import {
	ContentPushRule,
	PushRule,
	PushRuleAction,
	PushRuleKind,
	RidePushRule,
	UnknownPushRuleCondition,
} from "@/api/types"
import { ensureArray, ensureString, ensureTypedArray } from "@/util/validation.ts"
import ClientContext from "../ClientContext.ts"
import JSONView from "../util/JSONView"
import { DoneCallback, EventKind } from "./devtools-util.ts"
import DeleteIcon from "@/icons/delete.svg?react"
import "./PushRuleEditor.css"

interface PushRuleListProps {
	store: StateStore
	type: string
	onSelectRuleID: (ruleID: string) => void
	onBack: () => void
	onCreateNew: (kind: EventKind) => void
}

function isPushRule(content: unknown): content is PushRule {
	return typeof content === "object"
		&& content !== null
		&& "rule_id" in content
		&& typeof content.rule_id === "string"
		&& "actions" in content
		&& Array.isArray(content.actions)
		&& "default" in content
		&& typeof content.default === "boolean"
		&& "enabled" in content
		&& typeof content.enabled === "boolean"
}

function isPushRuleKind(kind: unknown): kind is PushRuleKind {
	return kind === "override" || kind === "content" || kind === "room" || kind === "sender" || kind === "underride"
}

export const PushRuleList = ({ store, type, onSelectRuleID, onBack, onCreateNew }: PushRuleListProps) => {
	const [filter, setFilter] = useState("")
	const rules = ensureTypedArray(useAccountData(store, "m.push_rules")?.global?.[type], isPushRule)
	if (!isPushRuleKind(type)) {
		return <div className="state-explorer">Invalid push rule kind</div>
	}
	return (
		<div className="state-explorer state-key-list">
			<div className="state-header">
				<h3><code>{type}</code> push rules</h3>
				<input
					type="search"
					className="search-field"
					placeholder="Filter rule IDs"
					value={filter}
					onChange={evt => setFilter(evt.target.value)}
				/>
			</div>
			<div className="state-button-list">
				{rules.map(pushRule => {
					const roomName = pushRule.rule_id.startsWith("!")
						? store.rooms.get(pushRule.rule_id)?.meta.current.name ?? "Unknown room" : ""
					return (pushRule.rule_id.includes(filter) || roomName.includes(filter))
						? <button key={pushRule.rule_id} onClick={() => onSelectRuleID(pushRule.rule_id)}>
							<code>{pushRule.rule_id}</code>
							{roomName ? ` (${roomName})` : ""}
						</button>
						: null
				})}
			</div>
			<div className="nav-buttons">
				<button onClick={onBack}>Back</button>
				<div className="spacer"/>
				<button onClick={() => onCreateNew(EventKind.PushRules)}>Create new rule</button>
			</div>
		</div>
	)
}

interface PushRuleViewProps {
	type: string
	id: string
	onBack: () => void
}

export const PushRuleView = ({ type, id, onBack }: PushRuleViewProps) => {
	const client = use(ClientContext)!
	const rulesOfType = ensureArray(useAccountData(client.store, "m.push_rules")?.global?.[type])
	const pushRule = rulesOfType.find((rule: unknown): rule is PushRule => isPushRule(rule) && rule.rule_id === id)
	const [editing, setEditing] = useState(false)
	const stopEditing = useCallback(() => setEditing(false), [])
	if (!isPushRuleKind(type)) {
		return <div className="state-explorer">Invalid push rule kind</div>
	}
	if (editing) {
		return <PushRuleEditor type={type} rule={pushRule} onBack={stopEditing} />
	}

	const doDelete = () => {
		client.rpc.updatePushRule(type, id, "delete").then(
			onBack,
			err => {
				console.error("Failed to delete push rule", err)
				window.alert(`Failed to delete push rule: ${err}`)
			},
		)
	}
	const doToggleEnable = () => {
		if (!pushRule) {
			return
		}
		client.rpc.updatePushRule(type, id, pushRule.enabled ? "disable" : "enable").catch(err => {
			console.error("Failed to toggle push rule", err)
			window.alert(`Failed to toggle push rule: ${err}`)
		})
	}
	const startEdit = () => {
		setEditing(true)
	}

	const roomName = id.startsWith("!") ? (client.store.rooms.get(id)?.meta.current.name ?? "Unknown room") : null
	return (
		<div className="state-explorer state-event-view">
			<div className="state-header">
				<h3>
					Push rule: <code>{type}/{id}</code>
					{roomName ? ` (${roomName})` : ""}
				</h3>
			</div>
			<div className="state-event-content">
				<JSONView data={pushRule ?? {}} />
			</div>
			<div className="nav-buttons">
				<button onClick={onBack}>Back</button>
				<div className="spacer"/>
				{pushRule && <>
					{!pushRule.default && <button onClick={doDelete}>Delete</button>}
					<button onClick={doToggleEnable}>{pushRule.enabled ? "Disable" : "Enable"}</button>
					{!pushRule.default && <button onClick={startEdit}>Edit</button>}
				</>}
			</div>
		</div>
	)
}

interface PushRuleEditorProps {
	type: string
	rule?: PushRule
	onBack: () => void
	onDone?: DoneCallback
}

const findHighlight = (action: PushRuleAction) => typeof action === "object" && action.set_tweak === "highlight"
const findSound = (action: PushRuleAction) => typeof action === "object" && action.set_tweak === "sound"

export const PushRuleEditor = ({ type, rule, onBack, onDone }: PushRuleEditorProps) => {
	const client = use(ClientContext)!
	const [newRuleID, setNewRuleID] = useState("")
	const [notify, setNotify] = useState(() => rule?.actions?.includes("notify") ?? false)
	const [highlight, setHighlight] = useState(() => !!rule?.actions?.find(findHighlight))
	const [sound, setSound] = useState(() => ensureString(rule?.actions?.find(findSound)?.value))
	const [pattern, setPattern] = useState((rule as ContentPushRule)?.pattern ?? "")
	const [conditions, setConditions] = useState<UnknownPushRuleCondition[]>((rule as RidePushRule)?.conditions ?? [])
	const onChangeCondition = useCallback((index: number, newCond: UnknownPushRuleCondition | null) => {
		setConditions(prev => {
			const upd = [...prev]
			if (newCond === null) {
				upd.splice(index, 1)
			} else {
				upd[index] = newCond
			}
			return upd
		})
	}, [])
	if (!isPushRuleKind(type)) {
		return <div className="state-explorer">Invalid push rule kind</div>
	}

	const isNewRule = !rule
	const sendEdit = () => {
		const realRuleID = newRuleID || rule?.rule_id
		if (!realRuleID) {
			return
		}
		const actions: PushRuleAction[] = []
		if (notify) {
			actions.push("notify")
		}
		if (highlight) {
			actions.push({ set_tweak: "highlight", value: true })
		}
		if (sound) {
			actions.push({ set_tweak: "sound", value: sound })
		}
		const newContent = {
			actions,
			conditions: type === "override" || type === "underride" ? conditions : undefined,
			pattern: type === "content" ? pattern : undefined,
		}
		client.rpc.updatePushRule(type, realRuleID, "put", newContent).then(
			() => onDone ? onDone(EventKind.PushRules, type, newRuleID, undefined) : onBack(),
			err => {
				console.error("Failed to put push rule", err)
				window.alert(`Failed to put push rule: ${err}`)
			},
		)
	}
	return (
		<div className="state-explorer state-event-view push-rule-editor">
			<div className="state-header">
				{isNewRule ? <>
					<h3>New <code>{type}</code> push rule</h3>
					<div className="new-event-type">
						<input
							autoFocus
							type="text"
							value={newRuleID}
							onChange={evt => setNewRuleID(evt.target.value)}
							placeholder="Rule ID"
						/>
					</div>
				</> : <h3>Edit push rule <code>{type}/{rule.rule_id}</code></h3>}
			</div>
			<div className="push-rule-actions">
				<h4>Actions</h4>
				<label>
					<input
						type="checkbox"
						checked={notify}
						onChange={evt => setNotify(evt.target.checked)}
					/>
					Notify
				</label>
				<label>
					<input
						type="checkbox"
						checked={highlight}
						onChange={evt => setHighlight(evt.target.checked)}
					/>
					Highlight
				</label>
				<label>
					Sound:
					<input
						type="text"
						value={sound}
						onChange={evt => setSound(evt.target.value)}
						placeholder="Sound name (e.g. default)"
					/>
				</label>
			</div>
			{type === "content" ? <div className="push-rule-pattern">
				<h4>Pattern</h4>
				<input
					type="text"
					value={pattern}
					onChange={evt => setPattern(evt.target.value)}
					placeholder="Pattern"
				/>
			</div> : null}
			{type === "override" || type === "underride" ? <div className="push-rule-conditions">
				<h4>Conditions</h4>
				{conditions.map((cond, idx) =>
					<PushRuleConditionEditor key={idx} index={idx} cond={cond} onChange={onChangeCondition} />)}
				<div>
					<button
						className="add-condition"
						onClick={() => setConditions(prev => [
							...prev, { kind: "event_match", key: "", pattern: "" },
						])}
					>Add condition</button>
				</div>
			</div> : null}
			<div className="nav-buttons">
				<button onClick={onBack}>Back</button>
				<div className="spacer"/>
				<button onClick={sendEdit}>Send</button>
			</div>
		</div>
	)
}

interface PushRuleConditionEditorProps {
	cond: UnknownPushRuleCondition
	index: number
	onChange: (index: number, newCond: UnknownPushRuleCondition | null) => void
}

const knownConditionKinds = [
	"event_match", "contains_display_name", "room_member_count", "sender_notification_permission",
	"event_property_is", "event_property_contains", "custom",
]

const knownConditionKindNames: Record<string, string> = {
	"event_match": "Event match",
	"contains_display_name": "Contains display name",
	"room_member_count": "Room member count",
	"sender_notification_permission": "Sender notification permission",
	"event_property_is": "Event property is",
	"event_property_contains": "Event property contains",
	"custom": "Custom",
}

const PushRuleConditionEditor = ({ cond, index, onChange }: PushRuleConditionEditorProps) => {
	const renderKind = knownConditionKinds.includes(cond.kind) ? cond.kind : "custom"
	const onChangeKind = (evt: React.ChangeEvent<HTMLSelectElement>) => {
		onChange(index, {
			...cond,
			kind: evt.currentTarget.value === "custom" ? "" : evt.currentTarget.value,
		})
	}
	return <div className={`push-rule-condition kind-${renderKind}`}>
		<select value={renderKind} onChange={onChangeKind}>
			{knownConditionKinds.map(kind => <option key={kind} value={kind}>
				{knownConditionKindNames[kind]}
			</option>)}
		</select>
		{renderConditionEditor(cond, index, onChange)}
		<button onClick={() => onChange(index, null)} className="delete-condition">
			<DeleteIcon />
		</button>
	</div>
}

const memberCountPrefixes = ["==", "<=", ">=", "<", ">"]

function parseMemberCountExpression(expr: string): [string, string] {
	for (const prefix of memberCountPrefixes) {
		if (expr.startsWith(prefix)) {
			return [prefix, expr.slice(prefix.length)]
		}
	}
	return ["==", expr]
}

function removeExtraFields(cond: UnknownPushRuleCondition): UnknownPushRuleCondition {
	switch (cond.kind) {
	case "event_match":
		return { kind: "event_match", key: ensureString(cond.key), pattern: ensureString(cond.pattern) }
	case "contains_display_name":
		return { kind: "contains_display_name" }
	case "room_member_count":
		return { kind: "room_member_count", is: ensureString(cond.is) }
	case "sender_notification_permission":
		return { kind: "sender_notification_permission", key: ensureString(cond.key) }
	case "event_property_is":
		return { kind: "event_property_is", key: ensureString(cond.key), value: cond.value }
	case "event_property_contains":
		return { kind: "event_property_contains", key: ensureString(cond.key), value: cond.value }
	default:
		return cond
	}
}

function renderConditionEditor(
	cond: UnknownPushRuleCondition, index: number, onChange: (index: number, newCond: UnknownPushRuleCondition) => void,
) {
	switch (cond.kind) {
	case "event_match":
	case "event_property_is":
	case "event_property_contains":
		return <>
			<label>
				Field
				<input
					type="text"
					value={ensureString(cond.key)}
					onChange={evt =>
						onChange(index, { ...removeExtraFields(cond), key: evt.currentTarget.value })}
					placeholder="Key"
				/>
			</label>
			<label>
				{cond.kind === "event_match" ? "matches" : cond.kind === "event_property_is" ? "is" : "contains"}
				<input
					type="text"
					value={cond.kind === "event_match" ? ensureString(cond.pattern) : cond.value}
					onChange={evt => onChange(index, {
						...removeExtraFields(cond),
						pattern: cond.kind === "event_match" ? evt.currentTarget.value : undefined,
						value: cond.kind !== "event_match" ? evt.currentTarget.value : undefined,
					})}
					placeholder="Pattern"
				/>
			</label>
		</>
	case "contains_display_name":
		return null
	case "room_member_count": {
		const [op, count] = parseMemberCountExpression(ensureString(cond.is))
		return <>
			<select
				value={op}
				onChange={evt =>
					onChange(index, { ...removeExtraFields(cond), is: `${evt.currentTarget.value}${count}` })}
			>
				{memberCountPrefixes.map(prefix => <option key={prefix} value={prefix}>{prefix}</option>)}
			</select>
			<input
				type="number"
				value={count}
				onChange={evt =>
					onChange(index, { ...removeExtraFields(cond), is: `${op}${evt.currentTarget.value}` })}
				min={0}
				placeholder="Count"
			/>
		</>
	}
	case "sender_notification_permission":
		return <label>
			for key
			<input
				type="text"
				value={ensureString(cond.key)}
				onChange={evt =>
					onChange(index, { ...removeExtraFields(cond), key: evt.currentTarget.value })}
				placeholder="Permission (e.g. room)"
			/>
		</label>
	default:
		return <RawConditionEditor cond={cond} index={index} onChange={onChange} />
	}
}

interface RawConditionEditorProps {
	cond: UnknownPushRuleCondition
	index: number
	onChange: (index: number, cond: UnknownPushRuleCondition) => void
}

const RawConditionEditor = ({ cond, index, onChange }: RawConditionEditorProps) => {
	const [value, setValue] = useState(() => JSON.stringify(cond, null, 4))
	const [isValid, setIsValid] = useState(true)
	return <textarea
		value={value}
		rows={6}
		className={isValid ? "" : "invalid"}
		onChange={evt => {
			setValue(evt.target.value)
			try {
				onChange(index, JSON.parse(evt.target.value))
				setIsValid(true)
			} catch {
				setIsValid(false)
			}
		}}
	/>
}
