FROM python

RUN pip install bzt # to cache the step

ADD taurus /tmp/taurus

RUN pip install /tmp/taurus

RUN bzt /tmp/taurus/dummy.yml # sanity test