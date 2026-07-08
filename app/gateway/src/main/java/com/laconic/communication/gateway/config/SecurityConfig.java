package com.laconic.communication.gateway.config;

import java.nio.charset.StandardCharsets;
import java.util.List;

import javax.crypto.SecretKey;
import javax.crypto.spec.SecretKeySpec;

import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.http.HttpHeaders;
import org.springframework.http.HttpMethod;
import org.springframework.security.config.annotation.method.configuration.EnableReactiveMethodSecurity;
import org.springframework.security.config.web.server.ServerHttpSecurity;
import org.springframework.security.oauth2.jwt.NimbusReactiveJwtDecoder;
import org.springframework.security.oauth2.jwt.ReactiveJwtDecoder;
import org.springframework.security.web.server.SecurityWebFilterChain;
import org.springframework.web.cors.CorsConfiguration;
import org.springframework.web.cors.reactive.CorsConfigurationSource;
import org.springframework.web.cors.reactive.UrlBasedCorsConfigurationSource;

@Configuration
@EnableReactiveMethodSecurity
public class SecurityConfig {

  @Bean
  SecurityWebFilterChain securityWebFilterChain(ServerHttpSecurity http) {
    return http
        .csrf(ServerHttpSecurity.CsrfSpec::disable)
        .cors(cors -> {})
        .authorizeExchange(exchanges -> exchanges
            .pathMatchers("/actuator/health", "/actuator/info").permitAll()
            .pathMatchers(HttpMethod.GET, "/comms/*/api/v1/healthz").permitAll()
            .pathMatchers(HttpMethod.POST, "/comms/*/api/v1/webhooks").permitAll()
            .pathMatchers(HttpMethod.GET, "/comms/*/api/v1/webhooks").permitAll()
            .pathMatchers(HttpMethod.OPTIONS, "/**").permitAll()
            .pathMatchers("/comms/*/api/v1/**").authenticated()
            .anyExchange().permitAll()
        )
        .oauth2ResourceServer(oauth2 -> oauth2.jwt(jwt -> {}))
        .build();
  }

  @Bean
  @ConditionalOnProperty(prefix = "gateway.jwt", name = "hmacSecret")
  ReactiveJwtDecoder reactiveJwtDecoder(
      @Value("${gateway.jwt.hmacSecret:}") String hmacSecret
  ) {
    SecretKey key = new SecretKeySpec(hmacSecret.getBytes(StandardCharsets.UTF_8), "HmacSHA256");
    return NimbusReactiveJwtDecoder.withSecretKey(key).build();
  }

  @Bean
  CorsConfigurationSource corsConfigurationSource(
      @Value("${gateway.cors.allowedOrigins:*}") List<String> allowedOrigins
  ) {
    CorsConfiguration cfg = new CorsConfiguration();
    if (allowedOrigins.size() == 1 && "*".equals(allowedOrigins.get(0))) {
      cfg.setAllowedOriginPatterns(List.of("*"));
      cfg.setAllowCredentials(false);
    } else {
      cfg.setAllowedOrigins(allowedOrigins);
      cfg.setAllowCredentials(true);
    }
    cfg.setAllowedMethods(List.of("GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"));
    cfg.setAllowedHeaders(List.of(
        HttpHeaders.AUTHORIZATION,
        HttpHeaders.CONTENT_TYPE,
        "X-Tenant-ID",
        "X-Actor-ID",
        "X-User-ID",
        "X-Line-Signature",
        "X-Hub-Signature-256"
    ));
    cfg.setExposedHeaders(List.of("X-Request-ID"));

    UrlBasedCorsConfigurationSource source = new UrlBasedCorsConfigurationSource();
    source.registerCorsConfiguration("/**", cfg);
    return source;
  }
}
