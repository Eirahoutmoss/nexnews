#!/usr/bin/env python3
"""Ekrandaki bir metne (ya da içerik açıklamasına) uiautomator ile dokunur."""
import re, subprocess, sys, time

def dump():
    subprocess.run(["adb", "shell", "uiautomator", "dump", "/sdcard/ui.xml"], capture_output=True)
    return subprocess.run(["adb", "shell", "cat", "/sdcard/ui.xml"], capture_output=True, text=True).stdout

def tap(label, index=0, contains=False):
    for _ in range(3):
        xml = dump()
        hits = []
        for m in re.finditer(r'<node [^>]*>', xml):
            n = m.group(0)
            t = re.search(r' text="([^"]*)"', n)
            d = re.search(r' content-desc="([^"]*)"', n)
            vals = [v.group(1) for v in (t, d) if v]
            ok = any((label in v) if contains else (v == label) for v in vals)
            if ok:
                b = re.search(r'bounds="\[(\d+),(\d+)\]\[(\d+),(\d+)\]"', n)
                if b:
                    x1, y1, x2, y2 = map(int, b.groups())
                    hits.append(((x1 + x2) // 2, (y1 + y2) // 2))
        if len(hits) > index:
            x, y = hits[index]
            subprocess.run(["adb", "shell", "input", "tap", str(x), str(y)])
            print(f"tap '{label}' -> {x},{y}")
            return True
        time.sleep(2)
    print(f"bulunamadı: '{label}'")
    return False

if __name__ == "__main__":
    label = sys.argv[1]
    idx = int(sys.argv[2]) if len(sys.argv) > 2 else 0
    contains = len(sys.argv) > 3 and sys.argv[3] == "contains"
    tap(label, idx, contains)
