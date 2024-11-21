FROM alpine:3.20.0 as builder
ARG OS
ARG ARCH
COPY bin/specter-${OS}-${ARCH} /specter

FROM grafana/k6:latest

WORKDIR /home/k6

ARG OS
ARG ARCH

COPY --from=builder /specter /home/k6/specter

RUN echo "OS=${OS}, ARCH=${ARCH}"

USER root

RUN apk update && apk add --no-cache curl

RUN curl -LO "https://dl.k8s.io/release/$(curl -Ls https://dl.k8s.io/release/stable.txt)/bin/${OS}/${ARCH}/kubectl" && \
    curl -LO "https://dl.k8s.io/$(curl -Ls https://dl.k8s.io/release/stable.txt)/bin/${OS}/${ARCH}/kubectl.sha256" && \
    install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl && \
    chmod +x /usr/local/bin/kubectl

RUN which kubectl
RUN kubectl version --client=true

RUN curl -LO https://get.helm.sh/helm-v3.7.0-${OS}-${ARCH}.tar.gz && \
    tar -zxvf helm-v3.7.0-${OS}-${ARCH}.tar.gz && \
    mv ${OS}-${ARCH}/helm /usr/local/bin/helm && \
    rm -rf helm-v3.7.0-${OS}-${ARCH}.tar.gz ${OS}-${ARCH}

RUN chmod +x /usr/local/bin/helm

RUN which helm
RUN helm version
RUN rm -rf /var/cache/apk/*
RUN chmod +x /home/k6/specter && \
    chown k6:k6 /home/k6/specter
USER k6
RUN chmod +x /home/k6/specter
ENTRYPOINT ["k6"]