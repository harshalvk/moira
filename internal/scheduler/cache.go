package scheduler

import (
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
)

// assumeTTL bounds how long an assumed pod counts against a node's capacity
// if we never see informer confirmation. Guards against a bind that failed
// after the API call succeeded server-side reservation but before we got a
// response, or any other case where confirmation never arrives.
const assumeTTL = 30 * time.Second

type assumedPod struct {
	pod       *corev1.Pod
	node      string
	expiresAt time.Time
}

// AssumeCache tracks pods this scheduler has decided to bind but that may
// not yet be visible via a fresh List call or the informer's watch stream.
// This closes the race flagged in ADR 0003: two pods scheduled back-to-back
// must not both see the same "free" capacity.
type AssumeCache struct {
	mu      sync.RWMutex
	assumed map[string]*assumedPod // keyed by namespace/name
}

func NewAssumeCache() *AssumeCache {
	return &AssumeCache{assumed: make(map[string]*assumedPod)}
}

func podKey(pod *corev1.Pod) string {
	return pod.Namespace + "/" + pod.Name
}

// Assume records that pod has been decided for node. Call this immediately
// after a successful bind, before returning control to the watch loop.
func (c *AssumeCache) Assume(pod *corev1.Pod, node string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.assumed[podKey(pod)] = &assumedPod{
		pod:       pod,
		node:      node,
		expiresAt: time.Now().Add(assumeTTL),
	}
}

// Forget removes a pod from the assumed set. Call this once the informer
// confirms the pod's real state reflects the binding (or the pod is deleted),
// so the assumption doesn't outlive its usefulness and isn't double-counted
// once the real pod list already accounts for it.
func (c *AssumeCache) Forget(pod *corev1.Pod) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.assumed, podKey(pod))
}

// PodsForNode returns non-expired assumed pods for a given node. Expired
// entries are lazily evicted on read rather than via a background goroutine —
// simpler, and the cache is small enough that scan-on-read is cheap.
func (c *AssumeCache) PodsForNode(node string) []*corev1.Pod {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	var result []*corev1.Pod
	for key, a := range c.assumed {
		if now.After(a.expiresAt) {
			delete(c.assumed, key)
			continue
		}
		if a.node == node {
			result = append(result, a.pod)
		}
	}
	return result
}
