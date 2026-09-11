// Seed script: populates aas_actors collection with mock AAS responses.
// Requirements used as design basis: axis_height=50, power=5.5, KHFT=false (Class A);
//   tolerance=mid, length=50, diameter=50 (Class C); axis_height=50 (Class D).
// Actors marked FAIL will not match once matching functions enforce limits.

const targetDB = process.env.MONGODB_DB || 'dsm_db';
const db = db.getSiblingDB(targetDB);

// Build a submodel property element.
const prop = (idShort, value, valueType) => ({
    idShort,
    modelType: { name: "Property" },
    value,
    valueType
});

// Build a full AAS response document for a Class A actor (Disassembly/Cleaning/etc.).
const makeClassA = (n, name, axisHeight, power, khft, cost, co2) => ({
    actorId: `urn:remanet:org:disassembler:${n}`,
    aasResponse: JSON.stringify({
        status: "success",
        data: {
            modelType: { name: "AssetAdministrationShell" },
            idShort: name,
            identification: { idType: "Custom", id: `urn:remanet:org:disassembler:${n}` },
            submodels: [{
                idShort: "CapabilityDescription",
                identification: { idType: "IRI", id: `urn:remanet:submodel:${name.toLowerCase()}` },
                modelType: { name: "Submodel" },
                kind: "Instance",
                submodelElements: [
                    prop("cost_per_hour",          String(cost),        "double"),
                    prop("co2_static_per_hour",    String(co2),         "double"),
                    prop("axis_height",            String(axisHeight),  "double"),
                    prop("power",                  String(power),       "double"),
                    { idShort: "know-how_formalization-tool", modelType: { name: "Property" }, value: khft, valueType: "boolean" }
                ]
            }],
            asset: {
                identification: { idType: "Custom", id: `urn:remanet:org:disassembler:${n}` },
                idShort: name
            }
        }
    })
});

// Build a full AAS response document for a Class C actor (Turning/Grinding).
const makeClassC = (n, name, tolerance, length, diameter, cost, co2) => ({
    actorId: `urn:remanet:org:shaft:${n}`,
    aasResponse: JSON.stringify({
        status: "success",
        data: {
            modelType: { name: "AssetAdministrationShell" },
            idShort: name,
            identification: { idType: "Custom", id: `urn:remanet:org:shaft:${n}` },
            submodels: [{
                idShort: "CapabilityDescription",
                identification: { idType: "IRI", id: `urn:remanet:submodel:${name.toLowerCase()}` },
                modelType: { name: "Submodel" },
                kind: "Instance",
                submodelElements: [
                    prop("cost_per_hour",       String(cost),      "double"),
                    prop("co2_static_per_hour", String(co2),       "double"),
                    prop("tolerance",           tolerance,         "string"),
                    prop("length",              String(length),    "double"),
                    prop("diameter",            String(diameter),  "double")
                ]
            }],
            asset: {
                identification: { idType: "Custom", id: `urn:remanet:org:shaft:${n}` },
                idShort: name
            }
        }
    })
});

// Build a full AAS response document for a Class D actor (Punching).
const makeClassD = (n, name, axisHeight, cost, co2) => ({
    actorId: `urn:remanet:org:lamination:${n}`,
    aasResponse: JSON.stringify({
        status: "success",
        data: {
            modelType: { name: "AssetAdministrationShell" },
            idShort: name,
            identification: { idType: "Custom", id: `urn:remanet:org:lamination:${n}` },
            submodels: [{
                idShort: "CapabilityDescription",
                identification: { idType: "IRI", id: `urn:remanet:submodel:${name.toLowerCase()}` },
                modelType: { name: "Submodel" },
                kind: "Instance",
                submodelElements: [
                    prop("cost_per_hour",       String(cost),       "double"),
                    prop("co2_static_per_hour", String(co2),        "double"),
                    prop("axis_height",         String(axisHeight), "double")
                ]
            }],
            asset: {
                identification: { idType: "Custom", id: `urn:remanet:org:lamination:${n}` },
                idShort: name
            }
        }
    })
});

db.aas_actors.drop();

db.aas_actors.insertMany([
    // ── Class A: Cleaning, Disassembly, BearingSubstitution, Reassembly, Certification ──
    // Req: axis_height >= 50, power >= 5.5, KHFT = false
    makeClassA(0, "Dissasembler0", 100, 11.0, true,  58.61, 64.03), // FAIL: KHFT mismatch
    makeClassA(1, "Dissasembler1", 160, 22.0, false, 45.00, 30.00), // PASS: high capacity, low cost
    makeClassA(2, "Dissasembler2",  80,  7.5, false, 52.00, 42.00), // PASS: mid capacity
    makeClassA(3, "Dissasembler3", 132, 15.0, false, 68.00, 55.00), // PASS: high capacity, premium
    makeClassA(4, "Dissasembler4",  50,  5.5, false, 55.00, 44.00), // PASS: exact boundary match
    makeClassA(5, "Dissasembler5", 200, 37.0, true,  88.00, 72.00), // FAIL: KHFT mismatch
    makeClassA(6, "Dissasembler6",  40,  4.0, false, 35.00, 28.00), // FAIL: axis_height + power too low
    makeClassA(7, "Dissasembler7",  90,  9.5, false, 62.00, 50.00), // PASS: mid-high capacity

    // ── Class C: Turning, Grinding ──
    // Req: tolerance >= mid, length >= 50, diameter >= 50
    makeClassC(0, "Shaft0", "good", 100, 100, 54.34, 34.07), // PASS: high precision, high cap
    makeClassC(1, "Shaft1", "high",  80,  80, 70.00, 40.00), // PASS: high precision
    makeClassC(2, "Shaft2", "mid",   60,  60, 48.00, 35.00), // PASS: standard precision
    makeClassC(3, "Shaft3", "good", 120, 100, 65.00, 45.00), // PASS: high precision, larger cap
    makeClassC(4, "Shaft4", "mid",   50,  50, 52.00, 38.00), // PASS: exact boundary match
    makeClassC(5, "Shaft5", "low",   40,  40, 32.00, 25.00), // FAIL: tolerance + dims too low
    makeClassC(6, "Shaft6", "high", 150, 120, 75.00, 52.00), // PASS: high precision, max cap

    // ── Class D: Punching ──
    // Req: axis_height >= 50
    makeClassD(0, "Lamination0", 100, 54.10, 37.49), // PASS: high capacity
    makeClassD(1, "Lamination1", 160, 70.00, 50.00), // PASS: very high capacity
    makeClassD(2, "Lamination2",  80, 45.00, 33.00), // PASS: mid capacity
    makeClassD(3, "Lamination3",  50, 40.00, 28.00), // PASS: exact boundary match
    makeClassD(4, "Lamination4",  40, 30.00, 22.00), // FAIL: axis_height too low
]);

db.aas_actors.createIndex({ actorId: 1 }, { unique: true });

print(`aas_actors seeded: ${db.aas_actors.countDocuments()} records in ${targetDB}`);
