FROM qaqcatz-mysql-env:env1

COPY *.rpm /root/

RUN cd /root \
&& yum -y install *.rpm

CMD mysqld_safe