FROM alpine:3.19.0 as builder
ARG OS
ARG ARCH
COPY bin/specter-${OS}-${ARCH} /specter

FROM alpine:3.19.0
WORKDIR /app
COPY --from=builder /specter /app/
RUN apk update && apk add --no-cache curl

RUN chmod +x /app/specter
RUN rm -rf /var/cache/apk/*
ENTRYPOINT ["/app/specter"]