#include "../gomuksffi.h"
#include <inttypes.h>
#include <signal.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

static volatile sig_atomic_t stop_requested = 0;

static void handle_signal(int sig) {
    (void)sig;
    stop_requested = 1;
}

static void event_callback(const char *command, int64_t request_id, GomuksOwnedBuffer data) {
    printf("[event] command=%s request_id=%" PRId64 " data=", command ? command : "(null)", request_id);
    if (data.base && data.length > 0) {
        fwrite(data.base, 1, data.length, stdout);
    }
    putchar('\n');
    fflush(stdout);
    GomuksFreeBuffer(data);
}

int main(void) {
    struct sigaction sa = {0};
    sa.sa_handler = handle_signal;
    sigaction(SIGINT, &sa, NULL);
    sigaction(SIGTERM, &sa, NULL);

    GomuksHandle handle = GomuksInit();
    if (handle == 0) {
        fprintf(stderr, "Failed to initialize gomuks.\n");
        return 1;
    }

    GomuksStart(handle, event_callback);

    char *line = NULL;
    size_t cap = 0;
    while (!stop_requested) {
        ssize_t nread = getline(&line, &cap, stdin);
        if (nread == -1) {
            break;
        }

        while (nread > 0 && (line[nread - 1] == '\n' || line[nread - 1] == '\r')) {
            line[--nread] = '\0';
        }
        if (nread == 0) {
            continue;
        }

        char *space = strchr(line, ' ');
        if (!space) {
            fprintf(stderr, "Expected '<command> <json>'.\n");
            continue;
        }

        *space = '\0';
        char *command = line;
        char *json = space + 1;
        if (*json == '\0') {
            fprintf(stderr, "Missing JSON payload.\n");
            continue;
        }

        GomuksBorrowedBuffer buf = {
            .base = (const uint8_t *)json,
            .length = strlen(json),
        };

        GomuksResponse resp = GomuksSubmitCommand(handle, command, buf);
        printf("[response] command=%s data=", resp.command ? resp.command : "(null)");
        if (resp.buf.base && resp.buf.length > 0) {
            fwrite(resp.buf.base, 1, resp.buf.length, stdout);
        }
        putchar('\n');
        fflush(stdout);
        GomuksFreeBuffer(resp.buf);
    }

    free(line);
    GomuksDestroy(handle);
    return 0;
}
