FROM alpine

RUN apk update \
        && apk upgrade \
        && apk add --no-cache \
        ca-certificates \
        && update-ca-certificates 2>/dev/null || true

COPY policybot /policybot
COPY policybot.yaml /policybot.yaml
COPY /generated /generated
COPY dashboard/static /dashboard/static

CMD ["/policybot", "server", "--configFile", "./policybot.yaml", "--https_only"]
