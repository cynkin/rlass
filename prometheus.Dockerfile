FROM alpine:3.19 AS tzdata
RUN apk add --no-cache tzdata

FROM prom/prometheus:latest
COPY --from=tzdata /usr/share/zoneinfo /usr/share/zoneinfo
ENV TZ=Asia/Kolkata
