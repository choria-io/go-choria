FROM almalinux:8

ARG REPO="https://yum.eu.choria.io/release/el/release.repo"

WORKDIR /

RUN yum -y update && \
    yum -y clean all

RUN curl -s "${REPO}" > /etc/yum.repos.d/choria.repo && \
    yum -y install choria nc procps-ng openssl && \
    yum -y clean all

RUN groupadd --gid 2048 choria && \
    useradd -c "Choria Orchestrator - choria.io" -m --uid 2048 --gid 2048 choria && \
    chown -R choria:choria /etc/choria && \
    mkdir /data && \
    chown choria:choria /data

USER choria
VOLUME /data

ENTRYPOINT ["/bin/choria"]
