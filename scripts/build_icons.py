#!/usr/bin/env python3
"""Generate all Tauri icon sizes from master PNG."""
import os, subprocess, sys
from PIL import Image

SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
MASTER = os.path.join(SCRIPT_DIR, "icon_master.png")
ICON_DIR = os.path.join(SCRIPT_DIR, "..", "desktop", "src-tauri", "icons")

if not os.path.isfile(MASTER):
    print(f"Master icon not found: {MASTER}")
    sys.exit(1)

os.makedirs(ICON_DIR, exist_ok=True)
master = Image.open(MASTER).convert("RGBA")

# Tauri required sizes
sizes = {
    "32x32.png": 32,
    "64x64.png": 64,
    "128x128.png": 128,
    "128x128@2x.png": 256,
    "icon.png": 512,
}

# Windows sizes
win_sizes = {
    "Square30x30Logo.png": 30,
    "Square44x44Logo.png": 44,
    "Square71x71Logo.png": 71,
    "Square89x89Logo.png": 89,
    "Square107x107Logo.png": 107,
    "Square142x142Logo.png": 142,
    "Square150x150Logo.png": 150,
    "Square284x284Logo.png": 284,
    "Square310x310Logo.png": 310,
    "StoreLogo.png": 50,
}

all_sizes = {**sizes, **win_sizes}

for name, size in all_sizes.items():
    resized = master.resize((size, size), Image.LANCZOS)
    path = os.path.join(ICON_DIR, name)
    resized.save(path, "PNG")
    print(f"  {name} ({size}x{size})")

# macOS .icns via iconset
import tempfile, shutil
tmpdir = tempfile.mkdtemp()
iconset = os.path.join(tmpdir, "icon.iconset")
os.makedirs(iconset)

icns_sizes = [
    ("icon_16x16.png", 16),
    ("icon_16x16@2x.png", 32),
    ("icon_32x32.png", 32),
    ("icon_32x32@2x.png", 64),
    ("icon_128x128.png", 128),
    ("icon_128x128@2x.png", 256),
    ("icon_256x256.png", 256),
    ("icon_256x256@2x.png", 512),
    ("icon_512x512.png", 512),
    ("icon_512x512@2x.png", 1024),
]

for fname, s in icns_sizes:
    resized = master.resize((s, s), Image.LANCZOS)
    resized.save(os.path.join(iconset, fname), "PNG")

icns_path = os.path.join(ICON_DIR, "icon.icns")
subprocess.run(["iconutil", "-c", "icns", iconset, "-o", icns_path], check=True)
print(f"  icon.icns")

# Windows .ico
ico_sizes = [16, 24, 32, 48, 64, 128, 256]
ico_imgs = [master.resize((s, s), Image.LANCZOS) for s in ico_sizes]
ico_path = os.path.join(ICON_DIR, "icon.ico")
ico_imgs[0].save(ico_path, format="ICO", sizes=[(s, s) for s in ico_sizes], append_images=ico_imgs[1:])
print(f"  icon.ico")

shutil.rmtree(tmpdir)
print(f"\nDone! All icons in {ICON_DIR}")
