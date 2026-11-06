/* DU-04 isolated reproduction.
 * Replicates the exact control flow of procRicSubscriptionRequest()
 * (o-du-l2 src/du_app/du_e2ap_msg_hdl.c):
 *   - line 3520: reject when numPendingSubsRsp >= MAX_PENDING_SUBSCRIPTION_RSP
 *   - line 3607: error path unconditionally does
 *       memset(&ranFuncDb->pendingSubsRspInfo[ranFuncDb->numPendingSubsRsp], ...)
 * With 5 pending responses and a 6th request arriving, the memset writes
 * pendingSubsRspInfo[5] -- one full element past the 5-element array.
 */
#include <stdio.h>
#include <string.h>
#include <stdlib.h>
#include <stdint.h>

#define MAX_PENDING_SUBSCRIPTION_RSP 5   /* du_e2ap_mgr.h:39 */
#define ROK  0
#define RFAILED 1

typedef struct { uint32_t requestId; uint32_t ranFuncId; char pad[56]; } PendingSubsRspInfo;

/* Mirrors RanFunction (du_e2ap_mgr.h:445): array of exactly MAX elements,
 * followed in memory by other struct members that the OOB memset clobbers. */
typedef struct {
   int numPendingSubsRsp;
   PendingSubsRspInfo pendingSubsRspInfo[MAX_PENDING_SUBSCRIPTION_RSP];
   int subscriptionListAlive;   /* sentinel: adjacent production state */
} RanFunction;

/* Replicates the shipped check + error-path memset verbatim. */
static int procRicSubsReq(RanFunction *ranFuncDb)
{
   /* 3520: slots full -> failure, break out (no index consumed) */
   if (ranFuncDb->numPendingSubsRsp >= MAX_PENDING_SUBSCRIPTION_RSP)
      goto failure;
   return ROK;

failure:
   /* 3602-3609: error path, executed unconditionally on failure */
   memset(&ranFuncDb->pendingSubsRspInfo[ranFuncDb->numPendingSubsRsp],
          0, sizeof(PendingSubsRspInfo));
   return RFAILED;
}

int main(void)
{
   RanFunction *db = calloc(1, sizeof(RanFunction));
   db->numPendingSubsRsp = MAX_PENDING_SUBSCRIPTION_RSP; /* 5 slots already pending */
   db->subscriptionListAlive = 0x41424344;               /* 'ABCD' sentinel */

   printf("before: numPendingSubsRsp=%d sentinel=0x%08X array_end=%p sentinel=%p\n",
          db->numPendingSubsRsp, db->subscriptionListAlive,
          (void*)&db->pendingSubsRspInfo[5], (void*)&db->subscriptionListAlive);

   int ret = procRicSubsReq(db);   /* 6th subscription request arrives */

   printf("after : ret=RFAILED, sentinel=0x%08X\n", db->subscriptionListAlive);
   if ((void*)&db->pendingSubsRspInfo[5] == (void*)&db->subscriptionListAlive)
      printf("layout check: memset target IS the adjacent member address\n");
   if (db->subscriptionListAlive == 0)
      printf("OBSERVED DAMAGE: adjacent struct member zeroed by the OOB memset\n");
   free(db);
   return 0;
}
