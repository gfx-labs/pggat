FROM postgres:14


RUN apt-get update && apt-get upgrade

RUN apt-get -y install git build-essential libreadline-dev zlib1g-dev flex bison libxml2-dev libxslt-dev libssl-dev libxml2-utils xsltproc ccache


WORKDIR /wd
RUN chown postgres:postgres /wd
USER postgres
RUN git clone https://github.com/postgres/postgres.git
WORKDIR /wd/postgres
RUN git checkout REL_14_STABLE
RUN ./configure
RUN make
USER root
RUN make install
USER postgres
