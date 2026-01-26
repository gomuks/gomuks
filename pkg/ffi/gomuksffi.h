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

typedef struct {
	const uint8_t *base;
	size_t length;
} GomuksBorrowedBuffer;

typedef struct {
	uint8_t *base;
	size_t length;
} GomuksOwnedBuffer;

typedef struct {
	GomuksOwnedBuffer buf;
	const char* command;
} GomuksResponse;

typedef uintptr_t GomuksHandle;
typedef void (*EventCallback)(const char *command, int64_t request_id, GomuksOwnedBuffer data);

GomuksHandle GomuksInit(void);
int GomuksStart(GomuksHandle handle, EventCallback callback);
void GomuksDestroy(GomuksHandle handle);
GomuksResponse GomuksSubmitCommand(GomuksHandle handle, char* command, GomuksBorrowedBuffer data);
void GomuksFreeBuffer(GomuksOwnedBuffer buf);

#ifdef __cplusplus
}
#endif
