package dev.httptape.demo;

import org.springframework.boot.test.context.TestConfiguration;
import org.springframework.context.annotation.Bean;
import org.springframework.test.context.DynamicPropertyRegistrar;
import org.testcontainers.containers.GenericContainer;
import org.testcontainers.containers.wait.strategy.Wait;
import org.testcontainers.utility.MountableFile;

/**
 * Shared Testcontainers configuration for the dev runner ({@link TestApplication})
 * and all integration tests.
 *
 * <p>Strategy: a single httptape container with all fixtures (OpenAI + users)
 * copied into one flat {@code /fixtures} directory. Both the Spring AI ChatClient
 * and the REST UserService point at the same container. This keeps startup fast
 * and simple -- one container per JVM, not per test class.
 *
 * <p>Integration tests import this configuration via
 * {@code @Import(TestcontainersConfig.class)}. The container is started once and
 * shared across all test classes within the same Spring application context.
 */
@TestConfiguration(proxyBeanMethods = false)
class TestcontainersConfig {

    /**
     * A single httptape container serving all fixtures (OpenAI SSE + REST users)
     * with realtime SSE timing for a realistic streaming experience.
     */
    @Bean
    GenericContainer<?> httptapeContainer() {
        return new GenericContainer<>("ghcr.io/vibewarden/httptape:0.10.1")
                .withCommand("serve", "--fixtures", "/fixtures", "--sse-timing=realtime")
                .withExposedPorts(8081)
                .withCopyFileToContainer(
                        MountableFile.forClasspathResource("fixtures/openai/chat-completion-headphones.json"),
                        "/fixtures/chat-completion-headphones.json"
                )
                .withCopyFileToContainer(
                        MountableFile.forClasspathResource("fixtures/users/get-user-1.json"),
                        "/fixtures/get-user-1.json"
                )
                .withCopyFileToContainer(
                        MountableFile.forClasspathResource("fixtures/users/get-users.json"),
                        "/fixtures/get-users.json"
                )
                .withCopyFileToContainer(
                        MountableFile.forClasspathResource("fixtures/users/get-user-999.json"),
                        "/fixtures/get-user-999.json"
                )
                .waitingFor(Wait.forHttp("/").forStatusCode(404));
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
