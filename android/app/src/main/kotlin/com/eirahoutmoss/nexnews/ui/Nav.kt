package com.eirahoutmoss.nexnews.ui

import android.content.Context
import android.net.Uri
import androidx.navigation.NavController
import com.eirahoutmoss.nexnews.Core
import com.eirahoutmoss.nexnews.NewsItem

/** Ekranlar arası geçişler tek yerde. */
class NexNav(private val nav: NavController, private val ctx: Context) {
    fun back() { nav.popBackStack() }
    fun story(link: String) = nav.navigate("story/" + Uri.encode(link))
    fun reader(link: String) = nav.navigate("reader/" + Uri.encode(link))
    fun category(name: String) = nav.navigate("category/" + Uri.encode(name))
    fun search() = nav.navigate("search")
    fun player() = nav.navigate("player")
    fun sources() = nav.navigate("sources")
    fun weekly() = nav.navigate("weekly")

    /** Çok kaynaklı haber → olay sayfası; tek kaynak → okuyucu ya da tarayıcı. */
    fun open(item: NewsItem) {
        Core.markRead(item)
        if (item.count > 1 || item.sources.size > 1) story(item.link) else article(item.link)
    }

    fun article(link: String) {
        if (Core.configBool("openInApp", true)) reader(link) else openInBrowser(ctx, link)
    }
}
