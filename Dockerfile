# syntax = docker/dockerfile:1.1-experimental
FROM ubuntu

RUN apt-get update && apt-get install -y ca-certificates git

COPY stardata /usr/local/bin
RUN chmod 777 /usr/local/bin/stardata

RUN groupadd -g 1001 stardata \
    && useradd -m -u 1001 -s /bin/sh -g stardata stardata
USER stardata

RUN stardata runtime install-duckdb-extensions

ENTRYPOINT ["stardata"]
CMD ["start"]
