package util 

/*
#include <stdlib.h>
#include <string.h>
#include <sys/prctl.h>
#include <unistd.h>

// Set short process name (shows up in `top`, `htop`, `ps -o comm=`)
void setProcName(const char* name) {
    prctl(PR_SET_NAME, name, 0, 0, 0);
}

// Overwrite argv memory (shows in `ps aux`, /proc/[pid]/cmdline)
extern char **environ;

void spoofCmdline(const char* fake) {
    size_t len = strlen(fake);
    char *end = NULL;

    // Find end of argv/environ region
    for (int i = 0; environ[i]; i++) {
        if (end == NULL || environ[i] > end) {
            end = environ[i] + strlen(environ[i]);
        }
    }

    // Wipe old argv/env to hide real command line
    memset(environ[0], 0, end - environ[0]);

    // Copy fake name into argv[0]
    strncpy(environ[0], fake, len);
}
*/
import "C"
import (
    "unsafe"
)

func SpoofPsName(shortName, fullCmdline string) {
    cShort := C.CString(shortName)
    C.setProcName(cShort)
    C.free(unsafe.Pointer(cShort))
    cFull := C.CString(fullCmdline)
    C.spoofCmdline(cFull)
    C.free(unsafe.Pointer(cFull))
}
