package src

import (
	"testing"

	execdb "github.com/EricChiquitoG/Remanet_DSM/client/src/mongo"
)

// Fixtures mirror real seeded data (Provider-001's Design capability and
// Provider-000's Coating capability), not invented values, so a change here
// should track a real change in the dataset.

func motorUseCase() execdb.UseCase {
	return execdb.UseCase{
		ID:                      "electric-motor",
		ExistenceOnlyOperations: []string{"AssemblyTesting", "Reuse", "Replace"},
		AttributeRules: map[string]execdb.AttributeRule{
			"AxisHeight":          {Type: "numeric_min"},
			"RatedPower":          {Type: "numeric_min"},
			"ShaftLength":         {Type: "numeric_min"},
			"ShaftDiameter":       {Type: "numeric_min"},
			"ShaftToleranceClass": {Type: "ordinal", Order: []string{"bad", "mid", "good"}},
		},
	}
}

func toolUseCase() execdb.UseCase {
	return execdb.UseCase{
		ID:                      "cutting-tool",
		ExistenceOnlyOperations: []string{"DimensionalTesting", "Grinding", "Inspection"},
	}
}

func designCapability() execdb.CapabilityRecord {
	return execdb.CapabilityRecord{
		UseCase:    "electric-motor",
		ProviderID: "1",
		Operation:  "Design",
		Equipment: []execdb.Equipment{
			{Name: "AsyMotorRedesignTool", Attributes: map[string][]string{
				"ShaftToleranceClass": {"mid"},
				"ShaftLength":         {"850 mm"},
				"ShaftDiameter":       {"28 mm"},
				"RatedPower":          {"45.000 kW"},
				"AxisHeight":          {"250 mm"},
			}},
			{Name: "FeaSimulationService", Attributes: map[string][]string{
				"SimulationType": {"electromagnetic / thermal"},
			}},
			{Name: "CadDesignService", Attributes: map[string][]string{
				"DesignStandard": {"ISO 2768"},
			}},
		},
	}
}

func coatingCapability() execdb.CapabilityRecord {
	return execdb.CapabilityRecord{
		UseCase:    "cutting-tool",
		ProviderID: "0",
		Operation:  "Coating",
		Equipment: []execdb.Equipment{
			{Name: "PvdCoatingService", Attributes: map[string][]string{
				"MaterialCategory": {"Ceramic"},
				"CoatingType":      {"TiN"},
			}},
		},
	}
}

func TestMatchCapability_KeyPresentValueMatches(t *testing.T) {
	if !MatchCapability(motorUseCase(), designCapability(), Requirements{"AxisHeight": {"250 mm"}}) {
		t.Error("expected match: AxisHeight 250 mm is exactly what the equipment offers")
	}
}

func TestMatchCapability_KeyPresentValueMismatch(t *testing.T) {
	if MatchCapability(motorUseCase(), designCapability(), Requirements{"AxisHeight": {"999 mm"}}) {
		t.Error("expected no match: equipment's 250 mm capacity can't cover a 999 mm requirement")
	}
}

func TestMatchCapability_NumericMinAcceptsRequirementBelowCapacity(t *testing.T) {
	// The whole point of numeric_min: a 250 mm-rated machine can obviously
	// also take a smaller 200 mm job. Plain string equality would have
	// rejected this (this is the exact failure mode the real trace-match
	// exercise against live data kept hitting).
	if !MatchCapability(motorUseCase(), designCapability(), Requirements{"AxisHeight": {"200"}}) {
		t.Error("expected match: equipment's 250 mm capacity covers a 200 mm requirement")
	}
}

func TestMatchCapability_NumericMinIgnoresUnitFormatting(t *testing.T) {
	// A requirement built straight from a batch's own (unit-less) attribute
	// value — "200", not "200 mm" — must still compare correctly against
	// equipment values that always carry a unit suffix.
	if !MatchCapability(motorUseCase(), designCapability(), Requirements{"RatedPower": {"40"}}) {
		t.Error("expected match: equipment's 45.000 kW capacity covers a bare '40' requirement")
	}
	if MatchCapability(motorUseCase(), designCapability(), Requirements{"RatedPower": {"50"}}) {
		t.Error("expected no match: equipment's 45.000 kW capacity doesn't cover a 50 requirement")
	}
}

func TestMatchCapability_OrdinalRankSatisfiesLesserOrEqualRequirement(t *testing.T) {
	// designCapability's equipment is ShaftToleranceClass=mid.
	cap := designCapability()
	if !MatchCapability(motorUseCase(), cap, Requirements{"ShaftToleranceClass": {"bad"}}) {
		t.Error("expected match: a mid-precision machine also satisfies a looser 'bad' requirement")
	}
	if !MatchCapability(motorUseCase(), cap, Requirements{"ShaftToleranceClass": {"mid"}}) {
		t.Error("expected match: exact rank requested")
	}
	if MatchCapability(motorUseCase(), cap, Requirements{"ShaftToleranceClass": {"good"}}) {
		t.Error("expected no match: a mid-precision machine can't satisfy a stricter 'good' requirement")
	}
}

func TestMatchCapability_PresenceOnlyWhenNoValueRequested(t *testing.T) {
	if !MatchCapability(motorUseCase(), designCapability(), Requirements{"ShaftToleranceClass": nil}) {
		t.Error("expected match: key exists, no specific value was requested")
	}
}

func TestMatchCapability_MissingKeyFails(t *testing.T) {
	if MatchCapability(motorUseCase(), designCapability(), Requirements{"CoolantType": nil}) {
		t.Error("expected no match: no equipment on this capability declares CoolantType")
	}
}

func TestMatchCapability_MustHaveAllRequestedValues(t *testing.T) {
	cap := coatingCapability()
	// equipment supports only TiN — asking for TiN *and* AlTiN must fail.
	if MatchCapability(toolUseCase(), cap, Requirements{"CoatingType": {"TiN", "AlTiN"}}) {
		t.Error("expected no match: equipment doesn't support AlTiN")
	}
	// a single requested value it does support should still pass.
	if !MatchCapability(toolUseCase(), cap, Requirements{"CoatingType": {"TiN"}}) {
		t.Error("expected match: equipment supports TiN")
	}
}

func TestMatchCapability_MultiValuedEquipmentAttribute(t *testing.T) {
	cap := execdb.CapabilityRecord{
		Operation: "Coating",
		Equipment: []execdb.Equipment{
			{Name: "AdvancedCoatingService", Attributes: map[string][]string{
				"MaterialCategory": {"Ceramic", "Carbide"},
			}},
		},
	}
	if !MatchCapability(toolUseCase(), cap, Requirements{"MaterialCategory": {"Ceramic", "Carbide"}}) {
		t.Error("expected match: equipment supports both requested materials")
	}
	if MatchCapability(toolUseCase(), cap, Requirements{"MaterialCategory": {"Ceramic", "Diamond"}}) {
		t.Error("expected no match: equipment doesn't support Diamond")
	}
}

func TestMatchCapability_ExistenceOnlyOperationIgnoresAttributes(t *testing.T) {
	// AssemblyTesting/Reuse/Replace carry no equipment attributes in the
	// source data at all — offering the capability is enough, regardless
	// of what a requirement asks for.
	cap := execdb.CapabilityRecord{Operation: "AssemblyTesting", Equipment: nil}
	if !MatchCapability(motorUseCase(), cap, Requirements{"AxisHeight": {"250 mm"}}) {
		t.Error("expected match: AssemblyTesting is existence-only, offering it is sufficient")
	}
}

func TestMatchCapability_EmptyRequirementAlwaysMatches(t *testing.T) {
	cap := execdb.CapabilityRecord{Operation: "DimensionalTesting", Equipment: nil}
	if !MatchCapability(toolUseCase(), cap, Requirements{}) {
		t.Error("expected match: nothing was requested, even though this operation has no equipment attributes at all")
	}
}

func TestMatchCapability_CuttingToolExistenceOnlyOperationsIgnoreAttributes(t *testing.T) {
	// DimensionalTesting/Grinding/Inspection carry no equipment attributes
	// for any of the 100 cutting-tool providers — same situation as the
	// motor exception — so they're declared existence-only too.
	for _, op := range []string{"DimensionalTesting", "Grinding", "Inspection"} {
		cap := execdb.CapabilityRecord{Operation: op, Equipment: nil}
		if !MatchCapability(toolUseCase(), cap, Requirements{"MeasurementAccuracy": {"< 0.01 mm"}}) {
			t.Errorf("expected match: %s is existence-only, offering it is sufficient", op)
		}
	}
}

func TestMatchCapability_UndeclaredNoAttributeOperationFailsRealRequirement(t *testing.T) {
	// Mechanism check, independent of either use case's current config: an
	// operation with no equipment attributes that is NOT declared
	// existence-only must fail any real (value-bearing) requirement —
	// there's nothing to satisfy it against.
	uc := execdb.UseCase{ID: "hypothetical", ExistenceOnlyOperations: []string{}}
	cap := execdb.CapabilityRecord{Operation: "SomeOperation", Equipment: nil}
	if MatchCapability(uc, cap, Requirements{"SomeAttribute": nil}) {
		t.Error("expected no match: no equipment attributes exist, and the operation isn't existence-only")
	}
}
