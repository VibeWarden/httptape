package dev.httptape.demo

import org.testcontainers.containers.GenericContainer
import org.testcontainers.containers.wait.strategy.Wait
import org.testcontainers.images.builder.ImageFromDockerfile
import org.testcontainers.utility.MountableFile
import java.nio.file.Paths

/**
 * JVM-singleton httptape container shared across all test classes.
 *
 * Lazy initialization ensures the container is started exactly once per
 * JVM and reused for every test that reads [baseUrl]. The container
 * serves three fixture files and uses a declarative matcher config that
 * distinguishes the two POST /v1/chat/completions requests via
 * `body_fuzzy` on `$.messages[*].role`.
 *
 * **Build-from-source note:** The container currently builds httptape from
 * source so the demo always exercises the matcher code in this checkout.
 * This is needed while PR #191 (v0.12.0 content-type awareness) is in
 * flight -- the migrated fixture format is not readable by older published
 * images. Once v0.12.0 ships to GHCR, swap back to
 * `ghcr.io/vibewarden/httptape:0.12.0`.
 */
object HttptapeContainer {

    val instance: GenericContainer<*> by lazy {
        val repoRoot = Paths.get("../..").toAbsolutePath().normalize()
        val httptapeImage = ImageFromDockerfile("httptape-test", false)
            .withFileFromPath(".", repoRoot)
            .withFileFromPath("Dockerfile", repoRoot.resolve("Dockerfile"))

        GenericContainer(httptapeImage)
            .withCommand(
                "serve",
                "--fixtures", "/fixtures",
                "--config", "/config/httptape.config.json"
            )
            .withExposedPorts(8081)
            .waitingFor(Wait.forHttp("/").forStatusCode(404))
            .apply {
                // Mount fixtures -- flatten subdirectories into /fixtures/
                copyFixture("fixtures/openai/chat-1.json")
                copyFixture("fixtures/openai/chat-2.json")
                copyFixture("fixtures/weather/weather-berlin.json")
                // Mount matcher config
                withCopyFileToContainer(
                    MountableFile.forClasspathResource("httptape.config.json"),
                    "/config/httptape.config.json"
                )
            }
            .also { it.start() }
    }

    /** Base URL pointing at the httptape container's mapped port. */
    val baseUrl: String
        get() = "http://${instance.host}:${instance.getMappedPort(8081)}"

    private fun GenericContainer<*>.copyFixture(classpathPath: String) {
        val filename = classpathPath.substringAfterLast("/")
        withCopyFileToContainer(
            MountableFile.forClasspathResource(classpathPath),
            "/fixtures/$filename"
        )
    }
}
