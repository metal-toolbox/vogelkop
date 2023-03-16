FROM ghcr.io/equinixmetal/ironlib:v0.2.4

COPY vogelkop /vogelkop

ENTRYPOINT ["/vogelkop"]
CMD ["--help"]
