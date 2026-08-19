package framework

import (
	"log/slog"

	corev1 "k8s.io/api/core/v1"
)

type ScorePluginWithWeight struct {
	Plugin ScorePlugin
	Weight int64
}

// Registry holds the active plugin set. Explicit registration (not a global
// init()-based registry like the real framework) keeps this simple to test
// and to reason about which plugins are active in a given build
type Registry struct {
	filters []FilterPlugin
	scores  []ScorePluginWithWeight
	logger  *slog.Logger
}

func NewRegistry(logger *slog.Logger) *Registry {
	return &Registry{logger: logger}
}

func (r *Registry) RegisterFilter(p FilterPlugin) {
	r.filters = append(r.filters, p)
}

func (r *Registry) RegisterScore(p ScorePlugin, weight int64) {
	r.scores = append(r.scores, ScorePluginWithWeight{Plugin: p, Weight: weight})
}

// RunFilterPlugins returns the subset of nodeInfos that pass every filter.
// Short-circuits per-node on first failing filter (matches real framework
// behavior - no point running remaining filters once one has rejected)
func (r *Registry) RunFilterPlugins(pod *corev1.Pod, nodeInfos []*NodeInfo) []*NodeInfo {
	var feasible []*NodeInfo
	for _, ni := range nodeInfos {
		ok := true
		for _, p := range r.filters {
			status := p.Filter(pod, ni)
			if !status.IsSuccess() {
				r.logger.Debug("node filtered out", "node", ni.Node.Name, "plugin", p.Name(), "reason", status.Messaeg)
				ok = false
				break
			}
		}
		if ok {
			feasible = append(feasible, ni)
		}
	}
	return feasible
}

// RunScorePlugins computes the weighted-sum score per node. A plugin
// returning an Error status contributes zero rather than aborting the
// whole scoring pass - one misbehaving plugin shouldn't block scheduling
func (r *Registry) RunScorePlugins(pod *corev1.Pod, nodeInfos []*NodeInfo) map[string]int64 {
	scores := make(map[string]int64, len(nodeInfos))
	for _, ni := range nodeInfos {
		var total int64
		for _, sw := range r.scores {
			score, status := sw.Plugin.Score(pod, ni)
			if !status.IsSuccess() {
				r.logger.Warn("score plugin failed", "node", ni.Node.Name, "plugin", sw.Plugin.Name(), "reason", status.Messaeg)
				continue
			}
			total += score * sw.Weight
		}
		scores[ni.Node.Name] = total
	}
	return scores
}
