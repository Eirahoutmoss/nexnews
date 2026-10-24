package core

// Android köprüsü (RC96): arayüz dili/sözlüğü, sahiplik kataloğu, kullanıcı
// kanal listeleri ve haftanın özeti.

import (
	"context"
	"encoding/json"
	"time"
)

// MobileUILang: arayüz dili kodu (tr, en, ar, de, fr, es, ru; sözlüğü olmayan → en).
func MobileUILang() string { return uiLangCode() }

// MobileI18n: seçili dilin sözlüğü (Türkçe metin → çeviri) JSON; Türkçe için "{}".
func MobileI18n() string {
	l := uiLangCode()
	if l == "tr" {
		return "{}"
	}
	i18nOnce.Do(loadI18nDict)
	m := map[string]string{}
	for k, v := range i18nDict["en"] {
		m[k] = v
	}
	for k, v := range i18nDict[l] {
		m[k] = v
	}
	b, _ := json.Marshal(m)
	return string(b)
}

// MobileOwnership: kaynak sahipliği kataloğu.
func MobileOwnership() string { return string(ownershipJSON()) }

// MobileLiveLists: kullanıcının kanal listeleri.
func MobileLiveLists() string { return liveListsJSON() }

// MobileLiveListAdd: listeyi adresinden indirir (url) ya da metinden (text) ekler.
func MobileLiveListAdd(name, url, text string) (string, error) {
	b, err := liveListAdd(context.Background(), name, url, text)
	return string(b), err
}

// MobileLiveListRefresh: listeyi yalnızca istenince kaynağından yeniler.
func MobileLiveListRefresh(id string) (string, error) {
	b, err := liveListRefresh(context.Background(), id)
	return string(b), err
}

func MobileLiveListDelete(id string) error       { return liveListDelete(id) }
func MobileLiveListRename(id, name string) error { return liveListRename(id, name) }
func MobileLiveBuiltin(on bool) error            { return liveSetBuiltin(on) }

// ---- RC97 ----

// MobileMirror: okuma aynası (son 30 gün).
func MobileMirror() string { b, _ := readingMirrorJSON(); return string(b) }

// MobileAgenda: önümüzdeki 14 günün gündem takvimi.
func MobileAgenda() string { b, _ := agendaJSON(14); return string(b) }

// MobileAudioBrief: sesli brifing metni.
func MobileAudioBrief() string {
	if !mobileReady() {
		return `{"text":"","items":[]}`
	}
	b, _ := store.audioBriefJSON()
	return string(b)
}

// MobileContext: olayın bağlam kartı (Vikipedi özetleri + önceki olaylar; ağ kullanır).
func MobileContext(link string) string {
	if !mobileReady() {
		return `{"entities":[],"related":[]}`
	}
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	b, _ := store.contextJSON(ctx, link)
	return string(b)
}
