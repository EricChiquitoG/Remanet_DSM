// seed_products.js
// Run via:  mongosh <connection-string> scripts/seed_products.js
// or:       docker exec -i test-mongo mongosh "mongodb://erichi:secret@localhost/dsm_db?authSource=admin" < scripts/seed_products.js

const db = connect("mongodb://erichi:secret@localhost:27017/dsm_db?authSource=admin");

// ── Reset ────────────────────────────────────────────────────────────────────
db.products.drop();
db.product_history.drop();
print("✓ Dropped products and product_history collections");

// ── Products ─────────────────────────────────────────────────────────────────
// Products 1–3 have been inspected (ownerHistory / components / inspectionNotes filled).
// Products 4–6 have never been inspected (those fields are null).
// History records exist only for products 1–3.

const now = new Date();

const products = [
  {
    id: "IE4-MOTOR-001",
    cls: "IE4",
    axis: 132,
    pow: 11.0,
    batch: "Batch #14",
    components: {
      bearings: "Worn",
      stator:   "Good",
      rotor:    "Good",
      housing:  "Good",
      windings: "Worn",
      shaft:    "Good"
    },
    ownerHistory:    ["NordicPlant AB", "EnergyTech SE"],
    inspectionNotes: "Bearings show early wear; windings slightly degraded",
    createdAt: now,
    updatedAt: now
  },
  {
    id: "IE3-MOTOR-002",
    cls: "IE3",
    axis: 100,
    pow: 5.5,
    batch: "Batch #26",
    components: {
      bearings: "Good",
      stator:   "Good",
      rotor:    "Degraded",
      housing:  "Good",
      windings: "Good",
      shaft:    "Good"
    },
    ownerHistory:    ["Volvo Logistics GmbH"],
    inspectionNotes: "Rotor shows minor deformation on inner ring",
    createdAt: now,
    updatedAt: now
  },
  {
    id: "IE2-MOTOR-003",
    cls: "IE2",
    axis: 80,
    pow: 3.0,
    batch: null,
    components: {
      bearings: "Failed",
      stator:   "Worn",
      rotor:    "Good",
      housing:  "Degraded",
      windings: "Failed",
      shaft:    "Good"
    },
    ownerHistory:    ["Siemens Plant Barcelona", "OldTech SA"],
    inspectionNotes: "End-of-life candidate; bearings and windings have failed",
    createdAt: now,
    updatedAt: now
  },
  // Products 4–6: no inspection data yet
  {
    id: "IE4-MOTOR-004",
    cls: "IE4",
    axis: 160,
    pow: 22.0,
    batch: "Batch #64",
    components:      null,
    ownerHistory:    null,
    inspectionNotes: null,
    createdAt: now,
    updatedAt: now
  },
  {
    id: "IE3-MOTOR-005",
    cls: "IE3",
    axis: 112,
    pow: 7.5,
    batch: null,
    components:      null,
    ownerHistory:    null,
    inspectionNotes: null,
    createdAt: now,
    updatedAt: now
  },
  {
    id: "IE4-MOTOR-006",
    cls: "IE4",
    axis: 200,
    pow: 45.0,
    batch: "Batch #85",
    components:      null,
    ownerHistory:    null,
    inspectionNotes: null,
    createdAt: now,
    updatedAt: now
  }
];

db.products.insertMany(products);
print(`✓ Inserted ${products.length} products`);

// ── History (only for products 1–3) ─────────────────────────────────────────
const history = [
  {
    productId:   "IE4-MOTOR-001",
    owners:      ["NordicPlant AB", "EnergyTech SE"],
    age:         5,
    lastService: "2024-03",
    notes:       "Bearing replacement in 2022; preventive winding check in 2024"
  },
  {
    productId:   "IE3-MOTOR-002",
    owners:      ["Volvo Logistics GmbH"],
    age:         3,
    lastService: "2023-11",
    notes:       "First inspection revealed rotor deformation; monitoring scheduled"
  },
  {
    productId:   "IE2-MOTOR-003",
    owners:      ["Siemens Plant Barcelona", "OldTech SA"],
    age:         12,
    lastService: "2022-06",
    notes:       "Multiple repair cycles; decommission recommended after current batch"
  }
];

db.product_history.insertMany(history);
print(`✓ Inserted ${history.length} history records`);

// ── Summary ──────────────────────────────────────────────────────────────────
print("\n── Product IDs with history (test GET /products/:id/history) ──");
history.forEach(h => print(`  ${h.productId}`));

print("\n── Product IDs without history (should return 404 on history endpoint) ──");
products.slice(3).forEach(p => print(`  ${p.id}`));

print("\nDone.");
