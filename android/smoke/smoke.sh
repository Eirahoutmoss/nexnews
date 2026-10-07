#!/bin/bash
# Emülatörde uygulamayı açar, ana ekranları gezer, ekran görüntüsü ve çökme günlüğü toplar.
set -x
OUT=smoke
mkdir -p $OUT
PKG=com.eirahoutmoss.nexnews
APK=android/app/build/outputs/apk/debug/app-debug.apk
T="tapd"
# Sistem "yanıt vermiyor" pencerelerini kapat (emülatör açılışında launcher ANR veriyor)
dis() {
  adb shell am broadcast -a android.intent.action.CLOSE_SYSTEM_DIALOGS >/dev/null 2>&1
  adb shell uiautomator dump /sdcard/ui.xml >/dev/null 2>&1
  if adb shell cat /sdcard/ui.xml 2>/dev/null | grep -q "isn't responding\|yanıt vermiyor"; then
    python3 android/smoke/tap.py "Wait" 0 || python3 android/smoke/tap.py "Bekle" 0 || adb shell input keyevent KEYCODE_BACK
    sleep 2
  fi
}
fg() { adb shell am start -n $PKG/.MainActivity >/dev/null 2>&1; sleep 2; }
tapd() { dis; python3 android/smoke/tap.py "$@"; }
shot() { sleep ${2:-3}; dis; adb exec-out screencap -p > "$OUT/$1.png"; }
files() { { adb shell "run-as $PKG ls -la files/NexNews"; adb shell "run-as $PKG cat files/NexNews/$1" | head -c ${2:-3000}; } > "$OUT/$3.txt" 2>&1; }

adb shell wm size > $OUT/ekran.txt
# Ağ hazır olana kadar bekle (açılışta emülatör DNS'i bir süre çalışmıyor)
for i in $(seq 1 30); do
  adb shell ping -c 1 -W 2 feeds.bbci.co.uk >/dev/null 2>&1 && { echo "ağ hazır ($i)"; break; }
  sleep 3
done
# Launcher'ın açılış ANR'ı geçsin
sleep 30; dis; adb shell input keyevent KEYCODE_HOME; sleep 5; dis
adb install -r "$APK" || exit 0
adb shell pm grant $PKG android.permission.POST_NOTIFICATIONS || true
adb logcat -c
adb shell am start -n $PKG/.MainActivity
sleep 15; shot 00-acilis 1
sleep 75; shot 01-akis 1
files feed-cache.json 4000 cache-ilk
adb shell input swipe 540 1800 540 700 300; shot 02-akis-asagi
adb shell input swipe 540 1800 540 700 300; shot 03-akis-asagi2
adb shell input swipe 540 700 540 2000 200; adb shell input swipe 540 700 540 2000 200; sleep 2
# çok kaynaklı ilk habere dokun (olay sayfası)
$T " +2" 0 contains || $T " +1" 0 contains
shot 04-olay 6
adb shell input swipe 540 1800 540 600 300; shot 05-olay-asagi
adb shell input keyevent KEYCODE_BACK; sleep 2; fg
$T "Kategoriler" 0; shot 06-kategoriler
$T "Türkiye" 0; shot 07-kategori 5
adb shell input keyevent KEYCODE_BACK; sleep 2; fg
$T "Keşfet" 0; shot 08-radyo 12
$T "Çal" 0; shot 09-radyo-caliyor 15
$T "Canlı TV" 0; shot 10-tv 4
$T "TRT Haber" 0; shot 11-tv-oynatici 20
adb shell input keyevent KEYCODE_BACK; sleep 1; fg; shot 12-mini-oynatici 4
$T "Hava" 0; shot 13-hava 15
$T "Piyasa" 0; shot 14-piyasa 20
adb shell input swipe 540 1800 540 500 300; shot 14b-altin
fg; $T "Kaydedilen" 0; shot 15-kaydedilen
$T "Ayarlar" 0; shot 16-ayarlar
adb shell input swipe 540 1800 540 500 300; shot 17-ayarlar-asagi
$T "Kaynaklar" 0; shot 18-kaynaklar 5
adb shell input keyevent KEYCODE_BACK; sleep 2; fg
$T "Akış" 0; sleep 2
$T "Ara" 0; sleep 2
adb shell input text "Trump"; adb shell input keyevent KEYCODE_ENTER; shot 19-arama 5
adb shell input keyevent KEYCODE_BACK; sleep 1; adb shell input keyevent KEYCODE_BACK; sleep 2; fg
adb shell cmd uimode night yes; shot 20-koyu 6
adb shell cmd uimode night no; sleep 2
# ana ekran widget'ı için yapılandırma bilgisi
adb shell dumpsys appwidget | grep -i -A3 nexnews | head -20 > $OUT/widget.txt
adb shell dumpsys jobscheduler | grep -i -A2 nexnews | head -40 > $OUT/jobs.txt
files config.json 1500 config
adb shell "run-as $PKG ls -la files/NexNews" > $OUT/dosyalar.txt 2>&1
adb logcat -d > $OUT/logcat-tam.txt
grep -E "FATAL EXCEPTION|AndroidRuntime|GoLog|$PKG" $OUT/logcat-tam.txt | grep -v "WindowManager\|ActivityTaskManager\|CoreBackPreview" | head -400 > $OUT/logcat-ozet.txt
grep -c "FATAL EXCEPTION" $OUT/logcat-tam.txt > $OUT/cokme-sayisi.txt || true
exit 0
