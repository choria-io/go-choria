FROM centos:8

WORKDIR /

RUN curl -s https://packagecloud.io/install/repositories/choria/release/script.rpm.sh | bash && \
    yum -y update && \
    yum -y install choria && \
    yum -y clean all

ENTRYPOINT ["/bin/choria"]

COPY go-choria /bin/choria
