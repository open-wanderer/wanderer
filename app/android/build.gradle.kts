allprojects {
    repositories {
        google()
        mavenCentral()
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

// receive_sharing_intent 1.9.0 hardcodes compileSdk 37, but Google only
// publishes "platforms;android-37.0" (an unfinalized point release) under a
// differently-named target hash — Gradle can't resolve "android-37" against
// it. Pin this one subproject to the highest installed stable platform.
subprojects {
    if (project.name == "receive_sharing_intent") {
        afterEvaluate {
            extensions.findByName("android")?.withGroovyBuilder {
                setProperty("compileSdk", 36)
            }
        }
    }
}

tasks.register<Delete>("clean") {
    delete(rootProject.layout.buildDirectory)
}
