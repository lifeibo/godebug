#include <runtime.h>

void ·GetGoId(int64 ret) {
    ret = g->goid;
    USED(&ret);
}

void ·GStack(Slice b, int64 goid, int32 n) {
    G* gp = 0;
    uintptr i;
    int32 found = 0;
    uintptr pc, sp;

    n = 0;

    sp = runtime·getcallersp(&b);
    pc = (uintptr)runtime·getcallerpc(&b);

    runtime·semacquire(&runtime·worldsema, false);
    m->gcing = 1;
    runtime·stoptheworld();

    if(b.len > 0) {
        for(i = 0; i < runtime·allglen; i++) {
            gp = runtime·allg[i];
            if(gp->status == Gdead)
                continue;

            if (gp->goid == goid) {
                found = 1;
                break;
            }
        }
        if (found) {
            g->writebuf = (byte*)b.array;
            g->writenbuf = b.len;
            runtime·goroutineheader(gp);

            if(gp->status == Grunning && gp != g) {
                runtime·printf("\tgoroutine running on other thread; stack unavailable\n");
                runtime·printcreatedby(gp);
            } else if (gp != g) {
                runtime·traceback(~(uintptr)0, ~(uintptr)0, 0, gp);
            } else {
                runtime·traceback(pc, sp, 0, gp);
            }

            n = b.len - g->writenbuf;
            g->writebuf = nil;
            g->writenbuf = 0;
        }
    }

    m->gcing = 0;
    runtime·semrelease(&runtime·worldsema);
    runtime·starttheworld();

    USED(&n);
}
