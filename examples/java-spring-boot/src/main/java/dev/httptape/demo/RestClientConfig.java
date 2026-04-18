package dev.httptape.demo;

import org.springframework.beans.factory.annotation.Value;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.web.client.RestClient;

/**
 * Provides a pre-configured {@link RestClient} bean with the external API
 * base URL injected from application properties.
 *
 * <p>In tests, {@code app.external-api.base-url} is overridden via
 * {@code @DynamicPropertySource} to point at the httptape Testcontainer.
 */
@Configuration
public class RestClientConfig {

    @Bean
    public RestClient restClient(@Value("${app.external-api.base-url}") String baseUrl) {
        return RestClient.builder()
                .baseUrl(baseUrl)
                .build();
    }
}
