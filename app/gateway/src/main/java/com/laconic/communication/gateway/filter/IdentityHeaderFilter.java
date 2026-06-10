package com.laconic.communication.gateway.filter;

import org.springframework.beans.factory.annotation.Value;
import org.springframework.cloud.gateway.filter.GatewayFilterChain;
import org.springframework.cloud.gateway.filter.GlobalFilter;
import org.springframework.core.Ordered;
import org.springframework.security.core.Authentication;
import org.springframework.security.oauth2.server.resource.authentication.JwtAuthenticationToken;
import org.springframework.stereotype.Component;
import org.springframework.web.server.ServerWebExchange;

import reactor.core.publisher.Mono;

@Component
public class IdentityHeaderFilter implements GlobalFilter, Ordered {

  private final String userIdClaim;
  private final String tenantIdClaim;

  public IdentityHeaderFilter(
      @Value("${gateway.auth.userIdClaim:sub}") String userIdClaim,
      @Value("${gateway.auth.tenantIdClaim:tenant_id}") String tenantIdClaim
  ) {
    this.userIdClaim = userIdClaim;
    this.tenantIdClaim = tenantIdClaim;
  }

  @Override
  public Mono<Void> filter(ServerWebExchange exchange, GatewayFilterChain chain) {
    return exchange.getPrincipal()
        .cast(Authentication.class)
        .flatMap(auth -> {
          if (auth instanceof JwtAuthenticationToken jwtAuth) {
            String actorId = jwtAuth.getToken().getClaimAsString(userIdClaim);
            String tenantId = jwtAuth.getToken().getClaimAsString(tenantIdClaim);
            if (tenantId == null || tenantId.isBlank()) {
              tenantId = "00000000-0000-0000-0000-000000000001";
            }

            if (actorId != null && !actorId.isBlank()) {
              String finalTenantId = tenantId;
              ServerWebExchange mutated = exchange.mutate()
                  .request(req -> req.headers(headers -> {
                    headers.set("X-Actor-ID", actorId);
                    headers.set("X-User-ID", actorId);
                    headers.set("X-Tenant-ID", finalTenantId);
                  }))
                  .build();
              return chain.filter(mutated);
            }
          }
          return chain.filter(exchange);
        })
        .switchIfEmpty(chain.filter(exchange));
  }

  @Override
  public int getOrder() {
    return Ordered.LOWEST_PRECEDENCE;
  }
}
