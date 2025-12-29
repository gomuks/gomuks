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
import React, { JSX, use, useEffect, useRef, useState } from "react"
import type { MediaEncodingOptions } from "@/api/types"
import { ModalCloseContext } from "@/ui/modal"
import { isMobileDevice } from "@/util/ismobile.ts"
import "./MediaUploadDialog.css"

export type UploadFileFunc = (file: Blob, filename: string, encodingOpts?: MediaEncodingOptions) => void

export interface MediaUploadDialogProps {
	file: File
	blobURL: string
	doUploadFile: UploadFileFunc
	isEncrypted: boolean
	isVoice?: boolean
}

function formatSize(bytes: number): string {
	const units = ["B", "KiB", "MiB", "GiB", "TiB"]
	let unitIndex = 0
	let size = bytes
	while (size >= 1024 && unitIndex < units.length - 1) {
		size /= 1024
		unitIndex++
	}
	return `${unitIndex === 0 ? size : size.toFixed(2)} ${units[unitIndex]}`
}

const imageReencTargets = ["image/webp", "image/jpeg", "image/png", "image/gif"]
const nonEncodableSources = ["image/bmp", "image/tiff", "image/heif", "image/heic"]
const imageReencSources = [...imageReencTargets, ...nonEncodableSources]
const videoReencTargets = ["video/webm", "video/mp4", "image/webp+anim"]
const voiceMimeType = "audio/ogg; codecs=opus"
const voiceReencTargets = [voiceMimeType]
const audioReencTargets = [...voiceReencTargets, "audio/mpeg", "audio/mp4"]

interface dimensions {
	width: number
	height: number
}

const MediaUploadDialog = ({ file, blobURL, doUploadFile, isEncrypted, isVoice }: MediaUploadDialogProps) => {
	const videoRef = useRef<HTMLVideoElement>(null)
	const [name, setName] = useState(file.name)
	const needsVoiceReenc = isVoice && file.type !== voiceMimeType
	const initialReencTarget = nonEncodableSources.includes(file.type)
		? "image/jpeg"
		: needsVoiceReenc
			? voiceMimeType
			: ""
	const [reencTarget, setReencTarget] = useState(initialReencTarget)
	const [jpegQuality, setJPEGQuality] = useState(80)
	const [resizeSlider, setResizeSlider] = useState(100)
	const [origDimensions, setOrigDimensions] = useState<dimensions | null>(null)
	const [noEncrypt, setNoEncrypt] = useState(false)
	const closeModal = use(ModalCloseContext)
	let previewContent: JSX.Element | null = null
	let reencTargets: string[] | null = null
	let resizedWidth: number | undefined = undefined
	let resizedHeight: number | undefined = undefined
	if (origDimensions) {
		resizedWidth = Math.floor(origDimensions.width * (resizeSlider / 100))
		resizedHeight = Math.floor(origDimensions.height * (resizeSlider / 100))
	}
	useEffect(() => {
		if (file.type.startsWith("image/")) {
			createImageBitmap(file).then(res => {
				setOrigDimensions({ width: res.width, height: res.height })
				res.close()
			})
		}
	}, [file, blobURL])
	if (file.type.startsWith("image/")) {
		previewContent = <img src={blobURL} alt={file.name} />
		if (imageReencSources.includes(file.type)) {
			reencTargets = imageReencTargets
		}
	} else if (file.type.startsWith("video/")) {
		const videoMetaLoaded = () => {
			if (videoRef.current) {
				setOrigDimensions({ width: videoRef.current.videoWidth, height: videoRef.current.videoHeight })
			}
		}
		previewContent = <video controls onLoadedMetadata={videoMetaLoaded} ref={videoRef}>
			<source src={blobURL} type={file.type} />
		</video>
		reencTargets = videoReencTargets
	} else if (file.type.startsWith("audio/")) {
		reencTargets = isVoice ? voiceReencTargets : audioReencTargets
		previewContent = <audio controls>
			<source src={blobURL} type={file.type} />
		</audio>
	}
	const submit = (evt: React.FormEvent) => {
		evt.preventDefault()
		doUploadFile(file, name, {
			encode_to: reencTarget || undefined,
			quality: reencTarget === "image/jpeg" ? jpegQuality : undefined,
			resize_width: resizeSlider !== 100 ? resizedWidth : undefined,
			resize_height: resizeSlider !== 100 ? resizedHeight : undefined,
			resize_percent: resizeSlider,
			_no_encrypt: noEncrypt,
			voice_message: isVoice,
		})
		closeModal()
	}
	return <form className="media-upload-modal" onSubmit={submit}>
		<h3>Upload attachment</h3>
		<div className="attachment-preview">{previewContent}</div>
		<div className="attachment-meta">
			<div className="meta-key">Original type</div>
			<div className="meta-value">{file.type}</div>

			<div className="meta-key">Original size</div>
			<div className="meta-value">{formatSize(file.size)}</div>

			<label htmlFor="input-file-name" className="meta-key">File name</label>
			<div className="meta-value">
				<input
					autoFocus={!isMobileDevice}
					type="text"
					value={name}
					id="input-file-name"
					onChange={evt => setName(evt.target.value)}
				/>
			</div>

			<div className="meta-key">{origDimensions ? "Dimensions" : null}</div>
			<div className="meta-value">
				{origDimensions ? `${resizedWidth}×${resizedHeight}` : null}
			</div>

			{reencTargets && <>
				<label htmlFor="select-reenc-type" className="meta-key">Re-encode</label>
				<div className="meta-value meta-value-long">
					<select value={reencTarget} id="select-reenc-type" onChange={evt => {
						setReencTarget(evt.target.value)
						setResizeSlider(100)
					}}>
						{!needsVoiceReenc && <option value="">No re-encoding</option>}
						{reencTargets.map(target => <option key={target} value={target}>{target}</option>)}
					</select>
				</div>

				<label htmlFor="slider-resize" className="meta-key">Resize</label>
				<div className="meta-value meta-value-long">
					<input
						type="range"
						min={1}
						max={100}
						value={resizeSlider}
						id="slider-resize"
						onChange={evt => {
							setResizeSlider(parseInt(evt.target.value))
							if (reencTarget === "") {
								setReencTarget(reencTargets?.includes(file.type) ? file.type : "image/jpeg")
							}
						}}
					/>
					<span>{resizeSlider}%</span>
				</div>
			</>}

			{(reencTarget === "image/jpeg" || reencTarget === "image/webp") && <>
				<label htmlFor="slider-reenc-quality" className="meta-key">Quality</label>
				<div className="meta-value meta-value-long">
					<input
						type="range"
						min={1}
						max={reencTarget === "image/webp" ? 101 : 100}
						id="slider-reenc-quality"
						value={jpegQuality}
						onChange={evt => setJPEGQuality(parseInt(evt.target.value))}
					/>
					<span>{jpegQuality === 101 ? "Lossless" : `${jpegQuality}%`}</span>
				</div>
			</>}

			{isEncrypted && <>
				<label htmlFor="checkbox-no-encrypt" className="meta-key">Don't encrypt</label>
				<div className="meta-value meta-value-long">
					<input
						type="checkbox"
						checked={noEncrypt}
						id="checkbox-no-encrypt"
						onChange={evt => setNoEncrypt(evt.target.checked)}
					/>
				</div>
			</>}
		</div>
		<div className="confirm-buttons">
			<button className="cancel-button" type="button" onClick={closeModal}>Cancel</button>
			<button className="confirm-button" type="submit">Upload</button>
		</div>
	</form>
}

export default MediaUploadDialog
