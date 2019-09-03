FROM centos:7
LABEL maintainers="Ceph-CSI Authors"
LABEL description="Ceph-CSI Plugin"

RUN true \
    && yum -y install centos-release-ceph \
    && yum -y install ceph-base \
    && yum -y update \
    && yum clean all \
    && true

ENV CSIBIN=/usr/local/bin/cephcsi
COPY cephcsi $CSIBIN
RUN chmod +x $CSIBIN

ENTRYPOINT ["/usr/local/bin/cephcsi"]
