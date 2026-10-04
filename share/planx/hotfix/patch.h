#ifndef _PATCH_H_
#define _PATCH_H_

#include <stdio.h>
#include <unistd.h>

extern int patch(const char *old_func_name, void *new_func);
extern void restore();


#endif
