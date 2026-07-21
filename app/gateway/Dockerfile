FROM maven:3.9-eclipse-temurin-21-alpine AS builder

WORKDIR /build
COPY pom.xml .
COPY src ./src
RUN mvn -q -DskipTests package

FROM eclipse-temurin:21-jre-alpine
WORKDIR /app
COPY --from=builder /build/target/communication-gateway-*.jar /app/gateway.jar
EXPOSE 8099
ENTRYPOINT ["java", "-jar", "/app/gateway.jar"]
