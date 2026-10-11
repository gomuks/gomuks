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
import { use, useEffect, useRef, useState } from "react"
import { ModalCloseContext } from "../modal"
import DeleteIcon from "@/icons/delete.svg?react"
import PauseIcon from "@/icons/pause.svg?react"
import PlayIcon from "@/icons/play.svg?react"
import StopIcon from "@/icons/stop.svg?react"
import "./VoiceRecorder.css"

interface VoiceRecorderProps {
	onFinish: (file: File) => void
}

function chooseMime() {
	for (const mime of ["audio/ogg; codecs=opus", "audio/webm; codecs=opus", "audio/mp4"]) {
		if (MediaRecorder.isTypeSupported(mime)) {
			console.log("Will use", mime, "for recording voice message")
			return mime
		}
	}
	return undefined
}

const VoiceRecorder = ({ onFinish }: VoiceRecorderProps) => {
	const [recording, setRecording] = useState<boolean>(false)
	const [duration, setDuration] = useState(0)
	const recorder = useRef<MediaRecorder>(null)
	const blobs = useRef<BlobPart[]>([])
	const closeModal = use(ModalCloseContext)
	const startPauseResumeRecord = () => {
		if (!recorder.current) {
			navigator.mediaDevices.getUserMedia({ audio: true }).then(actuallyStartRecord, err => {
				console.error("Failed to get user media for voice recording:", err)
				window.alert(`Failed to get microphone access: ${err}`)
				cancelRecord()
			})
		} else if (recording) {
			recorder.current.pause()
		} else {
			recorder.current.resume()
		}
	}
	const actuallyStartRecord = (stream: MediaStream) => {
		if (recorder.current) {
			return
		}
		const rec = new MediaRecorder(stream, { mimeType: chooseMime() })
		const b: BlobPart[] = []
		let recStart = 0
		let durationAtStart = 0
		let measureInterval: ReturnType<typeof setInterval> | null = null
		rec.onstart = evt => {
			setRecording(true)
			recStart = evt.timeStamp
			measureInterval = setInterval(() => {
				setDuration(durationAtStart + performance.now() - recStart)
			}, 100)
		}
		const onPause = (evt: Event) => {
			setRecording(false)
			durationAtStart += evt.timeStamp - recStart
			if (measureInterval) {
				clearInterval(measureInterval)
				measureInterval = null
			}
		}
		rec.onpause = onPause
		rec.onresume = rec.onstart
		rec.onstop = evt => {
			onPause(evt)
			actuallyFinishRecord()
		}
		rec.ondataavailable = evt => b.push(evt.data)
		blobs.current = b
		recorder.current = rec
		rec.start()
	}
	const actuallyFinishRecord = () => {
		if (!recorder.current) {
			return
		}
		onFinish(new File(blobs.current, "Voice message.ogg", { type: recorder.current.mimeType }))
	}
	const finishRecord = () => recorder.current?.stop()
	const cancelRecord = () => {
		if (duration > 0 && !window.confirm("Discard recording?")) {
			return
		}
		const rec = recorder.current
		recorder.current = null
		rec?.stop()
		closeModal()
	}
	useEffect(() => () => {
		const rec = recorder.current
		recorder.current = null
		rec?.stop()
	}, [])
	return <>
		<div className="progress">
			<div className="text">
				Recorded for {(duration / 1000).toFixed(1)}s
			</div>
			<div className="bar" style={{ width: `${duration / 600}%` }}/>
		</div>
		<div className="buttons">
			<button className="cancel" onClick={cancelRecord}><DeleteIcon /> Cancel</button>
			<button onClick={startPauseResumeRecord}>
				{recording ? <PauseIcon /> : <PlayIcon />}
				{!recorder.current ? "Start" : recording ? "Pause" : "Resume"}
			</button>
			<button onClick={finishRecord} disabled={!recorder.current || duration === 0}><StopIcon /> Finish</button>
		</div>
	</>
}

export default VoiceRecorder
