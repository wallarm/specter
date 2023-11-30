FROM alpine:3.19.0 as builder
ARG OS
ARG ARCH
COPY bin/specter-${OS}-${ARCH} /specter

FROM alpine:3.19.0
ARG OS
ARG ARCH
WORKDIR /app
COPY --from=builder /specter /app/
RUN echo "OS=${OS}, ARCH=${ARCH}"
RUN apk update && apk add --no-cache curl

# Install kubectl
RUN echo "https://dl.k8s.io/release/$(curl -Ls https://dl.k8s.io/release/stable.txt)/bin/${OS}/${ARCH}/kubectl" && \
    curl -LO "https://dl.k8s.io/release/$(curl -Ls https://dl.k8s.io/release/stable.txt)/bin/${OS}/${ARCH}/kubectl"

RUN curl -LO "https://dl.k8s.io/$(curl -Ls https://dl.k8s.io/release/stable.txt)/bin/${OS}/${ARCH}/kubectl.sha256"
RUN install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl
RUN chmod +x /usr/local/bin/kubectl

RUN which kubectl
RUN kubectl version --client=true

# Install helm
RUN curl -LO https://get.helm.sh/helm-v3.7.0-${OS}-${ARCH}.tar.gz && \
    tar -zxvf helm-v3.7.0-${OS}-${ARCH}.tar.gz && \
    mv ${OS}-${ARCH}/helm /usr/local/bin/helm && \
    rm -rf helm-v3.7.0-${OS}-${ARCH}.tar.gz ${OS}-${ARCH}

RUN chmod +x /usr/local/bin/helm

RUN which helm
RUN helm version

RUN rm -rf /var/cache/apk/*

RUN chmod +x /app/specter
ENTRYPOINT ["/app/specter"]
