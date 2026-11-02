# Upstream runtime pinned on 2026-11-02; dependency provenance in docs/UPSTREAM.md.
FROM bitxeno/atvloadly@sha256:55a3a594dbe345d1b9d76b54aa44226cec25dfb1f3d67e6d7537303bcb01b310
ARG TARGETARCH
COPY build/atvloadly-linux-${TARGETARCH} /usr/bin/atvloadly
COPY packaging/entrypoint.sh /opt/songlib-entrypoint.sh
COPY docs/config.yaml.example /opt/config.yaml
COPY LICENSE /opt/LICENSE
COPY build/songlib-source.tar.gz /opt/songlib-source.tar.gz
RUN chmod 755 /usr/bin/atvloadly /opt/songlib-entrypoint.sh
ENV SONGLIB_HARDENED=1
ENV HOME=/data/home XDG_CONFIG_HOME=/data/home/.config
USER 10001:10001
EXPOSE 5533
ENTRYPOINT ["/opt/songlib-entrypoint.sh"]
