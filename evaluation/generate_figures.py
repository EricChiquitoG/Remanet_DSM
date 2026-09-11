"""
Generate evaluation figures as SVG for paper import.
Run from the evaluation/ directory:
    python3 generate_figures.py
"""
import matplotlib
matplotlib.use("Agg")
import matplotlib.pyplot as plt
import matplotlib.patches as mpatches
import os

OUT = os.path.join(os.path.dirname(__file__), "figures")
os.makedirs(OUT, exist_ok=True)

BLUE  = "#2563EB"
AMBER = "#D97706"
GREEN = "#059669"
RUN1  = "#3B82F6"
RUN2  = "#F97316"
RUN3  = "#059669"
GREY  = "#94A3B8"

RUN_COLORS = [RUN1, RUN2, RUN3]
RUN_LABELS = ["Run 1", "Run 2", "Run 3"]

# Min manufacturing cost (best Pareto pole, averaged across 3 runs) per scenario
MFG_A = [295.1, 227.0, 227.0, 227.0, 177.0]   # 1–8 actors
MFG_P = [72.0,  142.0, 177.0, 236.0, 371.0]   # P₂–P₉
MFG_R = [177.0, 177.0, 177.0, 177.0]           # 1–8 concurrent routes


def _cost_overlay(ax, xs, mfg_values, flat=False):
    """Overlay min manufacturing cost on a secondary right y-axis."""
    ax2 = ax.twinx()
    ax2.plot(xs, mfg_values, color=GREEN, linewidth=1.5,
             marker="o", markersize=4, zorder=4)
    ax2.set_ylabel("min mfg cost (€)", fontsize=7.5, color=GREEN)
    ax2.tick_params(axis="y", labelcolor=GREEN, labelsize=8)
    ax2.spines["right"].set_visible(True)
    ax2.spines["right"].set_linewidth(0.7)
    ax2.spines["right"].set_color(GREEN)
    ax2.spines["top"].set_visible(False)
    ax2.spines["left"].set_visible(False)
    if flat:
        ax2.set_ylim(0, max(mfg_values) * 2.5)
    return ax2

# Pareto data shared by individual fig4 and combined figure
S1_RUNS = [
    [(11729,273),(10575,293),(7368,433),(17239,293),(17239,293)],
    [(11729,273),(17239,293),(6916,507),(12326,456),(7613,500)],
    [(17239,293),(11729,273),(10575,293),(6916,507),(7663,559)],
]
S2_RUNS = [
    [(16567,423),(7703,661),(13394,403),(8331,689),(11307,413)],
    [(13394,403),(16567,423),(8258,713),(7703,661),(13656,617)],
    [(13394,403),(18996,485),(7703,661),(7959,724),(13476,681)],
]

plt.rcParams.update({
    "font.family":       "sans-serif",
    "font.size":         9,
    "axes.spines.top":   False,
    "axes.spines.right": False,
    "axes.linewidth":    0.7,
    "axes.grid":         True,
    "grid.linewidth":    0.4,
    "grid.color":        "#E2E8F0",
    "grid.alpha":        1.0,
    "xtick.major.width": 0.7,
    "ytick.major.width": 0.7,
    "xtick.labelsize":   8,
    "ytick.labelsize":   8,
    "legend.fontsize":   8,
    "legend.frameon":    False,
})


def _pareto_panel(ax, runs, title):
    for ri, (run, color) in enumerate(zip(runs, RUN_COLORS)):
        pts = sorted(run, key=lambda p: p[0])
        ax.plot([p[0] for p in pts], [p[1] for p in pts],
                color=color, linewidth=0.8, alpha=0.3, zorder=2)
        ax.scatter([p[0] for p in run], [p[1] for p in run],
                   color=color, s=22, zorder=3, alpha=0.75,
                   label=RUN_LABELS[ri])
    ax.set_title(title, fontsize=7.5, pad=5, loc="left")
    ax.set_xlabel("transport cost (km-eq)", fontsize=7.5)
    ax.set_ylabel("mfg cost (€)", fontsize=7.5)
    ax.xaxis.set_major_formatter(plt.FuncFormatter(lambda x, _: f"{x/1000:.0f}k"))


# ── FIG 1: Actor pool size (A dimension) ─────────────────────────────────────
fig, ax = plt.subplots(figsize=(3.5, 2.6))

lA = ["1", "2", "4", "6", "8"]
vA = [1.66, 2.25, 24.62, 13.48, 11.45]
cA = [AMBER if i == 2 else BLUE for i in range(len(vA))]
bars = ax.bar(lA, vA, color=cA, width=0.55, zorder=3)
ax.bar_label(bars, [f"{v:.1f}s" if i == 2 else "" for i, v in enumerate(vA)],
             padding=3, fontsize=8, fontweight="bold", color=AMBER)
ax.set_xlabel("eligible actors")
ax.set_ylabel("avg wall time (s)")
ax.set_ylim(0, 28)
ax.yaxis.grid(True, zorder=0)
ax.set_axisbelow(True)
_cost_overlay(ax, range(len(lA)), MFG_A)
fig.tight_layout()
fig.savefig(os.path.join(OUT, "fig1_actor_pool.svg"), format="svg", bbox_inches="tight")
plt.close()
print("fig1_actor_pool.svg")


# ── FIG 2: Process count (P dimension) ───────────────────────────────────────
fig, ax = plt.subplots(figsize=(3.5, 2.6))

lP = ["P₂", "P₄", "P₅", "P₇", "P₉"]
vP = [10.50, 17.21, 11.37, 10.40, 10.90]
cP = [AMBER if i == 1 else BLUE for i in range(len(vP))]
bars = ax.bar(lP, vP, color=cP, width=0.55, zorder=3)
ax.bar_label(bars, [f"{v:.1f}s" if i == 1 else "" for i, v in enumerate(vP)],
             padding=3, fontsize=8, fontweight="bold", color=AMBER)
ax.set_xlabel("scenario")
ax.set_ylabel("avg wall time (s)")
ax.set_ylim(0, 20)
ax.yaxis.grid(True, zorder=0)
ax.set_axisbelow(True)
_cost_overlay(ax, range(len(lP)), MFG_P)
fig.tight_layout()
fig.savefig(os.path.join(OUT, "fig2_process_count.svg"), format="svg", bbox_inches="tight")
plt.close()
print("fig2_process_count.svg")


# ── FIG 3: Parallel routes — post-fix only ───────────────────────────────────
fig, ax = plt.subplots(figsize=(4.5, 2.6))

xR = [1, 2, 4, 8]
yR = [11.03, 11.56, 12.41, 12.94]
ax.plot(xR, yR, color=BLUE, linewidth=2, marker="o", markersize=5, zorder=3)
for xi, yi in zip(xR, yR):
    ax.annotate(f"{yi:.1f}s", xy=(xi, yi), xytext=(0, 8),
                textcoords="offset points", ha="center",
                fontsize=7.5, fontweight="bold", color=BLUE)
ax.set_xlabel("concurrent routes")
ax.set_ylabel("avg wall time (s)")
ax.set_xticks(xR)
ax.set_ylim(0, 18)
ax.yaxis.grid(True, zorder=0)
ax.set_axisbelow(True)
_cost_overlay(ax, xR, MFG_R, flat=True)
fig.tight_layout()
fig.savefig(os.path.join(OUT, "fig3_parallel_routes.svg"), format="svg", bbox_inches="tight")
plt.close()
print("fig3_parallel_routes.svg")


# ── FIG 4: Pareto fronts — S1 and S2 (fork-join strategies) ──────────────────
fig, axes = plt.subplots(1, 2, figsize=(7.0, 3.0))
_pareto_panel(axes[0], S1_RUNS, "S1 — Refurbishment\n(fork-join, 3 branches)")
_pareto_panel(axes[1], S2_RUNS, "S2 — Functional Upgrade\n(depth-2 branching)")
handles = [mpatches.Patch(color=c, label=l) for c, l in zip(RUN_COLORS, RUN_LABELS)]
fig.legend(handles=handles, loc="lower center", ncol=3,
           bbox_to_anchor=(0.5, -0.02), fontsize=8, frameon=False)
fig.tight_layout(rect=[0, 0.06, 1, 1])
fig.savefig(os.path.join(OUT, "fig4_pareto_fronts.svg"), format="svg", bbox_inches="tight")
plt.close()
print("fig4_pareto_fronts.svg")


# ── FIG: Performance (a, b, c) — 3-row × 1-col bar charts ───────────────────
xR_labels = ["1", "2", "4", "8"]

fig, (ax_a, ax_b, ax_c) = plt.subplots(3, 1, figsize=(3.5, 7.5))

# (a) Actor pool size
bars = ax_a.bar(lA, vA, color=cA, width=0.55, zorder=3)
ax_a.bar_label(bars, [f"{v:.1f}s" if i == 2 else "" for i, v in enumerate(vA)],
               padding=3, fontsize=8, fontweight="bold", color=AMBER)
ax_a.set_xlabel("eligible actors")
ax_a.set_ylabel("avg wall time (s)")
ax_a.set_ylim(0, 28)
ax_a.yaxis.grid(True, zorder=0)
ax_a.set_axisbelow(True)
ax_a.set_title("(a) Actor pool size", fontsize=9, loc="left", pad=4)
_cost_overlay(ax_a, range(len(lA)), MFG_A)

# (b) Process count
bars = ax_b.bar(lP, vP, color=cP, width=0.55, zorder=3)
ax_b.bar_label(bars, [f"{v:.1f}s" if i == 1 else "" for i, v in enumerate(vP)],
               padding=3, fontsize=8, fontweight="bold", color=AMBER)
ax_b.set_xlabel("scenario")
ax_b.set_ylabel("avg wall time (s)")
ax_b.set_ylim(0, 20)
ax_b.yaxis.grid(True, zorder=0)
ax_b.set_axisbelow(True)
ax_b.set_title("(b) Process count", fontsize=9, loc="left", pad=4)
_cost_overlay(ax_b, range(len(lP)), MFG_P)

# (c) Parallel routes — bar chart for consistency with (a) and (b)
bars = ax_c.bar(xR_labels, yR, color=BLUE, width=0.55, zorder=3)
ax_c.bar_label(bars, [f"{v:.1f}s" for v in yR],
               padding=3, fontsize=8, fontweight="bold", color=BLUE)
ax_c.set_xlabel("concurrent routes")
ax_c.set_ylabel("avg wall time (s)")
ax_c.set_ylim(0, 18)
ax_c.yaxis.grid(True, zorder=0)
ax_c.set_axisbelow(True)
ax_c.set_title("(c) Parallel routes", fontsize=9, loc="left", pad=4)
_cost_overlay(ax_c, range(len(xR_labels)), MFG_R, flat=True)

fig.tight_layout(pad=1.4)
fig.savefig(os.path.join(OUT, "fig_performance.svg"), format="svg", bbox_inches="tight")
plt.close()
print("fig_performance.svg")


# ── FIG: Strategies (d, e) — S1 and S2 Pareto, horizontal ───────────────────
fig, (ax_d, ax_e) = plt.subplots(1, 2, figsize=(7.0, 3.2))
_pareto_panel(ax_d, S1_RUNS, "(a) S1 — Refurbishment (fork-join, 3 branches)")
_pareto_panel(ax_e, S2_RUNS, "(b) S2 — Functional Upgrade (depth-2 branching)")
handles = [mpatches.Patch(color=c, label=l) for c, l in zip(RUN_COLORS, RUN_LABELS)]
fig.legend(handles=handles, loc="lower center", ncol=3,
           bbox_to_anchor=(0.5, -0.02), fontsize=8, frameon=False)
fig.tight_layout(rect=[0, 0.06, 1, 1])
fig.savefig(os.path.join(OUT, "fig_strategies.svg"), format="svg", bbox_inches="tight")
plt.close()
print("fig_strategies.svg")

print(f"\nAll figures written to {OUT}/")
