FROM centos:8

WORKDIR /

RUN curl -s https://packagecloud.io/install/repositories/choria/release/script.rpm.sh | bash && \
    yum -y update && \
    yum -y install choria nc && \
    yum -y clean all

RUN groupadd --gid 2048 choria && \
    useradd -c "Choria Orchestrator - choria.io" -m --uid 2048 --gid 2048 choria && \
    chown -R choria:choria /etc/choria && \
    mkdir /data && \
    chown choria:choria /data

USER choria
VOLUME /data

ENTRYPOINT ["/bin/choria"]
