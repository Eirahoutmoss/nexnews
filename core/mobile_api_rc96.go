package core

// Android köprüsü (RC96): arayüz dili/sözlüğü, sahiplik kataloğu, kullanıcı
// kanal listeleri ve haftanın özeti.

import (
	"context"
	"encoding/json"
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
