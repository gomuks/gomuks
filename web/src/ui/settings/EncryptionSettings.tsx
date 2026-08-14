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
import { JSX, use, useEffect, useMemo, useState } from "react"
import { GridLoader } from "react-spinners"
import { RoomStateStore } from "@/api/statestore"
import { DeviceID, GetOwnDevicesResponse, OwnDevice, ProfileDevice } from "@/api/types"
import { formatFullTime } from "@/util/datetime.ts"
import ClientContext from "../ClientContext.ts"
import KeyExportView from "./KeyExportView.tsx"
import EncryptedOffIcon from "@/icons/encrypted-off.svg?react"
import EncryptedQuestionIcon from "@/icons/encrypted-question.svg?react"
import EncryptedIcon from "@/icons/encrypted.svg?react"

const deltaFormatter = new Intl.RelativeTimeFormat("en-GB")

const MINUTE = 60 * 1000
const HOUR = 60 * MINUTE
const DAY = 24 * HOUR
const WEEK = 7 * DAY
const MONTH = 30 * DAY
const YEAR = 365 * DAY

function pickUnit(x: number): Intl.RelativeTimeFormatUnit {
	x = Math.abs(x)
	if (x < MINUTE) {
		return "second"
	} else if (x < HOUR) {
		return "minute"
	} else if (x < DAY) {
		return "hour"
	} else if (x < WEEK) {
		return "day"
	} else if (x < MONTH) {
		return "week"
	} else if (x < YEAR) {
		return "month"
	}
	return "year"
}

function roundToUnit(x: number, unit: Intl.RelativeTimeFormatUnit) {
	switch (unit) {
	case "second":
		return Math.round(x / 1000)
	case "minute":
		return Math.round(x / MINUTE)
	case "hour":
		return Math.round(x / HOUR)
	case "day":
		return Math.round(x / DAY)
	case "week":
		return Math.round(x / WEEK)
	case "month":
		return Math.round(x / MONTH)
	case "year":
		return Math.round(x / YEAR)
	default:
		throw new Error(`Unknown unit: ${unit}`)
	}
}

function formatLastSeen(sinceLastSeen: number): string {
	const unit = pickUnit(sinceLastSeen)
	return deltaFormatter.format(-roundToUnit(sinceLastSeen, unit), unit)
}

interface DeviceInfoProps {
	dev: OwnDevice
	enc?: ProfileDevice
	isCurrent: boolean
}

const DeviceInfo = ({ dev, enc, isCurrent }: DeviceInfoProps) => {
	let icon: null | JSX.Element
	if (!enc) {
		icon = <EncryptedOffIcon className="encryption-shield" />
	} else if (enc.trust_state === "cross-signed-verified" || enc.trust_state === "cross-signed-tofu") {
		icon = <EncryptedIcon color="var(--primary-color)" className="encryption-shield" />
	} else {
		icon = <EncryptedQuestionIcon color="var(--error-color)" className="encryption-shield" />
	}
	const lastSeen = new Date(dev.last_seen_ts)
	const sinceLastSeen = Date.now() - dev.last_seen_ts

	return <div key={dev.device_id} className="device-info">
		{icon}
		<div className="device-name">{dev.display_name}</div>
		<div className="metadata">
			<code className="device-id">{dev.device_id}</code>
			{isCurrent ? <span className="last-seen" title={formatFullTime(lastSeen)}>
				· Current device
			</span> : dev.last_seen_ts > 0 ? <span className="last-seen" title={formatFullTime(lastSeen)}>
				· Last seen {formatLastSeen(sinceLastSeen)}
			</span> : <span className="last-seen">
				· Never seen online
			</span>}
			{dev.last_seen_ip != "" ? <>
				<span className="last-seen-ip">at {dev.last_seen_ip}</span>
			</> : null}
		</div>
	</div>
}

const DevicesInfo = ({ info }: { info: GetOwnDevicesResponse }) => {
	const deviceMap = useMemo(() => {
		info.devices.sort((a, b) => {
			if (a.device_id === info.current_device.device_id) {
				return -1
			}
			if (b.device_id === info.current_device.device_id) {
				return 1
			}
			return b.last_seen_ts - a.last_seen_ts
		})
		const map = new Map<DeviceID, ProfileDevice>()
		for (const dev of info.encryption.devices) {
			map.set(dev.device_id, dev)
		}
		return map
	}, [info])
	return <>
		<div className="cross-signing-key">
			Cross-signing master key:
			<code>{info.encryption.master_key}</code>
		</div>

		{info.devices.map(dev => <DeviceInfo
			key={dev.device_id}
			isCurrent={dev.device_id === info.current_device.device_id}
			dev={dev}
			enc={deviceMap.get(dev.device_id)}
		/>)}
	</>
}

interface EncryptionSettingsProps {
	room?: RoomStateStore
}

const EncryptionSettings = ({ room }: EncryptionSettingsProps) => {
	const client = use(ClientContext)!
	const [info, setInfo] = useState<GetOwnDevicesResponse | null>(null)
	useEffect(() => {
		client.rpc.getOwnDevices()
			.then(setInfo, err => window.alert(`Failed to load devices: ${err}`))
	}, [client])

	return <>
		<div className="encryption-devices">
			<h3>Devices</h3>
			{info === null ? <GridLoader color="var(--primary-color)" /> : <DevicesInfo info={info} />}
		</div>
		<KeyExportView room={room} />
	</>
}

export default EncryptionSettings
