# Bounded workspace-symbol selection

Short queries already return at most 1,000 symbols. The old path first copied
all matching definitions, filtered out framework stubs, and then ranked the
survivors. The new path copies up to 256 definitions at a time and keeps at
most 1,000 in the selection heap. Filtering and ranking run between index
read locks. The generation check discards a partial result if an index write
interrupts the traversal, and the handler retries once with the existing
full snapshot. Long queries still return the complete matching set.

The ranking test checks exact, prefix, and substring matches in order,
excludes a matching framework stub, and checks which URI survives the cap.
The index test forces a write from the batch callback. It checks that copied
pointers keep their original definitions and that the scan reports an
interruption. Disabling the generation check makes that test accept a mixed
513-definition result and fail.

Five alternating samples per build used `BenchmarkWorkspaceSymbolWithMarshal`
on a 5,000-file, 40,000-definition synthetic index, with Go 1.27.4,
`GOMAXPROCS=2` and a 450 ms benchmark window. Allocation includes JSON encoding.
The baseline was `b99c6b6`.

| Query | Baseline time | Batches time | Baseline bytes/op | Batches bytes/op |
| --- | ---: | ---: | ---: | ---: |
| `m` | 3.421 ms | 3.291 ms | 606,974 | 282,117 |
| `method3` | 2.396 ms | 2.400 ms | 1,416,517 | 1,435,282 |
| `zzz` | 422 ns | 415 ns | 120 | 120 |

A separate instrumented run recorded the longest continuous index read-lock
hold per `m` request across 150 requests. The baseline p50/p95 were 15.375
and 78.5 µs. An initial implementation ranked under the lock and held it for
2.532/3.082 ms. Batching brought that to 0.375/1.917 µs. These measurements
exclude the full-snapshot retry when an index write interrupts traversal.
