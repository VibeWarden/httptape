package dev.httptape.demo;

import java.io.IOException;
import java.util.HashMap;
import java.util.Map;

import org.springframework.boot.test.context.TestConfiguration;
import org.springframework.context.annotation.Bean;
import org.springframework.core.io.Resource;
import org.springframework.core.io.support.PathMatchingResourcePatternResolver;
import org.springframework.test.context.DynamicPropertyRegistrar;
import org.testcontainers.containers.GenericContainer;
import org.testcontainers.containers.wait.strategy.Wait;
import org.testcontainers.utility.MountableFile;

/**
 * Shared Testcontainers configuration for the dev runner ({@link TestApplication})
 * and all integration tests.
 *
 * <p>Strategy: a single httptape container serving fixtures auto-discovered
 * from the classpath. All {@code .json} files under
 * {@code src/test/resources/fixtures/**} are copied into the container's
 * flat {@code /fixtures} directory at bean-construction time. Drop a new
 * fixture in the resources tree and it is picked up automatically -- no
 * code changes required.
 *
 * <p>One container per JVM (not per test class). Both the Spring AI ChatClient
 * and the REST UserService point at the same container.
 *
 * <p>Integration tests import this configuration via
 * {@code @Import(TestcontainersConfig.class)}.
 */
@TestConfiguration(proxyBeanMethods = false)
class TestcontainersConfig {

    /**
     * A single httptape container serving every {@code .json} fixture found
     * on the classpath under {@code fixtures/**}. Realtime SSE timing for a
     * realistic streaming experience.
     */
    @Bean
    GenericContainer<?> httptapeContainer() throws IOException {
        GenericContainer<?> container = new GenericContainer<>("ghcr.io/vibewarden/httptape:0.10.1")
                .withCommand("serve", "--fixtures", "/fixtures", "--sse-timing=realtime")
                .withExposedPorts(8081)
                .waitingFor(Wait.forHttp("/").forStatusCode(404));

        // Auto-discover all fixtures under classpath:fixtures/**/*.json and
        // mount each into the container's flat /fixtures dir. Httptape's
        // FileStore is flat (no recursive subdirectory scanning), so we
        // collapse subdirs (openai/, users/, ...) into one container-side dir.
        // Filename collisions across subdirs would be ambiguous -- detect and
        // fail fast at config time rather than silently overwriting.
        PathMatchingResourcePatternResolver resolver = new PathMatchingResourcePatternResolver();
        Resource[] fixtures = resolver.getResources("classpath:fixtures/**/*.json");
        Map<String, String> seen = new HashMap<>();
        for (Resource fixture : fixtures) {
            String filename = fixture.getFilename();
            if (filename == null || filename.isBlank()) {
                continue;
            }
            String previousPath = seen.put(filename, fixture.getURI().toString());
            if (previousPath != null) {
                throw new IllegalStateException(
                        "Fixture filename collision in flat /fixtures mount: '" + filename
                                + "' appears in both '" + previousPath
                                + "' and '" + fixture.getURI() + "'. "
                                + "Rename one (e.g., add a subject prefix) so filenames are unique across all subdirs."
                );
            }
            container.withCopyFileToContainer(
                    MountableFile.forHostPath(fixture.getFile().toPath()),
                    "/fixtures/" + filename
            );
        }
        return container;
    }

    /**
     * Wires the httptape container's dynamic port into the application properties
     * so that both Spring AI and the REST client point at the container.
     */
    @Bean
    DynamicPropertyRegistrar httptapeProperties(GenericContainer<?> httptapeContainer) {
        return registry -> {
            String baseUrl = "http://" + httptapeContainer.getHost()
                    + ":" + httptapeContainer.getMappedPort(8081);
            registry.add("spring.ai.openai.base-url", () -> baseUrl);
            registry.add("spring.ai.openai.api-key", () -> "sk-test-key");
            registry.add("app.external-api.base-url", () -> baseUrl);
        };
    }
}
