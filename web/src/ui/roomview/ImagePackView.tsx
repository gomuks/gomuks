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
import { DndContext, DragEndEvent, KeyboardSensor, PointerSensor, useSensor, useSensors } from "@dnd-kit/core"
import { SortableContext, arrayMove, sortableKeyboardCoordinates, useSortable } from "@dnd-kit/sortable"
import { CSS } from "@dnd-kit/utilities"
import React, { use, useState } from "react"
import { getMediaURL } from "@/api/media.ts"
import { useRoomImagePacks } from "@/api/statestore"
import {
	ImagePack, ImagePackEntry, ImagePackUsage, MediaEncodingOptions, MediaMessageEventContent,
	stringToRoomStateGUID,
} from "@/api/types"
import { ensureString, ensureStringArray } from "@/util/validation.ts"
import ClientContext from "../ClientContext.ts"
import { LightboxContext, ModalCloseContext, ModalContext, NestableModalContext, modals } from "../modal"
import { useRoomContext } from "./roomcontext.ts"
import FallbackPackIcon from "@/icons/category.svg?react"
import StickerAddIcon from "@/icons/sticker-add.svg?react"
import "./ImagePackView.css"

const ImagePackView = () => {
	const client = use(ClientContext)!
	const roomCtx = useRoomContext()
	const packs = useRoomImagePacks(roomCtx.store)
	const [selectedPackID, setSelectedPackID] = useState<string | null>(null)
	const selectedPack = selectedPackID ? packs[selectedPackID] : null
	const onWheel = (evt: React.WheelEvent) => {
		const chooser = evt.currentTarget
		if (evt.deltaY === 0 || evt.deltaX !== 0) {
			return
		}
		evt.preventDefault()
		// noinspection JSSuspiciousNameCombination
		chooser.scrollLeft += evt.deltaY
	}
	const createPack = () => {
		const packID = window.prompt("Enter pack ID")
		if (!packID) {
			return
		} else if (packs[packID]) {
			window.alert("A pack with that ID already exists.")
			return
		}
		const emptyPack: ImagePack = {
			pack: {
				usage: ["sticker", "emoticon"],
				display_name: packID,
			},
			images: {},
		}
		client.rpc.setState(roomCtx.store.roomID, "im.ponies.room_emotes", packID, emptyPack)
	}
	return <div className="image-pack-view">
		<div className="image-pack-chooser" onWheel={onWheel}>
			{Object.values(packs).map(pack => <button
				className={selectedPackID === pack.id ? "selected" : ""}
				key={pack.id}
				data-pack-id={pack.id}
				onClick={() => setSelectedPackID(pack.id)}
			>
				{pack.icon ? <img src={getMediaURL(pack.icon)} alt=""/> : <FallbackPackIcon/>}
				<div className="name">{pack.name}</div>
			</button>)}
			<button className="new-pack" onClick={createPack}>
				<StickerAddIcon />
				<div className="name">Create pack</div>
			</button>
		</div>
		{selectedPack && <ImagePackEditor key={selectedPackID} id={selectedPack.id} pack={selectedPack.source} />}
	</div>
}

interface ImagePackEditorProps {
	pack: ImagePack
	id: string
}

type ImagePackEntryWithID = ImagePackEntry & { id: string }

interface ImagePackItemProps {
	item: ImagePackEntryWithID
	openEditor: (item: ImagePackEntryWithID) => void
}

interface ImagePackItemEditorProps {
	item: ImagePackEntryWithID | null
	save: (old: ImagePackEntryWithID | null, updated: ImagePackEntryWithID | null) => void
	defaultUsages: Set<string> | null
}

const knownUsages = ["emoticon", "sticker"]

const ImagePackEditor = ({ id, pack }: ImagePackEditorProps) => {
	const [packName, setPackName] = useState(ensureString(pack.pack.display_name))
	const [packAvatar, setPackAvatar] = useState(ensureString(pack.pack.avatar_url))
	const [usages, setUsages] = useState<Set<string> | null>(() =>
		pack.pack.usage ? new Set(ensureStringArray(pack.pack.usage)) : null)
	const [images, setImages] = useState<ImagePackEntryWithID[]>(() =>
		Object.entries(pack.images)
			.map(([id, image]) => ({ id, ...image }))
			.toSorted((a, b) =>
				(a["fi.mau.msc4389.order"] ?? 0) - (b["fi.mau.msc4389.order"] ?? 0)))
	const [saving, setSaving] = useState(false)
	const client = use(ClientContext)!
	const openModal = use(NestableModalContext)
	const sensors = useSensors(
		useSensor(PointerSensor, {
			activationConstraint: {
				delay: 200,
				tolerance: 5,
			},
		}),
		useSensor(KeyboardSensor, {
			coordinateGetter: sortableKeyboardCoordinates,
		}),
	)
	const onDragEnd = (evt: DragEndEvent) => {
		const { active, over } = evt
		if (active.id !== over?.id) {
			setImages((items) => {
				const oldIndex = items.findIndex(i => i.id === active.id)
				const newIndex = items.findIndex(i => i.id === over?.id)
				return arrayMove(items, oldIndex, newIndex)
			})
		}
	}
	const saveImage = (oldItem: ImagePackEntryWithID | null, newItem: ImagePackEntryWithID | null) => {
		if (newItem === null) {
			if (oldItem !== null) {
				setImages(images.filter(img => img.id !== oldItem.id))
			}
		} else if (oldItem !== null) {
			setImages(images.map(img => img.id === oldItem.id ? newItem : img))
		} else {
			setImages([...images, newItem])
		}
	}
	const openEditor = (item: ImagePackEntryWithID | null) => {
		openModal({
			boxed: true,
			dimmed: true,
			// noDismiss: true,
			boxClass: "image-pack-item-editor-wrapper",
			innerBoxClass: "image-pack-item-editor",
			content: <ImagePackItemEditor item={item} save={saveImage} defaultUsages={usages} />,
		})
	}
	const guid = stringToRoomStateGUID(id)
	const savePack = () => {
		if (!guid || saving) {
			return
		}
		setSaving(true)
		client.rpc.setState(guid.room_id, guid.type, guid.state_key, {
			...pack,
			pack: {
				...pack.pack,
				display_name: packName,
				avatar_url: packAvatar,
				usage: usages ? Array.from(usages) as ImagePackUsage[] : [],
			},
			images: Object.fromEntries(images.map((img, i) =>
				[img.id, { ...img, "fi.mau.msc4389.order": i+1, id: undefined }])),
		})
			.then(() => {}, err => window.alert(`Failed to save image pack: ${err.message}`))
			.finally(() => setSaving(false))
	}
	return <div className="image-pack-editor">
		<div className="input-fields">
			<label htmlFor="image-pack-editor-id">Pack ID:</label>
			<input
				id="image-pack-editor-id"
				className="id"
				type="text"
				value={guid?.state_key}
				disabled
			/>
			<label htmlFor="image-pack-editor-name">Pack name:</label>
			<input
				id="image-pack-editor-name"
				className="name"
				type="text"
				value={packName}
				onChange={e => setPackName(e.target.value)}
				placeholder="Pack name"
			/>
			<label htmlFor="image-pack-editor-avatar">Pack avatar:</label>
			<input
				id="image-pack-editor-avatar"
				className="avatar"
				type="text"
				value={packAvatar}
				onChange={e => setPackAvatar(e.target.value)}
				placeholder="Pack avatar (defaults to first image)"
			/>
			{renderUsages(usages, setUsages)}
		</div>
		<div className="images">
			<DndContext sensors={sensors} onDragEnd={onDragEnd}>
				<SortableContext items={images}>
					{images.map((image) => <ImagePackItem key={image.id} item={image} openEditor={openEditor} />)}
				</SortableContext>
			</DndContext>
			<div className="item">
				<button onClick={() => openEditor(null)} title="Add new image"><StickerAddIcon /></button>
			</div>
		</div>
		<button className="global-save" disabled={saving} onClick={savePack}>
			{saving ? "Saving..." : "Save changes"}
		</button>
	</div>
}

const ImagePackItem = ({ item, openEditor }: ImagePackItemProps) => {
	const { attributes, listeners, setNodeRef, transform, transition, active } = useSortable({
		id: item.id,
	})

	const style = {
		transform: CSS.Transform.toString(transform),
		transition,
	}

	return <div
		className={`item ${active?.id === item.id ? "dragging" : ""}`}
		style={style}
		ref={setNodeRef}
		{...attributes}
		{...listeners}
		onClick={() => openEditor(item)}
	>
		<img src={getMediaURL(item.url)} alt="" />
	</div>
}

const ImagePackItemEditor = ({ item, save, defaultUsages }: ImagePackItemEditorProps) => {
	const [url, setURL] = useState(ensureString(item?.url))
	const [body, setBody] = useState(ensureString(item?.body))
	const [id, setID] = useState(ensureString(item?.id))
	const [info, setInfo] = useState(item?.info)
	const [usages, setUsages] = useState(() => item?.usage ? new Set(ensureStringArray(item.usage)) : null)
	const [uploading, setUploading] = useState(false)
	const openLightbox = use(LightboxContext)!
	const closeModal = use(ModalCloseContext)
	const openModal = use(ModalContext)
	const client = use(ClientContext)!
	const newItem = {
		...item,
		id,
		url,
		body,
		info,
		usage: usages ? Array.from(usages) as ImagePackUsage[] : undefined,
	}
	const doUploadFile = (file: Blob, filename: string, encodingOpts?: MediaEncodingOptions) => {
		setUploading(true)
		const unsetUploading = () => setUploading(false)
		const uploadComplete = (media: MediaMessageEventContent) => {
			setURL(media.url ?? "")
			setInfo(media.info)
		}
		const uploadFailed = (err: Error) => window.alert(`Failed to upload file: ${err.message}`)
		if (client.rpc.rpcMediaUpload) {
			client.rpc.uploadMedia(file, filename, false)
				.then(uploadComplete, uploadFailed)
				.finally(unsetUploading)
			return
		}
		const params = new URLSearchParams([
			["encrypt", "false"],
			["progress", "false"],
			["filename", filename],
			...Object.entries(encodingOpts ?? {})
				.filter(([key, value]) => !key.startsWith("_") && !!value)
				.map(([key, value]) => [key, value.toString()]),
		])
		fetch(`_gomuks/upload?${params.toString()}`, {
			method: "POST",
			body: file,
			headers: {
				"Content-Type": file.type,
			},
		})
			.then(async resp => uploadComplete(await resp.json()), uploadFailed)
			.finally(unsetUploading)
	}
	const openFileUpload = (evt: React.ChangeEvent<HTMLInputElement>) => {
		const file = evt.target.files?.[0]
		if (!file) {
			return
		}
		openModal(modals.mediaUpload(file, doUploadFile))
	}
	const doSave = () => {
		save(item, newItem)
		closeModal()
	}
	const doDelete = () => {
		save(item, null)
		closeModal()
	}
	return <>
		{url && <img src={getMediaURL(url)} alt="Image for the entry" onClick={openLightbox} />}
		<div className="input-fields">
			<label htmlFor="image-editor-upload">Upload image:</label>
			<input disabled={uploading} id="image-editor-upload" type="file" value="" onChange={openFileUpload} />
			<label htmlFor="image-editor-id">Shortcode:</label>
			<input
				id="image-editor-id"
				type="text"
				value={id}
				onChange={evt => setID(evt.target.value)}
				placeholder="The :shortcode: for this image (excluding colons)"
				required
			/>
			<label htmlFor="image-editor-description">Description:</label>
			<input
				id="image-editor-description"
				type="text"
				value={body}
				onChange={evt => setBody(evt.target.value)}
				placeholder="A textual representation or associated description of the image"
			/>
			{renderUsages(usages, setUsages, defaultUsages)}
		</div>
		<details>
			<summary>Raw image data</summary>
			<pre>
				{JSON.stringify(newItem, null, "  ")}
			</pre>
		</details>
		<div className="buttons">
			<div className="left-buttons">
				<button onClick={() => closeModal()} className="dangerous">Discard</button>
				{item && <button onClick={doDelete} className="dangerous">Delete</button>}
			</div>
			<button onClick={doSave} disabled={!url || !id}>Save</button>
		</div>
	</>
}

const renderUsages = (
	usages: Set<string> | null,
	setUsages: (newVal: Set<string> | null) => void,
	packDefault?: Set<string> | null,
) => {
	const realUsages = usages ?? packDefault
	return <>
		<div className="usage-label">Use as:</div>
		<div className="usage-options">
			{packDefault !== undefined ? <label>
				<input
					type="checkbox"
					checked={usages === null}
					onChange={e => {
						setUsages(e.target.checked ? null : (packDefault ?? new Set(knownUsages)))
					}}
				/>
				pack default
			</label> : null}
			{knownUsages.map(usage => <label key={usage}>
				<input
					type="checkbox"
					checked={realUsages ? realUsages.has(usage) : true}
					disabled={usages === null && packDefault !== undefined}
					onChange={e => {
						if (e.target.checked) {
							setUsages(new Set(usages ?? []).add(usage))
						} else {
							const newUsages = new Set(realUsages ?? knownUsages)
							newUsages.delete(usage)
							setUsages(newUsages)
						}
					}}
				/>
				{usage === "emoticon" ? "emoji" : usage}{!packDefault && " pack"}
			</label>)}
		</div>
	</>
}

export default ImagePackView
