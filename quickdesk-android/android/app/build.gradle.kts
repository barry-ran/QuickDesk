plugins {
    id("com.android.application")
    // The Flutter Gradle Plugin must be applied after the Android and Kotlin Gradle plugins.
    id("dev.flutter.flutter-gradle-plugin")
}

android {
    namespace = "com.quickcoder.quickdesk_android"
    // 部分插件（flutter_plugin_android_lifecycle 等）要求 compileSdk >= 36
    compileSdk = maxOf(36, flutter.compileSdkVersion)
    ndkVersion = flutter.ndkVersion

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }

    // 被控端用到 GestureDescription（API 24）、MotionEvent 注入与 Shizuku。
    buildFeatures {
        aidl = true
    }

    defaultConfig {
        applicationId = "com.quickcoder.quickdesk_android"
        // 被控输入（无障碍手势 / Shizuku 注入）要求 API 24+
        minSdk = maxOf(24, flutter.minSdkVersion)
        targetSdk = flutter.targetSdkVersion
        versionCode = flutter.versionCode
        versionName = flutter.versionName
    }

    buildTypes {
        release {
            // TODO: Add your own signing config for the release build.
            // Signing with the debug keys for now, so `flutter run --release` works.
            signingConfig = signingConfigs.getByName("debug")
        }
    }
}

kotlin {
    compilerOptions {
        jvmTarget = org.jetbrains.kotlin.gradle.dsl.JvmTarget.JVM_17
    }
}

dependencies {
    // Shizuku 增强档：通过 shell(adb) 权限的 UserService 注入真实输入事件
    val shizukuVersion = "13.1.5"
    implementation("dev.rikka.shizuku:api:$shizukuVersion")
    implementation("dev.rikka.shizuku:provider:$shizukuVersion")
}

flutter {
    source = "../.."
}
