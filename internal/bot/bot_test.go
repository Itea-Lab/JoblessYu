package bot

import (
	"fmt"
	"testing"
	"time"
)

func TestTrimCachesLockedCapsEntries(t *testing.T) {
	b := &Bot{
		jobsCache: make(map[string]cachedJobs),
		uiState:   make(map[string]cachedCriteria),
	}

	for i := 0; i < jobsCacheMaxEntries+10; i++ {
		b.jobsCache[fmt.Sprintf("job-%03d", i)] = cachedJobs{insertedAt: time.Unix(int64(i), 0)}
	}
	for i := 0; i < uiStateMaxEntries+10; i++ {
		b.uiState[fmt.Sprintf("state-%03d", i)] = cachedCriteria{insertedAt: time.Unix(int64(i), 0)}
	}

	b.cacheLock.Lock()
	b.trimCachesLocked()
	b.cacheLock.Unlock()

	if got := len(b.jobsCache); got != jobsCacheMaxEntries {
		t.Fatalf("jobs cache length = %d, want %d", got, jobsCacheMaxEntries)
	}
	if got := len(b.uiState); got != uiStateMaxEntries {
		t.Fatalf("ui state length = %d, want %d", got, uiStateMaxEntries)
	}
	if _, ok := b.jobsCache["job-000"]; ok {
		t.Error("oldest jobs cache entry was not evicted")
	}
	if _, ok := b.uiState["state-000"]; ok {
		t.Error("oldest UI state entry was not evicted")
	}
}
