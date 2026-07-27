// Copyright (c) 2026 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

#include <stddef.h>
#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

// GomuksBorrowedBuffer is a byte array that is "borrowed" from the creator to recipient.
// The recipient MUST NOT try to free it.
typedef struct {
	const uint8_t *base;
	size_t length;
} GomuksBorrowedBuffer;

// GomuksOwnedBuffer is a byte array whose ownership is transferred to the recipient.
// The recipient MUST free it using GomuksFreeBuffer (or a manual free() call) when done.
typedef struct {
	uint8_t *base;
	size_t length;
} GomuksOwnedBuffer;

// GomuksResponse wraps an owned buffer (which must be freed by the recipient) and a command string.
// The command string is valid forever and does not need to be freed, as gomuks will reuse the same string pointers.
typedef struct {
	GomuksOwnedBuffer buf;
	const char* command;
} GomuksResponse;

// GomuksHandle is an internal pointer to a gomuks instance.
// The actual value has no meaning outside of the Gomuks* functions (it is not a real memory pointer).
typedef uintptr_t GomuksHandle;

typedef void (*EventCallback)(const char *command, int64_t request_id, GomuksOwnedBuffer data);
typedef void (*ProgressCallback)(double progress);
typedef void (*StreamCallback)(GomuksBorrowedBuffer data);

// GomuksInit initializes a new gomuks instance and returns a handle.
// The handle can't be used before GomuksStart is called nor after GomuksDestroy is called.
// If root is non-NULL, it is used as the root directory for all gomuks data
// (config, cache, data, logs), bypassing environment variable lookups.
// Pass NULL to use the default directory resolution.
GomuksHandle GomuksInit(char* root);
// GomuksStart starts the gomuks instance and Matrix sync loop.
// If the return value is non-zero, the call failed and the handle isn't ready for use.
// The callback will be called to provide the initial room list as well as any new events.
int GomuksStart(GomuksHandle handle, EventCallback callback);
// GomuksDestroy stops the given gomuks instance and removes references to it.
void GomuksDestroy(GomuksHandle handle);

// GomuksSubmitCommand sends a command to gomuks and returns the response.
// The caller is responsible for memory management of the command string.
// A good approach is creating one string for each command and reusing them forever.
GomuksResponse GomuksSubmitCommand(GomuksHandle handle, char* command, GomuksBorrowedBuffer data);

// GomuksUploadMediaPath is equivalent to GomuksSubmitCommand with the upload_media command
// with an additional progress callback that will be called to report upload progress as a float64 between 0 and 100.
// The JSON in the params buffer must contain a "path" field with the file path to upload.
// The callback is optional, though without it this is equivalent to the upload_media command.
GomuksResponse GomuksUploadMediaPath(GomuksHandle handle, GomuksBorrowedBuffer params, ProgressCallback cb);
// GomuksMediaUploadBytes is an alternate media upload method which takes raw bytes instead of a file path.
// The callback is optional.
GomuksResponse GomuksUploadMediaBytes(GomuksHandle handle, GomuksBorrowedBuffer params, GomuksBorrowedBuffer mediaBytes, ProgressCallback cb);
// GomuksMediaDownloadPath is equivalent to GomuksSubmitCommand with the download_media command
// with an additional stream callback. When downloading an unencrypted file that isn't cached locally,
// the stream callback will receive chunks of the file as they are downloaded, followed by a final chunk
// with length 0. The callback is not used for encrypted files nor if the file is already downloaded.
// The callback is optional, though without it this is equivalent to the download_media command.
GomuksResponse GomuksDownloadMediaPath(GomuksHandle handle, GomuksBorrowedBuffer params, StreamCallback cb);

// GomuksFreeBuffer frees an owned buffer returned from gomuks.
void GomuksFreeBuffer(GomuksOwnedBuffer buf);

#ifdef __cplusplus
}
#endif
