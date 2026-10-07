#!/bin/bash
# Emülatörde uygulamayı açar, ana ekranları gezer, ekran görüntüsü ve çökme günlüğü toplar.
set -x
OUT=smoke
mkdir -p $OUT
PKG=com.eirahoutmoss.nexnews
APK=android/app/build/outputs/apk/debug/app-debug.apk
T="python3 android/smoke/tap.py"
shot() { sleep ${2:-3}; adb exec-out screencap -p > "$OUT/$1.png"; }

adb install -r "$APK" || exit 0
adb shell pm grant $PKG android.permission.POST_NOTIFICATIONS || true
adb logcat -c
adb shell am start -n $PKG/.MainActivity
sleep 20; shot 00-acilis 1
sleep 70; shot 01-akis 1
adb shell input swipe 540 1700 540 700 300; shot 02-akis-asagi
adb shell input swipe 540 1700 540 700 300; shot 03-akis-asagi2
adb shell input swipe 540 700 540 1900 200; adb shell input swipe 540 700 540 1900 200; sleep 2
# ilk "kaynak" etiketli habere dokun (olay sayfası)
$T "kaynak" 0 contains; shot 04-olay 6
adb shell input swipe 540 1700 540 600 300; shot 05-olay-asagi
adb shell input keyevent KEYCODE_BACK; sleep 2
$T "Kategoriler"; shot 06-kategoriler
$T "Türkiye"; shot 07-kategori 4
adb shell input keyevent KEYCODE_BACK; sleep 2
$T "Keşfet"; shot 08-radyo 10
$T "Radyo" 0; sleep 1
python3 - <<'PY'
import subprocess
# listedeki ilk istasyona dokun (sekme çubuğunun altındaki ilk satır)
subprocess.run(["adb","shell","input","tap","540","620"])
PY
shot 09-radyo-caliyor 12
$T "Canlı TV"; shot 10-tv 4
$T "TRT Haber"; shot 11-tv-oynatici 20
adb shell input keyevent KEYCODE_BACK; shot 12-mini-oynatici 4
$T "Hava"; shot 13-hava 12
$T "Piyasa"; shot 14-piyasa 15
$T "Kaydedilen"; shot 15-kaydedilen
$T "Ayarlar"; shot 16-ayarlar
adb shell input swipe 540 1700 540 500 300; shot 17-ayarlar-asagi
$T "Kaynaklar"; shot 18-kaynaklar 4
adb shell input keyevent KEYCODE_BACK; sleep 2
$T "Akış"; sleep 2
$T "Ara"; sleep 2
adb shell input text "deprem"; adb shell input keyevent KEYCODE_ENTER; shot 19-arama 5
adb shell input keyevent KEYCODE_BACK; adb shell input keyevent KEYCODE_BACK; sleep 2
# karanlık tema
adb shell cmd uimode night yes; shot 20-koyu 5
adb shell cmd uimode night no; shot 21-acik 5
# widget yapılandırması ve arka plan işi kontrolü
adb shell dumpsys jobscheduler | grep -i -A2 nexnews | head -40 > $OUT/jobs.txt
adb logcat -d > $OUT/logcat-tam.txt
grep -E "FATAL EXCEPTION|AndroidRuntime|$PKG|GoLog|nexnews" $OUT/logcat-tam.txt | head -400 > $OUT/logcat-ozet.txt
grep -c "FATAL EXCEPTION" $OUT/logcat-tam.txt > $OUT/cokme-sayisi.txt || true
exit 0
