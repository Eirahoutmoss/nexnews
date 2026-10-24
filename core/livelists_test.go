package core

import "testing"

func TestParseM3UList(t *testing.T) {
	txt := "#EXTM3U\n#EXTINF:-1 tvg-id=\"A.qa\" group-title=\"News\" http-referrer=\"https://r/\",Al Test\nhttps://x/a.m3u8\n#EXTINF:-1,Al Test\nhttps://y/a.m3u8\n#EXTINF:-1,B\n#EXTVLCOPT:http-user-agent=UA1\nhttp://z/b.m3u8\nrtmp://bad\n"
	ch := parseM3UList("x", txt)
	if len(ch) != 2 || len(ch[0].URLs) != 2 || ch[0].Group != "News" {
		t.Fatalf("parse: %+v", ch)
	}
	if string(ch[0].URLs[0]) != `{"r":"https://r/","u":"https://x/a.m3u8"}` || string(ch[1].URLs[0]) != `{"u":"http://z/b.m3u8","ua":"UA1"}` {
		t.Fatalf("urls: %s %s", ch[0].URLs[0], ch[1].URLs[0])
	}
}
