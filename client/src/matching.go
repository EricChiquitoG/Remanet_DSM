package src

// Generic capability matching against the use-case-agnostic collections in
// client/src/mongo/catalog.go. This replaces the old per-class
// matchClassA/C/D approach: one function, driven by whatever operations and
// requirement attributes a UseCase declares, with a single documented
// exception (ExistenceOnlyOperations) for operations the source data never
// attaches equipment attributes to.
//
// Wired into the live /optimize request path via techProviderSearch in
// messaging.go, which uses Requirements (defined there) as the shared
// wire-format/matching type.

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	execdb "github.com/EricChiquitoG/Remanet_DSM/client/src/mongo"
	"go.mongodb.org/mongo-driver/mongo"
)

// MatchCapability reports whether one provider's capability for an
// operation satisfies a requirement:
//
//   - Operations listed in the use case's ExistenceOnlyOperations match as
//     soon as the capability record exists (i.e. the provider offers the
//     operation at all) — the source data never carries equipment
//     attributes for these, so there's nothing further to check.
//   - For every other operation: every key in the requirement must exist
//     among the capability's equipment attributes (merged across all of
//     that provider's equipment for the operation). How a requested value
//     is checked against the equipment's values then depends on the use
//     case's AttributeRules for that key — numeric-capacity, ordinal-rank,
//     or (default, no rule declared) exact value-set membership.
//
// An empty requirement always matches (vacuously true), including for
// operations with no equipment attributes at all.
func MatchCapability(uc execdb.UseCase, cap execdb.CapabilityRecord, requirement Requirements) bool {
	if slices.Contains(uc.ExistenceOnlyOperations, cap.Operation) {
		return true
	}

	available := mergeEquipmentAttributes(cap.Equipment)
	for key, requested := range requirement {
		values, ok := available[key]
		if !ok {
			return false
		}
		rule, hasRule := uc.AttributeRules[key]
		for _, want := range requested {
			if !attributeSatisfies(hasRule, rule, values, want) {
				return false
			}
		}
	}
	return true
}

// attributeSatisfies checks one requested value against an equipment's
// values for the same attribute, per AttributeRule (see its doc comment).
func attributeSatisfies(hasRule bool, rule execdb.AttributeRule, values []string, want string) bool {
	if !hasRule {
		return slices.Contains(values, want)
	}
	switch rule.Type {
	case "numeric_min":
		wantNum, ok := leadingFloat(want)
		if !ok {
			return false
		}
		for _, v := range values {
			if n, ok := leadingFloat(v); ok && n >= wantNum {
				return true
			}
		}
		return false
	case "ordinal":
		wantRank := rankOf(rule.Order, want)
		if wantRank < 0 {
			return false
		}
		for _, v := range values {
			if r := rankOf(rule.Order, v); r >= wantRank {
				return true
			}
		}
		return false
	default:
		return slices.Contains(values, want)
	}
}

var leadingNumberPattern = regexp.MustCompile(`^\s*(-?\d+\.?\d*)`)

// leadingFloat extracts the leading numeric token from a value regardless
// of trailing formatting — "355", "355 mm", and "315.000 kW" all parse to
// the same number. This is what lets a requirement built straight from a
// batch's own (unit-less) attribute value compare correctly against
// equipment data that always carries a unit suffix.
func leadingFloat(s string) (float64, bool) {
	m := leadingNumberPattern.FindStringSubmatch(s)
	if m == nil {
		return 0, false
	}
	n, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return 0, false
	}
	return n, true
}

// rankOf returns v's index in order (case-insensitive), or -1 if absent.
func rankOf(order []string, v string) int {
	for i, o := range order {
		if strings.EqualFold(o, v) {
			return i
		}
	}
	return -1
}

// mergeEquipmentAttributes flattens every equipment item's attributes for
// one (provider, operation) capability into a single key -> values lookup.
// Matching operates at the provider/operation level — "does this provider
// have equipment somewhere that covers each requested attribute" — not
// per individual equipment item, since a requirement is not tied to a
// specific named machine.
func mergeEquipmentAttributes(equipment []execdb.Equipment) map[string][]string {
	merged := make(map[string][]string)
	for _, eq := range equipment {
		for k, vals := range eq.Attributes {
			merged[k] = append(merged[k], vals...)
		}
	}
	return merged
}

// MatchProvider fetches a provider's capability for one operation and
// reports whether it satisfies the requirement. Returns (false, nil) —
// not an error — when the provider simply doesn't offer the operation.
func MatchProvider(useCaseID, providerID, operation string, requirement Requirements) (bool, error) {
	uc, err := execdb.GetUseCase(useCaseID)
	if err != nil {
		return false, fmt.Errorf("use case %s: %w", useCaseID, err)
	}
	capability, err := execdb.GetCapability(useCaseID, providerID, operation)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return false, nil
		}
		return false, fmt.Errorf("capability %s/%s/%s: %w", useCaseID, providerID, operation, err)
	}
	return MatchCapability(uc, capability, requirement), nil
}
