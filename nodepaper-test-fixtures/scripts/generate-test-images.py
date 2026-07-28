from pathlib import Path
import matplotlib.pyplot as plt
import numpy as np

out = Path(__file__).resolve().parents[1] / "generated-images"
out.mkdir(exist_ok=True)

hours = np.arange(24)
demand = np.array([18,14,11,9,8,10,18,35,58,66,52,41,38,40,44,49,57,72,81,68,50,39,30,23])

plt.figure(figsize=(8, 4.5))
plt.plot(hours, demand, marker="o")
plt.xlabel("Hour")
plt.ylabel("Demand index")
plt.title("Synthetic Bicycle Demand by Hour")
plt.grid(True, alpha=0.25)
plt.tight_layout()
plt.savefig(out / "demand-trend.png", dpi=160)
plt.close()

rng = np.random.default_rng(20260727)
x = rng.uniform(0, 10, 24)
y = rng.uniform(0, 8, 24)
sizes = rng.integers(30, 180, 24)

plt.figure(figsize=(6, 5))
plt.scatter(x, y, s=sizes, alpha=0.7)
for i in range(6):
    plt.annotate(f"S{i+1}", (x[i], y[i]), xytext=(4, 4), textcoords="offset points")
plt.xlabel("East-west coordinate")
plt.ylabel("North-south coordinate")
plt.title("Synthetic Station Distribution")
plt.grid(True, alpha=0.25)
plt.tight_layout()
plt.savefig(out / "station-map.png", dpi=160)
plt.close()
