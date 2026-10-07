# gomobile: Java sınıfları yerel (JNI) koddan çağrılır; R8 bunları silmemeli.
-keep class go.** { *; }
-keep class com.eirahoutmoss.nexnews.go.** { *; }
-dontwarn go.**

# kotlinx.serialization model sınıfları
-keepattributes *Annotation*, InnerClasses
-keepclassmembers @kotlinx.serialization.Serializable class com.eirahoutmoss.nexnews.** {
    *** Companion;
    *** INSTANCE;
    kotlinx.serialization.KSerializer serializer(...);
}
-keep,includedescriptorclasses class com.eirahoutmoss.nexnews.**$$serializer { *; }

# Glance alıcıları manifestten açılır
-keep class com.eirahoutmoss.nexnews.widget.** { *; }
