#ifndef _PATCH_H_
#define _PATCH_H_

#include <dlfcn.h>
#include <stdint.h>
#include <sys/user.h>
#include <sys/mman.h>
#include <string.h>
#include <stdlib.h>
#include <stdio.h>
#include <unistd.h>

#ifndef PAGE_SIZE
#define PAGE_SIZE (size_t)(sysconf(_SC_PAGESIZE))
#endif

#ifndef PAGE_MASK
#define PAGE_MASK (PAGE_SIZE - 1)
#endif

struct node {
    void *addr; // 函数入口地址
    unsigned char buf[12]; // 现场保存
};

struct node g_patched_list[100]; // 目前最多允许替换100个函数
int g_patched_count=0;

// 将旧函数替换成新函数
int patch(const char *old_func_name, void *new_func)
{
    if (new_func == NULL)
        return -1;
    if (g_patched_count >= sizeof(g_patched_list)/sizeof(g_patched_list[0]))
        return -99;

    void *old_func = NULL;
    if (strlen(old_func_name) > 2 && old_func_name[0]=='0' && old_func_name[1] == 'x')
        old_func = (void*)strtoul(old_func_name, NULL, 16); // 把参数old_func_name所指向的字符串转换为16进制长整数
    else
        /*
             If	dlsym()	is called with the special handle NULL,	it is interpreted as a
             reference to the executable or shared object from which the call is being
             made.  Thus a shared object can reference its own symbols.
        */
        old_func = dlsym(NULL, old_func_name);

    if (old_func == NULL)
        return -2;

    if (old_func == new_func)
        return -3;

    unsigned char buf[12] = {0x48, 0xba, 0, 0, 0, 0, 0, 0, 0, 0, 0xff, 0xe2};
    memcpy(&buf[2], &new_func, sizeof(void*));
    void* page_align = (void*)((uintptr_t)old_func & PAGE_MASK);
    int pages = ((char*)old_func + 12 > (char*)page_align + PAGE_SIZE) ? 2 : 1;
    mprotect(page_align, PAGE_SIZE * pages, PROT_READ|PROT_WRITE|PROT_EXEC);

    struct node *pnode = &g_patched_list[g_patched_count];
    memcpy(pnode->buf, old_func, sizeof(buf));
    pnode->addr = old_func;
    memcpy(old_func, buf, sizeof(buf));

    g_patched_count++;
    return 0;
}
// 还原
void restore()
{
    int i;
    for (i = g_patched_count - 1; i >= 0; i--)
    {
        struct node *pnode = &g_patched_list[i];
        memcpy(pnode->addr, pnode->buf, sizeof(pnode->buf));
    }
    g_patched_count = 0;
}
#endif