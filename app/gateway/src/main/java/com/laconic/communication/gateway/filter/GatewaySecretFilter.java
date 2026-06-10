package com.laconic.communication.gateway.filter;

import org.springframework.beans.factory.annotation.Value;
import org.springframework.cloud.gateway.filter.GatewayFilterChain;
import org.springframework.cloud.gateway.filter.GlobalFilter;
import org.springframework.core.Ordered;
import org.springframework.http.server.reactive.ServerHttpRequest;
import org.springframework.stereotype.Component;
import org.springframework.web.server.ServerWebExchange;

import reactor.core.publisher.Mono;

@Component
public class GatewaySecretFilter implements GlobalFilter, Ordered {

  private final String sharedSecret;

  public GatewaySecretFilter(@Value("${gateway.sharedSecret:}") String sharedSecret) {
    this.sharedSecret = sharedSecret == null ? "" : sharedSecret;
  }

  @Override
  public Mono<Void> filter(ServerWebExchange exchange, GatewayFilterChain chain) {
    if (sharedSecret.isBlank()) {
      return chain.filter(exchange);
    }

    String path = exchange.getRequest().getURI().getPath();
    if (path != null && path.startsWith("/actuator/")) {
      return chain.filter(exchange);
    }

    ServerHttpRequest mutated = exchange.getRequest().mutate()
        .headers(headers -> headers.set("X-Gateway-Secret", sharedSecret))
        .build();
    return chain.filter(exchange.mutate().request(mutated).build());
  }

  @Override
  public int getOrder() {
    return Ordered.LOWEST_PRECEDENCE;
  }
}
