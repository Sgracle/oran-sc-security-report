/* DU-03 isolated reproduction.
 * Replicates FreeRicSubscriptionFailure() (o-du-l2 src/du_app/du_e2ap_msg_hdl.c:3630-3652)
 * verbatim, with the shipped DU_ALLOC/DU_FREE macros (du_utils.h:67-92) backed by a
 * counting fake pool instead of SGetSBuf/SPutSBuf.
 * The function is named Free but all four frees use DU_ALLOC, which allocates a new
 * buffer and overwrites the pointer: the original object becomes unreachable and the
 * new one is never freed => leak on every failure response build.
 */
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <stdint.h>

#define ROK 0
#define NULLP 0

/* ---- fake pool (stands in for SGetSBuf/SPutSBuf) ---- */
static long alloc_count = 0, free_count = 0;
typedef struct { void *p; size_t sz; } Rec;
static Rec recs[100000]; static int nrecs = 0;

static int SGetSBuf(void **p, size_t sz) { *p = calloc(1, sz); alloc_count++; recs[nrecs++] = (Rec){*p, sz}; return *p ? ROK : 1; }
static int SPutSBuf(void *p, size_t sz)  { free(p); free_count++; return ROK; }

/* ---- shipped macros, verbatim from du_utils.h:67-92 ---- */
#define DU_ALLOC(_datPtr, _size)                              {                                                                int _ret;                                                     _ret = SGetSBuf((void **)&_datPtr, _size);                    if(_ret == ROK) { memset(_datPtr, 0, _size); }                else _datPtr = NULLP;                                      }
#define DU_FREE(_datPtr, _size)                               {                                                                if(_datPtr != NULLP)                                          {                                                                SPutSBuf((void *)_datPtr, _size);                             _datPtr = NULLP;                                           }                                                          }

/* ---- minimal shape of the ASN.1 types involved ---- */
typedef struct { int dummy; } RICsubscriptionFailure_IEs_t;
typedef struct { void *unused; } UnsuccessfulOutcomeE2_t;
typedef struct {
   struct { RICsubscriptionFailure_IEs_t **array; int count; size_t size; } list;
} ProtocolIEsContainer_t;
typedef struct { ProtocolIEsContainer_t protocolIEs; } RICsubscriptionFailure_t;
typedef struct { RICsubscriptionFailure_t *unsuccessfulOutcome; } E2AP_PDU_t;

/* ---- production function, verbatim logic from du_e2ap_msg_hdl.c:3636-3652 ---- */
void FreeRicSubscriptionFailure(E2AP_PDU_t *e2apMsg)
{
   uint8_t elemIdx = 0;
   RICsubscriptionFailure_t *ricSubscriptionFailure = NULLP;

   if(e2apMsg)
   {
      if(e2apMsg->unsuccessfulOutcome)
      {
         ricSubscriptionFailure = (RICsubscriptionFailure_t*)e2apMsg->unsuccessfulOutcome;
         if(ricSubscriptionFailure->protocolIEs.list.array)
         {
            for(elemIdx = 0; elemIdx < ricSubscriptionFailure->protocolIEs.list.count; elemIdx++)
            {
               DU_ALLOC(ricSubscriptionFailure->protocolIEs.list.array[elemIdx], sizeof(RICsubscriptionFailure_IEs_t));
            }
            DU_ALLOC(ricSubscriptionFailure->protocolIEs.list.array, ricSubscriptionFailure->protocolIEs.list.size);
         }
         DU_ALLOC(e2apMsg->unsuccessfulOutcome, sizeof(UnsuccessfulOutcomeE2_t));
      }
      DU_ALLOC(e2apMsg, sizeof(E2AP_PDU_t));
   }
}

/* Build one failure-response-shaped message, then run the shipped 'free'. */
static void one_request_cycle(void)
{
   E2AP_PDU_t *e2apMsg = calloc(1, sizeof(E2AP_PDU_t));
   e2apMsg->unsuccessfulOutcome = calloc(1, sizeof(UnsuccessfulOutcomeE2_t));
   RICsubscriptionFailure_t *f = calloc(1, sizeof(RICsubscriptionFailure_t));
   f->protocolIEs.list.count = 3;
   f->protocolIEs.list.size = 3 * sizeof(RICsubscriptionFailure_IEs_t*);
   f->protocolIEs.list.array = calloc(3, sizeof(RICsubscriptionFailure_IEs_t*));
   e2apMsg->unsuccessfulOutcome = (UnsuccessfulOutcomeE2_t*)f;

   FreeRicSubscriptionFailure(e2apMsg);   /* shipped end-of-build cleanup */
}

int main(void)
{
   const int CYCLES = 1000;               /* 1000 rejected subscriptions */
   for (int i = 0; i < CYCLES; i++) one_request_cycle();
   printf("cycles=%d allocs=%ld frees=%ld leaked_allocs=%ld (per cycle: %.1f alloc vs %.1f free)\n",
          CYCLES, alloc_count, free_count, alloc_count - free_count,
          (double)alloc_count / CYCLES, (double)free_count / CYCLES);
   if (alloc_count > free_count)
      printf("LEAK CONFIRMED: every failure response leaks its allocations (DU_ALLOC used where DU_FREE intended)\n");
   return 0;
}
