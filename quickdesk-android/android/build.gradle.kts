import com.android.build.gradle.BaseExtension

allprojects {
    repositories {
        google()
        mavenCentral()
    }
}

// 统一把所有 Android 插件子模块的 compileSdk 提到 36
// （flutter_plugin_android_lifecycle 等依赖要求），否则 checkAarMetadata 失败。
subprojects {
    afterEvaluate {
        extensions.findByType(BaseExtension::class.java)?.apply {
            if (compileSdkVersion == null ||
                (compileSdkVersion?.substringAfter("android-")?.toIntOrNull() ?: 0) < 36
            ) {
                compileSdkVersion(36)
            }
        }
    }
}

val newBuildDir: Directory =
    rootProject.layout.buildDirectory
        .dir("../../build")
        .get()
rootProject.layout.buildDirectory.value(newBuildDir)

subprojects {
    val newSubprojectBuildDir: Directory = newBuildDir.dir(project.name)
    project.layout.buildDirectory.value(newSubprojectBuildDir)
}
subprojects {
    project.evaluationDependsOn(":app")
}

tasks.register<Delete>("clean") {
    delete(rootProject.layout.buildDirectory)
}
