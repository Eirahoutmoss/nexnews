plugins {
    id("com.android.application")
    id("org.jetbrains.kotlin.android")
    id("org.jetbrains.kotlin.plugin.compose")
    id("org.jetbrains.kotlin.plugin.serialization")
}

// Sürüm bilgisi CI'dan gelir (SURUM dosyası → versionName, çalıştırma numarası → versionCode).
val nexVersionName = System.getenv("NEXNEWS_VERSION_NAME") ?: "0.1.0-yerel"
val nexVersionCode = System.getenv("NEXNEWS_VERSION_CODE")?.toIntOrNull() ?: 1
val keystoreFile: String? = System.getenv("NEXNEWS_KEYSTORE_FILE")

android {
    namespace = "com.eirahoutmoss.nexnews"
    compileSdk = 35

    defaultConfig {
        applicationId = "com.eirahoutmoss.nexnews"
        minSdk = 24
        targetSdk = 35
        versionCode = nexVersionCode
        versionName = nexVersionName
        ndk { abiFilters += listOf("arm64-v8a", "armeabi-v7a") }
    }

    signingConfigs {
        create("release") {
            if (keystoreFile != null) {
                storeFile = file(keystoreFile)
                storePassword = System.getenv("NEXNEWS_KEYSTORE_PASSWORD")
                keyAlias = System.getenv("NEXNEWS_KEY_ALIAS")
                keyPassword = System.getenv("NEXNEWS_KEYSTORE_PASSWORD")
            }
        }
    }

    buildTypes {
        release {
            isMinifyEnabled = true
            isShrinkResources = true
            proguardFiles(getDefaultProguardFile("proguard-android-optimize.txt"), "proguard-rules.pro")
            // Anahtar yoksa hata ayıklama anahtarıyla imzalanır (deneme sürümü).
            signingConfig = if (keystoreFile != null) signingConfigs.getByName("release")
            else signingConfigs.getByName("debug")
        }
    }

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }

    buildFeatures {
        compose = true
        buildConfig = true
    }

    packaging {
        resources {
            excludes += "/META-INF/{AL2.0,LGPL2.1}"
        }
        jniLibs { useLegacyPackaging = false }
    }
}

kotlin {
    jvmToolchain(17)
}

dependencies {
    // Go çekirdeği (gomobile bind ile CI'da üretilir: libs/nexcore.aar)
    implementation(fileTree(mapOf("dir" to "libs", "include" to listOf("*.aar"))))

    implementation(platform("androidx.compose:compose-bom:2025.05.01"))
    implementation("androidx.compose.ui:ui")
    implementation("androidx.compose.foundation:foundation")
    implementation("androidx.compose.material3:material3")
    implementation("androidx.compose.material:material-icons-extended")
    implementation("androidx.activity:activity-compose:1.10.1")
    implementation("androidx.core:core-ktx:1.16.0")
    implementation("androidx.lifecycle:lifecycle-runtime-compose:2.9.0")
    implementation("androidx.lifecycle:lifecycle-process:2.9.0")
    implementation("androidx.navigation:navigation-compose:2.9.0")
    implementation("androidx.work:work-runtime-ktx:2.10.1")
    implementation("androidx.browser:browser:1.8.0")
    implementation("androidx.glance:glance-appwidget:1.1.1")
    implementation("androidx.media3:media3-exoplayer:1.6.1")
    implementation("androidx.media3:media3-exoplayer-hls:1.6.1")
    implementation("androidx.media3:media3-session:1.6.1")
    implementation("androidx.media3:media3-ui:1.6.1")
    implementation("io.coil-kt:coil-compose:2.7.0")
    implementation("org.jetbrains.kotlinx:kotlinx-serialization-json:1.8.1")
    implementation("org.jetbrains.kotlinx:kotlinx-coroutines-android:1.10.2")
    implementation("org.jetbrains.kotlinx:kotlinx-coroutines-guava:1.10.2")
}
