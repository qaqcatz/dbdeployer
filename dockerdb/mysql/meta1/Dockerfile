FROM qaqcatz-mysql-env:env1

COPY *.rpm /root/

RUN cd /root \
&& yum -y install *.rpm \
&& ./init.sh

CMD mysqld_safe