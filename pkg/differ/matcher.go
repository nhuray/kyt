package differ

import (
	"github.com/nhuray/kyt/pkg/manifest"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// MatchType describes how two resources were matched
type MatchType string

const (
	// MatchTypeExact indicates resources matched by exact ResourceKey
	MatchTypeExact MatchType = "exact"

	// MatchTypeSimilarity indicates resources matched by similarity score
	MatchTypeSimilarity MatchType = "similarity"
)

// Match represents a matched pair of resources
type Match struct {
	SourceKey       manifest.ResourceKey
	TargetKey       manifest.ResourceKey
	SourceResource  *unstructured.Unstructured
	TargetResource  *unstructured.Unstructured
	Type            MatchType
	SimilarityScore float64
}

// GVK is a grouping key for resources: (Group, Version, Kind)
// Namespace is NOT included to allow cross-namespace comparisons
type GVK struct {
	Group   string
	Version string
	Kind    string
}

// ResourceMatcher performs 2-stage matching: exact then similarity-based
type ResourceMatcher struct {
	scorer                *SimilarityScorer
	enableSimilarityMatch bool
	similarityThreshold   float64
}

// NewResourceMatcher creates a new matcher with given options
func NewResourceMatcher(enableSimilarity bool, threshold float64) *ResourceMatcher {
	return &ResourceMatcher{
		scorer:                NewDefaultSimilarityScorer(),
		enableSimilarityMatch: enableSimilarity,
		similarityThreshold:   threshold,
	}
}

// NewResourceMatcherWithStringThreshold creates a matcher with custom string similarity threshold
func NewResourceMatcherWithStringThreshold(enableSimilarity bool, threshold float64, stringThreshold int) *ResourceMatcher {
	return &ResourceMatcher{
		scorer:                NewSimilarityScorerWithThreshold(stringThreshold),
		enableSimilarityMatch: enableSimilarity,
		similarityThreshold:   threshold,
	}
}

// NewResourceMatcherWithOptions creates a matcher with custom options
func NewResourceMatcherWithOptions(enableSimilarity bool, threshold float64, stringThreshold int, dataBoost int) *ResourceMatcher {
	return &ResourceMatcher{
		scorer:                NewSimilarityScorerWithOptions(stringThreshold, dataBoost),
		enableSimilarityMatch: enableSimilarity,
		similarityThreshold:   threshold,
	}
}

// MatchResources performs 2-stage matching between source and target resources
// Returns: matched pairs, unmatched source keys, unmatched target keys
func (m *ResourceMatcher) MatchResources(
	source map[manifest.ResourceKey]*unstructured.Unstructured,
	target map[manifest.ResourceKey]*unstructured.Unstructured,
) ([]Match, []manifest.ResourceKey, []manifest.ResourceKey) {

	matches := []Match{}

	// Stage 1: Find exact matches
	unmatchedSource, unmatchedTarget := m.findExactMatches(source, target, &matches)

	// Stage 2: Find similarity-based matches (if enabled)
	if m.enableSimilarityMatch && (len(unmatchedSource) > 0 || len(unmatchedTarget) > 0) {
		stillUnmatchedSource, stillUnmatchedTarget := m.findSimilarityMatches(
			unmatchedSource,
			unmatchedTarget,
			&matches,
		)
		unmatchedSource = stillUnmatchedSource
		unmatchedTarget = stillUnmatchedTarget
	}

	return matches, unmatchedSource, unmatchedTarget
}

// findExactMatches finds resources with identical ResourceKeys
func (m *ResourceMatcher) findExactMatches(
	source map[manifest.ResourceKey]*unstructured.Unstructured,
	target map[manifest.ResourceKey]*unstructured.Unstructured,
	matches *[]Match,
) ([]manifest.ResourceKey, []manifest.ResourceKey) {

	unmatchedSource := []manifest.ResourceKey{}
	unmatchedTarget := make(map[manifest.ResourceKey]*unstructured.Unstructured)

	// Copy target to track unmatched
	for k, v := range target {
		unmatchedTarget[k] = v
	}

	// Try to match each source resource
	for sourceKey, sourceObj := range source {
		if targetObj, found := target[sourceKey]; found {
			// Exact match found
			*matches = append(*matches, Match{
				SourceKey:       sourceKey,
				TargetKey:       sourceKey,
				SourceResource:  sourceObj,
				TargetResource:  targetObj,
				Type:            MatchTypeExact,
				SimilarityScore: 1.0,
			})
			delete(unmatchedTarget, sourceKey)
		} else {
			// No exact match
			unmatchedSource = append(unmatchedSource, sourceKey)
		}
	}

	// Convert remaining unmatched target to slice
	unmatchedTargetKeys := []manifest.ResourceKey{}
	for k := range unmatchedTarget {
		unmatchedTargetKeys = append(unmatchedTargetKeys, k)
	}

	return unmatchedSource, unmatchedTargetKeys
}

// findSimilarityMatches finds matches based on structural similarity
func (m *ResourceMatcher) findSimilarityMatches(
	sourceKeys []manifest.ResourceKey,
	targetKeys []manifest.ResourceKey,
	matches *[]Match,
) ([]manifest.ResourceKey, []manifest.ResourceKey) {

	if len(sourceKeys) == 0 || len(targetKeys) == 0 {
		return sourceKeys, targetKeys
	}

	// Group unmatched resources by GVK
	sourceGroups := m.groupByGVK(sourceKeys)
	targetGroups := m.groupByGVK(targetKeys)

	stillUnmatchedSource := []manifest.ResourceKey{}
	stillUnmatchedTarget := make(map[manifest.ResourceKey]bool)

	for _, key := range targetKeys {
		stillUnmatchedTarget[key] = true
	}

	// Process each GVK group
	for gvk, sourceGroupKeys := range sourceGroups {
		targetGroupKeys, hasTargetGroup := targetGroups[gvk]
		if !hasTargetGroup {
			// No target resources of this type - all remain unmatched
			stillUnmatchedSource = append(stillUnmatchedSource, sourceGroupKeys...)
			continue
		}

		// Find best matches within this group
		groupMatches, groupUnmatchedSource, _ := m.findBestMatches(
			sourceGroupKeys,
			targetGroupKeys,
		)

		*matches = append(*matches, groupMatches...)
		stillUnmatchedSource = append(stillUnmatchedSource, groupUnmatchedSource...)

		// Remove successfully matched targets from unmatched set
		for _, match := range groupMatches {
			delete(stillUnmatchedTarget, match.TargetKey)
		}
	}

	// Target groups with no corresponding source group remain unmatched
	// They're already in stillUnmatchedTarget

	// Convert unmatched target map to slice
	stillUnmatchedTargetKeys := []manifest.ResourceKey{}
	for k := range stillUnmatchedTarget {
		stillUnmatchedTargetKeys = append(stillUnmatchedTargetKeys, k)
	}

	return stillUnmatchedSource, stillUnmatchedTargetKeys
}

// groupByGVK groups resources by their GVK (Group, Version, Kind)
// Namespace is NOT included to allow cross-namespace comparisons
func (m *ResourceMatcher) groupByGVK(keys []manifest.ResourceKey) map[GVK][]manifest.ResourceKey {
	groups := make(map[GVK][]manifest.ResourceKey)

	for _, key := range keys {
		gvk := GVK{
			Group:   key.Group,
			Version: key.Version,
			Kind:    key.Kind,
		}

		groups[gvk] = append(groups[gvk], key)
	}

	return groups
}

// findBestMatches finds best similarity-based matches within a GVK group
// Uses greedy matching: pair highest similarity first
func (m *ResourceMatcher) findBestMatches(
	sourceKeys []manifest.ResourceKey,
	targetKeys []manifest.ResourceKey,
) ([]Match, []manifest.ResourceKey, []manifest.ResourceKey) {

	if len(sourceKeys) == 0 || len(targetKeys) == 0 {
		return []Match{}, sourceKeys, targetKeys
	}

	// Build similarity matrix
	type scoredPair struct {
		sourceIdx int
		targetIdx int
		score     float64
	}

	var pairs []scoredPair

	for si, sKey := range sourceKeys {
		for ti, tKey := range targetKeys {
			// Get actual resources from cache
			sourceRes := m.getResource(sKey)
			targetRes := m.getResource(tKey)

			if sourceRes == nil || targetRes == nil {
				continue
			}

			score := m.scorer.CompareResources(sourceRes, targetRes)

			if score >= m.similarityThreshold {
				pairs = append(pairs, scoredPair{
					sourceIdx: si,
					targetIdx: ti,
					score:     score,
				})
			}
		}
	}

	// Sort pairs by score descending (greedy matching)
	// Using simple bubble sort for now
	for i := 0; i < len(pairs); i++ {
		for j := i + 1; j < len(pairs); j++ {
			if pairs[j].score > pairs[i].score {
				pairs[i], pairs[j] = pairs[j], pairs[i]
			}
		}
	}

	// Greedy pairing: take highest scoring pairs first
	matchedSource := make(map[int]bool)
	matchedTarget := make(map[int]bool)
	matches := []Match{}

	for _, pair := range pairs {
		if !matchedSource[pair.sourceIdx] && !matchedTarget[pair.targetIdx] {
			// This is the best available match for both resources
			sourceKey := sourceKeys[pair.sourceIdx]
			targetKey := targetKeys[pair.targetIdx]

			matches = append(matches, Match{
				SourceKey:       sourceKey,
				TargetKey:       targetKey,
				SourceResource:  m.getResource(sourceKey),
				TargetResource:  m.getResource(targetKey),
				Type:            MatchTypeSimilarity,
				SimilarityScore: pair.score,
			})

			matchedSource[pair.sourceIdx] = true
			matchedTarget[pair.targetIdx] = true
		}
	}

	// Collect unmatched
	unmatchedSource := []manifest.ResourceKey{}
	for i, key := range sourceKeys {
		if !matchedSource[i] {
			unmatchedSource = append(unmatchedSource, key)
		}
	}

	unmatchedTarget := []manifest.ResourceKey{}
	for i, key := range targetKeys {
		if !matchedTarget[i] {
			unmatchedTarget = append(unmatchedTarget, key)
		}
	}

	return matches, unmatchedSource, unmatchedTarget
}

// resourceCache stores resources for similarity comparison
var resourceCache = make(map[manifest.ResourceKey]*unstructured.Unstructured)

// getResource retrieves a resource from the cache
func (m *ResourceMatcher) getResource(key manifest.ResourceKey) *unstructured.Unstructured {
	return resourceCache[key]
}

// SetResourceCache sets the global resource cache for matching
// This should be called before MatchResources to populate the cache
func SetResourceCache(source, target map[manifest.ResourceKey]*unstructured.Unstructured) {
	resourceCache = make(map[manifest.ResourceKey]*unstructured.Unstructured)
	for k, v := range source {
		resourceCache[k] = v
	}
	for k, v := range target {
		resourceCache[k] = v
	}
}
