import { use, useEffect, useState } from "react"
import Client from "@/api/client.ts"
import { JSONValue, PronounSet, UserID, UserProfile } from "@/api/types"
import { ensureArray, ensureString } from "@/util/validation.ts"
import { ModalContext, modals } from "../modal"

interface ExtendedProfileProps {
	profile: UserProfile | null
	refreshProfile: () => void
	client: Client
	userID: string
}

interface SetTimezoneProps {
	tz?: string
	client: Client
	refreshProfile: () => void
}

const getCurrentTimezone = () => new Intl.DateTimeFormat().resolvedOptions().timeZone

const currentTimeAdjusted = (tz: string) => {
	try {
		return new Intl.DateTimeFormat("en-GB", {
			hour: "numeric",
			minute: "numeric",
			second: "numeric",
			timeZoneName: "short",
			timeZone: tz,
		}).format(new Date())
	} catch {
		return null
	}
}

const ClockElement = ({ tz }: { tz: string }) => {
	const cta = currentTimeAdjusted(tz)
	const isValidTZ = cta !== null
	const [time, setTime] = useState(cta)
	useEffect(() => {
		if (!isValidTZ) {
			return
		}
		let interval: ReturnType<typeof setInterval> | undefined
		const updateTime = () => setTime(currentTimeAdjusted(tz))
		const timeout = setTimeout(() => {
			interval = setInterval(updateTime, 1000)
			updateTime()
		}, (1001 - Date.now() % 1000))
		return () => interval ? clearInterval(interval) : clearTimeout(timeout)
	}, [tz, isValidTZ])

	if (!isValidTZ) {
		return null
	}
	return <>
		<div title={tz}>Time:</div>
		<div title={tz}>{time}</div>
	</>
}

const SetTimeZoneElement = ({ tz, client, refreshProfile }: SetTimezoneProps) =>  {
	const zones = Intl.supportedValuesOf("timeZone")
	const saveTz = (newTz: string) => {
		if (!zones.includes(newTz)) {
			return
		}
		client.rpc.setProfileField("m.tz", newTz).then(
			() => refreshProfile(),
			err => {
				console.error("Failed to set time zone:", err)
				window.alert(`Failed to set time zone: ${err}`)
			},
		)
	}

	const defaultValue = tz || getCurrentTimezone()
	return <>
		<label htmlFor="userprofile-timezone-input">Set time zone:</label>
		<input
			list="timezones"
			id="userprofile-timezone-input"
			defaultValue={defaultValue}
			onKeyDown={evt => evt.key === "Enter" && saveTz(evt.currentTarget.value)}
			onBlur={evt => evt.currentTarget.value !== defaultValue && saveTz(evt.currentTarget.value)}
		/>
		<datalist id="timezones">
			{zones.map((zone) => <option key={zone} value={zone} />)}
		</datalist>
	</>
}

interface PronounInputProps {
	pronouns: PronounSet[]
	client: Client
	refreshProfile: () => void
	userID: UserID
}

const simplePronounOptions: PronounSet[] = [
	{ grammatical_gender: "", summary: "unset", language: "" },
	{ grammatical_gender: "masculine", summary: "he/him", language: "en" },
	{ grammatical_gender: "feminine", summary: "she/her", language: "en" },
	{ grammatical_gender: "neuter", summary: "they/them", language: "en" },
	{ grammatical_gender: "inanimate", summary: "it/its", language: "en" },
]

function simplePronounID(pronouns: PronounSet[]): string | null {
	if (pronouns.length === 0) {
		return ""
	} else if (pronouns.length === 1) {
		const p = pronouns[0]
		return simplePronounOptions.find(option =>
			option.grammatical_gender === p.grammatical_gender
			&& option.language === p.language
			&& option.summary === p.summary)?.grammatical_gender ?? null
	} else {
		return null
	}
}

const SimplePronouns = ({ pronouns, client, refreshProfile, userID }: PronounInputProps) => {
	const id = simplePronounID(pronouns)
	if (userID !== client.userID || id === null) {
		return <div>
			{pronouns.map(pronounSet => ensureString(pronounSet.summary)).join(", ")}
		</div>
	}
	const savePronouns = (evt: React.ChangeEvent<HTMLSelectElement>) => {
		const set = simplePronounOptions.find(option => option.grammatical_gender === evt.currentTarget.value)
		if (!set) {
			return
		}
		const val = set.summary === "unset" ? undefined : [set] as unknown as JSONValue
		client.rpc.setProfileField("io.fsky.nyx.pronouns", val).then(
			() => refreshProfile(),
			err => {
				console.error("Failed to set pronouns:", err)
				window.alert(`Failed to set pronouns: ${err}`)
			},
		)
	}
	return <select value={id} onChange={savePronouns}>
		{simplePronounOptions.map(item => <option value={item.grammatical_gender}>{item.summary}</option>)}
	</select>
}

const UserExtendedProfile = ({ profile, refreshProfile, client, userID }: ExtendedProfileProps)=>  {
	const openModal = use(ModalContext)!
	if (!profile) {
		return null
	}

	const hasExtendedProfile = Object.keys(profile).some((key) => key !== "displayname" && key !== "avatar_url")
	if (!hasExtendedProfile && client.userID !== userID) {
		return null
	}

	const viewExtensibleProfile = () => {
		openModal(modals.jsonView(profile))
	}
	// Explicitly only return something if the profile has the keys we're looking for.
	// otherwise there's an ugly and pointless <hr/> for no real reason.

	const pronouns = ensureArray(profile["io.fsky.nyx.pronouns"]) as PronounSet[]
	const userTimeZone = ensureString(profile["m.tz"] ?? profile["us.cloke.msc4175.tz"])
	return <div className="extended-profile">
		{userTimeZone && <ClockElement tz={userTimeZone} />}
		{userID === client.userID &&
			<SetTimeZoneElement tz={userTimeZone} client={client} refreshProfile={refreshProfile} />}
		{(pronouns.length > 0 || userID === client.userID) && <>
			<div>Pronouns:</div>
			<SimplePronouns pronouns={pronouns} client={client} refreshProfile={refreshProfile} userID={userID} />
		</>}
		<button onClick={viewExtensibleProfile}>View raw profile</button>
	</div>
}

export default UserExtendedProfile
