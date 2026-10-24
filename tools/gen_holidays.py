#!/usr/bin/env python3
"""NexNews resmî tatil/bayram verisi üretici (RC98).

python-holidays (MIT) kütüphanesinden ülke bazında resmî tatilleri, arayüz
dillerinde (tr, en, ar, de, fr, es, ru) mümkün olduğunca yerelleştirilmiş adlarla
core/assets/holidays.json'a yazar. Uygulama bu dosyayı gömülü taşır; ağ gerekmez.

Kullanım: pip install holidays && python3 tools/gen_holidays.py 2026 2028
"""
import json, sys, datetime
import holidays

UI = ["tr", "en", "ar", "de", "fr", "es", "ru"]

def main():
    y0, y1 = (int(sys.argv[1]), int(sys.argv[2])) if len(sys.argv) > 2 else (datetime.date.today().year, datetime.date.today().year + 2)
    years = list(range(y0, y1 + 1))
    sup = holidays.list_supported_countries()
    loc = holidays.list_localized_countries()
    out = {"years": [y0, y1], "src": "python-holidays " + holidays.__version__, "c": {}}
    for code in sorted(c for c in sup if len(c) == 2):
        langs = loc.get(code, [])
        langmap = {}  # ui lang -> library lang
        for ui in UI:
            for l in langs:
                if l == ui or l.split("_")[0] == ui:
                    langmap[ui] = l
                    break
        try:
            base = holidays.country_holidays(code, years=years)
        except Exception as e:
            print("skip", code, e, file=sys.stderr)
            continue
        per = {}
        for ui, l in langmap.items():
            per[ui] = holidays.country_holidays(code, years=years, language=l)
        names, idx, days = [], {}, []
        for d, n in sorted(base.items()):
            rec = {"_": n}
            for ui, h in per.items():
                v = h.get(d)
                if v and v != n:
                    rec[ui] = v
            # Aynı günde birden çok tatil "; " ile birleşik gelir; olduğu gibi kalır.
            key = json.dumps(rec, ensure_ascii=False, sort_keys=True)
            if key not in idx:
                idx[key] = len(names)
                names.append(rec)
            days.append([d.isoformat(), idx[key]])
        if days:
            out["c"][code] = {"n": names, "d": days}
    json.dump(out, open("core/assets/holidays.json", "w", encoding="utf-8"), ensure_ascii=False, separators=(",", ":"))
    print("countries", len(out["c"]))

if __name__ == "__main__":
    main()
