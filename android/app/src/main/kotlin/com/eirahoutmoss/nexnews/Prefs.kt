package com.eirahoutmoss.nexnews

import android.content.Context
import android.content.SharedPreferences
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow

/** Yalnızca Android'e özgü tercihler (haber ayarları Go config.json'da durur). */
class Prefs private constructor(private val sp: SharedPreferences) {
    companion object {
        @Volatile private var inst: Prefs? = null
        fun get(ctx: Context): Prefs = inst ?: synchronized(this) {
            inst ?: Prefs(ctx.applicationContext.getSharedPreferences("nexnews", Context.MODE_PRIVATE)).also { inst = it }
        }
    }

    private val _version = MutableStateFlow(0)
    /** Herhangi bir tercih değişince artar; Compose bunu dinleyip yeniden çizer. */
    val version: StateFlow<Int> = _version.asStateFlow()
    private fun bump() { _version.value = _version.value + 1 }

    var refreshMinutes: Int
        get() = sp.getInt("refreshMinutes", 60)
        set(v) { sp.edit().putInt("refreshMinutes", v).apply(); bump() }

    var wifiOnly: Boolean
        get() = sp.getBoolean("wifiOnly", false)
        set(v) { sp.edit().putBoolean("wifiOnly", v).apply(); bump() }

    var imagesWifiOnly: Boolean
        get() = sp.getBoolean("imagesWifiOnly", true)
        set(v) { sp.edit().putBoolean("imagesWifiOnly", v).apply(); bump() }

    /** "sistem" / "koyu" / "acik" */
    var theme: String
        get() = sp.getString("theme", "sistem") ?: "sistem"
        set(v) { sp.edit().putString("theme", v).apply(); bump() }

    /** "rahat" / "siki" */
    var density: String
        get() = sp.getString("density", "siki") ?: "siki"
        set(v) { sp.edit().putString("density", v).apply(); bump() }

    var digestOn: Boolean
        get() = sp.getBoolean("digestOn", true)
        set(v) { sp.edit().putBoolean("digestOn", v).apply(); bump() }

    var digestMorning: Int
        get() = sp.getInt("digestMorning", 8)
        set(v) { sp.edit().putInt("digestMorning", v).apply(); bump() }

    var digestEvening: Int
        get() = sp.getInt("digestEvening", 19)
        set(v) { sp.edit().putInt("digestEvening", v).apply(); bump() }

    /** Widget zemin opaklığı 0–100 (0 = tamamen saydam). */
    var widgetOpacity: Int
        get() = sp.getInt("widgetOpacity", 70)
        set(v) { sp.edit().putInt("widgetOpacity", v).apply(); bump() }

    var askedNotificationPermission: Boolean
        get() = sp.getBoolean("askedNotif", false)
        set(v) { sp.edit().putBoolean("askedNotif", v).apply() }

    var onboarded: Boolean
        get() = sp.getBoolean("onboarded", false)
        set(v) { sp.edit().putBoolean("onboarded", v).apply(); bump() }
}
