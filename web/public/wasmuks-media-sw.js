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
self.addEventListener("install", () => {
	self.skipWaiting()
})
self.addEventListener("activate", (event) => {
	event.waitUntil(clients.claim())
})

const mediaCache = caches.open("wasmuks-media-v1")
const bc = new BroadcastChannel("wasmuks-media-download")
const mediaPromises = new Map()

self.addEventListener("fetch", (evt) => {
	const url = new URL(evt.request.url)
	if (url.origin === self.location.origin && url.pathname.includes("_gomuks/media/")) {
		evt.respondWith(serveFromCache(evt.request).catch(err => {
			console.error("Error serving media from cache", url, err)
			return new Response("Error serving media", {status: 500})
		}))
	}
})

bc.addEventListener("message", evt => {
	if (evt.data.type === "response") {
		const waiter = mediaPromises.get(evt.data.url)
		if (waiter) {
			waiter.resolve()
			mediaPromises.delete(evt.data.url)
		}
	}
})

async function requestAndWaitForMedia(url) {
	if (mediaPromises.has(url)) {
		return mediaPromises.get(url).promise
	}
	let resolve
	const promise = new Promise(innerResolve => {
		resolve = innerResolve
	})
	mediaPromises.set(url, {resolve, promise})
	try {
		bc.postMessage({type: "request", url})
	} catch (err) {
		throw new Error("PostMessage failed")
	}
	return promise
}

async function serveFromCache(request) {
	const cache = await mediaCache
	let hit = await cache.match(request, {ignoreSearch: true})
	if (!hit) {
		await requestAndWaitForMedia(request.url)
		hit = await cache.match(request, {ignoreSearch: true})
		if (!hit) {
			console.log("Cache entry not found after request for", request.url)
			return new Response("Cache entry not found", {status: 404})
		} else {
			console.log("Found cache entry after request for", request.url)
		}
	} else {
		console.log("Cache hit for", request.url)
	}
	return hit
}
