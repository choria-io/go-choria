FROM oraclelinux:5

RUN yum install -y rpm-build rubygem-rake && \
    yum clean all

WORKDIR /tmp
CMD /build/build.sh
