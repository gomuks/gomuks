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

typedef uintptr_t GomuksHandle;
typedef void (*EventCallback)(const char *command, int64_t request_id, GomuksOwnedBuffer data);
typedef void (*ProgressCallback)(double progress);

// GomuksInit initializes a new gomuks instance and returns a handle.
// The handle can't be used before GomuksStart is called nor after GomuksDestroy is called.
GomuksHandle GomuksInit(void);
// GomuksStart starts the gomuks instance and Matrix sync loop.
// If the return value is non-zero, the call failed and the handle isn't ready for use.
// The callback will be called to provide the initial room list as well as any new events.
int GomuksStart(GomuksHandle handle, EventCallback callback);
// GomuksDestroy stops the given gomuks instance and removes references to it.
void GomuksDestroy(GomuksHandle handle);
// GomuksSubmitCommand sends a command to gomuks and returns the response.
GomuksResponse GomuksSubmitCommand(GomuksHandle handle, char* command, GomuksBorrowedBuffer data);
// GomuksUploadMediaPath is equivalent to GomuksSubmitCommand with the upload_media command
// with an additional progress callback that will be called to report upload progress as a float64 between 0 and 100.
// The JSON in the params buffer must contain a "path" field with the file path to upload.
GomuksResponse GomuksUploadMediaPath(GomuksHandle handle, GomuksBorrowedBuffer params, ProgressCallback cb);
// GomuksMediaUploadBytes is an alternate media upload method which takes raw bytes instead of a file path.
GomuksResponse GomuksUploadMediaBytes(GomuksHandle handle, GomuksBorrowedBuffer params, GomuksBorrowedBuffer mediaBytes, ProgressCallback cb);
// GomuksFreeBuffer frees an owned buffer returned from gomuks.
void GomuksFreeBuffer(GomuksOwnedBuffer buf);

#ifdef __cplusplus
}
#endif
